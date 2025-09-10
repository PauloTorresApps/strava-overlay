// internal/cache/manager.go
package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strava-overlay/internal/gps"
	"strava-overlay/internal/strava"
	"time"
)

type CacheManager struct {
	cacheDir string
}

type ActivityCache struct {
	ActivityID   int64                  `json:"activity_id"`
	ProcessedAt  time.Time              `json:"processed_at"`
	GPSPoints    []gps.GPSPoint         `json:"gps_points"`
	Streams      map[string]interface{} `json:"streams"`
	ActivityData *strava.ActivityDetail `json:"activity_data"`
	Hash         string                 `json:"hash"`
}

func NewCacheManager() *CacheManager {
	homeDir, _ := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".strava-overlay", "cache")
	os.MkdirAll(cacheDir, 0755)

	return &CacheManager{
		cacheDir: cacheDir,
	}
}

func (cm *CacheManager) GetCacheKey(activityID int64, dataType string) string {
	return fmt.Sprintf("%d_%s.json", activityID, dataType)
}

func (cm *CacheManager) hashStreamsData(streams map[string]interface{}) string {
	data, _ := json.Marshal(streams)
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// Cache de dados GPS processados
func (cm *CacheManager) CacheGPSData(activityID int64, points []gps.GPSPoint, streams map[string]interface{}, detail *strava.ActivityDetail) error {
	cache := ActivityCache{
		ActivityID:   activityID,
		ProcessedAt:  time.Now(),
		GPSPoints:    points,
		Streams:      streams,
		ActivityData: detail,
		Hash:         cm.hashStreamsData(streams),
	}

	filename := cm.GetCacheKey(activityID, "gps")
	filepath := filepath.Join(cm.cacheDir, filename)

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

func (cm *CacheManager) GetCachedGPSData(activityID int64, streams map[string]interface{}) (*ActivityCache, bool) {
	filename := cm.GetCacheKey(activityID, "gps")
	filepath := filepath.Join(cm.cacheDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, false
	}

	var cache ActivityCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}

	// Verifica se o hash dos streams ainda é válido
	currentHash := cm.hashStreamsData(streams)
	if cache.Hash != currentHash {
		return nil, false // Cache inválido
	}

	// Verifica idade do cache (7 dias)
	if time.Since(cache.ProcessedAt) > 7*24*time.Hour {
		return nil, false
	}

	return &cache, true
}

// Cache de overlays gerados
type OverlayCache struct {
	ActivityID   int64           `json:"activity_id"`
	VideoHash    string          `json:"video_hash"`
	GeneratedAt  time.Time       `json:"generated_at"`
	OverlayPaths []string        `json:"overlay_paths"`
	Settings     OverlaySettings `json:"settings"`
}

type OverlaySettings struct {
	StartTime    time.Time `json:"start_time"`
	Duration     float64   `json:"duration"`
	OverlayStyle string    `json:"overlay_style"`
	MaxSpeed     float64   `json:"max_speed"`
}

func (cm *CacheManager) CacheOverlays(activityID int64, videoHash string, overlayPaths []string, settings OverlaySettings) error {
	cache := OverlayCache{
		ActivityID:   activityID,
		VideoHash:    videoHash,
		GeneratedAt:  time.Now(),
		OverlayPaths: overlayPaths,
		Settings:     settings,
	}

	filename := fmt.Sprintf("%d_%s_overlays.json", activityID, videoHash[:8])
	filepath := filepath.Join(cm.cacheDir, filename)

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

func (cm *CacheManager) GetCachedOverlays(activityID int64, videoHash string, settings OverlaySettings) (*OverlayCache, bool) {
	filename := fmt.Sprintf("%d_%s_overlays.json", activityID, videoHash[:8])
	filepath := filepath.Join(cm.cacheDir, filename)

	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, false
	}

	var cache OverlayCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, false
	}

	// Verifica se as configurações são compatíveis
	if !cm.settingsMatch(cache.Settings, settings) {
		return nil, false
	}

	// Verifica se os arquivos ainda existem
	for _, path := range cache.OverlayPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, false
		}
	}

	// Cache válido por 24 horas
	if time.Since(cache.GeneratedAt) > 24*time.Hour {
		return nil, false
	}

	return &cache, true
}

func (cm *CacheManager) settingsMatch(cached, current OverlaySettings) bool {
	return cached.StartTime.Equal(current.StartTime) &&
		cached.Duration == current.Duration &&
		cached.OverlayStyle == current.OverlayStyle &&
		cached.MaxSpeed == current.MaxSpeed
}

// Limpeza de cache antigo
func (cm *CacheManager) CleanupOldCache() error {
	files, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-30 * 24 * time.Hour) // 30 dias

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filepath := filepath.Join(cm.cacheDir, file.Name())
			os.Remove(filepath)
		}
	}

	return nil
}

// Estatísticas do cache
type CacheStats struct {
	TotalFiles        int       `json:"total_files"`
	TotalSize         int64     `json:"total_size_bytes"`
	GPSCacheFiles     int       `json:"gps_cache_files"`
	OverlayCacheFiles int       `json:"overlay_cache_files"`
	OldestFile        time.Time `json:"oldest_file"`
	NewestFile        time.Time `json:"newest_file"`
}

func (cm *CacheManager) GetCacheStats() (*CacheStats, error) {
	files, err := os.ReadDir(cm.cacheDir)
	if err != nil {
		return nil, err
	}

	stats := &CacheStats{
		OldestFile: time.Now(),
		NewestFile: time.Unix(0, 0),
	}

	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			continue
		}

		stats.TotalFiles++
		stats.TotalSize += info.Size()

		if info.ModTime().Before(stats.OldestFile) {
			stats.OldestFile = info.ModTime()
		}
		if info.ModTime().After(stats.NewestFile) {
			stats.NewestFile = info.ModTime()
		}

		filename := file.Name()
		if filepath.Ext(filename) == ".json" {
			if contains(filename, "_gps.json") {
				stats.GPSCacheFiles++
			} else if contains(filename, "_overlays.json") {
				stats.OverlayCacheFiles++
			}
		}
	}

	return stats, nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr
}
