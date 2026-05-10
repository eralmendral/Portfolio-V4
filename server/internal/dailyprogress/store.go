package dailyprogress

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("daily progress entry not found")
	ErrConflict = errors.New("daily progress entry already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]DailyProgress, error)
	Get(context.Context, string) (DailyProgress, error)
	Create(context.Context, DailyProgress) (DailyProgress, error)
	Update(context.Context, string, func(*DailyProgress) error) (DailyProgress, error)
	Delete(context.Context, string) error
}
