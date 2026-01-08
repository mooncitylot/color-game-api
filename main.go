package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/color-game/api/api"
	"github.com/color-game/api/datastore"
	"github.com/color-game/api/migrations"
	"github.com/color-game/api/scheduler"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load()

	// Get configuration from environment
	config := api.Config{
		HTTPPort:           getEnv("HTTP_PORT", ":8080"),
		DatabaseType:       getEnv("DB_TYPE", "postgres"),
		DatabaseUser:       getEnv("DB_USER", "postgres"),
		DatabasePassword:   getEnv("DB_PASSWORD", ""),
		DatabaseName:       getEnv("DB_NAME", "colorgame"),
		SSLMode:            getEnv("SSL_MODE", "disable"),
		JwtSecret:          getEnv("JWT_SECRET", "your-secret-key-change-this"),
		JwtAccessDuration:  getEnvInt("JWT_ACCESS_DURATION", 900),     // 15 minutes
		JwtRefreshDuration: getEnvInt("JWT_REFRESH_DURATION", 604800), // 7 days
		JwtDomain:          getEnv("JWT_DOMAIN", ""),
		AllowedOrigins:     getEnvSlice("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"),
		DevMode:            getEnvBool("DEV_MODE", true),
	}

	// Create database connection
	connStr := datastore.BuildDBConnStr(
		config.DatabasePassword,
		config.DatabaseUser,
		config.DatabaseName,
		config.SSLMode,
	)

	dbConn, dbErr := datastore.NewDB(config.DatabaseType, connStr)
	if dbErr != nil {
		log.Fatalf("Failed to connect to database: %v", dbErr)
	}
	defer dbConn.Close()

	// Run database migrations
	fmt.Println("Running database migrations...")
	if err := migrations.RunMigrations(dbConn); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create user repository
	userRepo, userRepoErr := datastore.NewUserDatabase(dbConn)
	if userRepoErr != nil {
		log.Fatalf("Failed to create user repository: %v", userRepoErr)
	}

	// Create daily color repository
	dailyColorRepo, dailyColorRepoErr := datastore.NewDailyColorDatabase(dbConn)
	if dailyColorRepoErr != nil {
		log.Fatalf("Failed to create daily color repository: %v", dailyColorRepoErr)
	}

	// Create daily score repository
	dailyScoreRepo, dailyScoreRepoErr := datastore.NewDailyScoreDatabase(dbConn)
	if dailyScoreRepoErr != nil {
		log.Fatalf("Failed to create daily score repository: %v", dailyScoreRepoErr)
	}

	// Create daily leaderboard repository
	dailyLeaderboardRepo, dailyLeaderboardRepoErr := datastore.NewDailyLeaderboardDatabase(dbConn)
	if dailyLeaderboardRepoErr != nil {
		log.Fatalf("Failed to create daily leaderboard repository: %v", dailyLeaderboardRepoErr)
	}

	// Create application
	app := &api.Application{
		Config:               config,
		UserRepo:             userRepo,
		DailyColorRepo:       dailyColorRepo,
		DailyScoreRepo:       dailyScoreRepo,
		DailyLeaderboardRepo: dailyLeaderboardRepo,
	}

	// Start scheduler for daily color generation
	colorScheduler := scheduler.NewScheduler(dailyColorRepo)
	colorScheduler.Start()

	// Create and start server
	mux := http.NewServeMux()

	fmt.Println("Color Game API Starting...")
	if err := app.Serve(mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolVal
}

func getEnvSlice(key, defaultValue string) []string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}
	return strings.Split(value, ",")
}
