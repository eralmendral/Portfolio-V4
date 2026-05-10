package products

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

	productHandler := NewHandler(newMemoryStore())

	tokens := auth.NewTokenService("secret", "tests", time.Hour)
	token, err := tokens.Sign("admin", []string{"admin"})
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	requireJWT := auth.RequireJWT(tokens)
	optionalJWT := auth.OptionalJWT(tokens)
	mux := http.NewServeMux()
	mux.Handle("GET /products/section", http.HandlerFunc(productHandler.HandleSection))
	mux.Handle("PATCH /products/section", requireJWT(http.HandlerFunc(productHandler.HandleSection)))
	mux.Handle("GET /products", optionalJWT(http.HandlerFunc(productHandler.HandleCollection)))
	mux.Handle("POST /products", requireJWT(http.HandlerFunc(productHandler.HandleCollection)))
	mux.Handle("GET /products/", optionalJWT(http.HandlerFunc(productHandler.HandleItem)))
	mux.Handle("PATCH /products/", requireJWT(http.HandlerFunc(productHandler.HandleItem)))
	mux.Handle("PUT /products/", requireJWT(http.HandlerFunc(productHandler.HandleItem)))
	mux.Handle("DELETE /products/", requireJWT(http.HandlerFunc(productHandler.HandleItem)))

	return testAPI{
		handler: mux,
		token:   token,
	}
}

func TestProductsPublicReadAndAdminWriteProtection(t *testing.T) {
	api := newTestAPI(t)

	getResponse := api.request(t, http.MethodGet, "/products", nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("public get status = %d, want %d", getResponse.Code, http.StatusOK)
	}

	postResponse := api.request(t, http.MethodPost, "/products", strings.NewReader(`{}`), "application/json", false)
	if postResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized post status = %d, want %d", postResponse.Code, http.StatusUnauthorized)
	}

	sectionPatch := api.request(t, http.MethodPatch, "/products/section", strings.NewReader(`{"enabled":true}`), "application/json", false)
	if sectionPatch.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized section patch = %d, want %d", sectionPatch.Code, http.StatusUnauthorized)
	}
}

func TestProductCRUDPublicFilteringAndAdminAccess(t *testing.T) {
	api := newTestAPI(t)

	anki := createProduct(t, api, `{
		"slug": "valuable-anki-deck",
		"title": "Valuable Anki Deck",
		"summary": "A practical spaced repetition deck.",
		"description": "Curated cards for durable study.",
		"cover_image_url": "https://example.com/anki.png",
		"price_label": "$19",
		"cta_label": "Get the deck",
		"cta_url": "https://example.com/checkout",
		"category": "Learning",
		"tags": ["anki", "study", "anki"],
		"sort_order": 10,
		"featured": true,
		"status": "published"
	}`)
	draft := createProduct(t, api, `{
		"title": "Draft Deck",
		"cta_url": "https://example.com/draft",
		"category": "Learning",
		"tags": ["anki"]
	}`)
	archived := createProduct(t, api, `{
		"slug": "old-deck",
		"title": "Old Deck",
		"cta_url": "https://example.com/old",
		"category": "Archive",
		"status": "archived"
	}`)

	if anki.Slug != "valuable-anki-deck" || anki.CTALabel != "Get the deck" || !anki.Featured {
		t.Fatalf("unexpected product: %+v", anki)
	}
	if len(anki.Tags) != 2 {
		t.Fatalf("tags = %+v, want normalized unique tags", anki.Tags)
	}
	if draft.Slug != "draft-deck" {
		t.Fatalf("default slug = %q, want draft-deck", draft.Slug)
	}
	if draft.Status != StatusDraft {
		t.Fatalf("default status = %q, want %q", draft.Status, StatusDraft)
	}

	publicList := api.request(t, http.MethodGet, "/products", nil, "", false)
	if publicList.Code != http.StatusOK {
		t.Fatalf("public list status = %d, want %d", publicList.Code, http.StatusOK)
	}
	var list struct {
		Products []Product `json:"products"`
	}
	decodeResponse(t, publicList, &list)
	if len(list.Products) != 1 || list.Products[0].ID != anki.ID {
		t.Fatalf("public products = %+v, want only published product", list.Products)
	}

	publicDraft := api.request(t, http.MethodGet, "/products/"+draft.Slug, nil, "", false)
	if publicDraft.Code != http.StatusNotFound {
		t.Fatalf("public draft status = %d, want %d", publicDraft.Code, http.StatusNotFound)
	}

	adminList := api.request(t, http.MethodGet, "/products?status=all", nil, "", true)
	if adminList.Code != http.StatusOK {
		t.Fatalf("admin list status = %d, want %d", adminList.Code, http.StatusOK)
	}
	var adminPayload struct {
		Products []Product `json:"products"`
	}
	decodeResponse(t, adminList, &adminPayload)
	if len(adminPayload.Products) != 3 {
		t.Fatalf("admin products count = %d, want 3", len(adminPayload.Products))
	}

	adminDraft := api.request(t, http.MethodGet, "/products/"+draft.ID, nil, "", true)
	if adminDraft.Code != http.StatusOK {
		t.Fatalf("admin draft status = %d, want %d", adminDraft.Code, http.StatusOK)
	}

	categoryResponse := api.request(t, http.MethodGet, "/products?category=Learning&featured=true", nil, "", true)
	if categoryResponse.Code != http.StatusOK {
		t.Fatalf("category status = %d, want %d", categoryResponse.Code, http.StatusOK)
	}
	var categoryFiltered struct {
		Products []Product `json:"products"`
	}
	decodeResponse(t, categoryResponse, &categoryFiltered)
	if len(categoryFiltered.Products) != 1 || categoryFiltered.Products[0].ID != anki.ID {
		t.Fatalf("unexpected category products: %+v", categoryFiltered.Products)
	}

	tagResponse := api.request(t, http.MethodGet, "/products?tag=anki", nil, "", true)
	if tagResponse.Code != http.StatusOK {
		t.Fatalf("tag status = %d, want %d", tagResponse.Code, http.StatusOK)
	}
	var tagFiltered struct {
		Products []Product `json:"products"`
	}
	decodeResponse(t, tagResponse, &tagFiltered)
	if len(tagFiltered.Products) != 2 {
		t.Fatalf("unexpected tag products: %+v", tagFiltered.Products)
	}

	searchResponse := api.request(t, http.MethodGet, "/products?q=spaced", nil, "", true)
	if searchResponse.Code != http.StatusOK {
		t.Fatalf("search status = %d, want %d", searchResponse.Code, http.StatusOK)
	}
	var search struct {
		Products []Product `json:"products"`
	}
	decodeResponse(t, searchResponse, &search)
	if len(search.Products) != 1 || search.Products[0].ID != anki.ID {
		t.Fatalf("unexpected search products: %+v", search.Products)
	}

	patchBody := `{"price_label":"$29","featured":false,"sort_order":0,"status":"draft"}`
	patchResponse := api.request(t, http.MethodPatch, "/products/"+anki.Slug, strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated Product
	decodeResponse(t, patchResponse, &updated)
	if updated.PriceLabel != "$29" || updated.Featured || updated.SortOrder != 0 || updated.Status != StatusDraft {
		t.Fatalf("unexpected updated product: %+v", updated)
	}

	deleteResponse := api.request(t, http.MethodDelete, "/products/"+archived.ID, nil, "", true)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want %d", deleteResponse.Code, http.StatusNoContent)
	}
}

