package services

import (
	"context"
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

// ProgressCallback é chamado durante o processamento para reportar progresso
type ProgressCallback func(stage string, progress float64, message string)

// VideoService encapsula toda a lógica complexa de processamento de vídeo
type VideoService struct {
	progressCallback   ProgressCallback
	completionCallback func(success bool, outputPath string, err error)
}

func (s *VideoService) SetCompletionCallback(callback func(success bool, outputPath string, err error)) {
	s.completionCallback = callback
}

// NewVideoService cria um novo serviço de vídeo
func NewVideoService() *VideoService {
	return &VideoService{}
}

// SetProgressCallback define o callback de progresso
func (s *VideoService) SetProgressCallback(callback ProgressCallback) {
	s.progressCallback = callback
}

// reportProgress envia atualização de progresso se callback estiver definido
func (s *VideoService) reportProgress(stage string, progress float64, message string) {
	if s.progressCallback != nil {
		s.progressCallback(stage, progress, message)
	}
	log.Printf("📊 [%s] %.1f%% - %s", stage, progress, message)
}

// ProcessVideoWithOverlay processa um vídeo aplicando overlay com dados GPS
func (s *VideoService) ProcessVideoWithOverlay(
	ctx context.Context,
	client *strava.Client,
	activityID int64,
	videoPath string,
	manualStartTimeStr string,
	overlayPosition string,
	gpsService *GPSService,
) (string, error) {
	s.reportProgress("init", 0, "Iniciando processamento...")

	// Verifica cancelamento em cada etapa
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	s.reportProgress("metadata", 5, "Obtendo metadados do vídeo...")
	videoMeta, err := video.GetVideoMetadata(videoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get video metadata: %w", err)
	}
	s.reportProgress("metadata", 10, fmt.Sprintf("Vídeo: %.1fs, %dx%d",
		videoMeta.Duration.Seconds(), videoMeta.Width, videoMeta.Height))

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	s.reportProgress("activity", 15, "Carregando atividade do Strava...")
	detail, err := client.GetActivityDetail(activityID)
	if err != nil {
		return "", fmt.Errorf("failed to get activity detail: %w", err)
	}
	s.reportProgress("activity", 20, fmt.Sprintf("Atividade: %s", detail.Name))

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	s.reportProgress("sync", 25, "Sincronizando tempo GPS-vídeo...")
	correctedVideoStartTime, err := s.determineVideoStartTime(videoMeta, detail, manualStartTimeStr)
	if err != nil {
		return "", fmt.Errorf("failed to determine video start time: %w", err)
	}
	s.reportProgress("sync", 30, "Sincronização concluída")

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	s.reportProgress("gps", 35, "Carregando dados GPS...")
	correctedVideoEndTime := correctedVideoStartTime.Add(videoMeta.Duration)
	gpsPoints, err := gpsService.GetPointsForTimeRange(client, activityID, correctedVideoStartTime, correctedVideoEndTime)
	if err != nil {
		return "", fmt.Errorf("failed to get GPS points: %w", err)
	}
	if len(gpsPoints) == 0 {
		return "", fmt.Errorf("no GPS data found for video time range")
	}
	s.reportProgress("gps", 40, fmt.Sprintf("%d pontos GPS carregados", len(gpsPoints)))

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	s.reportProgress("overlay", 45, "Gerando overlays...")
	overlayGen := overlay.NewGeneratorWithPosition(overlayPosition)
	defer overlayGen.Cleanup()

	overlayGen.SetProgressCallback(func(current, total int) {
		if ctx.Err() != nil {
			return
		}
		progress := 45 + (15 * float64(current) / float64(total))
		s.reportProgress("overlay", progress, fmt.Sprintf("Gerando overlay %d/%d", current, total))
	})

	overlayImages, err := overlayGen.GenerateOverlaySequence(gpsPoints, videoMeta.FrameRate)
	if err != nil {
		return "", fmt.Errorf("failed to generate overlays: %w", err)
	}
	s.reportProgress("overlay", 60, fmt.Sprintf("%d overlays gerados", len(overlayImages)))

	if ctx.Err() != nil {
		overlayGen.Cleanup()
		return "", ctx.Err()
	}

	s.reportProgress("output", 65, "Preparando arquivo de saída...")
	outputPath, err := s.generateOutputPath(activityID)
	if err != nil {
		return "", fmt.Errorf("failed to generate output path: %w", err)
	}
	s.reportProgress("output", 70, "Caminho definido")

	s.reportProgress("encoding", 70, "Iniciando codificação do vídeo...")
	videoProcessor := video.NewProcessor()

	videoProcessor.SetProgressCallback(func(progress float64) {
		encodingProgress := 70 + (25 * progress / 100)
		s.reportProgress("encoding", encodingProgress, fmt.Sprintf("Codificando: %.1f%%", progress))
	})

	err = videoProcessor.ApplyOverlaysWithPosition(ctx, videoPath, overlayImages, outputPath, overlayPosition)
	if err != nil {
		if s.completionCallback != nil {
			s.completionCallback(false, "", err)
		}
		return "", fmt.Errorf("failed to apply overlays: %w", err)
	}

	if s.completionCallback != nil {
		s.completionCallback(true, outputPath, nil)
	}

	s.reportProgress("complete", 100, "Processamento concluído!")
	log.Printf("✅ Vídeo processado com sucesso: %s", outputPath)
	return outputPath, nil
}

// === MÉTODOS AUXILIARES (sem mudanças) ===

func (s *VideoService) determineVideoStartTime(
	videoMeta *video.VideoMetadata,
	detail *strava.ActivityDetail,
	manualStartTimeStr string,
) (time.Time, error) {
	if manualStartTimeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, manualStartTimeStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("failed to parse manual start time: %w", err)
		}
		log.Printf("🎯 Usando tempo de início manual: %s", parsedTime.Format("15:04:05"))
		return parsedTime, nil
	}

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

func (s *VideoService) ValidateVideoFile(videoPath string) error {
	if _, err := os.Stat(videoPath); os.IsNotExist(err) {
		return fmt.Errorf("video file does not exist: %s", videoPath)
	}

	ext := strings.ToLower(filepath.Ext(videoPath))
	supportedExts := []string{".mp4", ".mov", ".avi", ".mkv"}

	for _, supportedExt := range supportedExts {
		if ext == supportedExt {
			return nil
		}
	}

	return fmt.Errorf("unsupported video format: %s. Supported formats: %v", ext, supportedExts)
}

func (s *VideoService) getFileSize(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

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

func (vi *VideoInfo) GetResolutionString() string {
	return fmt.Sprintf("%dx%d", vi.Width, vi.Height)
}
