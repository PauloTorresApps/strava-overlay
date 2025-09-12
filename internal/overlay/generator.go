package overlay

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"

	"strava-overlay/internal/gps"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

// Generator cria imagens de overlay.
type Generator struct {
	width, height int
	tempDir       string
	fontLoaded    bool
	fontPath      string
}

// WidgetConfig define as configurações de um widget orbital
type WidgetConfig struct {
	Color     color.Color
	Label     string
	Unit      string
	ValueFunc func(gps.GPSPoint) float64
	MaxValue  float64
	Position  float64 // Ângulo em graus (0-360)
	Size      float64 // Raio do widget
}

// NewGenerator cria um novo gerador de overlay.
func NewGenerator() *Generator {
	tempDir, err := os.MkdirTemp("", "strava_overlays_*")
	if err != nil {
		log.Printf("Não foi possível criar o diretório temporário: %v", err)
		tempDir = "." // Usa o diretório atual como fallback
	}

	var fontPath string
	switch runtime.GOOS {
	case "windows":
		fontPath = "C:/Windows/Fonts/arial.ttf"
	case "darwin":
		fontPath = "/System/Library/Fonts/Supplemental/Arial.ttf"
	default: // linux
		commonFonts := []string{
			"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
			"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
			"/usr/share/fonts/truetype/ubuntu/Ubuntu-R.ttf",
		}
		for _, fp := range commonFonts {
			if _, err := os.Stat(fp); err == nil {
				fontPath = fp
				break
			}
		}
	}

	return &Generator{
		// Dimensões aumentadas para acomodar os widgets orbitais
		width:      450, // Aumentado de 300 para 450
		height:     450, // Aumentado de 300 para 450
		tempDir:    tempDir,
		fontLoaded: false,
		fontPath:   fontPath,
	}
}

// loadFont tenta carregar uma fonte do sistema.
func (g *Generator) loadFont(dc *gg.Context, size float64) {
	if g.fontLoaded && g.fontPath != "" {
		if err := dc.LoadFontFace(g.fontPath, size); err == nil {
			return
		}
	}

	if g.fontPath == "" {
		dc.SetFontFace(basicfont.Face7x13)
		g.fontLoaded = false
		return
	}

	err := dc.LoadFontFace(g.fontPath, size)
	if err != nil {
		log.Printf("Não foi possível carregar a fonte do sistema de %s: %v. Usando fonte básica.", g.fontPath, err)
		dc.SetFontFace(basicfont.Face7x13)
		g.fontLoaded = false
	} else {
		g.fontLoaded = true
	}
}

// GenerateOverlaySequence cria uma sequência de imagens PNG para o overlay.
func (g *Generator) GenerateOverlaySequence(points []gps.GPSPoint, frameRate float64) ([]string, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("nenhum ponto GPS fornecido")
	}

	maxSpeed := 0.0
	for _, point := range points {
		speed := point.Velocity * 3.6 // m/s para km/h
		if speed > maxSpeed {
			maxSpeed = speed
		}
	}
	maxSpeedScale := math.Ceil(maxSpeed/10) * 10
	if maxSpeedScale < 50 {
		maxSpeedScale = 50 // Escala mínima
	}

	var imagePaths []string
	for i, point := range points {
		imagePath := filepath.Join(g.tempDir, fmt.Sprintf("overlay_%06d.png", i))
		err := g.generateEnhancedOverlay(point, maxSpeedScale, imagePath)
		if err != nil {
			g.Cleanup()
			return nil, fmt.Errorf("erro ao gerar o frame de overlay %d: %w", i, err)
		}
		imagePaths = append(imagePaths, imagePath)
	}
	return imagePaths, nil
}

// generateEnhancedOverlay cria o overlay principal com velocímetro central e widgets orbitais
func (g *Generator) generateEnhancedOverlay(point gps.GPSPoint, maxSpeed float64, outputPath string) error {
	dc := gg.NewContext(g.width, g.height)
	dc.SetRGBA(0, 0, 0, 0) // Fundo transparente
	dc.Clear()

	centerX := float64(g.width) / 2
	centerY := float64(g.height) / 2

	// 1. Desenha o velocímetro principal no centro
	g.drawMainSpeedometer(dc, centerX, centerY, point.Velocity*3.6, maxSpeed, point)

	// 2. Desenha os widgets orbitais
	g.drawOrbitalWidgets(dc, centerX, centerY, point)

	return dc.SavePNG(outputPath)
}

