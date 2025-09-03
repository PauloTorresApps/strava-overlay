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
		// --- MODIFICAÇÃO: Dimensões otimizadas ---
		width:  300, // Largura reduzida de 400 para 300
		height: 300, // Altura ajustada para o conteúdo
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
	centerY := float64(g.width) / 2 // Mantém a proporção circular

	// --- MODIFICAÇÃO: Posição Y do texto da velocidade ajustada para baixo ---
	digitalSpeedY := centerY + 68.0
	// --------------------------------------------------------------------

	// --- Desenha os Componentes ---
	g.drawSpeedometerArc(dc, centerX, centerY, point.Velocity*3.6, maxSpeed)
	g.drawCompass(dc, centerX, centerY, point.Bearing)
	g.drawDigitalSpeed(dc, centerX, digitalSpeedY, point.Velocity*3.6)

	return dc.SavePNG(outputPath)
}

// drawSpeedometerArc desenha o arco do velocímetro e as marcações.
func (g *Generator) drawSpeedometerArc(dc *gg.Context, cx, cy float64, speed, maxSpeed float64) {
	// --- MODIFICAÇÃO: Valores ajustados para a nova escala ---
	radius := 110.0       // Raio reduzido de 150.0
	lineWidth := 20.0     // Largura da linha reduzida de 25
	progressWidth := 18.0 // Largura do progresso reduzida de 22
	fontSize := 14.0      // Tamanho da fonte reduzido de 18
	textOffset := 25.0    // Deslocamento do texto ajustado
	// --------------------------------------------------------

	startAngle := gg.Radians(135)
	totalArc := gg.Radians(270)
	g.loadFont(dc, fontSize)

	// 1. Cria a máscara para o efeito de desvanecimento.
	maskContext := gg.NewContext(g.width, g.height)
	maskGradient := gg.NewLinearGradient(cx, cy+radius-40, cx, cy+radius+20)
	maskGradient.AddColorStop(0, color.White)
	maskGradient.AddColorStop(1, color.Black)
	maskContext.SetFillStyle(maskGradient)
	maskContext.DrawRectangle(0, 0, float64(g.width), float64(g.height))
	maskContext.Fill()

	// 2. Desenha o círculo de fundo usando a máscara.
	dc.Push()
	dc.SetMask(maskContext.AsMask())
	dc.SetLineWidth(lineWidth)
	dc.SetRGBA(0.1, 0.1, 0.1, 0.5)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()
	dc.Pop()

	// 3. Desenha o arco de progresso por cima.
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

	// 4. Marcadores e números
	dc.SetLineWidth(2) // Linha mais fina para os marcadores
	dc.SetRGBA(1, 1, 1, 0.9)
	for i := 0.0; i <= maxSpeed; i += 10 {
		angle := startAngle + (totalArc * (i / maxSpeed))
		if i/maxSpeed <= 1.0 {
			x1 := cx + (radius-12)*math.Cos(angle) // Ajustado
			y1 := cy + (radius-12)*math.Sin(angle) // Ajustado
			x2 := cx + (radius-6)*math.Cos(angle)  // Ajustado
			y2 := cy + (radius-6)*math.Sin(angle)  // Ajustado
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
	// --- MODIFICAÇÃO: Valores ajustados para a nova escala ---
	radius := 50.0   // Raio reduzido de 70.0
	fontSize := 12.0 // Tamanho da fonte reduzido de 16
	// --------------------------------------------------------

	g.loadFont(dc, fontSize)

	dc.SetRGBA(0.1, 0.1, 0.1, 0.7)
	dc.DrawCircle(cx, cy, radius)
	dc.Fill()

	dc.SetLineWidth(1.5) // Linha mais fina
	dc.SetRGBA(0.5, 0.5, 0.5, 1)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()

	if g.fontLoaded {
		dc.SetRGBA(1, 1, 1, 0.9)
		cardinals := map[string]float64{"N": 270, "E": 0, "S": 90, "W": 180}
		for text, angle := range cardinals {
			rad := gg.Radians(angle)
			textX := cx + (radius-12)*math.Cos(rad) // Ajustado
			textY := cy + (radius-12)*math.Sin(rad) // Ajustado
			dc.DrawStringAnchored(text, textX, textY, 0.5, 0.5)
		}
	}

	dc.Push()
	dc.Translate(cx, cy)
	dc.Rotate(gg.Radians(bearing))

	// Ponteiro da bússola ajustado
	dc.SetRGBA(1, 0.2, 0.2, 1)
	dc.MoveTo(0, -radius+12)
	dc.LineTo(-9, 0)
	dc.LineTo(9, 0)
	dc.ClosePath()
	dc.Fill()

	dc.SetRGBA(1, 1, 1, 1)
	dc.MoveTo(0, radius-12)
	dc.LineTo(-9, 0)
	dc.LineTo(9, 0)
	dc.ClosePath()
	dc.Fill()

	dc.Pop()
}

// drawDigitalSpeed desenha o valor numérico da velocidade na parte inferior.
func (g *Generator) drawDigitalSpeed(dc *gg.Context, cx, cy float64, speed float64) {
	// --- MODIFICAÇÃO: Cor da fonte alterada para azul néon ---
	g.loadFont(dc, 28)
	dc.SetRGB255(0, 221, 255) // Cor azul néon vivo
	dc.DrawStringAnchored(fmt.Sprintf("%.1f", speed), cx, cy, 0.5, 0.5)

	g.loadFont(dc, 14)
	dc.SetRGB255(0, 221, 255) // Cor azul néon para consistência
	dc.DrawStringAnchored("km/h", cx, cy+20, 0.5, 0.5)
	// --------------------------------------------------
}

// Cleanup remove o diretório temporário.
func (g *Generator) Cleanup() {
	if g.tempDir != "" && g.tempDir != "." {
		os.RemoveAll(g.tempDir)
	}
}
