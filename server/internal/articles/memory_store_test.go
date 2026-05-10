package articles

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu       sync.RWMutex
	articles map[string]Article
	now      func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		articles: make(map[string]Article),
		now:      time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	articles := make([]Article, 0, len(s.articles))
	for _, article := range s.articles {
		if filter.Status != "" && article.Status != filter.Status {
			continue
		}
		if filter.Featured != nil && article.Featured != *filter.Featured {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(article, filter.Query) {
			continue
		}

		articles = append(articles, article)
	}

	sort.Slice(articles, func(i, j int) bool {
		return articleLess(articles[i], articles[j])
	})

	return articles, nil
}

func (s *memoryStore) Get(_ context.Context, id string) (Article, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if article, ok := s.articles[id]; ok {
		return article, nil
	}

	return Article{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, article Article) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if article.ID == "" {
		article.ID = newID()
	}
	if article.Status == "" {
		article.Status = StatusDraft
	}
	if article.CreatedAt.IsZero() {
		article.CreatedAt = now
	}
	article.UpdatedAt = now

	if _, exists := s.articles[article.ID]; exists {
		return Article{}, ErrConflict
	}
	if s.urlExists(article.URL, article.ID) {
		return Article{}, ErrConflict
	}

	s.articles[article.ID] = article
	return article, nil
}

func (s *memoryStore) Update(_ context.Context, id string, mutate func(*Article) error) (Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.articles[id]
	if !ok {
		return Article{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Article{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}
	if s.urlExists(next.URL, next.ID) {
		return Article{}, ErrConflict
	}

	s.articles[id] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.articles[id]; !ok {
		return ErrNotFound
	}

	delete(s.articles, id)
	return nil
}

func (s *memoryStore) urlExists(url string, exceptID string) bool {
	for id, article := range s.articles {
		if id != exceptID && article.URL == url {
			return true
		}
	}

	return false
}

func articleLess(left Article, right Article) bool {
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	if left.PublishedAt != nil && right.PublishedAt != nil && !left.PublishedAt.Equal(*right.PublishedAt) {
		return left.PublishedAt.After(*right.PublishedAt)
	}
	if left.PublishedAt != nil && right.PublishedAt == nil {
		return true
	}
	if left.PublishedAt == nil && right.PublishedAt != nil {
		return false
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func testMatchesQuery(article Article, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		article.Title,
		article.URL,
		article.Source,
		article.Summary,
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}
