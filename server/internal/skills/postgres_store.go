package skills

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

func (s *PostgresStore) ListCategories(ctx context.Context, filter CategoryListFilter) ([]SkillCategory, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 2)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("c.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(c.slug) LIKE $%[1]d OR
			lower(c.name) LIKE $%[1]d OR
			lower(c.description) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT c.id, c.slug, c.name, c.description, c.icon_class,
			c.sort_order, c.status, c.created_at, c.updated_at
		FROM skill_categories c
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY c.sort_order ASC, c.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := make([]SkillCategory, 0)
	for rows.Next() {
		category, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

func (s *PostgresStore) GetCategory(ctx context.Context, idOrSlug string) (SkillCategory, error) {
	return s.getCategory(ctx, s.db, idOrSlug)
}

func (s *PostgresStore) CreateCategory(ctx context.Context, category SkillCategory) (SkillCategory, error) {
	now := s.now().UTC()
	if category.ID == "" {
		category.ID = newID()
	}
	if category.Status == "" {
		category.Status = StatusDraft
	}
	if category.CreatedAt.IsZero() {
		category.CreatedAt = now
	}
	category.UpdatedAt = now

	if err := upsertCategory(ctx, s.db, category); err != nil {
		return SkillCategory{}, normalizePostgresError(err)
	}

	return category, nil
}

func (s *PostgresStore) UpdateCategory(ctx context.Context, idOrSlug string, mutate func(*SkillCategory) error) (SkillCategory, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SkillCategory{}, err
	}
	defer rollback(tx)

	current, err := s.getCategoryForUpdate(ctx, tx, idOrSlug)
	if err != nil {
		return SkillCategory{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return SkillCategory{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertCategory(ctx, tx, next); err != nil {
		return SkillCategory{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return SkillCategory{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) DeleteCategory(ctx context.Context, idOrSlug string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM skill_categories
		WHERE id = $1 OR slug = $1
	`, idOrSlug)
	if err != nil {
		return normalizeCategoryDeleteError(err)
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

func (s *PostgresStore) ListSkills(ctx context.Context, filter SkillListFilter) ([]Skill, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 5)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("s.status = $%d", len(args)))
	}
	if filter.Featured != nil {
		args = append(args, *filter.Featured)
		where = append(where, fmt.Sprintf("s.featured = $%d", len(args)))
	}
	if strings.TrimSpace(filter.CategoryID) != "" {
		args = append(args, strings.TrimSpace(filter.CategoryID))
		where = append(where, fmt.Sprintf("s.category_id = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Category) != "" {
		args = append(args, strings.ToLower(strings.TrimSpace(filter.Category)))
		where = append(where, fmt.Sprintf("(lower(c.id) = $%[1]d OR lower(c.slug) = $%[1]d OR lower(c.name) = $%[1]d)", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(s.name) LIKE $%[1]d OR
			lower(s.summary) LIKE $%[1]d OR
			lower(c.name) LIKE $%[1]d OR
			lower(c.description) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT s.id, s.category_id, s.name, s.summary, s.icon_class,
			s.sort_order, s.featured, s.status, s.created_at, s.updated_at
		FROM skills s
		JOIN skill_categories c ON c.id = s.category_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY c.sort_order ASC, s.featured DESC, s.sort_order ASC, s.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	skills := make([]Skill, 0)
	for rows.Next() {
		skill, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		skills = append(skills, skill)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return skills, nil
}

func (s *PostgresStore) GetSkill(ctx context.Context, id string) (Skill, error) {
	return s.getSkill(ctx, s.db, id)
}

func (s *PostgresStore) CreateSkill(ctx context.Context, skill Skill) (Skill, error) {
	now := s.now().UTC()
	if skill.ID == "" {
		skill.ID = newID()
	}
	if skill.Status == "" {
		skill.Status = StatusDraft
	}
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	skill.UpdatedAt = now

	if err := upsertSkill(ctx, s.db, skill); err != nil {
		return Skill{}, normalizePostgresError(err)
	}

	return skill, nil
}

func (s *PostgresStore) UpdateSkill(ctx context.Context, id string, mutate func(*Skill) error) (Skill, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Skill{}, err
	}
	defer rollback(tx)

	current, err := s.getSkillForUpdate(ctx, tx, id)
	if err != nil {
		return Skill{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Skill{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertSkill(ctx, tx, next); err != nil {
		return Skill{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Skill{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) DeleteSkill(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM skills
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

func (s *PostgresStore) getCategory(ctx context.Context, runner dbRunner, idOrSlug string) (SkillCategory, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT c.id, c.slug, c.name, c.description, c.icon_class,
			c.sort_order, c.status, c.created_at, c.updated_at
		FROM skill_categories c
		WHERE c.id = $1 OR c.slug = $1
	`, idOrSlug)

	category, err := scanCategory(row)
	if errors.Is(err, sql.ErrNoRows) {
		return SkillCategory{}, ErrNotFound
	}
	return category, err
}

func (s *PostgresStore) getCategoryForUpdate(ctx context.Context, runner dbRunner, idOrSlug string) (SkillCategory, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT c.id, c.slug, c.name, c.description, c.icon_class,
			c.sort_order, c.status, c.created_at, c.updated_at
		FROM skill_categories c
		WHERE c.id = $1 OR c.slug = $1
		FOR UPDATE
	`, idOrSlug)

	category, err := scanCategory(row)
	if errors.Is(err, sql.ErrNoRows) {
		return SkillCategory{}, ErrNotFound
	}
	return category, err
}

func (s *PostgresStore) getSkill(ctx context.Context, runner dbRunner, id string) (Skill, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT s.id, s.category_id, s.name, s.summary, s.icon_class,
			s.sort_order, s.featured, s.status, s.created_at, s.updated_at
		FROM skills s
		WHERE s.id = $1
	`, id)

	skill, err := scanSkill(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Skill{}, ErrNotFound
	}
	return skill, err
}

func (s *PostgresStore) getSkillForUpdate(ctx context.Context, runner dbRunner, id string) (Skill, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT s.id, s.category_id, s.name, s.summary, s.icon_class,
			s.sort_order, s.featured, s.status, s.created_at, s.updated_at
		FROM skills s
		WHERE s.id = $1
		FOR UPDATE
	`, id)

	skill, err := scanSkill(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Skill{}, ErrNotFound
	}
	return skill, err
}

func upsertCategory(ctx context.Context, runner dbRunner, category SkillCategory) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO skill_categories (
			id, slug, name, description, icon_class, sort_order, status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			slug = EXCLUDED.slug,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			icon_class = EXCLUDED.icon_class,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, category.ID, category.Slug, category.Name, category.Description, category.IconClass,
		category.SortOrder, category.Status, category.CreatedAt, category.UpdatedAt)
	return err
}

func upsertSkill(ctx context.Context, runner dbRunner, skill Skill) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO skills (
			id, category_id, name, summary, icon_class, sort_order, featured,
			status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			category_id = EXCLUDED.category_id,
			name = EXCLUDED.name,
			summary = EXCLUDED.summary,
			icon_class = EXCLUDED.icon_class,
			sort_order = EXCLUDED.sort_order,
			featured = EXCLUDED.featured,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, skill.ID, skill.CategoryID, skill.Name, skill.Summary, skill.IconClass,
		skill.SortOrder, skill.Featured, skill.Status, skill.CreatedAt, skill.UpdatedAt)
	return err
}

type categoryScanner interface {
	Scan(...any) error
}

func scanCategory(scanner categoryScanner) (SkillCategory, error) {
	var category SkillCategory
	if err := scanner.Scan(
		&category.ID,
		&category.Slug,
		&category.Name,
		&category.Description,
		&category.IconClass,
		&category.SortOrder,
		&category.Status,
		&category.CreatedAt,
		&category.UpdatedAt,
	); err != nil {
		return SkillCategory{}, err
	}

	return category, nil
}

type skillScanner interface {
	Scan(...any) error
}

func scanSkill(scanner skillScanner) (Skill, error) {
	var skill Skill
	if err := scanner.Scan(
		&skill.ID,
		&skill.CategoryID,
		&skill.Name,
		&skill.Summary,
		&skill.IconClass,
		&skill.SortOrder,
		&skill.Featured,
		&skill.Status,
		&skill.CreatedAt,
		&skill.UpdatedAt,
	); err != nil {
		return Skill{}, err
	}

	return skill, nil
}

func normalizePostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrInvalidCategory
		}
	}
	return err
}

func normalizeCategoryDeleteError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrCategoryInUse
	}
	return err
}

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

const postgresSchema = `
CREATE TABLE IF NOT EXISTS skill_categories (
	id TEXT PRIMARY KEY,
	slug TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	icon_class TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS skills (
	id TEXT PRIMARY KEY,
	category_id TEXT NOT NULL REFERENCES skill_categories(id),
	name TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	icon_class TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	featured BOOLEAN NOT NULL DEFAULT FALSE,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE (category_id, name)
);

CREATE INDEX IF NOT EXISTS skill_categories_status_sort_order
	ON skill_categories(status, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS skill_categories_sort_order_created_at
	ON skill_categories(sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS skills_category_sort_order
	ON skills(category_id, featured DESC, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS skills_status_sort_order
	ON skills(status, featured DESC, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS skills_featured_sort_order
	ON skills(featured, sort_order ASC, created_at DESC);
`
