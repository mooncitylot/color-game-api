package api

import (
	"github.com/color-game/api/datastore"
)

type Config struct {
	HTTPPort           string
	DatabaseType       string
	DatabaseUser       string
	DatabasePassword   string
	DatabaseName       string
	SSLMode            string
	JwtSecret          string
	JwtAccessDuration  int // seconds
	JwtRefreshDuration int // seconds
	JwtDomain          string
	AllowedOrigins     []string
	DevMode            bool
}

type Application struct {
	Config               Config
	UserRepo             datastore.UserRepository
	DailyColorRepo       datastore.DailyColorRepository
	DailyScoreRepo       datastore.DailyScoreRepository
	DailyLeaderboardRepo datastore.DailyLeaderboardRepository
}
