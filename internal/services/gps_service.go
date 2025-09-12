package services

import (
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"strava-overlay/internal/gps"
	"strava-overlay/internal/strava"
	"strava-overlay/internal/video"
)

// GPSService encapsula toda a lógica complexa de processamento de GPS
type GPSService struct{}

// NewGPSService cria um novo serviço de GPS
func NewGPSService() *GPSService {
	return &GPSService{}
}

// GetGPSPointForVideoTime encontra o ponto GPS correspondente ao tempo de início do vídeo
func (s *GPSService) GetGPSPointForVideoTime(client *strava.Client, activityID int64, videoPath string) (gps.GPSPoint, error) {
	videoMeta, err := video.GetVideoMetadata(videoPath)
	if err != nil {
		return gps.GPSPoint{}, fmt.Errorf("failed to get video metadata: %w", err)
	}

	detail, err := client.GetActivityDetail(activityID)
	if err != nil {
		return gps.GPSPoint{}, fmt.Errorf("failed to get activity detail: %w", err)
	}

	// Correção de fuso horário
	correctedVideoStartTime := s.correctVideoTimeZone(videoMeta.CreationTime, detail.Timezone)

	// Debug melhorado
	fmt.Printf("=== SINCRONIZAÇÃO GPS-VÍDEO (VERSÃO CORRIGIDA) ===\n")
	fmt.Printf("Vídeo (UTC original): %s\n", videoMeta.CreationTime.Format("15:04:05 MST"))
	fmt.Printf("Vídeo (fuso corrigido): %s\n", correctedVideoStartTime.Format("15:04:05 MST"))
	fmt.Printf("Atividade início: %s\n", detail.StartDate.Format("15:04:05 MST"))
	fmt.Printf("Diferença temporal: %.1f segundos\n", correctedVideoStartTime.Sub(detail.StartDate).Seconds())

	streams, err := client.GetActivityStreams(activityID)
	if err != nil {
		return gps.GPSPoint{}, fmt.Errorf("failed to get activity streams: %w", err)
	}

	processor, err := s.createGPSProcessor(streams, detail.StartDate)
	if err != nil {
		return gps.GPSPoint{}, err
	}

	point, found := processor.GetPointForTime(correctedVideoStartTime)
	if !found {
		return gps.GPSPoint{}, fmt.Errorf("no matching GPS point found")
	}

	// Validação final das coordenadas
	if err := s.validateGPSCoordinates(point); err != nil {
		return gps.GPSPoint{}, err
	}

	fmt.Printf("Ponto encontrado: %.6f, %.6f em %s\n",
		point.Lat, point.Lng, point.Time.Format("15:04:05"))
	fmt.Printf("===========================\n")

	return point, nil
}

// GetGPSPointForMapClick encontra o ponto GPS mais próximo de um clique no mapa
func (s *GPSService) GetGPSPointForMapClick(client *strava.Client, activityID int64, lat, lng float64) (gps.GPSPoint, error) {
	detail, err := client.GetActivityDetail(activityID)
	if err != nil {
		return gps.GPSPoint{}, fmt.Errorf("failed to get activity detail: %w", err)
	}

	streams, err := client.GetActivityStreams(activityID)
	if err != nil {
		return gps.GPSPoint{}, fmt.Errorf("failed to get activity streams: %w", err)
	}

	processor, err := s.createGPSProcessor(streams, detail.StartDate)
	if err != nil {
		return gps.GPSPoint{}, err
	}

	point, found := processor.GetPointForCoords(lat, lng)
	if !found {
		return gps.GPSPoint{}, fmt.Errorf("no matching GPS point found for coordinates")
	}

	return point, nil
}

// GetIntelligentGPSPoints retorna pontos GPS selecionados inteligentemente para marcadores
func (s *GPSService) GetIntelligentGPSPoints(client *strava.Client, activityID int64) ([]gps.GPSPoint, error) {
	detail, err := client.GetActivityDetail(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity detail: %w", err)
	}

	streams, err := client.GetActivityStreams(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity streams: %w", err)
	}

	processor, err := s.createGPSProcessor(streams, detail.StartDate)
	if err != nil {
		return nil, err
	}

	// Pega todos os pontos interpolados
	allPoints := processor.GetAllPoints()

	// Aplica seleção inteligente
	selectedPoints := s.selectIntelligentGPSPoints(allPoints)

	fmt.Printf("DEBUG: Selecionados %d pontos GPS inteligentes de %d pontos interpolados totais\n",
		len(selectedPoints), len(allPoints))

	return selectedPoints, nil
}

// GetFullGPSTrajectory retorna TODOS os pontos GPS interpolados
func (s *GPSService) GetFullGPSTrajectory(client *strava.Client, activityID int64) ([]gps.GPSPoint, error) {
	detail, err := client.GetActivityDetail(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity detail: %w", err)
	}

	streams, err := client.GetActivityStreams(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity streams: %w", err)
	}

	processor, err := s.createGPSProcessor(streams, detail.StartDate)
	if err != nil {
		return nil, err
	}

	return processor.GetAllPoints(), nil
}

