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
		width:      340,
		height:     340,
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

// generateEnhancedOverlay cria o overlay principal com velocímetro e widgets
func (g *Generator) generateEnhancedOverlay(point gps.GPSPoint, maxSpeed float64, outputPath string) error {
	dc := gg.NewContext(g.width, g.height)
	dc.SetRGBA(0, 0, 0, 0) // Fundo transparente
	dc.Clear()

	radius := 95.0

	// Centro do velocímetro
	centerX := float64(g.width) - radius - 15.0
	centerY := float64(g.height) / 2

	// 1. Desenha o velocímetro principal
	g.drawMainSpeedometer(dc, centerX, centerY, point.Velocity*3.6, maxSpeed, point, radius)

	// 2. Desenha os widgets empilhados à esquerda
	g.drawStackedWidgets(dc, point, centerX, centerY, radius)

	return dc.SavePNG(outputPath)
}

// drawStackedWidgets desenha os widgets empilhados à esquerda com fundo
func (g *Generator) drawStackedWidgets(dc *gg.Context, point gps.GPSPoint, speedometerCenterX, speedometerCenterY, speedometerRadius float64) {
	spacing := 35.0
	widgetHeight := 25.0
	padding := 10.0

	// Container mais estreito e mais distante do velocímetro
	totalHeight := (spacing * 3) + widgetHeight + (padding * 2)
	containerWidth := 95.0 // Reduzido de 120px

	// Posiciona com maior distância do velocímetro
	containerX := 10.0
	containerY := speedometerCenterY - (totalHeight / 2)

	// Desenha fundo escuro com transparência
	dc.SetRGBA(0.1, 0.1, 0.1, 0.5)
	dc.DrawRoundedRectangle(containerX, containerY, containerWidth, totalHeight, 8)
	dc.Fill()

	// Posição inicial para os widgets
	startX := containerX + padding
	startY := containerY + padding

	// G-Force
	gForce := math.Abs(point.GForce)
	g.drawTextWidget(dc, startX, startY, "G-FORCE", fmt.Sprintf("%.2f G", gForce),
		color.RGBA{R: 255, G: 100, B: 50, A: 255})

	// Altitude
	startY += spacing
	g.drawTextWidget(dc, startX, startY, "ALTITUDE", fmt.Sprintf("%.0f m", point.Altitude),
		color.RGBA{R: 100, G: 255, B: 150, A: 255})

	// Cadência estimada
	startY += spacing
	cadence := g.estimateCadence(point)
	g.drawTextWidget(dc, startX, startY, "CADENCE", fmt.Sprintf("%.0f RPM", cadence),
		color.RGBA{R: 255, G: 200, B: 50, A: 255})

	// BPM estimado
	startY += spacing
	heartRate := g.estimateHeartRate(point)
	g.drawTextWidget(dc, startX, startY, "HEART", fmt.Sprintf("%.0f BPM", heartRate),
		color.RGBA{R: 255, G: 50, B: 200, A: 255})
}

// drawTextWidget desenha um widget de texto individual
func (g *Generator) drawTextWidget(dc *gg.Context, x, y float64, label, value string, textColor color.RGBA) {
	// Label menor
	g.loadFont(dc, 9)
	dc.SetRGBA(0.6, 0.6, 0.6, 0.9)
	dc.DrawString(label, x, y)

	// Valor maior e colorido
	g.loadFont(dc, 16)
	dc.SetRGBA(float64(textColor.R)/255, float64(textColor.G)/255, float64(textColor.B)/255, 1.0)
	dc.DrawString(value, x, y+15)
}

// estimateCadence estima a cadência baseada na velocidade
func (g *Generator) estimateCadence(p gps.GPSPoint) float64 {
	if p.Velocity < 1.0 {
		return 0
	}
	speed_kmh := p.Velocity * 3.6
	return math.Min(120, 70+(speed_kmh*1.5))
}

// estimateHeartRate estima a frequência cardíaca
func (g *Generator) estimateHeartRate(p gps.GPSPoint) float64 {
	if p.Velocity < 1.0 {
		return 65
	}
	speed_kmh := p.Velocity * 3.6
	intensity := math.Min(1.0, speed_kmh/40.0)
	gForceEffect := math.Abs(p.GForce) * 10
	baseHR := 65 + (intensity * 85) + gForceEffect
	return math.Min(180, baseHR)
}

// getSpeedColor retorna a cor baseada na velocidade
func (g *Generator) getSpeedColor(speed float64) color.RGBA {
	if speed < 15 {
		return color.RGBA{R: 20, G: 180, B: 20, A: 255}
	} else if speed < 25 {
		ratio := (speed - 15) / 10
		r := uint8(20 + (255-20)*ratio)
		g := uint8(180 + (200-180)*ratio)
		return color.RGBA{R: r, G: g, B: 0, A: 255}
	} else if speed < 35 {
		ratio := (speed - 25) / 10
		g := uint8(200 - 100*ratio)
		return color.RGBA{R: 255, G: g, B: 0, A: 255}
	} else {
		return color.RGBA{R: 220, G: 30, B: 30, A: 255}
	}
}