// drawMainSpeedometer desenha o velocímetro principal no centro
func (g *Generator) drawMainSpeedometer(dc *gg.Context, cx, cy float64, speed, maxSpeed float64, point gps.GPSPoint) {
	// Configurações do velocímetro principal
	radius := 90.0        // Raio principal
	lineWidth := 18.0     // Largura da linha de fundo
	progressWidth := 16.0 // Largura da linha de progresso
	fontSize := 12.0      // Tamanho da fonte para marcações
	textOffset := 22.0    // Deslocamento do texto

	startAngle := gg.Radians(135)
	totalArc := gg.Radians(270)
	g.loadFont(dc, fontSize)

	// 1. Máscara para efeito de desvanecimento
	maskContext := gg.NewContext(g.width, g.height)
	maskGradient := gg.NewLinearGradient(cx, cy+radius-35, cx, cy+radius+15)
	maskGradient.AddColorStop(0, color.White)
	maskGradient.AddColorStop(1, color.Black)
	maskContext.SetFillStyle(maskGradient)
	maskContext.DrawRectangle(0, 0, float64(g.width), float64(g.height))
	maskContext.Fill()

	// 2. Círculo de fundo com máscara
	dc.Push()
	dc.SetMask(maskContext.AsMask())
	dc.SetLineWidth(lineWidth)
	dc.SetRGBA(0.1, 0.1, 0.1, 0.5)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()
	dc.Pop()

	// 3. Arco de progresso da velocidade
	dc.SetLineWidth(progressWidth)
	progress := speed / maxSpeed
	if progress > 1 {
		progress = 1
	}
	progressAngle := startAngle + (totalArc * progress)
	gradient := gg.NewLinearGradient(cx-radius, cy, cx+radius, cy)
	gradient.AddColorStop(0, color.RGBA{R: 20, G: 180, B: 20, A: 204})
	gradient.AddColorStop(0.7, color.RGBA{R: 255, G: 200, B: 0, A: 204})
	gradient.AddColorStop(1, color.RGBA{R: 220, G: 30, B: 30, A: 204})
	dc.SetStrokeStyle(gradient)
	dc.DrawArc(cx, cy, radius, startAngle, progressAngle)
	dc.Stroke()

	// 4. Marcadores e números do velocímetro
	dc.SetLineWidth(2)
	dc.SetRGBA(1, 1, 1, 0.9)
	for i := 0.0; i <= maxSpeed; i += 10 {
		angle := startAngle + (totalArc * (i / maxSpeed))
		if i/maxSpeed <= 1.0 {
			x1 := cx + (radius-10)*math.Cos(angle)
			y1 := cy + (radius-10)*math.Sin(angle)
			x2 := cx + (radius-5)*math.Cos(angle)
			y2 := cy + (radius-5)*math.Sin(angle)
			dc.DrawLine(x1, y1, x2, y2)
			dc.Stroke()

			if g.fontLoaded {
				textX := cx + (radius-textOffset)*math.Cos(angle)
				textY := cy + (radius-textOffset)*math.Sin(angle)
				dc.DrawStringAnchored(fmt.Sprintf("%.0f", i), textX, textY, 0.5, 0.5)
			}
		}
	}

	// 5. Bússola interna
	g.drawCompactCompass(dc, cx, cy, point.Bearing) // Usa o bearing real do ponto GPS

	// 6. Velocidade digital
	g.drawDigitalSpeed(dc, cx, cy+55, speed)
}

