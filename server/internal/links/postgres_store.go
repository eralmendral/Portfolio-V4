package links

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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Link, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 3)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("l.status = $%d", len(args)))
	}
	if filter.Star != nil {
		args = append(args, *filter.Star)
		where = append(where, fmt.Sprintf("l.star = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(l.label) LIKE $%[1]d OR
			lower(l.url) LIKE $%[1]d OR
			lower(l.icon_class) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT l.id, l.label, l.url, l.icon_class, l.sort_order,
			l.star, l.status, l.created_at, l.updated_at
		FROM links l
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY l.sort_order ASC, l.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := make([]Link, 0)
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return links, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Link, error) {
	return s.get(ctx, s.db, id)
}

func (s *PostgresStore) Create(ctx context.Context, link Link) (Link, error) {
	now := s.now().UTC()
	if link.ID == "" {
		link.ID = newID()
	}
	if link.Status == "" {
		link.Status = StatusDraft
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = now
	}
	link.UpdatedAt = now

	if err := upsertLink(ctx, s.db, link); err != nil {
		return Link{}, normalizePostgresError(err)
	}

	return link, nil
}

func (s *PostgresStore) Update(ctx context.Context, id string, mutate func(*Link) error) (Link, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Link{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, id)
	if err != nil {
		return Link{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Link{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertLink(ctx, tx, next); err != nil {
		return Link{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Link{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM links
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, id string) (Link, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT l.id, l.label, l.url, l.icon_class, l.sort_order,
			l.star, l.status, l.created_at, l.updated_at
		FROM links l
		WHERE l.id = $1
	`, id)

	link, err := scanLink(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	return link, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, id string) (Link, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT l.id, l.label, l.url, l.icon_class, l.sort_order,
			l.star, l.status, l.created_at, l.updated_at
		FROM links l
		WHERE l.id = $1
		FOR UPDATE
	`, id)

	link, err := scanLink(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	return link, err
}

func upsertLink(ctx context.Context, runner dbRunner, link Link) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO links (
			id, label, url, icon_class, sort_order, star, status,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			label = EXCLUDED.label,
			url = EXCLUDED.url,
			icon_class = EXCLUDED.icon_class,
			sort_order = EXCLUDED.sort_order,
			star = EXCLUDED.star,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, link.ID, link.Label, link.URL, link.IconClass, link.SortOrder, link.Star,
		link.Status, link.CreatedAt, link.UpdatedAt)
	return err
}

type linkScanner interface {
	Scan(...any) error
}

func scanLink(scanner linkScanner) (Link, error) {
	var link Link
	if err := scanner.Scan(
		&link.ID,
		&link.Label,
		&link.URL,
		&link.IconClass,
		&link.SortOrder,
		&link.Star,
		&link.Status,
		&link.CreatedAt,
		&link.UpdatedAt,
	); err != nil {
		return Link{}, err
	}

	return link, nil
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
CREATE TABLE IF NOT EXISTS links (
	id TEXT PRIMARY KEY,
	label TEXT NOT NULL,
	url TEXT NOT NULL,
	icon_class TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	star BOOLEAN NOT NULL DEFAULT FALSE,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS links_status_sort_order
	ON links(status, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS links_star_sort_order
	ON links(star, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS links_sort_order_created_at
	ON links(sort_order ASC, created_at DESC);
`
