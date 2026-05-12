package series

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("series entry not found")
	ErrConflict = errors.New("series entry already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Series, error)
	Get(context.Context, string) (Series, error)
	Create(context.Context, Series) (Series, error)
	Update(context.Context, string, func(*Series) error) (Series, error)
	Delete(context.Context, string) error
}
