package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/eralme/server/internal/articles"
	"github.com/eralme/server/internal/certificates"
	"github.com/eralme/server/internal/intro"
	"github.com/eralme/server/internal/links"
	"github.com/eralme/server/internal/products"
	"github.com/eralme/server/internal/projects"
	"github.com/eralme/server/internal/skills"
	"github.com/eralme/server/internal/tools"
	"github.com/eralme/server/internal/workexperience"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const maxSampleRecords = 6

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("connect postgres: %v", err)
	}

	projectStore, err := projects.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate projects postgres: %v", err)
	}
	certificateStore, err := certificates.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate certificates postgres: %v", err)
	}
	articleStore, err := articles.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate articles postgres: %v", err)
	}
	workExperienceStore, err := workexperience.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate work experiences postgres: %v", err)
	}
	skillStore, err := skills.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate skills postgres: %v", err)
	}
	linkStore, err := links.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate links postgres: %v", err)
	}
	toolStore, err := tools.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate tools postgres: %v", err)
	}
	productStore, err := products.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate products postgres: %v", err)
	}
	introStore, err := intro.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate intro postgres: %v", err)
	}

	projectSamples := sampleProjects()
	validateSampleLimit("projects", len(projectSamples))
	for _, project := range projectSamples {
		if err := deleteProjectIfExists(ctx, projectStore, project.ID); err != nil {
			log.Fatalf("delete sample %q: %v", project.ID, err)
		}
		if err := deleteProjectIfExists(ctx, projectStore, project.Slug); err != nil {
			log.Fatalf("delete sample %q: %v", project.Slug, err)
		}
		if _, err := projectStore.Create(ctx, project); err != nil {
			log.Fatalf("create sample %q: %v", project.Slug, err)
		}
	}

	certificateSamples := sampleCertificates()
	validateSampleLimit("certificates", len(certificateSamples))
	for _, certificate := range certificateSamples {
		if err := deleteCertificateIfExists(ctx, certificateStore, certificate.ID); err != nil {
			log.Fatalf("delete certificate sample %q: %v", certificate.ID, err)
		}
		if err := deleteCertificateIfExists(ctx, certificateStore, certificate.Slug); err != nil {
			log.Fatalf("delete certificate sample %q: %v", certificate.Slug, err)
		}
		if _, err := certificateStore.Create(ctx, certificate); err != nil {
			log.Fatalf("create certificate sample %q: %v", certificate.Slug, err)
		}
	}

	articleSamples := sampleArticles()
	validateSampleLimit("articles", len(articleSamples))
	for _, article := range articleSamples {
		if err := deleteArticleIfExists(ctx, articleStore, article.ID); err != nil {
			log.Fatalf("delete article sample %q: %v", article.ID, err)
		}
		if _, err := articleStore.Create(ctx, article); err != nil {
			log.Fatalf("create article sample %q: %v", article.ID, err)
		}
	}

	workExperienceSamples := sampleWorkExperiences()
	validateSampleLimit("work experiences", len(workExperienceSamples))
	for _, workExperience := range workExperienceSamples {
		if err := deleteWorkExperienceIfExists(ctx, workExperienceStore, workExperience.ID); err != nil {
			log.Fatalf("delete work experience sample %q: %v", workExperience.ID, err)
		}
		if err := deleteWorkExperienceIfExists(ctx, workExperienceStore, workExperience.Slug); err != nil {
			log.Fatalf("delete work experience sample %q: %v", workExperience.Slug, err)
		}
		if _, err := workExperienceStore.Create(ctx, workExperience); err != nil {
			log.Fatalf("create work experience sample %q: %v", workExperience.Slug, err)
		}
	}

	skillSamples := sampleSkills()
	validateSampleLimit("skills", len(skillSamples))
	for _, skill := range skillSamples {
		if err := deleteSkillIfExists(ctx, skillStore, skill.ID); err != nil {
			log.Fatalf("delete skill sample %q: %v", skill.ID, err)
		}
	}

	skillCategorySamples := sampleSkillCategories()
	validateSampleLimit("skill categories", len(skillCategorySamples))
	for _, category := range skillCategorySamples {
		if err := deleteSkillCategoryIfExists(ctx, skillStore, category.ID); err != nil {
			log.Fatalf("delete skill category sample %q: %v", category.ID, err)
		}
		if err := deleteSkillCategoryIfExists(ctx, skillStore, category.Slug); err != nil {
			log.Fatalf("delete skill category sample %q: %v", category.Slug, err)
		}
		if _, err := skillStore.CreateCategory(ctx, category); err != nil {
			log.Fatalf("create skill category sample %q: %v", category.Slug, err)
		}
	}
	for _, skill := range skillSamples {
		if _, err := skillStore.CreateSkill(ctx, skill); err != nil {
			log.Fatalf("create skill sample %q: %v", skill.ID, err)
		}
	}

	linkSamples := sampleLinks()
	validateSampleLimit("links", len(linkSamples))
	for _, link := range linkSamples {
		if err := deleteLinkIfExists(ctx, linkStore, link.ID); err != nil {
			log.Fatalf("delete link sample %q: %v", link.ID, err)
		}
		if _, err := linkStore.Create(ctx, link); err != nil {
			log.Fatalf("create link sample %q: %v", link.ID, err)
		}
	}

	toolSamples := sampleTools()
	validateSampleLimit("tools", len(toolSamples))
	for _, tool := range toolSamples {
		if err := deleteToolIfExists(ctx, toolStore, tool.ID); err != nil {
			log.Fatalf("delete tool sample %q: %v", tool.ID, err)
		}
		if _, err := toolStore.Create(ctx, tool); err != nil {
			log.Fatalf("create tool sample %q: %v", tool.ID, err)
		}
	}

	productSamples := sampleProducts()
	validateSampleLimit("products", len(productSamples))
	for _, product := range productSamples {
		if err := deleteProductIfExists(ctx, productStore, product.ID); err != nil {
			log.Fatalf("delete product sample %q: %v", product.ID, err)
		}
		if err := deleteProductIfExists(ctx, productStore, product.Slug); err != nil {
			log.Fatalf("delete product sample %q: %v", product.Slug, err)
		}
		if _, err := productStore.Create(ctx, product); err != nil {
			log.Fatalf("create product sample %q: %v", product.Slug, err)
		}
	}
	if _, err := productStore.UpdateSection(ctx, func(settings *products.ProductSectionSettings) error {
		settings.Enabled = true
		settings.Title = "Esoteric Section"
		settings.Description = "Study tools, decks, and small digital products worth keeping close."
		return nil
	}); err != nil {
		log.Fatalf("save products section sample: %v", err)
	}

	introSample := sampleIntro()
	if _, err := introStore.Save(ctx, introSample); err != nil {
		log.Fatalf("save intro sample: %v", err)
	}

	fmt.Printf("seeded %d sample projects, %d sample certificates, %d sample articles, %d sample work experiences, %d sample skill categories, %d sample skills, %d sample links, %d sample tools, %d sample products, products section, and intro\n", len(projectSamples), len(certificateSamples), len(articleSamples), len(workExperienceSamples), len(skillCategorySamples), len(skillSamples), len(linkSamples), len(toolSamples), len(productSamples))
}

