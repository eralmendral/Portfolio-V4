package certificates

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("certificate not found")
	ErrConflict = errors.New("certificate already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Certificate, error)
	Get(context.Context, string) (Certificate, error)
	Create(context.Context, Certificate) (Certificate, error)
	Update(context.Context, string, func(*Certificate) error) (Certificate, error)
	Delete(context.Context, string) error
}
