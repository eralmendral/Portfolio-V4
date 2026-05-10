package articles

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

	articleHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /articles", requireJWT(http.HandlerFunc(articleHandler.HandleCollection)))
	mux.Handle("POST /articles", requireJWT(http.HandlerFunc(articleHandler.HandleCollection)))
	mux.Handle("/articles/", requireJWT(http.HandlerFunc(articleHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestArticlesRequireJWT(t *testing.T) {
	api := newTestAPI(t)

	request := httptest.NewRequest(http.MethodGet, "/articles", nil)
	response := httptest.NewRecorder()

	api.handler.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestArticleCRUD(t *testing.T) {
	api := newTestAPI(t)

	article := createArticle(t, api)
	if article.Title != "Publishing Go APIs on Dev.to" {
		t.Fatalf("title = %q, want Publishing Go APIs on Dev.to", article.Title)
	}
	if article.URL != "https://dev.to/example/publishing-go-apis" {
		t.Fatalf("url = %q, want dev.to URL", article.URL)
	}
	if article.PublishedAt == nil {
		t.Fatal("published_at should be set")
	}
	if article.CreatedAt.IsZero() {
		t.Fatal("created_at should be set")
	}

	getResponse := api.request(t, http.MethodGet, "/articles/"+article.ID, nil, "")
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	listResponse := api.request(t, http.MethodGet, "/articles?status=published&q=dev.to", nil, "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Articles []Article `json:"articles"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Articles) != 1 {
		t.Fatalf("articles count = %d, want 1", len(list.Articles))
	}

	patchBody := `{"summary":"Updated summary","featured":false,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/articles/"+article.ID, strings.NewReader(patchBody), "application/json")
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var patched Article
	decodeResponse(t, patchResponse, &patched)
	if patched.Summary != "Updated summary" || patched.Featured {
		t.Fatalf("unexpected patched article: %+v", patched)
	}
	if patched.Status != StatusDraft {
		t.Fatalf("status = %q, want %q", patched.Status, StatusDraft)
	}

	putBody := `{
		"title": "Publishing Go APIs on Medium",
		"url": "https://medium.com/example/publishing-go-apis",
		"source": "Medium",
		"summary": "Replaced summary",
		"sort_order": 5,
		"status": "published"
	}`
	putResponse := api.request(t, http.MethodPut, "/articles/"+article.ID, strings.NewReader(putBody), "application/json")
	if putResponse.Code != http.StatusOK {
		t.Fatalf("put status = %d, want %d: %s", putResponse.Code, http.StatusOK, putResponse.Body.String())
	}
	var replaced Article
	decodeResponse(t, putResponse, &replaced)
	if replaced.Source != "Medium" || replaced.SortOrder != 5 {
		t.Fatalf("unexpected replaced article: %+v", replaced)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/articles/"+article.ID, nil, "")
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}

	missingResponse := api.request(t, http.MethodGet, "/articles/"+article.ID, nil, "")
	if missingResponse.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want %d", missingResponse.Code, http.StatusNotFound)
	}
}

func TestArticleListFilteringSearchAndOrdering(t *testing.T) {
	api := newTestAPI(t)

	oldMedium := createArticleFromBody(t, api, `{
		"title": "Clean Portfolio Data Models",
		"url": "https://medium.com/example/clean-portfolio-data-models",
		"source": "Medium",
		"summary": "Designing metadata for portfolio previews.",
		"featured": true,
		"sort_order": 10,
		"status": "published",
		"published_at": "2026-04-01T09:00:00Z"
	}`)
	newMedium := createArticleFromBody(t, api, `{
		"title": "Portfolio Admin Workflows",
		"url": "https://medium.com/example/portfolio-admin-workflows",
		"source": "Medium",
		"summary": "Admin workflows for article cards.",
		"featured": false,
		"sort_order": 10,
		"status": "published",
		"published_at": "2026-05-01T09:00:00Z"
	}`)
	devArticle := createArticleFromBody(t, api, `{
		"title": "Go API Routing Notes",
		"url": "https://dev.to/example/go-api-routing-notes",
		"source": "Dev.to",
		"summary": "Routing details for Go APIs.",
		"featured": true,
		"sort_order": 20,
		"status": "published",
		"published_at": "2026-05-08T09:00:00Z"
	}`)

	listResponse := api.request(t, http.MethodGet, "/articles?status=published", nil, "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Articles []Article `json:"articles"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Articles) != 3 {
		t.Fatalf("articles count = %d, want 3", len(list.Articles))
	}
	wantOrder := []string{newMedium.ID, oldMedium.ID, devArticle.ID}
	for i, wantID := range wantOrder {
		if list.Articles[i].ID != wantID {
			t.Fatalf("article[%d] = %q, want %q", i, list.Articles[i].ID, wantID)
		}
	}

	filteredResponse := api.request(t, http.MethodGet, "/articles?featured=true&q=medium", nil, "")
	if filteredResponse.Code != http.StatusOK {
		t.Fatalf("filtered status = %d, want %d", filteredResponse.Code, http.StatusOK)
	}
	var filtered struct {
		Articles []Article `json:"articles"`
	}
	decodeResponse(t, filteredResponse, &filtered)
	if len(filtered.Articles) != 1 || filtered.Articles[0].ID != oldMedium.ID {
		t.Fatalf("filtered articles = %+v, want only %q", filtered.Articles, oldMedium.ID)
	}
}

func createArticle(t *testing.T, api testAPI) Article {
	t.Helper()

	return createArticleFromBody(t, api, `{
		"title": "Publishing Go APIs on Dev.to",
		"url": "https://dev.to/example/publishing-go-apis",
		"source": "Dev.to",
		"summary": "How to publish external article previews.",
		"cover_image_url": "https://images.example.com/go-apis.png",
		"featured": true,
		"sort_order": 10,
		"status": "published",
		"published_at": "2026-04-20T10:30:00Z"
	}`)
}

func createArticleFromBody(t *testing.T, api testAPI, body string) Article {
	t.Helper()

	response := api.request(t, http.MethodPost, "/articles", strings.NewReader(body), "application/json")
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var article Article
	decodeResponse(t, response, &article)
	return article
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
