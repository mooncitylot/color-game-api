package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/color-game/api/datastore"
	"github.com/color-game/api/models"
	"github.com/golang-jwt/jwt/v5"
)

// GET /
func (app *Application) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Color Game API")
}

// POST /v1/auth/signup
func (app *Application) signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	userSignup := &models.UserSignupRequest{}
	errParsingJson := json.NewDecoder(r.Body).Decode(userSignup)
	if errParsingJson != nil {
		app.badJSONRequest(w, r, errParsingJson)
		return
	}

	// Validate username doesn't contain spaces
	if len(userSignup.Username) == 0 {
		app.badRequest(w, r, errors.New("username is required"))
		return
	}

	// Check for spaces in username
	for _, char := range userSignup.Username {
		if char == ' ' {
			app.badRequest(w, r, errors.New("username cannot contain spaces"))
			return
		}
	}

	// Create new user
	newUser, newUserErr := models.NewUser(*userSignup)
	if newUserErr != nil {
		app.internalServerError(w, r, newUserErr)
		return
	}

	// Check if email already exists
	_, getErr := app.UserRepo.GetUserByEmail(newUser.Email)
	if getErr == nil {
		app.userAlreadyExists(w, r, getErr)
		return
	}

	// Check if username already exists
	_, getUsernameErr := app.UserRepo.GetUserByUsername(newUser.Username)
	if getUsernameErr == nil {
		app.badRequest(w, r, errors.New("username already taken"))
		return
	}

	// Store new user in database
	storedUser, errStoringNewUser := app.UserRepo.Create(newUser)
	if errStoringNewUser != nil {
		app.internalServerError(w, r, errStoringNewUser)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(storedUser)
}

// POST /v1/auth/login
func (app *Application) login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	// Parse credentials with device fingerprint
	creds := &models.Credentials{}
	if err := json.NewDecoder(r.Body).Decode(creds); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Validate device fingerprint is provided
	if creds.DeviceFingerprint == "" {
		app.badJSONRequest(w, r, errors.New("deviceFingerprint is required"))
		return
	}

	// Validate user credentials
	user, err := app.UserRepo.ValidateAndGetUser(*creds)
	if err != nil {
		app.invalidCredentials(w, r, err)
		return
	}

	if !user.Approved {
		app.invalidCredentials(w, r, errors.New("user not yet approved"))
		return
	}

	// Create/update device record
	deviceExpiry := time.Now().Add(time.Second * time.Duration(app.Config.JwtRefreshDuration))
	device := models.UserDevice{
		UserID:      user.UserID,
		Fingerprint: creds.DeviceFingerprint,
		DeviceData:  r.Header.Get("User-Agent"),
		Expiry:      deviceExpiry,
	}

	if err := app.UserRepo.CreateDevice(device); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Generate JWT access token
	accessExpiry := time.Now().Add(time.Second * time.Duration(app.Config.JwtAccessDuration))

	// Create access token claims
	accessClaims := models.JWTClaims{
		UserID:            user.UserID,
		Email:             user.Email,
		Kind:              user.Kind,
		DeviceFingerprint: creds.DeviceFingerprint,
		Scope:             "authentication",
		TokenType:         models.JWT.ACCESS_COOKIE_NAME,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(app.Config.JwtSecret))
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	sameSite := http.SameSiteStrictMode
	if app.Config.JwtDomain == "" {
		sameSite = http.SameSiteNoneMode
	}

	// Set access token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     models.JWT.ACCESS_COOKIE_NAME,
		Value:    accessTokenString,
		HttpOnly: true,
		Secure:   true,
		SameSite: sameSite,
		Path:     "/",
		Domain:   app.Config.JwtDomain,
		Expires:  accessExpiry,
	})

	// Generate refresh token
	refreshExpiry := deviceExpiry
	refreshClaims := models.JWTClaims{
		UserID:            user.UserID,
		Email:             user.Email,
		Kind:              user.Kind,
		DeviceFingerprint: creds.DeviceFingerprint,
		Scope:             "refresh",
		TokenType:         models.JWT.REFRESH_COOKIE_NAME,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(app.Config.JwtSecret))
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Set refresh token cookie
	http.SetCookie(w, &http.Cookie{
		Name:     models.JWT.REFRESH_COOKIE_NAME,
		Value:    refreshTokenString,
		HttpOnly: true,
		Secure:   true,
		SameSite: sameSite,
		Path:     "/",
		Domain:   app.Config.JwtDomain,
		Expires:  refreshExpiry,
	})

	w.WriteHeader(http.StatusOK)
}

