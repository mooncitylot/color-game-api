package api

import (
	"net/http"
	"regexp"
	"strings"
)

func cleanOrigin(origin string) string {
	cleanedOrigin := strings.TrimPrefix(origin, "https://")
	cleanedOrigin = strings.TrimPrefix(cleanedOrigin, "wss://")
	if idx := strings.Index(cleanedOrigin, "/"); idx != -1 {
		cleanedOrigin = cleanedOrigin[:idx]
	}
	return cleanedOrigin
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	cleanedRequest := cleanOrigin(origin)

	// Allow localhost for development
	localhostPattern := regexp.MustCompile(`^localhost:\d+$`)
	if localhostPattern.MatchString(cleanedRequest) {
		return true
	}

	// Check against configured allowed origins
	for _, allowed := range allowedOrigins {
		cleanedAllowed := cleanOrigin(allowed)
		if cleanedAllowed == cleanedRequest {
			return true
		}
	}

	return false
}

func wrapMuxWithCorsAndOrigins(mux *http.ServeMux, app Application) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "" {
			referer := r.Header.Get("Referer")
			if referer != "" {
				origin = referer
			}
		}

		if origin == "" {
			handleCors(mux.ServeHTTP)(w, r)
			return
		}

		// Check if origin is allowed
		if isAllowedOrigin(origin, app.Config.AllowedOrigins) {
			handleCors(mux.ServeHTTP)(w, r)
			return
		}

		w.WriteHeader(403)
		w.Write([]byte("origin not allowed: " + cleanOrigin(origin)))
	})
}

func (app Application) BuildRoutes(mux *http.ServeMux) *http.ServeMux {
	finalMux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/v1/auth/signup", app.signup)
	mux.HandleFunc("/v1/auth/login", app.login)

	// Authenticated endpoints
	mux.HandleFunc("/v1/users/me", app.authenticate(app.getCurrentUser))
	mux.HandleFunc("/v1/users/me/update", app.authenticate(app.updateCurrentUser))

	// Admin endpoints
	mux.HandleFunc("/v1/users", app.verifyPermissions(app.getAllUsers))

	// Wrap entire mux with CORS and origins check
	finalMux.Handle("/", wrapMuxWithCorsAndOrigins(mux, app))

	return finalMux
}
