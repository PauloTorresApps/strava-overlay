package overlay

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"strava-overlay/internal/gps"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

type Generator struct {
	width, height int
	tempDir       string
}

func NewGenerator() *Generator {
	tempDir, _ := os.MkdirTemp("", "strava_overlays_")

	return &Generator{
		width:   400, // Quadrado para evitar deformação
		height:  400,
		tempDir: tempDir,
	}
}

func (g *Generator) GenerateOverlaySequence(points []gps.GPSPoint, frameRate float64) ([]string, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("no GPS points provided")
	}

	maxSpeed := 0.0
	for _, point := range points {
		speed := point.Velocity * 3.6
		if speed > maxSpeed {
			maxSpeed = speed
		}
	}

	maxSpeedScale := math.Ceil(maxSpeed/10) * 10
	if maxSpeedScale == 0 {
		maxSpeedScale = 10
	}

	var imagePaths []string
	for i, point := range points {
		imagePath := filepath.Join(g.tempDir, fmt.Sprintf("overlay_%06d.png", i))
		err := g.generateOptimizedOverlay(point, maxSpeedScale, imagePath)
		if err != nil {
			return nil, fmt.Errorf("erro ao gerar overlay %d: %w", i, err)
		}
		imagePaths = append(imagePaths, imagePath)
	}

	return imagePaths, nil
}

func (g *Generator) generateOptimizedOverlay(point gps.GPSPoint, maxSpeed float64, outputPath string) error {
	dc := gg.NewContext(g.width, g.height)

	dc.SetLineJoin(gg.LineJoinRound)
	dc.SetLineCap(gg.LineCapRound)

	// Background transparente
	dc.SetRGBA(0, 0, 0, 0)
	dc.Clear()

	// Layout reposicionado - sem cortes
	centerX := float64(g.width) / 2
	centerY := float64(g.height)/2 - 20 // Ajuste para evitar corte inferior
	radius := 80.0                      // Reduzido para caber medidores
	currentSpeed := point.Velocity * 3.6

	// === VELOCÍMETRO CENTRAL ===
	g.drawNeonSpeedometer(dc, centerX, centerY, radius, currentSpeed, maxSpeed)

	// === BÚSSOLA DESTACADA NO CENTRO ===
	g.drawIntegratedCompass(dc, centerX, centerY, 35, point.Bearing)

	// === G-FORCE (ESQUERDA SUPERIOR) ===
	gaugeX := centerX - radius - 50
	g.drawCompactGauge(dc, gaugeX, centerY-30, 25, math.Abs(point.GForce), 3.0,
		fmt.Sprintf("%.1fG", math.Abs(point.GForce)), 0.2, 1, 0.3)

	// === ALTÍMETRO (ESQUERDA INFERIOR) ===
	g.drawCompactGauge(dc, gaugeX, centerY+30, 25, point.Altitude/1000, 5.0,
		fmt.Sprintf("%.0fm", point.Altitude), 0.3, 0.7, 1)

	// === DISPLAY INTEGRADO NO VELOCÍMETRO ===
	g.drawIntegratedSpeedDisplay(dc, centerX, centerY+radius*0.6, currentSpeed)

	return dc.SavePNG(outputPath)
}

// drawNeonSpeedometer - Velocímetro com borda única cinza
func (g *Generator) drawNeonSpeedometer(dc *gg.Context, x, y, radius, currentSpeed, maxSpeed float64) {
	// Background escuro
	dc.SetRGBA(0.08, 0.08, 0.08, 0.95)
	dc.DrawCircle(x, y, radius)
	dc.Fill()

	// === BORDA ÚNICA CINZA FINA ===
	dc.SetRGBA(0.6, 0.6, 0.6, 0.8)
	dc.SetLineWidth(2)
	dc.DrawCircle(x, y, radius)
	dc.Stroke()

	// Marcações de velocidade
	steps := int(maxSpeed / 10)
	startAngle := 135 * math.Pi / 180
	totalArc := 270 * math.Pi / 180

	dc.SetFontFace(basicfont.Face7x13)

	for i := 0; i <= steps; i++ {
		speed := float64(i) * 10
		angle := startAngle + (speed/maxSpeed)*totalArc

		innerRadius := radius - 20
		outerRadius := radius - 8

		dc.SetLineWidth(2)
		dc.SetRGBA(0.7, 0.7, 0.7, 1)

		x1 := x + innerRadius*math.Cos(angle)
		y1 := y + innerRadius*math.Sin(angle)
		x2 := x + outerRadius*math.Cos(angle)
		y2 := y + outerRadius*math.Sin(angle)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()

		// Números - fonte simples
		if speed > 0 && i%2 == 0 {
			textRadius := radius - 30
			textX := x + textRadius*math.Cos(angle)
			textY := y + textRadius*math.Sin(angle)

			dc.SetRGBA(1, 1, 1, 1)
			dc.DrawStringAnchored(fmt.Sprintf("%.0f", speed), textX, textY, 0.5, 0.5)
		}
	}

	// === BORDA DE PROGRESSO VERDE NEON ===
	progressRatio := currentSpeed / maxSpeed
	if progressRatio > 1 {
		progressRatio = 1
	}
	progressAngle := startAngle + (progressRatio * totalArc)

	dc.SetLineWidth(8)
	dc.SetRGBA(0.1, 1, 0.1, 0.9)
	dc.DrawArc(x, y, radius+8, startAngle, progressAngle)
	dc.Stroke()
}

