package main

import (
	"context"
	"fmt"
	"log"
	"math"
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

// AuthStatus representa o status da autenticação
type AuthStatus struct {
	IsAuthenticated bool   `json:"is_authenticated"`
	Message         string `json:"message"`
	Error           string `json:"error,omitempty"`
}

// --- ESTRUTURAS PARA O FRONTEND ---
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
	HasGPS      bool       `json:"has_gps"` // Nova propriedade para indicar se tem GPS
}

type FrontendActivityDetail struct {
	*FrontendActivity
	Calories      float64 `json:"calories"`
	ElevationGain float64 `json:"total_elevation_gain"`
}

// NOVA ESTRUTURA: Resposta paginada de atividades
type PaginatedActivities struct {
	Activities  []FrontendActivity `json:"activities"`
	Page        int                `json:"page"`
	PerPage     int                `json:"per_page"`
	HasMore     bool               `json:"has_more"`
	TotalLoaded int                `json:"total_loaded"`
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

// CheckAuthenticationStatus verifica automaticamente se há um token válido
func (a *App) CheckAuthenticationStatus() AuthStatus {
	log.Printf("🔍 Verificando status de autenticação...")

	token, err := a.stravaAuth.GetValidToken(a.ctx)
	if err != nil {
		log.Printf("❌ Falha na verificação de token: %v", err)
		return AuthStatus{
			IsAuthenticated: false,
			Message:         "Autenticação necessária",
			Error:           err.Error(),
		}
	}

	// Se chegou até aqui, o token é válido
	a.stravaClient = strava.NewClient(token)
	log.Printf("✅ Token válido encontrado - Cliente Strava inicializado")

	return AuthStatus{
		IsAuthenticated: true,
		Message:         "Conectado automaticamente ao Strava",
	}
}

func (a *App) SelectVideoFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Selecione um arquivo de vídeo",
		Filters: []runtime.FileFilter{{DisplayName: "Vídeos (*.mp4, *.mov,*.MP4)", Pattern: "*.mp4;*.mov;*.MP4"}},
	})
}

// AuthenticateStrava handles Strava authentication (mantida para autenticação manual)
func (a *App) AuthenticateStrava() error {
	log.Printf("🔐 Iniciando autenticação manual do Strava...")

	token, err := a.stravaAuth.GetValidToken(a.ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	a.stravaClient = strava.NewClient(token)

	log.Printf("✅ Autenticação manual concluída com sucesso")
	return nil
}

// NOVA FUNÇÃO: GetActivitiesPage retorna uma página de atividades
func (a *App) GetActivitiesPage(page int) (*PaginatedActivities, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	perPage := 30 // Máximo permitido pelo Strava
	log.Printf("📋 Carregando página %d de atividades (até %d itens)...", page, perPage)

	activities, err := a.stravaClient.GetActivitiesPage(page, perPage)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar atividades: %w", err)
	}

	// Converte para o formato do frontend
	frontendActivities := make([]FrontendActivity, len(activities))
	gpsCount := 0

	for i, act := range activities {
		hasGPS := act.Map.SummaryPolyline != ""
		if hasGPS {
			gpsCount++
		}

		frontendActivities[i] = FrontendActivity{
			ID:          act.ID,
			Name:        act.Name,
			Type:        act.Type,
			StartDate:   act.StartDate.Format(time.RFC3339),
			Distance:    act.Distance,
			MovingTime:  act.MovingTime,
			MaxSpeed:    act.MaxSpeed,
			StartLatLng: act.StartLatLng,
			EndLatLng:   act.EndLatLng,
			Map:         act.Map,
			HasGPS:      hasGPS,
		}
	}

	// Determina se há mais páginas (se retornou o máximo de itens, provavelmente há mais)
	hasMore := len(activities) == perPage
	totalLoaded := (page-1)*perPage + len(activities)

	log.Printf("✅ Página %d carregada: %d atividades (%d com GPS)", page, len(activities), gpsCount)

	return &PaginatedActivities{
		Activities:  frontendActivities,
		Page:        page,
		PerPage:     perPage,
		HasMore:     hasMore,
		TotalLoaded: totalLoaded,
	}, nil
}

// GetActivities - MANTIDA PARA COMPATIBILIDADE, mas recomenda-se usar GetActivitiesPage
func (a *App) GetActivities() ([]FrontendActivity, error) {
	result, err := a.GetActivitiesPage(1)
	if err != nil {
		return nil, err
	}
	return result.Activities, nil
}

// GetActivityDetail retrieves detailed activity information
func (a *App) GetActivityDetail(activityID int64) (*strava.ActivityDetail, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	return a.stravaClient.GetActivityDetail(activityID)
}

// GetGPSPointForVideoTime finds the GPS point corresponding to a video's start time
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

