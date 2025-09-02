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

func (p *Processor) ApplyOverlays(inputVideo string, overlayImages []string, outputPath string) error {
	if len(overlayImages) == 0 {
		return fmt.Errorf("no overlay images provided")
	}

	// Obtém metadados do vídeo para calcular frame rate e duração
	metadata, err := GetVideoMetadata(inputVideo)
	if err != nil {
		return fmt.Errorf("erro ao obter metadados do vídeo: %w", err)
	}

	// Cria diretório temporário para lista de imagens
	tempDir := filepath.Dir(overlayImages[0])
	listFile := filepath.Join(tempDir, "overlay_list.txt")

	// Calcula duração por frame GPS
	totalDuration := metadata.Duration.Seconds()
	durationPerGPSPoint := totalDuration / float64(len(overlayImages))

	// Cria lista de imagens com duração correta
	err = p.createImageListWithDuration(overlayImages, listFile, durationPerGPSPoint)
	if err != nil {
		return fmt.Errorf("erro ao criar lista de imagens: %w", err)
	}

	// Comando FFmpeg para aplicar overlays
	cmd := exec.Command("ffmpeg",
		"-i", inputVideo, // vídeo original
		"-f", "concat", // formato concatenação
		"-safe", "0", // permite caminhos absolutos
		"-i", listFile, // lista de overlays
		"-filter_complex", "[1:v]scale=250:180[overlay];[0:v][overlay]overlay=10:H-190", // redimensiona e posiciona overlay
		"-c:a", "copy", // copia áudio sem recodificar
		"-c:v", "libx264", // codec de vídeo
		"-preset", "medium", // preset de qualidade/velocidade
		"-crf", "23", // qualidade (menor = melhor)
		"-shortest", // para quando o mais curto terminar
		"-y",        // sobrescreve arquivo de saída
		outputPath,
	)

	fmt.Printf("DEBUG: Aplicando %d overlays ao vídeo\n", len(overlayImages))
	fmt.Printf("DEBUG: Duração por ponto GPS: %.3f segundos\n", durationPerGPSPoint)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %s, output: %s", err, string(output))
	}

	// Remove arquivo temporário
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
