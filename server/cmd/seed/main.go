package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/eralme/server/internal/certificates"
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

	projectStore, err := projects.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate projects postgres: %v", err)
	}
	certificateStore, err := certificates.NewPostgresStore(ctx, db)
	if err != nil {
		log.Fatalf("migrate certificates postgres: %v", err)
	}

	projectSamples := sampleProjects()
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

	fmt.Printf("seeded %d sample projects and %d sample certificates\n", len(projectSamples), len(certificateSamples))
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
