package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

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

	// Create new user
	newUser, newUserErr := models.NewUser(*userSignup)
	if newUserErr != nil {
		app.internalServerError(w, r, newUserErr)
		return
	}

	// Check if user already exists
	_, getErr := app.UserRepo.GetUserByEmail(newUser.Email)
	if getErr == nil {
		app.userAlreadyExists(w, r, getErr)
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

	// Maximum 5 attempts per day
	if attemptCount >= 5 {
		http.Error(w, "Maximum attempts (5) reached for today", http.StatusBadRequest)
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
	isNewBest := false
	bestScore := score

	if err != nil {
		// No existing entry, this is the first attempt and best score
		isNewBest = true
	} else {
		// Check if this is a new best
		if score > existingLeaderboard.BestScore {
			isNewBest = true
			bestScore = score
		} else {
			bestScore = existingLeaderboard.BestScore
		}
	}

	// Update leaderboard if this is the best score
	if isNewBest {
		leaderboardEntry := models.DailyLeaderboard{
			UserID:       user.UserID,
			Date:         normalizedToday,
			BestScore:    score,
			AttemptsUsed: savedScore.AttemptNumber,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		_, err = app.DailyLeaderboardRepo.CreateOrUpdate(leaderboardEntry)
		if err != nil {
			app.internalServerError(w, r, err)
			return
		}
	}

	// Build response
	attemptsLeft := 5 - savedScore.AttemptNumber
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
	}

	response := models.ScoreSubmissionResponse{
		Score:          score,
		AttemptNumber:  savedScore.AttemptNumber,
		AttemptsLeft:   attemptsLeft,
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

	response := models.UserScoreHistory{
		Date:         normalizedToday.Format("2006-01-02"),
		Attempts:     attempts,
		BestScore:    bestScore,
		AttemptsUsed: attemptsUsed,
		AttemptsLeft: 5 - attemptsUsed,
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
