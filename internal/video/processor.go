package video

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Processor struct {
	progressCallback func(progress float64)
}

type ProgressCallback func(progress float64)

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) SetProgressCallback(callback ProgressCallback) {
	p.progressCallback = callback
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
		"-progress", "pipe:1",
		"-y",
		outputPath,
	)

	// Captura stdout e stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("erro ao criar pipe stdout: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("erro ao criar pipe stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("erro ao iniciar ffmpeg: %w", err)
	}

	// Monitora progresso do FFmpeg (corrigido: passa io.Reader)
	go p.monitorFFmpegProgress(stdout, totalDuration)

	// Captura erros
	var stderrOutput strings.Builder
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrOutput.WriteString(scanner.Text() + "\n")
		}
	}()

	// Aguarda conclusão
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg falhou: %s\nOutput: %s", err, stderrOutput.String())
	}

	os.Remove(listFile)
	return nil
}

// monitorFFmpegProgress monitora o progresso do FFmpeg em tempo real
// Corrigido: recebe io.Reader em vez de *bufio.Scanner
func (p *Processor) monitorFFmpegProgress(reader io.Reader, totalDuration float64) {
	timeRegex := regexp.MustCompile(`out_time_ms=(\d+)`)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		if matches := timeRegex.FindStringSubmatch(line); len(matches) > 1 {
			timeMicros, err := strconv.ParseFloat(matches[1], 64)
			if err == nil {
				currentSeconds := timeMicros / 1000000.0
				progress := (currentSeconds / totalDuration) * 100

				if progress > 100 {
					progress = 100
				}

				if p.progressCallback != nil {
					p.progressCallback(progress)
				}
			}
		}
	}
}

func (p *Processor) calculateOverlayCoordinates(position string) (string, string) {
	margin := "10"

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
		return margin, fmt.Sprintf("main_h-overlay_h-%s", margin)
	}
}

func (p *Processor) createImageListWithDuration(images []string, listFile string, duration float64) error {
	var content strings.Builder
	content.WriteString("ffconcat version 1.0\n")

	for _, img := range images {
		absPath, err := filepath.Abs(img)
		if err != nil {
			return err
		}
		safePath := strings.ReplaceAll(absPath, `\`, `\\`)
		content.WriteString(fmt.Sprintf("file '%s'\n", safePath))
		content.WriteString(fmt.Sprintf("duration %.6f\n", duration))
	}

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
