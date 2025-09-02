package video

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

// Substitua a função ApplyOverlays em internal/video/processor.go

// Substitua a função ApplyOverlays em internal/video/processor.go

func (p *Processor) ApplyOverlays(inputVideo string, overlayImages []string, outputPath string) error {
	if len(overlayImages) == 0 {
		return fmt.Errorf("no overlay images provided")
	}

	metadata, err := GetVideoMetadata(inputVideo)
	if err != nil {
		return fmt.Errorf("erro ao obter metadados: %w", err)
	}

	tempDir := filepath.Dir(overlayImages[0])
	listFile := filepath.Join(tempDir, "overlay_list.txt")

	totalDuration := metadata.Duration.Seconds()
	durationPerGPSPoint := totalDuration / float64(len(overlayImages))

	err = p.createImageListWithDuration(overlayImages, listFile, durationPerGPSPoint)
	if err != nil {
		return fmt.Errorf("erro ao criar lista: %w", err)
	}

	// CORREÇÃO CRÍTICA: FFmpeg pipeline simplificado
	cmd := exec.Command("ffmpeg",
		"-i", inputVideo,
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-filter_complex",
		"[1:v]scale=500:500[overlay];"+ // Scale sem flags problemáticas
			"[0:v][overlay]overlay=30:main_h-530", // Posição fixa válida
		"-c:a", "copy",
		"-c:v", "libx264",
		"-preset", "fast", // Preset compatível
		"-crf", "20", // Qualidade balanceada
		"-y",
		outputPath,
	)

	fmt.Printf("DEBUG: Aplicando %d overlays HD ao vídeo\n", len(overlayImages))
	fmt.Printf("DEBUG: Comando FFmpeg: %s\n", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %s\nOutput: %s", err, string(output))
	}

	os.Remove(listFile)
	return nil
}

func (p *Processor) createImageListWithDuration(images []string, listFile string, duration float64) error {
	var content strings.Builder

	for _, img := range images {
		content.WriteString(fmt.Sprintf("file '%s'\n", img))
		content.WriteString(fmt.Sprintf("duration %.6f\n", duration))
	}

	// Adiciona última imagem sem duração (necessário para concat)
	if len(images) > 0 {
		content.WriteString(fmt.Sprintf("file '%s'\n", images[len(images)-1]))
	}

	return os.WriteFile(listFile, []byte(content.String()), 0644)
}
