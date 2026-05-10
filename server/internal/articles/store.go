package articles

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("article not found")
	ErrConflict = errors.New("article already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Article, error)
	Get(context.Context, string) (Article, error)
	Create(context.Context, Article) (Article, error)
	Update(context.Context, string, func(*Article) error) (Article, error)
	Delete(context.Context, string) error
}
