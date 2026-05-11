package series

import (
	"context"
	"database/sql"
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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Series, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 6)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("s.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Category) != "" {
		args = append(args, normalizeCategory(filter.Category))
		where = append(where, fmt.Sprintf("s.category = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(s.title) LIKE $%[1]d OR
			lower(s.creator) LIKE $%[1]d OR
			lower(s.platform) LIKE $%[1]d OR
			lower(s.notes) LIKE $%[1]d
		)`, len(args)))
	}
	if strings.TrimSpace(filter.From) != "" {
		args = append(args, normalizeDate(filter.From))
		where = append(where, fmt.Sprintf("s.mostly_watched_on >= $%d::date", len(args)))
	}
	if strings.TrimSpace(filter.To) != "" {
		args = append(args, normalizeDate(filter.To))
		where = append(where, fmt.Sprintf("s.mostly_watched_on <= $%d::date", len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.title, s.category, s.creator, s.platform, s.watch_url,
			s.mostly_watched_on, s.notes, s.sort_order, s.status, s.created_at, s.updated_at
		FROM series s
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY s.sort_order ASC, s.mostly_watched_on DESC, s.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Series, 0)
	for rows.Next() {
		entry, err := scanSeries(rows)
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

func (s *PostgresStore) Get(ctx context.Context, id string) (Series, error) {
	return s.get(ctx, s.db, id)
}

func (s *PostgresStore) Create(ctx context.Context, series Series) (Series, error) {
	now := s.now().UTC()
	if series.ID == "" {
		series.ID = newID()
	}
	if series.Status == "" {
		series.Status = StatusDraft
	}
	series.Category = normalizeCategory(series.Category)
	series.MostlyWatchedOn = normalizeDate(series.MostlyWatchedOn)
	if series.CreatedAt.IsZero() {
		series.CreatedAt = now
	}
	series.UpdatedAt = now

	if err := upsertSeries(ctx, s.db, series); err != nil {
		return Series{}, normalizePostgresError(err)
	}

	return series, nil
}

func (s *PostgresStore) Update(ctx context.Context, id string, mutate func(*Series) error) (Series, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Series{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, id)
	if err != nil {
		return Series{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Series{}, err
	}
	next.ID = current.ID
	next.Category = normalizeCategory(next.Category)
	next.MostlyWatchedOn = normalizeDate(next.MostlyWatchedOn)
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertSeries(ctx, tx, next); err != nil {
		return Series{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Series{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM series
		WHERE id = $1
	`, id)
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, id string) (Series, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT s.id, s.title, s.category, s.creator, s.platform, s.watch_url,
			s.mostly_watched_on, s.notes, s.sort_order, s.status, s.created_at, s.updated_at
		FROM series s
		WHERE s.id = $1
	`, id)

	series, err := scanSeries(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Series{}, ErrNotFound
	}
	return series, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, id string) (Series, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT s.id, s.title, s.category, s.creator, s.platform, s.watch_url,
			s.mostly_watched_on, s.notes, s.sort_order, s.status, s.created_at, s.updated_at
		FROM series s
		WHERE s.id = $1
		FOR UPDATE
	`, id)

	series, err := scanSeries(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Series{}, ErrNotFound
	}
	return series, err
}

func upsertSeries(ctx context.Context, runner dbRunner, series Series) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO series (
			id, title, category, creator, platform, watch_url,
			mostly_watched_on, notes, sort_order, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			category = EXCLUDED.category,
			creator = EXCLUDED.creator,
			platform = EXCLUDED.platform,
			watch_url = EXCLUDED.watch_url,
			mostly_watched_on = EXCLUDED.mostly_watched_on,
			notes = EXCLUDED.notes,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, series.ID, series.Title, series.Category, series.Creator, series.Platform, series.WatchURL,
		series.MostlyWatchedOn, series.Notes, series.SortOrder, series.Status, series.CreatedAt, series.UpdatedAt)
	return err
}

type seriesScanner interface {
	Scan(...any) error
}

func scanSeries(scanner seriesScanner) (Series, error) {
	var series Series
	var mostlyWatchedOn time.Time

	if err := scanner.Scan(
		&series.ID,
		&series.Title,
		&series.Category,
		&series.Creator,
		&series.Platform,
		&series.WatchURL,
		&mostlyWatchedOn,
		&series.Notes,
		&series.SortOrder,
		&series.Status,
		&series.CreatedAt,
		&series.UpdatedAt,
	); err != nil {
		return Series{}, err
	}

	series.MostlyWatchedOn = mostlyWatchedOn.Format(dateLayout)
	return series, nil
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
CREATE TABLE IF NOT EXISTS series (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	category TEXT NOT NULL CHECK (category IN ('tv_series', 'anime')),
	creator TEXT NOT NULL DEFAULT '',
	platform TEXT NOT NULL DEFAULT '',
	watch_url TEXT NOT NULL DEFAULT '',
	mostly_watched_on DATE NOT NULL,
	notes TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS series_status_category_sort_order_mostly_watched_on
	ON series(status, category, sort_order ASC, mostly_watched_on DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS series_mostly_watched_on
	ON series(mostly_watched_on DESC, created_at DESC);
`