func validateSampleLimit(name string, count int) {
	if count > maxSampleRecords {
		log.Fatalf("sample %s has %d records; max is %d", name, count, maxSampleRecords)
	}
}

func deleteProjectIfExists(ctx context.Context, store projects.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, projects.ErrNotFound) {
		return nil
	}
	return err
}

func deleteCertificateIfExists(ctx context.Context, store certificates.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, certificates.ErrNotFound) {
		return nil
	}
	return err
}

func deleteArticleIfExists(ctx context.Context, store articles.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, articles.ErrNotFound) {
		return nil
	}
	return err
}

func deleteWorkExperienceIfExists(ctx context.Context, store workexperience.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, workexperience.ErrNotFound) {
		return nil
	}
	return err
}

func deleteSkillCategoryIfExists(ctx context.Context, store skills.Store, idOrSlug string) error {
	err := store.DeleteCategory(ctx, idOrSlug)
	if errors.Is(err, skills.ErrNotFound) {
		return nil
	}
	return err
}

func deleteSkillIfExists(ctx context.Context, store skills.Store, id string) error {
	err := store.DeleteSkill(ctx, id)
	if errors.Is(err, skills.ErrNotFound) {
		return nil
	}
	return err
}

func deleteLinkIfExists(ctx context.Context, store links.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, links.ErrNotFound) {
		return nil
	}
	return err
}

