package video

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type VideoMetadata struct {
	CreationTime time.Time
	Duration     time.Duration
	Width        int
	Height       int
	FrameRate    float64
}

type FFProbeOutput struct {
	Format struct {
		Duration string            `json:"duration"`
		Tags     map[string]string `json:"tags"`
	} `json:"format"`
	Streams []struct {
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
	} `json:"streams"`
}

// Substitua esta função em internal/video/metadata.go
func GetVideoMetadata(filePath string) (*VideoMetadata, error) {
	// --- LOG DE DEPURAÇÃO ADICIONADO ---
	fmt.Printf("DEBUG: ffprobe está tentando analisar o arquivo no caminho: '%s'\n", filePath)
	// ------------------------------------

	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filePath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w. Output: %s", err, string(output))
	}

	// ... o resto da função continua o mesmo ...
	var probe FFProbeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, err
	}

	metadata := &VideoMetadata{}

	if duration, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		metadata.Duration = time.Duration(duration * float64(time.Second))
	}

	creationTimeStr := ""
	if probe.Format.Tags != nil {
		for key, value := range probe.Format.Tags {
			if strings.ToLower(key) == "creation_time" || strings.ToLower(key) == "date" {
				creationTimeStr = value
				break
			}
		}
	}

	if creationTimeStr != "" {
		layouts := []string{
			time.RFC3339,
			"2006-01-02T15:04:05.000000Z",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, creationTimeStr); err == nil {
				metadata.CreationTime = t
				break
			}
		}
	}

	if len(probe.Streams) > 0 {
		for _, stream := range probe.Streams {
			if stream.Width > 0 && stream.Height > 0 {
				metadata.Width = stream.Width
				metadata.Height = stream.Height

				if stream.RFrameRate != "" {
					parts := strings.Split(stream.RFrameRate, "/")
					if len(parts) == 2 {
						num, _ := strconv.ParseFloat(parts[0], 64)
						den, _ := strconv.ParseFloat(parts[1], 64)
						if den != 0 {
							metadata.FrameRate = num / den
						}
					}
				}
				break
			}
		}
	}

	return metadata, nil
}
