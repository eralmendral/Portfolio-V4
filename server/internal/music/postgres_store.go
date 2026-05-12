package music

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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Music, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 5)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("m.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(m.title) LIKE $%[1]d OR
			lower(m.artist) LIKE $%[1]d OR
			lower(m.album) LIKE $%[1]d OR
			lower(m.notes) LIKE $%[1]d
		)`, len(args)))
	}
	if strings.TrimSpace(filter.From) != "" {
		args = append(args, normalizeDate(filter.From))
		where = append(where, fmt.Sprintf("m.mostly_listened_on >= $%d::date", len(args)))
	}
	if strings.TrimSpace(filter.To) != "" {
		args = append(args, normalizeDate(filter.To))
		where = append(where, fmt.Sprintf("m.mostly_listened_on <= $%d::date", len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.title, m.artist, m.album, m.spotify_url, m.youtube_url,
			m.mostly_listened_on, m.notes, m.sort_order, m.status, m.created_at, m.updated_at
		FROM music m
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY m.sort_order ASC, m.mostly_listened_on DESC, m.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Music, 0)
	for rows.Next() {
		entry, err := scanMusic(rows)
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

func (s *PostgresStore) Get(ctx context.Context, id string) (Music, error) {
	return s.get(ctx, s.db, id)
}

func (s *PostgresStore) Create(ctx context.Context, music Music) (Music, error) {
	now := s.now().UTC()
	if music.ID == "" {
		music.ID = newID()
	}
	if music.Status == "" {
		music.Status = StatusDraft
	}
	music.MostlyListenedOn = normalizeDate(music.MostlyListenedOn)
	if music.CreatedAt.IsZero() {
		music.CreatedAt = now
	}
	music.UpdatedAt = now

	if err := upsertMusic(ctx, s.db, music); err != nil {
		return Music{}, normalizePostgresError(err)
	}

	return music, nil
}

func (s *PostgresStore) Update(ctx context.Context, id string, mutate func(*Music) error) (Music, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Music{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, id)
	if err != nil {
		return Music{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Music{}, err
	}
	next.ID = current.ID
	next.MostlyListenedOn = normalizeDate(next.MostlyListenedOn)
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertMusic(ctx, tx, next); err != nil {
		return Music{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Music{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM music
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, id string) (Music, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT m.id, m.title, m.artist, m.album, m.spotify_url, m.youtube_url,
			m.mostly_listened_on, m.notes, m.sort_order, m.status, m.created_at, m.updated_at
		FROM music m
		WHERE m.id = $1
	`, id)

	music, err := scanMusic(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Music{}, ErrNotFound
	}
	return music, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, id string) (Music, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT m.id, m.title, m.artist, m.album, m.spotify_url, m.youtube_url,
			m.mostly_listened_on, m.notes, m.sort_order, m.status, m.created_at, m.updated_at
		FROM music m
		WHERE m.id = $1
		FOR UPDATE
	`, id)

	music, err := scanMusic(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Music{}, ErrNotFound
	}
	return music, err
}

func upsertMusic(ctx context.Context, runner dbRunner, music Music) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO music (
			id, title, artist, album, spotify_url, youtube_url,
			mostly_listened_on, notes, sort_order, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			artist = EXCLUDED.artist,
			album = EXCLUDED.album,
			spotify_url = EXCLUDED.spotify_url,
			youtube_url = EXCLUDED.youtube_url,
			mostly_listened_on = EXCLUDED.mostly_listened_on,
			notes = EXCLUDED.notes,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, music.ID, music.Title, music.Artist, music.Album, music.SpotifyURL, music.YouTubeURL,
		music.MostlyListenedOn, music.Notes, music.SortOrder, music.Status, music.CreatedAt, music.UpdatedAt)
	return err
}

type musicScanner interface {
	Scan(...any) error
}

func scanMusic(scanner musicScanner) (Music, error) {
	var music Music
	var mostlyListenedOn time.Time

	if err := scanner.Scan(
		&music.ID,
		&music.Title,
		&music.Artist,
		&music.Album,
		&music.SpotifyURL,
		&music.YouTubeURL,
		&mostlyListenedOn,
		&music.Notes,
		&music.SortOrder,
		&music.Status,
		&music.CreatedAt,
		&music.UpdatedAt,
	); err != nil {
		return Music{}, err
	}

	music.MostlyListenedOn = mostlyListenedOn.Format(dateLayout)
	return music, nil
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
CREATE TABLE IF NOT EXISTS music (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	artist TEXT NOT NULL,
	album TEXT NOT NULL DEFAULT '',
	spotify_url TEXT NOT NULL DEFAULT '',
	youtube_url TEXT NOT NULL DEFAULT '',
	mostly_listened_on DATE NOT NULL,
	notes TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS music_status_sort_order_mostly_listened_on
	ON music(status, sort_order ASC, mostly_listened_on DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS music_mostly_listened_on
	ON music(mostly_listened_on DESC, created_at DESC);
`
