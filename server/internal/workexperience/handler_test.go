package workexperience

import (
	"encoding/json"
	"io"
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

	workExperienceHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /work-experiences", http.HandlerFunc(workExperienceHandler.HandleCollection))
	mux.Handle("POST /work-experiences", requireJWT(http.HandlerFunc(workExperienceHandler.HandleCollection)))
	mux.Handle("GET /work-experiences/", http.HandlerFunc(workExperienceHandler.HandleItem))
	mux.Handle("PATCH /work-experiences/", requireJWT(http.HandlerFunc(workExperienceHandler.HandleItem)))
	mux.Handle("PUT /work-experiences/", requireJWT(http.HandlerFunc(workExperienceHandler.HandleItem)))
	mux.Handle("DELETE /work-experiences/", requireJWT(http.HandlerFunc(workExperienceHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestWorkExperienceWritesRequireJWT(t *testing.T) {
	api := newTestAPI(t)

	request := httptest.NewRequest(http.MethodPost, "/work-experiences", strings.NewReader(`{}`))
	response := httptest.NewRecorder()

	api.handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestWorkExperienceReadRoutesArePublic(t *testing.T) {
	api := newTestAPI(t)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/work-experiences", nil)

	api.handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestWorkExperienceCRUD(t *testing.T) {
	api := newTestAPI(t)

	workExperience := createWorkExperience(t, api)
	if workExperience.Title != "Senior Software Engineer" {
		t.Fatalf("title = %q, want Senior Software Engineer", workExperience.Title)
	}
	if workExperience.Slug != "arete-labs-senior-software-engineer" {
		t.Fatalf("slug = %q, want generated company-title slug", workExperience.Slug)
	}
	if !workExperience.Current {
		t.Fatal("current should be true")
	}
	if workExperience.EndedAt != nil {
		t.Fatal("ended_at should be empty for current experience")
	}
	if len(workExperience.Highlights) != 2 {
		t.Fatalf("highlights count = %d, want 2", len(workExperience.Highlights))
	}
	if workExperience.PublishedAt == nil {
		t.Fatal("published_at should be set")
	}
	if workExperience.CreatedAt.IsZero() {
		t.Fatal("created_at should be set")
	}

	getResponse := api.request(t, http.MethodGet, "/work-experiences/"+workExperience.Slug, nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	listResponse := api.request(t, http.MethodGet, "/work-experiences?status=published&q=go&current=true", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		WorkExperiences []WorkExperience `json:"work_experiences"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.WorkExperiences) != 1 {
		t.Fatalf("work experiences count = %d, want 1", len(list.WorkExperiences))
	}

	patchBody := `{
		"summary": "Updated role summary",
		"ended_at": "2026-05-01T00:00:00Z",
		"status": "draft"
	}`
	patchResponse := api.request(t, http.MethodPatch, "/work-experiences/"+workExperience.ID, strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var patched WorkExperience
	decodeResponse(t, patchResponse, &patched)
	if patched.Summary != "Updated role summary" || patched.Current {
		t.Fatalf("unexpected patched work experience: %+v", patched)
	}
	if patched.EndedAt == nil {
		t.Fatal("ended_at should be set")
	}
	if patched.Status != StatusDraft {
		t.Fatalf("status = %q, want %q", patched.Status, StatusDraft)
	}
	if patched.PublishedAt != nil {
		t.Fatal("published_at should clear when status is not published")
	}

	putBody := `{
		"slug": "arete-labs-staff-engineer",
		"title": "Staff Software Engineer",
		"company": "Arete Labs",
		"company_url": "https://example.com",
		"employment_type": "Full-time",
		"location": "Manila, Philippines",
		"location_type": "Hybrid",
		"summary": "Replaced role summary",
		"description": "Owned API and web delivery for portfolio systems.",
		"highlights": ["Shipped admin workflows"],
		"responsibilities": ["Designed backend APIs"],
		"tech_stack": ["Go", "PostgreSQL"],
		"skills": ["Architecture"],
		"started_at": "2024-01-01T00:00:00Z",
		"current": true,
		"featured": false,
		"sort_order": 5,
		"status": "published"
	}`
	putResponse := api.request(t, http.MethodPut, "/work-experiences/"+workExperience.ID, strings.NewReader(putBody), "application/json", true)
	if putResponse.Code != http.StatusOK {
		t.Fatalf("put status = %d, want %d: %s", putResponse.Code, http.StatusOK, putResponse.Body.String())
	}
	var replaced WorkExperience
	decodeResponse(t, putResponse, &replaced)
	if replaced.Title != "Staff Software Engineer" || replaced.SortOrder != 5 || !replaced.Current {
		t.Fatalf("unexpected replaced work experience: %+v", replaced)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/work-experiences/"+replaced.Slug, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}

	missingResponse := api.request(t, http.MethodGet, "/work-experiences/"+replaced.ID, nil, "", false)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missingResponse.Code, http.StatusNotFound)
	}
}

func TestWorkExperienceListFilteringSearchAndOrdering(t *testing.T) {
	api := newTestAPI(t)

	older := createWorkExperienceFromBody(t, api, `{
		"title": "Backend Engineer",
		"company": "Northstar Systems",
		"summary": "Built API services with Go.",
		"tech_stack": ["Go", "PostgreSQL"],
		"started_at": "2022-01-01T00:00:00Z",
		"ended_at": "2023-12-31T00:00:00Z",
		"featured": true,
		"sort_order": 10,
		"status": "published"
	}`)
	current := createWorkExperienceFromBody(t, api, `{
		"title": "Lead Software Engineer",
		"company": "Arete Labs",
		"summary": "Leads portfolio platform delivery.",
		"responsibilities": ["Mentor engineers", "Review architecture"],
		"started_at": "2024-01-01T00:00:00Z",
		"current": true,
		"featured": true,
		"sort_order": 10,
		"status": "published"
	}`)
	laterSort := createWorkExperienceFromBody(t, api, `{
		"title": "Consulting Engineer",
		"company": "Orbit Studio",
		"summary": "Short consulting engagement.",
		"started_at": "2025-01-01T00:00:00Z",
		"current": true,
		"featured": false,
		"sort_order": 20,
		"status": "published"
	}`)

	listResponse := api.request(t, http.MethodGet, "/work-experiences?status=published", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		WorkExperiences []WorkExperience `json:"work_experiences"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.WorkExperiences) != 3 {
		t.Fatalf("work experiences count = %d, want 3", len(list.WorkExperiences))
	}
	wantOrder := []string{current.ID, older.ID, laterSort.ID}
	for i, wantID := range wantOrder {
		if list.WorkExperiences[i].ID != wantID {
			t.Fatalf("work_experience[%d] = %q, want %q", i, list.WorkExperiences[i].ID, wantID)
		}
	}

	filteredResponse := api.request(t, http.MethodGet, "/work-experiences?featured=true&q=northstar", nil, "", false)
	if filteredResponse.Code != http.StatusOK {
		t.Fatalf("filtered status = %d, want %d", filteredResponse.Code, http.StatusOK)
	}
	var filtered struct {
		WorkExperiences []WorkExperience `json:"work_experiences"`
	}
	decodeResponse(t, filteredResponse, &filtered)
	if len(filtered.WorkExperiences) != 1 || filtered.WorkExperiences[0].ID != older.ID {
		t.Fatalf("filtered work experiences = %+v, want only %q", filtered.WorkExperiences, older.ID)
	}
}

func TestWorkExperienceValidation(t *testing.T) {
	api := newTestAPI(t)

	missingRequired := api.request(t, http.MethodPost, "/work-experiences", strings.NewReader(`{}`), "application/json", true)
	if missingRequired.Code != http.StatusBadRequest {
		t.Fatalf("missing required status = %d, want %d", missingRequired.Code, http.StatusBadRequest)
	}

	invalidRange := api.request(t, http.MethodPost, "/work-experiences", strings.NewReader(`{
		"title": "Engineer",
		"company": "Example Co",
		"started_at": "2026-01-01T00:00:00Z",
		"ended_at": "2025-01-01T00:00:00Z"
	}`), "application/json", true)
	if invalidRange.Code != http.StatusBadRequest {
		t.Fatalf("invalid range status = %d, want %d", invalidRange.Code, http.StatusBadRequest)
	}

	currentWithEnd := api.request(t, http.MethodPost, "/work-experiences", strings.NewReader(`{
		"title": "Engineer",
		"company": "Example Co",
		"started_at": "2024-01-01T00:00:00Z",
		"ended_at": "2025-01-01T00:00:00Z",
		"current": true
	}`), "application/json", true)
	if currentWithEnd.Code != http.StatusBadRequest {
		t.Fatalf("current with end status = %d, want %d", currentWithEnd.Code, http.StatusBadRequest)
	}
}

func createWorkExperience(t *testing.T, api testAPI) WorkExperience {
	t.Helper()

	return createWorkExperienceFromBody(t, api, `{
		"title": "Senior Software Engineer",
		"company": "Arete Labs",
		"company_url": "https://example.com",
		"company_logo_url": "https://images.example.com/arete-logo.png",
		"employment_type": "Full-time",
		"location": "Manila, Philippines",
		"location_type": "Remote",
		"summary": "Built portfolio APIs and admin workflows.",
		"description": "Led backend delivery for portfolio content management and publishing workflows.",
		"highlights": ["Reduced admin publishing time", "Improved API reliability"],
		"responsibilities": ["Designed Go services", "Reviewed TypeScript UI flows"],
		"tech_stack": ["Go", "PostgreSQL", "TypeScript"],
		"skills": ["API Design", "Testing"],
		"started_at": "2024-01-01T00:00:00Z",
		"current": true,
		"featured": true,
		"sort_order": 10,
		"status": "published"
	}`)
}

func createWorkExperienceFromBody(t *testing.T, api testAPI, body string) WorkExperience {
	t.Helper()

	response := api.request(t, http.MethodPost, "/work-experiences", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var workExperience WorkExperience
	decodeResponse(t, response, &workExperience)
	return workExperience
}

func (api testAPI) request(t *testing.T, method string, path string, body io.Reader, contentType string, authenticated bool) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, body)
	if authenticated {
		request.Header.Set("Authorization", "Bearer "+api.token)
	}
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
