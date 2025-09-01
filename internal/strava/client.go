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
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"`
	StartDate        time.Time `json:"start_date"`
	Distance         float64   `json:"distance"`
	MovingTime       int       `json:"moving_time"`
	MaxSpeed         float64   `json:"max_speed"`
	HasHeartrate     bool      `json:"has_heartrate"`
	StartLatLng      []float64 `json:"start_latlng"`
	EndLatLng        []float64 `json:"end_latlng"`
	Map              Map       `json:"map"`
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

func (c *Client) GetActivities(limit int) ([]Activity, error) {
	url := fmt.Sprintf("%s/athlete/activities?per_page=%d", c.baseURL, limit)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var activities []Activity
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		return nil, err
	}

	var gpsActivities []Activity
	for _, activity := range activities {
		if len(activity.StartLatLng) > 0 && activity.Map.Polyline != "" {
			gpsActivities = append(gpsActivities, activity)
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
