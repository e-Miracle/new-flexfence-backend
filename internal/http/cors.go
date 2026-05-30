package http

import (
	"net/http"
	"strings"
)

// CORSMiddleware handles browser preflight and cross-origin API access from the dashboard.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAll := false
	for _, o := range allowedOrigins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			break
		}
		allowed[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAll || originAllowed(allowed, origin) {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			w.Header().Set("Access-Control-Max-Age", "86400")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func originAllowed(allowed map[string]struct{}, origin string) bool {
	if _, ok := allowed[origin]; ok {
		return true
	}
	// Dev convenience: allow any localhost port if 127.0.0.1 or localhost is listed.
	for candidate := range allowed {
		if strings.Contains(candidate, "localhost") && strings.Contains(origin, "localhost") {
			return true
		}
		if strings.Contains(candidate, "127.0.0.1") && strings.Contains(origin, "127.0.0.1") {
			return true
		}
	}
	return false
}
