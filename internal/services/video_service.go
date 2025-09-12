package services

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"strava-overlay/internal/overlay"
	"strava-overlay/internal/strava"
	"strava-overlay/internal/video"
)

// VideoService encapsula toda a lógica complexa de processamento de vídeo
type VideoService struct{}

// NewVideoService cria um novo serviço de vídeo
func NewVideoService() *VideoService {
	return &VideoService{}
}

// ProcessVideoWithOverlay processa um vídeo aplicando overlay com dados GPS
func (s *VideoService) ProcessVideoWithOverlay(
	client *strava.Client,
	activityID int64,
	videoPath string,
	manualStartTimeStr string,
	gpsService *GPSService,
) (string, error) {
	log.Printf("🎬 Iniciando processamento de vídeo...")

	// 1. Obter metadados do vídeo
	videoMeta, err := video.GetVideoMetadata(videoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get video metadata: %w", err)
	}
	log.Printf("📹 Metadados do vídeo obtidos: duração=%.1fs, resolução=%dx%d",
		videoMeta.Duration.Seconds(), videoMeta.Width, videoMeta.Height)

	// 2. Obter detalhes da atividade
	detail, err := client.GetActivityDetail(activityID)
	if err != nil {
		return "", fmt.Errorf("failed to get activity detail: %w", err)
	}
	log.Printf("🚴 Detalhes da atividade obtidos: %s", detail.Name)

	// 3. Determinar tempo de início (manual ou automático)
	correctedVideoStartTime, err := s.determineVideoStartTime(videoMeta, detail, manualStartTimeStr)
	if err != nil {
		return "", fmt.Errorf("failed to determine video start time: %w", err)
	}

	// 4. Obter pontos GPS para o intervalo do vídeo
	correctedVideoEndTime := correctedVideoStartTime.Add(videoMeta.Duration)
	gpsPoints, err := gpsService.GetPointsForTimeRange(client, activityID, correctedVideoStartTime, correctedVideoEndTime)
	if err != nil {
		return "", fmt.Errorf("failed to get GPS points: %w", err)
	}

	if len(gpsPoints) == 0 {
		return "", fmt.Errorf("no GPS data found for video time range")
	}
	log.Printf("📍 %d pontos GPS obtidos para o vídeo", len(gpsPoints))

	// 5. Gerar sequência de overlays
	overlayGen := overlay.NewGenerator()
	defer overlayGen.Cleanup()

	overlayImages, err := overlayGen.GenerateOverlaySequence(gpsPoints, videoMeta.FrameRate)
	if err != nil {
		return "", fmt.Errorf("failed to generate overlays: %w", err)
	}
	log.Printf("🎨 %d imagens de overlay geradas", len(overlayImages))

	// 6. Definir caminho de saída
	outputPath, err := s.generateOutputPath(activityID)
	if err != nil {
		return "", fmt.Errorf("failed to generate output path: %w", err)
	}

	// 7. Aplicar overlays ao vídeo
	videoProcessor := video.NewProcessor()
	err = videoProcessor.ApplyOverlays(videoPath, overlayImages, outputPath)
	if err != nil {
		return "", fmt.Errorf("failed to apply overlays: %w", err)
	}

	log.Printf("✅ Vídeo processado com sucesso: %s", outputPath)
	return outputPath, nil
}

// === MÉTODOS AUXILIARES PRIVADOS ===

// determineVideoStartTime determina o tempo de início do vídeo (manual ou automático)
func (s *VideoService) determineVideoStartTime(
	videoMeta *video.VideoMetadata,
	detail *strava.ActivityDetail,
	manualStartTimeStr string,
) (time.Time, error) {
	if manualStartTimeStr != "" {
		// Usa o tempo manual se fornecido
		parsedTime, err := time.Parse(time.RFC3339, manualStartTimeStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse manual start time: %w", err)
		}
		log.Printf("🎯 Usando tempo de início manual: %s", parsedTime.Format("15:04:05"))
		return parsedTime, nil
	}

	// Lógica de fallback (automática)
	videoTimeUTC := videoMeta.CreationTime
	tzParts := strings.Split(detail.Timezone, " ")
	ianaTZ := tzParts[len(tzParts)-1]
	location, err := time.LoadLocation(ianaTZ)
	if err != nil {
		log.Printf("Aviso: fuso horário desconhecido '%s', usando UTC como padrão. Erro: %v", ianaTZ, err)
		location = time.UTC
	}

	correctedTime := time.Date(
		videoTimeUTC.Year(), videoTimeUTC.Month(), videoTimeUTC.Day(),
		videoTimeUTC.Hour(), videoTimeUTC.Minute(), videoTimeUTC.Second(), videoTimeUTC.Nanosecond(),
		location,
	)

	log.Printf("🕐 Usando tempo de início automático: %s", correctedTime.Format("15:04:05"))
	return correctedTime, nil
}

// generateOutputPath gera o caminho de saída para o vídeo processado
func (s *VideoService) generateOutputPath(activityID int64) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	outputDir := filepath.Join(homeDir, "Strava Add Overlay")
	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(outputDir, fmt.Sprintf("activity_%d_overlay.mp4", activityID))
	return outputPath, nil
}

// ValidateVideoFile valida se o arquivo de vídeo é suportado
func (s *VideoService) ValidateVideoFile(videoPath string) error {
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file does not exist: %s", videoPath)
	}

	// Verifica extensão
	ext := strings.ToLower(filepath.Ext(videoPath))
	supportedExts := []string{".mp4", ".mov", ".avi", ".mkv"}

	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return nil
		}
	}

	return fmt.Errorf("unsupported video format: %s. Supported formats: %v", ext, supportedExts)
}

// GetVideoInfo retorna informações básicas sobre um arquivo de vídeo
func (s *VideoService) GetVideoInfo(videoPath string) (*VideoInfo, error) {
	if err := s.ValidateVideoFile(videoPath); err != nil {
		return nil, err
	}

	meta, err := video.GetVideoMetadata(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get video metadata: %w", err)
	}

	return &VideoInfo{
		Path:         videoPath,
		FileName:     filepath.Base(videoPath),
		Duration:     meta.Duration,
		Width:        meta.Width,
		Height:       meta.Height,
		FrameRate:    meta.FrameRate,
		CreationTime: meta.CreationTime,
		Size:         s.getFileSize(videoPath),
	}, nil
}

// getFileSize obtém o tamanho do arquivo em bytes
func (s *VideoService) getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

// VideoInfo representa informações de um arquivo de vídeo
type VideoInfo struct {
	Path         string        `json:"path"`
	FileName     string        `json:"file_name"`
	Duration     time.Duration `json:"duration"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	FrameRate    float64       `json:"frame_rate"`
	CreationTime time.Time     `json:"creation_time"`
	Size         int64         `json:"size_bytes"`
}

// FormatDuration formata a duração para exibição
func (vi *VideoInfo) FormatDuration() string {
	hours := int(vi.Duration.Hours())
	minutes := int(vi.Duration.Minutes()) % 60
	seconds := int(vi.Duration.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

// FormatSize formata o tamanho do arquivo para exibição
func (vi *VideoInfo) FormatSize() string {
	const unit = 1024
	if vi.Size < unit {
		return fmt.Sprintf("%d B", vi.Size)
	}
	div, exp := int64(unit), 0
	for n := vi.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(vi.Size)/float64(div), "KMGTPE"[exp])
}

// GetResolutionString retorna a resolução como string
func (vi *VideoInfo) GetResolutionString() string {
	return fmt.Sprintf("%dx%d", vi.Width, vi.Height)
}
