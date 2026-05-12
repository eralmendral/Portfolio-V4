package games

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("game entry not found")
	ErrConflict = errors.New("game entry already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Game, error)
	Get(context.Context, string) (Game, error)
	Create(context.Context, Game) (Game, error)
	Update(context.Context, string, func(*Game) error) (Game, error)
	Delete(context.Context, string) error
}
