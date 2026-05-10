package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eralme/server/internal/auth"
	"github.com/eralme/server/internal/certificates"
	"github.com/eralme/server/internal/intro"
	"github.com/eralme/server/internal/links"
	"github.com/eralme/server/internal/projects"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type config struct {
	Addr           string
	DatabaseURL    string
	UploadStorage  string
	UploadDir      string
	UploadBaseURL  string
	SpacesBucket   string
	SpacesRegion   string
	SpacesEndpoint string
	SpacesKey      string
	SpacesSecret   string
	SpacesBaseURL  string
	SpacesACL      string
	JWTSecret      string
	JWTIssuer      string
	TokenTTL       time.Duration
	AdminUsername  string
	AdminPassword  string
	MaxUploadBytes int64
}

func main() {
	cfg := loadConfig()

	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := openDatabase(startupCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	projectStore, err := projects.NewPostgresStore(startupCtx, db)
	if err != nil {
		log.Fatal(err)
	}

	certificateStore, err := certificates.NewPostgresStore(startupCtx, db)
	if err != nil {
		log.Fatal(err)
	}

	linkStore, err := links.NewPostgresStore(startupCtx, db)
	if err != nil {
		log.Fatal(err)
	}

	introStore, err := intro.NewPostgresStore(startupCtx, db)
	if err != nil {
		log.Fatal(err)
	}

	router, err := buildRouter(cfg, projectStore, certificateStore, linkStore, introStore)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("portfolio server listening on %s", cfg.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

func buildRouter(cfg config, projectStore projects.Store, certificateStore certificates.Store, linkStore links.Store, introStore intro.Store) (http.Handler, error) {
	tokenService := auth.NewTokenService(cfg.JWTSecret, cfg.JWTIssuer, cfg.TokenTTL)

	assetStore, err := uploadStore(cfg)
	if err != nil {
		return nil, err
	}

	projectHandler := projects.NewHandler(projectStore, projects.UploadConfig{
		Store:    assetStore,
		MaxBytes: cfg.MaxUploadBytes,
	})
	certificateHandler := certificates.NewHandler(certificateStore, certificates.UploadConfig{
		Store:    assetStore,
		MaxBytes: cfg.MaxUploadBytes,
	})
	linkHandler := links.NewHandler(linkStore)
	introHandler := intro.NewHandler(introStore, intro.UploadConfig{
		Store:    assetStore,
		MaxBytes: cfg.MaxUploadBytes,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.Handle("POST /auth/login", auth.LoginHandler(tokenService, cfg.AdminUsername, cfg.AdminPassword))

	requireJWT := auth.RequireJWT(tokenService)
	mux.Handle("GET /projects", http.HandlerFunc(projectHandler.HandleCollection))
	mux.Handle("POST /projects", requireJWT(http.HandlerFunc(projectHandler.HandleCollection)))
	mux.Handle("GET /projects/", http.HandlerFunc(projectHandler.HandleItem))
	mux.Handle("POST /projects/", requireJWT(http.HandlerFunc(projectHandler.HandleItem)))
	mux.Handle("PATCH /projects/", requireJWT(http.HandlerFunc(projectHandler.HandleItem)))
	mux.Handle("PUT /projects/", requireJWT(http.HandlerFunc(projectHandler.HandleItem)))
	mux.Handle("DELETE /projects/", requireJWT(http.HandlerFunc(projectHandler.HandleItem)))
	mux.Handle("GET /certificates", http.HandlerFunc(certificateHandler.HandleCollection))
	mux.Handle("POST /certificates", requireJWT(http.HandlerFunc(certificateHandler.HandleCollection)))
	mux.Handle("GET /certificates/", http.HandlerFunc(certificateHandler.HandleItem))
	mux.Handle("POST /certificates/", requireJWT(http.HandlerFunc(certificateHandler.HandleItem)))
	mux.Handle("PATCH /certificates/", requireJWT(http.HandlerFunc(certificateHandler.HandleItem)))
	mux.Handle("PUT /certificates/", requireJWT(http.HandlerFunc(certificateHandler.HandleItem)))
	mux.Handle("DELETE /certificates/", requireJWT(http.HandlerFunc(certificateHandler.HandleItem)))
	mux.Handle("GET /links", http.HandlerFunc(linkHandler.HandleCollection))
	mux.Handle("POST /links", requireJWT(http.HandlerFunc(linkHandler.HandleCollection)))
	mux.Handle("GET /links/", http.HandlerFunc(linkHandler.HandleItem))
	mux.Handle("PATCH /links/", requireJWT(http.HandlerFunc(linkHandler.HandleItem)))
	mux.Handle("PUT /links/", requireJWT(http.HandlerFunc(linkHandler.HandleItem)))
	mux.Handle("DELETE /links/", requireJWT(http.HandlerFunc(linkHandler.HandleItem)))
	mux.Handle("GET /intro", http.HandlerFunc(introHandler.Handle))
	mux.Handle("PATCH /intro", requireJWT(http.HandlerFunc(introHandler.Handle)))
	mux.Handle("PUT /intro", requireJWT(http.HandlerFunc(introHandler.Handle)))
	mux.Handle("DELETE /intro", requireJWT(http.HandlerFunc(introHandler.Handle)))
	mux.Handle("POST /intro/profile-picture", requireJWT(http.HandlerFunc(introHandler.HandleProfilePicture)))
	mux.Handle("DELETE /intro/profile-picture", requireJWT(http.HandlerFunc(introHandler.HandleProfilePicture)))

	if cfg.UploadStorage == "local" && cfg.UploadDir != "" {
		mux.Handle("/uploads/projects/", http.StripPrefix("/uploads/projects/", http.FileServer(http.Dir(cfg.UploadDir))))
	}

	return withCommonHeaders(mux), nil
}

func openDatabase(ctx context.Context, databaseURL string) (*sql.DB, error) {
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	return db, nil
}

func loadConfig() config {
	secret := getenv("JWT_SECRET", "development-secret-change-me")
	if secret == "development-secret-change-me" {
		log.Print("JWT_SECRET is not set; using an insecure development secret")
	}

	return config{
		Addr:           getenv("ADDR", ":8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		UploadStorage:  strings.ToLower(getenv("UPLOAD_STORAGE", "local")),
		UploadDir:      getenv("UPLOAD_DIR", filepath.Join("public", "uploads", "projects")),
		UploadBaseURL:  getenv("UPLOAD_BASE_URL", "/uploads/projects"),
		SpacesBucket:   os.Getenv("DO_SPACES_BUCKET"),
		SpacesRegion:   os.Getenv("DO_SPACES_REGION"),
		SpacesEndpoint: os.Getenv("DO_SPACES_ENDPOINT"),
		SpacesKey:      os.Getenv("DO_SPACES_KEY"),
		SpacesSecret:   os.Getenv("DO_SPACES_SECRET"),
		SpacesBaseURL:  os.Getenv("DO_SPACES_PUBLIC_BASE_URL"),
		SpacesACL:      getenv("DO_SPACES_ACL", "public-read"),
		JWTSecret:      secret,
		JWTIssuer:      getenv("JWT_ISSUER", "portfolio-server"),
		TokenTTL:       durationFromEnv("TOKEN_TTL", 24*time.Hour),
		AdminUsername:  os.Getenv("ADMIN_USERNAME"),
		AdminPassword:  os.Getenv("ADMIN_PASSWORD"),
		MaxUploadBytes: int64FromEnv("MAX_UPLOAD_BYTES", 300<<20),
	}
}

func uploadStore(cfg config) (projects.AssetStore, error) {
	switch cfg.UploadStorage {
	case "", "local":
		return projects.LocalAssetStore{
			Dir:     cfg.UploadDir,
			BaseURL: cfg.UploadBaseURL,
		}, nil
	case "spaces", "digitalocean", "digitalocean-spaces":
		store := projects.SpacesAssetStore{
			Bucket:          cfg.SpacesBucket,
			Region:          cfg.SpacesRegion,
			Endpoint:        cfg.SpacesEndpoint,
			AccessKeyID:     cfg.SpacesKey,
			SecretAccessKey: cfg.SpacesSecret,
			PublicBaseURL:   cfg.SpacesBaseURL,
			ACL:             cfg.SpacesACL,
		}
		return store, store.Validate()
	default:
		return nil, fmt.Errorf("unsupported UPLOAD_STORAGE %q", cfg.UploadStorage)
	}
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationFromEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		log.Printf("invalid %s=%q; using %s", key, value, fallback)
		return fallback
	}

	return parsed
}

func int64FromEnv(key string, fallback int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		log.Printf("invalid %s=%q; using %d", key, value, fallback)
		return fallback
	}

	return parsed
}