// drawCompactCompass desenha uma bússola compacta no centro
func (g *Generator) drawCompactCompass(dc *gg.Context, cx, cy, bearing float64) {
	radius := 35.0
	fontSize := 10.0

	g.loadFont(dc, fontSize)

	// Fundo da bússola
	dc.SetRGBA(0.1, 0.1, 0.1, 0.7)
	dc.DrawCircle(cx, cy, radius)
	dc.Fill()

	// Borda da bússola
	dc.SetLineWidth(1.0)
	dc.SetRGBA(0.5, 0.5, 0.5, 1)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()

	// Pontos cardeais
	if g.fontLoaded {
		dc.SetRGBA(1, 1, 1, 0.9)
		cardinals := map[string]float64{"N": 270, "E": 0, "S": 90, "W": 180}
		for text, angle := range cardinals {
			rad := gg.Radians(angle)
			textX := cx + (radius-8)*math.Cos(rad)
			textY := cy + (radius-8)*math.Sin(rad)
			dc.DrawStringAnchored(text, textX, textY, 0.5, 0.5)
		}
	}

	// Ponteiro da bússola
	dc.Push()
	dc.Translate(cx, cy)
	dc.Rotate(gg.Radians(bearing))

	dc.SetRGBA(1, 0.2, 0.2, 1)
	dc.MoveTo(0, -radius+8)
	dc.LineTo(-6, 0)
	dc.LineTo(6, 0)
	dc.ClosePath()
	dc.Fill()

	dc.Pop()
}

// drawDigitalSpeed desenha a velocidade digital
func (g *Generator) drawDigitalSpeed(dc *gg.Context, cx, cy, speed float64) {
	g.loadFont(dc, 24)
	dc.SetRGB255(0, 221, 255) // Azul néon
	dc.DrawStringAnchored(fmt.Sprintf("%.1f", speed), cx, cy, 0.5, 0.5)

	g.loadFont(dc, 12)
	dc.SetRGB255(0, 221, 255)
	dc.DrawStringAnchored("km/h", cx, cy+15, 0.5, 0.5)
}

// drawOrbitalWidgets desenha os widgets pequenos ao redor do velocímetro principal
func (g *Generator) drawOrbitalWidgets(dc *gg.Context, centerX, centerY float64, point gps.GPSPoint) {
	// Distância orbital dos widgets em relação ao centro
	orbitRadius := 145.0

	// Configuração dos widgets orbitais
	widgets := []WidgetConfig{
		{
			Color: color.RGBA{R: 255, G: 69, B: 0, A: 255}, // Laranja vibrante
			Label: "G-FORCE",
			Unit:  "G",
			ValueFunc: func(p gps.GPSPoint) float64 {
				return math.Abs(p.GForce) // Força G absoluta
			},
			MaxValue: 2.0, // Máximo 2G para atividades normais
			Position: 45,  // 45 graus (superior direito)
			Size:     45,  // Raio do widget
		},
		{
			Color: color.RGBA{R: 50, G: 205, B: 50, A: 255}, // Verde lima
			Label: "CADENCE",
			Unit:  "RPM",
			ValueFunc: func(p gps.GPSPoint) float64 {
				// Estimativa de cadência baseada na velocidade
				// Para ciclismo: cadência média = (velocidade_kmh * 60) / (circunferência_roda * relação_marcha)
				// Aproximação simples: 80-100 RPM para velocidades normais
				if p.Velocity < 1.0 { // Parado
					return 0
				}
				speed_kmh := p.Velocity * 3.6
				// Fórmula aproximada: cadência base + variação baseada na velocidade
				return math.Min(120, 70+(speed_kmh*1.5))
			},
			MaxValue: 120, // RPM máximo
			Position: 135, // 135 graus (superior esquerdo)
			Size:     45,
		},
		{
			Color: color.RGBA{R: 255, G: 20, B: 147, A: 255}, // Rosa vibrante
			Label: "ELEVATION",
			Unit:  "m",
			ValueFunc: func(p gps.GPSPoint) float64 {
				return p.Altitude
			},
			MaxValue: 2000, // Máximo 2000m de altitude
			Position: 225,  // 225 graus (inferior esquerdo)
			Size:     45,
		},
		{
			Color: color.RGBA{R: 255, G: 0, B: 255, A: 255}, // Magenta
			Label: "HEART",
			Unit:  "BPM",
			ValueFunc: func(p gps.GPSPoint) float64 {
				// Estimativa de frequência cardíaca baseada na velocidade e G-force
				// Fórmula aproximada para ciclismo recreativo
				if p.Velocity < 1.0 { // Parado
					return 65 // Batimentos de repouso
				}
				speed_kmh := p.Velocity * 3.6
				intensity := math.Min(1.0, speed_kmh/40.0) // Normaliza até 40 km/h
				gForceEffect := math.Abs(p.GForce) * 10    // G-force adiciona intensidade

				// BPM = repouso + (intensidade * variação) + efeito G-force
				baseHR := 65 + (intensity * 85) + gForceEffect // 65-150 BPM + G-force
				return math.Min(180, baseHR)                   // Máximo 180 BPM
			},
			MaxValue: 180, // BPM máximo
			Position: 315, // 315 graus (inferior direito)
			Size:     45,
		},
	}

	// Desenha cada widget
	for _, widget := range widgets {
		g.drawOrbitalWidget(dc, centerX, centerY, orbitRadius, widget, point)
	}
}

