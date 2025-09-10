// internal/analytics/tracker.go
package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type AnalyticsTracker struct {
	sessionID   string
	startTime   time.Time
	events      []Event
	metricsPath string
	enabled     bool
}

type Event struct {
	Timestamp   time.Time              `json:"timestamp"`
	SessionID   string                 `json:"session_id"`
	EventType   string                 `json:"event_type"`
	Category    string                 `json:"category"`
	Action      string                 `json:"action"`
	Label       string                 `json:"label,omitempty"`
	Value       interface{}            `json:"value,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
	Performance *PerformanceMetrics    `json:"performance,omitempty"`
}

type PerformanceMetrics struct {
	Duration        time.Duration `json:"duration"`
	MemoryUsed      uint64        `json:"memory_used_bytes"`
	MemoryAllocated uint64        `json:"memory_allocated_bytes"`
	GoroutineCount  int           `json:"goroutine_count"`
	CPUUsage        float64       `json:"cpu_usage_percent"`
}

type SessionSummary struct {
	SessionID        string           `json:"session_id"`
	StartTime        time.Time        `json:"start_time"`
	EndTime          time.Time        `json:"end_time"`
	Duration         time.Duration    `json:"duration"`
	EventCount       int              `json:"event_count"`
	ActivitiesLoaded int              `json:"activities_loaded"`
	VideosProcessed  int              `json:"videos_processed"`
	ErrorCount       int              `json:"error_count"`
	PerformanceStats PerformanceStats `json:"performance_stats"`
	FeatureUsage     map[string]int   `json:"feature_usage"`
	SystemInfo       SystemInfo       `json:"system_info"`
}

type PerformanceStats struct {
	AvgMemoryUsage      uint64        `json:"avg_memory_usage_bytes"`
	PeakMemoryUsage     uint64        `json:"peak_memory_usage_bytes"`
	AvgProcessingTime   time.Duration `json:"avg_processing_time"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
}

type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	AppVersion   string `json:"app_version"`
}

func NewAnalyticsTracker(enabled bool) *AnalyticsTracker {
	homeDir, _ := os.UserHomeDir()
	metricsPath := filepath.Join(homeDir, ".strava-overlay", "analytics")
	os.MkdirAll(metricsPath, 0755)

	sessionID := fmt.Sprintf("%d_%d", time.Now().Unix(), os.Getpid())

	tracker := &AnalyticsTracker{
		sessionID:   sessionID,
		startTime:   time.Now(),
		events:      make([]Event, 0),
		metricsPath: metricsPath,
		enabled:     enabled,
	}

	if enabled {
		tracker.TrackEvent("system", "session_start", "", nil)
	}

	return tracker
}

func (at *AnalyticsTracker) TrackEvent(category, action, label string, properties map[string]interface{}) {
	if !at.enabled {
		return
	}

	event := Event{
		Timestamp:  time.Now(),
		SessionID:  at.sessionID,
		EventType:  "user_action",
		Category:   category,
		Action:     action,
		Label:      label,
		Properties: properties,
	}

	at.events = append(at.events, event)
}

