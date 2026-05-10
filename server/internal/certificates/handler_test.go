package certificates

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

	certificateHandler := NewHandler(newMemoryStore(), UploadConfig{
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
	mux.Handle("GET /certificates", requireJWT(http.HandlerFunc(certificateHandler.HandleCollection)))
	mux.Handle("POST /certificates", requireJWT(http.HandlerFunc(certificateHandler.HandleCollection)))
	mux.Handle("/certificates/", requireJWT(http.HandlerFunc(certificateHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestCertificatesRequireJWT(t *testing.T) {
	api := newTestAPI(t)

	request := httptest.NewRequest(http.MethodGet, "/certificates", nil)
	response := httptest.NewRecorder()

	api.handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCertificateCRUD(t *testing.T) {
	api := newTestAPI(t)

	certificate := createCertificate(t, api)
	if certificate.Title != "Go Professional Certificate" {
		t.Fatalf("title = %q, want Go Professional Certificate", certificate.Title)
	}
	if certificate.Slug != "go-professional-certificate" {
		t.Fatalf("slug = %q, want go-professional-certificate", certificate.Slug)
	}
	if certificate.SortOrder != 10 {
		t.Fatalf("sort_order = %d, want 10", certificate.SortOrder)
	}
	if certificate.CreatedAt.IsZero() {
		t.Fatal("created_at should be set")
	}

	getResponse := api.request(t, http.MethodGet, "/certificates/"+certificate.Slug, nil, "")
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	listResponse := api.request(t, http.MethodGet, "/certificates?status=published&q=professional", nil, "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Certificates []Certificate `json:"certificates"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Certificates) != 1 {
		t.Fatalf("certificates count = %d, want 1", len(list.Certificates))
	}

	patchBody := `{"summary":"Updated summary","featured":false,"sort_order":2,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/certificates/"+certificate.ID, strings.NewReader(patchBody), "application/json")
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated Certificate
	decodeResponse(t, patchResponse, &updated)
	if updated.Summary != "Updated summary" || updated.Featured || updated.SortOrder != 2 {
		t.Fatalf("unexpected updated certificate: %+v", updated)
	}
	if updated.Status != StatusDraft {
		t.Fatalf("status = %q, want %q", updated.Status, StatusDraft)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/certificates/"+certificate.ID, nil, "")
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}

	missingResponse := api.request(t, http.MethodGet, "/certificates/"+certificate.ID, nil, "")
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missingResponse.Code, http.StatusNotFound)
	}
}

func TestUploadCertificateImage(t *testing.T) {
	api := newTestAPI(t)
	certificate := createCertificate(t, api)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("alt_text", "Certificate image"); err != nil {
		t.Fatalf("write field: %v", err)
	}

	part, err := writer.CreateFormFile("image", "certificate.png")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(testPNG(t)); err != nil {
		t.Fatalf("write png: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	response := api.request(t, http.MethodPost, "/certificates/"+certificate.ID+"/image", &body, writer.FormDataContentType())
	if response.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var updated Certificate
	decodeResponse(t, response, &updated)
	if updated.Image == nil {
		t.Fatal("image should be set")
	}
	if updated.Image.AltText != "Certificate image" {
		t.Fatalf("alt_text = %q, want Certificate image", updated.Image.AltText)
	}
	if updated.Image.ContentType != "image/png" {
		t.Fatalf("content_type = %q, want image/png", updated.Image.ContentType)
	}
	if updated.Image.Width != 2 || updated.Image.Height != 1 {
		t.Fatalf("dimensions = %dx%d, want 2x1", updated.Image.Width, updated.Image.Height)
	}
}

func createCertificate(t *testing.T, api testAPI) Certificate {
	t.Helper()

	body := `{
		"title": "Go Professional Certificate",
		"issuer": "Open Source Academy",
		"summary": "Certificate for production Go services",
		"credential_url": "https://example.com/certificates/go-professional",
		"featured": true,
		"sort_order": 10,
		"status": "published"
	}`

	response := api.request(t, http.MethodPost, "/certificates", strings.NewReader(body), "application/json")
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var certificate Certificate
	decodeResponse(t, response, &certificate)
	return certificate
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
