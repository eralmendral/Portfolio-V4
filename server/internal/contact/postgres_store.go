package contact

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

func (s *PostgresStore) Create(ctx context.Context, submission Submission) (Submission, error) {
	if err := validateSaved(submission); err != nil {
		return Submission{}, err
	}
	if submission.CreatedAt.IsZero() {
		submission.CreatedAt = s.now().UTC()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO contact_submissions (
			id, name, email, subject, message, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, submission.ID, submission.Name, submission.Email, submission.Subject, submission.Message, submission.CreatedAt)
	if err != nil {
		return Submission{}, err
	}

	return submission, nil
}

func (s *PostgresStore) GetProfile(ctx context.Context) (Profile, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, work_email, phone_number, created_at, updated_at
		FROM contact_profile
		WHERE id = $1
	`, DefaultProfileID)

	profile, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	return profile, err
}

func (s *PostgresStore) SaveProfile(ctx context.Context, profile Profile) (Profile, error) {
	if profile.ID == "" {
		profile.ID = DefaultProfileID
	}
	if err := validateSavedProfile(profile); err != nil {
		return Profile{}, err
	}

	now := s.now().UTC()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO contact_profile (
			id, work_email, phone_number, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			work_email = EXCLUDED.work_email,
			phone_number = EXCLUDED.phone_number,
			updated_at = EXCLUDED.updated_at
	`, profile.ID, profile.WorkEmail, profile.PhoneNumber, profile.CreatedAt, profile.UpdatedAt)
	if err != nil {
		return Profile{}, err
	}

	return profile, nil
}

type profileScanner interface {
	Scan(...any) error
}

func scanProfile(scanner profileScanner) (Profile, error) {
	var profile Profile
	if err := scanner.Scan(
		&profile.ID,
		&profile.WorkEmail,
		&profile.PhoneNumber,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

const postgresSchema = `
CREATE TABLE IF NOT EXISTS contact_submissions (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	email TEXT NOT NULL,
	subject TEXT NOT NULL DEFAULT '',
	message TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS contact_submissions_created_at
	ON contact_submissions(created_at DESC);

CREATE TABLE IF NOT EXISTS contact_profile (
	id TEXT PRIMARY KEY CHECK (id = 'default'),
	work_email TEXT NOT NULL,
	phone_number TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`
