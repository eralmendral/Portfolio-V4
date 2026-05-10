package intro

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("intro not found")

type Store interface {
	Get(context.Context) (Intro, error)
	Save(context.Context, Intro) (Intro, error)
	Delete(context.Context) error
}
