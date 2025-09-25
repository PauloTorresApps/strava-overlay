package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"strava-overlay/internal/services"
	"strava-overlay/internal/strava"
)

// VideoHandler gerencia todas as operações relacionadas ao processamento de vídeo
type VideoHandler struct {
	getStravaClient func() *strava.Client
	videoService    *services.VideoService
	gpsService      *services.GPSService
}

// NewVideoHandler cria um novo handler de vídeo
func NewVideoHandler(
	getStravaClient func() *strava.Client,
	videoService *services.VideoService,
	gpsService *services.GPSService,
) *VideoHandler {
	return &VideoHandler{
		getStravaClient: getStravaClient,
		videoService:    videoService,
		gpsService:      gpsService,
	}
}

// ProcessVideoOverlay aplica o overlay ao vídeo
func (h *VideoHandler) ProcessVideoOverlay(activityID int64, videoPath string, manualStartTimeStr string, overlayPosition string) (string, error) {
	client := h.getStravaClient()
	if client == nil {
		return "", fmt.Errorf("not authenticated")
	}

	log.Printf("🎬 Iniciando processamento de vídeo para atividade %d com overlay na posição %s", activityID, overlayPosition)

	// Valida a posição
	validPositions := map[string]bool{
		"top-left":     true,
		"top-right":    true,
		"bottom-left":  true,
		"bottom-right": true,
	}

	if !validPositions[overlayPosition] {
		overlayPosition = "bottom-left" // Fallback para padrão
	}

	// Processa o vídeo usando o serviço especializado
	outputPath, err := h.videoService.ProcessVideoWithOverlay(
		client,
		activityID,
		videoPath,
		manualStartTimeStr,
		overlayPosition, // Novo parâmetro
		h.gpsService,
	)

	if err != nil {
		log.Printf("❌ Erro no processamento do vídeo: %v", err)
		return "", err
	}

	// Garante que o diretório de saída existe e retorna o caminho completo
	homeDir, _ := os.UserHomeDir()
	outputDir := filepath.Join(homeDir, "Strava Add Overlay")
	fullOutputPath := filepath.Join(outputDir, filepath.Base(outputPath))

	log.Printf("✅ Vídeo processado com sucesso: %s", fullOutputPath)
	return fullOutputPath, nil
}
