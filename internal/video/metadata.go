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

func GetVideoMetadata(filePath string) (*VideoMetadata, error) {
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filePath)
	
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var probe FFProbeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, err
	}

	metadata := &VideoMetadata{}

	if duration, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
		metadata.Duration = time.Duration(duration * float64(time.Second))
	}

	creationTimeStr := ""
	for key, value := range probe.Format.Tags {
		if strings.ToLower(key) == "creation_time" || strings.ToLower(key) == "date" {
			creationTimeStr = value
			break
		}
	}

	if creationTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, creationTimeStr); err == nil {
			metadata.CreationTime = t
		}
	}

	if len(probe.Streams) > 0 {
		stream := probe.Streams[0]
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
	}

	return metadata, nil
}
