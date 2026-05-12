package music

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

	musicHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /music", http.HandlerFunc(musicHandler.HandleCollection))
	mux.Handle("POST /music", requireJWT(http.HandlerFunc(musicHandler.HandleCollection)))
	mux.Handle("GET /music/", http.HandlerFunc(musicHandler.HandleItem))
	mux.Handle("PATCH /music/", requireJWT(http.HandlerFunc(musicHandler.HandleItem)))
	mux.Handle("PUT /music/", requireJWT(http.HandlerFunc(musicHandler.HandleItem)))
	mux.Handle("DELETE /music/", requireJWT(http.HandlerFunc(musicHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestMusicPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)

	getResponse := api.request(t, http.MethodGet, "/music", nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("public get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	postResponse := api.request(t, http.MethodPost, "/music", strings.NewReader(`{}`), "application/json", false)
	if postResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized post status = %d, want %d", postResponse.Code, http.StatusUnauthorized)
	}
}

func TestMusicCRUDAndFilters(t *testing.T) {
	api := newTestAPI(t)

	first := createMusic(t, api, `{
		"title": "Night Drive",
		"artist": "Aster",
		"album": "Road Notes",
		"spotify_url": "https://open.spotify.com/track/night-drive",
		"youtube_url": "https://www.youtube.com/watch?v=nightdrive",
		"mostly_listened_on": "2026-05-10",
		"notes": "For late work sessions and quiet resets.",
		"sort_order": 20,
		"status": "published"
	}`)
	second := createMusic(t, api, `{
		"title": "Morning Signal",
		"artist": "Beacon",
		"album": "Light Map",
		"mostly_listened_on": "2026-05-10",
		"notes": "A bright start track.",
		"sort_order": 10,
		"status": "published"
	}`)
	third := createMusic(t, api, `{
		"title": "Quiet Loop",
		"artist": "Harbor",
		"album": "Still Water",
		"mostly_listened_on": "2026-05-11",
		"notes": "Ambient focus track.",
		"sort_order": 10,
		"status": "published"
	}`)
	draft := createMusic(t, api, `{
		"title": "Private Draft",
		"artist": "Hidden",
		"mostly_listened_on": "2026-05-12"
	}`)

	if first.Title != "Night Drive" || first.Artist != "Aster" || first.SpotifyURL == "" || first.YouTubeURL == "" {
		t.Fatalf("unexpected first music entry: %+v", first)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("default status = %q, want %q", draft.Status, StatusDraft)
	}

	listResponse := api.request(t, http.MethodGet, "/music", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Music []Music `json:"music"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Music) != 3 {
		t.Fatalf("music count = %d, want 3", len(list.Music))
	}
	wantOrder := []string{third.ID, second.ID, first.ID}
	for i, wantID := range wantOrder {
		if list.Music[i].ID != wantID {
			t.Fatalf("music[%d] = %q, want %q", i, list.Music[i].ID, wantID)
		}
	}

	rangeResponse := api.request(t, http.MethodGet, "/music?from=2026-05-10&to=2026-05-10", nil, "", false)
	if rangeResponse.Code != http.StatusOK {
		t.Fatalf("range status = %d, want %d", rangeResponse.Code, http.StatusOK)
	}
	var rangeFiltered struct {
		Music []Music `json:"music"`
	}
	decodeResponse(t, rangeResponse, &rangeFiltered)
	if len(rangeFiltered.Music) != 2 {
		t.Fatalf("range music count = %d, want 2", len(rangeFiltered.Music))
	}

	searchResponse := api.request(t, http.MethodGet, "/music?q=ambient", nil, "", false)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		Music []Music `json:"music"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.Music) != 1 || search.Music[0].ID != third.ID {
		t.Fatalf("unexpected search music: %+v", search.Music)
	}

	getResponse := api.request(t, http.MethodGet, "/music/"+second.ID, nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	patchBody := `{"notes":"Updated from tests.","sort_order":0,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/music/"+second.ID, strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated Music
	decodeResponse(t, patchResponse, &updated)
	if updated.Notes != "Updated from tests." || updated.SortOrder != 0 || updated.Status != StatusDraft {
		t.Fatalf("unexpected updated music: %+v", updated)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/music/"+draft.ID, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}
}

func TestMusicValidationAndFilterValidation(t *testing.T) {
	api := newTestAPI(t)

	body := `{
		"title": " ",
		"artist": " ",
		"spotify_url": "ftp://example.com/song",
		"youtube_url": "not-a-url",
		"mostly_listened_on": "2026/05/10",
		"sort_order": -1,
		"status": "live"
	}`
	response := api.request(t, http.MethodPost, "/music", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}

	var payload struct {
		Fields map[string]string `json:"fields"`
	}
	decodeResponse(t, response, &payload)
	for _, field := range []string{"title", "artist", "spotify_url", "youtube_url", "mostly_listened_on", "sort_order", "status"} {
		if payload.Fields[field] == "" {
			t.Fatalf("missing validation field %q in %+v", field, payload.Fields)
		}
	}

	filterResponse := api.request(t, http.MethodGet, "/music?from=2026-05-11&to=2026-05-10", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("date range status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}

	filterResponse = api.request(t, http.MethodGet, "/music?status=live", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("status filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}
}

func createMusic(t *testing.T, api testAPI, body string) Music {
	t.Helper()

	response := api.request(t, http.MethodPost, "/music", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var entry Music
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