// drawIntegratedCompass - Bússola destacada com pontos cardeais
func (g *Generator) drawIntegratedCompass(dc *gg.Context, x, y, radius, bearing float64) {
	// Background principal da bússola
	dc.SetRGBA(0.12, 0.12, 0.15, 0.9)
	dc.DrawCircle(x, y, radius)
	dc.Fill()

	// Círculo interno mais escuro
	dc.SetRGBA(0.08, 0.08, 0.1, 0.95)
	dc.DrawCircle(x, y, radius-4)
	dc.Fill()

	// Borda externa destacada
	dc.SetRGBA(0.5, 0.5, 0.6, 0.9)
	dc.SetLineWidth(2)
	dc.DrawCircle(x, y, radius)
	dc.Stroke()

	// === PONTOS CARDEAIS DESTACADOS ===
	cardinals := []struct {
		text    string
		angle   float64
		r, g, b float64
	}{
		{"N", -90, 1, 0.3, 0.3},   // Norte - vermelho
		{"E", 0, 0.9, 0.9, 0.9},   // Leste - branco
		{"S", 90, 0.9, 0.9, 0.9},  // Sul - branco
		{"W", 180, 0.9, 0.9, 0.9}, // Oeste - branco
	}

	dc.SetFontFace(basicfont.Face7x13)
	for _, card := range cardinals {
		angleRad := card.angle * math.Pi / 180
		textX := x + (radius+15)*math.Cos(angleRad)
		textY := y + (radius+15)*math.Sin(angleRad)

		// Background para pontos cardeais
		dc.SetRGBA(0, 0, 0, 0.6)
		dc.DrawCircle(textX, textY, 8)
		dc.Fill()

		// Texto cardinal
		dc.SetRGBA(card.r, card.g, card.b, 1)
		dc.DrawStringAnchored(card.text, textX, textY, 0.5, 0.5)
	}

	// === AGULHA BONITA COM DESIGN ELABORADO ===
	bearingRad := bearing*math.Pi/180 - math.Pi/2
	needleLength := radius * 0.85

	// Sombra da agulha
	tipX := x + needleLength*math.Cos(bearingRad) + 1.5
	tipY := y + needleLength*math.Sin(bearingRad) + 1.5
	dc.SetRGBA(0, 0, 0, 0.5)
	dc.SetLineWidth(4)
	dc.DrawLine(x, y, tipX, tipY)
	dc.Stroke()

	// Agulha principal - vermelha brilhante
	tipX = x + needleLength*math.Cos(bearingRad)
	tipY = y + needleLength*math.Sin(bearingRad)

	// Corpo da agulha com gradiente simulado
	dc.SetRGBA(1, 0.1, 0.1, 1)
	dc.SetLineWidth(3)
	dc.DrawLine(x, y, tipX, tipY)
	dc.Stroke()

	// Brilho na agulha
	dc.SetRGBA(1, 0.4, 0.4, 0.8)
	dc.SetLineWidth(1)
	dc.DrawLine(x, y, tipX, tipY)
	dc.Stroke()

	// Ponta da agulha - triangular
	pointLength := 8.0
	pointWidth := 3.0

	// Triângulo da ponta
	tipFinalX := x + (needleLength+pointLength)*math.Cos(bearingRad)
	tipFinalY := y + (needleLength+pointLength)*math.Sin(bearingRad)

	leftAngle := bearingRad + math.Pi*0.85
	rightAngle := bearingRad - math.Pi*0.85

	leftX := tipX + pointWidth*math.Cos(leftAngle)
	leftY := tipY + pointWidth*math.Sin(leftAngle)
	rightX := tipX + pointWidth*math.Cos(rightAngle)
	rightY := tipY + pointWidth*math.Sin(rightAngle)

	dc.SetRGBA(1, 0.2, 0.2, 1)
	dc.MoveTo(tipFinalX, tipFinalY)
	dc.LineTo(leftX, leftY)
	dc.LineTo(rightX, rightY)
	dc.ClosePath()
	dc.Fill()

	// Centro da bússola - design elaborado
	dc.SetRGBA(0.2, 0.2, 0.2, 1)
	dc.DrawCircle(x, y, 6)
	dc.Fill()

	dc.SetRGBA(1, 0.2, 0.2, 1)
	dc.DrawCircle(x, y, 4)
	dc.Fill()

	dc.SetRGBA(1, 0.6, 0.6, 0.8)
	dc.DrawCircle(x, y, 2)
	dc.Fill()
}

