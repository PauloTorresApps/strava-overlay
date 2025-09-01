package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"strava-overlay/internal/auth"
	"strava-overlay/internal/gps"
	"strava-overlay/internal/overlay"
	"strava-overlay/internal/strava"
	"strava-overlay/internal/video"
)

// App struct
type App struct {
	ctx          context.Context
	stravaAuth   *auth.StravaAuth
	stravaClient *strava.Client
}

// NewApp creates a new App application struct
func NewApp() *App {
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found: %v", err)
	}

	clientID := os.Getenv("STRAVA_CLIENT_ID")
	clientSecret := os.Getenv("STRAVA_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		log.Fatal("STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET must be set")
	}

	return &App{
		stravaAuth: auth.NewStravaAuth(clientID, clientSecret),
	}
}

// Startup is called when the app starts up
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// AuthenticateStrava handles Strava authentication
func (a *App) AuthenticateStrava() error {
	token, err := a.stravaAuth.GetValidToken(a.ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	a.stravaClient = strava.NewClient(token)
	return nil
}

// GetActivities retrieves user activities
func (a *App) GetActivities() ([]strava.Activity, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	return a.stravaClient.GetActivities(20)
}

// GetActivityDetail retrieves detailed activity information
func (a *App) GetActivityDetail(activityID int64) (*strava.ActivityDetail, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	return a.stravaClient.GetActivityDetail(activityID)
}

// ProcessVideoOverlay processes video with GPS overlay
func (a *App) ProcessVideoOverlay(activityID int64, videoPath string) (string, error) {
	if a.stravaClient == nil {
		return "", fmt.Errorf("not authenticated")
	}

	videoMeta, err := video.GetVideoMetadata(videoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get video metadata: %w", err)
	}

	streams, err := a.stravaClient.GetActivityStreams(activityID)
	if err != nil {
		return "", fmt.Errorf("failed to get activity streams: %w", err)
	}

	detail, err := a.stravaClient.GetActivityDetail(activityID)
	if err != nil {
		return "", fmt.Errorf("failed to get activity detail: %w", err)
	}

	processor := gps.NewGPSProcessor()
	err = processor.ProcessStreamData(
		streams["time"].Data.([]interface{}),
		streams["latlng"].Data.([]interface{}),
		streams["velocity_smooth"].Data.([]interface{}),
		streams["altitude"].Data.([]interface{}),
		detail.StartDate,
	)
	if err != nil {
		return "", fmt.Errorf("failed to process GPS data: %w", err)
	}

	videoEndTime := videoMeta.CreationTime.Add(videoMeta.Duration)
	gpsPoints := processor.GetPointsForTimeRange(videoMeta.CreationTime, videoEndTime)

	if len(gpsPoints) == 0 {
		return "", fmt.Errorf("no GPS data found for video time range")
	}

	overlayGen := overlay.NewGenerator()
	defer overlayGen.Cleanup()

	overlayImages, err := overlayGen.GenerateOverlaySequence(gpsPoints, videoMeta.FrameRate)
	if err != nil {
		return "", fmt.Errorf("failed to generate overlays: %w", err)
	}

	homeDir, _ := os.UserHomeDir()
	outputDir := filepath.Join(homeDir, "Strava Add Overlay")
	os.MkdirAll(outputDir, 0755)

	outputPath := filepath.Join(outputDir, fmt.Sprintf("activity_%d_overlay.mp4", activityID))

	videoProcessor := video.NewProcessor()
	err = videoProcessor.ApplyOverlays(videoPath, overlayImages, outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to apply overlays: %w", err)
	}

	return outputPath, nil
}