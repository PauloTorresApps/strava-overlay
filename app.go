package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	goruntime "runtime"
	"sync"

	"strava-overlay/internal/auth"
	"strava-overlay/internal/config"
	"strava-overlay/internal/handlers"
	"strava-overlay/internal/services"
	"strava-overlay/internal/strava"

	"github.com/gen2brain/beeep"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx          context.Context
	stravaAuth   *auth.StravaAuth
	stravaClient *strava.Client

	authHandler     *handlers.AuthHandler
	activityHandler *handlers.ActivityHandler
	videoHandler    *handlers.VideoHandler
	gpsHandler      *handlers.GPSHandler
	configHandler   *handlers.ConfigHandler

	videoService *services.VideoService
	gpsService   *services.GPSService

	processingCancel context.CancelFunc
	processingMutex  sync.Mutex
}

func NewApp() *App {
	clientID := config.AppConfig.StravaClientID
	clientSecret := config.AppConfig.StravaClientSecret

	if clientID == "" || clientSecret == "" {
		log.Fatal("STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET must be set")
	}

	stravaAuth := auth.NewStravaAuth(clientID, clientSecret)

	videoService := services.NewVideoService()
	gpsService := services.NewGPSService()

	app := &App{
		stravaAuth:   stravaAuth,
		videoService: videoService,
		gpsService:   gpsService,
	}

	app.authHandler = handlers.NewAuthHandler(stravaAuth, app.setStravaClient)
	app.activityHandler = handlers.NewActivityHandler(app.getStravaClient)
	app.videoHandler = handlers.NewVideoHandler(app.getStravaClient, videoService, gpsService)
	app.gpsHandler = handlers.NewGPSHandler(app.getStravaClient, gpsService)
	app.configHandler = handlers.NewConfigHandler()

	return app
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Define callback de progresso para o VideoService
	a.videoService.SetProgressCallback(func(stage string, progress float64, message string) {
		runtime.EventsEmit(ctx, "video:progress", map[string]interface{}{
			"stage":    stage,
			"progress": progress,
			"message":  message,
		})
	})

	a.videoService.SetCompletionCallback(func(success bool, outputPath string, err error) {
		if success {
			runtime.EventsEmit(ctx, "video:completed", map[string]interface{}{
				"success":    true,
				"outputPath": outputPath,
			})
		} else {
			runtime.EventsEmit(ctx, "video:completed", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}
	})
}

func (a *App) ProcessVideoOverlay(activityID int64, videoPath string, manualStartTimeStr string, overlayPosition string) (string, error) {
	a.processingMutex.Lock()

	// Cria contexto cancelável
	ctx, cancel := context.WithCancel(a.ctx)
	a.processingCancel = cancel
	a.processingMutex.Unlock()

	defer func() {
		a.processingMutex.Lock()
		a.processingCancel = nil
		a.processingMutex.Unlock()
	}()

	client := a.getStravaClient()
	if client == nil {
		return "", fmt.Errorf("not authenticated")
	}

	return a.videoService.ProcessVideoWithOverlay(
		ctx,
		client,
		activityID,
		videoPath,
		manualStartTimeStr,
		overlayPosition,
		a.gpsService,
	)
}

func (a *App) CancelVideoProcessing() error {
	a.processingMutex.Lock()
	defer a.processingMutex.Unlock()

	if a.processingCancel == nil {
		return fmt.Errorf("nenhum processamento em andamento")
	}

	log.Println("🛑 Cancelando processamento de vídeo...")
	a.processingCancel()
	return nil
}

func (a *App) setStravaClient(client *strava.Client) {
	a.stravaClient = client
}

func (a *App) getStravaClient() *strava.Client {
	return a.stravaClient
}

func (a *App) SelectVideoFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Selecione um arquivo de vídeo",
		Filters: []runtime.FileFilter{{DisplayName: "Vídeos (*.mp4, *.mov,*.MP4)", Pattern: "*.mp4;*.mov;*.MP4"}},
	})
}

// === Métodos de configuração ===
func (a *App) GetFrontendConfig() *handlers.FrontendConfig {
	return a.configHandler.GetFrontendConfig()
}

func (a *App) GetSecureAPIKeys() map[string]string {
	return a.configHandler.GetSecureAPIKeys()
}

func (a *App) GetMapProviderConfig() map[string]interface{} {
	return a.configHandler.GetMapProviderConfig()
}

// === Métodos delegados ===
func (a *App) CheckAuthenticationStatus() handlers.AuthStatus {
	return a.authHandler.CheckAuthenticationStatus(a.ctx)
}

func (a *App) AuthenticateStrava() error {
	return a.authHandler.AuthenticateStrava(a.ctx)
}

func (a *App) GetActivitiesPage(page int) (*handlers.PaginatedActivities, error) {
	return a.activityHandler.GetActivitiesPage(page)
}

func (a *App) GetActivities() ([]handlers.FrontendActivity, error) {
	return a.activityHandler.GetActivities()
}

func (a *App) GetActivityDetail(activityID int64) (*strava.ActivityDetail, error) {
	return a.activityHandler.GetActivityDetail(activityID)
}

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

// SendDesktopNotification envia notificação nativa
func (a *App) SendDesktopNotification(title, body string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   title,
		Message: body,
	})
}

// SendNotification envia notificação nativa do sistema operacional
func (a *App) SendNotification(title, message string) {
	go func() {
		var cmd *exec.Cmd

		switch goruntime.GOOS { // Usa goruntime
		case "linux":
			cmd = exec.Command("notify-send", "-u", "normal", "-t", "10000", title, message)
		case "darwin":
			script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
			cmd = exec.Command("osascript", "-e", script)
		case "windows":
			err := beeep.Notify(title, message, "")
			if err != nil {
				log.Printf("⚠️ Erro notificação Windows: %v", err)
			}
			return
		default:
			log.Printf("⚠️ SO não suportado para notificações: %s", goruntime.GOOS)
			return
		}

		if cmd != nil {
			err := cmd.Run()
			if err != nil {
				log.Printf("⚠️ Erro ao enviar notificação: %v", err)
			}
		}
	}()
}
