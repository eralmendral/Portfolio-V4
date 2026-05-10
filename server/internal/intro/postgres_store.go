package intro

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type PostgresStore struct {
	db  *sql.DB
	now func() time.Time
}

type dbRunner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
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

func (s *PostgresStore) Get(ctx context.Context) (Intro, error) {
	intro, err := s.get(ctx, s.db)
	if err != nil {
		return Intro{}, err
	}
	return intro, s.loadProfilePicture(ctx, s.db, &intro)
}

func (s *PostgresStore) Save(ctx context.Context, intro Intro) (Intro, error) {
	now := s.now().UTC()
	if intro.ID == "" {
		intro.ID = DefaultID
	}
	if intro.CreatedAt.IsZero() {
		intro.CreatedAt = now
	}
	intro.UpdatedAt = now

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Intro{}, err
	}
	defer rollback(tx)

	if err := upsertIntro(ctx, tx, intro); err != nil {
		return Intro{}, err
	}
	if err := replaceProfilePicture(ctx, tx, intro); err != nil {
		return Intro{}, err
	}
	if err := tx.Commit(); err != nil {
		return Intro{}, err
	}

	return intro, nil
}

func (s *PostgresStore) Delete(ctx context.Context) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM intro
		WHERE id = $1
	`, DefaultID)
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner) (Intro, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT id, title, description, created_at, updated_at
		FROM intro
		WHERE id = $1
	`, DefaultID)

	intro, err := scanIntro(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Intro{}, ErrNotFound
	}
	return intro, err
}

func (s *PostgresStore) loadProfilePicture(ctx context.Context, runner dbRunner, intro *Intro) error {
	row := runner.QueryRowContext(ctx, `
		SELECT id, url, path, alt_text, caption, content_type,
			size_bytes, width, height, uploaded_at
		FROM intro_profile_pictures
		WHERE intro_id = $1
	`, intro.ID)

	var image IntroImage
	err := row.Scan(
		&image.ID,
		&image.URL,
		&image.Path,
		&image.AltText,
		&image.Caption,
		&image.ContentType,
		&image.SizeBytes,
		&image.Width,
		&image.Height,
		&image.UploadedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		intro.ProfilePicture = nil
		return nil
	}
	if err != nil {
		return err
	}

	intro.ProfilePicture = &image
	return nil
}

func upsertIntro(ctx context.Context, runner dbRunner, intro Intro) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO intro (
			id, title, description, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at
	`, intro.ID, intro.Title, intro.Description, intro.CreatedAt, intro.UpdatedAt)
	return err
}

func replaceProfilePicture(ctx context.Context, runner dbRunner, intro Intro) error {
	if _, err := runner.ExecContext(ctx, `
		DELETE FROM intro_profile_pictures
		WHERE intro_id = $1
	`, intro.ID); err != nil {
		return err
	}

	if intro.ProfilePicture != nil {
		return insertProfilePicture(ctx, runner, intro.ID, *intro.ProfilePicture)
	}

	return nil
}

func insertProfilePicture(ctx context.Context, runner dbRunner, introID string, image IntroImage) error {
	if image.ID == "" {
		image.ID = newID()
	}
	if image.UploadedAt.IsZero() {
		image.UploadedAt = time.Now().UTC()
	}

	_, err := runner.ExecContext(ctx, `
		INSERT INTO intro_profile_pictures (
			id, intro_id, url, path, alt_text, caption, content_type,
			size_bytes, width, height, uploaded_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, image.ID, introID, image.URL, image.Path, image.AltText, image.Caption,
		image.ContentType, image.SizeBytes, image.Width, image.Height, image.UploadedAt)
	return err
}

type introScanner interface {
	Scan(...any) error
}

func scanIntro(scanner introScanner) (Intro, error) {
	var intro Intro
	if err := scanner.Scan(
		&intro.ID,
		&intro.Title,
		&intro.Description,
		&intro.CreatedAt,
		&intro.UpdatedAt,
	); err != nil {
		return Intro{}, err
	}

	return intro, nil
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

const postgresSchema = `
CREATE TABLE IF NOT EXISTS intro (
	id TEXT PRIMARY KEY CHECK (id = 'default'),
	title TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS intro_profile_pictures (
	id TEXT PRIMARY KEY,
	intro_id TEXT NOT NULL UNIQUE REFERENCES intro(id) ON DELETE CASCADE,
	url TEXT NOT NULL,
	path TEXT NOT NULL DEFAULT '',
	alt_text TEXT NOT NULL DEFAULT '',
	caption TEXT NOT NULL DEFAULT '',
	content_type TEXT NOT NULL DEFAULT '',
	size_bytes BIGINT NOT NULL DEFAULT 0,
	width INTEGER NOT NULL DEFAULT 0,
	height INTEGER NOT NULL DEFAULT 0,
	uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`
