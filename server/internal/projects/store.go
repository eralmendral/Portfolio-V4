package projects

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("project not found")
	ErrConflict = errors.New("project already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Project, error)
	Get(context.Context, string) (Project, error)
	Create(context.Context, Project) (Project, error)
	Update(context.Context, string, func(*Project) error) (Project, error)
	Delete(context.Context, string) error
}
