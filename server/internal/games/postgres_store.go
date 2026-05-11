package games

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

func (s *PostgresStore) List(ctx context.Context, filter ListFilter) ([]Game, error) {
	where := []string{"1 = 1"}
	args := make([]any, 0, 6)

	if filter.Status != "" {
		args = append(args, filter.Status)
		where = append(where, fmt.Sprintf("g.status = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Platform) != "" {
		args = append(args, strings.ToLower(strings.TrimSpace(filter.Platform)))
		where = append(where, fmt.Sprintf("lower(g.platform) = $%d", len(args)))
	}
	if strings.TrimSpace(filter.Query) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(filter.Query))+"%")
		where = append(where, fmt.Sprintf(`(
			lower(g.title) LIKE $%[1]d OR
			lower(g.studio) LIKE $%[1]d OR
			lower(g.platform) LIKE $%[1]d OR
			lower(g.genre) LIKE $%[1]d OR
			lower(g.notes) LIKE $%[1]d
		)`, len(args)))
	}
	if strings.TrimSpace(filter.From) != "" {
		args = append(args, normalizeDate(filter.From))
		where = append(where, fmt.Sprintf("g.mostly_played_on >= $%d::date", len(args)))
	}
	if strings.TrimSpace(filter.To) != "" {
		args = append(args, normalizeDate(filter.To))
		where = append(where, fmt.Sprintf("g.mostly_played_on <= $%d::date", len(args)))
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT g.id, g.title, g.studio, g.platform, g.genre, g.store_url,
			g.mostly_played_on, g.notes, g.sort_order, g.status, g.created_at, g.updated_at
		FROM games g
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY g.sort_order ASC, g.mostly_played_on DESC, g.created_at DESC
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]Game, 0)
	for rows.Next() {
		entry, err := scanGame(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}

func (s *PostgresStore) Get(ctx context.Context, id string) (Game, error) {
	return s.get(ctx, s.db, id)
}

func (s *PostgresStore) Create(ctx context.Context, game Game) (Game, error) {
	now := s.now().UTC()
	if game.ID == "" {
		game.ID = newID()
	}
	if game.Status == "" {
		game.Status = StatusDraft
	}
	game.MostlyPlayedOn = normalizeDate(game.MostlyPlayedOn)
	if game.CreatedAt.IsZero() {
		game.CreatedAt = now
	}
	game.UpdatedAt = now

	if err := upsertGame(ctx, s.db, game); err != nil {
		return Game{}, normalizePostgresError(err)
	}

	return game, nil
}

func (s *PostgresStore) Update(ctx context.Context, id string, mutate func(*Game) error) (Game, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Game{}, err
	}
	defer rollback(tx)

	current, err := s.getForUpdate(ctx, tx, id)
	if err != nil {
		return Game{}, err
	}

	next := current
	if err := mutate(&next); err != nil {
		return Game{}, err
	}
	next.ID = current.ID
	next.MostlyPlayedOn = normalizeDate(next.MostlyPlayedOn)
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	if err := upsertGame(ctx, tx, next); err != nil {
		return Game{}, normalizePostgresError(err)
	}
	if err := tx.Commit(); err != nil {
		return Game{}, normalizePostgresError(err)
	}

	return next, nil
}

func (s *PostgresStore) Delete(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM games
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

func (s *PostgresStore) get(ctx context.Context, runner dbRunner, id string) (Game, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT g.id, g.title, g.studio, g.platform, g.genre, g.store_url,
			g.mostly_played_on, g.notes, g.sort_order, g.status, g.created_at, g.updated_at
		FROM games g
		WHERE g.id = $1
	`, id)

	game, err := scanGame(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, ErrNotFound
	}
	return game, err
}

func (s *PostgresStore) getForUpdate(ctx context.Context, runner dbRunner, id string) (Game, error) {
	row := runner.QueryRowContext(ctx, `
		SELECT g.id, g.title, g.studio, g.platform, g.genre, g.store_url,
			g.mostly_played_on, g.notes, g.sort_order, g.status, g.created_at, g.updated_at
		FROM games g
		WHERE g.id = $1
		FOR UPDATE
	`, id)

	game, err := scanGame(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Game{}, ErrNotFound
	}
	return game, err
}

func upsertGame(ctx context.Context, runner dbRunner, game Game) error {
	_, err := runner.ExecContext(ctx, `
		INSERT INTO games (
			id, title, studio, platform, genre, store_url,
			mostly_played_on, notes, sort_order, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::date, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			studio = EXCLUDED.studio,
			platform = EXCLUDED.platform,
			genre = EXCLUDED.genre,
			store_url = EXCLUDED.store_url,
			mostly_played_on = EXCLUDED.mostly_played_on,
			notes = EXCLUDED.notes,
			sort_order = EXCLUDED.sort_order,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, game.ID, game.Title, game.Studio, game.Platform, game.Genre, game.StoreURL,
		game.MostlyPlayedOn, game.Notes, game.SortOrder, game.Status, game.CreatedAt, game.UpdatedAt)
	return err
}

type gameScanner interface {
	Scan(...any) error
}

func scanGame(scanner gameScanner) (Game, error) {
	var game Game
	var mostlyPlayedOn time.Time

	if err := scanner.Scan(
		&game.ID,
		&game.Title,
		&game.Studio,
		&game.Platform,
		&game.Genre,
		&game.StoreURL,
		&mostlyPlayedOn,
		&game.Notes,
		&game.SortOrder,
		&game.Status,
		&game.CreatedAt,
		&game.UpdatedAt,
	); err != nil {
		return Game{}, err
	}

	game.MostlyPlayedOn = mostlyPlayedOn.Format(dateLayout)
	return game, nil
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
CREATE TABLE IF NOT EXISTS games (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	studio TEXT NOT NULL DEFAULT '',
	platform TEXT NOT NULL,
	genre TEXT NOT NULL DEFAULT '',
	store_url TEXT NOT NULL DEFAULT '',
	mostly_played_on DATE NOT NULL,
	notes TEXT NOT NULL DEFAULT '',
	sort_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS games_status_platform_sort_order_mostly_played_on
	ON games(status, platform, sort_order ASC, mostly_played_on DESC, created_at DESC);

CREATE INDEX IF NOT EXISTS games_mostly_played_on
	ON games(mostly_played_on DESC, created_at DESC);
`