func deleteToolIfExists(ctx context.Context, store tools.Store, id string) error {
	err := store.Delete(ctx, id)
	if errors.Is(err, tools.ErrNotFound) {
		return nil
	}
	return err
}

func deleteProductIfExists(ctx context.Context, store products.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, products.ErrNotFound) {
		return nil
	}
	return err
}

func sampleProjects() []projects.Project {
	firstPublishedAt := time.Date(2026, time.May, 1, 9, 0, 0, 0, time.UTC)
	secondPublishedAt := time.Date(2026, time.May, 8, 9, 0, 0, 0, time.UTC)

	return []projects.Project{
		{
			ID:          "sample-portfolio-api-server",
			Slug:        "portfolio-api-server",
			Title:       "Portfolio API Server",
			Summary:     "A Go API for managing portfolio projects, images, and publishing state.",
			Description: "Backend service with JWT-protected project management, PostgreSQL storage, and local or Spaces-backed image uploads.",
			Body:        "This sample demonstrates a production-oriented portfolio API with persistent project records, image metadata, and Docker-based local development.",
			TechStack: []string{
				"Go",
				"PostgreSQL",
				"Docker",
			},
			Tags: []string{
				"backend",
				"api",
				"portfolio",
			},
			MainImage: &projects.ProjectImage{
				ID:         "sample-portfolio-api-main",
				URL:        "https://picsum.photos/seed/portfolio-api/1200/800",
				AltText:    "Abstract server dashboard preview",
				Caption:    "Portfolio API server overview",
				SortOrder:  0,
				UploadedAt: firstPublishedAt,
			},
			Images: []projects.ProjectImage{
				{
					ID:         "sample-portfolio-api-gallery-1",
					URL:        "https://picsum.photos/seed/portfolio-api-detail/1200/800",
					AltText:    "Project endpoint detail preview",
					Caption:    "Project management workflow",
					SortOrder:  1,
					UploadedAt: firstPublishedAt,
				},
			},
			GitHubURL:   "https://github.com/eralmendral/Portfolio-V4",
			DemoURL:     "https://example.com/portfolio-api-server",
			Featured:    true,
			SortOrder:   10,
			Status:      projects.StatusPublished,
			CreatedAt:   firstPublishedAt,
			PublishedAt: &firstPublishedAt,
		},
		{
			ID:          "sample-portfolio-admin-dashboard",
			Slug:        "portfolio-admin-dashboard",
			Title:       "Portfolio Admin Dashboard",
			Summary:     "An admin interface concept for curating featured work and project media.",
			Description: "Sample project data for testing list, search, update, and image-management flows in the portfolio API.",
			Body:        "This seeded project gives Postman and local UI tests a second realistic record with different tags, status, and metadata.",
			TechStack: []string{
				"React",
				"TypeScript",
				"Tailwind CSS",
			},
			Tags: []string{
				"frontend",
				"dashboard",
				"admin",
			},
			MainImage: &projects.ProjectImage{
				ID:         "sample-admin-dashboard-main",
				URL:        "https://picsum.photos/seed/admin-dashboard/1200/800",
				AltText:    "Admin dashboard project preview",
				Caption:    "Portfolio admin dashboard",
				SortOrder:  0,
				UploadedAt: secondPublishedAt,
			},
			Images: []projects.ProjectImage{
				{
					ID:         "sample-admin-dashboard-gallery-1",
					URL:        "https://picsum.photos/seed/admin-dashboard-detail/1200/800",
					AltText:    "Dashboard detail preview",
					Caption:    "Editing project metadata",
					SortOrder:  1,
					UploadedAt: secondPublishedAt,
				},
			},
			GitHubURL:   "https://github.com/eralmendral/Portfolio-V4",
			DemoURL:     "https://example.com/portfolio-admin-dashboard",
			Featured:    false,
			SortOrder:   20,
			Status:      projects.StatusPublished,
			CreatedAt:   secondPublishedAt,
			PublishedAt: &secondPublishedAt,
		},
	}
}

