package scheduler

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/color-game/api/datastore"
	"github.com/color-game/api/models"
)

type Scheduler struct {
	DailyColorRepo datastore.DailyColorRepository
	ticker         *time.Ticker
	done           chan bool
}

func NewScheduler(repo datastore.DailyColorRepository) *Scheduler {
	return &Scheduler{
		DailyColorRepo: repo,
		done:           make(chan bool),
	}
}

// Start begins the scheduler to run at midnight every day
func (s *Scheduler) Start() {
	// Calculate time until next midnight
	now := time.Now()
	nextMidnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	durationUntilMidnight := nextMidnight.Sub(now)

	log.Printf("Scheduler started. Next daily color generation in %v", durationUntilMidnight)

	// Wait until midnight, then generate first color
	time.AfterFunc(durationUntilMidnight, func() {
		s.GenerateDailyColor()

		// After first run, schedule to run every 24 hours
		s.ticker = time.NewTicker(24 * time.Hour)
		go func() {
			for {
				select {
				case <-s.ticker.C:
					s.GenerateDailyColor()
				case <-s.done:
					return
				}
			}
		}()
	})
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	log.Println("Scheduler stopped")
}

// GenerateDailyColor generates and saves a new daily color
func (s *Scheduler) GenerateDailyColor() error {
	log.Println("Generating daily color...")

	// Check if today's color already exists
	today := time.Now()
	normalizedToday := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())

	existingColor, err := s.DailyColorRepo.GetByDate(normalizedToday)
	if err == nil && existingColor.ID != 0 {
		log.Printf("Daily color already exists for %s: %s", normalizedToday.Format("2006-01-02"), existingColor.ColorName)
		return nil
	}

	// Generate random RGB values
	r := rand.Intn(256)
	g := rand.Intn(256)
	b := rand.Intn(256)

	// Build the URL for thecolorapi.com
	url := fmt.Sprintf("https://www.thecolorapi.com/scheme?rgb=%d,%d,%d&mode=analogic&count=6&format=json", r, g, b)

	// Make HTTP request to the color API
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching color from API: %v", err)
		return err
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("color API returned status: %d", resp.StatusCode)
		log.Printf("Error: %v", err)
		return err
	}

	// Parse the response
	var colorResponse models.ColorAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&colorResponse); err != nil {
		log.Printf("Error parsing color API response: %v", err)
		return err
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
	savedColor, err := s.DailyColorRepo.Create(dailyColor)
	if err != nil {
		log.Printf("Error saving daily color to database: %v", err)
		return err
	}

	log.Printf("Successfully generated daily color: %s (RGB: %d,%d,%d) for %s",
		savedColor.ColorName, savedColor.R, savedColor.G, savedColor.B,
		savedColor.Date.Format("2006-01-02"))

	return nil
}
