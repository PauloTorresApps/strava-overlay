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
		width:      820, // Aumentado para acomodar 1 velocímetro grande + 6 círculos em linha
		height:     190, // Altura reduzida para alinhar na base
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
		err := g.generateHybridDashboard(point, maxSpeedScale, imagePath)
		if err != nil {
			g.Cleanup()
			return nil, fmt.Errorf("erro ao gerar o frame de overlay %d: %w", i, err)
		}
		imagePaths = append(imagePaths, imagePath)
	}
	return imagePaths, nil
}

// generateHybridDashboard cria o layout com o velocímetro grande à esquerda e círculos de dados à direita.
func (g *Generator) generateHybridDashboard(point gps.GPSPoint, maxSpeed float64, outputPath string) error {
	dc := gg.NewContext(g.width, g.height)
	dc.SetRGBA(0, 0, 0, 0) // Fundo transparente
	dc.Clear()

	// Definir uma linha de base inferior comum com margem para alinhar todos os elementos por baixo
	bottomPadding := 10.0
	bottomY := float64(g.height) - bottomPadding

	// 1. Desenha o Velocímetro Grande à Esquerda
	speedometerRadius := 85.0
	speedometerCX := speedometerRadius + 10 // Posição X com uma pequena margem
	// Alinhar o centro Y para que a base do velocímetro toque a linha de base
	speedometerCY := bottomY - speedometerRadius
	g.drawSpeedometer(dc, speedometerCX, speedometerCY, point.Velocity*3.6, maxSpeed)
	g.drawCompass(dc, speedometerCX, speedometerCY, point.Bearing)
	g.drawDigitalSpeed(dc, speedometerCX, speedometerCY+60, point.Velocity*3.6)

	// 2. Desenha os Círculos de Dados à Direita em uma única linha
	circleRadius := 40.0 // Raio diminuído em 5px
	spacing := 15.0
	// Começa a desenhar os círculos após o velocímetro grande
	startX := (speedometerCX + speedometerRadius) + spacing + circleRadius
	currentX := startX
	// Alinhar o centro Y dos círculos para que suas bases toquem a linha de base
	startY := bottomY - circleRadius

	// Desenha todos os círculos de dados em uma linha
	g.drawDataCircle(dc, currentX, startY, circleRadius, color.RGBA{220, 40, 80, 255}, "Frequência", "BPM", fmt.Sprintf("%.0f", point.HeartRate), point.HeartRate > 0)
	currentX += (circleRadius * 2) + spacing
	g.drawDataCircle(dc, currentX, startY, circleRadius, color.RGBA{0, 200, 255, 255}, "Cadência", "RPM", fmt.Sprintf("%.0f", point.Cadence), point.Cadence > 0)
	currentX += (circleRadius * 2) + spacing
	g.drawDataCircle(dc, currentX, startY, circleRadius, color.RGBA{200, 200, 200, 255}, "Inclinação", "%", fmt.Sprintf("%.1f", point.Grade), true)
	currentX += (circleRadius * 2) + spacing
	g.drawDataCircle(dc, currentX, startY, circleRadius, color.RGBA{140, 90, 255, 255}, "Altitude", "m", fmt.Sprintf("%.0f", point.Altitude), true)
	currentX += (circleRadius * 2) + spacing
	g.drawDataCircle(dc, currentX, startY, circleRadius, color.RGBA{255, 180, 0, 255}, "Força-G", "g", fmt.Sprintf("%.2f", point.GForce), true)
	currentX += (circleRadius * 2) + spacing
	g.drawDataCircle(dc, currentX, startY, circleRadius, color.RGBA{60, 220, 60, 255}, "Distância", "km", fmt.Sprintf("%.2f", point.Distance/1000), point.Distance > 0)

	return dc.SavePNG(outputPath)
}

// drawDataCircle desenha um widget de dados circular genérico.
func (g *Generator) drawDataCircle(dc *gg.Context, cx, cy, radius float64, borderColor color.Color, label, unit, value string, available bool) {
	// Fundo do círculo
	dc.SetRGBA(0.1, 0.1, 0.1, 0.5)
	dc.DrawCircle(cx, cy, radius)
	dc.Fill()

	// Borda colorida
	dc.SetColor(borderColor)
	dc.SetLineWidth(2.5)
	dc.DrawCircle(cx, cy, radius)
	dc.Stroke()

	if !available {
		value = "- -"
	}

	// Rótulo (ex: "Cadência")
	g.loadFont(dc, 12)
	if available {
		dc.SetRGBA(1, 1, 1, 0.7)
	} else {
		dc.SetRGBA(1, 1, 1, 0.4)
	}
	dc.DrawStringAnchored(label, cx, cy-18, 0.5, 0.5)

	// Valor (ex: "90")
	g.loadFont(dc, 26)
	if available {
		dc.SetRGBA(1, 1, 1, 0.95)
	} else {
		dc.SetRGBA(1, 1, 1, 0.4)
	}
	dc.DrawStringAnchored(value, cx, cy+4, 0.5, 0.5)

	// Unidade (ex: "RPM")
	g.loadFont(dc, 11)
	if available {
		dc.SetRGBA(1, 1, 1, 0.6)
	} else {
		dc.SetRGBA(1, 1, 1, 0.4)
	}
	dc.DrawStringAnchored(unit, cx, cy+25, 0.5, 0.5)
}

