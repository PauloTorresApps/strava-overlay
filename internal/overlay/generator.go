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
		// --- MODIFICAÇÃO: Dimensões otimizadas para novo layout ---
		width:  480, // Largura aumentada para acomodar mais dados
		height: 220, // Altura ajustada
		// -----------------------------------------------------------
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
		err := g.generateDataDashboard(point, maxSpeedScale, imagePath)
		if err != nil {
			g.Cleanup()
			return nil, fmt.Errorf("erro ao gerar o frame de overlay %d: %w", i, err)
		}
		imagePaths = append(imagePaths, imagePath)
	}
	return imagePaths, nil
}

// generateDataDashboard cria um painel com todos os elementos de dados.
func (g *Generator) generateDataDashboard(point gps.GPSPoint, maxSpeed float64, outputPath string) error {
	dc := gg.NewContext(g.width, g.height)
	dc.SetRGBA(0, 0, 0, 0) // Fundo transparente
	dc.Clear()

	// Layout: Velocímetro à esquerda, painéis de dados à direita
	speedometerCX := float64(g.height) / 2.0
	speedometerCY := float64(g.height) / 2.0
	dataPanelX := speedometerCX*2 + 15

	// --- Desenho dos Componentes ---
	g.drawSpeedometer(dc, speedometerCX, speedometerCY, point.Velocity*3.6, maxSpeed)
	g.drawCompass(dc, speedometerCX, speedometerCY, point.Bearing)
	g.drawDigitalSpeed(dc, speedometerCX, speedometerCY+60, point.Velocity*3.6)

	// Painéis de dados
	panelY := 28.0
	panelSpacing := 50.0

	// Linha 1
	g.drawDataPanel(dc, dataPanelX, panelY, "❤️", "BPM", fmt.Sprintf("%.0f", point.HeartRate), point.HeartRate > 0, color.RGBA{220, 40, 80, 255})
	g.drawDataPanel(dc, dataPanelX+120, panelY, "👟", "RPM", fmt.Sprintf("%.0f", point.Cadence), point.Cadence > 0, color.RGBA{0, 200, 255, 255})
	// Linha 2
	panelY += panelSpacing
	g.drawDataPanel(dc, dataPanelX, panelY, "G", "Força-G", fmt.Sprintf("%.2f", point.GForce), true, color.RGBA{255, 180, 0, 255})
	g.drawDataPanel(dc, dataPanelX+120, panelY, "📐", "% Incl.", fmt.Sprintf("%.1f", point.Grade), true, color.RGBA{200, 200, 200, 255})
	// Linha 3
	panelY += panelSpacing
	g.drawDataPanel(dc, dataPanelX, panelY, "⛰️", "Altitude", fmt.Sprintf("%.0f m", point.Altitude), true, color.RGBA{140, 90, 255, 255})
	g.drawDataPanel(dc, dataPanelX+120, panelY, "🌡️", "Temp.", fmt.Sprintf("%.0f°", point.Temp), point.Temp != 0, color.RGBA{255, 100, 0, 255})
	// Linha 4 - Distância
	panelY += panelSpacing
	g.drawDistancePanel(dc, dataPanelX, panelY, "📏", "Distância", fmt.Sprintf("%.2f km", point.Distance/1000), point.Distance > 0)

	return dc.SavePNG(outputPath)
}

// drawDataPanel desenha um widget de dados genérico.
func (g *Generator) drawDataPanel(dc *gg.Context, x, y float64, icon, label, value string, available bool, iconColor color.Color) {
	if !available {
		value = "- -"
		dc.SetRGBA(1, 1, 1, 0.4) // Cor esmaecida se não estiver disponível
	} else {
		dc.SetRGBA(1, 1, 1, 0.9)
	}

	// Ícone
	g.loadFont(dc, 20)
	dc.SetColor(iconColor)
	dc.DrawString(icon, x, y)

	// Valor
	g.loadFont(dc, 24)
	dc.SetRGBA(1, 1, 1, 0.9)
	dc.DrawString(value, x+32, y)

	// Rótulo
	g.loadFont(dc, 12)
	dc.SetRGBA(1, 1, 1, 0.7)
	dc.DrawString(label, x+32, y+14)
}

// drawDistancePanel é um painel maior para a distância.
func (g *Generator) drawDistancePanel(dc *gg.Context, x, y float64, icon, label, value string, available bool) {
	if !available {
		value = "- -"
		dc.SetRGBA(1, 1, 1, 0.4)
	} else {
		dc.SetRGBA(1, 1, 1, 0.9)
	}
	// Ícone
	g.loadFont(dc, 20)
	dc.SetRGB(0.8, 0.8, 0.8)
	dc.DrawString(icon, x, y)

	// Valor
	g.loadFont(dc, 24)
	dc.SetRGBA(1, 1, 1, 0.9)
	dc.DrawString(value, x+32, y)

	// Rótulo
	g.loadFont(dc, 12)
	dc.SetRGBA(1, 1, 1, 0.7)
	dc.DrawString(label, x+32, y+14)
}

