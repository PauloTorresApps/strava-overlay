package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"strava-overlay/internal/auth"
	"strava-overlay/internal/gps"
	"strava-overlay/internal/overlay"
	"strava-overlay/internal/strava"
	"strava-overlay/internal/video"

	"github.com/joho/godotenv"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx          context.Context
	stravaAuth   *auth.StravaAuth
	stravaClient *strava.Client
}

// --- ESTRUTURAS PARA O FRONTEND ---
// Criamos versões das nossas structs que usam `string` para datas,
// que é um formato que o Wails e o JavaScript entendem perfeitamente.

type FrontendGPSPoint struct {
	Time     string  `json:"time"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Velocity float64 `json:"velocity"`
	Altitude float64 `json:"altitude"`
	Bearing  float64 `json:"bearing"`
	GForce   float64 `json:"gForce"`
}

type FrontendActivity struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	StartDate   string     `json:"start_date"`
	Distance    float64    `json:"distance"`
	MovingTime  int        `json:"moving_time"`
	MaxSpeed    float64    `json:"max_speed"`
	StartLatLng []float64  `json:"start_latlng"`
	EndLatLng   []float64  `json:"end_latlng"`
	Map         strava.Map `json:"map"`
}

type FrontendActivityDetail struct {
	*FrontendActivity
	Calories      float64 `json:"calories"`
	ElevationGain float64 `json:"total_elevation_gain"`
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

func (a *App) SelectVideoFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Selecione um arquivo de vídeo",
		Filters: []runtime.FileFilter{{DisplayName: "Vídeos (*.mp4, *.mov,*.MP4)", Pattern: "*.mp4;*.mov;*.MP4"}},
	})
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

// GetActivities retrieves user activities and converts them for the frontend
func (a *App) GetActivities() ([]FrontendActivity, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	activities, err := a.stravaClient.GetActivities()
	if err != nil {
		return nil, err
	}

	// Converte a lista de atividades para a versão do frontend
	frontendActivities := make([]FrontendActivity, len(activities))
	for i, act := range activities {
		frontendActivities[i] = FrontendActivity{
			ID:          act.ID,
			Name:        act.Name,
			Type:        act.Type,
			StartDate:   act.StartDate.Format(time.RFC3339), // Converte time.Time para string
			Distance:    act.Distance,
			MovingTime:  act.MovingTime,
			MaxSpeed:    act.MaxSpeed,
			StartLatLng: act.StartLatLng,
			EndLatLng:   act.EndLatLng,
			Map:         act.Map,
		}
	}
	return frontendActivities, nil
}

// GetActivityDetail retrieves detailed activity information
func (a *App) GetActivityDetail(activityID int64) (*strava.ActivityDetail, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	return a.stravaClient.GetActivityDetail(activityID)
}

// GetGPSPointForVideoTime finds the GPS point corresponding to a video's start time
// Substitua a função GetGPSPointForVideoTime em app.go

func (a *App) GetGPSPointForVideoTime(activityID int64, videoPath string) (FrontendGPSPoint, error) {
	if a.stravaClient == nil {
		return FrontendGPSPoint{}, fmt.Errorf("not authenticated")
	}

	videoMeta, err := video.GetVideoMetadata(videoPath)
	if err != nil {
		return FrontendGPSPoint{}, fmt.Errorf("failed to get video metadata: %w", err)
	}

	detail, err := a.stravaClient.GetActivityDetail(activityID)
	if err != nil {
		return FrontendGPSPoint{}, fmt.Errorf("failed to get activity detail: %w", err)
	}

	// Correção de fuso horário
	videoTimeUTC := videoMeta.CreationTime
	tzParts := strings.Split(detail.Timezone, " ")
	ianaTZ := tzParts[len(tzParts)-1]
	location, err := time.LoadLocation(ianaTZ)
	if err != nil {
		log.Printf("Aviso: fuso horário desconhecido '%s', usando UTC. Erro: %v", ianaTZ, err)
		location = time.UTC
	}

	correctedVideoStartTime := time.Date(
		videoTimeUTC.Year(), videoTimeUTC.Month(), videoTimeUTC.Day(),
		videoTimeUTC.Hour(), videoTimeUTC.Minute(), videoTimeUTC.Second(), videoTimeUTC.Nanosecond(),
		location,
	)

	// Debug
	fmt.Printf("=== SINCRONIZAÇÃO GPS-VÍDEO ===\n")
	fmt.Printf("Vídeo (UTC): %s\n", videoTimeUTC.Format("15:04:05"))
	fmt.Printf("Vídeo (corrigido): %s\n", correctedVideoStartTime.Format("15:04:05"))
	fmt.Printf("Atividade: %s\n", detail.StartDate.Format("15:04:05"))

	streams, err := a.stravaClient.GetActivityStreams(activityID)
	if err != nil {
		return FrontendGPSPoint{}, fmt.Errorf("failed to get activity streams: %w", err)
	}

	// Valida streams
	timeStream, timeExists := streams["time"]
	latlngStream, latlngExists := streams["latlng"]
	if !timeExists || !latlngExists || timeStream.Data == nil || latlngStream.Data == nil {
		return FrontendGPSPoint{}, fmt.Errorf("streams GPS ausentes ou vazios")
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
		return FrontendGPSPoint{}, fmt.Errorf("failed to process GPS data: %w", err)
	}

	point, found := processor.GetPointForTime(correctedVideoStartTime)
	if !found {
		return FrontendGPSPoint{}, fmt.Errorf("no matching GPS point found")
	}

	// Validação final das coordenadas
	if point.Lat == 0 && point.Lng == 0 {
		return FrontendGPSPoint{}, fmt.Errorf("coordenadas GPS inválidas (0,0)")
	}
	if point.Lat < -90 || point.Lat > 90 || point.Lng < -180 || point.Lng > 180 {
		return FrontendGPSPoint{}, fmt.Errorf("coordenadas GPS fora dos limites válidos")
	}

	fmt.Printf("Ponto encontrado: %.6f, %.6f em %s\n",
		point.Lat, point.Lng, point.Time.Format("15:04:05"))
	fmt.Printf("===========================\n")

	return FrontendGPSPoint{
		Time:     point.Time.Format(time.RFC3339),
		Lat:      point.Lat,
		Lng:      point.Lng,
		Velocity: point.Velocity,
		Altitude: point.Altitude,
		Bearing:  point.Bearing,
		GForce:   point.GForce,
	}, nil
}

// Substitua também esta função em app.go
func (a *App) ProcessVideoOverlay(activityID int64, videoPath string) (string, error) {
	if a.stravaClient == nil {
		return "", fmt.Errorf("not authenticated")
	}
	videoMeta, err := video.GetVideoMetadata(videoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get video metadata: %w", err)
	}
	detail, err := a.stravaClient.GetActivityDetail(activityID)
	if err != nil {
		return "", fmt.Errorf("failed to get activity detail: %w", err)
	}

	// --- LÓGICA DE CORREÇÃO DE FUSO HORÁRIO ---
	videoTimeUTC := videoMeta.CreationTime
	tzParts := strings.Split(detail.Timezone, " ")
	ianaTZ := tzParts[len(tzParts)-1]
	location, err := time.LoadLocation(ianaTZ)
	if err != nil {
		log.Printf("Aviso: fuso horário desconhecido '%s', usando UTC como padrão. Erro: %v", ianaTZ, err)
		location = time.UTC
	}
	correctedVideoStartTime := time.Date(
		videoTimeUTC.Year(), videoTimeUTC.Month(), videoTimeUTC.Day(),
		videoTimeUTC.Hour(), videoTimeUTC.Minute(), videoTimeUTC.Second(), videoTimeUTC.Nanosecond(),
		location,
	)
	// --- FIM DA LÓGICA DE CORREÇÃO ---

	streams, err := a.stravaClient.GetActivityStreams(activityID)
	if err != nil {
		return "", fmt.Errorf("failed to get activity streams: %w", err)
	}
	processor := gps.NewGPSProcessor()
	err = processor.ProcessStreamData(
		streams["time"].Data.([]interface{}), streams["latlng"].Data.([]interface{}),
		streams["velocity_smooth"].Data.([]interface{}), streams["altitude"].Data.([]interface{}),
		detail.StartDate,
	)
	if err != nil {
		return "", fmt.Errorf("failed to process GPS data: %w", err)
	}

	correctedVideoEndTime := correctedVideoStartTime.Add(videoMeta.Duration)
	gpsPoints := processor.GetPointsForTimeRange(correctedVideoStartTime, correctedVideoEndTime) // Usa a hora corrigida
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