func sampleCertificates() []certificates.Certificate {
	firstIssuedAt := time.Date(2026, time.February, 10, 9, 0, 0, 0, time.UTC)
	secondIssuedAt := time.Date(2026, time.March, 18, 9, 0, 0, 0, time.UTC)

	return []certificates.Certificate{
		{
			ID:            "sample-go-api-certificate",
			Slug:          "go-api-certificate",
			Title:         "Go API Engineering Certificate",
			Issuer:        "Open Source Academy",
			Summary:       "Credential for building production-grade Go HTTP APIs.",
			Description:   "Covers PostgreSQL-backed CRUD, JWT authentication, containerized deployment, and image upload workflows.",
			CredentialURL: "https://example.com/certificates/go-api-certificate",
			Image: &certificates.CertificateImage{
				ID:         "sample-go-api-certificate-image",
				URL:        "https://picsum.photos/seed/go-api-certificate/1200/800",
				AltText:    "Go API Engineering Certificate preview",
				Caption:    "Go API Engineering Certificate",
				UploadedAt: firstIssuedAt,
			},
			Featured:  true,
			SortOrder: 10,
			Status:    certificates.StatusPublished,
			IssuedAt:  &firstIssuedAt,
			CreatedAt: firstIssuedAt,
		},
		{
			ID:            "sample-cloud-deployment-certificate",
			Slug:          "cloud-deployment-certificate",
			Title:         "Cloud Deployment Certificate",
			Issuer:        "Portfolio Labs",
			Summary:       "Credential for Docker-based app and database deployments.",
			Description:   "Demonstrates container orchestration, environment configuration, persistent database volumes, and operational checks.",
			CredentialURL: "https://example.com/certificates/cloud-deployment-certificate",
			Image: &certificates.CertificateImage{
				ID:         "sample-cloud-deployment-certificate-image",
				URL:        "https://picsum.photos/seed/cloud-deployment-certificate/1200/800",
				AltText:    "Cloud Deployment Certificate preview",
				Caption:    "Cloud Deployment Certificate",
				UploadedAt: secondIssuedAt,
			},
			Featured:  false,
			SortOrder: 20,
			Status:    certificates.StatusPublished,
			IssuedAt:  &secondIssuedAt,
			CreatedAt: secondIssuedAt,
		},
	}
}

