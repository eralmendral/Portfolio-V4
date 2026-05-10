package links

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("link not found")
	ErrConflict = errors.New("link already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Link, error)
	Get(context.Context, string) (Link, error)
	Create(context.Context, Link) (Link, error)
	Update(context.Context, string, func(*Link) error) (Link, error)
	Delete(context.Context, string) error
}
