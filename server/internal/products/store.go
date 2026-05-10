package products

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("product not found")
	ErrConflict = errors.New("product already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Product, error)
	Get(context.Context, string) (Product, error)
	Create(context.Context, Product) (Product, error)
	Update(context.Context, string, func(*Product) error) (Product, error)
	Delete(context.Context, string) error
	GetSection(context.Context) (ProductSectionSettings, error)
	UpdateSection(context.Context, func(*ProductSectionSettings) error) (ProductSectionSettings, error)
}