// GetGPSPointsWithDensity retorna pontos com densidade customizável
func (s *GPSService) GetGPSPointsWithDensity(client *strava.Client, activityID int64, density string) ([]gps.GPSPoint, error) {
	detail, err := client.GetActivityDetail(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity detail: %w", err)
	}

	streams, err := client.GetActivityStreams(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity streams: %w", err)
	}

	processor, err := s.createGPSProcessor(streams, detail.StartDate)
	if err != nil {
		return nil, err
	}

	allPoints := processor.GetAllPoints()
	var selectedPoints []gps.GPSPoint

	// Seleção baseada na densidade escolhida
	switch density {
	case "high":
		selectedPoints = s.selectPointsByInterval(allPoints, 15*time.Second)
	case "medium":
		selectedPoints = s.selectIntelligentGPSPoints(allPoints)
	case "low":
		selectedPoints = s.selectPointsByInterval(allPoints, 60*time.Second)
	case "ultra_high":
		selectedPoints = s.selectPointsByInterval(allPoints, 5*time.Second)
	default:
		selectedPoints = s.selectIntelligentGPSPoints(allPoints)
	}

	return selectedPoints, nil
}

// GetPointsForTimeRange retorna pontos GPS para um intervalo de tempo específico
func (s *GPSService) GetPointsForTimeRange(client *strava.Client, activityID int64, startTime, endTime time.Time) ([]gps.GPSPoint, error) {
	detail, err := client.GetActivityDetail(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity detail: %w", err)
	}

	streams, err := client.GetActivityStreams(activityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity streams: %w", err)
	}

	processor, err := s.createGPSProcessor(streams, detail.StartDate)
	if err != nil {
		return nil, err
	}

	return processor.GetPointsForTimeRange(startTime, endTime), nil
}

// === MÉTODOS AUXILIARES PRIVADOS ===

// correctVideoTimeZone corrige o fuso horário do vídeo baseado na atividade
func (s *GPSService) correctVideoTimeZone(videoTimeUTC time.Time, timezone string) time.Time {
	tzParts := strings.Split(timezone, " ")
	ianaTZ := tzParts[len(tzParts)-1]
	location, err := time.LoadLocation(ianaTZ)
	if err != nil {
		log.Printf("Aviso: fuso horário desconhecido '%s', usando UTC. Erro: %v", ianaTZ, err)
		location = time.UTC
	}

	return time.Date(
		videoTimeUTC.Year(), videoTimeUTC.Month(), videoTimeUTC.Day(),
		videoTimeUTC.Hour(), videoTimeUTC.Minute(), videoTimeUTC.Second(), videoTimeUTC.Nanosecond(),
		location,
	)
}

// createGPSProcessor cria um processador GPS a partir dos streams
func (s *GPSService) createGPSProcessor(streams map[string]strava.ActivityStream, startDate time.Time) (*gps.GPSProcessor, error) {
	// Valida streams
	timeStream, timeExists := streams["time"]
	latlngStream, latlngExists := streams["latlng"]
	if !timeExists || !latlngExists || timeStream.Data == nil || latlngStream.Data == nil {
		return nil, fmt.Errorf("streams GPS ausentes ou vazios")
	}

	processor := gps.NewGPSProcessor()
	err := processor.ProcessStreamData(
		timeStream.Data.([]interface{}),
		latlngStream.Data.([]interface{}),
		s.getOptionalStreamData(streams, "velocity_smooth"),
		s.getOptionalStreamData(streams, "altitude"),
		startDate,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process GPS data: %w", err)
	}

	return processor, nil
}

// getOptionalStreamData obtém dados de stream opcionais
func (s *GPSService) getOptionalStreamData(streams map[string]strava.ActivityStream, key string) []interface{} {
	if stream, exists := streams[key]; exists && stream.Data != nil {
		return stream.Data.([]interface{})
	}
	return nil
}

// validateGPSCoordinates valida se as coordenadas GPS são válidas
func (s *GPSService) validateGPSCoordinates(point gps.GPSPoint) error {
	if point.Lat == 0 && point.Lng == 0 {
		return fmt.Errorf("coordenadas GPS inválidas (0,0)")
	}
	if point.Lat < -90 || point.Lat > 90 || point.Lng < -180 || point.Lng > 180 {
		return fmt.Errorf("coordenadas GPS fora dos limites válidos")
	}
	return nil
}

// selectIntelligentGPSPoints aplica seleção inteligente de pontos GPS para marcadores
func (s *GPSService) selectIntelligentGPSPoints(allPoints []gps.GPSPoint) []gps.GPSPoint {
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
		distance := s.calculateDistance(lastSelectedPoint, currentPoint)

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

// selectPointsByInterval seleção por intervalo fixo
func (s *GPSService) selectPointsByInterval(allPoints []gps.GPSPoint, interval time.Duration) []gps.GPSPoint {
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

// calculateDistance calcula distância entre dois pontos GPS usando fórmula de Haversine
func (s *GPSService) calculateDistance(p1, p2 gps.GPSPoint) float64 {
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
