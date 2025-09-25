package main

import (
	"context"
	"log"

	"strava-overlay/internal/auth"
	"strava-overlay/internal/config"
	"strava-overlay/internal/handlers"
	"strava-overlay/internal/services"
	"strava-overlay/internal/strava"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct - Estrutura principal simplificada
type App struct {
	ctx          context.Context
	stravaAuth   *auth.StravaAuth
	stravaClient *strava.Client

	// Handlers - Responsáveis pela lógica de negócio específica
	authHandler     *handlers.AuthHandler
	activityHandler *handlers.ActivityHandler
	videoHandler    *handlers.VideoHandler
	gpsHandler      *handlers.GPSHandler
	configHandler   *handlers.ConfigHandler // NOVO: Handler de configuração

	// Services - Responsáveis por operações complexas
	videoService *services.VideoService
	gpsService   *services.GPSService
}

// NewApp creates a new App application struct
func NewApp() *App {
	clientID := config.STRAVA_CLIENT_ID
	clientSecret := config.STRAVA_CLIENT_SECRET

	if clientID == "" || clientSecret == "" {
		log.Fatal("STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET must be set in internal/config/credentials.go")
	}

	stravaAuth := auth.NewStravaAuth(clientID, clientSecret)

	// Inicializa os services
	videoService := services.NewVideoService()
	gpsService := services.NewGPSService()

	app := &App{
		stravaAuth:   stravaAuth,
		videoService: videoService,
		gpsService:   gpsService,
	}

	// Inicializa os handlers passando as dependências necessárias
	app.authHandler = handlers.NewAuthHandler(stravaAuth, app.setStravaClient)
	app.activityHandler = handlers.NewActivityHandler(app.getStravaClient)
	app.videoHandler = handlers.NewVideoHandler(app.getStravaClient, videoService, gpsService)
	app.gpsHandler = handlers.NewGPSHandler(app.getStravaClient, gpsService)
	app.configHandler = handlers.NewConfigHandler() // NOVO: Inicializa config handler

	return app
}

// Startup is called when the app starts up
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Helper methods para gerenciar o cliente Strava
func (a *App) setStravaClient(client *strava.Client) {
	a.stravaClient = client
}

func (a *App) getStravaClient() *strava.Client {
	return a.stravaClient
}

// Seletor de arquivos - permanece aqui por ser específico do Wails
func (a *App) SelectVideoFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Selecione um arquivo de vídeo",
		Filters: []runtime.FileFilter{{DisplayName: "Vídeos (*.mp4, *.mov,*.MP4)", Pattern: "*.mp4;*.mov;*.MP4"}},
	})
}

// === NOVOS MÉTODOS DE CONFIGURAÇÃO ===

// GetFrontendConfig retorna configurações seguras para o frontend
func (a *App) GetFrontendConfig() *handlers.FrontendConfig {
	return a.configHandler.GetFrontendConfig()
}

// GetSecureAPIKeys retorna chaves de API seguras para o frontend
func (a *App) GetSecureAPIKeys() map[string]string {
	return a.configHandler.GetSecureAPIKeys()
}

// GetMapProviderConfig retorna configuração de provedores de mapa
func (a *App) GetMapProviderConfig() map[string]interface{} {
	return a.configHandler.GetMapProviderConfig()
}

// === MÉTODOS QUE DELEGAM PARA OS HANDLERS ===

// Auth methods
func (a *App) CheckAuthenticationStatus() handlers.AuthStatus {
	return a.authHandler.CheckAuthenticationStatus(a.ctx)
}

func (a *App) AuthenticateStrava() error {
	return a.authHandler.AuthenticateStrava(a.ctx)
}

// Activity methods
func (a *App) GetActivitiesPage(page int) (*handlers.PaginatedActivities, error) {
	return a.activityHandler.GetActivitiesPage(page)
}

func (a *App) GetActivities() ([]handlers.FrontendActivity, error) {
	return a.activityHandler.GetActivities()
}

func (a *App) GetActivityDetail(activityID int64) (*strava.ActivityDetail, error) {
	return a.activityHandler.GetActivityDetail(activityID)
}

// GPS methods
func (a *App) GetGPSPointForVideoTime(activityID int64, videoPath string) (handlers.FrontendGPSPoint, error) {
	return a.gpsHandler.GetGPSPointForVideoTime(activityID, videoPath)
}

func (a *App) GetGPSPointForMapClick(activityID int64, lat, lng float64) (handlers.FrontendGPSPoint, error) {
	return a.gpsHandler.GetGPSPointForMapClick(activityID, lat, lng)
}

func (a *App) GetAllGPSPoints(activityID int64) ([]handlers.FrontendGPSPoint, error) {
	return a.gpsHandler.GetAllGPSPoints(activityID)
}

func (a *App) GetFullGPSTrajectory(activityID int64) ([]handlers.FrontendGPSPoint, error) {
	return a.gpsHandler.GetFullGPSTrajectory(activityID)
}

func (a *App) GetGPSPointsWithDensity(activityID int64, density string) ([]handlers.FrontendGPSPoint, error) {
	return a.gpsHandler.GetGPSPointsWithDensity(activityID, density)
}

// Video methods
func (a *App) ProcessVideoOverlay(activityID int64, videoPath string, manualStartTimeStr string, overlayPosition string) (string, error) {
	return a.videoHandler.ProcessVideoOverlay(activityID, videoPath, manualStartTimeStr, overlayPosition)
}
