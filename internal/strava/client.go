package strava

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type Activity struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	StartDate    time.Time `json:"start_date"`
	Timezone     string    `json:"timezone"`
	Distance     float64   `json:"distance"`
	MovingTime   int       `json:"moving_time"`
	MaxSpeed     float64   `json:"max_speed"`
	HasHeartrate bool      `json:"has_heartrate"`
	StartLatLng  []float64 `json:"start_latlng"`
	EndLatLng    []float64 `json:"end_latlng"`
	Map          Map       `json:"map"`
}

type Map struct {
	ID              string `json:"id"`
	Polyline        string `json:"polyline"`
	SummaryPolyline string `json:"summary_polyline"`
}

type ActivityDetail struct {
	*Activity
	Calories      float64 `json:"calories"`
	ElevationGain float64 `json:"total_elevation_gain"`
}

type ActivityStream struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewClient(token *oauth2.Token) *Client {
	config := &oauth2.Config{}
	client := config.Client(context.Background(), token)

	return &Client{
		httpClient: client,
		baseURL:    "https://www.strava.com/api/v3",
	}
}

// GetActivitiesPage busca uma página específica de atividades
func (c *Client) GetActivitiesPage(page, perPage int) ([]Activity, error) {
	// Validação dos parâmetros
	if perPage > 30 {
		perPage = 30 // Strava limita a 30 por página
	}
	if perPage < 1 {
		perPage = 1
	}
	if page < 1 {
		page = 1
	}

	url := fmt.Sprintf("%s/athlete/activities?page=%d&per_page=%d", c.baseURL, page, perPage)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer requisição: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resposta HTTP inválida: %d", resp.StatusCode)
	}

	var activities []Activity
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return activities, nil
}

// GetAllActivities busca todas as atividades disponíveis (com limite para evitar sobrecarga)
func (c *Client) GetAllActivities(maxPages int) ([]Activity, error) {
	var allActivities []Activity

	if maxPages <= 0 {
		maxPages = 10 // Limite padrão de 10 páginas (300 atividades)
	}

	for page := 1; page <= maxPages; page++ {
		pageActivities, err := c.GetActivitiesPage(page, 30)
		if err != nil {
			return allActivities, fmt.Errorf("erro ao buscar página %d: %w", page, err)
		}

		allActivities = append(allActivities, pageActivities...)

		// Se recebeu menos que 30 atividades, não há mais páginas
		if len(pageActivities) < 30 {
			break
		}
	}

	return allActivities, nil
}

// GetActivities - mantida para compatibilidade, busca primeira página
func (c *Client) GetActivities() ([]Activity, error) {
	page := 1
	perPage := 30

	url := fmt.Sprintf("%s/athlete/activities?page=%d&per_page=%d", c.baseURL, page, perPage)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pageActivities []Activity
	if err := json.NewDecoder(resp.Body).Decode(&pageActivities); err != nil {
		return nil, err
	}

	// Retorna todas as atividades da primeira página
	return pageActivities, nil
}

func (c *Client) GetActivityDetail(activityID int64) (*ActivityDetail, error) {
	url := fmt.Sprintf("%s/activities/%d", c.baseURL, activityID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var detail ActivityDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, err
	}

	return &detail, nil
}

func (c *Client) GetActivityStreams(activityID int64) (map[string]ActivityStream, error) {
	url := fmt.Sprintf("%s/activities/%d/streams?keys=time,latlng,velocity_smooth,altitude&key_by_type=true",
		c.baseURL, activityID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var streams map[string]ActivityStream
	if err := json.NewDecoder(resp.Body).Decode(&streams); err != nil {
		return nil, err
	}

	return streams, nil
}
