package music

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu      sync.RWMutex
	entries map[string]Music
	now     func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		entries: make(map[string]Music),
		now:     time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Music, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]Music, 0, len(s.entries))
	for _, entry := range s.entries {
		if filter.Status != "" && entry.Status != filter.Status {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(entry, filter.Query) {
			continue
		}
		if filter.From != "" && entry.MostlyListenedOn < normalizeDate(filter.From) {
			continue
		}
		if filter.To != "" && entry.MostlyListenedOn > normalizeDate(filter.To) {
			continue
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return musicLess(entries[i], entries[j])
	})

	return entries, nil
}

func (s *memoryStore) Get(_ context.Context, id string) (Music, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entry, ok := s.entries[id]; ok {
		return entry, nil
	}

	return Music{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, entry Music) (Music, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if entry.ID == "" {
		entry.ID = newID()
	}
	if entry.Status == "" {
		entry.Status = StatusDraft
	}
	entry.MostlyListenedOn = normalizeDate(entry.MostlyListenedOn)
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	if _, exists := s.entries[entry.ID]; exists {
		return Music{}, ErrConflict
	}
	s.entries[entry.ID] = entry
	return entry, nil
}

func (s *memoryStore) Update(_ context.Context, id string, mutate func(*Music) error) (Music, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.entries[id]
	if !ok {
		return Music{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Music{}, err
	}
	next.ID = current.ID
	next.MostlyListenedOn = normalizeDate(next.MostlyListenedOn)
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}
	s.entries[id] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[id]; !ok {
		return ErrNotFound
	}

	delete(s.entries, id)
	return nil
}

func musicLess(left Music, right Music) bool {
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	if left.MostlyListenedOn != right.MostlyListenedOn {
		return left.MostlyListenedOn > right.MostlyListenedOn
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func testMatchesQuery(entry Music, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		entry.Title,
		entry.Artist,
		entry.Album,
		entry.Notes,
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}

func TestMemoryStoreCRUD(t *testing.T) {
	store := newMemoryStore()
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	store.now = func() time.Time { return createdAt }

	created, err := store.Create(context.Background(), Music{
		ID:               "song-one",
		Title:            "First Song",
		Artist:           "First Artist",
		MostlyListenedOn: "2026-05-10",
	})
	if err != nil {
		t.Fatalf("create music: %v", err)
	}
	if created.Status != StatusDraft {
		t.Fatalf("status = %q, want %q", created.Status, StatusDraft)
	}
	if created.CreatedAt != createdAt || created.UpdatedAt != createdAt {
		t.Fatalf("unexpected timestamps: created=%s updated=%s", created.CreatedAt, created.UpdatedAt)
	}

	if _, err := store.Create(context.Background(), created); err != ErrConflict {
		t.Fatalf("duplicate create error = %v, want %v", err, ErrConflict)
	}

	got, err := store.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get music: %v", err)
	}
	if got.Title != "First Song" {
		t.Fatalf("title = %q, want First Song", got.Title)
	}

	store.now = func() time.Time { return updatedAt }
	updated, err := store.Update(context.Background(), created.ID, func(entry *Music) error {
		entry.Title = "Updated Song"
		entry.Status = StatusPublished
		return nil
	})
	if err != nil {
		t.Fatalf("update music: %v", err)
	}
	if updated.Title != "Updated Song" || updated.Status != StatusPublished || updated.UpdatedAt != updatedAt {
		t.Fatalf("unexpected updated music: %+v", updated)
	}

	if err := store.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete music: %v", err)
	}
	if _, err := store.Get(context.Background(), created.ID); err != ErrNotFound {
		t.Fatalf("get deleted error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemoryStoreListFiltersAndOrdering(t *testing.T) {
	store := newMemoryStore()
	now := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	for _, entry := range []Music{
		{ID: "later-ranked", Title: "Night Drive", Artist: "Aster", Album: "Road Notes", MostlyListenedOn: "2026-05-10", SortOrder: 20, Status: StatusPublished},
		{ID: "same-rank-newer", Title: "Morning Signal", Artist: "Beacon", Album: "Light Map", MostlyListenedOn: "2026-05-11", SortOrder: 10, Status: StatusPublished},
		{ID: "same-rank-older", Title: "Quiet Loop", Artist: "Harbor", Album: "Still Water", MostlyListenedOn: "2026-05-09", Notes: "Ambient focus track", SortOrder: 10, Status: StatusPublished},
		{ID: "draft", Title: "Private Draft", Artist: "Hidden", MostlyListenedOn: "2026-05-12", Status: StatusDraft},
	} {
		if _, err := store.Create(context.Background(), entry); err != nil {
			t.Fatalf("create %s: %v", entry.ID, err)
		}
	}

	list, err := store.List(context.Background(), ListFilter{Status: StatusPublished})
	if err != nil {
		t.Fatalf("list music: %v", err)
	}
	wantOrder := []string{"same-rank-newer", "same-rank-older", "later-ranked"}
	if len(list) != len(wantOrder) {
		t.Fatalf("music count = %d, want %d", len(list), len(wantOrder))
	}
	for i, wantID := range wantOrder {
		if list[i].ID != wantID {
			t.Fatalf("music[%d] = %q, want %q", i, list[i].ID, wantID)
		}
	}

	search, err := store.List(context.Background(), ListFilter{Status: StatusPublished, Query: "ambient"})
	if err != nil {
		t.Fatalf("search music: %v", err)
	}
	if len(search) != 1 || search[0].ID != "same-rank-older" {
		t.Fatalf("unexpected search music: %+v", search)
	}

	dated, err := store.List(context.Background(), ListFilter{Status: StatusPublished, From: "2026-05-10", To: "2026-05-11"})
	if err != nil {
		t.Fatalf("date filter music: %v", err)
	}
	if len(dated) != 2 {
		t.Fatalf("dated music count = %d, want 2", len(dated))
	}
}