// interpolateColor calcula a cor intermediária entre duas cores.
func interpolateColor(t float64, c1, c2 color.Color) color.Color {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	return color.RGBA{
		R: uint8((float64(r1>>8)*(1-t) + float64(r2>>8)*t)),
		G: uint8((float64(g1>>8)*(1-t) + float64(g2>>8)*t)),
		B: uint8((float64(b1>>8)*(1-t) + float64(b2>>8)*t)),
		A: uint8((float64(a1>>8)*(1-t) + float64(a2>>8)*t)),
	}
}

// drawSpeedometer desenha o arco do velocímetro com um visual moderno e gradiente suave.
func (g *Generator) drawSpeedometer(dc *gg.Context, cx, cy float64, speed, maxSpeed float64) {
	radius := 85.0
	lineWidth := 18.0
	progressWidth := 12.0 // Um pouco mais fino para dar espaço ao brilho
	fontSize := 12.0
	textOffset := 20.0

	startAngle := gg.Radians(135)
	totalArc := gg.Radians(270)
	g.loadFont(dc, fontSize)

	// 1. Círculo de fundo
	dc.SetLineWidth(lineWidth)
	dc.SetRGBA(0.1, 0.1, 0.1, 0.6)
	dc.DrawArc(cx, cy, radius, startAngle, startAngle+totalArc)
	dc.Stroke()

	// 2. Borda de Progresso com Gradiente Suave
	progress := speed / maxSpeed
	if progress > 1 {
		progress = 1
	}
	if progress < 0 {
		progress = 0
	}
	currentAngle := startAngle + (totalArc * progress)

	// Cores e Zonas
	colorGreen := color.RGBA{R: 0, G: 255, B: 127, A: 255}
	colorYellow := color.RGBA{R: 255, G: 215, B: 0, A: 255}
	colorRed := color.RGBA{R: 255, G: 69, B: 0, A: 255}

	// Pontos de transição do gradiente (0.0 a 1.0 do progresso total)
	greenEndProgress := 0.5
	yellowEndProgress := 0.8

	if progress > 0.001 {
		numSegments := int(math.Ceil(270 * progress)) // ~1 segmento por grau de progresso
		if numSegments < 2 {
			numSegments = 2
		}

		angleStep := (currentAngle - startAngle) / float64(numSegments)

		for i := 0; i < numSegments; i++ {
			segmentStart := startAngle + (float64(i) * angleStep)
			segmentEnd := segmentStart + angleStep

			// Calcula o progresso no meio do segmento para definir a cor
			segmentProgress := (float64(i) + 0.5) / float64(numSegments)

			var segmentColor color.Color
			if segmentProgress < greenEndProgress {
				t := segmentProgress / greenEndProgress
				segmentColor = interpolateColor(t, colorGreen, colorYellow)
			} else if segmentProgress < yellowEndProgress {
				t := (segmentProgress - greenEndProgress) / (yellowEndProgress - greenEndProgress)
				segmentColor = interpolateColor(t, colorYellow, colorRed)
			} else {
				segmentColor = colorRed
			}

			// Desenha o segmento com brilho
			dc.SetLineCap(gg.LineCapRound)
			r, g, b, _ := segmentColor.RGBA()

			// Efeito de brilho
			dc.SetRGBA255(int(r>>8), int(g>>8), int(b>>8), 80)
			dc.SetLineWidth(progressWidth + 6)
			dc.DrawArc(cx, cy, radius, segmentStart, segmentEnd)
			dc.Stroke()

			// Arco principal
			dc.SetColor(segmentColor)
			dc.SetLineWidth(progressWidth)
			dc.DrawArc(cx, cy, radius, segmentStart, segmentEnd)
			dc.Stroke()
		}
	}

	// 3. "Ponteiro" moderno no final do arco
	if progress > 0.001 {
		pointerX := cx + radius*math.Cos(currentAngle)
		pointerY := cy + radius*math.Sin(currentAngle)

		// Sombra suave para o ponteiro
		dc.SetRGBA(0, 0, 0, 0.5)
		dc.DrawCircle(pointerX+1, pointerY+1, progressWidth/2+1)
		dc.Fill()

		// Base do ponteiro
		dc.SetRGBA(1, 1, 1, 0.9)
		dc.DrawCircle(pointerX, pointerY, progressWidth/2)
		dc.Fill()
	}

	// 4. Marcadores e números
	dc.SetLineWidth(2)
	dc.SetRGBA(1, 1, 1, 0.9)
	dc.SetLineCap(gg.LineCapButt) // Reseta a terminação da linha para os marcadores
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

// drawCompass desenha a bússola no centro do velocímetro (versão original).
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

	// Ponteiro da bússola
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

// drawDigitalSpeed desenha o valor numérico da velocidade (versão original).
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
