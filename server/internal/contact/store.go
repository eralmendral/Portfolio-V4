package contact

import (
	"context"
	"errors"
)

var ErrInvalid = errors.New("invalid contact submission")
var ErrNotFound = errors.New("contact profile not found")

type Store interface {
	Create(context.Context, Submission) (Submission, error)
	GetProfile(context.Context) (Profile, error)
	SaveProfile(context.Context, Profile) (Profile, error)
}
