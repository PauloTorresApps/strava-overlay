package handlers

import (
	"fmt"
	"log"
	"time"

	"strava-overlay/internal/strava"
)

// FrontendActivity representa uma atividade formatada para o frontend
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
	HasGPS      bool       `json:"has_gps"`
}

// PaginatedActivities representa uma resposta paginada de atividades
type PaginatedActivities struct {
	Activities  []FrontendActivity `json:"activities"`
	Page        int                `json:"page"`
	PerPage     int                `json:"per_page"`
	HasMore     bool               `json:"has_more"`
	TotalLoaded int                `json:"total_loaded"`
}

// ActivityHandler gerencia todas as operações relacionadas às atividades
type ActivityHandler struct {
	getStravaClient func() *strava.Client
}

// NewActivityHandler cria um novo handler de atividades
func NewActivityHandler(getStravaClient func() *strava.Client) *ActivityHandler {
	return &ActivityHandler{
		getStravaClient: getStravaClient,
	}
}

// GetActivitiesPage retorna uma página específica de atividades
func (h *ActivityHandler) GetActivitiesPage(page int) (*PaginatedActivities, error) {
	client := h.getStravaClient()
	if client == nil {
		return nil, fmt.Errorf("not authenticated")
	}

	perPage := 30 // Máximo permitido pelo Strava
	log.Printf("📋 Carregando página %d de atividades (até %d itens)...", page, perPage)

	activities, err := client.GetActivitiesPage(page, perPage)
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

	// Determina se há mais páginas
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

// GetActivities - mantida para compatibilidade, mas recomenda-se usar GetActivitiesPage
func (h *ActivityHandler) GetActivities() ([]FrontendActivity, error) {
	result, err := h.GetActivitiesPage(1)
	if err != nil {
		return nil, err
	}
	return result.Activities, nil
}

// GetActivityDetail retrieves detailed activity information
func (h *ActivityHandler) GetActivityDetail(activityID int64) (*strava.ActivityDetail, error) {
	client := h.getStravaClient()
	if client == nil {
		return nil, fmt.Errorf("not authenticated")
	}
	return client.GetActivityDetail(activityID)
}
