package intro

import (
	"context"
	"sync"
	"time"
)

type memoryStore struct {
	mu    sync.RWMutex
	intro *Intro
	now   func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		now: time.Now,
	}
}

func (s *memoryStore) Get(_ context.Context) (Intro, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.intro == nil {
		return Intro{}, ErrNotFound
	}
	return *s.intro, nil
}

func (s *memoryStore) Save(_ context.Context, intro Intro) (Intro, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if intro.ID == "" {
		intro.ID = DefaultID
	}
	if intro.CreatedAt.IsZero() {
		intro.CreatedAt = now
	}
	intro.UpdatedAt = now

	s.intro = &intro
	return intro, nil
}

func (s *memoryStore) Delete(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.intro == nil {
		return ErrNotFound
	}

	s.intro = nil
	return nil
}