func TestProductSectionSettings(t *testing.T) {
	api := newTestAPI(t)

	getResponse := api.request(t, http.MethodGet, "/products/section", nil, "", false)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("section status = %d, want %d", getResponse.Code, http.StatusOK)
	}
	var initial ProductSectionSettings
	decodeResponse(t, getResponse, &initial)
	if initial.Enabled || initial.Title != "Products" {
		t.Fatalf("initial section = %+v, want disabled Products", initial)
	}

	patchBody := `{"enabled":true,"title":"Esoteric Section","description":"Selected products and study assets."}`
	patchResponse := api.request(t, http.MethodPatch, "/products/section", strings.NewReader(patchBody), "application/json", true)
	if patchResponse.Code != http.StatusOK {
		t.Fatalf("section patch status = %d, want %d: %s", patchResponse.Code, http.StatusOK, patchResponse.Body.String())
	}
	var updated ProductSectionSettings
	decodeResponse(t, patchResponse, &updated)
	if !updated.Enabled || updated.Title != "Esoteric Section" || updated.Description != "Selected products and study assets." {
		t.Fatalf("updated section = %+v", updated)
	}

	reload := api.request(t, http.MethodGet, "/products/section", nil, "", false)
	if reload.Code != http.StatusOK {
		t.Fatalf("section reload status = %d, want %d", reload.Code, http.StatusOK)
	}
	var public ProductSectionSettings
	decodeResponse(t, reload, &public)
	if !public.Enabled || public.Title != "Esoteric Section" {
		t.Fatalf("public section = %+v", public)
	}
}

func TestProductValidationAndFilterValidation(t *testing.T) {
	api := newTestAPI(t)

	body := `{
		"slug": "Bad Slug",
		"title": " ",
		"cover_image_url": "ftp://example.com/image.png",
		"cta_url": "notaurl",
		"sort_order": -1,
		"status": "live"
	}`
	response := api.request(t, http.MethodPost, "/products", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusBadRequest, response.Body.String())
	}

	var payload struct {
		Fields map[string]string `json:"fields"`
	}
	decodeResponse(t, response, &payload)
	for _, field := range []string{"title", "slug", "cover_image_url", "cta_url", "sort_order", "status"} {
		if payload.Fields[field] == "" {
			t.Fatalf("missing validation field %q in %+v", field, payload.Fields)
		}
	}

	missingCTA := api.request(t, http.MethodPost, "/products", strings.NewReader(`{"title":"Deck"}`), "application/json", true)
	if missingCTA.Code != http.StatusBadRequest {
		t.Fatalf("missing cta status = %d, want %d", missingCTA.Code, http.StatusBadRequest)
	}

	filterResponse := api.request(t, http.MethodGet, "/products?featured=maybe", nil, "", false)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("featured filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}

	filterResponse = api.request(t, http.MethodGet, "/products?status=live", nil, "", true)
	if filterResponse.Code != http.StatusBadRequest {
		t.Fatalf("status filter status = %d, want %d", filterResponse.Code, http.StatusBadRequest)
	}

	sectionResponse := api.request(t, http.MethodPatch, "/products/section", strings.NewReader(`{"title":" "}`), "application/json", true)
	if sectionResponse.Code != http.StatusBadRequest {
		t.Fatalf("section validation status = %d, want %d", sectionResponse.Code, http.StatusBadRequest)
	}
}

func createProduct(t *testing.T, api testAPI, body string) Product {
	t.Helper()

	response := api.request(t, http.MethodPost, "/products", strings.NewReader(body), "application/json", true)
	if response.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}

	var product Product
	decodeResponse(t, response, &product)
	return product
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
