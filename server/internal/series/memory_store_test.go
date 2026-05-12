package series

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
	entries map[string]Series
	now     func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		entries: make(map[string]Series),
		now:     time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Series, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]Series, 0, len(s.entries))
	for _, entry := range s.entries {
		if filter.Status != "" && entry.Status != filter.Status {
			continue
		}
		if filter.Category != "" && entry.Category != normalizeCategory(filter.Category) {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(entry, filter.Query) {
			continue
		}
		if filter.From != "" && entry.MostlyWatchedOn < normalizeDate(filter.From) {
			continue
		}
		if filter.To != "" && entry.MostlyWatchedOn > normalizeDate(filter.To) {
			continue
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return seriesLess(entries[i], entries[j])
	})

	return entries, nil
}

func (s *memoryStore) Get(_ context.Context, id string) (Series, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entry, ok := s.entries[id]; ok {
		return entry, nil
	}

	return Series{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, entry Series) (Series, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if entry.ID == "" {
		entry.ID = newID()
	}
	if entry.Status == "" {
		entry.Status = StatusDraft
	}
	entry.Category = normalizeCategory(entry.Category)
	entry.MostlyWatchedOn = normalizeDate(entry.MostlyWatchedOn)
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	if _, exists := s.entries[entry.ID]; exists {
		return Series{}, ErrConflict
	}
	s.entries[entry.ID] = entry
	return entry, nil
}

func (s *memoryStore) Update(_ context.Context, id string, mutate func(*Series) error) (Series, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.entries[id]
	if !ok {
		return Series{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Series{}, err
	}
	next.ID = current.ID
	next.Category = normalizeCategory(next.Category)
	next.MostlyWatchedOn = normalizeDate(next.MostlyWatchedOn)
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

func seriesLess(left Series, right Series) bool {
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	if left.MostlyWatchedOn != right.MostlyWatchedOn {
		return left.MostlyWatchedOn > right.MostlyWatchedOn
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func testMatchesQuery(entry Series, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		entry.Title,
		entry.Creator,
		entry.Platform,
		entry.Notes,
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}

func TestMemoryStoreCRUD(t *testing.T) {
	store := newMemoryStore()
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	store.now = func() time.Time { return createdAt }

	created, err := store.Create(context.Background(), Series{
		ID:              "series-one",
		Title:           "First Series",
		Category:        "TV Series",
		MostlyWatchedOn: "2026-05-10",
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	if created.Status != StatusDraft {
		t.Fatalf("status = %q, want %q", created.Status, StatusDraft)
	}
	if created.Category != CategoryTVSeries {
		t.Fatalf("category = %q, want %q", created.Category, CategoryTVSeries)
	}
	if created.CreatedAt != createdAt || created.UpdatedAt != createdAt {
		t.Fatalf("unexpected timestamps: created=%s updated=%s", created.CreatedAt, created.UpdatedAt)
	}

	if _, err := store.Create(context.Background(), created); err != ErrConflict {
		t.Fatalf("duplicate create error = %v, want %v", err, ErrConflict)
	}

	got, err := store.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get series: %v", err)
	}
	if got.Title != "First Series" {
		t.Fatalf("title = %q, want First Series", got.Title)
	}

	store.now = func() time.Time { return updatedAt }
	updated, err := store.Update(context.Background(), created.ID, func(entry *Series) error {
		entry.Title = "Updated Series"
		entry.Status = StatusPublished
		return nil
	})
	if err != nil {
		t.Fatalf("update series: %v", err)
	}
	if updated.Title != "Updated Series" || updated.Status != StatusPublished || updated.UpdatedAt != updatedAt {
		t.Fatalf("unexpected updated series: %+v", updated)
	}

	if err := store.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete series: %v", err)
	}
	if _, err := store.Get(context.Background(), created.ID); err != ErrNotFound {
		t.Fatalf("get deleted error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemoryStoreListFiltersAndOrdering(t *testing.T) {
	store := newMemoryStore()
	now := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	for _, entry := range []Series{
		{ID: "later-ranked", Title: "The Long Room", Category: CategoryTVSeries, Creator: "North Studio", Platform: "StreamBox", MostlyWatchedOn: "2026-05-10", SortOrder: 20, Status: StatusPublished},
		{ID: "same-rank-newer", Title: "Moon Harbor", Category: CategoryAnime, Creator: "Blue House", Platform: "Crunchyroll", MostlyWatchedOn: "2026-05-11", SortOrder: 10, Status: StatusPublished},
		{ID: "same-rank-older", Title: "Home Signals", Category: CategoryTVSeries, Creator: "City Room", Platform: "StreamBox", MostlyWatchedOn: "2026-05-09", Notes: "Comfort watch about chosen family.", SortOrder: 10, Status: StatusPublished},
		{ID: "draft", Title: "Private Draft", Category: CategoryAnime, MostlyWatchedOn: "2026-05-12", Status: StatusDraft},
	} {
		if _, err := store.Create(context.Background(), entry); err != nil {
			t.Fatalf("create %s: %v", entry.ID, err)
		}
	}

	list, err := store.List(context.Background(), ListFilter{Status: StatusPublished})
	if err != nil {
		t.Fatalf("list series: %v", err)
	}
	wantOrder := []string{"same-rank-newer", "same-rank-older", "later-ranked"}
	if len(list) != len(wantOrder) {
		t.Fatalf("series count = %d, want %d", len(list), len(wantOrder))
	}
	for i, wantID := range wantOrder {
		if list[i].ID != wantID {
			t.Fatalf("series[%d] = %q, want %q", i, list[i].ID, wantID)
		}
	}

	category, err := store.List(context.Background(), ListFilter{Status: StatusPublished, Category: "tv-series"})
	if err != nil {
		t.Fatalf("category series: %v", err)
	}
	if len(category) != 2 {
		t.Fatalf("category count = %d, want 2", len(category))
	}

	search, err := store.List(context.Background(), ListFilter{Status: StatusPublished, Query: "chosen family"})
	if err != nil {
		t.Fatalf("search series: %v", err)
	}
	if len(search) != 1 || search[0].ID != "same-rank-older" {
		t.Fatalf("unexpected search series: %+v", search)
	}

	dated, err := store.List(context.Background(), ListFilter{Status: StatusPublished, From: "2026-05-10", To: "2026-05-11"})
	if err != nil {
		t.Fatalf("date filter series: %v", err)
	}
	if len(dated) != 2 {
		t.Fatalf("dated series count = %d, want 2", len(dated))
	}
}
