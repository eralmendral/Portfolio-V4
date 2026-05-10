package skills

import (
	"context"
	"errors"
)

var (
	ErrNotFound        = errors.New("skill resource not found")
	ErrConflict        = errors.New("skill resource already exists")
	ErrInvalidCategory = errors.New("skill category does not exist")
	ErrCategoryInUse   = errors.New("skill category has skills")
)

type Store interface {
	ListCategories(context.Context, CategoryListFilter) ([]SkillCategory, error)
	GetCategory(context.Context, string) (SkillCategory, error)
	CreateCategory(context.Context, SkillCategory) (SkillCategory, error)
	UpdateCategory(context.Context, string, func(*SkillCategory) error) (SkillCategory, error)
	DeleteCategory(context.Context, string) error

	ListSkills(context.Context, SkillListFilter) ([]Skill, error)
	GetSkill(context.Context, string) (Skill, error)
	CreateSkill(context.Context, Skill) (Skill, error)
	UpdateSkill(context.Context, string, func(*Skill) error) (Skill, error)
	DeleteSkill(context.Context, string) error
}
