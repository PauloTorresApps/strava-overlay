package handlers

import (
	"strava-overlay/internal/config"
)

type ConfigHandler struct{}

func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

type FrontendConfig struct {
	ThunderforestAPIKey string   `json:"thunderforest_api_key,omitempty"`
	MapboxPublicToken   string   `json:"mapbox_public_token,omitempty"`
	AppVersion          string   `json:"app_version"`
	Environment         string   `json:"environment"`
	StravaAPIURL        string   `json:"strava_api_url"`
	ThunderforestURL    string   `json:"thunderforest_url"`
	MapboxAPIURL        string   `json:"mapbox_api_url"`
	DefaultMapProvider  string   `json:"default_map_provider"`
	AvailableProviders  []string `json:"available_providers"`
}

func (h *ConfigHandler) GetFrontendConfig() *FrontendConfig {
	cfg := config.AppConfig

	frontendCfg := &FrontendConfig{
		AppVersion:  cfg.AppVersion,
		Environment: cfg.Environment,

		StravaAPIURL:     "https://www.strava.com/api/v3",
		ThunderforestURL: "https://tile.thunderforest.com",
		MapboxAPIURL:     "https://api.mapbox.com",

		DefaultMapProvider: cfg.DefaultMapProvider,
		AvailableProviders: []string{
			"osm", "osmDark", "satellite", "terrain",
			"cartodb_dark", "cyclemap",
		},
	}

	// Adiciona chaves públicas se existirem
	if cfg.ThunderforestAPIKey != "" {
		frontendCfg.ThunderforestAPIKey = cfg.ThunderforestAPIKey
		frontendCfg.AvailableProviders = append(frontendCfg.AvailableProviders,
			"thunderforest_cycle", "thunderforest_outdoors")
	}

	if cfg.MapboxPublicToken != "" {
		frontendCfg.MapboxPublicToken = cfg.MapboxPublicToken
		frontendCfg.AvailableProviders = append(frontendCfg.AvailableProviders,
			"mapbox_streets", "mapbox_satellite", "mapbox_outdoors",
			"mapbox_dark", "mapbox_light", "mapbox_satellite_streets")
	}

	return frontendCfg
}

func (h *ConfigHandler) GetSecureAPIKeys() map[string]string {
	keys := make(map[string]string)
	cfg := config.AppConfig

	if cfg.ThunderforestAPIKey != "" {
		keys["thunderforest"] = cfg.ThunderforestAPIKey
	}

	if cfg.MapboxPublicToken != "" {
		keys["mapbox_public"] = cfg.MapboxPublicToken
	}

	// NUNCA expor:
	// - STRAVA_CLIENT_SECRET
	// - MAPBOX_SECRET_TOKEN

	return keys
}

func (h *ConfigHandler) GetMapProviderConfig() map[string]interface{} {
	providerConfig := make(map[string]interface{})
	cfg := config.AppConfig

	// Thunderforest
	if cfg.ThunderforestAPIKey != "" {
		providerConfig["thunderforest"] = map[string]interface{}{
			"api_key":  cfg.ThunderforestAPIKey,
			"enabled":  true,
			"styles":   []string{"cycle", "outdoors", "landscape", "transport"},
			"base_url": "https://tile.thunderforest.com",
		}
	} else {
		providerConfig["thunderforest"] = map[string]interface{}{
			"enabled": false,
			"reason":  "API key not configured",
		}
	}

	// Mapbox
	if cfg.MapboxPublicToken != "" {
		providerConfig["mapbox"] = map[string]interface{}{
			"public_token": cfg.MapboxPublicToken,
			"enabled":      true,
			"styles": []string{
				"streets-v12", "satellite-v9", "satellite-streets-v12",
				"outdoors-v12", "dark-v11", "light-v11",
			},
			"base_url": "https://api.mapbox.com/styles/v1/mapbox",
		}
	} else {
		providerConfig["mapbox"] = map[string]interface{}{
			"enabled": false,
			"reason":  "Public token not configured",
		}
	}

	// OpenStreetMap (sempre disponível)
	providerConfig["openstreetmap"] = map[string]interface{}{
		"enabled":  true,
		"base_url": "https://tile.openstreetmap.org",
		"styles":   []string{"standard"},
	}

	return providerConfig
}
