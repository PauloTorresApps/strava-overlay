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

	// Comando FFmpeg para aplicar a sequência de imagens como um overlay
	cmd := exec.Command("ffmpeg",
		"-i", inputVideo, // Entrada 0: vídeo principal
		"-f", "concat", // Entrada 1: sequência de imagens
		"-safe", "0",
		"-i", listFile,
		"-filter_complex",
		// **CORREÇÃO**: Garante que o formato de píxeis com transparência (rgba) seja mantido
		// e que os timestamps sejam reiniciados para o stream de overlay.
		"[1:v]format=rgba,setpts=PTS-STARTPTS[ovr];[0:v][ovr]overlay=30:main_h-overlay_h-30",
		"-c:a", "copy", // Copia o stream de áudio sem re-codificar
		"-c:v", "libx264", // Codec de vídeo
		"-preset", "fast", // Preset de velocidade de codificação
		"-crf", "22", // Fator de Qualidade Constante (menor é melhor)
		"-y", // Sobrescreve o arquivo de saída se ele existir
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