func sampleArticles() []articles.Article {
	firstPublishedAt := time.Date(2026, time.April, 12, 9, 0, 0, 0, time.UTC)
	secondPublishedAt := time.Date(2026, time.May, 4, 9, 0, 0, 0, time.UTC)

	return []articles.Article{
		{
			ID:            "sample-devto-go-api-routing",
			Title:         "Routing Patterns for Go Portfolio APIs",
			URL:           "https://dev.to/example/routing-patterns-for-go-portfolio-apis",
			Source:        "Dev.to",
			Summary:       "A practical walkthrough of organizing authenticated CRUD routes in a Go portfolio backend.",
			CoverImageURL: "https://picsum.photos/seed/devto-go-api-routing/1200/630",
			Featured:      true,
			SortOrder:     10,
			Status:        articles.StatusPublished,
			PublishedAt:   &firstPublishedAt,
			CreatedAt:     firstPublishedAt,
		},
		{
			ID:            "sample-medium-portfolio-content-models",
			Title:         "Content Models for Portfolio Preview Cards",
			URL:           "https://medium.com/example/content-models-for-portfolio-preview-cards",
			Source:        "Medium",
			Summary:       "How lightweight metadata can power reusable project, certificate, and article previews.",
			CoverImageURL: "https://picsum.photos/seed/medium-portfolio-content-models/1200/630",
			Featured:      false,
			SortOrder:     20,
			Status:        articles.StatusPublished,
			PublishedAt:   &secondPublishedAt,
			CreatedAt:     secondPublishedAt,
		},
	}
}

func sampleWorkExperiences() []workexperience.WorkExperience {
	firstStartedAt := time.Date(2024, time.January, 1, 9, 0, 0, 0, time.UTC)
	secondStartedAt := time.Date(2022, time.March, 1, 9, 0, 0, 0, time.UTC)
	secondEndedAt := time.Date(2023, time.December, 31, 17, 0, 0, 0, time.UTC)
	publishedAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []workexperience.WorkExperience{
		{
			ID:             "sample-arete-labs-senior-software-engineer",
			Slug:           "arete-labs-senior-software-engineer",
			Title:          "Senior Software Engineer",
			Company:        "Arete Labs",
			CompanyURL:     "https://example.com",
			CompanyLogoURL: "https://picsum.photos/seed/arete-labs-logo/512/512",
			EmploymentType: "Full-time",
			Location:       "Manila, Philippines",
			LocationType:   "Remote",
			Summary:        "Builds portfolio APIs, admin workflows, and content-management tools with practical deployment paths.",
			Description:    "Owns backend modeling, authenticated CRUD APIs, upload flows, and frontend integration details for portfolio operations.",
			Highlights: []string{
				"Shipped PostgreSQL-backed portfolio content APIs.",
				"Improved admin publishing workflows with focused validation and tests.",
			},
			Responsibilities: []string{
				"Design and implement Go HTTP APIs.",
				"Model portfolio content data and persistence behavior.",
				"Review frontend data contracts and operational workflows.",
			},
			TechStack: []string{
				"Go",
				"PostgreSQL",
				"TypeScript",
			},
			Skills: []string{
				"API Design",
				"Data Modeling",
				"Testing",
			},
			StartedAt:   firstStartedAt,
			Current:     true,
			Featured:    true,
			SortOrder:   10,
			Status:      workexperience.StatusPublished,
			PublishedAt: &publishedAt,
			CreatedAt:   firstStartedAt,
		},
		{
			ID:             "sample-northstar-systems-backend-engineer",
			Slug:           "northstar-systems-backend-engineer",
			Title:          "Backend Engineer",
			Company:        "Northstar Systems",
			CompanyURL:     "https://example.com",
			CompanyLogoURL: "https://picsum.photos/seed/northstar-logo/512/512",
			EmploymentType: "Contract",
			Location:       "Remote",
			LocationType:   "Remote",
			Summary:        "Delivered backend services and database-backed features for small product teams.",
			Description:    "Implemented HTTP APIs, relational schemas, and maintenance workflows for operational tools.",
			Highlights: []string{
				"Reduced manual data cleanup through clearer API validation.",
				"Added regression coverage for critical content workflows.",
			},
			Responsibilities: []string{
				"Build backend service endpoints.",
				"Maintain PostgreSQL data models.",
				"Collaborate on release validation.",
			},
			TechStack: []string{
				"Go",
				"PostgreSQL",
				"React",
			},
			Skills: []string{
				"Backend Engineering",
				"Reliability",
				"Code Review",
			},
			StartedAt:   secondStartedAt,
			EndedAt:     &secondEndedAt,
			Current:     false,
			Featured:    false,
			SortOrder:   20,
			Status:      workexperience.StatusPublished,
			PublishedAt: &publishedAt,
			CreatedAt:   secondStartedAt,
		},
	}
}

