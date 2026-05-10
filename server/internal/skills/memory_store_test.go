package skills

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type memoryStore struct {
	mu         sync.RWMutex
	categories map[string]SkillCategory
	skills     map[string]Skill
	now        func() time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		categories: make(map[string]SkillCategory),
		skills:     make(map[string]Skill),
		now:        time.Now,
	}
}

func (s *memoryStore) ListCategories(_ context.Context, filter CategoryListFilter) ([]SkillCategory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	categories := make([]SkillCategory, 0, len(s.categories))
	for _, category := range s.categories {
		if filter.Status != "" && category.Status != filter.Status {
			continue
		}
		if filter.Query != "" && !testCategoryMatchesQuery(category, filter.Query) {
			continue
		}

		categories = append(categories, category)
	}

	sort.Slice(categories, func(i, j int) bool {
		return categoryLess(categories[i], categories[j])
	})

	return categories, nil
}

func (s *memoryStore) GetCategory(_ context.Context, idOrSlug string) (SkillCategory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if category, ok := s.categories[idOrSlug]; ok {
		return category, nil
	}

	for _, category := range s.categories {
		if category.Slug == idOrSlug {
			return category, nil
		}
	}

	return SkillCategory{}, ErrNotFound
}

func (s *memoryStore) CreateCategory(_ context.Context, category SkillCategory) (SkillCategory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if category.ID == "" {
		category.ID = newID()
	}
	if category.Status == "" {
		category.Status = StatusDraft
	}
	if category.CreatedAt.IsZero() {
		category.CreatedAt = now
	}
	category.UpdatedAt = now

	if _, exists := s.categories[category.ID]; exists {
		return SkillCategory{}, ErrConflict
	}
	if s.categorySlugExists(category.Slug, category.ID) {
		return SkillCategory{}, ErrConflict
	}

	s.categories[category.ID] = category
	return category, nil
}

func (s *memoryStore) UpdateCategory(_ context.Context, idOrSlug string, mutate func(*SkillCategory) error) (SkillCategory, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, current, ok := s.findCategoryLocked(idOrSlug)
	if !ok {
		return SkillCategory{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return SkillCategory{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}
	if s.categorySlugExists(next.Slug, next.ID) {
		return SkillCategory{}, ErrConflict
	}

	s.categories[id] = next
	return next, nil
}

func (s *memoryStore) DeleteCategory(_ context.Context, idOrSlug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	id, _, ok := s.findCategoryLocked(idOrSlug)
	if !ok {
		return ErrNotFound
	}

	for _, skill := range s.skills {
		if skill.CategoryID == id {
			return ErrCategoryInUse
		}
	}

	delete(s.categories, id)
	return nil
}

func (s *memoryStore) ListSkills(_ context.Context, filter SkillListFilter) ([]Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	skills := make([]Skill, 0, len(s.skills))
	for _, skill := range s.skills {
		category, ok := s.categories[skill.CategoryID]
		if !ok {
			continue
		}
		if filter.Status != "" && skill.Status != filter.Status {
			continue
		}
		if filter.Featured != nil && skill.Featured != *filter.Featured {
			continue
		}
		if filter.CategoryID != "" && skill.CategoryID != filter.CategoryID {
			continue
		}
		if filter.Category != "" && !categoryMatchesFilter(category, filter.Category) {
			continue
		}
		if filter.Query != "" && !testSkillMatchesQuery(skill, category, filter.Query) {
			continue
		}

		skills = append(skills, skill)
	}

	sort.Slice(skills, func(i, j int) bool {
		return skillLess(skills[i], skills[j], s.categories)
	})

	return skills, nil
}

func (s *memoryStore) GetSkill(_ context.Context, id string) (Skill, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if skill, ok := s.skills[id]; ok {
		return skill, nil
	}

	return Skill{}, ErrNotFound
}

func (s *memoryStore) CreateSkill(_ context.Context, skill Skill) (Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now().UTC()
	if skill.ID == "" {
		skill.ID = newID()
	}
	if skill.Status == "" {
		skill.Status = StatusDraft
	}
	if skill.CreatedAt.IsZero() {
		skill.CreatedAt = now
	}
	skill.UpdatedAt = now

	if _, exists := s.skills[skill.ID]; exists {
		return Skill{}, ErrConflict
	}
	if _, exists := s.categories[skill.CategoryID]; !exists {
		return Skill{}, ErrInvalidCategory
	}
	if s.skillNameExists(skill.CategoryID, skill.Name, skill.ID) {
		return Skill{}, ErrConflict
	}

	s.skills[skill.ID] = skill
	return skill, nil
}

func (s *memoryStore) UpdateSkill(_ context.Context, id string, mutate func(*Skill) error) (Skill, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	current, ok := s.skills[id]
	if !ok {
		return Skill{}, ErrNotFound
	}

	next := current
	if err := mutate(&next); err != nil {
		return Skill{}, err
	}
	next.ID = current.ID
	next.CreatedAt = current.CreatedAt
	next.UpdatedAt = s.now().UTC()
	if next.Status == "" {
		next.Status = StatusDraft
	}
	if _, exists := s.categories[next.CategoryID]; !exists {
		return Skill{}, ErrInvalidCategory
	}
	if s.skillNameExists(next.CategoryID, next.Name, next.ID) {
		return Skill{}, ErrConflict
	}

	s.skills[id] = next
	return next, nil
}

func (s *memoryStore) DeleteSkill(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.skills[id]; !ok {
		return ErrNotFound
	}

	delete(s.skills, id)
	return nil
}

func (s *memoryStore) findCategoryLocked(idOrSlug string) (string, SkillCategory, bool) {
	if category, ok := s.categories[idOrSlug]; ok {
		return idOrSlug, category, true
	}

	for id, category := range s.categories {
		if category.Slug == idOrSlug {
			return id, category, true
		}
	}

	return "", SkillCategory{}, false
}

func (s *memoryStore) categorySlugExists(slug string, exceptID string) bool {
	if slug == "" {
		return false
	}

	for id, category := range s.categories {
		if id != exceptID && category.Slug == slug {
			return true
		}
	}

	return false
}

func (s *memoryStore) skillNameExists(categoryID string, name string, exceptID string) bool {
	for id, skill := range s.skills {
		if id != exceptID && skill.CategoryID == categoryID && skill.Name == name {
			return true
		}
	}

	return false
}

func categoryLess(left SkillCategory, right SkillCategory) bool {
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func skillLess(left Skill, right Skill, categories map[string]SkillCategory) bool {
	leftCategory := categories[left.CategoryID]
	rightCategory := categories[right.CategoryID]
	if leftCategory.SortOrder != rightCategory.SortOrder {
		return leftCategory.SortOrder < rightCategory.SortOrder
	}
	if left.Featured != right.Featured {
		return left.Featured
	}
	if left.SortOrder != right.SortOrder {
		return left.SortOrder < right.SortOrder
	}
	return left.CreatedAt.After(right.CreatedAt)
}

func testCategoryMatchesQuery(category SkillCategory, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		category.Slug,
		category.Name,
		category.Description,
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}

func testSkillMatchesQuery(skill Skill, category SkillCategory, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	haystack := strings.Join([]string{
		skill.Name,
		skill.Summary,
		category.Name,
		category.Description,
	}, " ")

	return strings.Contains(strings.ToLower(haystack), query)
}

func categoryMatchesFilter(category SkillCategory, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.ToLower(category.ID) == value ||
		strings.ToLower(category.Slug) == value ||
		strings.ToLower(category.Name) == value
}
