// internal/config/settings.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppSettings struct {
	// Configurações de Overlay
	Overlay OverlaySettings `json:"overlay"`

	// Configurações de Mapa
	Map MapSettings `json:"map"`

	// Configurações de Vídeo
	Video VideoSettings `json:"video"`

	// Configurações de Performance
	Performance PerformanceSettings `json:"performance"`

	// Configurações de UI
	UI UISettings `json:"ui"`
}

type OverlaySettings struct {
	// Estilo do overlay
	Style string `json:"style"` // "modern", "classic", "minimal"

	// Posição do overlay
	Position string `json:"position"` // "bottom-left", "bottom-right", "top-left", "top-right"

	// Tamanho do overlay (escala)
	Scale float64 `json:"scale"` // 0.5 - 2.0

	// Transparência
	Opacity float64 `json:"opacity"` // 0.0 - 1.0

	// Cores customizadas
	Colors OverlayColors `json:"colors"`

	// Elementos visíveis
	Elements OverlayElements `json:"elements"`

	// Qualidade de saída
	Quality string `json:"quality"` // "low", "medium", "high", "ultra"

	// Margem do overlay
	Margin int `json:"margin"` // pixels
}

type OverlayColors struct {
	Primary    string `json:"primary"`    // Cor principal
	Secondary  string `json:"secondary"`  // Cor secundária
	Accent     string `json:"accent"`     // Cor de destaque
	Text       string `json:"text"`       // Cor do texto
	Background string `json:"background"` // Cor de fundo
}

type OverlayElements struct {
	Speedometer  bool `json:"speedometer"`
	Compass      bool `json:"compass"`
	DigitalSpeed bool `json:"digital_speed"`
	Altitude     bool `json:"altitude"`
	Distance     bool `json:"distance"`
	Time         bool `json:"time"`
	Heartrate    bool `json:"heartrate"`
	Power        bool `json:"power"`
	Cadence      bool `json:"cadence"`
}

type MapSettings struct {
	// Densidade padrão de marcadores GPS
	DefaultDensity string `json:"default_density"` // "low", "medium", "high", "ultra_high"

	// Estilo do mapa
	TileProvider string `json:"tile_provider"` // "openstreetmap", "satellite", "terrain"

	// Zoom automático
	AutoZoom bool `json:"auto_zoom"`

	// Cores do trajeto
	TrajectoryColors TrajectoryColors `json:"trajectory_colors"`

	// Cache de tiles
	CacheTiles bool `json:"cache_tiles"`
}

type TrajectoryColors struct {
	VerySlow string `json:"very_slow"` // < 8 km/h
	Slow     string `json:"slow"`      // 8-15 km/h
	Medium   string `json:"medium"`    // 15-25 km/h
	Fast     string `json:"fast"`      // 25-40 km/h
	VeryFast string `json:"very_fast"` // > 40 km/h
}

type VideoSettings struct {
	// Qualidade de codificação padrão
	DefaultCRF int `json:"default_crf"` // 18-28 (menor = melhor qualidade)

	// Preset de velocidade do FFmpeg
	Preset string `json:"preset"` // "ultrafast", "fast", "medium", "slow"

	// Formato de saída
	OutputFormat string `json:"output_format"` // "mp4", "mov", "avi"

	// Pasta de saída personalizada
	OutputDirectory string `json:"output_directory"`

	// Manter arquivo original
	KeepOriginal bool `json:"keep_original"`

	// Processamento em lote
	BatchProcessing bool `json:"batch_processing"`
}

type PerformanceSettings struct {
	// Usar cache para dados GPS
	UseGPSCache bool `json:"use_gps_cache"`

	// Usar cache para overlays
	UseOverlayCache bool `json:"use_overlay_cache"`

	// Número de workers para processamento paralelo
	WorkerCount int `json:"worker_count"`

	// Limite de memória (MB)
	MemoryLimit int `json:"memory_limit"`

	// Limpeza automática de cache
	AutoCleanup bool `json:"auto_cleanup"`

	// Dias para manter cache
	CacheRetentionDays int `json:"cache_retention_days"`
}

type UISettings struct {
	// Tema
	Theme string `json:"theme"` // "dark", "light", "auto"

	// Idioma
	Language string `json:"language"` // "pt-BR", "en-US", "es-ES"

	// Animações
	EnableAnimations bool `json:"enable_animations"`

	// Notificações
	EnableNotifications bool `json:"enable_notifications"`

	// Auto-save de configurações
	AutoSave bool `json:"auto_save"`

	// Densidade da interface
	Density string `json:"density"` // "compact", "comfortable", "spacious"
}

type SettingsManager struct {
	settingsPath string
	settings     *AppSettings
}

func NewSettingsManager() *SettingsManager {
	homeDir, _ := os.UserHomeDir()
	settingsPath := filepath.Join(homeDir, ".strava-overlay", "settings.json")

	sm := &SettingsManager{
		settingsPath: settingsPath,
		settings:     getDefaultSettings(),
	}

	sm.Load()
	return sm
}

