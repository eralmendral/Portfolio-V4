package certificates

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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Certificate, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 3)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("c.status = $%d", len(args)))
	}
	if filter.Featured != nil {
		args = append(args, *filter.Featured)
		where = append(where, fmt.Sprintf("c.featured = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(c.title) LIKE $%[1]d OR
			lower(c.slug) LIKE $%[1]d OR
			lower(c.issuer) LIKE $%[1]d OR
			lower(c.summary) LIKE $%[1]d OR
			lower(c.description) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.slug, c.title, c.issuer, c.summary, c.description,
			c.credential_url, c.featured, c.sort_order, c.status, c.issued_at,
			c.expires_at, c.created_at, c.updated_at
		FROM certificates c
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY c.sort_order ASC, c.issued_at DESC NULLS LAST, c.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	certificates := make([]Certificate, 0)
	for rows.Next() {
		certificate, err := scanCertificate(rows)
		if err != nil {
			return nil, err
		}
		certificates = append(certificates, certificate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range certificates {
		if err := s.loadImage(ctx, s.db, &certificates[i]); err != nil {
			return nil, err
		}
	}

	return certificates, nil
}

func (s *PostgresStore) Get(ctx context.Context, idOrSlug string) (Certificate, error) {
	certificate, err := s.get(ctx, s.db, idOrSlug)
	if err != nil {
		return Certificate{}, err
	}
	return certificate, s.loadImage(ctx, s.db, &certificate)
}

func (s *PostgresStore) Create(ctx context.Context, certificate Certificate) (Certificate, error) {
	now := s.now().UTC()
	if certificate.ID == "" {
		certificate.ID = newID()
	}
	if certificate.Status == "" {
		certificate.Status = StatusDraft
	}
	if certificate.CreatedAt.IsZero() {
		certificate.CreatedAt = now
	}
	certificate.UpdatedAt = now

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Certificate{}, err
	}
	defer rollback(tx)

	if err := upsertCertificate(ctx, tx, certificate); err != nil {
		return Certificate{}, normalizePostgresError(err)
	}
	if err := replaceImage(ctx, tx, certificate); err != nil {
		return Certificate{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Certificate{}, normalizePostgresError(err)
	}

	return certificate, nil
}

func (s *PostgresStore) Update(ctx context.Context, idOrSlug string, mutate func(*Certificate) error) (Certificate, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Certificate{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, idOrSlug)
	if err != nil {
		return Certificate{}, err
	}
	if err := s.loadImage(ctx, tx, &current); err != nil {
		return Certificate{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Certificate{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertCertificate(ctx, tx, next); err != nil {
		return Certificate{}, normalizePostgresError(err)
	}
	if err := replaceImage(ctx, tx, next); err != nil {
		return Certificate{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Certificate{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, idOrSlug string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM certificates
		WHERE id = $1 OR slug = $1
	`, idOrSlug)
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, idOrSlug string) (Certificate, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT c.id, c.slug, c.title, c.issuer, c.summary, c.description,
			c.credential_url, c.featured, c.sort_order, c.status, c.issued_at,
			c.expires_at, c.created_at, c.updated_at
		FROM certificates c
		WHERE c.id = $1 OR c.slug = $1
	`, idOrSlug)

	certificate, err := scanCertificate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Certificate{}, ErrNotFound
	}
	return certificate, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, idOrSlug string) (Certificate, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT c.id, c.slug, c.title, c.issuer, c.summary, c.description,
			c.credential_url, c.featured, c.sort_order, c.status, c.issued_at,
			c.expires_at, c.created_at, c.updated_at
		FROM certificates c
		WHERE c.id = $1 OR c.slug = $1
		FOR UPDATE
	`, idOrSlug)

	certificate, err := scanCertificate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Certificate{}, ErrNotFound
	}
	return certificate, err
}

func (s *PostgresStore) loadImage(ctx context.Context, runner dbRunner, certificate *Certificate) error {
	row := runner.QueryRowContext(ctx, `
		SELECT id, url, path, alt_text, caption, content_type,
			size_bytes, width, height, uploaded_at
		FROM certificate_images
		WHERE certificate_id = $1
	`, certificate.ID)

	var image CertificateImage
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
		certificate.Image = nil
		return nil
	}
	if err != nil {
		return err
	}

	certificate.Image = &image
	return nil
}

func upsertCertificate(ctx context.Context, runner dbRunner, certificate Certificate) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO certificates (
			id, slug, title, issuer, summary, description, credential_url,
			featured, sort_order, status, issued_at, expires_at, created_at,
			updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (id) DO UPDATE SET
			slug = EXCLUDED.slug,
			title = EXCLUDED.title,
			issuer = EXCLUDED.issuer,
			summary = EXCLUDED.summary,
			description = EXCLUDED.description,
			credential_url = EXCLUDED.credential_url,
			featured = EXCLUDED.featured,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status,
			issued_at = EXCLUDED.issued_at,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at
	`, certificate.ID, certificate.Slug, certificate.Title, certificate.Issuer, certificate.Summary,
		certificate.Description, certificate.CredentialURL, certificate.Featured, certificate.SortOrder,
		certificate.Status, certificate.IssuedAt, certificate.ExpiresAt, certificate.CreatedAt,
		certificate.UpdatedAt)
	return err
}

func replaceImage(ctx context.Context, runner dbRunner, certificate Certificate) error {
	if _, err := runner.ExecContext(ctx, `
		DELETE FROM certificate_images
		WHERE certificate_id = $1
	`, certificate.ID); err != nil {
		return err
	}

	if certificate.Image != nil {
		return insertImage(ctx, runner, certificate.ID, *certificate.Image)
	}

	return nil
}

func insertImage(ctx context.Context, runner dbRunner, certificateID string, image CertificateImage) error {
	if image.ID == "" {
		image.ID = newID()
	}
	if image.UploadedAt.IsZero() {
		image.UploadedAt = time.Now().UTC()
	}

	_, err := runner.ExecContext(ctx, `
		INSERT INTO certificate_images (
			id, certificate_id, url, path, alt_text, caption, content_type,
			size_bytes, width, height, uploaded_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, image.ID, certificateID, image.URL, image.Path, image.AltText, image.Caption,
		image.ContentType, image.SizeBytes, image.Width, image.Height, image.UploadedAt)
	return err
}

type certificateScanner interface {
	Scan(...any) error
}

func scanCertificate(scanner certificateScanner) (Certificate, error) {
	var certificate Certificate
	if err := scanner.Scan(
		&certificate.ID,
		&certificate.Slug,
		&certificate.Title,
		&certificate.Issuer,
		&certificate.Summary,
		&certificate.Description,
		&certificate.CredentialURL,
		&certificate.Featured,
		&certificate.SortOrder,
		&certificate.Status,
		&certificate.IssuedAt,
		&certificate.ExpiresAt,
		&certificate.CreatedAt,
		&certificate.UpdatedAt,
	); err != nil {
		return Certificate{}, err
	}

	return certificate, nil
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
CREATE TABLE IF NOT EXISTS certificates (
	id TEXT PRIMARY KEY,
	slug TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	issuer TEXT NOT NULL DEFAULT '',
	summary TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	credential_url TEXT NOT NULL DEFAULT '',
	featured BOOLEAN NOT NULL DEFAULT FALSE,
	sort_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	issued_at TIMESTAMPTZ,
	expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS certificate_images (
	id TEXT PRIMARY KEY,
	certificate_id TEXT NOT NULL UNIQUE REFERENCES certificates(id) ON DELETE CASCADE,
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

CREATE INDEX IF NOT EXISTS certificates_status_sort_order
	ON certificates(status, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS certificates_featured_sort_order
	ON certificates(featured, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS certificates_sort_order_issued_at
	ON certificates(sort_order ASC, issued_at DESC, created_at DESC);
`
