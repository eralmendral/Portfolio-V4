package dailyprogress

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu      sync.RWMutex
	entries map[string]DailyProgress
	now     func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		entries: make(map[string]DailyProgress),
		now:     time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]DailyProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]DailyProgress, 0, len(s.entries))
	for _, entry := range s.entries {
		if filter.Status != "" && entry.Status != filter.Status {
			continue
		}
		if filter.Tag != "" && !hasTag(entry.Tags, filter.Tag) {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(entry, filter.Query) {
			continue
		}
		if filter.From != "" && entry.EntryDate < normalizeDate(filter.From) {
			continue
		}
		if filter.To != "" && entry.EntryDate > normalizeDate(filter.To) {
			continue
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].EntryDate != entries[j].EntryDate {
			return entries[i].EntryDate > entries[j].EntryDate
		}
		return entries[i].CreatedAt.After(entries[j].CreatedAt)
	})

	return entries, nil
}

func (s *memoryStore) Get(_ context.Context, idOrDate string) (DailyProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, entry := range s.entries {
		if entry.ID == idOrDate || entry.EntryDate == idOrDate {
			return entry, nil
		}
	}

	return DailyProgress{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, entry DailyProgress) (DailyProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if entry.ID == "" {
		entry.ID = newID()
	}
	if entry.Status == "" {
		entry.Status = StatusDraft
	}
	entry.EntryDate = normalizeDate(entry.EntryDate)
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	for _, current := range s.entries {
		if current.ID == entry.ID || current.EntryDate == entry.EntryDate {
			return DailyProgress{}, ErrConflict
		}
	}

	s.entries[entry.ID] = entry
	return entry, nil
}

func (s *memoryStore) Update(_ context.Context, idOrDate string, mutate func(*DailyProgress) error) (DailyProgress, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var current DailyProgress
	var key string
	for id, entry := range s.entries {
		if entry.ID == idOrDate || entry.EntryDate == idOrDate {
			current = entry
			key = id
			break
		}
	}
	if key == "" {
		return DailyProgress{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return DailyProgress{}, err
	}
	next.ID = current.ID
	next.EntryDate = normalizeDate(next.EntryDate)
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}
	for id, entry := range s.entries {
		if id != key && entry.EntryDate == next.EntryDate {
			return DailyProgress{}, ErrConflict
		}
	}

	s.entries[key] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, idOrDate string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, entry := range s.entries {
		if entry.ID == idOrDate || entry.EntryDate == idOrDate {
			delete(s.entries, id)
			return nil
		}
	}

	return ErrNotFound
}

func hasTag(tags []string, tag string) bool {
	for _, current := range tags {
		if strings.EqualFold(current, strings.TrimSpace(tag)) {
			return true
		}
	}
	return false
}

func testMatchesQuery(entry DailyProgress, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		entry.Title,
		entry.Summary,
		entry.Content,
		entry.Mood,
		strings.Join(entry.Wins, " "),
		strings.Join(entry.Learnings, " "),
		strings.Join(entry.NextSteps, " "),
		strings.Join(entry.Tags, " "),
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}
