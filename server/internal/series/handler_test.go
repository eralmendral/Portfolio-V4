package series

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

	seriesHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /series", http.HandlerFunc(seriesHandler.HandleCollection))
	mux.Handle("POST /series", requireJWT(http.HandlerFunc(seriesHandler.HandleCollection)))
	mux.Handle("GET /series/", http.HandlerFunc(seriesHandler.HandleItem))
	mux.Handle("PATCH /series/", requireJWT(http.HandlerFunc(seriesHandler.HandleItem)))
	mux.Handle("PUT /series/", requireJWT(http.HandlerFunc(seriesHandler.HandleItem)))
	mux.Handle("DELETE /series/", requireJWT(http.HandlerFunc(seriesHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestSeriesPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)

	getResponse := api.request(t, http.MethodGet, "/series", nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("public get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	postResponse := api.request(t, http.MethodPost, "/series", strings.NewReader(`{}`), "application/json", false)
	if postResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized post status = %d, want %d", postResponse.Code, http.StatusUnauthorized)
	}
}

func TestSeriesCRUDAndFilters(t *testing.T) {
	api := newTestAPI(t)

	first := createSeries(t, api, `{
		"title": "The Long Room",
		"category": "tv_series",
		"creator": "North Studio",
		"platform": "StreamBox",
		"watch_url": "https://example.com/the-long-room",
		"mostly_watched_on": "2026-05-10",
		"notes": "A quiet favorite about work, friendship, and ordinary courage.",
		"sort_order": 20,
		"status": "published"
	}`)
	second := createSeries(t, api, `{
		"title": "Home Signals",
		"category": "TV Series",
		"creator": "City Room",
		"platform": "StreamBox",
		"mostly_watched_on": "2026-05-10",
		"notes": "A comfort watch about chosen family.",
		"sort_order": 10,
		"status": "published"
	}`)
	third := createSeries(t, api, `{
		"title": "Moon Harbor",
		"category": "anime",
		"creator": "Blue House",
		"platform": "Crunchyroll",
		"mostly_watched_on": "2026-05-11",
		"notes": "Soft science fiction with a patient emotional center.",
		"sort_order": 10,
		"status": "published"
	}`)
	draft := createSeries(t, api, `{
		"title": "Private Draft",
		"category": "anime",
		"mostly_watched_on": "2026-05-12"
	}`)

	if first.Title != "The Long Room" || first.Category != CategoryTVSeries || first.WatchURL == "" {
		t.Fatalf("unexpected first series entry: %+v", first)
	}
	if second.Category != CategoryTVSeries {
		t.Fatalf("normalized category = %q, want %q", second.Category, CategoryTVSeries)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("default status = %q, want %q", draft.Status, StatusDraft)
	}

	listResponse := api.request(t, http.MethodGet, "/series", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Series []Series `json:"series"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Series) != 3 {
		t.Fatalf("series count = %d, want 3", len(list.Series))
	}
	wantOrder := []string{third.ID, second.ID, first.ID}
	for i, wantID := range wantOrder {
		if list.Series[i].ID != wantID {
			t.Fatalf("series[%d] = %q, want %q", i, list.Series[i].ID, wantID)
		}
	}

	categoryResponse := api.request(t, http.MethodGet, "/series?category=tv-series", nil, "", false)
	if categoryResponse.Code != http.StatusOK {
		t.Fatalf("category status = %d, want %d", categoryResponse.Code, http.StatusOK)
	}
	var categoryFiltered struct {
		Series []Series `json:"series"`
	}
	decodeResponse(t, categoryResponse, &categoryFiltered)
	if len(categoryFiltered.Series) != 2 {
		t.Fatalf("category series count = %d, want 2", len(categoryFiltered.Series))
	}

	rangeResponse := api.request(t, http.MethodGet, "/series?from=2026-05-10&to=2026-05-10", nil, "", false)
	if rangeResponse.Code != http.StatusOK {
		t.Fatalf("range status = %d, want %d", rangeResponse.Code, http.StatusOK)
	}
	var rangeFiltered struct {
		Series []Series `json:"series"`
	}
	decodeResponse(t, rangeResponse, &rangeFiltered)
	if len(rangeFiltered.Series) != 2 {
		t.Fatalf("range series count = %d, want 2", len(rangeFiltered.Series))
	}

	searchResponse := api.request(t, http.MethodGet, "/series?q=emotional", nil, "", false)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		Series []Series `json:"series"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.Series) != 1 || search.Series[0].ID != third.ID {
		t.Fatalf("unexpected search series: %+v", search.Series)
	}

	getResponse := api.request(t, http.MethodGet, "/series/"+second.ID, nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	patchBody := `{"notes":"Updated from tests.","sort_order":0,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/series/"+second.ID, strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated Series
	decodeResponse(t, patchResponse, &updated)
	if updated.Notes != "Updated from tests." || updated.SortOrder != 0 || updated.Status != StatusDraft {
		t.Fatalf("unexpected updated series: %+v", updated)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/series/"+draft.ID, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}
}

func TestSeriesValidationAndFilterValidation(t *testing.T) {
	api := newTestAPI(t)

	body := `{
		"title": " ",
		"category": "movie",
		"watch_url": "ftp://example.com/series",
		"mostly_watched_on": "2026/05/10",
		"sort_order": -1,
		"status": "live"
	}`
	response := api.request(t, http.MethodPost, "/series", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}

	var payload struct {
		Fields map[string]string `json:"fields"`
	}
	decodeResponse(t, response, &payload)
	for _, field := range []string{"title", "category", "watch_url", "mostly_watched_on", "sort_order", "status"} {
		if payload.Fields[field] == "" {
			t.Fatalf("missing validation field %q in %+v", field, payload.Fields)
		}
	}

	filterResponse := api.request(t, http.MethodGet, "/series?from=2026-05-11&to=2026-05-10", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("date range status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}

	filterResponse = api.request(t, http.MethodGet, "/series?category=movie", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("category filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}
}

func createSeries(t *testing.T, api testAPI, body string) Series {
	t.Helper()

	response := api.request(t, http.MethodPost, "/series", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var entry Series
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
