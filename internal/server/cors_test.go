package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dokku-mcp/dokku-mcp/pkg/config"
)

func TestCORSMiddleware_Disabled(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled: false,
	}

	handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// When disabled, no CORS headers should be set by our middleware
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("Expected no CORS headers when disabled, got: %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_AllowAll(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{}, // Empty means allow all
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
		MaxAge:         300,
	}

	handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got: %s", got)
	}

	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Errorf("Expected Access-Control-Allow-Methods: GET, POST, got: %s", got)
	}
}

func TestCORSMiddleware_SpecificOrigins(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://app.example.com", "https://dashboard.example.com"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	}

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
		expectVary     bool
	}{
		{
			name:           "allowed origin",
			origin:         "https://app.example.com",
			expectedOrigin: "https://app.example.com",
			expectVary:     true,
		},
		{
			name:           "another allowed origin",
			origin:         "https://dashboard.example.com",
			expectedOrigin: "https://dashboard.example.com",
			expectVary:     true,
		},
		{
			name:           "disallowed origin",
			origin:         "https://evil.com",
			expectedOrigin: "",
			expectVary:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.expectedOrigin {
				t.Errorf("Expected Access-Control-Allow-Origin: %s, got: %s", tt.expectedOrigin, got)
			}

			if tt.expectVary {
				if got := w.Header().Get("Vary"); got != "Origin" {
					t.Errorf("Expected Vary: Origin, got: %s", got)
				}
			}
		})
	}
}

func TestCORSMiddleware_WildcardSubdomain(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"*.example.com"},
		AllowedMethods: []string{"GET", "POST"},
	}

	tests := []struct {
		name           string
		origin         string
		expectedOrigin string
	}{
		{
			name:           "subdomain match",
			origin:         "https://app.example.com",
			expectedOrigin: "https://app.example.com",
		},
		{
			name:           "another subdomain match",
			origin:         "https://api.example.com",
			expectedOrigin: "https://api.example.com",
		},
		{
			name:           "no match - different domain",
			origin:         "https://example.org",
			expectedOrigin: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.expectedOrigin {
				t.Errorf("Expected Access-Control-Allow-Origin: %s, got: %s", tt.expectedOrigin, got)
			}
		})
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	cfg := &config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"https://app.example.com"},
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
		MaxAge:         300,
	}

	handler := CORSMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS request")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d for OPTIONS request, got: %d", http.StatusNoContent, w.Code)
	}

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin: https://app.example.com, got: %s", got)
	}

	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Errorf("Expected Access-Control-Allow-Methods: GET, POST, OPTIONS, got: %s", got)
	}
}

func TestIsOriginAllowed(t *testing.T) {
	tests := []struct {
		name           string
		origin         string
		allowedOrigins []string
		expected       bool
	}{
		{
			name:           "wildcard allows all",
			origin:         "https://anything.com",
			allowedOrigins: []string{"*"},
			expected:       true,
		},
		{
			name:           "exact match",
			origin:         "https://app.example.com",
			allowedOrigins: []string{"https://app.example.com"},
			expected:       true,
		},
		{
			name:           "no match",
			origin:         "https://evil.com",
			allowedOrigins: []string{"https://app.example.com"},
			expected:       false,
		},
		{
			name:           "wildcard subdomain match",
			origin:         "https://api.example.com",
			allowedOrigins: []string{"*.example.com"},
			expected:       true,
		},
		{
			name:           "wildcard subdomain no match",
			origin:         "https://example.org",
			allowedOrigins: []string{"*.example.com"},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isOriginAllowed(tt.origin, tt.allowedOrigins)
			if got != tt.expected {
				t.Errorf("isOriginAllowed(%s, %v) = %v, expected %v",
					tt.origin, tt.allowedOrigins, got, tt.expected)
			}
		})
	}
}
