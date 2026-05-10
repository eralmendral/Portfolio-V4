package workexperience

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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]WorkExperience, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 4)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("w.status = $%d", len(args)))
	}
	if filter.Featured != nil {
		args = append(args, *filter.Featured)
		where = append(where, fmt.Sprintf("w.featured = $%d", len(args)))
	}
	if filter.Current != nil {
		args = append(args, *filter.Current)
		where = append(where, fmt.Sprintf("w.is_current = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(w.title) LIKE $%[1]d OR
			lower(w.slug) LIKE $%[1]d OR
			lower(w.company) LIKE $%[1]d OR
			lower(w.employment_type) LIKE $%[1]d OR
			lower(w.location) LIKE $%[1]d OR
			lower(w.location_type) LIKE $%[1]d OR
			lower(w.summary) LIKE $%[1]d OR
			lower(w.description) LIKE $%[1]d OR
			lower(w.highlights::text) LIKE $%[1]d OR
			lower(w.responsibilities::text) LIKE $%[1]d OR
			lower(w.tech_stack::text) LIKE $%[1]d OR
			lower(w.skills::text) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.slug, w.title, w.company, w.company_url,
			w.company_logo_url, w.employment_type, w.location, w.location_type,
			w.summary, w.description, w.highlights, w.responsibilities,
			w.tech_stack, w.skills, w.started_at, w.ended_at, w.is_current,
			w.featured, w.sort_order, w.status, w.published_at, w.created_at,
			w.updated_at
		FROM work_experiences w
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY w.sort_order ASC, w.is_current DESC, w.started_at DESC, w.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workExperiences := make([]WorkExperience, 0)
	for rows.Next() {
		workExperience, err := scanWorkExperience(rows)
		if err != nil {
			return nil, err
		}
		workExperiences = append(workExperiences, workExperience)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return workExperiences, nil
}

func (s *PostgresStore) Get(ctx context.Context, idOrSlug string) (WorkExperience, error) {
	return s.get(ctx, s.db, idOrSlug)
}

func (s *PostgresStore) Create(ctx context.Context, workExperience WorkExperience) (WorkExperience, error) {
	now := s.now().UTC()
	if workExperience.ID == "" {
		workExperience.ID = newID()
	}
	if workExperience.Status == "" {
		workExperience.Status = StatusDraft
	}
	if workExperience.CreatedAt.IsZero() {
		workExperience.CreatedAt = now
	}
	workExperience.UpdatedAt = now

	if err := upsertWorkExperience(ctx, s.db, workExperience); err != nil {
		return WorkExperience{}, normalizePostgresError(err)
	}

	return workExperience, nil
}

func (s *PostgresStore) Update(ctx context.Context, idOrSlug string, mutate func(*WorkExperience) error) (WorkExperience, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return WorkExperience{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, idOrSlug)
	if err != nil {
		return WorkExperience{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return WorkExperience{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertWorkExperience(ctx, tx, next); err != nil {
		return WorkExperience{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return WorkExperience{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, idOrSlug string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM work_experiences
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, idOrSlug string) (WorkExperience, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT w.id, w.slug, w.title, w.company, w.company_url,
			w.company_logo_url, w.employment_type, w.location, w.location_type,
			w.summary, w.description, w.highlights, w.responsibilities,
			w.tech_stack, w.skills, w.started_at, w.ended_at, w.is_current,
			w.featured, w.sort_order, w.status, w.published_at, w.created_at,
			w.updated_at
		FROM work_experiences w
		WHERE w.id = $1 OR w.slug = $1
	`, idOrSlug)

	workExperience, err := scanWorkExperience(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkExperience{}, ErrNotFound
	}
	return workExperience, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, idOrSlug string) (WorkExperience, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT w.id, w.slug, w.title, w.company, w.company_url,
			w.company_logo_url, w.employment_type, w.location, w.location_type,
			w.summary, w.description, w.highlights, w.responsibilities,
			w.tech_stack, w.skills, w.started_at, w.ended_at, w.is_current,
			w.featured, w.sort_order, w.status, w.published_at, w.created_at,
			w.updated_at
		FROM work_experiences w
		WHERE w.id = $1 OR w.slug = $1
		FOR UPDATE
	`, idOrSlug)

	workExperience, err := scanWorkExperience(row)
	if errors.Is(err, sql.ErrNoRows) {
		return WorkExperience{}, ErrNotFound
	}
	return workExperience, err
}

func upsertWorkExperience(ctx context.Context, runner dbRunner, workExperience WorkExperience) error {
	highlights, err := json.Marshal(workExperience.Highlights)
	if err != nil {
		return err
	}
	responsibilities, err := json.Marshal(workExperience.Responsibilities)
	if err != nil {
		return err
	}
	techStack, err := json.Marshal(workExperience.TechStack)
	if err != nil {
		return err
	}
	skills, err := json.Marshal(workExperience.Skills)
	if err != nil {
		return err
	}

	_, err = runner.ExecContext(ctx, `
		INSERT INTO work_experiences (
			id, slug, title, company, company_url, company_logo_url,
			employment_type, location, location_type, summary, description,
			highlights, responsibilities, tech_stack, skills, started_at,
			ended_at, is_current, featured, sort_order, status, published_at,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12::jsonb, $13::jsonb, $14::jsonb, $15::jsonb,
			$16, $17, $18, $19, $20, $21, $22, $23, $24
		)
		ON CONFLICT (id) DO UPDATE SET
			slug = EXCLUDED.slug,
			title = EXCLUDED.title,
			company = EXCLUDED.company,
			company_url = EXCLUDED.company_url,
			company_logo_url = EXCLUDED.company_logo_url,
			employment_type = EXCLUDED.employment_type,
			location = EXCLUDED.location,
			location_type = EXCLUDED.location_type,
			summary = EXCLUDED.summary,
			description = EXCLUDED.description,
			highlights = EXCLUDED.highlights,
			responsibilities = EXCLUDED.responsibilities,
			tech_stack = EXCLUDED.tech_stack,
			skills = EXCLUDED.skills,
			started_at = EXCLUDED.started_at,
			ended_at = EXCLUDED.ended_at,
			is_current = EXCLUDED.is_current,
			featured = EXCLUDED.featured,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status,
			published_at = EXCLUDED.published_at,
			updated_at = EXCLUDED.updated_at
	`, workExperience.ID, workExperience.Slug, workExperience.Title, workExperience.Company,
		workExperience.CompanyURL, workExperience.CompanyLogoURL, workExperience.EmploymentType,
		workExperience.Location, workExperience.LocationType, workExperience.Summary,
		workExperience.Description, string(highlights), string(responsibilities),
		string(techStack), string(skills), workExperience.StartedAt, workExperience.EndedAt,
		workExperience.Current, workExperience.Featured, workExperience.SortOrder,
		workExperience.Status, workExperience.PublishedAt, workExperience.CreatedAt,
		workExperience.UpdatedAt)
	return err
}

type workExperienceScanner interface {
	Scan(...any) error
}

func scanWorkExperience(scanner workExperienceScanner) (WorkExperience, error) {
	var workExperience WorkExperience
	var highlights []byte
	var responsibilities []byte
	var techStack []byte
	var skills []byte
	if err := scanner.Scan(
		&workExperience.ID,
		&workExperience.Slug,
		&workExperience.Title,
		&workExperience.Company,
		&workExperience.CompanyURL,
		&workExperience.CompanyLogoURL,
		&workExperience.EmploymentType,
		&workExperience.Location,
		&workExperience.LocationType,
		&workExperience.Summary,
		&workExperience.Description,
		&highlights,
		&responsibilities,
		&techStack,
		&skills,
		&workExperience.StartedAt,
		&workExperience.EndedAt,
		&workExperience.Current,
		&workExperience.Featured,
		&workExperience.SortOrder,
		&workExperience.Status,
		&workExperience.PublishedAt,
		&workExperience.CreatedAt,
		&workExperience.UpdatedAt,
	); err != nil {
		return WorkExperience{}, err
	}

	if len(highlights) > 0 {
		if err := json.Unmarshal(highlights, &workExperience.Highlights); err != nil {
			return WorkExperience{}, err
		}
	}
	if len(responsibilities) > 0 {
		if err := json.Unmarshal(responsibilities, &workExperience.Responsibilities); err != nil {
			return WorkExperience{}, err
		}
	}
	if len(techStack) > 0 {
		if err := json.Unmarshal(techStack, &workExperience.TechStack); err != nil {
			return WorkExperience{}, err
		}
	}
	if len(skills) > 0 {
		if err := json.Unmarshal(skills, &workExperience.Skills); err != nil {
			return WorkExperience{}, err
		}
	}

	return workExperience, nil
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
CREATE TABLE IF NOT EXISTS work_experiences (
	id TEXT PRIMARY KEY,
	slug TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	company TEXT NOT NULL,
	company_url TEXT NOT NULL DEFAULT '',
	company_logo_url TEXT NOT NULL DEFAULT '',
	employment_type TEXT NOT NULL DEFAULT '',
	location TEXT NOT NULL DEFAULT '',
	location_type TEXT NOT NULL DEFAULT '',
	summary TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	highlights JSONB NOT NULL DEFAULT '[]'::jsonb,
	responsibilities JSONB NOT NULL DEFAULT '[]'::jsonb,
	tech_stack JSONB NOT NULL DEFAULT '[]'::jsonb,
	skills JSONB NOT NULL DEFAULT '[]'::jsonb,
	started_at TIMESTAMPTZ NOT NULL,
	ended_at TIMESTAMPTZ,
	is_current BOOLEAN NOT NULL DEFAULT FALSE,
	featured BOOLEAN NOT NULL DEFAULT FALSE,
	sort_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	published_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	CHECK (ended_at IS NULL OR ended_at >= started_at),
	CHECK (is_current = FALSE OR ended_at IS NULL)
);

CREATE INDEX IF NOT EXISTS work_experiences_status_sort_order
	ON work_experiences(status, sort_order ASC, is_current DESC, started_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS work_experiences_featured_sort_order
	ON work_experiences(featured, sort_order ASC, is_current DESC, started_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS work_experiences_current_sort_order
	ON work_experiences(is_current, sort_order ASC, started_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS work_experiences_sort_order_started_at
	ON work_experiences(sort_order ASC, is_current DESC, started_at DESC, created_at DESC);
`
