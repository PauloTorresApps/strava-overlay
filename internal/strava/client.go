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
	page := 1
	// CORREÇÃO: Ajustado para 30 itens por página, conforme exigência do Strava.
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

	gpsActivities := make([]Activity, 0)
	for _, activity := range pageActivities {
		if activity.Map.SummaryPolyline != "" {
			gpsActivities = append(gpsActivities, activity)
		}
		// Para quando encontrarmos as 10 primeiras atividades com GPS
		if len(gpsActivities) >= 10 {
			break
		}
	}

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