func sampleSkillCategories() []skills.SkillCategory {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []skills.SkillCategory{
		{
			ID:          "sample-skill-category-backend-engineering",
			Slug:        "backend-engineering",
			Name:        "Backend Engineering",
			Description: "API design, Go services, authentication, and relational data modeling.",
			IconClass:   "lucide-server",
			SortOrder:   10,
			Status:      skills.StatusPublished,
			CreatedAt:   createdAt,
		},
		{
			ID:          "sample-skill-category-frontend-engineering",
			Slug:        "frontend-engineering",
			Name:        "Frontend Engineering",
			Description: "React, TypeScript, accessible forms, and responsive admin interfaces.",
			IconClass:   "lucide-monitor",
			SortOrder:   20,
			Status:      skills.StatusPublished,
			CreatedAt:   createdAt,
		},
		{
			ID:          "sample-skill-category-cloud-devops",
			Slug:        "cloud-devops",
			Name:        "Cloud & DevOps",
			Description: "Containerized local workflows, cloud storage, deployment, and CI checks.",
			IconClass:   "lucide-cloud",
			SortOrder:   30,
			Status:      skills.StatusPublished,
			CreatedAt:   createdAt,
		},
	}
}

func sampleSkills() []skills.Skill {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []skills.Skill{
		{
			ID:         "sample-skill-api-design",
			CategoryID: "sample-skill-category-backend-engineering",
			Name:       "API Design",
			Summary:    "Designs clear HTTP resources, validation paths, and response contracts.",
			SortOrder:  10,
			Featured:   true,
			Status:     skills.StatusPublished,
			CreatedAt:  createdAt,
		},
		{
			ID:         "sample-skill-go",
			CategoryID: "sample-skill-category-backend-engineering",
			Name:       "Go",
			Summary:    "Builds practical Go services with standard-library HTTP routing and tests.",
			SortOrder:  20,
			Featured:   true,
			Status:     skills.StatusPublished,
			CreatedAt:  createdAt,
		},
		{
			ID:         "sample-skill-postgresql",
			CategoryID: "sample-skill-category-backend-engineering",
			Name:       "PostgreSQL",
			Summary:    "Models relational content, migrations, indexing, and query filtering.",
			SortOrder:  30,
			Featured:   true,
			Status:     skills.StatusPublished,
			CreatedAt:  createdAt,
		},
	}
}

func sampleLinks() []links.Link {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []links.Link{
		{
			ID:        "sample-link-github",
			Label:     "GitHub",
			URL:       "https://github.com/eralmendral",
			IconClass: "fa-brands fa-github",
			SortOrder: 10,
			Star:      true,
			Status:    links.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-link-linkedin",
			Label:     "LinkedIn",
			URL:       "https://www.linkedin.com/in/eralmendral",
			IconClass: "fa-brands fa-linkedin",
			SortOrder: 20,
			Star:      true,
			Status:    links.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-link-resume",
			Label:     "Resume",
			URL:       "https://example.com/resume.pdf",
			IconClass: "fa-solid fa-file-lines",
			SortOrder: 30,
			Star:      true,
			Status:    links.StatusPublished,
			CreatedAt: createdAt,
		},
	}
}

