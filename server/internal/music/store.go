package music

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("music entry not found")
	ErrConflict = errors.New("music entry already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Music, error)
	Get(context.Context, string) (Music, error)
	Create(context.Context, Music) (Music, error)
	Update(context.Context, string, func(*Music) error) (Music, error)
	Delete(context.Context, string) error
}
