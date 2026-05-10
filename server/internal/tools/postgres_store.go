package tools

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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Tool, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 5)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("t.status = $%d", len(args)))
	}
	if filter.Featured != nil {
		args = append(args, *filter.Featured)
		where = append(where, fmt.Sprintf("t.featured = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Category) != "" {
		args = append(args, strings.ToLower(strings.TrimSpace(filter.Category)))
		where = append(where, fmt.Sprintf("lower(t.category) = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Tag) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Tag))+"%")
		where = append(where, fmt.Sprintf("lower(t.tags::text) LIKE $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(t.name) LIKE $%[1]d OR
			lower(t.category) LIKE $%[1]d OR
			lower(t.summary) LIKE $%[1]d OR
			lower(t.tags::text) LIKE $%[1]d
		)`, len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.category, t.summary, t.icon_class,
			t.tags, t.sort_order, t.featured, t.status, t.created_at,
			t.updated_at
		FROM tools t
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY t.category ASC, t.featured DESC, t.sort_order ASC, t.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tools := make([]Tool, 0)
	for rows.Next() {
		tool, err := scanTool(rows)
		if err != nil {
			return nil, err
		}
		tools = append(tools, tool)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tools, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Tool, error) {
	return s.get(ctx, s.db, id)
}

func (s *PostgresStore) Create(ctx context.Context, tool Tool) (Tool, error) {
	now := s.now().UTC()
	if tool.ID == "" {
		tool.ID = newID()
	}
	if tool.Status == "" {
		tool.Status = StatusDraft
	}
	if tool.CreatedAt.IsZero() {
		tool.CreatedAt = now
	}
	tool.UpdatedAt = now

	if err := upsertTool(ctx, s.db, tool); err != nil {
		return Tool{}, normalizePostgresError(err)
	}

	return tool, nil
}

func (s *PostgresStore) Update(ctx context.Context, id string, mutate func(*Tool) error) (Tool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Tool{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, id)
	if err != nil {
		return Tool{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Tool{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertTool(ctx, tx, next); err != nil {
		return Tool{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Tool{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM tools
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, id string) (Tool, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT t.id, t.name, t.category, t.summary, t.icon_class,
			t.tags, t.sort_order, t.featured, t.status, t.created_at,
			t.updated_at
		FROM tools t
		WHERE t.id = $1
	`, id)

	tool, err := scanTool(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Tool{}, ErrNotFound
	}
	return tool, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, id string) (Tool, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT t.id, t.name, t.category, t.summary, t.icon_class,
			t.tags, t.sort_order, t.featured, t.status, t.created_at,
			t.updated_at
		FROM tools t
		WHERE t.id = $1
		FOR UPDATE
	`, id)

	tool, err := scanTool(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Tool{}, ErrNotFound
	}
	return tool, err
}

func upsertTool(ctx context.Context, runner dbRunner, tool Tool) error {
	tags, err := json.Marshal(tool.Tags)
	if err != nil {
		return err
	}

	_, err = runner.ExecContext(ctx, `
		INSERT INTO tools (
			id, name, category, summary, icon_class, tags, sort_order,
			featured, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			category = EXCLUDED.category,
			summary = EXCLUDED.summary,
			icon_class = EXCLUDED.icon_class,
			tags = EXCLUDED.tags,
			sort_order = EXCLUDED.sort_order,
			featured = EXCLUDED.featured,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, tool.ID, tool.Name, tool.Category, tool.Summary, tool.IconClass, string(tags),
		tool.SortOrder, tool.Featured, tool.Status, tool.CreatedAt, tool.UpdatedAt)
	return err
}

type toolScanner interface {
	Scan(...any) error
}

func scanTool(scanner toolScanner) (Tool, error) {
	var tool Tool
	var tags []byte
	if err := scanner.Scan(
		&tool.ID,
		&tool.Name,
		&tool.Category,
		&tool.Summary,
		&tool.IconClass,
		&tags,
		&tool.SortOrder,
		&tool.Featured,
		&tool.Status,
		&tool.CreatedAt,
		&tool.UpdatedAt,
	); err != nil {
		return Tool{}, err
	}

	if len(tags) > 0 {
		if err := json.Unmarshal(tags, &tool.Tags); err != nil {
			return Tool{}, err
		}
	}

	return tool, nil
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
CREATE TABLE IF NOT EXISTS tools (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	category TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	icon_class TEXT NOT NULL DEFAULT '',
	tags JSONB NOT NULL DEFAULT '[]'::jsonb,
	sort_order INTEGER NOT NULL DEFAULT 0,
	featured BOOLEAN NOT NULL DEFAULT FALSE,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS tools_status_category_sort_order
	ON tools(status, category ASC, featured DESC, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS tools_featured_sort_order
	ON tools(featured, sort_order ASC, created_at DESC);

CREATE INDEX IF NOT EXISTS tools_category_sort_order
	ON tools(category ASC, featured DESC, sort_order ASC, created_at DESC);
`
