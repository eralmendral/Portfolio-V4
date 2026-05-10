package projects

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eralme/server/internal/auth"
)

type testAPI struct {
	handler http.Handler
	token   string
}

func newTestAPI(t *testing.T) testAPI {
	t.Helper()

	uploadDir := t.TempDir()

	projectHandler := NewHandler(newMemoryStore(), UploadConfig{
		Dir:      uploadDir,
		BaseURL:  "/uploads/projects",
		MaxBytes: 1 << 20,
	})

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /projects", requireJWT(http.HandlerFunc(projectHandler.HandleCollection)))
	mux.Handle("POST /projects", requireJWT(http.HandlerFunc(projectHandler.HandleCollection)))
	mux.Handle("/projects/", requireJWT(http.HandlerFunc(projectHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestProjectsRequireJWT(t *testing.T) {
	api := newTestAPI(t)

	request := httptest.NewRequest(http.MethodGet, "/projects", nil)
	response := httptest.NewRecorder()

	api.handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestProjectCRUD(t *testing.T) {
	api := newTestAPI(t)

	project := createProject(t, api)
	if project.Title != "Portfolio API" {
		t.Fatalf("title = %q, want Portfolio API", project.Title)
	}
	if project.Slug != "portfolio-api" {
		t.Fatalf("slug = %q, want portfolio-api", project.Slug)
	}
	if project.CreatedAt.IsZero() {
		t.Fatal("created_at should be set")
	}

	getResponse := api.request(t, http.MethodGet, "/projects/"+project.Slug, nil, "")
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	listResponse := api.request(t, http.MethodGet, "/projects?status=published&q=portfolio", nil, "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Projects []Project `json:"projects"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Projects) != 1 {
		t.Fatalf("projects count = %d, want 1", len(list.Projects))
	}

	patchBody := `{"summary":"Updated summary","featured":false,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/projects/"+project.ID, strings.NewReader(patchBody), "application/json")
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated Project
	decodeResponse(t, patchResponse, &updated)
	if updated.Summary != "Updated summary" || updated.Featured {
		t.Fatalf("unexpected updated project: %+v", updated)
	}
	if updated.Status != StatusDraft {
		t.Fatalf("status = %q, want %q", updated.Status, StatusDraft)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/projects/"+project.ID, nil, "")
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}

	missingResponse := api.request(t, http.MethodGet, "/projects/"+project.ID, nil, "")
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missingResponse.Code, http.StatusNotFound)
	}
}

func TestUploadMainProjectImage(t *testing.T) {
	api := newTestAPI(t)
	project := createProject(t, api)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("alt_text", "Project thumbnail"); err != nil {
		t.Fatalf("write field: %v", err)
	}

	part, err := writer.CreateFormFile("image", "thumbnail.png")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(testPNG(t)); err != nil {
		t.Fatalf("write png: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	response := api.request(t, http.MethodPost, "/projects/"+project.ID+"/images/main", &body, writer.FormDataContentType())
	if response.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var updated Project
	decodeResponse(t, response, &updated)
	if updated.MainImage == nil {
		t.Fatal("main_image should be set")
	}
	if updated.MainImage.AltText != "Project thumbnail" {
		t.Fatalf("alt_text = %q, want Project thumbnail", updated.MainImage.AltText)
	}
	if updated.MainImage.ContentType != "image/png" {
		t.Fatalf("content_type = %q, want image/png", updated.MainImage.ContentType)
	}
	if updated.MainImage.Width != 2 || updated.MainImage.Height != 1 {
		t.Fatalf("dimensions = %dx%d, want 2x1", updated.MainImage.Width, updated.MainImage.Height)
	}
}

func TestUploadRejectsInvalidImageFileType(t *testing.T) {
	api := newTestAPI(t)
	project := createProject(t, api)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", "thumbnail.gif")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(testPNG(t)); err != nil {
		t.Fatalf("write png: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	response := api.request(t, http.MethodPost, "/projects/"+project.ID+"/images/main", &body, writer.FormDataContentType())
	if response.Code != http.StatusBadRequest {
		t.Fatalf("upload status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}
}

func TestUploadDefaultsTo300MiBLimit(t *testing.T) {
	config := UploadConfig{}.withDefaults()
	if config.MaxBytes != 300<<20 {
		t.Fatalf("max bytes = %d, want %d", config.MaxBytes, 300<<20)
	}
}

func createProject(t *testing.T, api testAPI) Project {
	t.Helper()

	body := `{
		"title": "Portfolio API",
		"summary": "Backend service for the portfolio",
		"tech_stack": ["Go", "net/http", "Go"],
		"github_url": "https://github.com/example/portfolio",
		"demo_url": "https://example.com",
		"featured": true,
		"status": "published"
	}`

	response := api.request(t, http.MethodPost, "/projects", strings.NewReader(body), "application/json")
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var project Project
	decodeResponse(t, response, &project)
	return project
}

func (api testAPI) request(t *testing.T, method string, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, body)
	request.Header.Set("Authorization", "Bearer "+api.token)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	response := httptest.NewRecorder()
	api.handler.ServeHTTP(response, request)
	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func testPNG(t *testing.T) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{B: 255, A: 255})

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buffer.Bytes()
}
