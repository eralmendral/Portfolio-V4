package links

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

	linkHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /links", http.HandlerFunc(linkHandler.HandleCollection))
	mux.Handle("POST /links", requireJWT(http.HandlerFunc(linkHandler.HandleCollection)))
	mux.Handle("GET /links/", http.HandlerFunc(linkHandler.HandleItem))
	mux.Handle("PATCH /links/", requireJWT(http.HandlerFunc(linkHandler.HandleItem)))
	mux.Handle("PUT /links/", requireJWT(http.HandlerFunc(linkHandler.HandleItem)))
	mux.Handle("DELETE /links/", requireJWT(http.HandlerFunc(linkHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestLinksPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)

	getResponse := api.request(t, http.MethodGet, "/links", nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("public get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	postResponse := api.request(t, http.MethodPost, "/links", strings.NewReader(`{}`), "application/json", false)
	if postResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized post status = %d, want %d", postResponse.Code, http.StatusUnauthorized)
	}
}

func TestLinkCRUDAndFilters(t *testing.T) {
	api := newTestAPI(t)

	github := createLink(t, api, `{
		"label": "GitHub",
		"url": "https://github.com/example",
		"icon_class": "fa-brands fa-github",
		"sort_order": 2,
		"star": true,
		"status": "published"
	}`)
	youtube := createLink(t, api, `{
		"label": "YouTube",
		"url": "https://youtube.com/@example",
		"icon_class": "fa-brands fa-youtube",
		"sort_order": 1,
		"status": "published"
	}`)
	draft := createLink(t, api, `{
		"label": "HackerRank",
		"url": "https://www.hackerrank.com/example",
		"icon_class": "fa-brands fa-hackerrank"
	}`)

	if github.Label != "GitHub" || github.IconClass != "fa-brands fa-github" || !github.Star {
		t.Fatalf("unexpected github link: %+v", github)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("default status = %q, want %q", draft.Status, StatusDraft)
	}
	if github.CreatedAt.IsZero() {
		t.Fatal("created_at should be set")
	}

	listResponse := api.request(t, http.MethodGet, "/links", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Links []Link `json:"links"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Links) != 2 {
		t.Fatalf("links count = %d, want 2", len(list.Links))
	}
	if list.Links[0].ID != youtube.ID || list.Links[1].ID != github.ID {
		t.Fatalf("unexpected list order: %+v", list.Links)
	}

	starResponse := api.request(t, http.MethodGet, "/links?star=true", nil, "", false)
	if starResponse.Code != http.StatusOK {
		t.Fatalf("star list status = %d, want %d", starResponse.Code, http.StatusOK)
	}
	var starred struct {
		Links []Link `json:"links"`
	}
	decodeResponse(t, starResponse, &starred)
	if len(starred.Links) != 1 || starred.Links[0].ID != github.ID {
		t.Fatalf("unexpected starred links: %+v", starred.Links)
	}

	searchResponse := api.request(t, http.MethodGet, "/links?q=git", nil, "", false)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		Links []Link `json:"links"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.Links) != 1 || search.Links[0].ID != github.ID {
		t.Fatalf("unexpected search links: %+v", search.Links)
	}

	getResponse := api.request(t, http.MethodGet, "/links/"+github.ID, nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	patchBody := `{"icon_class":"fa-brands fa-square-github","star":false,"sort_order":0,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/links/"+github.ID, strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated Link
	decodeResponse(t, patchResponse, &updated)
	if updated.IconClass != "fa-brands fa-square-github" || updated.Star || updated.SortOrder != 0 || updated.Status != StatusDraft {
		t.Fatalf("unexpected updated link: %+v", updated)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/links/"+github.ID, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}

	missingResponse := api.request(t, http.MethodGet, "/links/"+github.ID, nil, "", false)
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missingResponse.Code, http.StatusNotFound)
	}
}

func TestLinkValidation(t *testing.T) {
	api := newTestAPI(t)

	body := `{
		"label": " ",
		"url": "ftp://example.com/profile",
		"sort_order": -1,
		"status": "live"
	}`
	response := api.request(t, http.MethodPost, "/links", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}

	var payload struct {
		Error  string            `json:"error"`
		Fields map[string]string `json:"fields"`
	}
	decodeResponse(t, response, &payload)
	for _, field := range []string{"label", "url", "sort_order", "status"} {
		if payload.Fields[field] == "" {
			t.Fatalf("missing validation field %q in %+v", field, payload.Fields)
		}
	}
}

func TestLinkFilterValidation(t *testing.T) {
	api := newTestAPI(t)

	response := api.request(t, http.MethodGet, "/links?star=maybe", nil, "", false)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("star status = %d, want %d", response.Code, http.StatusBadRequest)
	}

	response = api.request(t, http.MethodGet, "/links?status=live", nil, "", false)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status filter status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func createLink(t *testing.T, api testAPI, body string) Link {
	t.Helper()

	response := api.request(t, http.MethodPost, "/links", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var link Link
	decodeResponse(t, response, &link)
	return link
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
