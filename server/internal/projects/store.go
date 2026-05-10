package projects

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNotFound = errors.New("project not found")
	ErrConflict = errors.New("project already exists")
)

type Store interface {
	List(context.Context, ListFilter) ([]Project, error)
	Get(context.Context, string) (Project, error)
	Create(context.Context, Project) (Project, error)
	Update(context.Context, string, func(*Project) error) (Project, error)
	Delete(context.Context, string) error
}

type FileStore struct {
	mu       sync.RWMutex
	path     string
	projects map[string]Project
	now      func() time.Time
}

func NewFileStore(path string) (*FileStore, error) {
	store := &FileStore{
		path:     path,
		projects: make(map[string]Project),
		now:      time.Now,
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *FileStore) List(_ context.Context, filter ListFilter) ([]Project, error) {
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
		if filter.Query != "" && !matchesQuery(project, filter.Query) {
			continue
		}

		projects = append(projects, project)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].CreatedAt.After(projects[j].CreatedAt)
	})

	return projects, nil
}

func (s *FileStore) Get(_ context.Context, idOrSlug string) (Project, error) {
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

func (s *FileStore) Create(_ context.Context, project Project) (Project, error) {
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
	if err := s.saveLocked(); err != nil {
		delete(s.projects, project.ID)
		return Project{}, err
	}

	return project, nil
}

func (s *FileStore) Update(_ context.Context, idOrSlug string, mutate func(*Project) error) (Project, error) {
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
	if err := s.saveLocked(); err != nil {
		s.projects[id] = current
		return Project{}, err
	}

	return next, nil
}

func (s *FileStore) Delete(_ context.Context, idOrSlug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, _, ok := s.findLocked(idOrSlug)
	if !ok {
		return ErrNotFound
	}

	current := s.projects[id]
	delete(s.projects, id)
	if err := s.saveLocked(); err != nil {
		s.projects[id] = current
		return err
	}

	return nil
}

func (s *FileStore) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return err
	}

	for _, project := range projects {
		if project.ID != "" {
			s.projects[project.ID] = project
		}
	}

	return nil
}

func (s *FileStore) saveLocked() error {
	projects := make([]Project, 0, len(s.projects))
	for _, project := range s.projects {
		projects = append(projects, project)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].CreatedAt.Before(projects[j].CreatedAt)
	})

	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := s.path + ".tmp"
	if err := os.WriteFile(tmpPath, append(data, '\n'), 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, s.path)
}

func (s *FileStore) findLocked(idOrSlug string) (string, Project, bool) {
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

func (s *FileStore) slugExists(slug string, exceptID string) bool {
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

func matchesQuery(project Project, query string) bool {
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
