package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCommonHeadersHandlesConfiguredCORSPreflight(t *testing.T) {
	called := false
	handler := withCommonHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}), "http://localhost:3000")

	request := httptest.NewRequest(http.MethodOptions, "/projects", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if called {
		t.Fatal("next handler should not be called for allowed preflight")
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("allow origin = %q, want http://localhost:3000", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, Accept" {
		t.Fatalf("allow headers = %q, want Authorization, Content-Type, Accept", got)
	}
}

func TestCommonHeadersSkipsCORSWhenOriginIsNotConfigured(t *testing.T) {
	handler := withCommonHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}), "")

	request := httptest.NewRequest(http.MethodOptions, "/projects", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow origin = %q, want empty", got)
	}
}