// GetGPSPointForMapClick encontra o ponto GPS mais próximo de um clique no mapa.
func (a *App) GetGPSPointForMapClick(activityID int64, lat, lng float64) (FrontendGPSPoint, error) {
	if a.stravaClient == nil {
		return FrontendGPSPoint{}, fmt.Errorf("not authenticated")
	}

	detail, err := a.stravaClient.GetActivityDetail(activityID)
	if err != nil {
		return FrontendGPSPoint{}, fmt.Errorf("failed to get activity detail: %w", err)
	}
	streams, err := a.stravaClient.GetActivityStreams(activityID)
	if err != nil {
		return FrontendGPSPoint{}, fmt.Errorf("failed to get activity streams: %w", err)
	}

	// Re-processa os streams para garantir que o processador esteja populado
	processor := gps.NewGPSProcessor()
	err = processor.ProcessStreamData(
		streams["time"].Data.([]interface{}),
		streams["latlng"].Data.([]interface{}),
		streams["velocity_smooth"].Data.([]interface{}),
		streams["altitude"].Data.([]interface{}),
		detail.StartDate,
	)
	if err != nil {
		return FrontendGPSPoint{}, fmt.Errorf("failed to process GPS data for map click: %w", err)
	}

	point, found := processor.GetPointForCoords(lat, lng)
	if !found {
		return FrontendGPSPoint{}, fmt.Errorf("no matching GPS point found for coordinates")
	}

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

// ProcessVideoOverlay aplica o overlay ao vídeo
func (a *App) ProcessVideoOverlay(activityID int64, videoPath string, manualStartTimeStr string) (string, error) {
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

	// --- LÓGICA DE SINCRONIZAÇÃO DE TEMPO ---
	var correctedVideoStartTime time.Time

	if manualStartTimeStr != "" {
		// Usa o tempo manual se fornecido
		parsedTime, err := time.Parse(time.RFC3339, manualStartTimeStr)
		if err != nil {
			return "", fmt.Errorf("failed to parse manual start time: %w", err)
		}
		correctedVideoStartTime = parsedTime
		fmt.Printf("DEBUG: Usando tempo de início manual: %s\n", correctedVideoStartTime.Format("15:04:05"))
	} else {
		// Lógica de fallback (automática)
		videoTimeUTC := videoMeta.CreationTime
		tzParts := strings.Split(detail.Timezone, " ")
		ianaTZ := tzParts[len(tzParts)-1]
		location, err := time.LoadLocation(ianaTZ)
		if err != nil {
			log.Printf("Aviso: fuso horário desconhecido '%s', usando UTC como padrão. Erro: %v", ianaTZ, err)
			location = time.UTC
		}
		correctedVideoStartTime = time.Date(
			videoTimeUTC.Year(), videoTimeUTC.Month(), videoTimeUTC.Day(),
			videoTimeUTC.Hour(), videoTimeUTC.Minute(), videoTimeUTC.Second(), videoTimeUTC.Nanosecond(),
			location,
		)
		fmt.Printf("DEBUG: Usando tempo de início automático: %s\n", correctedVideoStartTime.Format("15:04:05"))
	}
	// --- FIM DA LÓGICA DE SINCRONIZAÇÃO ---

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
	gpsPoints := processor.GetPointsForTimeRange(correctedVideoStartTime, correctedVideoEndTime)
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

// GetAllGPSPoints retorna pontos GPS selecionados inteligentemente para marcadores
func (a *App) GetAllGPSPoints(activityID int64) ([]FrontendGPSPoint, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	detail, err := a.stravaClient.GetActivityDetail(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity detail: %w", err)
	}

	streams, err := a.stravaClient.GetActivityStreams(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity streams: %w", err)
	}

	// Valida streams
	timeStream, timeExists := streams["time"]
	latlngStream, latlngExists := streams["latlng"]
	if !timeExists || !latlngExists || timeStream.Data == nil || latlngStream.Data == nil {
		return nil, fmt.Errorf("streams GPS ausentes ou vazios")
	}

	processor := gps.NewGPSProcessor()
	err = processor.ProcessStreamData(
		timeStream.Data.([]interface{}),
		latlngStream.Data.([]interface{}),
		streams["velocity_smooth"].Data.([]interface{}),
		streams["altitude"].Data.([]interface{}),
		detail.StartDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process GPS data: %w", err)
	}

	// Pega TODOS os pontos interpolados
	allPoints := processor.GetAllPoints()

	// NOVA LÓGICA DE FILTRAGEM INTELIGENTE
	selectedPoints := a.selectIntelligentGPSPoints(allPoints)

	// Converte para formato frontend
	var frontendPoints []FrontendGPSPoint
	for _, point := range selectedPoints {
		frontendPoints = append(frontendPoints, FrontendGPSPoint{
			Time:     point.Time.Format(time.RFC3339),
			Lat:      point.Lat,
			Lng:      point.Lng,
			Velocity: point.Velocity,
			Altitude: point.Altitude,
			Bearing:  point.Bearing,
			GForce:   point.GForce,
		})
	}

	fmt.Printf("DEBUG: Selecionados %d pontos GPS inteligentes de %d pontos interpolados totais\n", len(selectedPoints), len(allPoints))
	return frontendPoints, nil
}

// selectIntelligentGPSPoints - Seleção inteligente de pontos GPS para marcadores
func (a *App) selectIntelligentGPSPoints(allPoints []gps.GPSPoint) []gps.GPSPoint {
	if len(allPoints) == 0 {
		return allPoints
	}

	var selectedPoints []gps.GPSPoint

	// Sempre inclui o primeiro ponto
	selectedPoints = append(selectedPoints, allPoints[0])

	// Parâmetros da seleção inteligente
	minTimeInterval := 30 * time.Second       // Mínimo 30 segundos entre pontos
	maxTimeInterval := 120 * time.Second      // Máximo 2 minutos sem ponto
	minSpeedChange := 5.0 * (1000.0 / 3600.0) // 5 km/h em m/s
	minDistanceChange := 100.0                // 100 metros

	lastSelectedTime := allPoints[0].Time
	lastSelectedSpeed := allPoints[0].Velocity
	lastSelectedPoint := allPoints[0]

	for i := 1; i < len(allPoints)-1; i++ {
		currentPoint := allPoints[i]
		timeSinceLastSelected := currentPoint.Time.Sub(lastSelectedTime)
		speedChange := math.Abs(currentPoint.Velocity - lastSelectedSpeed)

		// Calcula distância desde o último ponto selecionado
		distance := a.calculateDistance(lastSelectedPoint, currentPoint)

		// Critérios para seleção:
		shouldSelect := false

		// 1. Intervalo de tempo obrigatório (máximo)
		if timeSinceLastSelected >= maxTimeInterval {
			shouldSelect = true
		}

		// 2. Mudança significativa de velocidade (após tempo mínimo)
		if timeSinceLastSelected >= minTimeInterval && speedChange >= minSpeedChange {
			shouldSelect = true
		}

		// 3. Distância significativa percorrida
		if timeSinceLastSelected >= minTimeInterval && distance >= minDistanceChange {
			shouldSelect = true
		}

		// 4. Pontos de parada (velocidade muito baixa)
		if currentPoint.Velocity < 1.0 && lastSelectedSpeed > 3.0 {
			shouldSelect = true
		}

		// 5. Retomada de movimento (após parada)
		if currentPoint.Velocity > 3.0 && lastSelectedSpeed < 1.0 {
			shouldSelect = true
		}

		// 6. Picos de velocidade
		if currentPoint.Velocity > 15.0 && // > 54 km/h
			timeSinceLastSelected >= minTimeInterval &&
			speedChange >= minSpeedChange {
			shouldSelect = true
		}

		if shouldSelect {
			selectedPoints = append(selectedPoints, currentPoint)
			lastSelectedTime = currentPoint.Time
			lastSelectedSpeed = currentPoint.Velocity
			lastSelectedPoint = currentPoint
		}
	}

	// Sempre inclui o último ponto
	if len(allPoints) > 1 {
		lastPoint := allPoints[len(allPoints)-1]
		// Só adiciona se não for muito próximo do penúltimo selecionado
		if len(selectedPoints) == 0 || lastPoint.Time.Sub(selectedPoints[len(selectedPoints)-1].Time) > 10*time.Second {
			selectedPoints = append(selectedPoints, lastPoint)
		}
	}

	fmt.Printf("DEBUG: Seleção inteligente - critérios aplicados:\n")
	fmt.Printf("  - Intervalo mínimo: %v\n", minTimeInterval)
	fmt.Printf("  - Intervalo máximo: %v\n", maxTimeInterval)
	fmt.Printf("  - Mudança mín. velocidade: %.1f km/h\n", minSpeedChange*3.6)
	fmt.Printf("  - Distância mínima: %.0f m\n", minDistanceChange)

	return selectedPoints
}

// GetFullGPSTrajectory retorna TODOS os pontos GPS interpolados para desenhar o trajeto completo
func (a *App) GetFullGPSTrajectory(activityID int64) ([]FrontendGPSPoint, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	detail, err := a.stravaClient.GetActivityDetail(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity detail: %w", err)
	}

	streams, err := a.stravaClient.GetActivityStreams(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity streams: %w", err)
	}

	// Valida streams
	timeStream, timeExists := streams["time"]
	latlngStream, latlngExists := streams["latlng"]
	if !timeExists || !latlngExists || timeStream.Data == nil || latlngStream.Data == nil {
		return nil, fmt.Errorf("streams GPS ausentes ou vazios")
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
		return nil, fmt.Errorf("failed to process GPS data: %w", err)
	}

	// DIFERENÇA: Pega TODOS os pontos, sem filtragem
	allPoints := processor.GetAllPoints()

	// Converte todos os pontos para o formato frontend
	var fullTrajectory []FrontendGPSPoint
	for _, point := range allPoints {
		fullTrajectory = append(fullTrajectory, FrontendGPSPoint{
			Time:     point.Time.Format(time.RFC3339),
			Lat:      point.Lat,
			Lng:      point.Lng,
			Velocity: point.Velocity,
			Altitude: point.Altitude,
			Bearing:  point.Bearing,
			GForce:   point.GForce,
		})
	}

	fmt.Printf("DEBUG: Retornando trajeto COMPLETO com %d pontos GPS interpolados\n", len(fullTrajectory))
	return fullTrajectory, nil
}

// calculateDistance calcula distância entre dois pontos GPS
func (a *App) calculateDistance(p1, p2 gps.GPSPoint) float64 {
	const R = 6371000 // Raio da Terra em metros

	lat1Rad := p1.Lat * math.Pi / 180
	lon1Rad := p1.Lng * math.Pi / 180
	lat2Rad := p2.Lat * math.Pi / 180
	lon2Rad := p2.Lng * math.Pi / 180

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	haversineA := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(haversineA), math.Sqrt(1-haversineA))

	return R * c
}

// GetGPSPointsWithDensity - Versão com densidade customizável
func (a *App) GetGPSPointsWithDensity(activityID int64, density string) ([]FrontendGPSPoint, error) {
	if a.stravaClient == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	detail, err := a.stravaClient.GetActivityDetail(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity detail: %w", err)
	}

	streams, err := a.stravaClient.GetActivityStreams(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity streams: %w", err)
	}

	// Valida streams
	timeStream, timeExists := streams["time"]
	latlngStream, latlngExists := streams["latlng"]
	if !timeExists || !latlngExists || timeStream.Data == nil || latlngStream.Data == nil {
		return nil, fmt.Errorf("streams GPS ausentes ou vazios")
	}

	processor := gps.NewGPSProcessor()
	err = processor.ProcessStreamData(
		timeStream.Data.([]interface{}),
		latlngStream.Data.([]interface{}),
		streams["velocity_smooth"].Data.([]interface{}),
		streams["altitude"].Data.([]interface{}),
		detail.StartDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process GPS data: %w", err)
	}

	allPoints := processor.GetAllPoints()
	var selectedPoints []gps.GPSPoint

	// Seleção baseada na densidade escolhida
	switch density {
	case "high":
		// A cada 15 segundos
		selectedPoints = a.selectPointsByInterval(allPoints, 15*time.Second)
	case "medium":
		// A cada 30 segundos (padrão inteligente)
		selectedPoints = a.selectIntelligentGPSPoints(allPoints)
	case "low":
		// A cada 60 segundos
		selectedPoints = a.selectPointsByInterval(allPoints, 60*time.Second)
	case "ultra_high":
		// A cada 5 segundos (para debug/sincronização precisa)
		selectedPoints = a.selectPointsByInterval(allPoints, 5*time.Second)
	default:
		selectedPoints = a.selectIntelligentGPSPoints(allPoints)
	}

	// Converte para formato frontend
	var frontendPoints []FrontendGPSPoint
	for _, point := range selectedPoints {
		frontendPoints = append(frontendPoints, FrontendGPSPoint{
			Time:     point.Time.Format(time.RFC3339),
			Lat:      point.Lat,
			Lng:      point.Lng,
			Velocity: point.Velocity,
			Altitude: point.Altitude,
			Bearing:  point.Bearing,
			GForce:   point.GForce,
		})
	}

	fmt.Printf("DEBUG: Densidade '%s' - %d pontos selecionados de %d totais\n", density, len(selectedPoints), len(allPoints))
	return frontendPoints, nil
}

// selectPointsByInterval - Seleção por intervalo fixo
func (a *App) selectPointsByInterval(allPoints []gps.GPSPoint, interval time.Duration) []gps.GPSPoint {
	if len(allPoints) == 0 {
		return allPoints
	}

	var selectedPoints []gps.GPSPoint
	selectedPoints = append(selectedPoints, allPoints[0]) // Primeiro ponto

	lastSelectedTime := allPoints[0].Time

	for _, point := range allPoints[1:] {
		if point.Time.Sub(lastSelectedTime) >= interval {
			selectedPoints = append(selectedPoints, point)
			lastSelectedTime = point.Time
		}
	}

	// Último ponto
	if len(allPoints) > 1 {
		lastPoint := allPoints[len(allPoints)-1]
		if lastPoint.Time.Sub(lastSelectedTime) > 5*time.Second {
			selectedPoints = append(selectedPoints, lastPoint)
		}
	}

	return selectedPoints
}
