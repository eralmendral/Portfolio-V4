package articles

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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Article, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 3)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("a.status = $%d", len(args)))
	}
	if filter.Featured != nil {
		args = append(args, *filter.Featured)
		where = append(where, fmt.Sprintf("a.featured = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(a.title) LIKE $%[1]d OR
			lower(a.url) LIKE $%[1]d OR
			lower(a.source) LIKE $%[1]d OR
			lower(a.summary) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.title, a.url, a.source, a.summary, a.cover_image_url,
			a.featured, a.sort_order, a.status, a.published_at, a.created_at,
			a.updated_at
		FROM articles a
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY a.sort_order ASC, a.published_at DESC NULLS LAST, a.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	articles := make([]Article, 0)
	for rows.Next() {
		article, err := scanArticle(rows)
		if err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return articles, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Article, error) {
	return s.get(ctx, s.db, id)
}

func (s *PostgresStore) Create(ctx context.Context, article Article) (Article, error) {
	now := s.now().UTC()
	if article.ID == "" {
		article.ID = newID()
	}
	if article.Status == "" {
		article.Status = StatusDraft
	}
	if article.CreatedAt.IsZero() {
		article.CreatedAt = now
	}
	article.UpdatedAt = now

	if err := upsertArticle(ctx, s.db, article); err != nil {
		return Article{}, normalizePostgresError(err)
	}

	return article, nil
}

func (s *PostgresStore) Update(ctx context.Context, id string, mutate func(*Article) error) (Article, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Article{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, id)
	if err != nil {
		return Article{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Article{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertArticle(ctx, tx, next); err != nil {
		return Article{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Article{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM articles
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, id string) (Article, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT a.id, a.title, a.url, a.source, a.summary, a.cover_image_url,
			a.featured, a.sort_order, a.status, a.published_at, a.created_at,
			a.updated_at
		FROM articles a
		WHERE a.id = $1
	`, id)

	article, err := scanArticle(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Article{}, ErrNotFound
	}
	return article, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, id string) (Article, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT a.id, a.title, a.url, a.source, a.summary, a.cover_image_url,
			a.featured, a.sort_order, a.status, a.published_at, a.created_at,
			a.updated_at
		FROM articles a
		WHERE a.id = $1
		FOR UPDATE
	`, id)

	article, err := scanArticle(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Article{}, ErrNotFound
	}
	return article, err
}

func upsertArticle(ctx context.Context, runner dbRunner, article Article) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO articles (
			id, title, url, source, summary, cover_image_url, featured,
			sort_order, status, published_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12
		)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			url = EXCLUDED.url,
			source = EXCLUDED.source,
			summary = EXCLUDED.summary,
			cover_image_url = EXCLUDED.cover_image_url,
			featured = EXCLUDED.featured,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status,
			published_at = EXCLUDED.published_at,
			updated_at = EXCLUDED.updated_at
	`, article.ID, article.Title, article.URL, article.Source, article.Summary,
		article.CoverImageURL, article.Featured, article.SortOrder, article.Status,
		article.PublishedAt, article.CreatedAt, article.UpdatedAt)
	return err
}

type articleScanner interface {
	Scan(...any) error
}

func scanArticle(scanner articleScanner) (Article, error) {
	var article Article
	if err := scanner.Scan(
		&article.ID,
		&article.Title,
		&article.URL,
		&article.Source,
		&article.Summary,
		&article.CoverImageURL,
		&article.Featured,
		&article.SortOrder,
		&article.Status,
		&article.PublishedAt,
		&article.CreatedAt,
		&article.UpdatedAt,
	); err != nil {
		return Article{}, err
	}

	return article, nil
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
CREATE TABLE IF NOT EXISTS articles (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	url TEXT NOT NULL UNIQUE,
	source TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	cover_image_url TEXT NOT NULL DEFAULT '',
	featured BOOLEAN NOT NULL DEFAULT FALSE,
	sort_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	published_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS articles_status_sort_order
	ON articles(status, sort_order ASC, published_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS articles_featured_sort_order
	ON articles(featured, sort_order ASC, published_at DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS articles_sort_order_published_at
	ON articles(sort_order ASC, published_at DESC, created_at DESC);
`
