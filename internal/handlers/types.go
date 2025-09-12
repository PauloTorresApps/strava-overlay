package handlers

// types.go - Define tipos compartilhados entre os handlers

import "strava-overlay/internal/strava"

// FrontendActivityDetail representa detalhes de atividade formatados para o frontend
type FrontendActivityDetail struct {
	*FrontendActivity
	Calories      float64 `json:"calories"`
	ElevationGain float64 `json:"total_elevation_gain"`
}

// ConvertToFrontendActivityDetail converte ActivityDetail para formato do frontend
func ConvertToFrontendActivityDetail(detail *strava.ActivityDetail) *FrontendActivityDetail {
	return &FrontendActivityDetail{
		FrontendActivity: &FrontendActivity{
			ID:          detail.ID,
			Name:        detail.Name,
			Type:        detail.Type,
			StartDate:   detail.StartDate.Format("2006-01-02T15:04:05Z07:00"),
			Distance:    detail.Distance,
			MovingTime:  detail.MovingTime,
			MaxSpeed:    detail.MaxSpeed,
			StartLatLng: detail.StartLatLng,
			EndLatLng:   detail.EndLatLng,
			Map:         detail.Map,
			HasGPS:      detail.Map.SummaryPolyline != "",
		},
		Calories:      detail.Calories,
		ElevationGain: detail.ElevationGain,
	}
}
