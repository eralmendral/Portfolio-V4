package games

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
	entries map[string]Game
	now     func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		entries: make(map[string]Game),
		now:     time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Game, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]Game, 0, len(s.entries))
	for _, entry := range s.entries {
		if filter.Status != "" && entry.Status != filter.Status {
			continue
		}
		if filter.Platform != "" && !strings.EqualFold(entry.Platform, filter.Platform) {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(entry, filter.Query) {
			continue
		}
		if filter.From != "" && entry.MostlyPlayedOn < normalizeDate(filter.From) {
			continue
		}
		if filter.To != "" && entry.MostlyPlayedOn > normalizeDate(filter.To) {
			continue
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return gameLess(entries[i], entries[j])
	})

	return entries, nil
}

func (s *memoryStore) Get(_ context.Context, id string) (Game, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if entry, ok := s.entries[id]; ok {
		return entry, nil
	}

	return Game{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, entry Game) (Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if entry.ID == "" {
		entry.ID = newID()
	}
	if entry.Status == "" {
		entry.Status = StatusDraft
	}
	entry.MostlyPlayedOn = normalizeDate(entry.MostlyPlayedOn)
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now

	if _, exists := s.entries[entry.ID]; exists {
		return Game{}, ErrConflict
	}
	s.entries[entry.ID] = entry
	return entry, nil
}

func (s *memoryStore) Update(_ context.Context, id string, mutate func(*Game) error) (Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.entries[id]
	if !ok {
		return Game{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Game{}, err
	}
	next.ID = current.ID
	next.MostlyPlayedOn = normalizeDate(next.MostlyPlayedOn)
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

func gameLess(left Game, right Game) bool {
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	if left.MostlyPlayedOn != right.MostlyPlayedOn {
		return left.MostlyPlayedOn > right.MostlyPlayedOn
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func testMatchesQuery(entry Game, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		entry.Title,
		entry.Studio,
		entry.Platform,
		entry.Genre,
		entry.Notes,
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}

func TestMemoryStoreCRUD(t *testing.T) {
	store := newMemoryStore()
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	store.now = func() time.Time { return createdAt }

	created, err := store.Create(context.Background(), Game{
		ID:             "game-one",
		Title:          "First Game",
		Platform:       "PC",
		MostlyPlayedOn: "2026-05-10",
	})
	if err != nil {
		t.Fatalf("create game: %v", err)
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
		t.Fatalf("get game: %v", err)
	}
	if got.Title != "First Game" {
		t.Fatalf("title = %q, want First Game", got.Title)
	}

	store.now = func() time.Time { return updatedAt }
	updated, err := store.Update(context.Background(), created.ID, func(entry *Game) error {
		entry.Title = "Updated Game"
		entry.Status = StatusPublished
		return nil
	})
	if err != nil {
		t.Fatalf("update game: %v", err)
	}
	if updated.Title != "Updated Game" || updated.Status != StatusPublished || updated.UpdatedAt != updatedAt {
		t.Fatalf("unexpected updated game: %+v", updated)
	}

	if err := store.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete game: %v", err)
	}
	if _, err := store.Get(context.Background(), created.ID); err != ErrNotFound {
		t.Fatalf("get deleted error = %v, want %v", err, ErrNotFound)
	}
}

func TestMemoryStoreListFiltersAndOrdering(t *testing.T) {
	store := newMemoryStore()
	now := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	for _, entry := range []Game{
		{ID: "later-ranked", Title: "Starlit Roads", Studio: "North Play", Platform: "PC", Genre: "Adventure", MostlyPlayedOn: "2026-05-10", SortOrder: 20, Status: StatusPublished},
		{ID: "same-rank-newer", Title: "Garden Tactics", Studio: "Green Tile", Platform: "Switch", Genre: "Strategy", MostlyPlayedOn: "2026-05-11", SortOrder: 10, Status: StatusPublished},
		{ID: "same-rank-older", Title: "Kindling", Studio: "Small Fire", Platform: "PC", Genre: "Cozy Simulation", MostlyPlayedOn: "2026-05-09", Notes: "A cozy reset game with patient rituals.", SortOrder: 10, Status: StatusPublished},
		{ID: "draft", Title: "Private Draft", Platform: "PC", MostlyPlayedOn: "2026-05-12", Status: StatusDraft},
	} {
		if _, err := store.Create(context.Background(), entry); err != nil {
			t.Fatalf("create %s: %v", entry.ID, err)
		}
	}

	list, err := store.List(context.Background(), ListFilter{Status: StatusPublished})
	if err != nil {
		t.Fatalf("list games: %v", err)
	}
	wantOrder := []string{"same-rank-newer", "same-rank-older", "later-ranked"}
	if len(list) != len(wantOrder) {
		t.Fatalf("games count = %d, want %d", len(list), len(wantOrder))
	}
	for i, wantID := range wantOrder {
		if list[i].ID != wantID {
			t.Fatalf("games[%d] = %q, want %q", i, list[i].ID, wantID)
		}
	}

	platform, err := store.List(context.Background(), ListFilter{Status: StatusPublished, Platform: "pc"})
	if err != nil {
		t.Fatalf("platform games: %v", err)
	}
	if len(platform) != 2 {
		t.Fatalf("platform count = %d, want 2", len(platform))
	}

	search, err := store.List(context.Background(), ListFilter{Status: StatusPublished, Query: "cozy"})
	if err != nil {
		t.Fatalf("search games: %v", err)
	}
	if len(search) != 1 || search[0].ID != "same-rank-older" {
		t.Fatalf("unexpected search games: %+v", search)
	}

	dated, err := store.List(context.Background(), ListFilter{Status: StatusPublished, From: "2026-05-10", To: "2026-05-11"})
	if err != nil {
		t.Fatalf("date filter games: %v", err)
	}
	if len(dated) != 2 {
		t.Fatalf("dated games count = %d, want 2", len(dated))
	}
}
