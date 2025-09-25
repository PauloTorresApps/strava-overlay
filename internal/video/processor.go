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

func (p *Processor) ApplyOverlaysWithPosition(inputVideo string, overlayImages []string, outputPath string, position string) error {
	if len(overlayImages) == 0 {
		return fmt.Errorf("nenhuma imagem de overlay fornecida")
	}

	metadata, err := GetVideoMetadata(inputVideo)
	if err != nil {
		return fmt.Errorf("erro ao obter metadados do vídeo: %w", err)
	}

	tempDir := filepath.Dir(overlayImages[0])
	listFile := filepath.Join(tempDir, "overlay_list.txt")

	// Calcula a duração de cada imagem de overlay com base na duração total do vídeo e no número de pontos GPS
	totalDuration := metadata.Duration.Seconds()
	durationPerImage := totalDuration / float64(len(overlayImages))

	err = p.createImageListWithDuration(overlayImages, listFile, durationPerImage)
	if err != nil {
		return fmt.Errorf("erro ao criar a lista de imagens para o FFmpeg: %w", err)
	}

	overlayX, overlayY := p.calculateOverlayCoordinates(position)

	filterComplex := fmt.Sprintf(
		"[1:v]format=rgba,setpts=PTS-STARTPTS[ovr];[0:v][ovr]overlay=%s:%s",
		overlayX, overlayY,
	)

	// Comando FFmpeg para aplicar a sequência de imagens como um overlay
	cmd := exec.Command("ffmpeg",
		"-i", inputVideo,
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-filter_complex", filterComplex,
		"-map_metadata", "0",
		"-c:a", "copy",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "18",
		"-y",
		outputPath,
	)

	fmt.Printf("DEBUG: Aplicando %d overlays ao vídeo\n", len(overlayImages))
	fmt.Printf("DEBUG: Comando FFmpeg: %s\n", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Fornece uma mensagem de erro detalhada incluindo a saída do ffmpeg
		return fmt.Errorf("o comando ffmpeg falhou: %s\nSaída do FFmpeg: %s", err, string(output))
	}

	// É uma boa prática remover o arquivo de lista temporário
	os.Remove(listFile)
	return nil
}

func (p *Processor) calculateOverlayCoordinates(position string) (string, string) {
	margin := "10" // margem em pixels

	switch position {
	case "top-left":
		return margin, margin
	case "top-right":
		return fmt.Sprintf("main_w-overlay_w-%s", margin), margin
	case "bottom-left":
		return margin, fmt.Sprintf("main_h-overlay_h-%s", margin)
	case "bottom-right":
		return fmt.Sprintf("main_w-overlay_w-%s", margin), fmt.Sprintf("main_h-overlay_h-%s", margin)
	default:
		// Padrão: bottom-left
		return margin, fmt.Sprintf("main_h-overlay_h-%s", margin)
	}
}

// createImageListWithDuration cria um arquivo de texto para o demuxer concat do ffmpeg.
// A cada imagem é atribuída uma duração específica.
func (p *Processor) createImageListWithDuration(images []string, listFile string, duration float64) error {
	var content strings.Builder

	// Este formato é exigido pelo demuxer concat
	content.WriteString("ffconcat version 1.0\n")

	for _, img := range images {
		// Usa caminhos absolutos para segurança
		absPath, err := filepath.Abs(img)
		if err != nil {
			return err
		}
		// Escapa barras invertidas para caminhos do Windows
		safePath := strings.ReplaceAll(absPath, `\`, `\\`)
		content.WriteString(fmt.Sprintf("file '%s'\n", safePath))
		content.WriteString(fmt.Sprintf("duration %.6f\n", duration))
	}

	// A última imagem precisa ser especificada novamente sem duração para finalizar a sequência corretamente.
	if len(images) > 0 {
		absPath, err := filepath.Abs(images[len(images)-1])
		if err != nil {
			return err
		}
		safePath := strings.ReplaceAll(absPath, `\`, `\\`)
		content.WriteString(fmt.Sprintf("file '%s'\n", safePath))
	}

	return os.WriteFile(listFile, []byte(content.String()), 0644)
}