func sampleTools() []tools.Tool {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return []tools.Tool{
		{
			ID:        "sample-tool-codex",
			Name:      "Codex",
			Category:  "AI & Coding Assistants",
			Summary:   "Agentic coding workflow for implementing, testing, and reviewing repository changes.",
			IconClass: "lucide-bot",
			Tags: []string{
				"ai",
				"coding",
				"agent",
			},
			SortOrder: 10,
			Featured:  true,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-claude",
			Name:      "Claude",
			Category:  "AI & Coding Assistants",
			Summary:   "AI assistant for reasoning, drafting, coding support, and technical exploration.",
			IconClass: "lucide-sparkles",
			Tags: []string{
				"ai",
				"assistant",
				"reasoning",
			},
			SortOrder: 20,
			Featured:  true,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-opencode",
			Name:      "OpenCode",
			Category:  "AI & Coding Assistants",
			Summary:   "Terminal-based AI coding workflow for codebase edits and review loops.",
			IconClass: "lucide-terminal",
			Tags: []string{
				"ai",
				"terminal",
				"coding",
			},
			SortOrder: 30,
			Featured:  false,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-google-cloud-platform",
			Name:      "Google Cloud Platform",
			Category:  "Cloud & DevOps",
			Summary:   "Cloud platform for deploying, operating, and scaling production services.",
			IconClass: "lucide-cloud",
			Tags: []string{
				"cloud",
				"gcp",
				"deployment",
			},
			SortOrder: 40,
			Featured:  true,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-google-artifact-registry",
			Name:      "Google Artifact Registry",
			Category:  "Cloud & DevOps",
			Summary:   "Managed registry for storing and distributing container images and build artifacts.",
			IconClass: "lucide-package",
			Tags: []string{
				"registry",
				"containers",
				"gcp",
			},
			SortOrder: 50,
			Featured:  false,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
		{
			ID:        "sample-tool-github-container-registry",
			Name:      "GitHub Container Registry",
			Category:  "Cloud & DevOps",
			Summary:   "Container registry for publishing release images directly from GitHub Actions.",
			IconClass: "lucide-box",
			Tags: []string{
				"registry",
				"containers",
				"ghcr",
			},
			SortOrder: 60,
			Featured:  false,
			Status:    tools.StatusPublished,
			CreatedAt: createdAt,
		},
	}
}

func sampleProducts() []products.Product {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)
	publishedAt := time.Date(2026, time.May, 10, 10, 0, 0, 0, time.UTC)

	return []products.Product{
		{
			ID:            "sample-product-valuable-anki-deck-from-stranger",
			Slug:          "valuable-anki-deck-from-stranger",
			Title:         "Valuable Anki Deck from Stranger",
			Summary:       "A focused Anki deck for memorizing high-signal ideas from obscure notes.",
			Description:   "A compact deck built for daily spaced repetition, with cards organized around durable concepts, recall prompts, and practical review cadence.",
			CoverImageURL: "https://picsum.photos/seed/valuable-anki-deck/1200/800",
			PriceLabel:    "$19",
			CTALabel:      "Get the deck",
			CTAURL:        "https://example.com/valuable-anki-deck-from-stranger",
			Category:      "Anki Decks",
			Tags: []string{
				"anki",
				"study",
				"spaced-repetition",
			},
			Featured:    true,
			SortOrder:   10,
			Status:      products.StatusPublished,
			PublishedAt: &publishedAt,
			CreatedAt:   createdAt,
		},
	}
}

func sampleIntro() intro.Intro {
	createdAt := time.Date(2026, time.May, 10, 9, 0, 0, 0, time.UTC)

	return intro.Intro{
		ID:          intro.DefaultID,
		Title:       "AI Engineer",
		Description: "I build AI-enabled products, backend systems, and polished interfaces that turn model capabilities into reliable user workflows.",
		ProfilePicture: &intro.IntroImage{
			ID:         "sample-intro-profile-picture",
			URL:        "https://picsum.photos/seed/ai-engineer-profile/1200/1200",
			AltText:    "AI Engineer profile portrait",
			Caption:    "AI Engineer profile picture",
			UploadedAt: createdAt,
		},
		CreatedAt: createdAt,
	}
}
