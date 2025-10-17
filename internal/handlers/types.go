package handlers

// types.go - Define tipos compartilhados entre os handlers

// FrontendActivityDetail representa detalhes de atividade formatados para o frontend
type FrontendActivityDetail struct {
	*FrontendActivity
	Calories      float64 `json:"calories"`
	ElevationGain float64 `json:"total_elevation_gain"`
}
