package skills

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	skillHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /skill-categories", http.HandlerFunc(skillHandler.HandleCategoryCollection))
	mux.Handle("POST /skill-categories", requireJWT(http.HandlerFunc(skillHandler.HandleCategoryCollection)))
	mux.Handle("GET /skill-categories/", http.HandlerFunc(skillHandler.HandleCategoryItem))
	mux.Handle("PATCH /skill-categories/", requireJWT(http.HandlerFunc(skillHandler.HandleCategoryItem)))
	mux.Handle("PUT /skill-categories/", requireJWT(http.HandlerFunc(skillHandler.HandleCategoryItem)))
	mux.Handle("DELETE /skill-categories/", requireJWT(http.HandlerFunc(skillHandler.HandleCategoryItem)))
	mux.Handle("GET /skills", http.HandlerFunc(skillHandler.HandleSkillCollection))
	mux.Handle("POST /skills", requireJWT(http.HandlerFunc(skillHandler.HandleSkillCollection)))
	mux.Handle("GET /skills/", http.HandlerFunc(skillHandler.HandleSkillItem))
	mux.Handle("PATCH /skills/", requireJWT(http.HandlerFunc(skillHandler.HandleSkillItem)))
	mux.Handle("PUT /skills/", requireJWT(http.HandlerFunc(skillHandler.HandleSkillItem)))
	mux.Handle("DELETE /skills/", requireJWT(http.HandlerFunc(skillHandler.HandleSkillItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestSkillsPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)

	for _, path := range []string{"/skill-categories", "/skills"} {
		response := api.request(t, http.MethodGet, path, nil, "", false)
		if response.Code != http.StatusOK {
			t.Fatalf("public get %s status = %d, want %d", path, response.Code, http.StatusOK)
		}
	}

	response := api.request(t, http.MethodPost, "/skill-categories", strings.NewReader(`{}`), "application/json", false)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized category write status = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	response = api.request(t, http.MethodPost, "/skills", strings.NewReader(`{}`), "application/json", false)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized skill write status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSkillCategoryCRUDAndFilters(t *testing.T) {
	api := newTestAPI(t)

	backend := createCategory(t, api, `{
		"name": "Backend Engineering",
		"description": "API design, services, and data modeling.",
		"icon_class": "lucide-server",
		"sort_order": 20,
		"status": "published"
	}`)
	frontend := createCategory(t, api, `{
		"name": "Frontend Engineering",
		"description": "React and responsive UI.",
		"sort_order": 10,
		"status": "published"
	}`)
	draft := createCategory(t, api, `{
		"slug": "cloud-devops",
		"name": "Cloud & DevOps",
		"status": "draft"
	}`)

	if backend.Slug != "backend-engineering" {
		t.Fatalf("slug = %q, want backend-engineering", backend.Slug)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("status = %q, want %q", draft.Status, StatusDraft)
	}

	getResponse := api.request(t, http.MethodGet, "/skill-categories/"+backend.Slug, nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get by slug status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	listResponse := api.request(t, http.MethodGet, "/skill-categories", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Categories []SkillCategory `json:"skill_categories"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Categories) != 2 {
		t.Fatalf("category count = %d, want 2", len(list.Categories))
	}
	if list.Categories[0].ID != frontend.ID || list.Categories[1].ID != backend.ID {
		t.Fatalf("unexpected category order: %+v", list.Categories)
	}

	searchResponse := api.request(t, http.MethodGet, "/skill-categories?q=data", nil, "", false)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		Categories []SkillCategory `json:"skill_categories"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.Categories) != 1 || search.Categories[0].ID != backend.ID {
		t.Fatalf("unexpected search categories: %+v", search.Categories)
	}

	patchResponse := api.request(t, http.MethodPatch, "/skill-categories/"+backend.ID, strings.NewReader(`{"description":"Updated","status":"draft"}`), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var patched SkillCategory
	decodeResponse(t, patchResponse, &patched)
	if patched.Description != "Updated" || patched.Status != StatusDraft {
		t.Fatalf("unexpected patched category: %+v", patched)
	}

	conflictResponse := api.request(t, http.MethodPost, "/skill-categories", strings.NewReader(`{
		"slug": "frontend-engineering",
		"name": "Duplicate Frontend"
	}`), "application/json", true)
	if conflictResponse.Code != http.StatusConflict {
		t.Fatalf("conflict status = %d, want %d", conflictResponse.Code, http.StatusConflict)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/skill-categories/"+draft.ID, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}
}

func TestSkillCRUDRelationshipFiltersAndOrdering(t *testing.T) {
	api := newTestAPI(t)

	backend := createCategory(t, api, `{
		"name": "Backend Engineering",
		"description": "API design, services, and data modeling.",
		"sort_order": 10,
		"status": "published"
	}`)
	frontend := createCategory(t, api, `{
		"name": "Frontend Engineering",
		"description": "React and TypeScript interfaces.",
		"sort_order": 20,
		"status": "published"
	}`)

	postgres := createSkill(t, api, skillBody(backend.ID, "PostgreSQL", "Relational data modeling.", 5, false, "published"))
	apiDesign := createSkill(t, api, skillBody(backend.ID, "API Design", "REST API design and validation.", 10, true, "published"))
	goSkill := createSkill(t, api, skillBody(backend.ID, "Go", "Production HTTP services.", 20, true, "published"))
	react := createSkill(t, api, skillBody(frontend.ID, "React", "Component-driven UI.", 1, true, "published"))
	draft := createSkill(t, api, skillBody(backend.ID, "Observability", "Logs and runtime signals.", 30, false, "draft"))

	listResponse := api.request(t, http.MethodGet, "/skills?category=backend-engineering", nil, "", false)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var list struct {
		Skills []Skill `json:"skills"`
	}
	decodeResponse(t, listResponse, &list)
	if len(list.Skills) != 3 {
		t.Fatalf("skills count = %d, want 3", len(list.Skills))
	}
	wantOrder := []string{apiDesign.ID, goSkill.ID, postgres.ID}
	for i, wantID := range wantOrder {
		if list.Skills[i].ID != wantID {
			t.Fatalf("skill[%d] = %q, want %q", i, list.Skills[i].ID, wantID)
		}
	}

	featuredResponse := api.request(t, http.MethodGet, "/skills?featured=true&category_id="+backend.ID, nil, "", false)
	if featuredResponse.Code != http.StatusOK {
		t.Fatalf("featured status = %d, want %d", featuredResponse.Code, http.StatusOK)
	}
	var featured struct {
		Skills []Skill `json:"skills"`
	}
	decodeResponse(t, featuredResponse, &featured)
	if len(featured.Skills) != 2 {
		t.Fatalf("featured skills count = %d, want 2", len(featured.Skills))
	}

	searchResponse := api.request(t, http.MethodGet, "/skills?q=interfaces", nil, "", false)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		Skills []Skill `json:"skills"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.Skills) != 1 || search.Skills[0].ID != react.ID {
		t.Fatalf("unexpected search skills: %+v", search.Skills)
	}

	patchResponse := api.request(t, http.MethodPatch, "/skills/"+apiDesign.ID, strings.NewReader(`{"featured":false,"status":"draft","sort_order":1}`), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var patched Skill
	decodeResponse(t, patchResponse, &patched)
	if patched.Featured || patched.Status != StatusDraft || patched.SortOrder != 1 {
		t.Fatalf("unexpected patched skill: %+v", patched)
	}

	deleteInUseResponse := api.request(t, http.MethodDelete, "/skill-categories/"+backend.ID, nil, "", true)
	if deleteInUseResponse.Code != http.StatusConflict {
		t.Fatalf("delete in-use category status = %d, want %d", deleteInUseResponse.Code, http.StatusConflict)
	}

	deleteSkillResponse := api.request(t, http.MethodDelete, "/skills/"+draft.ID, nil, "", true)
	if deleteSkillResponse.Code != http.StatusNoContent {
		t.Fatalf("delete skill status = %d, want %d", deleteSkillResponse.Code, http.StatusNoContent)
	}
}

func TestSkillValidationAndInvalidCategory(t *testing.T) {
	api := newTestAPI(t)

	categoryResponse := api.request(t, http.MethodPost, "/skill-categories", strings.NewReader(`{
		"slug": "Invalid Slug",
		"name": " ",
		"sort_order": -1,
		"status": "live"
	}`), "application/json", true)
	if categoryResponse.Code != http.StatusBadRequest {
		t.Fatalf("category status = %d, want %d", categoryResponse.Code, http.StatusBadRequest)
	}
	var categoryPayload struct {
		Fields map[string]string `json:"fields"`
	}
	decodeResponse(t, categoryResponse, &categoryPayload)
	for _, field := range []string{"slug", "name", "sort_order", "status"} {
		if categoryPayload.Fields[field] == "" {
			t.Fatalf("missing category validation field %q in %+v", field, categoryPayload.Fields)
		}
	}

	skillResponse := api.request(t, http.MethodPost, "/skills", strings.NewReader(`{
		"category_id": "missing-category",
		"name": "Go",
		"status": "published"
	}`), "application/json", true)
	if skillResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid category status = %d, want %d", skillResponse.Code, http.StatusBadRequest)
	}

	filterResponse := api.request(t, http.MethodGet, "/skills?featured=maybe", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("featured filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}

	filterResponse = api.request(t, http.MethodGet, "/skill-categories?status=live", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("status filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}
}

func createCategory(t *testing.T, api testAPI, body string) SkillCategory {
	t.Helper()

	response := api.request(t, http.MethodPost, "/skill-categories", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create category status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var category SkillCategory
	decodeResponse(t, response, &category)
	return category
}

func createSkill(t *testing.T, api testAPI, body string) Skill {
	t.Helper()

	response := api.request(t, http.MethodPost, "/skills", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create skill status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var skill Skill
	decodeResponse(t, response, &skill)
	return skill
}

func skillBody(categoryID string, name string, summary string, sortOrder int, featured bool, status string) string {
	return `{
		"category_id": "` + categoryID + `",
		"name": "` + name + `",
		"summary": "` + summary + `",
		"sort_order": ` + strconvItoa(sortOrder) + `,
		"featured": ` + boolString(featured) + `,
		"status": "` + status + `"
	}`
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

func strconvItoa(value int) string {
	return strconv.FormatInt(int64(value), 10)
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
