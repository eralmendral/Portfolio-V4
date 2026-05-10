package intro

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

	introHandler := NewHandler(newMemoryStore(), UploadConfig{
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
	mux.Handle("GET /intro", http.HandlerFunc(introHandler.Handle))
	mux.Handle("PATCH /intro", requireJWT(http.HandlerFunc(introHandler.Handle)))
	mux.Handle("PUT /intro", requireJWT(http.HandlerFunc(introHandler.Handle)))
	mux.Handle("DELETE /intro", requireJWT(http.HandlerFunc(introHandler.Handle)))
	mux.Handle("POST /intro/profile-picture", requireJWT(http.HandlerFunc(introHandler.HandleProfilePicture)))
	mux.Handle("DELETE /intro/profile-picture", requireJWT(http.HandlerFunc(introHandler.HandleProfilePicture)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestIntroPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)
	createIntro(t, api)

	readResponse := api.publicRequest(t, http.MethodGet, "/intro", nil, "")
	if readResponse.Code != http.StatusOK {
		t.Fatalf("public read status = %d, want %d", readResponse.Code, http.StatusOK)
	}

	writeResponse := api.publicRequest(t, http.MethodPatch, "/intro", strings.NewReader(`{}`), "application/json")
	if writeResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized write status = %d, want %d", writeResponse.Code, http.StatusUnauthorized)
	}
}

func TestIntroUpdateGetAndDelete(t *testing.T) {
	api := newTestAPI(t)

	updateBody := `{
		"title": "Software Engineer",
		"description": "I build backend systems and polished web experiences."
	}`
	updateResponse := api.request(t, http.MethodPatch, "/intro", strings.NewReader(updateBody), "application/json")
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d: %s", updateResponse.Code, http.StatusOK, updateResponse.Body.String())
	}

	var updated Intro
	decodeResponse(t, updateResponse, &updated)
	if updated.ID != DefaultID {
		t.Fatalf("id = %q, want %q", updated.ID, DefaultID)
	}
	if updated.Title != "Software Engineer" {
		t.Fatalf("title = %q, want Software Engineer", updated.Title)
	}
	if updated.Description == "" {
		t.Fatal("description should be set")
	}

	getResponse := api.publicRequest(t, http.MethodGet, "/intro", nil, "")
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/intro", nil, "")
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}

	missingResponse := api.publicRequest(t, http.MethodGet, "/intro", nil, "")
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missingResponse.Code, http.StatusNotFound)
	}
}

func TestUploadProfilePicture(t *testing.T) {
	api := newTestAPI(t)
	createIntro(t, api)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("alt_text", "Profile portrait"); err != nil {
		t.Fatalf("write field: %v", err)
	}

	part, err := writer.CreateFormFile("image", "profile.png")
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(testPNG(t)); err != nil {
		t.Fatalf("write png: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	response := api.request(t, http.MethodPost, "/intro/profile-picture", &body, writer.FormDataContentType())
	if response.Code != http.StatusOK {
		t.Fatalf("upload status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var updated Intro
	decodeResponse(t, response, &updated)
	if updated.ProfilePicture == nil {
		t.Fatal("profile_picture should be set")
	}
	if updated.ProfilePicture.AltText != "Profile portrait" {
		t.Fatalf("alt_text = %q, want Profile portrait", updated.ProfilePicture.AltText)
	}
	if updated.ProfilePicture.ContentType != "image/png" {
		t.Fatalf("content_type = %q, want image/png", updated.ProfilePicture.ContentType)
	}
	if updated.ProfilePicture.Width != 2 || updated.ProfilePicture.Height != 1 {
		t.Fatalf("dimensions = %dx%d, want 2x1", updated.ProfilePicture.Width, updated.ProfilePicture.Height)
	}
}

func createIntro(t *testing.T, api testAPI) Intro {
	t.Helper()

	body := `{
		"title": "Software Engineer",
		"description": "I build backend systems and polished web experiences."
	}`

	response := api.request(t, http.MethodPatch, "/intro", strings.NewReader(body), "application/json")
	if response.Code != http.StatusOK {
		t.Fatalf("create intro status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var intro Intro
	decodeResponse(t, response, &intro)
	return intro
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

func (api testAPI) publicRequest(t *testing.T, method string, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, body)
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
