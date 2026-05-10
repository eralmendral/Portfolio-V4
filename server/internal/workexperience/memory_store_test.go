package workexperience

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu              sync.RWMutex
	workExperiences map[string]WorkExperience
	now             func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		workExperiences: make(map[string]WorkExperience),
		now:             time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]WorkExperience, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workExperiences := make([]WorkExperience, 0, len(s.workExperiences))
	for _, workExperience := range s.workExperiences {
		if filter.Status != "" && workExperience.Status != filter.Status {
			continue
		}
		if filter.Featured != nil && workExperience.Featured != *filter.Featured {
			continue
		}
		if filter.Current != nil && workExperience.Current != *filter.Current {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(workExperience, filter.Query) {
			continue
		}

		workExperiences = append(workExperiences, workExperience)
	}

	sort.Slice(workExperiences, func(i, j int) bool {
		return workExperienceLess(workExperiences[i], workExperiences[j])
	})

	return workExperiences, nil
}

func (s *memoryStore) Get(_ context.Context, idOrSlug string) (WorkExperience, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, workExperience := range s.workExperiences {
		if workExperience.ID == idOrSlug || workExperience.Slug == idOrSlug {
			return workExperience, nil
		}
	}

	return WorkExperience{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, workExperience WorkExperience) (WorkExperience, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if workExperience.ID == "" {
		workExperience.ID = newID()
	}
	if workExperience.Status == "" {
		workExperience.Status = StatusDraft
	}
	if workExperience.CreatedAt.IsZero() {
		workExperience.CreatedAt = now
	}
	workExperience.UpdatedAt = now

	if _, exists := s.workExperiences[workExperience.ID]; exists {
		return WorkExperience{}, ErrConflict
	}
	if s.slugExists(workExperience.Slug, workExperience.ID) {
		return WorkExperience{}, ErrConflict
	}

	s.workExperiences[workExperience.ID] = workExperience
	return workExperience, nil
}

func (s *memoryStore) Update(_ context.Context, idOrSlug string, mutate func(*WorkExperience) error) (WorkExperience, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var current WorkExperience
	var foundID string
	for id, workExperience := range s.workExperiences {
		if workExperience.ID == idOrSlug || workExperience.Slug == idOrSlug {
			current = workExperience
			foundID = id
			break
		}
	}
	if foundID == "" {
		return WorkExperience{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return WorkExperience{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}
	if s.slugExists(next.Slug, next.ID) {
		return WorkExperience{}, ErrConflict
	}

	s.workExperiences[foundID] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, idOrSlug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, workExperience := range s.workExperiences {
		if workExperience.ID == idOrSlug || workExperience.Slug == idOrSlug {
			delete(s.workExperiences, id)
			return nil
		}
	}

	return ErrNotFound
}

func (s *memoryStore) slugExists(slug string, exceptID string) bool {
	for id, workExperience := range s.workExperiences {
		if id != exceptID && workExperience.Slug == slug {
			return true
		}
	}

	return false
}

func workExperienceLess(left WorkExperience, right WorkExperience) bool {
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	if left.Current != right.Current {
		return left.Current
	}
	if !left.StartedAt.Equal(right.StartedAt) {
		return left.StartedAt.After(right.StartedAt)
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func testMatchesQuery(workExperience WorkExperience, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		workExperience.Title,
		workExperience.Slug,
		workExperience.Company,
		workExperience.EmploymentType,
		workExperience.Location,
		workExperience.LocationType,
		workExperience.Summary,
		workExperience.Description,
		strings.Join(workExperience.Highlights, " "),
		strings.Join(workExperience.Responsibilities, " "),
		strings.Join(workExperience.TechStack, " "),
		strings.Join(workExperience.Skills, " "),
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}
