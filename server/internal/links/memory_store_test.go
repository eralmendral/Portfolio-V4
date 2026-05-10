package links

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu    sync.RWMutex
	links map[string]Link
	now   func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		links: make(map[string]Link),
		now:   time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	links := make([]Link, 0, len(s.links))
	for _, link := range s.links {
		if filter.Status != "" && link.Status != filter.Status {
			continue
		}
		if filter.Star != nil && link.Star != *filter.Star {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(link, filter.Query) {
			continue
		}

		links = append(links, link)
	}

	sort.Slice(links, func(i, j int) bool {
		if links[i].SortOrder != links[j].SortOrder {
			return links[i].SortOrder < links[j].SortOrder
		}
		return links[i].CreatedAt.After(links[j].CreatedAt)
	})

	return links, nil
}

func (s *memoryStore) Get(_ context.Context, id string) (Link, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if link, ok := s.links[id]; ok {
		return link, nil
	}

	return Link{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, link Link) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if link.ID == "" {
		link.ID = newID()
	}
	if link.Status == "" {
		link.Status = StatusDraft
	}
	if link.CreatedAt.IsZero() {
		link.CreatedAt = now
	}
	link.UpdatedAt = now

	if _, exists := s.links[link.ID]; exists {
		return Link{}, ErrConflict
	}

	s.links[link.ID] = link
	return link, nil
}

func (s *memoryStore) Update(_ context.Context, id string, mutate func(*Link) error) (Link, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.links[id]
	if !ok {
		return Link{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Link{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}

	s.links[id] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.links[id]; !ok {
		return ErrNotFound
	}

	delete(s.links, id)
	return nil
}

func testMatchesQuery(link Link, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		link.Label,
		link.URL,
		link.IconClass,
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}
