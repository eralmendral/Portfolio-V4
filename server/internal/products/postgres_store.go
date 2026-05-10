package products

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

const sectionID = "products"

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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Product, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 5)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("p.status = $%d", len(args)))
	}
	if filter.Featured != nil {
		args = append(args, *filter.Featured)
		where = append(where, fmt.Sprintf("p.featured = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Category) != "" {
		args = append(args, strings.ToLower(strings.TrimSpace(filter.Category)))
		where = append(where, fmt.Sprintf("lower(p.category) = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Tag) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Tag))+"%")
		where = append(where, fmt.Sprintf("lower(p.tags::text) LIKE $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(p.slug) LIKE $%[1]d OR
			lower(p.title) LIKE $%[1]d OR
			lower(p.summary) LIKE $%[1]d OR
			lower(p.description) LIKE $%[1]d OR
			lower(p.price_label) LIKE $%[1]d OR
			lower(p.cta_label) LIKE $%[1]d OR
			lower(p.category) LIKE $%[1]d OR
			lower(p.tags::text) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT p.id, p.slug, p.title, p.summary, p.description,
			p.cover_image_url, p.price_label, p.cta_label, p.cta_url,
			p.category, p.tags, p.featured, p.sort_order, p.status,
			p.published_at, p.created_at, p.updated_at
		FROM products p
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY p.featured DESC, p.sort_order ASC, p.published_at DESC NULLS LAST, p.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func (s *PostgresStore) Get(ctx context.Context, idOrSlug string) (Product, error) {
	return s.get(ctx, s.db, idOrSlug)
}

func (s *PostgresStore) Create(ctx context.Context, product Product) (Product, error) {
	now := s.now().UTC()
	if product.ID == "" {
		product.ID = newID()
	}
	if product.Status == "" {
		product.Status = StatusDraft
	}
	if product.CreatedAt.IsZero() {
		product.CreatedAt = now
	}
	product.UpdatedAt = now

	if err := upsertProduct(ctx, s.db, product); err != nil {
		return Product{}, normalizePostgresError(err)
	}

	return product, nil
}

func (s *PostgresStore) Update(ctx context.Context, idOrSlug string, mutate func(*Product) error) (Product, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Product{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, idOrSlug)
	if err != nil {
		return Product{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Product{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertProduct(ctx, tx, next); err != nil {
		return Product{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Product{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, idOrSlug string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM products
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

func (s *PostgresStore) GetSection(ctx context.Context) (ProductSectionSettings, error) {
	return s.getSection(ctx, s.db)
}

func (s *PostgresStore) UpdateSection(ctx context.Context, mutate func(*ProductSectionSettings) error) (ProductSectionSettings, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProductSectionSettings{}, err
	}
	defer rollback(tx)

	current, err := s.getSectionForUpdate(ctx, tx)
	if err != nil {
		return ProductSectionSettings{}, err
	}
	next := current
	if err := mutate(&next); err != nil {
		return ProductSectionSettings{}, err
	}
	next.UpdatedAt = s.now().UTC()

	if err := upsertSection(ctx, tx, next); err != nil {
		return ProductSectionSettings{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProductSectionSettings{}, err
	}

	return next, nil
}

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, idOrSlug string) (Product, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT p.id, p.slug, p.title, p.summary, p.description,
			p.cover_image_url, p.price_label, p.cta_label, p.cta_url,
			p.category, p.tags, p.featured, p.sort_order, p.status,
			p.published_at, p.created_at, p.updated_at
		FROM products p
		WHERE p.id = $1 OR p.slug = $1
	`, idOrSlug)

	product, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return product, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, idOrSlug string) (Product, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT p.id, p.slug, p.title, p.summary, p.description,
			p.cover_image_url, p.price_label, p.cta_label, p.cta_url,
			p.category, p.tags, p.featured, p.sort_order, p.status,
			p.published_at, p.created_at, p.updated_at
		FROM products p
		WHERE p.id = $1 OR p.slug = $1
		FOR UPDATE
	`, idOrSlug)

	product, err := scanProduct(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return product, err
}

func (s *PostgresStore) getSection(ctx context.Context, runner dbRunner) (ProductSectionSettings, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT enabled, title, description, updated_at
		FROM product_section_settings
		WHERE id = $1
	`, sectionID)
	return scanSection(row)
}

func (s *PostgresStore) getSectionForUpdate(ctx context.Context, runner dbRunner) (ProductSectionSettings, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT enabled, title, description, updated_at
		FROM product_section_settings
		WHERE id = $1
		FOR UPDATE
	`, sectionID)
	return scanSection(row)
}

func upsertProduct(ctx context.Context, runner dbRunner, product Product) error {
	tags, err := json.Marshal(product.Tags)
	if err != nil {
		return err
	}

	_, err = runner.ExecContext(ctx, `
		INSERT INTO products (
			id, slug, title, summary, description, cover_image_url,
			price_label, cta_label, cta_url, category, tags, featured,
			sort_order, status, published_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11::jsonb, $12, $13, $14, $15, $16, $17
		)
		ON CONFLICT (id) DO UPDATE SET
			slug = EXCLUDED.slug,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			description = EXCLUDED.description,
			cover_image_url = EXCLUDED.cover_image_url,
			price_label = EXCLUDED.price_label,
			cta_label = EXCLUDED.cta_label,
			cta_url = EXCLUDED.cta_url,
			category = EXCLUDED.category,
			tags = EXCLUDED.tags,
			featured = EXCLUDED.featured,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status,
			published_at = EXCLUDED.published_at,
			updated_at = EXCLUDED.updated_at
	`, product.ID, product.Slug, product.Title, product.Summary, product.Description,
		product.CoverImageURL, product.PriceLabel, product.CTALabel, product.CTAURL,
		product.Category, string(tags), product.Featured, product.SortOrder,
		product.Status, product.PublishedAt, product.CreatedAt, product.UpdatedAt)
	return err
}

func upsertSection(ctx context.Context, runner dbRunner, settings ProductSectionSettings) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO product_section_settings (id, enabled, title, description, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET
			enabled = EXCLUDED.enabled,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at
	`, sectionID, settings.Enabled, settings.Title, settings.Description, settings.UpdatedAt)
	return err
}

type productScanner interface {
	Scan(...any) error
}

func scanProduct(scanner productScanner) (Product, error) {
	var product Product
	var tags []byte
	if err := scanner.Scan(
		&product.ID,
		&product.Slug,
		&product.Title,
		&product.Summary,
		&product.Description,
		&product.CoverImageURL,
		&product.PriceLabel,
		&product.CTALabel,
		&product.CTAURL,
		&product.Category,
		&tags,
		&product.Featured,
		&product.SortOrder,
		&product.Status,
		&product.PublishedAt,
		&product.CreatedAt,
		&product.UpdatedAt,
	); err != nil {
		return Product{}, err
	}

	if len(tags) > 0 {
		if err := json.Unmarshal(tags, &product.Tags); err != nil {
			return Product{}, err
		}
	}

	return product, nil
}

func scanSection(scanner productScanner) (ProductSectionSettings, error) {
	var settings ProductSectionSettings
	err := scanner.Scan(
		&settings.Enabled,
		&settings.Title,
		&settings.Description,
		&settings.UpdatedAt,
	)
	return settings, err
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
CREATE TABLE IF NOT EXISTS products (
	id TEXT PRIMARY KEY,
	slug TEXT NOT NULL UNIQUE,
	title TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	cover_image_url TEXT NOT NULL DEFAULT '',
	price_label TEXT NOT NULL DEFAULT '',
	cta_label TEXT NOT NULL DEFAULT '',
	cta_url TEXT NOT NULL,
	category TEXT NOT NULL DEFAULT '',
	tags JSONB NOT NULL DEFAULT '[]'::jsonb,
	featured BOOLEAN NOT NULL DEFAULT FALSE,
	sort_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	published_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS products_status_sort_order
	ON products(status, featured DESC, sort_order ASC, published_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS products_category_sort_order
	ON products(category ASC, featured DESC, sort_order ASC, published_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS products_featured_sort_order
	ON products(featured, sort_order ASC, published_at DESC, created_at DESC);

CREATE TABLE IF NOT EXISTS product_section_settings (
	id TEXT PRIMARY KEY,
	enabled BOOLEAN NOT NULL DEFAULT FALSE,
	title TEXT NOT NULL DEFAULT 'Products',
	description TEXT NOT NULL DEFAULT '',
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO product_section_settings (id, enabled, title, description)
VALUES ('products', FALSE, 'Products', '')
ON CONFLICT (id) DO NOTHING;
`
