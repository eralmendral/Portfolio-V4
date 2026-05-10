package tools

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

	toolHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /tools", http.HandlerFunc(toolHandler.HandleCollection))
	mux.Handle("POST /tools", requireJWT(http.HandlerFunc(toolHandler.HandleCollection)))
	mux.Handle("GET /tools/", http.HandlerFunc(toolHandler.HandleItem))
	mux.Handle("PATCH /tools/", requireJWT(http.HandlerFunc(toolHandler.HandleItem)))
	mux.Handle("PUT /tools/", requireJWT(http.HandlerFunc(toolHandler.HandleItem)))
	mux.Handle("DELETE /tools/", requireJWT(http.HandlerFunc(toolHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestToolsPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)

	getResponse := api.request(t, http.MethodGet, "/tools", nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("public get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	postResponse := api.request(t, http.MethodPost, "/tools", strings.NewReader(`{}`), "application/json", false)
	if postResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized post status = %d, want %d", postResponse.Code, http.StatusUnauthorized)
	}
}

func TestToolCRUDAndFilters(t *testing.T) {
	api := newTestAPI(t)

	codex := createTool(t, api, `{
		"name": "Codex",
		"category": "AI & Coding Assistants",
		"summary": "AI coding workflow for repo changes.",
		"icon_class": "lucide-bot",
		"tags": ["ai", "coding", "agent", "ai"],
		"sort_order": 20,
		"featured": true,
		"status": "published"
	}`)
	claude := createTool(t, api, `{
		"name": "Claude",
		"category": "AI & Coding Assistants",
		"summary": "AI assistant for reasoning and drafting.",
		"tags": ["ai", "assistant"],
		"sort_order": 10,
		"featured": false,
		"status": "published"
	}`)
	vscode := createTool(t, api, `{
		"name": "VS Code",
		"category": "IDEs & Editors",
		"summary": "Primary editor for TypeScript and Go projects.",
		"tags": ["editor", "typescript"],
		"sort_order": 1,
		"featured": true,
		"status": "published"
	}`)
	draft := createTool(t, api, `{
		"name": "Blender",
		"category": "Design & Creative",
		"tags": ["3d", "creative"]
	}`)

	if codex.Name != "Codex" || codex.IconClass != "lucide-bot" || !codex.Featured {
		t.Fatalf("unexpected codex tool: %+v", codex)
	}
	if len(codex.Tags) != 3 {
		t.Fatalf("tags = %+v, want normalized unique tags", codex.Tags)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("default status = %q, want %q", draft.Status, StatusDraft)
	}

	listResponse := api.request(t, http.MethodGet, "/tools", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Tools []Tool `json:"tools"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Tools) != 3 {
		t.Fatalf("tools count = %d, want 3", len(list.Tools))
	}
	wantOrder := []string{codex.ID, claude.ID, vscode.ID}
	for i, wantID := range wantOrder {
		if list.Tools[i].ID != wantID {
			t.Fatalf("tool[%d] = %q, want %q", i, list.Tools[i].ID, wantID)
		}
	}

	categoryResponse := api.request(t, http.MethodGet, "/tools?category=AI%20%26%20Coding%20Assistants&featured=true", nil, "", false)
	if categoryResponse.Code != http.StatusOK {
		t.Fatalf("category status = %d, want %d", categoryResponse.Code, http.StatusOK)
	}
	var categoryFiltered struct {
		Tools []Tool `json:"tools"`
	}
	decodeResponse(t, categoryResponse, &categoryFiltered)
	if len(categoryFiltered.Tools) != 1 || categoryFiltered.Tools[0].ID != codex.ID {
		t.Fatalf("unexpected category tools: %+v", categoryFiltered.Tools)
	}

	tagResponse := api.request(t, http.MethodGet, "/tools?tag=typescript", nil, "", false)
	if tagResponse.Code != http.StatusOK {
		t.Fatalf("tag status = %d, want %d", tagResponse.Code, http.StatusOK)
	}
	var tagFiltered struct {
		Tools []Tool `json:"tools"`
	}
	decodeResponse(t, tagResponse, &tagFiltered)
	if len(tagFiltered.Tools) != 1 || tagFiltered.Tools[0].ID != vscode.ID {
		t.Fatalf("unexpected tag tools: %+v", tagFiltered.Tools)
	}

	searchResponse := api.request(t, http.MethodGet, "/tools?q=reasoning", nil, "", false)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		Tools []Tool `json:"tools"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.Tools) != 1 || search.Tools[0].ID != claude.ID {
		t.Fatalf("unexpected search tools: %+v", search.Tools)
	}

	getResponse := api.request(t, http.MethodGet, "/tools/"+codex.ID, nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	patchBody := `{"summary":"Updated from tests.","featured":false,"sort_order":0,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/tools/"+codex.ID, strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated Tool
	decodeResponse(t, patchResponse, &updated)
	if updated.Summary != "Updated from tests." || updated.Featured || updated.SortOrder != 0 || updated.Status != StatusDraft {
		t.Fatalf("unexpected updated tool: %+v", updated)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/tools/"+draft.ID, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}
}

func TestToolValidationAndFilterValidation(t *testing.T) {
	api := newTestAPI(t)

	body := `{
		"name": " ",
		"category": " ",
		"sort_order": -1,
		"status": "live"
	}`
	response := api.request(t, http.MethodPost, "/tools", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}

	var payload struct {
		Fields map[string]string `json:"fields"`
	}
	decodeResponse(t, response, &payload)
	for _, field := range []string{"name", "category", "sort_order", "status"} {
		if payload.Fields[field] == "" {
			t.Fatalf("missing validation field %q in %+v", field, payload.Fields)
		}
	}

	filterResponse := api.request(t, http.MethodGet, "/tools?featured=maybe", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("featured filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}

	filterResponse = api.request(t, http.MethodGet, "/tools?status=live", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("status filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}
}

func createTool(t *testing.T, api testAPI, body string) Tool {
	t.Helper()

	response := api.request(t, http.MethodPost, "/tools", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var tool Tool
	decodeResponse(t, response, &tool)
	return tool
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
