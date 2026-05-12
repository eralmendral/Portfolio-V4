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
	}), []string{"http://localhost:4200"})

	request := httptest.NewRequest(http.MethodOptions, "/projects", nil)
	request.Header.Set("Origin", "http://localhost:4200")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if called {
		t.Fatal("next handler should not be called for allowed preflight")
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:4200" {
		t.Fatalf("allow origin = %q, want http://localhost:4200", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Headers"); got != "Authorization, Content-Type, Accept" {
		t.Fatalf("allow headers = %q, want Authorization, Content-Type, Accept", got)
	}
}

func TestCommonHeadersAllowsAnyConfiguredOrigin(t *testing.T) {
	handler := withCommonHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), []string{"http://localhost:4200", "http://127.0.0.1:4200"})

	request := httptest.NewRequest(http.MethodGet, "/projects", nil)
	request.Header.Set("Origin", "http://127.0.0.1:4200")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://127.0.0.1:4200" {
		t.Fatalf("allow origin = %q, want http://127.0.0.1:4200", got)
	}
}

func TestClientOriginsFromEnvParsesCommaSeparatedOrigins(t *testing.T) {
	t.Setenv("CLIENT_ORIGINS", " http://localhost:4200, http://127.0.0.1:4200/ ")
	t.Setenv("CLIENT_ORIGIN", "http://localhost:4200")

	origins := clientOriginsFromEnv()

	if len(origins) != 2 {
		t.Fatalf("origin count = %d, want 2: %#v", len(origins), origins)
	}
	if origins[0] != "http://localhost:4200" || origins[1] != "http://127.0.0.1:4200" {
		t.Fatalf("origins = %#v, want localhost and 127.0.0.1", origins)
	}
}

func TestCommonHeadersSkipsCORSWhenOriginIsNotConfigured(t *testing.T) {
	handler := withCommonHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}), nil)

	request := httptest.NewRequest(http.MethodOptions, "/projects", nil)
	request.Header.Set("Origin", "http://localhost:4200")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow origin = %q, want empty", got)
	}
}

func TestRequestMonitoringAddsRequestID(t *testing.T) {
	handler := withRequestMonitoring(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := response.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID header should be set")
	}
}

func TestRequestMonitoringKeepsIncomingRequestID(t *testing.T) {
	handler := withRequestMonitoring(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	request := httptest.NewRequest(http.MethodGet, "/projects", nil)
	request.Header.Set("X-Request-ID", "existing-request-id")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusAccepted)
	}
	if got := response.Header().Get("X-Request-ID"); got != "existing-request-id" {
		t.Fatalf("X-Request-ID = %q, want existing-request-id", got)
	}
}

func TestRequestMonitoringRecoversPanics(t *testing.T) {
	handler := withRequestMonitoring(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("X-Request-ID header should be set")
	}
}
