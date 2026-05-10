package workexperience

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("work experience not found")
	ErrConflict = errors.New("work experience already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]WorkExperience, error)
	Get(context.Context, string) (WorkExperience, error)
	Create(context.Context, WorkExperience) (WorkExperience, error)
	Update(context.Context, string, func(*WorkExperience) error) (WorkExperience, error)
	Delete(context.Context, string) error
}
