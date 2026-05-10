package projects

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu       sync.RWMutex
	projects map[string]Project
	now      func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		projects: make(map[string]Project),
		now:      time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]Project, 0, len(s.projects))
	for _, project := range s.projects {
		if filter.Status != "" && project.Status != filter.Status {
			continue
		}
		if filter.Featured != nil && project.Featured != *filter.Featured {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(project, filter.Query) {
			continue
		}

		projects = append(projects, project)
	}

	sort.Slice(projects, func(i, j int) bool {
		if projects[i].SortOrder != projects[j].SortOrder {
			return projects[i].SortOrder < projects[j].SortOrder
		}
		return projects[i].CreatedAt.After(projects[j].CreatedAt)
	})

	return projects, nil
}

func (s *memoryStore) Get(_ context.Context, idOrSlug string) (Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if project, ok := s.projects[idOrSlug]; ok {
		return project, nil
	}

	for _, project := range s.projects {
		if project.Slug == idOrSlug {
			return project, nil
		}
	}

	return Project{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, project Project) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if project.ID == "" {
		project.ID = newID()
	}
	if project.Status == "" {
		project.Status = StatusDraft
	}
	if project.CreatedAt.IsZero() {
		project.CreatedAt = now
	}
	project.UpdatedAt = now

	if _, exists := s.projects[project.ID]; exists {
		return Project{}, ErrConflict
	}
	if s.slugExists(project.Slug, project.ID) {
		return Project{}, ErrConflict
	}

	s.projects[project.ID] = project
	return project, nil
}

func (s *memoryStore) Update(_ context.Context, idOrSlug string, mutate func(*Project) error) (Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, current, ok := s.findLocked(idOrSlug)
	if !ok {
		return Project{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Project{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()

	if next.Status == "" {
		next.Status = StatusDraft
	}
	if s.slugExists(next.Slug, next.ID) {
		return Project{}, ErrConflict
	}

	s.projects[id] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, idOrSlug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, _, ok := s.findLocked(idOrSlug)
	if !ok {
		return ErrNotFound
	}

	delete(s.projects, id)
	return nil
}

func (s *memoryStore) findLocked(idOrSlug string) (string, Project, bool) {
	if project, ok := s.projects[idOrSlug]; ok {
		return idOrSlug, project, true
	}

	for id, project := range s.projects {
		if project.Slug == idOrSlug {
			return id, project, true
		}
	}

	return "", Project{}, false
}

func (s *memoryStore) slugExists(slug string, exceptID string) bool {
	if slug == "" {
		return false
	}

	for id, project := range s.projects {
		if id != exceptID && project.Slug == slug {
			return true
		}
	}

	return false
}

func testMatchesQuery(project Project, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		project.Title,
		project.Slug,
		project.Summary,
		project.Description,
		strings.Join(project.TechStack, " "),
		strings.Join(project.Tags, " "),
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}
