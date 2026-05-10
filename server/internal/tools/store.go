package tools

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("tool not found")
	ErrConflict = errors.New("tool already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Tool, error)
	Get(context.Context, string) (Tool, error)
	Create(context.Context, Tool) (Tool, error)
	Update(context.Context, string, func(*Tool) error) (Tool, error)
	Delete(context.Context, string) error
}
