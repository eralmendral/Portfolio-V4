package games

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

	gameHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /games", http.HandlerFunc(gameHandler.HandleCollection))
	mux.Handle("POST /games", requireJWT(http.HandlerFunc(gameHandler.HandleCollection)))
	mux.Handle("GET /games/", http.HandlerFunc(gameHandler.HandleItem))
	mux.Handle("PATCH /games/", requireJWT(http.HandlerFunc(gameHandler.HandleItem)))
	mux.Handle("PUT /games/", requireJWT(http.HandlerFunc(gameHandler.HandleItem)))
	mux.Handle("DELETE /games/", requireJWT(http.HandlerFunc(gameHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestGamesPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)

	getResponse := api.request(t, http.MethodGet, "/games", nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("public get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	postResponse := api.request(t, http.MethodPost, "/games", strings.NewReader(`{}`), "application/json", false)
	if postResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized post status = %d, want %d", postResponse.Code, http.StatusUnauthorized)
	}
}

func TestGamesCRUDAndFilters(t *testing.T) {
	api := newTestAPI(t)

	first := createGame(t, api, `{
		"title": "Starlit Roads",
		"studio": "North Play",
		"platform": "PC",
		"genre": "Adventure",
		"store_url": "https://example.com/games/starlit-roads",
		"mostly_played_on": "2026-05-10",
		"notes": "A wandering game for decompressing after long days.",
		"sort_order": 20,
		"status": "published"
	}`)
	second := createGame(t, api, `{
		"title": "Kindling",
		"studio": "Small Fire",
		"platform": "PC",
		"genre": "Cozy Simulation",
		"mostly_played_on": "2026-05-10",
		"notes": "A cozy reset game with patient rituals.",
		"sort_order": 10,
		"status": "published"
	}`)
	third := createGame(t, api, `{
		"title": "Garden Tactics",
		"studio": "Green Tile",
		"platform": "Switch",
		"genre": "Strategy",
		"mostly_played_on": "2026-05-11",
		"notes": "Small decisions that feel satisfying to revisit.",
		"sort_order": 10,
		"status": "published"
	}`)
	draft := createGame(t, api, `{
		"title": "Private Draft",
		"platform": "PC",
		"mostly_played_on": "2026-05-12"
	}`)

	if first.Title != "Starlit Roads" || first.Platform != "PC" || first.StoreURL == "" {
		t.Fatalf("unexpected first game entry: %+v", first)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("default status = %q, want %q", draft.Status, StatusDraft)
	}

	listResponse := api.request(t, http.MethodGet, "/games", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Games []Game `json:"games"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Games) != 3 {
		t.Fatalf("games count = %d, want 3", len(list.Games))
	}
	wantOrder := []string{third.ID, second.ID, first.ID}
	for i, wantID := range wantOrder {
		if list.Games[i].ID != wantID {
			t.Fatalf("games[%d] = %q, want %q", i, list.Games[i].ID, wantID)
		}
	}

	platformResponse := api.request(t, http.MethodGet, "/games?platform=pc", nil, "", false)
	if platformResponse.Code != http.StatusOK {
		t.Fatalf("platform status = %d, want %d", platformResponse.Code, http.StatusOK)
	}
	var platformFiltered struct {
		Games []Game `json:"games"`
	}
	decodeResponse(t, platformResponse, &platformFiltered)
	if len(platformFiltered.Games) != 2 {
		t.Fatalf("platform games count = %d, want 2", len(platformFiltered.Games))
	}

	rangeResponse := api.request(t, http.MethodGet, "/games?from=2026-05-10&to=2026-05-10", nil, "", false)
	if rangeResponse.Code != http.StatusOK {
		t.Fatalf("range status = %d, want %d", rangeResponse.Code, http.StatusOK)
	}
	var rangeFiltered struct {
		Games []Game `json:"games"`
	}
	decodeResponse(t, rangeResponse, &rangeFiltered)
	if len(rangeFiltered.Games) != 2 {
		t.Fatalf("range games count = %d, want 2", len(rangeFiltered.Games))
	}

	searchResponse := api.request(t, http.MethodGet, "/games?q=rituals", nil, "", false)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		Games []Game `json:"games"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.Games) != 1 || search.Games[0].ID != second.ID {
		t.Fatalf("unexpected search games: %+v", search.Games)
	}

	getResponse := api.request(t, http.MethodGet, "/games/"+second.ID, nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	patchBody := `{"notes":"Updated from tests.","sort_order":0,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/games/"+second.ID, strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated Game
	decodeResponse(t, patchResponse, &updated)
	if updated.Notes != "Updated from tests." || updated.SortOrder != 0 || updated.Status != StatusDraft {
		t.Fatalf("unexpected updated game: %+v", updated)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/games/"+draft.ID, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}
}

func TestGamesValidationAndFilterValidation(t *testing.T) {
	api := newTestAPI(t)

	body := `{
		"title": " ",
		"platform": " ",
		"store_url": "ftp://example.com/game",
		"mostly_played_on": "2026/05/10",
		"sort_order": -1,
		"status": "live"
	}`
	response := api.request(t, http.MethodPost, "/games", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}

	var payload struct {
		Fields map[string]string `json:"fields"`
	}
	decodeResponse(t, response, &payload)
	for _, field := range []string{"title", "platform", "store_url", "mostly_played_on", "sort_order", "status"} {
		if payload.Fields[field] == "" {
			t.Fatalf("missing validation field %q in %+v", field, payload.Fields)
		}
	}

	filterResponse := api.request(t, http.MethodGet, "/games?from=2026-05-11&to=2026-05-10", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("date range status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}

	filterResponse = api.request(t, http.MethodGet, "/games?status=live", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("status filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}
}

func createGame(t *testing.T, api testAPI, body string) Game {
	t.Helper()

	response := api.request(t, http.MethodPost, "/games", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var entry Game
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
