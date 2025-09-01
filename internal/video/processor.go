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

	tempDir := filepath.Dir(overlayImages[0])
	imagePattern := filepath.Join(tempDir, "overlay_%06d.png")

	cmd := exec.Command("ffmpeg",
		"-i", inputVideo,
		"-i", imagePattern,
		"-filter_complex", "[1:v]scale=200:200[overlay];[0:v][overlay]overlay=10:H-210",
		"-c:a", "copy",
		"-c:v", "libx264",
		"-preset", "medium",
		"-crf", "23",
		"-y",
		outputPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %s, output: %s", err, string(output))
	}

	return nil
}

func (p *Processor) createImageList(images []string, listFile string) error {
	var content strings.Builder
	for _, img := range images {
		content.WriteString(fmt.Sprintf("file '%s'\n", img))
		content.WriteString("duration 0.033333\n")
	}
	
	return os.WriteFile(listFile, []byte(content.String()), 0644)
}
