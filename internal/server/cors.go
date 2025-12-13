package server

import (
	"net/http"
	"strings"

	"github.com/dokku-mcp/dokku-mcp/pkg/config"
)

// CORSMiddleware wraps an HTTP handler with CORS headers based on configuration
func CORSMiddleware(cfg *config.CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If CORS is not enabled, skip middleware
			if !cfg.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Set Access-Control-Allow-Origin
			origin := r.Header.Get("Origin")
			if len(cfg.AllowedOrigins) == 0 {
				// No origins specified, allow all
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if isOriginAllowed(origin, cfg.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}

			// Set Access-Control-Allow-Methods
			if len(cfg.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowedMethods, ", "))
			}

			// Set Access-Control-Allow-Headers
			if len(cfg.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowedHeaders, ", "))
			}

			// Set Access-Control-Max-Age
			if cfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", string(rune(cfg.MaxAge)))
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the origin is in the allowed list
// Supports exact matches and wildcard subdomains (e.g., "*.example.com")
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Check for wildcard subdomain match
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
