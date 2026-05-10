package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/eralme/server/internal/projects"
	_ "github.com/jackc/pgx/v5/stdlib"
)

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

	store, err := projects.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate postgres: %v", err)
	}

	samples := sampleProjects()
	for _, project := range samples {
		if err := deleteIfExists(ctx, store, project.ID); err != nil {
			log.Fatalf("delete sample %q: %v", project.ID, err)
		}
		if err := deleteIfExists(ctx, store, project.Slug); err != nil {
			log.Fatalf("delete sample %q: %v", project.Slug, err)
		}
		if _, err := store.Create(ctx, project); err != nil {
			log.Fatalf("create sample %q: %v", project.Slug, err)
		}
	}

	fmt.Printf("seeded %d sample projects\n", len(samples))
}

func deleteIfExists(ctx context.Context, store projects.Store, idOrSlug string) error {
	err := store.Delete(ctx, idOrSlug)
	if errors.Is(err, projects.ErrNotFound) {
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
			Status:      projects.StatusPublished,
			CreatedAt:   secondPublishedAt,
			PublishedAt: &secondPublishedAt,
		},
	}
}
