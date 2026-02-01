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
	mux.HandleFunc("/v1/colors/random", app.getRandomColor)
	mux.HandleFunc("/v1/colors/daily", app.getDailyColor)
	mux.HandleFunc("/v1/colors/daily/all", app.getAllDailyColors)
	mux.HandleFunc("/v1/leaderboard", app.getLeaderboard)

	// Authenticated endpoints
	mux.HandleFunc("/v1/users/me", app.authenticate(app.getCurrentUser))
	mux.HandleFunc("/v1/users/me/update", app.authenticate(app.updateCurrentUser))
	mux.HandleFunc("/v1/scores/submit", app.authenticate(app.submitScore))
	mux.HandleFunc("/v1/scores/history", app.authenticate(app.getUserScoreHistory))

	// Friends endpoints
	mux.HandleFunc("/v1/friends", app.authenticate(app.getFriends))
	mux.HandleFunc("/v1/friends/requests", app.authenticate(app.getFriendRequests))
	mux.HandleFunc("/v1/friends/search", app.authenticate(app.searchFriends))
	mux.HandleFunc("/v1/friends/request", app.authenticate(app.createFriendRequest))
	mux.HandleFunc("/v1/friends/respond", app.authenticate(app.respondToFriendRequest))
	mux.HandleFunc("/v1/friends/remove", app.authenticate(app.removeFriend))
	mux.HandleFunc("/v1/friends/activity", app.authenticate(app.getFriendActivity))

	// Shop endpoints (public - browse items)
	mux.HandleFunc("/v1/shop/items", app.getShopItems)

	// Shop endpoints (authenticated)
	mux.HandleFunc("/v1/shop/purchase", app.authenticate(app.purchaseItem))
	mux.HandleFunc("/v1/inventory", app.authenticate(app.getUserInventory))
	mux.HandleFunc("/v1/inventory/equipped", app.authenticate(app.getEquippedItems))
	mux.HandleFunc("/v1/inventory/equip", app.authenticate(app.equipItem))
	mux.HandleFunc("/v1/inventory/use", app.authenticate(app.useItem))
	mux.HandleFunc("/v1/shop/purchases", app.authenticate(app.getPurchaseHistory))

	// Admin endpoints
	mux.HandleFunc("/v1/users", app.verifyPermissions(app.getAllUsers))
	mux.HandleFunc("/v1/admin/colors/generate", app.verifyPermissions(app.generateDailyColor))
	mux.HandleFunc("/v1/admin/shop/items", app.verifyPermissions(app.createShopItem))
	mux.HandleFunc("/v1/admin/shop/items/all", app.verifyPermissions(app.getAllShopItems))
	mux.HandleFunc("/v1/admin/shop/items/update", app.verifyPermissions(app.updateShopItem))
	mux.HandleFunc("/v1/admin/shop/items/delete", app.verifyPermissions(app.deactivateShopItem))
	mux.HandleFunc("/v1/admin/users/credits", app.verifyPermissions(app.addUserCredits))
	mux.HandleFunc("/v1/admin/shop/purchases", app.verifyPermissions(app.getAdminPurchases))
	mux.HandleFunc("/v1/admin/scores/reset", app.verifyPermissions(app.resetUserDailyAttempts))

	// Wrap entire mux with CORS and origins check
	finalMux.Handle("/", wrapMuxWithCorsAndOrigins(mux, app))

	return finalMux
}