// GET /v1/users/me - Get current authenticated user
func (app *Application) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}

// PUT /v1/users/me - Update current authenticated user
func (app *Application) updateCurrentUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		app.requirePutMethod(w, r, ErrPUT)
		return
	}

	// Get current user from token
	currentUser, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Parse update request
	updateReq := &models.UserUpdateRequest{}
	if err := json.NewDecoder(r.Body).Decode(updateReq); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Update user fields
	currentUser.Username = updateReq.Username
	currentUser.Email = updateReq.Email
	currentUser.UpdatedAt = time.Now()

	// Save to database
	updatedUser, updateErr := app.UserRepo.Update(currentUser)
	if updateErr != nil {
		app.internalServerError(w, r, updateErr)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(updatedUser)
}

// GET /v1/users - Get all users
func (app *Application) getAllUsers(w http.ResponseWriter, r *http.Request) {
	users, retrieveErr := app.UserRepo.GetAllUsers()
	if retrieveErr != nil {
		app.internalServerError(w, r, retrieveErr)
		return
	}

	json.NewEncoder(w).Encode(users)
}

// GET /v1/colors/random - Get a random color palette
func (app *Application) getRandomColor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate random RGB values
	r1 := rand.Intn(256)
	g := rand.Intn(256)
	b := rand.Intn(256)

	// Build the URL for thecolorapi.com
	url := fmt.Sprintf("https://www.thecolorapi.com/scheme?rgb=%d,%d,%d&mode=analogic&count=6&format=json", r1, g, b)

	// Make HTTP request to the color API
	resp, err := http.Get(url)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		app.internalServerError(w, r, fmt.Errorf("color API returned status: %d", resp.StatusCode))
		return
	}

	// Parse the response
	var colorResponse models.ColorAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&colorResponse); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Return the color palette
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(colorResponse)
}

