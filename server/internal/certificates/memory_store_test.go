package certificates

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu           sync.RWMutex
	certificates map[string]Certificate
	now          func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		certificates: make(map[string]Certificate),
		now:          time.Now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	certificates := make([]Certificate, 0, len(s.certificates))
	for _, certificate := range s.certificates {
		if filter.Status != "" && certificate.Status != filter.Status {
			continue
		}
		if filter.Featured != nil && certificate.Featured != *filter.Featured {
			continue
		}
		if filter.Query != "" && !testMatchesQuery(certificate, filter.Query) {
			continue
		}

		certificates = append(certificates, certificate)
	}

	sort.Slice(certificates, func(i, j int) bool {
		if certificates[i].SortOrder != certificates[j].SortOrder {
			return certificates[i].SortOrder < certificates[j].SortOrder
		}
		return certificates[i].CreatedAt.After(certificates[j].CreatedAt)
	})

	return certificates, nil
}

func (s *memoryStore) Get(_ context.Context, idOrSlug string) (Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if certificate, ok := s.certificates[idOrSlug]; ok {
		return certificate, nil
	}

	for _, certificate := range s.certificates {
		if certificate.Slug == idOrSlug {
			return certificate, nil
		}
	}

	return Certificate{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, certificate Certificate) (Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if certificate.ID == "" {
		certificate.ID = newID()
	}
	if certificate.Status == "" {
		certificate.Status = StatusDraft
	}
	if certificate.CreatedAt.IsZero() {
		certificate.CreatedAt = now
	}
	certificate.UpdatedAt = now

	if _, exists := s.certificates[certificate.ID]; exists {
		return Certificate{}, ErrConflict
	}
	if s.slugExists(certificate.Slug, certificate.ID) {
		return Certificate{}, ErrConflict
	}

	s.certificates[certificate.ID] = certificate
	return certificate, nil
}

func (s *memoryStore) Update(_ context.Context, idOrSlug string, mutate func(*Certificate) error) (Certificate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, current, ok := s.findLocked(idOrSlug)
	if !ok {
		return Certificate{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Certificate{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()

	if next.Status == "" {
		next.Status = StatusDraft
	}
	if s.slugExists(next.Slug, next.ID) {
		return Certificate{}, ErrConflict
	}

	s.certificates[id] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, idOrSlug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, _, ok := s.findLocked(idOrSlug)
	if !ok {
		return ErrNotFound
	}

	delete(s.certificates, id)
	return nil
}

func (s *memoryStore) findLocked(idOrSlug string) (string, Certificate, bool) {
	if certificate, ok := s.certificates[idOrSlug]; ok {
		return idOrSlug, certificate, true
	}

	for id, certificate := range s.certificates {
		if certificate.Slug == idOrSlug {
			return id, certificate, true
		}
	}

	return "", Certificate{}, false
}

func (s *memoryStore) slugExists(slug string, exceptID string) bool {
	if slug == "" {
		return false
	}

	for id, certificate := range s.certificates {
		if id != exceptID && certificate.Slug == slug {
			return true
		}
	}

	return false
}

func testMatchesQuery(certificate Certificate, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		certificate.Title,
		certificate.Slug,
		certificate.Issuer,
		certificate.Summary,
		certificate.Description,
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}
