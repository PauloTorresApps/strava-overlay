package handlers

import (
	"fmt"
	"log"
	"time"

	"strava-overlay/internal/gps"
	"strava-overlay/internal/services"
	"strava-overlay/internal/strava"
)

// FrontendGPSPoint representa um ponto GPS formatado para o frontend
type FrontendGPSPoint struct {
	Time     string  `json:"time"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Velocity float64 `json:"velocity"`
	Altitude float64 `json:"altitude"`
	Bearing  float64 `json:"bearing"`
	GForce   float64 `json:"gForce"`
}

// GPSHandler gerencia todas as operações relacionadas aos dados GPS
type GPSHandler struct {
	getStravaClient func() *strava.Client
	gpsService      *services.GPSService
}

// NewGPSHandler cria um novo handler de GPS
func NewGPSHandler(getStravaClient func() *strava.Client, gpsService *services.GPSService) *GPSHandler {
	return &GPSHandler{
		getStravaClient: getStravaClient,
		gpsService:      gpsService,
	}
}

// GetGPSPointForVideoTime finds the GPS point corresponding to a video's start time
func (h *GPSHandler) GetGPSPointForVideoTime(activityID int64, videoPath string) (FrontendGPSPoint, error) {
	client := h.getStravaClient()
	if client == nil {
		return FrontendGPSPoint{}, fmt.Errorf("not authenticated")
	}

	point, err := h.gpsService.GetGPSPointForVideoTime(client, activityID, videoPath)
	if err != nil {
		return FrontendGPSPoint{}, err
	}

	return h.convertToFrontendGPSPoint(point), nil
}

// GetGPSPointForMapClick encontra o ponto GPS mais próximo de um clique no mapa
func (h *GPSHandler) GetGPSPointForMapClick(activityID int64, lat, lng float64) (FrontendGPSPoint, error) {
	client := h.getStravaClient()
	if client == nil {
		return FrontendGPSPoint{}, fmt.Errorf("not authenticated")
	}

	point, err := h.gpsService.GetGPSPointForMapClick(client, activityID, lat, lng)
	if err != nil {
		return FrontendGPSPoint{}, err
	}

	return h.convertToFrontendGPSPoint(point), nil
}

// GetAllGPSPoints retorna pontos GPS selecionados inteligentemente para marcadores
func (h *GPSHandler) GetAllGPSPoints(activityID int64) ([]FrontendGPSPoint, error) {
	client := h.getStravaClient()
	if client == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	points, err := h.gpsService.GetIntelligentGPSPoints(client, activityID)
	if err != nil {
		return nil, err
	}

	return h.convertToFrontendGPSPoints(points), nil
}

// GetFullGPSTrajectory retorna TODOS os pontos GPS interpolados para desenhar o trajeto completo
func (h *GPSHandler) GetFullGPSTrajectory(activityID int64) ([]FrontendGPSPoint, error) {
	client := h.getStravaClient()
	if client == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	points, err := h.gpsService.GetFullGPSTrajectory(client, activityID)
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: Retornando trajeto COMPLETO com %d pontos GPS interpolados", len(points))
	return h.convertToFrontendGPSPoints(points), nil
}

// GetGPSPointsWithDensity - Versão com densidade customizável
func (h *GPSHandler) GetGPSPointsWithDensity(activityID int64, density string) ([]FrontendGPSPoint, error) {
	client := h.getStravaClient()
	if client == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	points, err := h.gpsService.GetGPSPointsWithDensity(client, activityID, density)
	if err != nil {
		return nil, err
	}

	log.Printf("DEBUG: Densidade '%s' - %d pontos selecionados", density, len(points))
	return h.convertToFrontendGPSPoints(points), nil
}

// Métodos auxiliares para conversão de tipos

func (h *GPSHandler) convertToFrontendGPSPoint(point gps.GPSPoint) FrontendGPSPoint {
	return FrontendGPSPoint{
		Time:     point.Time.Format(time.RFC3339),
		Lat:      point.Lat,
		Lng:      point.Lng,
		Velocity: point.Velocity,
		Altitude: point.Altitude,
		Bearing:  point.Bearing,
		GForce:   point.GForce,
	}
}

func (h *GPSHandler) convertToFrontendGPSPoints(points []gps.GPSPoint) []FrontendGPSPoint {
	frontendPoints := make([]FrontendGPSPoint, len(points))
	for i, point := range points {
		frontendPoints[i] = h.convertToFrontendGPSPoint(point)
	}
	return frontendPoints
}
