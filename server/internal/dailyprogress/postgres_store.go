package dailyprogress

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type PostgresStore struct {
	db  *sql.DB
	now func() time.Time
}

type dbRunner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func NewPostgresStore(ctx context.Context, db *sql.DB) (*PostgresStore, error) {
	store := &PostgresStore{
		db:  db,
		now: time.Now,
	}

	if err := store.Migrate(ctx); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *PostgresStore) Migrate(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, postgresSchema)
	return err
}

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]DailyProgress, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 5)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("d.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Tag) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Tag))+"%")
		where = append(where, fmt.Sprintf("lower(d.tags::text) LIKE $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(d.title) LIKE $%[1]d OR
			lower(d.summary) LIKE $%[1]d OR
			lower(d.content) LIKE $%[1]d OR
			lower(d.mood) LIKE $%[1]d OR
			lower(d.wins::text) LIKE $%[1]d OR
			lower(d.learnings::text) LIKE $%[1]d OR
			lower(d.next_steps::text) LIKE $%[1]d OR
			lower(d.tags::text) LIKE $%[1]d
		)`, len(args)))
	}
	if strings.TrimSpace(filter.From) != "" {
		args = append(args, normalizeDate(filter.From))
		where = append(where, fmt.Sprintf("d.entry_date >= $%d::date", len(args)))
	}
	if strings.TrimSpace(filter.To) != "" {
		args = append(args, normalizeDate(filter.To))
		where = append(where, fmt.Sprintf("d.entry_date <= $%d::date", len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT d.id, d.entry_date, d.title, d.summary, d.content, d.mood,
			d.progress_score, d.wins, d.blockers, d.learnings, d.next_steps,
			d.tags, d.status, d.created_at, d.updated_at
		FROM daily_progress d
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY d.entry_date DESC, d.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]DailyProgress, 0)
	for rows.Next() {
		entry, err := scanDailyProgress(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (s *PostgresStore) Get(ctx context.Context, idOrDate string) (DailyProgress, error) {
	return s.get(ctx, s.db, idOrDate)
}

func (s *PostgresStore) Create(ctx context.Context, entry DailyProgress) (DailyProgress, error) {
	now := s.now().UTC()
	if entry.ID == "" {
		entry.ID = newID()
	}
	if entry.Status == "" {
		entry.Status = StatusDraft
	}
	entry.EntryDate = normalizeDate(entry.EntryDate)
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	if err := upsertDailyProgress(ctx, s.db, entry); err != nil {
		return DailyProgress{}, normalizePostgresError(err)
	}

	return entry, nil
}

func (s *PostgresStore) Update(ctx context.Context, idOrDate string, mutate func(*DailyProgress) error) (DailyProgress, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return DailyProgress{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, idOrDate)
	if err != nil {
		return DailyProgress{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return DailyProgress{}, err
	}
	next.ID = current.ID
	next.EntryDate = normalizeDate(next.EntryDate)
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertDailyProgress(ctx, tx, next); err != nil {
		return DailyProgress{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return DailyProgress{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, idOrDate string) error {
	where, args := lookupWhere("", idOrDate)
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM daily_progress
		WHERE `+where, args...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, idOrDate string) (DailyProgress, error) {
	where, args := lookupWhere("d.", idOrDate)
	row := runner.QueryRowContext(ctx, `
		SELECT d.id, d.entry_date, d.title, d.summary, d.content, d.mood,
			d.progress_score, d.wins, d.blockers, d.learnings, d.next_steps,
			d.tags, d.status, d.created_at, d.updated_at
		FROM daily_progress d
		WHERE `+where, args...)

	entry, err := scanDailyProgress(row)
	if errors.Is(err, sql.ErrNoRows) {
		return DailyProgress{}, ErrNotFound
	}
	return entry, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, idOrDate string) (DailyProgress, error) {
	where, args := lookupWhere("d.", idOrDate)
	row := runner.QueryRowContext(ctx, `
		SELECT d.id, d.entry_date, d.title, d.summary, d.content, d.mood,
			d.progress_score, d.wins, d.blockers, d.learnings, d.next_steps,
			d.tags, d.status, d.created_at, d.updated_at
		FROM daily_progress d
		WHERE `+where+`
		FOR UPDATE
	`, args...)

	entry, err := scanDailyProgress(row)
	if errors.Is(err, sql.ErrNoRows) {
		return DailyProgress{}, ErrNotFound
	}
	return entry, err
}

func lookupWhere(prefix string, idOrDate string) (string, []any) {
	if _, err := parseDate(idOrDate); err == nil {
		return prefix + "id = $1 OR " + prefix + "entry_date = $2::date", []any{idOrDate, normalizeDate(idOrDate)}
	}
	return prefix + "id = $1", []any{idOrDate}
}

func upsertDailyProgress(ctx context.Context, runner dbRunner, entry DailyProgress) error {
	wins, err := json.Marshal(entry.Wins)
	if err != nil {
		return err
	}
	blockers, err := json.Marshal(entry.Blockers)
	if err != nil {
		return err
	}
	learnings, err := json.Marshal(entry.Learnings)
	if err != nil {
		return err
	}
	nextSteps, err := json.Marshal(entry.NextSteps)
	if err != nil {
		return err
	}
	tags, err := json.Marshal(entry.Tags)
	if err != nil {
		return err
	}

	_, err = runner.ExecContext(ctx, `
		INSERT INTO daily_progress (
			id, entry_date, title, summary, content, mood, progress_score,
			wins, blockers, learnings, next_steps, tags, status, created_at, updated_at
		)
		VALUES ($1, $2::date, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			entry_date = EXCLUDED.entry_date,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			content = EXCLUDED.content,
			mood = EXCLUDED.mood,
			progress_score = EXCLUDED.progress_score,
			wins = EXCLUDED.wins,
			blockers = EXCLUDED.blockers,
			learnings = EXCLUDED.learnings,
			next_steps = EXCLUDED.next_steps,
			tags = EXCLUDED.tags,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, entry.ID, entry.EntryDate, entry.Title, entry.Summary, entry.Content, entry.Mood,
		entry.ProgressScore, string(wins), string(blockers), string(learnings), string(nextSteps),
		string(tags), entry.Status, entry.CreatedAt, entry.UpdatedAt)
	return err
}

type dailyProgressScanner interface {
	Scan(...any) error
}

func scanDailyProgress(scanner dailyProgressScanner) (DailyProgress, error) {
	var entry DailyProgress
	var entryDate time.Time
	var wins []byte
	var blockers []byte
	var learnings []byte
	var nextSteps []byte
	var tags []byte

	if err := scanner.Scan(
		&entry.ID,
		&entryDate,
		&entry.Title,
		&entry.Summary,
		&entry.Content,
		&entry.Mood,
		&entry.ProgressScore,
		&wins,
		&blockers,
		&learnings,
		&nextSteps,
		&tags,
		&entry.Status,
		&entry.CreatedAt,
		&entry.UpdatedAt,
	); err != nil {
		return DailyProgress{}, err
	}

	entry.EntryDate = entryDate.Format(dateLayout)
	if err := unmarshalStringSlice(wins, &entry.Wins); err != nil {
		return DailyProgress{}, err
	}
	if err := unmarshalStringSlice(blockers, &entry.Blockers); err != nil {
		return DailyProgress{}, err
	}
	if err := unmarshalStringSlice(learnings, &entry.Learnings); err != nil {
		return DailyProgress{}, err
	}
	if err := unmarshalStringSlice(nextSteps, &entry.NextSteps); err != nil {
		return DailyProgress{}, err
	}
	if err := unmarshalStringSlice(tags, &entry.Tags); err != nil {
		return DailyProgress{}, err
	}

	return entry, nil
}

func unmarshalStringSlice(data []byte, target *[]string) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func normalizePostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		}
	}
	return err
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

const postgresSchema = `
CREATE TABLE IF NOT EXISTS daily_progress (
	id TEXT PRIMARY KEY,
	entry_date DATE NOT NULL UNIQUE,
	title TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	content TEXT NOT NULL DEFAULT '',
	mood TEXT NOT NULL DEFAULT '',
	progress_score INTEGER NOT NULL DEFAULT 0 CHECK (progress_score >= 0 AND progress_score <= 100),
	wins JSONB NOT NULL DEFAULT '[]'::jsonb,
	blockers JSONB NOT NULL DEFAULT '[]'::jsonb,
	learnings JSONB NOT NULL DEFAULT '[]'::jsonb,
	next_steps JSONB NOT NULL DEFAULT '[]'::jsonb,
	tags JSONB NOT NULL DEFAULT '[]'::jsonb,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS daily_progress_status_entry_date
	ON daily_progress(status, entry_date DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS daily_progress_entry_date
	ON daily_progress(entry_date DESC, created_at DESC);
`
