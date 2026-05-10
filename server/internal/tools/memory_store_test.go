package tools

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu    sync.RWMutex
	tools map[string]Tool
	now   func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		tools: make(map[string]Tool),
		now:   time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Tool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		if filter.Status != "" && tool.Status != filter.Status {
			continue
		}
		if filter.Featured != nil && tool.Featured != *filter.Featured {
			continue
		}
		if filter.Category != "" && !strings.EqualFold(tool.Category, filter.Category) {
			continue
		}
		if filter.Tag != "" && !hasTag(tool.Tags, filter.Tag) {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(tool, filter.Query) {
			continue
		}

		tools = append(tools, tool)
	}

	sort.Slice(tools, func(i, j int) bool {
		return toolLess(tools[i], tools[j])
	})

	return tools, nil
}

func (s *memoryStore) Get(_ context.Context, id string) (Tool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if tool, ok := s.tools[id]; ok {
		return tool, nil
	}

	return Tool{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, tool Tool) (Tool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if tool.ID == "" {
		tool.ID = newID()
	}
	if tool.Status == "" {
		tool.Status = StatusDraft
	}
	if tool.CreatedAt.IsZero() {
		tool.CreatedAt = now
	}
	tool.UpdatedAt = now

	if _, exists := s.tools[tool.ID]; exists {
		return Tool{}, ErrConflict
	}
	s.tools[tool.ID] = tool
	return tool, nil
}

func (s *memoryStore) Update(_ context.Context, id string, mutate func(*Tool) error) (Tool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.tools[id]
	if !ok {
		return Tool{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Tool{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}
	s.tools[id] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tools[id]; !ok {
		return ErrNotFound
	}

	delete(s.tools, id)
	return nil
}

func toolLess(left Tool, right Tool) bool {
	if !strings.EqualFold(left.Category, right.Category) {
		return strings.ToLower(left.Category) < strings.ToLower(right.Category)
	}
	if left.Featured != right.Featured {
		return left.Featured
	}
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func hasTag(tags []string, tag string) bool {
	for _, current := range tags {
		if strings.EqualFold(current, strings.TrimSpace(tag)) {
			return true
		}
	}
	return false
}

func testMatchesQuery(tool Tool, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		tool.Name,
		tool.Category,
		tool.Summary,
		strings.Join(tool.Tags, " "),
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}