func getDefaultSettings() *AppSettings {
	return &AppSettings{
		Overlay: OverlaySettings{
			Style:    "modern",
			Position: "bottom-right",
			Scale:    1.0,
			Opacity:  0.9,
			Colors: OverlayColors{
				Primary:    "#58a6ff",
				Secondary:  "#7d8590",
				Accent:     "#56d364",
				Text:       "#e6edf3",
				Background: "#0d1117",
			},
			Elements: OverlayElements{
				Speedometer:  true,
				Compass:      true,
				DigitalSpeed: true,
				Altitude:     true,
				Distance:     false,
				Time:         false,
				Heartrate:    false,
				Power:        false,
				Cadence:      false,
			},
			Quality: "high",
			Margin:  10,
		},
		Map: MapSettings{
			DefaultDensity: "medium",
			TileProvider:   "openstreetmap",
			AutoZoom:       true,
			TrajectoryColors: TrajectoryColors{
				VerySlow: "#6c757d",
				Slow:     "#28a745",
				Medium:   "#ffc107",
				Fast:     "#fd7e14",
				VeryFast: "#dc3545",
			},
			CacheTiles: true,
		},
		Video: VideoSettings{
			DefaultCRF:      18,
			Preset:          "fast",
			OutputFormat:    "mp4",
			OutputDirectory: "",
			KeepOriginal:    true,
			BatchProcessing: false,
		},
		Performance: PerformanceSettings{
			UseGPSCache:        true,
			UseOverlayCache:    true,
			WorkerCount:        4,
			MemoryLimit:        2048,
			AutoCleanup:        true,
			CacheRetentionDays: 30,
		},
		UI: UISettings{
			Theme:               "dark",
			Language:            "pt-BR",
			EnableAnimations:    true,
			EnableNotifications: true,
			AutoSave:            true,
			Density:             "comfortable",
		},
	}
}

func (sm *SettingsManager) Load() error {
	data, err := os.ReadFile(sm.settingsPath)
	if err != nil {
		// Se arquivo não existe, cria com configurações padrão
		return sm.Save()
	}

	return json.Unmarshal(data, sm.settings)
}

func (sm *SettingsManager) Save() error {
	os.MkdirAll(filepath.Dir(sm.settingsPath), 0755)

	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sm.settingsPath, data, 0644)
}

func (sm *SettingsManager) Get() *AppSettings {
	return sm.settings
}

func (sm *SettingsManager) Update(newSettings *AppSettings) error {
	sm.settings = newSettings
	return sm.Save()
}

func (sm *SettingsManager) UpdateOverlay(overlay OverlaySettings) error {
	sm.settings.Overlay = overlay
	return sm.Save()
}

func (sm *SettingsManager) UpdateMap(mapSettings MapSettings) error {
	sm.settings.Map = mapSettings
	return sm.Save()
}

func (sm *SettingsManager) UpdateVideo(video VideoSettings) error {
	sm.settings.Video = video
	return sm.Save()
}

func (sm *SettingsManager) UpdatePerformance(perf PerformanceSettings) error {
	sm.settings.Performance = perf
	return sm.Save()
}

func (sm *SettingsManager) UpdateUI(ui UISettings) error {
	sm.settings.UI = ui
	return sm.Save()
}

func (sm *SettingsManager) Reset() error {
	sm.settings = getDefaultSettings()
	return sm.Save()
}

func (sm *SettingsManager) Export(path string) error {
	data, err := json.MarshalIndent(sm.settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (sm *SettingsManager) Import(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var importedSettings AppSettings
	if err := json.Unmarshal(data, &importedSettings); err != nil {
		return err
	}

	sm.settings = &importedSettings
	return sm.Save()
}

// Validação de configurações
func (sm *SettingsManager) Validate() []string {
	var errors []string

	// Valida configurações do overlay
	if sm.settings.Overlay.Scale < 0.5 || sm.settings.Overlay.Scale > 2.0 {
		errors = append(errors, "Escala do overlay deve estar entre 0.5 e 2.0")
	}

	if sm.settings.Overlay.Opacity < 0.0 || sm.settings.Overlay.Opacity > 1.0 {
		errors = append(errors, "Opacidade do overlay deve estar entre 0.0 e 1.0")
	}

	// Valida configurações de vídeo
	if sm.settings.Video.DefaultCRF < 0 || sm.settings.Video.DefaultCRF > 51 {
		errors = append(errors, "CRF deve estar entre 0 e 51")
	}

	// Valida configurações de performance
	if sm.settings.Performance.WorkerCount < 1 || sm.settings.Performance.WorkerCount > 16 {
		errors = append(errors, "Número de workers deve estar entre 1 e 16")
	}

	if sm.settings.Performance.MemoryLimit < 512 {
		errors = append(errors, "Limite de memória deve ser pelo menos 512MB")
	}

	return errors
}

// Métodos para configurações específicas
func (sm *SettingsManager) GetOverlayStyle() string {
	return sm.settings.Overlay.Style
}

func (sm *SettingsManager) GetMapDensity() string {
	return sm.settings.Map.DefaultDensity
}

func (sm *SettingsManager) GetVideoQuality() int {
	return sm.settings.Video.DefaultCRF
}

func (sm *SettingsManager) IsGPSCacheEnabled() bool {
	return sm.settings.Performance.UseGPSCache
}

func (sm *SettingsManager) GetWorkerCount() int {
	return sm.settings.Performance.WorkerCount
}

func (sm *SettingsManager) GetTheme() string {
	return sm.settings.UI.Theme
}

func (sm *SettingsManager) GetLanguage() string {
	return sm.settings.UI.Language
}