// GET /v1/colors/daily - Get today's daily color
func (app *Application) getDailyColor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get today's color from database
	dailyColor, err := app.DailyColorRepo.GetToday()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Format response
	response := models.DailyColorResponse{
		Date:      dailyColor.Date.Format("2006-01-02"),
		ColorName: dailyColor.ColorName,
		RGB:       fmt.Sprintf("rgb(%d,%d,%d)", dailyColor.R, dailyColor.G, dailyColor.B),
		Hex:       fmt.Sprintf("#%02X%02X%02X", dailyColor.R, dailyColor.G, dailyColor.B),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GET /v1/colors/daily/all - Get all daily colors
func (app *Application) getAllDailyColors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all colors from database
	dailyColors, err := app.DailyColorRepo.GetAll()
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Format response
	var responses []models.DailyColorResponse
	for _, dc := range dailyColors {
		responses = append(responses, models.DailyColorResponse{
			Date:      dc.Date.Format("2006-01-02"),
			ColorName: dc.ColorName,
			RGB:       fmt.Sprintf("rgb(%d,%d,%d)", dc.R, dc.G, dc.B),
			Hex:       fmt.Sprintf("#%02X%02X%02X", dc.R, dc.G, dc.B),
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(responses)
}

// calculateColorScore calculates a score (0-100) based on color similarity
// Uses Euclidean distance in RGB space, normalized to 0-100
func calculateColorScore(targetR, targetG, targetB, submittedR, submittedG, submittedB int) int {
	// Calculate Euclidean distance
	distance := math.Sqrt(
		math.Pow(float64(targetR-submittedR), 2) +
			math.Pow(float64(targetG-submittedG), 2) +
			math.Pow(float64(targetB-submittedB), 2),
	)

	// Maximum possible distance in RGB space is sqrt(255^2 + 255^2 + 255^2) ≈ 441.67
	maxDistance := 441.67

	// Convert distance to score (0-100, where 100 is perfect match)
	score := int(math.Round((1 - (distance / maxDistance)) * 100))

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// POST /v1/scores/submit - Submit a score attempt
func (app *Application) submitScore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	// Get current user from token
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Parse submission
	var submission models.ScoreSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	// Validate RGB values
	if submission.SubmittedColorR < 0 || submission.SubmittedColorR > 255 ||
		submission.SubmittedColorG < 0 || submission.SubmittedColorG > 255 ||
		submission.SubmittedColorB < 0 || submission.SubmittedColorB > 255 {
		app.badJSONRequest(w, r, errors.New("RGB values must be between 0 and 255"))
		return
	}

	// Get today's color
	today := time.Now()
	normalizedToday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	dailyColor, err := app.DailyColorRepo.GetToday()
	if err != nil {
		app.internalServerError(w, r, errors.New("no daily color available for today"))
		return
	}

	// Check how many attempts the user has made today
	attemptCount, err := app.DailyScoreRepo.GetUserAttemptCount(user.UserID, normalizedToday)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	extraAttempts := 0
	modifier, err := app.DailyScoreRepo.GetDailyAttemptModifier(user.UserID, normalizedToday)
	if err == nil {
		extraAttempts = modifier.ExtraAttempts
	} else if _, ok := err.(datastore.NoRowsError); !ok {
		app.internalServerError(w, r, err)
		return
	}

	maxAttempts := 5 + extraAttempts
	if maxAttempts > 10 {
		maxAttempts = 10
	}

	if attemptCount >= maxAttempts {
		http.Error(w, fmt.Sprintf("Maximum attempts (%d) reached for today", maxAttempts), http.StatusBadRequest)
		return
	}

	// Calculate score
	score := calculateColorScore(
		dailyColor.R, dailyColor.G, dailyColor.B,
		submission.SubmittedColorR, submission.SubmittedColorG, submission.SubmittedColorB,
	)

	// Create daily score entry
	dailyScore := models.DailyScore{
		UserID:          user.UserID,
		Date:            normalizedToday,
		AttemptNumber:   attemptCount + 1,
		Score:           score,
		SubmittedColorR: submission.SubmittedColorR,
		SubmittedColorG: submission.SubmittedColorG,
		SubmittedColorB: submission.SubmittedColorB,
		TargetColorR:    dailyColor.R,
		TargetColorG:    dailyColor.G,
		TargetColorB:    dailyColor.B,
		CreatedAt:       time.Now(),
	}

	// Save the score
	savedScore, err := app.DailyScoreRepo.Create(dailyScore)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Get user's best score for today
	existingLeaderboard, err := app.DailyLeaderboardRepo.GetByUserAndDate(user.UserID, normalizedToday)
	hasExistingLeaderboard := true
	if err != nil {
		if _, ok := err.(datastore.NoRowsError); ok {
			hasExistingLeaderboard = false
		} else {
			app.internalServerError(w, r, err)
			return
		}
	}

	isNewBest := false
	bestScore := score
	bestAttemptsUsed := savedScore.AttemptNumber

	if !hasExistingLeaderboard {
		isNewBest = true
	} else {
		bestScore = existingLeaderboard.BestScore
		bestAttemptsUsed = existingLeaderboard.AttemptsUsed

		if score > existingLeaderboard.BestScore {
			isNewBest = true
			bestScore = score
			bestAttemptsUsed = savedScore.AttemptNumber
		}
	}

	// Update leaderboard if this is the best score
	if isNewBest {
		leaderboardEntry := models.DailyLeaderboard{
			UserID:       user.UserID,
			Date:         normalizedToday,
			BestScore:    bestScore,
			AttemptsUsed: bestAttemptsUsed,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err = app.DailyLeaderboardRepo.CreateOrUpdate(leaderboardEntry)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}
	}

	if err := app.FriendRepo.RecordFriendActivity(user.UserID, normalizedToday, bestScore, bestAttemptsUsed); err != nil {
		log.Printf("failed to record friend activity for user %s: %v", user.UserID, err)
	}

	// Build response
	attemptsLeft := maxAttempts - savedScore.AttemptNumber
	message := ""

	if score == 100 {
		message = "Perfect match! You got the exact color!"
	} else if score >= 90 {
		message = "Excellent! Very close!"
	} else if score >= 75 {
		message = "Great job! Pretty close!"
	} else if score >= 50 {
		message = "Not bad! Keep trying!"
	} else {
		message = "Keep practicing!"
	}

	if attemptsLeft == 0 {
		message += " No more attempts left for today."

		pointsAward := bestScore
		newTotalPoints := user.Points + pointsAward
		prevMilestones := user.Points / 1000
		newMilestones := newTotalPoints / 1000
		levelUps := newMilestones - prevMilestones
		if levelUps < 0 {
			levelUps = 0
		}

		if levelUps > 0 {
			user.Level += levelUps
		}

		user.Points = newTotalPoints

		creditAward := int(math.Ceil(float64(bestScore) / 2.0))
		user.Credits += creditAward
		user.UpdatedAt = time.Now()

		if _, err := app.UserRepo.Update(user); err != nil {
			app.internalServerError(w, r, fmt.Errorf("failed to finalize daily rewards: %v", err))
			return
		}
	}

	response := models.ScoreSubmissionResponse{
		Score:          score,
		AttemptNumber:  savedScore.AttemptNumber,
		AttemptsLeft:   attemptsLeft,
		MaxAttempts:    maxAttempts,
		BestScore:      bestScore,
		IsNewBest:      isNewBest,
		SubmittedColor: fmt.Sprintf("rgb(%d,%d,%d)", submission.SubmittedColorR, submission.SubmittedColorG, submission.SubmittedColorB),
		TargetColor:    fmt.Sprintf("rgb(%d,%d,%d)", dailyColor.R, dailyColor.G, dailyColor.B),
		Message:        message,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GET /v1/leaderboard - Get today's leaderboard
func (app *Application) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get today's leaderboard (top 100)
	today := time.Now()
	leaderboard, err := app.DailyLeaderboardRepo.GetLeaderboardByDate(today, 100)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(leaderboard)
}

// GET /v1/scores/history - Get user's score history
func (app *Application) getUserScoreHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get current user from token
	user, err := app.getUserFromToken(w, r)
	if err != nil {
		return
	}

	// Get today's attempts
	today := time.Now()
	normalizedToday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	attempts, err := app.DailyScoreRepo.GetUserScoresByDate(user.UserID, normalizedToday)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Get leaderboard entry for best score
	leaderboardEntry, err := app.DailyLeaderboardRepo.GetByUserAndDate(user.UserID, normalizedToday)

	bestScore := 0
	attemptsUsed := len(attempts)
	if err == nil {
		bestScore = leaderboardEntry.BestScore
	} else if len(attempts) > 0 {
		// Calculate best score from attempts
		for _, attempt := range attempts {
			if attempt.Score > bestScore {
				bestScore = attempt.Score
			}
		}
	}

	extraAttempts := 0
	modifier, err := app.DailyScoreRepo.GetDailyAttemptModifier(user.UserID, normalizedToday)
	if err == nil {
		extraAttempts = modifier.ExtraAttempts
	} else if _, ok := err.(datastore.NoRowsError); !ok {
		app.internalServerError(w, r, err)
		return
	}

	maxAttempts := 5 + extraAttempts
	if maxAttempts > 10 {
		maxAttempts = 10
	}

	attemptsLeft := maxAttempts - attemptsUsed
	if attemptsLeft < 0 {
		attemptsLeft = 0
	}

	response := models.UserScoreHistory{
		Date:          normalizedToday.Format("2006-01-02"),
		Attempts:      attempts,
		BestScore:     bestScore,
		AttemptsUsed:  attemptsUsed,
		AttemptsLeft:  attemptsLeft,
		ExtraAttempts: extraAttempts,
		MaxAttempts:   maxAttempts,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

type resetAttemptsRequest struct {
	UserID string `json:"user_id"`
	Date   string `json:"date"`
}

type resetAttemptsResponse struct {
	UserID              string `json:"user_id"`
	Date                string `json:"date"`
	ScoresDeleted       int64  `json:"scores_deleted"`
	LeaderboardCleared  bool   `json:"leaderboard_cleared"`
	FriendActivityReset bool   `json:"friend_activity_reset"`
}

// POST /v1/admin/scores/reset - Reset a user's daily attempts (Admin only)
func (app *Application) resetUserDailyAttempts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	var req resetAttemptsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.badJSONRequest(w, r, err)
		return
	}

	if req.UserID == "" {
		app.badRequest(w, r, errors.New("user_id is required"))
		return
	}

	var targetDate time.Time
	if req.Date == "" {
		targetDate = time.Now()
	} else {
		parsed, err := time.Parse("2006-01-02", req.Date)
		if err != nil {
			app.badRequest(w, r, errors.New("date must be in YYYY-MM-DD format"))
			return
		}
		targetDate = parsed
	}

	normalizedDate := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())

	scoresDeleted, err := app.DailyScoreRepo.DeleteUserScoresByDate(req.UserID, normalizedDate)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	leaderboardRows, err := app.DailyLeaderboardRepo.DeleteByUserAndDate(req.UserID, normalizedDate)
	if err != nil {
		app.internalServerError(w, r, err)
		return
	}

	friendActivityReset := false
	if err := app.FriendRepo.RecordFriendActivity(req.UserID, normalizedDate, 0, 0); err == nil {
		friendActivityReset = true
	} else {
		log.Printf("failed to reset friend activity for user %s: %v", req.UserID, err)
	}

	response := resetAttemptsResponse{
		UserID:              req.UserID,
		Date:                normalizedDate.Format("2006-01-02"),
		ScoresDeleted:       scoresDeleted,
		LeaderboardCleared:  leaderboardRows > 0,
		FriendActivityReset: friendActivityReset,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// POST /v1/admin/colors/generate - Manually generate today's color (Admin only)
func (app *Application) generateDailyColor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		app.requirePostMethod(w, r, ErrPOST)
		return
	}

	// Get today's date
	today := time.Now()
	normalizedToday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	// Check if today's color already exists
	existingColor, err := app.DailyColorRepo.GetByDate(normalizedToday)
	if err == nil && existingColor.ID != 0 {
		// Color already exists, return it
		response := models.DailyColorResponse{
			Date:      existingColor.Date.Format("2006-01-02"),
			ColorName: existingColor.ColorName,
			RGB:       fmt.Sprintf("rgb(%d,%d,%d)", existingColor.R, existingColor.G, existingColor.B),
			Hex:       fmt.Sprintf("#%02X%02X%02X", existingColor.R, existingColor.G, existingColor.B),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Daily color already exists for today",
			"color":   response,
		})
		return
	}

	// Generate random RGB values
	r1 := rand.Intn(256)
	g := rand.Intn(256)
	b := rand.Intn(256)

	// Build the URL for thecolorapi.com
	url := fmt.Sprintf("https://www.thecolorapi.com/scheme?rgb=%d,%d,%d&mode=analogic&count=6&format=json", r1, g, b)

	// Make HTTP request to the color API
	resp, httpErr := http.Get(url)
	if httpErr != nil {
		app.internalServerError(w, r, httpErr)
		return
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		app.internalServerError(w, r, fmt.Errorf("color API returned status: %d", resp.StatusCode))
		return
	}

	// Parse the response
	var colorResponse models.ColorAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&colorResponse); err != nil {
		app.internalServerError(w, r, err)
		return
	}

	// Use the seed color (the original random color)
	seedColor := colorResponse.Seed
	colorName := seedColor.Name.Value

	// Create daily color entry
	dailyColor := models.DailyColor{
		Date:      normalizedToday,
		ColorName: colorName,
		R:         seedColor.RGB.R,
		G:         seedColor.RGB.G,
		B:         seedColor.RGB.B,
		CreatedAt: time.Now(),
	}

	// Save to database
	savedColor, saveErr := app.DailyColorRepo.Create(dailyColor)
	if saveErr != nil {
		app.internalServerError(w, r, saveErr)
		return
	}

	// Format response
	response := models.DailyColorResponse{
		Date:      savedColor.Date.Format("2006-01-02"),
		ColorName: savedColor.ColorName,
		RGB:       fmt.Sprintf("rgb(%d,%d,%d)", savedColor.R, savedColor.G, savedColor.B),
		Hex:       fmt.Sprintf("#%02X%02X%02X", savedColor.R, savedColor.G, savedColor.B),
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Successfully generated daily color",
		"color":   response,
	})
}
