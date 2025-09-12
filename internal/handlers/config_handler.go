package handlers

import (
	"os"
)

// ConfigHandler gerencia as configurações de ambiente para o frontend
type ConfigHandler struct{}

// NewConfigHandler cria um novo handler de configuração
func NewConfigHandler() *ConfigHandler {
	return &ConfigHandler{}
}

// FrontendConfig representa as configurações expostas para o frontend
type FrontendConfig struct {
	// Chaves de API seguras para o frontend
	ThunderforestAPIKey string `json:"thunderforest_api_key,omitempty"`
	MapboxPublicToken   string `json:"mapbox_public_token,omitempty"`

	// Configurações gerais
	AppVersion  string `json:"app_version"`
	Environment string `json:"environment"`

	// URLs de API (sem chaves sensíveis)
	StravaAPIURL     string `json:"strava_api_url"`
	ThunderforestURL string `json:"thunderforest_url"`
	MapboxAPIURL     string `json:"mapbox_api_url"`

	// Configurações de mapa
	DefaultMapProvider string   `json:"default_map_provider"`
	AvailableProviders []string `json:"available_providers"`
}

// GetFrontendConfig retorna configurações seguras para o frontend
func (h *ConfigHandler) GetFrontendConfig() *FrontendConfig {
	config := &FrontendConfig{
		// Versão da aplicação
		AppVersion:  getEnvWithDefault("APP_VERSION", "1.0.0"),
		Environment: getEnvWithDefault("APP_ENV", "development"),

		// URLs de API
		StravaAPIURL:     "https://www.strava.com/api/v3",
		ThunderforestURL: "https://tile.thunderforest.com",
		MapboxAPIURL:     "https://api.mapbox.com",

		// Configurações padrão
		DefaultMapProvider: getEnvWithDefault("DEFAULT_MAP_PROVIDER", "osm"),
		AvailableProviders: []string{
			"osm", "osmDark", "satellite", "terrain",
			"cartodb_dark", "cyclemap",
		},
	}

	// Chaves de API públicas (seguro expor)
	if thunderforestKey := os.Getenv("THUNDERFOREST_API_KEY"); thunderforestKey != "" {
		config.ThunderforestAPIKey = thunderforestKey
		config.AvailableProviders = append(config.AvailableProviders, "thunderforest_cycle", "thunderforest_outdoors")
	}

	// Token público do Mapbox (diferente do secret)
	if mapboxPublicToken := os.Getenv("MAPBOX_PUBLIC_TOKEN"); mapboxPublicToken != "" {
		config.MapboxPublicToken = mapboxPublicToken
		config.AvailableProviders = append(config.AvailableProviders,
			"mapbox_streets", "mapbox_satellite", "mapbox_outdoors",
			"mapbox_dark", "mapbox_light", "mapbox_satellite_streets")
	}

	return config
}

// GetSecureAPIKeys retorna apenas as chaves que são seguras para o frontend
func (h *ConfigHandler) GetSecureAPIKeys() map[string]string {
	keys := make(map[string]string)

	// Apenas chaves públicas/frontend-safe
	if thunderforestKey := os.Getenv("THUNDERFOREST_API_KEY"); thunderforestKey != "" {
		keys["thunderforest"] = thunderforestKey
	}

	if mapboxPublicToken := os.Getenv("MAPBOX_PUBLIC_TOKEN"); mapboxPublicToken != "" {
		keys["mapbox_public"] = mapboxPublicToken
	}

	// NUNCA expor essas chaves:
	// - STRAVA_CLIENT_SECRET
	// - MAPBOX_SECRET_TOKEN
	// - DATABASE_URL
	// - etc.

	return keys
}

// GetMapProviderConfig retorna configuração específica de provedores de mapa
func (h *ConfigHandler) GetMapProviderConfig() map[string]interface{} {
	config := make(map[string]interface{})

	// Thunderforest
	if thunderforestKey := os.Getenv("THUNDERFOREST_API_KEY"); thunderforestKey != "" {
		config["thunderforest"] = map[string]interface{}{
			"api_key":  thunderforestKey,
			"enabled":  true,
			"styles":   []string{"cycle", "outdoors", "landscape", "transport"},
			"base_url": "https://tile.thunderforest.com",
		}
	} else {
		config["thunderforest"] = map[string]interface{}{
			"enabled": false,
			"reason":  "API key not configured",
		}
	}

	// Mapbox
	if mapboxPublicToken := os.Getenv("MAPBOX_PUBLIC_TOKEN"); mapboxPublicToken != "" {
		config["mapbox"] = map[string]interface{}{
			"public_token": mapboxPublicToken,
			"enabled":      true,
			"styles": []string{
				"streets-v12", "satellite-v9", "satellite-streets-v12",
				"outdoors-v12", "dark-v11", "light-v11",
			},
			"base_url": "https://api.mapbox.com/styles/v1/mapbox",
		}
	} else {
		config["mapbox"] = map[string]interface{}{
			"enabled": false,
			"reason":  "Public token not configured",
		}
	}

	// OpenStreetMap (sempre disponível)
	config["openstreetmap"] = map[string]interface{}{
		"enabled":  true,
		"base_url": "https://tile.openstreetmap.org",
		"styles":   []string{"standard"},
	}

	return config
}

// Função auxiliar para pegar variável de ambiente com valor padrão
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
