package products

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu       sync.RWMutex
	products map[string]Product
	section  ProductSectionSettings
	now      func() time.Time
}

func newMemoryStore() *memoryStore {
	now := time.Now
	return &memoryStore{
		products: make(map[string]Product),
		section: ProductSectionSettings{
			Enabled:     false,
			Title:       "Products",
			Description: "",
			UpdatedAt:   now().UTC(),
		},
		now: now,
	}
}

func (s *memoryStore) List(_ context.Context, filter ListFilter) ([]Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	products := make([]Product, 0, len(s.products))
	for _, product := range s.products {
		if filter.Status != "" && product.Status != filter.Status {
			continue
		}
		if filter.Featured != nil && product.Featured != *filter.Featured {
			continue
		}
		if filter.Category != "" && !strings.EqualFold(product.Category, filter.Category) {
			continue
		}
		if filter.Tag != "" && !hasTag(product.Tags, filter.Tag) {
			continue
		}
		if filter.Query != "" && !productMatchesQuery(product, filter.Query) {
			continue
		}

		products = append(products, product)
	}

	sort.Slice(products, func(i, j int) bool {
		return productLess(products[i], products[j])
	})

	return products, nil
}

func (s *memoryStore) Get(_ context.Context, idOrSlug string) (Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if product, ok := s.products[idOrSlug]; ok {
		return product, nil
	}

	for _, product := range s.products {
		if product.Slug == idOrSlug {
			return product, nil
		}
	}

	return Product{}, ErrNotFound
}

func (s *memoryStore) Create(_ context.Context, product Product) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if product.ID == "" {
		product.ID = newID()
	}
	if product.Status == "" {
		product.Status = StatusDraft
	}
	if product.CreatedAt.IsZero() {
		product.CreatedAt = now
	}
	product.UpdatedAt = now

	if _, exists := s.products[product.ID]; exists {
		return Product{}, ErrConflict
	}
	if s.slugExists(product.Slug, product.ID) {
		return Product{}, ErrConflict
	}

	s.products[product.ID] = product
	return product, nil
}

func (s *memoryStore) Update(_ context.Context, idOrSlug string, mutate func(*Product) error) (Product, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, current, ok := s.findLocked(idOrSlug)
	if !ok {
		return Product{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Product{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}
	if s.slugExists(next.Slug, next.ID) {
		return Product{}, ErrConflict
	}

	s.products[id] = next
	return next, nil
}

func (s *memoryStore) Delete(_ context.Context, idOrSlug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, _, ok := s.findLocked(idOrSlug)
	if !ok {
		return ErrNotFound
	}

	delete(s.products, id)
	return nil
}

func (s *memoryStore) GetSection(_ context.Context) (ProductSectionSettings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.section, nil
}

func (s *memoryStore) UpdateSection(_ context.Context, mutate func(*ProductSectionSettings) error) (ProductSectionSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.section
	if err := mutate(&next); err != nil {
		return ProductSectionSettings{}, err
	}
	next.UpdatedAt = s.now().UTC()
	s.section = next
	return next, nil
}

func (s *memoryStore) findLocked(idOrSlug string) (string, Product, bool) {
	if product, ok := s.products[idOrSlug]; ok {
		return idOrSlug, product, true
	}

	for id, product := range s.products {
		if product.Slug == idOrSlug {
			return id, product, true
		}
	}

	return "", Product{}, false
}

func (s *memoryStore) slugExists(slug string, exceptID string) bool {
	if slug == "" {
		return false
	}

	for id, product := range s.products {
		if id != exceptID && product.Slug == slug {
			return true
		}
	}

	return false
}

func productLess(left Product, right Product) bool {
	if left.Featured != right.Featured {
		return left.Featured
	}
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	leftPublished := time.Time{}
	rightPublished := time.Time{}
	if left.PublishedAt != nil {
		leftPublished = *left.PublishedAt
	}
	if right.PublishedAt != nil {
		rightPublished = *right.PublishedAt
	}
	if !leftPublished.Equal(rightPublished) {
		return leftPublished.After(rightPublished)
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

func productMatchesQuery(product Product, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		product.Title,
		product.Slug,
		product.Summary,
		product.Description,
		product.PriceLabel,
		product.CTALabel,
		product.Category,
		strings.Join(product.Tags, " "),
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}
