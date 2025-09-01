package strava

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func (c *Client) GetActivities() ([]Activity, error) {
	allActivities := make([]Activity, 0)
	page := 1
	perPage := 100

	log.Println("DEBUG: Iniciando busca de atividades no Strava...")

	for {
		url := fmt.Sprintf("%s/athlete/activities?page=%d&per_page=%d", c.baseURL, page, perPage)
		log.Printf("DEBUG: Buscando atividades da URL: %s", url)

		resp, err := c.httpClient.Get(url)
		if err != nil {
			log.Printf("ERRO: Falha ao fazer a requisição para o Strava: %v", err)
			return nil, err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("ERRO: Falha ao ler o corpo da resposta: %v", err)
			return nil, err
		}

		log.Printf("DEBUG: Resposta recebida do Strava (página %d): %s", page, string(body))

		var pageActivities []Activity
		if err := json.Unmarshal(body, &pageActivities); err != nil {
			log.Printf("ERRO: Falha ao decodificar o JSON das atividades: %v", err)
			return nil, err
		}

		log.Printf("DEBUG: %d atividades decodificadas da página %d.", len(pageActivities), page)

		if len(pageActivities) == 0 {
			log.Println("DEBUG: Nenhuma atividade retornada na página atual, encerrando busca.")
			break
		}

		allActivities = append(allActivities, pageActivities...)
		page++
	}

	log.Printf("DEBUG: Total de atividades brutas encontradas: %d", len(allActivities))

	gpsActivities := make([]Activity, 0)
	for _, activity := range allActivities {
		// CORREÇÃO: Apenas verificamos se existe um resumo de mapa (polyline).
		// Esta é a melhor forma de saber se a atividade tem um trajeto de GPS.
		if activity.Map.SummaryPolyline != "" {
			gpsActivities = append(gpsActivities, activity)
		}
	}

	log.Printf("DEBUG: Total de atividades com GPS após o filtro: %d", len(gpsActivities))

	return gpsActivities, nil
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
