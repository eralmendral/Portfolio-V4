package dailyprogress

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

	progressHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /daily-progress", http.HandlerFunc(progressHandler.HandleCollection))
	mux.Handle("POST /daily-progress", requireJWT(http.HandlerFunc(progressHandler.HandleCollection)))
	mux.Handle("GET /daily-progress/", http.HandlerFunc(progressHandler.HandleItem))
	mux.Handle("PATCH /daily-progress/", requireJWT(http.HandlerFunc(progressHandler.HandleItem)))
	mux.Handle("PUT /daily-progress/", requireJWT(http.HandlerFunc(progressHandler.HandleItem)))
	mux.Handle("DELETE /daily-progress/", requireJWT(http.HandlerFunc(progressHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestDailyProgressPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)

	getResponse := api.request(t, http.MethodGet, "/daily-progress", nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("public get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	postResponse := api.request(t, http.MethodPost, "/daily-progress", strings.NewReader(`{}`), "application/json", false)
	if postResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized post status = %d, want %d", postResponse.Code, http.StatusUnauthorized)
	}
}

func TestDailyProgressCRUDAndFilters(t *testing.T) {
	api := newTestAPI(t)

	first := createDailyProgress(t, api, `{
		"entry_date": "2026-05-09",
		"title": "Set up CI checks",
		"summary": "Quality gates are now visible.",
		"content": "Added formatting, linting, tests, coverage, and security scanning.",
		"mood": "focused",
		"progress_score": 72,
		"wins": ["CI runs on pushes", "coverage is reported", "CI runs on pushes"],
		"blockers": ["gosec findings need triage"],
		"learnings": ["Release tags can drive image publishing"],
		"next_steps": ["Clean up security findings"],
		"tags": ["ci", "backend", "ci"],
		"status": "published"
	}`)
	second := createDailyProgress(t, api, `{
		"entry_date": "2026-05-10",
		"title": "Added journal planning",
		"summary": "Daily progress now has a backend shape.",
		"content": "Designed a motivating progress track record.",
		"mood": "motivated",
		"progress_score": 84,
		"wins": ["Clear backend plan"],
		"learnings": ["Daily entries need unique dates"],
		"next_steps": ["Build UI"],
		"tags": ["journal", "backend"],
		"status": "published"
	}`)
	draft := createDailyProgress(t, api, `{
		"entry_date": "2026-05-11",
		"title": "Draft private note",
		"progress_score": 10
	}`)

	if first.EntryDate != "2026-05-09" || first.ProgressScore != 72 || len(first.Wins) != 2 {
		t.Fatalf("unexpected first entry: %+v", first)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("default status = %q, want %q", draft.Status, StatusDraft)
	}

	listResponse := api.request(t, http.MethodGet, "/daily-progress", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		DailyProgress []DailyProgress `json:"daily_progress"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.DailyProgress) != 2 {
		t.Fatalf("daily progress count = %d, want 2", len(list.DailyProgress))
	}
	if list.DailyProgress[0].ID != second.ID || list.DailyProgress[1].ID != first.ID {
		t.Fatalf("unexpected order: %+v", list.DailyProgress)
	}

	tagResponse := api.request(t, http.MethodGet, "/daily-progress?tag=journal", nil, "", false)
	if tagResponse.Code != http.StatusOK {
		t.Fatalf("tag status = %d, want %d", tagResponse.Code, http.StatusOK)
	}
	var tagFiltered struct {
		DailyProgress []DailyProgress `json:"daily_progress"`
	}
	decodeResponse(t, tagResponse, &tagFiltered)
	if len(tagFiltered.DailyProgress) != 1 || tagFiltered.DailyProgress[0].ID != second.ID {
		t.Fatalf("unexpected tag entries: %+v", tagFiltered.DailyProgress)
	}

	rangeResponse := api.request(t, http.MethodGet, "/daily-progress?from=2026-05-10&to=2026-05-10", nil, "", false)
	if rangeResponse.Code != http.StatusOK {
		t.Fatalf("range status = %d, want %d", rangeResponse.Code, http.StatusOK)
	}
	var rangeFiltered struct {
		DailyProgress []DailyProgress `json:"daily_progress"`
	}
	decodeResponse(t, rangeResponse, &rangeFiltered)
	if len(rangeFiltered.DailyProgress) != 1 || rangeFiltered.DailyProgress[0].ID != second.ID {
		t.Fatalf("unexpected range entries: %+v", rangeFiltered.DailyProgress)
	}

	searchResponse := api.request(t, http.MethodGet, "/daily-progress?q=security", nil, "", false)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		DailyProgress []DailyProgress `json:"daily_progress"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.DailyProgress) != 1 || search.DailyProgress[0].ID != first.ID {
		t.Fatalf("unexpected search entries: %+v", search.DailyProgress)
	}

	getByDateResponse := api.request(t, http.MethodGet, "/daily-progress/2026-05-10", nil, "", false)
	if getByDateResponse.Code != http.StatusOK {
		t.Fatalf("get by date status = %d, want %d", getByDateResponse.Code, http.StatusOK)
	}

	patchBody := `{"progress_score":91,"next_steps":["Ship the UI"],"status":"published"}`
	patchResponse := api.request(t, http.MethodPatch, "/daily-progress/"+second.ID, strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated DailyProgress
	decodeResponse(t, patchResponse, &updated)
	if updated.ProgressScore != 91 || len(updated.NextSteps) != 1 || updated.NextSteps[0] != "Ship the UI" {
		t.Fatalf("unexpected updated entry: %+v", updated)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/daily-progress/"+draft.EntryDate, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}
}

func TestDailyProgressValidationAndConflict(t *testing.T) {
	api := newTestAPI(t)

	body := `{
		"entry_date": "2026/05/10",
		"title": " ",
		"progress_score": 101,
		"status": "live"
	}`
	response := api.request(t, http.MethodPost, "/daily-progress", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}

	var payload struct {
		Fields map[string]string `json:"fields"`
	}
	decodeResponse(t, response, &payload)
	for _, field := range []string{"entry_date", "title", "progress_score", "status"} {
		if payload.Fields[field] == "" {
			t.Fatalf("missing validation field %q in %+v", field, payload.Fields)
		}
	}

	createDailyProgress(t, api, `{
		"entry_date": "2026-05-10",
		"title": "Original",
		"progress_score": 50,
		"status": "published"
	}`)
	duplicateResponse := api.request(t, http.MethodPost, "/daily-progress", strings.NewReader(`{
		"entry_date": "2026-05-10",
		"title": "Duplicate",
		"progress_score": 60,
		"status": "published"
	}`), "application/json", true)
	if duplicateResponse.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want %d", duplicateResponse.Code, http.StatusConflict)
	}

	filterResponse := api.request(t, http.MethodGet, "/daily-progress?from=2026-05-11&to=2026-05-10", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("date range status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}
}

func createDailyProgress(t *testing.T, api testAPI, body string) DailyProgress {
	t.Helper()

	response := api.request(t, http.MethodPost, "/daily-progress", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var entry DailyProgress
	decodeResponse(t, response, &entry)
	return entry
}

func (api testAPI) request(t *testing.T, method string, path string, body io.Reader, contentType string, authorized bool) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, body)
	if authorized {
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
