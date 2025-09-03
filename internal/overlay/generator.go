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
		width:      400, // Largura mantida
		height:     450, // Altura aumentada para evitar cortes
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
		err := g.generateCircularOverlay(point, maxSpeedScale, imagePath)
		if err != nil {
			g.Cleanup()
			return nil, fmt.Errorf("erro ao gerar o frame de overlay %d: %w", i, err)
		}
		imagePaths = append(imagePaths, imagePath)
	}
	return imagePaths, nil
}

// generateCircularOverlay cria um painel circular único com todos os elementos.
func (g *Generator) generateCircularOverlay(point gps.GPSPoint, maxSpeed float64, outputPath string) error {
	dc := gg.NewContext(g.width, g.height)
	dc.SetRGBA(0, 0, 0, 0) // Fundo transparente para o PNG
	dc.Clear()

	centerX := float64(g.width) / 2
	centerY := float64(g.width) / 2

	digitalSpeedY := float64(g.height) - 60.0

	// --- Desenha os Componentes ---
	g.drawSpeedometerArc(dc, centerX, centerY, point.Velocity*3.6, maxSpeed)
	g.drawCompass(dc, centerX, centerY, point.Bearing)
	g.drawDigitalSpeed(dc, centerX, digitalSpeedY, point.Velocity*3.6)

	return dc.SavePNG(outputPath)
}

// drawSpeedometerArc desenha o arco do velocímetro e as marcações.
func (g *Generator) drawSpeedometerArc(dc *gg.Context, cx, cy float64, speed, maxSpeed float64) {
	radius := 150.0
	startAngle := gg.Radians(135)
	totalArc := gg.Radians(270)
	g.loadFont(dc, 18)

	// 1. Cria a máscara para o efeito de desvanecimento em um contexto temporário.
	maskContext := gg.NewContext(g.width, g.height)
	maskGradient := gg.NewLinearGradient(cx, cy+radius-50, cx, cy+radius+25)
	maskGradient.AddColorStop(0, color.White) // Manter (opaco)
	maskGradient.AddColorStop(1, color.Black) // Apagar (transparente na máscara)
	maskContext.SetFillStyle(maskGradient)
	maskContext.DrawRectangle(0, 0, float64(g.width), float64(g.height))
	maskContext.Fill()

	// 2. Desenha o círculo de fundo usando a máscara.
	dc.Push() // Salva o estado atual do contexto (sem máscara)
	dc.SetMask(maskContext.AsMask())
	dc.SetLineWidth(25)
	dc.SetRGBA(0.1, 0.1, 0.1, 0.5) // Cor sólida com 50% de transparência
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()
	dc.Pop() // Restaura o estado anterior, removendo a máscara.

	// 3. Desenha o arco de progresso por cima.
	dc.SetLineWidth(22)
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

	// 4. Marcadores e números
	dc.SetLineWidth(3)
	dc.SetRGBA(1, 1, 1, 0.9)
	for i := 0.0; i <= maxSpeed; i += 10 {
		angle := startAngle + (totalArc * (i / maxSpeed))
		if i/maxSpeed <= 1.0 {
			x1 := cx + (radius-16)*math.Cos(angle)
			y1 := cy + (radius-16)*math.Sin(angle)
			x2 := cx + (radius-8)*math.Cos(angle)
			y2 := cy + (radius-8)*math.Sin(angle)
			dc.DrawLine(x1, y1, x2, y2)
			dc.Stroke()

			textX := cx + (radius-35)*math.Cos(angle)
			textY := cy + (radius-35)*math.Sin(angle)
			if g.fontLoaded {
				dc.DrawStringAnchored(fmt.Sprintf("%.0f", i), textX, textY, 0.5, 0.5)
			}
		}
	}
}

// drawCompass desenha a bússola no centro do velocímetro.
func (g *Generator) drawCompass(dc *gg.Context, cx, cy float64, bearing float64) {
	radius := 70.0
	g.loadFont(dc, 16)

	dc.SetRGBA(0.1, 0.1, 0.1, 0.7)
	dc.DrawCircle(cx, cy, radius)
	dc.Fill()

	dc.SetLineWidth(2)
	dc.SetRGBA(0.5, 0.5, 0.5, 1)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()

	if g.fontLoaded {
		dc.SetRGBA(1, 1, 1, 0.9)
		cardinals := map[string]float64{"N": 270, "E": 0, "S": 90, "W": 180}
		for text, angle := range cardinals {
			rad := gg.Radians(angle)
			textX := cx + (radius-15)*math.Cos(rad)
			textY := cy + (radius-15)*math.Sin(rad)
			dc.DrawStringAnchored(text, textX, textY, 0.5, 0.5)
		}
	}

	dc.Push()
	dc.Translate(cx, cy)
	dc.Rotate(gg.Radians(bearing))

	dc.SetRGBA(1, 0.2, 0.2, 1)
	dc.MoveTo(0, -radius+15)
	dc.LineTo(-12, 0)
	dc.LineTo(12, 0)
	dc.ClosePath()
	dc.Fill()

	dc.SetRGBA(1, 1, 1, 1)
	dc.MoveTo(0, radius-15)
	dc.LineTo(-12, 0)
	dc.LineTo(12, 0)
	dc.ClosePath()
	dc.Fill()

	dc.Pop()
}

// drawDigitalSpeed desenha o valor numérico da velocidade na parte inferior.
func (g *Generator) drawDigitalSpeed(dc *gg.Context, cx, cy float64, speed float64) {
	g.loadFont(dc, 35)
	dc.SetRGBA(1, 1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.1f", speed), cx, cy, 0.5, 0.5)

	g.loadFont(dc, 18)
	dc.SetRGBA(0.8, 0.8, 0.8, 1)
	dc.DrawStringAnchored("km/h", cx, cy+30, 0.5, 0.5)
}

// Cleanup remove o diretório temporário.
func (g *Generator) Cleanup() {
	if g.tempDir != "" && g.tempDir != "." {
		os.RemoveAll(g.tempDir)
	}
}