// drawMainSpeedometer desenha o velocímetro principal
func (g *Generator) drawMainSpeedometer(dc *gg.Context, cx, cy float64, speed, maxSpeed float64, point gps.GPSPoint, radius float64) {
	fontSize := 11.0
	textOffset := 22.0

	startAngle := gg.Radians(135)
	totalArc := gg.Radians(270)
	g.loadFont(dc, fontSize)

	// 1. Máscara para efeito de desvanecimento
	maskContext := gg.NewContext(g.width, g.height)
	maskGradient := gg.NewLinearGradient(cx, cy+radius-30, cx, cy+radius+15)
	maskGradient.AddColorStop(0, color.White)
	maskGradient.AddColorStop(1, color.Black)
	maskContext.SetFillStyle(maskGradient)
	maskContext.DrawRectangle(0, 0, float64(g.width), float64(g.height))
	maskContext.Fill()

	// 2. Círculo de fundo com máscara
	dc.Push()
	dc.SetMask(maskContext.AsMask())
	dc.SetLineWidth(16.0)
	dc.SetRGBA(0.1, 0.1, 0.1, 0.5)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()
	dc.Pop()

	// 3. Desenha os traços de velocidade
	for kmh := 0.0; kmh <= maxSpeed; kmh++ {
		angle := startAngle + (totalArc * (kmh / maxSpeed))

		var tickLength, tickWidth float64
		isMajor := int(kmh)%10 == 0

		if isMajor {
			tickLength = 14.0
			tickWidth = 2.5
		} else {
			tickLength = 9.0
			tickWidth = 1.0
		}

		innerRadius := radius - tickLength
		x1 := cx + innerRadius*math.Cos(angle)
		y1 := cy + innerRadius*math.Sin(angle)
		x2 := cx + radius*math.Cos(angle)
		y2 := cy + radius*math.Sin(angle)

		dc.SetLineWidth(tickWidth)
		if kmh <= speed {
			speedColor := g.getSpeedColor(kmh)
			dc.SetRGBA(float64(speedColor.R)/255, float64(speedColor.G)/255, float64(speedColor.B)/255, 0.9)
		} else {
			dc.SetRGBA(0.5, 0.5, 0.5, 0.3)
		}

		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}

	// 4. Marcadores numéricos
	dc.SetLineWidth(2)
	dc.SetRGBA(1, 1, 1, 0.9)
	for i := 0.0; i <= maxSpeed; i += 10 {
		angle := startAngle + (totalArc * (i / maxSpeed))
		if i/maxSpeed <= 1.0 && g.fontLoaded {
			textX := cx + (radius-textOffset)*math.Cos(angle)
			textY := cy + (radius-textOffset)*math.Sin(angle)
			dc.DrawStringAnchored(fmt.Sprintf("%.0f", i), textX, textY, 0.5, 0.5)
		}
	}

	// 5. Bússola interna com raio aumentado
	g.drawCompactCompass(dc, cx, cy, point.Bearing)

	// 6. Velocidade digital
	g.drawDigitalSpeed(dc, cx, cy+58, speed)
}

// drawCompactCompass desenha uma bússola compacta no centro
func (g *Generator) drawCompactCompass(dc *gg.Context, cx, cy, bearing float64) {
	radius := 42.0 // Aumentado em 10px (de 32 para 42)
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
			textX := cx + (radius-10)*math.Cos(rad)
			textY := cy + (radius-10)*math.Sin(rad)
			dc.DrawStringAnchored(text, textX, textY, 0.5, 0.5)
		}
	}

	// Agulha da bússola
	dc.Push()
	dc.Translate(cx, cy)
	dc.Rotate(gg.Radians(bearing))

	needleLength := radius - 13
	needleWidth := 8.0

	// Ponta vermelha (Norte)
	dc.SetRGBA(0.9, 0.15, 0.15, 1)
	dc.MoveTo(0, -needleLength)
	dc.LineTo(-needleWidth/2, -4)
	dc.LineTo(needleWidth/2, -4)
	dc.ClosePath()
	dc.Fill()

	// Ponta branca (Sul)
	dc.SetRGBA(0.95, 0.95, 0.95, 1)
	dc.MoveTo(0, needleLength)
	dc.LineTo(-needleWidth/2, 4)
	dc.LineTo(needleWidth/2, 4)
	dc.ClosePath()
	dc.Fill()

	// Corpo central
	dc.SetRGBA(0.4, 0.4, 0.4, 0.8)
	dc.MoveTo(-needleWidth/2, -4)
	dc.LineTo(-1.5, -4)
	dc.LineTo(-1.5, 4)
	dc.LineTo(-needleWidth/2, 4)
	dc.ClosePath()
	dc.Fill()

	dc.SetRGBA(0.7, 0.7, 0.7, 0.8)
	dc.MoveTo(1.5, -4)
	dc.LineTo(needleWidth/2, -4)
	dc.LineTo(needleWidth/2, 4)
	dc.LineTo(1.5, 4)
	dc.ClosePath()
	dc.Fill()

	// Ponto central
	dc.SetRGBA(0.2, 0.2, 0.2, 0.9)
	dc.DrawCircle(0, 0, 2.8)
	dc.Fill()

	dc.SetRGBA(0.8, 0.8, 0.8, 0.7)
	dc.DrawCircle(0, 0, 1.6)
	dc.Fill()

	dc.Pop()
}

// drawDigitalSpeed desenha a velocidade digital
func (g *Generator) drawDigitalSpeed(dc *gg.Context, cx, cy, speed float64) {
	g.loadFont(dc, 24)
	dc.SetRGB255(0, 221, 255)
	dc.DrawStringAnchored(fmt.Sprintf("%.1f", speed), cx, cy, 0.5, 0.5)

	g.loadFont(dc, 12)
	dc.SetRGB255(0, 221, 255)
	dc.DrawStringAnchored("km/h", cx, cy+14, 0.5, 0.5)
}

// Cleanup remove o diretório temporário.
func (g *Generator) Cleanup() {
	if g.tempDir != "" && g.tempDir != "." {
		os.RemoveAll(g.tempDir)
	}
}