// drawCompactGauge - Medidores compactos laterais
func (g *Generator) drawCompactGauge(dc *gg.Context, x, y, radius, value, maxValue float64, label string, r, gVal, b float64) {
	// Background escuro
	dc.SetRGBA(0.1, 0.1, 0.1, 0.9)
	dc.DrawCircle(x, y, radius)
	dc.Fill()

	// Borda colorida
	dc.SetRGBA(r, gVal, b, 0.8)
	dc.SetLineWidth(3)
	dc.DrawCircle(x, y, radius)
	dc.Stroke()

	// Arco de progresso
	ratio := math.Min(value/maxValue, 1.0)
	startAngle := -math.Pi * 0.5
	endAngle := startAngle + ratio*math.Pi

	dc.SetLineWidth(6)
	dc.SetRGBA(r, gVal, b, 1)
	dc.DrawArc(x, y, radius-8, startAngle, endAngle)
	dc.Stroke()

	// Label central - FONTE SIMPLES
	dc.SetFontFace(basicfont.Face7x13)
	dc.SetRGBA(1, 1, 1, 1)
	dc.DrawStringAnchored(label, x, y, 0.5, 0.5)
}

// drawMainSpeedDisplay - Display principal com fonte única
func (g *Generator) drawMainSpeedDisplay(dc *gg.Context, x, y, speed float64) {
	displayWidth := 100.0
	displayHeight := 40.0

	// Background preto
	dc.SetRGBA(0, 0, 0, 0.95)
	dc.DrawRoundedRectangle(x-displayWidth/2, y-displayHeight/2,
		displayWidth, displayHeight, 6)
	dc.Fill()

	// Borda verde neon
	dc.SetRGBA(0.1, 1, 0.1, 1)
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(x-displayWidth/2, y-displayHeight/2,
		displayWidth, displayHeight, 6)
	dc.Stroke()

	// Velocidade - FONTE ÚNICA, SEM SOBREPOSIÇÃO
	dc.SetFontFace(basicfont.Face7x13)
	dc.SetRGBA(0.1, 1, 0.1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.1f", speed), x, y-6, 0.5, 0.5)

	// Unidade
	dc.SetRGBA(0.8, 0.8, 0.8, 1)
	dc.DrawStringAnchored("km/h", x, y+12, 0.5, 0.5)
}

// drawIntegratedSpeedDisplay - Display sem fundo
func (g *Generator) drawIntegratedSpeedDisplay(dc *gg.Context, x, y, speed float64) {
	// VELOCIDADE - SEM BACKGROUND
	dc.SetFontFace(basicfont.Face7x13)
	dc.SetRGBA(0.1, 1, 0.1, 1)

	speedText := fmt.Sprintf("%.1f", speed)

	// Grid 2x2 controlado
	for dx := -0.5; dx <= 0.5; dx += 0.5 {
		for dy := -0.5; dy <= 0.5; dy += 0.5 {
			dc.DrawStringAnchored(speedText, x+dx, y-4+dy, 0.5, 0.5)
		}
	}

	// Unidade
	dc.SetRGBA(0.8, 0.8, 0.8, 0.9)
	dc.DrawStringAnchored("km/h", x, y+10, 0.5, 0.5)
}

func (g *Generator) Cleanup() {
	os.RemoveAll(g.tempDir)
}