// drawOrbitalWidget desenha um widget individual em posição orbital
func (g *Generator) drawOrbitalWidget(dc *gg.Context, centerX, centerY, orbitRadius float64, config WidgetConfig, point gps.GPSPoint) {
	// Calcula posição do widget
	angle := gg.Radians(config.Position)
	widgetX := centerX + orbitRadius*math.Cos(angle)
	widgetY := centerY + orbitRadius*math.Sin(angle)

	// Obtém o valor atual
	currentValue := config.ValueFunc(point)
	progress := currentValue / config.MaxValue
	if progress > 1 {
		progress = 1
	}

	// 1. Fundo do widget (círculo escuro com transparência)
	dc.SetRGBA(0.1, 0.1, 0.1, 0.6) // 60% transparência
	dc.DrawCircle(widgetX, widgetY, config.Size)
	dc.Fill()

	// 2. Borda colorida do widget
	dc.SetLineWidth(3.0)
	colorR, colorG, colorB, colorA := config.Color.RGBA()
	dc.SetRGBA(float64(colorR)/65535, float64(colorG)/65535, float64(colorB)/65535, float64(colorA)/65535)
	dc.DrawCircle(widgetX, widgetY, config.Size)
	dc.Stroke()

	// 3. Arco de progresso interno
	if progress > 0 {
		dc.SetLineWidth(4.0)
		dc.SetRGBA(float64(colorR)/65535, float64(colorG)/65535, float64(colorB)/65535, 0.8)
		startArc := gg.Radians(-90) // Começa no topo
		endArc := startArc + (gg.Radians(360) * progress)
		dc.DrawArc(widgetX, widgetY, config.Size-8, startArc, endArc)
		dc.Stroke()
	}

	// 4. Texto do valor
	g.loadFont(dc, 14)
	dc.SetRGBA(1, 1, 1, 1) // Branco sólido

	// Formata o valor baseado no tipo
	var valueText string
	if config.Unit == "G" {
		valueText = fmt.Sprintf("%.2f", currentValue)
	} else if config.Unit == "m" {
		valueText = fmt.Sprintf("%.0f", currentValue)
	} else {
		valueText = fmt.Sprintf("%.0f", currentValue)
	}

	dc.DrawStringAnchored(valueText, widgetX, widgetY-5, 0.5, 0.5)

	// 5. Label e unidade
	g.loadFont(dc, 8)
	dc.SetRGBA(0.9, 0.9, 0.9, 1)
	dc.DrawStringAnchored(config.Label, widgetX, widgetY+8, 0.5, 0.5)

	g.loadFont(dc, 7)
	dc.SetRGBA(0.7, 0.7, 0.7, 1)
	dc.DrawStringAnchored(config.Unit, widgetX, widgetY+18, 0.5, 0.5)
}

// Cleanup remove o diretório temporário.
func (g *Generator) Cleanup() {
	if g.tempDir != "" && g.tempDir != "." {
		os.RemoveAll(g.tempDir)
	}
}