func (at *AnalyticsTracker) TrackPerformance(category, action string, duration time.Duration, properties map[string]interface{}) {
	if !at.enabled {
		return
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	perf := &PerformanceMetrics{
		Duration:        duration,
		MemoryUsed:      m.Alloc,
		MemoryAllocated: m.TotalAlloc,
		GoroutineCount:  runtime.NumGoroutine(),
	}

	event := Event{
		Timestamp:   time.Now(),
		SessionID:   at.sessionID,
		EventType:   "performance",
		Category:    category,
		Action:      action,
		Properties:  properties,
		Performance: perf,
	}

	at.events = append(at.events, event)
}

func (at *AnalyticsTracker) TrackError(category, action, errorMsg string, properties map[string]interface{}) {
	if !at.enabled {
		return
	}

	if properties == nil {
		properties = make(map[string]interface{})
	}
	properties["error_message"] = errorMsg

	event := Event{
		Timestamp:  time.Now(),
		SessionID:  at.sessionID,
		EventType:  "error",
		Category:   category,
		Action:     action,
		Properties: properties,
	}

	at.events = append(at.events, event)
}

// Métodos específicos para o domínio da aplicação
func (at *AnalyticsTracker) TrackStravaAuth(success bool, duration time.Duration) {
	status := "failed"
	if success {
		status = "success"
	}

	at.TrackPerformance("strava", "authentication", duration, map[string]interface{}{
		"status": status,
	})
}

func (at *AnalyticsTracker) TrackActivityLoad(activityCount int, hasGPSCount int, duration time.Duration) {
	at.TrackPerformance("strava", "load_activities", duration, map[string]interface{}{
		"activity_count": activityCount,
		"gps_count":      hasGPSCount,
		"gps_ratio":      float64(hasGPSCount) / float64(activityCount),
	})
}

func (at *AnalyticsTracker) TrackGPSProcessing(activityID int64, pointCount int, interpolatedCount int, duration time.Duration) {
	at.TrackPerformance("gps", "process_data", duration, map[string]interface{}{
		"activity_id":         activityID,
		"raw_point_count":     pointCount,
		"interpolated_count":  interpolatedCount,
		"interpolation_ratio": float64(interpolatedCount) / float64(pointCount),
	})
}

func (at *AnalyticsTracker) TrackVideoProcessing(activityID int64, videoDuration time.Duration, overlayCount int, processingDuration time.Duration, success bool) {
	status := "failed"
	if success {
		status = "success"
	}

	at.TrackPerformance("video", "process_overlay", processingDuration, map[string]interface{}{
		"activity_id":      activityID,
		"video_duration":   videoDuration.Seconds(),
		"overlay_count":    overlayCount,
		"status":           status,
		"processing_ratio": processingDuration.Seconds() / videoDuration.Seconds(),
	})
}

func (at *AnalyticsTracker) TrackMapInteraction(action string, density string, pointCount int) {
	at.TrackEvent("map", action, density, map[string]interface{}{
		"point_count": pointCount,
	})
}

func (at *AnalyticsTracker) TrackCacheUsage(cacheType string, hit bool, size int64) {
	status := "miss"
	if hit {
		status = "hit"
	}

	at.TrackEvent("cache", status, cacheType, map[string]interface{}{
		"size_bytes": size,
	})
}

func (at *AnalyticsTracker) TrackFeatureUsage(feature string, properties map[string]interface{}) {
	at.TrackEvent("feature", "usage", feature, properties)
}

// Finalização e persistência
func (at *AnalyticsTracker) EndSession() {
	if !at.enabled {
		return
	}

	at.TrackEvent("system", "session_end", "", nil)

	summary := at.generateSessionSummary()
	at.persistSession(summary)
	at.persistEvents()
}

func (at *AnalyticsTracker) generateSessionSummary() SessionSummary {
	endTime := time.Now()
	duration := endTime.Sub(at.startTime)

	summary := SessionSummary{
		SessionID:    at.sessionID,
		StartTime:    at.startTime,
		EndTime:      endTime,
		Duration:     duration,
		EventCount:   len(at.events),
		FeatureUsage: make(map[string]int),
		SystemInfo: SystemInfo{
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			AppVersion:   "1.0.0", // Pode ser injetado durante build
		},
	}

	// Análise de eventos
	var totalMemory uint64
	var peakMemory uint64
	var totalProcessingTime time.Duration
	var processingCount int

	for _, event := range at.events {
		// Contagem de features
		if event.EventType == "user_action" {
			key := fmt.Sprintf("%s:%s", event.Category, event.Action)
			summary.FeatureUsage[key]++
		}

		// Contagem de erros
		if event.EventType == "error" {
			summary.ErrorCount++
		}

		// Estatísticas de performance
		if event.Performance != nil {
			totalMemory += event.Performance.MemoryUsed
			if event.Performance.MemoryUsed > peakMemory {
				peakMemory = event.Performance.MemoryUsed
			}
			totalProcessingTime += event.Performance.Duration
			processingCount++
		}

		// Contagem específica de atividades e vídeos
		if event.Category == "strava" && event.Action == "load_activities" {
			if count, exists := event.Properties["activity_count"]; exists {
				if activityCount, ok := count.(int); ok {
					summary.ActivitiesLoaded += activityCount
				}
			}
		}

		if event.Category == "video" && event.Action == "process_overlay" {
			summary.VideosProcessed++
		}
	}

	// Calcula médias
	if processingCount > 0 {
		summary.PerformanceStats.AvgMemoryUsage = totalMemory / uint64(processingCount)
		summary.PerformanceStats.AvgProcessingTime = totalProcessingTime / time.Duration(processingCount)
	}
	summary.PerformanceStats.PeakMemoryUsage = peakMemory
	summary.PerformanceStats.TotalProcessingTime = totalProcessingTime

	return summary
}

func (at *AnalyticsTracker) persistSession(summary SessionSummary) {
	filename := fmt.Sprintf("session_%s.json", at.sessionID)
	filepath := filepath.Join(at.metricsPath, filename)

	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(filepath, data, 0644)
}

func (at *AnalyticsTracker) persistEvents() {
	filename := fmt.Sprintf("events_%s.json", at.sessionID)
	filepath := filepath.Join(at.metricsPath, filename)

	data, err := json.MarshalIndent(at.events, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(filepath, data, 0644)
}

// Análise e relatórios
func (at *AnalyticsTracker) GetSessionSummaries(days int) ([]SessionSummary, error) {
	files, err := os.ReadDir(at.metricsPath)
	if err != nil {
		return nil, err
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	var summaries []SessionSummary

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" &&
			filepath.Base(file.Name())[:8] == "session_" {

			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().After(cutoff) {
				filePath := filepath.Join(at.metricsPath, file.Name())
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}

				var summary SessionSummary
				if err := json.Unmarshal(data, &summary); err != nil {
					continue
				}

				summaries = append(summaries, summary)
			}
		}
	}

	return summaries, nil
}

func (at *AnalyticsTracker) GenerateUsageReport(days int) (*UsageReport, error) {
	summaries, err := at.GetSessionSummaries(days)
	if err != nil {
		return nil, err
	}

	report := &UsageReport{
		PeriodDays:         days,
		GeneratedAt:        time.Now(),
		SessionCount:       len(summaries),
		FeatureUsage:       make(map[string]FeatureStats),
		SystemStats:        make(map[string]int),
		PerformanceMetrics: AggregatedPerformanceMetrics{},
	}

	var totalDuration time.Duration
	var totalEvents int
	var totalErrors int
	var totalActivities int
	var totalVideos int
	var totalMemoryUsage uint64
	var peakMemoryOverall uint64
	var sessionWithMemoryData int

	for _, summary := range summaries {
		totalDuration += summary.Duration
		totalEvents += summary.EventCount
		totalErrors += summary.ErrorCount
		totalActivities += summary.ActivitiesLoaded
		totalVideos += summary.VideosProcessed

		// Agregação de uso de features
		for feature, count := range summary.FeatureUsage {
			if stats, exists := report.FeatureUsage[feature]; exists {
				stats.TotalUses += count
				stats.SessionsUsed++
				report.FeatureUsage[feature] = stats
			} else {
				report.FeatureUsage[feature] = FeatureStats{
					TotalUses:    count,
					SessionsUsed: 1,
				}
			}
		}

		// Estatísticas do sistema
		systemKey := fmt.Sprintf("%s_%s", summary.SystemInfo.OS, summary.SystemInfo.Architecture)
		report.SystemStats[systemKey]++

		// Métricas de performance
		if summary.PerformanceStats.AvgMemoryUsage > 0 {
			totalMemoryUsage += summary.PerformanceStats.AvgMemoryUsage
			sessionWithMemoryData++
		}
		if summary.PerformanceStats.PeakMemoryUsage > peakMemoryOverall {
			peakMemoryOverall = summary.PerformanceStats.PeakMemoryUsage
		}
		report.PerformanceMetrics.TotalProcessingTime += summary.PerformanceStats.TotalProcessingTime
	}

	// Calcula médias e estatísticas finais
	if len(summaries) > 0 {
		report.AvgSessionDuration = totalDuration / time.Duration(len(summaries))
		report.TotalUsageTime = totalDuration
		report.TotalEvents = totalEvents
		report.TotalErrors = totalErrors
		report.ActivitiesProcessed = totalActivities
		report.VideosProcessed = totalVideos
		report.ErrorRate = float64(totalErrors) / float64(totalEvents)
	}

	if sessionWithMemoryData > 0 {
		report.PerformanceMetrics.AvgMemoryUsage = totalMemoryUsage / uint64(sessionWithMemoryData)
	}
	report.PerformanceMetrics.PeakMemoryUsage = peakMemoryOverall

	// Calcula uso relativo de features
	for feature, stats := range report.FeatureUsage {
		stats.UsageRate = float64(stats.SessionsUsed) / float64(len(summaries))
		report.FeatureUsage[feature] = stats
	}

	return report, nil
}

type UsageReport struct {
	PeriodDays          int                          `json:"period_days"`
	GeneratedAt         time.Time                    `json:"generated_at"`
	SessionCount        int                          `json:"session_count"`
	AvgSessionDuration  time.Duration                `json:"avg_session_duration"`
	TotalUsageTime      time.Duration                `json:"total_usage_time"`
	TotalEvents         int                          `json:"total_events"`
	TotalErrors         int                          `json:"total_errors"`
	ErrorRate           float64                      `json:"error_rate"`
	ActivitiesProcessed int                          `json:"activities_processed"`
	VideosProcessed     int                          `json:"videos_processed"`
	FeatureUsage        map[string]FeatureStats      `json:"feature_usage"`
	SystemStats         map[string]int               `json:"system_stats"`
	PerformanceMetrics  AggregatedPerformanceMetrics `json:"performance_metrics"`
}

type FeatureStats struct {
	TotalUses    int     `json:"total_uses"`
	SessionsUsed int     `json:"sessions_used"`
	UsageRate    float64 `json:"usage_rate"` // Percentage of sessions that used this feature
}

type AggregatedPerformanceMetrics struct {
	AvgMemoryUsage      uint64        `json:"avg_memory_usage_bytes"`
	PeakMemoryUsage     uint64        `json:"peak_memory_usage_bytes"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
}

// Cleanup de arquivos antigos
func (at *AnalyticsTracker) CleanupOldAnalytics(retentionDays int) error {
	files, err := os.ReadDir(at.metricsPath)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	deletedCount := 0

	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				filePath := filepath.Join(at.metricsPath, file.Name())
				if err := os.Remove(filePath); err == nil {
					deletedCount++
				}
			}
		}
	}

	fmt.Printf("Cleaned up %d old analytics files\n", deletedCount)
	return nil
}

// Métodos para exportação de dados
func (at *AnalyticsTracker) ExportAnalytics(outputPath string, days int) error {
	summaries, err := at.GetSessionSummaries(days)
	if err != nil {
		return err
	}

	report, err := at.GenerateUsageReport(days)
	if err != nil {
		return err
	}

	exportData := struct {
		Report     *UsageReport     `json:"report"`
		Sessions   []SessionSummary `json:"sessions"`
		ExportedAt time.Time        `json:"exported_at"`
	}{
		Report:     report,
		Sessions:   summaries,
		ExportedAt: time.Now(),
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}

// Sistema de métricas em tempo real
type RealTimeMetrics struct {
	CurrentMemoryUsage uint64        `json:"current_memory_usage_bytes"`
	GoroutineCount     int           `json:"goroutine_count"`
	SessionDuration    time.Duration `json:"session_duration"`
	EventsThisSession  int           `json:"events_this_session"`
	ErrorsThisSession  int           `json:"errors_this_session"`
	LastActivity       time.Time     `json:"last_activity"`
}

func (at *AnalyticsTracker) GetRealTimeMetrics() RealTimeMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	errorCount := 0
	lastActivity := at.startTime

	for _, event := range at.events {
		if event.EventType == "error" {
			errorCount++
		}
		if event.Timestamp.After(lastActivity) {
			lastActivity = event.Timestamp
		}
	}

	return RealTimeMetrics{
		CurrentMemoryUsage: m.Alloc,
		GoroutineCount:     runtime.NumGoroutine(),
		SessionDuration:    time.Since(at.startTime),
		EventsThisSession:  len(at.events),
		ErrorsThisSession:  errorCount,
		LastActivity:       lastActivity,
	}
}

// Helper function para performance profiling
func (at *AnalyticsTracker) ProfileFunction(category, action string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	properties := make(map[string]interface{})
	if err != nil {
		properties["error"] = err.Error()
		at.TrackError(category, action, err.Error(), properties)
	} else {
		properties["success"] = true
	}

	at.TrackPerformance(category, action, duration, properties)
	return err
}

// Wrapper para tracking automático de funções críticas
func (at *AnalyticsTracker) TrackCriticalOperation(name string, fn func() error) error {
	return at.ProfileFunction("critical", name, fn)
}
