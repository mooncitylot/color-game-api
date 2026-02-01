package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/color-game/api/datastore"
	"github.com/color-game/api/models"
)

// GET /v1/friends
func (app *Application) getFriends(w http.ResponseWriter, r *http.Request) {
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	friends, err := app.FriendRepo.ListFriends(user.UserID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"friends": friends,
	})
}

// GET /v1/friends/requests
func (app *Application) getFriendRequests(w http.ResponseWriter, r *http.Request) {
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	requests, err := app.FriendRepo.ListFriendRequests(user.UserID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"requests": requests,
	})
}

// POST /v1/friends/search
func (app *Application) searchFriends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	var payload struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	query := strings.TrimSpace(payload.Query)
	if len(query) < 2 {
		app.badRequest(w, r, errors.New("search query must be at least 2 characters"))
		return
	}

	results, err := app.FriendRepo.SearchUsersForFriend(user.UserID, query, 20)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
	})
}

// POST /v1/friends/request
func (app *Application) createFriendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	var payload struct {
		TargetUserID string `json:"targetUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	if payload.TargetUserID == "" {
		app.badRequest(w, r, errors.New("targetUserId is required"))
		return
	}

	// Ensure target exists
	if _, err := app.UserRepo.Get(payload.TargetUserID); err != nil {
		app.badRequest(w, r, errors.New("user not found"))
		return
	}

	friendship, err := app.FriendRepo.CreateFriendRequest(user.UserID, payload.TargetUserID)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(friendship)
}

// POST /v1/friends/respond
func (app *Application) respondToFriendRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	var payload struct {
		FriendshipID int    `json:"friendshipId"`
		Action       string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	if payload.FriendshipID == 0 || payload.Action == "" {
		app.badRequest(w, r, errors.New("friendshipId and action are required"))
		return
	}

	action := strings.ToLower(payload.Action)
	var newStatus string
	switch action {
	case "accept":
		newStatus = models.FriendshipStatusAccepted
	case "decline":
		newStatus = models.FriendshipStatusDeclined
	default:
		app.badRequest(w, r, errors.New("invalid action"))
		return
	}

	friendship, err := app.FriendRepo.UpdateFriendshipStatus(payload.FriendshipID, newStatus)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Ensure current user is part of this friendship
	if friendship.RequesterID != user.UserID && friendship.AddresseeID != user.UserID {
		app.invalidAuthorization(w, r, errors.New("not authorized for this friendship"))
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(friendship)
}

// POST /v1/friends/remove
func (app *Application) removeFriend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	var payload struct {
		FriendshipID int `json:"friendshipId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	if payload.FriendshipID == 0 {
		app.badRequest(w, r, errors.New("friendshipId is required"))
		return
	}

	friendship, err := app.FriendRepo.DeleteFriendship(payload.FriendshipID, user.UserID)
	if err != nil {
		if _, ok := err.(datastore.NoRowsError); ok {
			app.badRequest(w, r, errors.New("friendship not found"))
			return
		}
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"removedFriendship": friendship,
	})
}

// GET /v1/friends/activity
func (app *Application) getFriendActivity(w http.ResponseWriter, r *http.Request) {
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	activities, err := app.FriendRepo.GetFriendActivities(user.UserID, 7)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activity": activities,
	})
}
