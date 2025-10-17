package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config armazena todas as configurações da aplicação
type Config struct {
	// Strava
	StravaClientID     string
	StravaClientSecret string

	// Thunderforest
	ThunderforestAPIKey string

	// Mapbox
	MapboxPublicToken string
	MapboxSecretToken string

	// App
	AppVersion         string
	Environment        string
	DefaultMapProvider string
}

var AppConfig *Config

// LoadConfig carrega configurações de variáveis de ambiente
func LoadConfig() error {
	// Tenta carregar .env do diretório atual
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️ Arquivo .env não encontrado, usando variáveis de ambiente do sistema")
	}

	AppConfig = &Config{
		// Strava (obrigatório)
		StravaClientID:     getEnv("STRAVA_CLIENT_ID", ""),
		StravaClientSecret: getEnv("STRAVA_CLIENT_SECRET", ""),

		// Thunderforest (opcional)
		ThunderforestAPIKey: getEnv("THUNDERFOREST_API_KEY", ""),

		// Mapbox (opcional)
		MapboxPublicToken: getEnv("MAPBOX_PUBLIC_TOKEN", ""),
		MapboxSecretToken: getEnv("MAPBOX_SECRET_TOKEN", ""),

		// App
		AppVersion:         getEnv("APP_VERSION", "1.0.0"),
		Environment:        getEnv("APP_ENV", "development"),
		DefaultMapProvider: getEnv("DEFAULT_MAP_PROVIDER", "osm"),
	}

	// Valida configurações obrigatórias
	if err := validateConfig(AppConfig); err != nil {
		return err
	}

	log.Printf("✅ Configurações carregadas com sucesso")
	log.Printf("   Ambiente: %s", AppConfig.Environment)
	log.Printf("   Versão: %s", AppConfig.AppVersion)
	log.Printf("   Strava Client ID: %s", maskString(AppConfig.StravaClientID))

	return nil
}

// validateConfig valida se as configurações obrigatórias estão presentes
func validateConfig(cfg *Config) error {
	if cfg.StravaClientID == "" {
		return fmt.Errorf("STRAVA_CLIENT_ID é obrigatório")
	}
	if cfg.StravaClientSecret == "" {
		return fmt.Errorf("STRAVA_CLIENT_SECRET é obrigatório")
	}
	return nil
}

// getEnv obtém variável de ambiente com valor padrão
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// maskString mascara string sensível para logs
func maskString(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// GetConfigPath retorna o caminho para o arquivo de configuração
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".strava-overlay", "config")
}