// drawSpeedometer desenha o arco do velocímetro e as marcações.
func (g *Generator) drawSpeedometer(dc *gg.Context, cx, cy float64, speed, maxSpeed float64) {
	radius := 85.0
	lineWidth := 18.0
	progressWidth := 16.0
	fontSize := 12.0
	textOffset := 20.0

	startAngle := gg.Radians(135)
	totalArc := gg.Radians(270)
	g.loadFont(dc, fontSize)

	// Círculo de fundo
	dc.SetLineWidth(lineWidth)
	dc.SetRGBA(0.1, 0.1, 0.1, 0.6)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()

	// Arco de progresso
	dc.SetLineWidth(progressWidth)
	progress := speed / maxSpeed
	if progress > 1 {
		progress = 1
	}
	progressAngle := startAngle + (totalArc * progress)
	gradient := gg.NewLinearGradient(cx-radius, cy, cx+radius, cy)
	// --- MODIFICAÇÃO: Ajuste de cores do gradiente do velocímetro ---
	gradient.AddColorStop(0, color.RGBA{R: 60, G: 220, B: 60, A: 220})   // Verde para velocidades baixas
	gradient.AddColorStop(0.7, color.RGBA{R: 255, G: 200, B: 0, A: 220}) // Amarelo para velocidades médias
	gradient.AddColorStop(1, color.RGBA{R: 220, G: 30, B: 30, A: 220})   // Vermelho para velocidades altas
	// -----------------------------------------------------------------
	dc.SetStrokeStyle(gradient)
	dc.DrawArc(cx, cy, radius, startAngle, progressAngle)
	dc.Stroke()

	// Marcadores e números
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

			textX := cx + (radius-textOffset)*math.Cos(angle)
			textY := cy + (radius-textOffset)*math.Sin(angle)
			if g.fontLoaded {
				dc.DrawStringAnchored(fmt.Sprintf("%.0f", i), textX, textY, 0.5, 0.5)
			}
		}
	}
}

// drawCompass desenha a bússola no centro do velocímetro.
func (g *Generator) drawCompass(dc *gg.Context, cx, cy float64, bearing float64) {
	radius := 40.0
	fontSize := 11.0

	g.loadFont(dc, fontSize)

	dc.SetRGBA(0.1, 0.1, 0.1, 0.7)
	dc.DrawCircle(cx, cy, radius)
	dc.Fill()

	dc.SetLineWidth(1.5)
	dc.SetRGBA(0.5, 0.5, 0.5, 1)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()

	if g.fontLoaded {
		dc.SetRGBA(1, 1, 1, 0.9)
		cardinals := map[string]float64{"N": 270, "E": 0, "S": 90, "W": 180}
		for text, angle := range cardinals {
			rad := gg.Radians(angle)
			textX := cx + (radius-12)*math.Cos(rad)
			textY := cy + (radius-12)*math.Sin(rad)
			dc.DrawStringAnchored(text, textX, textY, 0.5, 0.5)
		}
	}

	dc.Push()
	dc.Translate(cx, cy)
	dc.Rotate(gg.Radians(bearing))

	// Ponteiro da bússola ajustado
	dc.SetRGBA(1, 0.2, 0.2, 1)
	dc.MoveTo(0, -radius+8)
	dc.LineTo(-7, 0)
	dc.LineTo(7, 0)
	dc.ClosePath()
	dc.Fill()

	dc.SetRGBA(1, 1, 1, 1)
	dc.MoveTo(0, radius-8)
	dc.LineTo(-7, 0)
	dc.LineTo(7, 0)
	dc.ClosePath()
	dc.Fill()

	dc.Pop()
}

// drawDigitalSpeed desenha o valor numérico da velocidade na parte inferior.
func (g *Generator) drawDigitalSpeed(dc *gg.Context, cx, cy float64, speed float64) {
	g.loadFont(dc, 26)
	dc.SetRGB255(255, 255, 255)
	dc.DrawStringAnchored(fmt.Sprintf("%.1f", speed), cx, cy, 0.5, 0.5)

	g.loadFont(dc, 13)
	dc.SetRGB(0.8, 0.8, 0.8)
	dc.DrawStringAnchored("km/h", cx, cy+18, 0.5, 0.5)
}

// Cleanup remove o diretório temporário.
func (g *Generator) Cleanup() {
	if g.tempDir != "" && g.tempDir != "." {
		os.RemoveAll(g.tempDir)
	}
}
