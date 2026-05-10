package projects

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Project, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 3)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("p.status = $%d", len(args)))
	}
	if filter.Featured != nil {
		args = append(args, *filter.Featured)
		where = append(where, fmt.Sprintf("p.featured = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(p.title) LIKE $%[1]d OR
			lower(p.slug) LIKE $%[1]d OR
			lower(p.summary) LIKE $%[1]d OR
			lower(p.description) LIKE $%[1]d OR
			lower(p.tech_stack::text) LIKE $%[1]d OR
			lower(p.tags::text) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.slug, p.title, p.summary, p.description, p.body,
			p.tech_stack, p.tags, p.github_url, p.demo_url, p.featured,
			p.status, p.created_at, p.updated_at, p.published_at
		FROM projects p
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY p.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	projects := make([]Project, 0)
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range projects {
		if err := s.loadImages(ctx, s.db, &projects[i]); err != nil {
			return nil, err
		}
	}

	return projects, nil
}

func (s *PostgresStore) Get(ctx context.Context, idOrSlug string) (Project, error) {
	project, err := s.get(ctx, s.db, idOrSlug)
	if err != nil {
		return Project{}, err
	}
	return project, s.loadImages(ctx, s.db, &project)
}

func (s *PostgresStore) Create(ctx context.Context, project Project) (Project, error) {
	now := s.now().UTC()
	if project.ID == "" {
		project.ID = newID()
	}
	if project.Status == "" {
		project.Status = StatusDraft
	}
	if project.CreatedAt.IsZero() {
		project.CreatedAt = now
	}
	project.UpdatedAt = now

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, err
	}
	defer rollback(tx)

	if err := upsertProject(ctx, tx, project); err != nil {
		return Project{}, normalizePostgresError(err)
	}
	if err := replaceImages(ctx, tx, project); err != nil {
		return Project{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Project{}, normalizePostgresError(err)
	}

	return project, nil
}

func (s *PostgresStore) Update(ctx context.Context, idOrSlug string, mutate func(*Project) error) (Project, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Project{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, idOrSlug)
	if err != nil {
		return Project{}, err
	}
	if err := s.loadImages(ctx, tx, &current); err != nil {
		return Project{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Project{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertProject(ctx, tx, next); err != nil {
		return Project{}, normalizePostgresError(err)
	}
	if err := replaceImages(ctx, tx, next); err != nil {
		return Project{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Project{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, idOrSlug string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM projects
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, idOrSlug string) (Project, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT p.id, p.slug, p.title, p.summary, p.description, p.body,
			p.tech_stack, p.tags, p.github_url, p.demo_url, p.featured,
			p.status, p.created_at, p.updated_at, p.published_at
		FROM projects p
		WHERE p.id = $1 OR p.slug = $1
	`, idOrSlug)

	project, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return project, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, idOrSlug string) (Project, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT p.id, p.slug, p.title, p.summary, p.description, p.body,
			p.tech_stack, p.tags, p.github_url, p.demo_url, p.featured,
			p.status, p.created_at, p.updated_at, p.published_at
		FROM projects p
		WHERE p.id = $1 OR p.slug = $1
		FOR UPDATE
	`, idOrSlug)

	project, err := scanProject(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return project, err
}

func (s *PostgresStore) loadImages(ctx context.Context, runner dbRunner, project *Project) error {
	rows, err := runner.QueryContext(ctx, `
		SELECT id, role, url, path, alt_text, caption, content_type,
			size_bytes, width, height, sort_order, uploaded_at
		FROM project_images
		WHERE project_id = $1
		ORDER BY role DESC, sort_order ASC, uploaded_at ASC
	`, project.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	project.MainImage = nil
	project.Images = nil
	for rows.Next() {
		var role string
		var image ProjectImage
		if err := rows.Scan(
			&image.ID,
			&role,
			&image.URL,
			&image.Path,
			&image.AltText,
			&image.Caption,
			&image.ContentType,
			&image.SizeBytes,
			&image.Width,
			&image.Height,
			&image.SortOrder,
			&image.UploadedAt,
		); err != nil {
			return err
		}

		if role == "main" {
			project.MainImage = &image
			continue
		}
		project.Images = append(project.Images, image)
	}

	return rows.Err()
}

func upsertProject(ctx context.Context, runner dbRunner, project Project) error {
	techStack, err := json.Marshal(project.TechStack)
	if err != nil {
		return err
	}
	tags, err := json.Marshal(project.Tags)
	if err != nil {
		return err
	}

	_, err = runner.ExecContext(ctx, `
		INSERT INTO projects (
			id, slug, title, summary, description, body, tech_stack, tags,
			github_url, demo_url, featured, status, created_at, updated_at,
			published_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb,
			$9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT (id) DO UPDATE SET
			slug = EXCLUDED.slug,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			description = EXCLUDED.description,
			body = EXCLUDED.body,
			tech_stack = EXCLUDED.tech_stack,
			tags = EXCLUDED.tags,
			github_url = EXCLUDED.github_url,
			demo_url = EXCLUDED.demo_url,
			featured = EXCLUDED.featured,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at,
			published_at = EXCLUDED.published_at
	`, project.ID, project.Slug, project.Title, project.Summary, project.Description, project.Body,
		string(techStack), string(tags), project.GitHubURL, project.DemoURL, project.Featured,
		project.Status, project.CreatedAt, project.UpdatedAt, project.PublishedAt)
	return err
}

func replaceImages(ctx context.Context, runner dbRunner, project Project) error {
	if _, err := runner.ExecContext(ctx, `
		DELETE FROM project_images
		WHERE project_id = $1
	`, project.ID); err != nil {
		return err
	}

	if project.MainImage != nil {
		if err := insertImage(ctx, runner, project.ID, "main", *project.MainImage); err != nil {
			return err
		}
	}
	for _, image := range project.Images {
		if err := insertImage(ctx, runner, project.ID, "gallery", image); err != nil {
			return err
		}
	}

	return nil
}

func insertImage(ctx context.Context, runner dbRunner, projectID string, role string, image ProjectImage) error {
	if image.ID == "" {
		image.ID = newID()
	}
	if image.UploadedAt.IsZero() {
		image.UploadedAt = time.Now().UTC()
	}

	_, err := runner.ExecContext(ctx, `
		INSERT INTO project_images (
			id, project_id, role, url, path, alt_text, caption, content_type,
			size_bytes, width, height, sort_order, uploaded_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, image.ID, projectID, role, image.URL, image.Path, image.AltText, image.Caption,
		image.ContentType, image.SizeBytes, image.Width, image.Height, image.SortOrder,
		image.UploadedAt)
	return err
}

type projectScanner interface {
	Scan(...any) error
}

func scanProject(scanner projectScanner) (Project, error) {
	var project Project
	var techStack []byte
	var tags []byte
	if err := scanner.Scan(
		&project.ID,
		&project.Slug,
		&project.Title,
		&project.Summary,
		&project.Description,
		&project.Body,
		&techStack,
		&tags,
		&project.GitHubURL,
		&project.DemoURL,
		&project.Featured,
		&project.Status,
		&project.CreatedAt,
		&project.UpdatedAt,
		&project.PublishedAt,
	); err != nil {
		return Project{}, err
	}

	if len(techStack) > 0 {
		if err := json.Unmarshal(techStack, &project.TechStack); err != nil {
			return Project{}, err
		}
	}
	if len(tags) > 0 {
		if err := json.Unmarshal(tags, &project.Tags); err != nil {
			return Project{}, err
		}
	}

	return project, nil
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
CREATE TABLE IF NOT EXISTS projects (
	id TEXT PRIMARY KEY,
	slug TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	body TEXT NOT NULL DEFAULT '',
	tech_stack JSONB NOT NULL DEFAULT '[]'::jsonb,
	tags JSONB NOT NULL DEFAULT '[]'::jsonb,
	github_url TEXT NOT NULL DEFAULT '',
	demo_url TEXT NOT NULL DEFAULT '',
	featured BOOLEAN NOT NULL DEFAULT FALSE,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	published_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS project_images (
	id TEXT PRIMARY KEY,
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	role TEXT NOT NULL CHECK (role IN ('main', 'gallery')),
	url TEXT NOT NULL,
	path TEXT NOT NULL DEFAULT '',
	alt_text TEXT NOT NULL DEFAULT '',
	caption TEXT NOT NULL DEFAULT '',
	content_type TEXT NOT NULL DEFAULT '',
	size_bytes BIGINT NOT NULL DEFAULT 0,
	width INTEGER NOT NULL DEFAULT 0,
	height INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0,
	uploaded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS project_images_one_main_per_project
	ON project_images(project_id)
	WHERE role = 'main';

CREATE INDEX IF NOT EXISTS project_images_project_order
	ON project_images(project_id, role, sort_order, uploaded_at);

CREATE INDEX IF NOT EXISTS projects_status_created_at
	ON projects(status, created_at DESC);

CREATE INDEX IF NOT EXISTS projects_featured_created_at
	ON projects(featured, created_at DESC);
`
