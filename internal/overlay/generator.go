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
		width:   400,
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
		maxSpeedScale = 50
	}

	var imagePaths []string
	for i, point := range points {
		imagePath := filepath.Join(g.tempDir, fmt.Sprintf("overlay_%06d.png", i))
		err := g.generateSpeedometerImage(point, maxSpeedScale, imagePath)
		if err != nil {
			return nil, fmt.Errorf("erro ao gerar overlay %d: %w", i, err)
		}
		imagePaths = append(imagePaths, imagePath)
	}
	return imagePaths, nil
}

func (g *Generator) generateSpeedometerImage(point gps.GPSPoint, maxSpeed float64, outputPath string) error {
	dc := gg.NewContext(g.width, g.height)
	dc.SetLineJoin(gg.LineJoinRound)
	dc.SetLineCap(gg.LineCapRound)
	dc.SetRGBA(0, 0, 0, 0)
	dc.Clear()

	centerX := float64(g.width) / 2
	centerY := float64(g.height)/2 - 20
	radius := 80.0
	currentSpeed := point.Velocity * 3.6

	// VELOCÍMETRO
	dc.SetRGBA(0.08, 0.08, 0.08, 0.95)
	dc.DrawCircle(centerX, centerY, radius)
	dc.Fill()
	dc.SetRGBA(0.6, 0.6, 0.6, 0.8)
	dc.SetLineWidth(2)
	dc.DrawCircle(centerX, centerY, radius)
	dc.Stroke()

	// MARCAÇÕES E NÚMEROS
	startAngle := 135 * math.Pi / 180
	totalArc := 270 * math.Pi / 180
	dc.SetFontFace(basicfont.Face7x13)

	for speed := 0.0; speed <= maxSpeed; speed += 1.0 {
		angle := startAngle + (speed/maxSpeed)*totalArc

		if int(speed)%5 == 0 {
			// MARCAÇÃO PRINCIPAL (5 em 5)
			innerR := radius - 25
			outerR := radius - 8
			dc.SetLineWidth(3)
			dc.SetRGBA(0.9, 0.9, 0.9, 1)

			x1 := centerX + innerR*math.Cos(angle)
			y1 := centerY + innerR*math.Sin(angle)
			x2 := centerX + outerR*math.Cos(angle)
			y2 := centerY + outerR*math.Sin(angle)
			dc.DrawLine(x1, y1, x2, y2)
			dc.Stroke()

			// NÚMEROS
			textR := radius - 35
			textX := centerX + textR*math.Cos(angle)
			textY := centerY + textR*math.Sin(angle)
			dc.SetRGBA(1, 1, 1, 1)
			dc.DrawStringAnchored(fmt.Sprintf("%.0f", speed), textX, textY, 0.5, 0.5)

		} else {
			// MARCAÇÃO INTERMEDIÁRIA (1 em 1)
			innerR := radius - 15
			outerR := radius - 8
			dc.SetLineWidth(1)
			dc.SetRGBA(0.5, 0.5, 0.5, 0.7)

			x1 := centerX + innerR*math.Cos(angle)
			y1 := centerY + innerR*math.Sin(angle)
			x2 := centerX + outerR*math.Cos(angle)
			y2 := centerY + outerR*math.Sin(angle)
			dc.DrawLine(x1, y1, x2, y2)
			dc.Stroke()
		}
	}

	// ARCO PROGRESSO
	progressRatio := currentSpeed / maxSpeed
	if progressRatio > 1 {
		progressRatio = 1
	}
	progressAngle := startAngle + (progressRatio * totalArc)
	dc.SetLineWidth(8)
	dc.SetRGBA(0.1, 1, 0.1, 0.9)
	dc.DrawArc(centerX, centerY, radius+8, startAngle, progressAngle)
	dc.Stroke()

	// BÚSSOLA DESTACADA
	compassRadius := 35.0
	g.drawCompass(dc, centerX, centerY, compassRadius, point.Bearing)

	// MEDIDORES LATERAIS
	gaugeX := centerX - radius - 50
	g.drawGauge(dc, gaugeX, centerY-30, 25, math.Abs(point.GForce), "G")
	g.drawGauge(dc, gaugeX, centerY+30, 25, point.Altitude/100, "m")

	// DISPLAY SEM FUNDO
	dc.SetRGBA(0.1, 1, 0.1, 1)
	speedText := fmt.Sprintf("%.1f", currentSpeed)
	for dx := -0.5; dx <= 0.5; dx += 0.5 {
		for dy := -0.5; dy <= 0.5; dy += 0.5 {
			dc.DrawStringAnchored(speedText, centerX+dx, centerY+radius*0.6+dy, 0.5, 0.5)
		}
	}
	dc.SetRGBA(0.8, 0.8, 0.8, 0.9)
	dc.DrawStringAnchored("km/h", centerX, centerY+radius*0.6+15, 0.5, 0.5)

	return dc.SavePNG(outputPath)
}

func (g *Generator) drawCompass(dc *gg.Context, x, y, radius, bearing float64) {
	// Background
	dc.SetRGBA(0.12, 0.12, 0.15, 0.9)
	dc.DrawCircle(x, y, radius)
	dc.Fill()
	dc.SetRGBA(0.5, 0.5, 0.6, 0.9)
	dc.SetLineWidth(2)
	dc.DrawCircle(x, y, radius)
	dc.Stroke()

	// PONTOS CARDEAIS - REPOSICIONADOS PARA FORA DO VELOCÍMETRO
	cardinals := []struct {
		text    string
		angle   float64
		r, g, b float64
	}{
		{"N", -90, 1, 0.3, 0.3},
		{"E", 0, 0.9, 0.9, 0.9},
		{"S", 90, 0.9, 0.9, 0.9},
		{"W", 180, 0.9, 0.9, 0.9},
	}

	dc.SetFontFace(basicfont.Face7x13)
	for _, card := range cardinals {
		angleRad := card.angle * math.Pi / 180
		// DISTÂNCIA FINAL: 62px da borda
		textX := x + (radius+62)*math.Cos(angleRad)
		textY := y + (radius+62)*math.Sin(angleRad)

		// FONTE DEFINIDA SEM FUNDO
		dc.SetRGBA(card.r, card.g, card.b, 1)
		// Grid 2x2 para definição aumentada
		for dx := -0.5; dx <= 0.5; dx += 0.5 {
			for dy := -0.5; dy <= 0.5; dy += 0.5 {
				dc.DrawStringAnchored(card.text, textX+dx, textY+dy, 0.5, 0.5)
			}
		}
	}

	// AGULHA COM BASE LARGA
	bearingRad := bearing*math.Pi/180 - math.Pi/2
	needleLength := radius * 0.85

	// BASE LARGA
	baseLength := radius * 0.25
	baseWidth := 6.0
	baseEndX := x - baseLength*math.Cos(bearingRad)
	baseEndY := y - baseLength*math.Sin(bearingRad)
	baseLeftX := baseEndX + baseWidth*math.Cos(bearingRad+math.Pi/2)
	baseLeftY := baseEndY + baseWidth*math.Sin(bearingRad+math.Pi/2)
	baseRightX := baseEndX + baseWidth*math.Cos(bearingRad-math.Pi/2)
	baseRightY := baseEndY + baseWidth*math.Sin(bearingRad-math.Pi/2)

	dc.SetRGBA(0.9, 0.1, 0.1, 1)
	dc.MoveTo(x, y)
	dc.LineTo(baseLeftX, baseLeftY)
	dc.LineTo(baseRightX, baseRightY)
	dc.ClosePath()
	dc.Fill()

	// CORPO DA AGULHA
	tipX := x + needleLength*math.Cos(bearingRad)
	tipY := y + needleLength*math.Sin(bearingRad)
	dc.SetRGBA(1, 0.1, 0.1, 1)
	dc.SetLineWidth(5)
	dc.DrawLine(x, y, tipX, tipY)
	dc.Stroke()

	// RELEVO CENTRAL
	dc.SetRGBA(1, 0.7, 0.7, 0.9)
	dc.SetLineWidth(1)
	dc.DrawLine(baseEndX, baseEndY, tipX, tipY)
	dc.Stroke()

	// EIXO RECUADO
	eixoX := x + (radius*0.08)*math.Cos(bearingRad)
	eixoY := y + (radius*0.08)*math.Sin(bearingRad)
	dc.SetRGBA(0.2, 0.2, 0.2, 1)
	dc.DrawCircle(eixoX, eixoY, 4)
	dc.Fill()
	dc.SetRGBA(0.7, 0.7, 0.7, 1)
	dc.DrawCircle(eixoX, eixoY, 3)
	dc.Fill()
}

func (g *Generator) drawGauge(dc *gg.Context, x, y, radius, value float64, unit string) {
	dc.SetRGBA(0.1, 0.1, 0.1, 0.9)
	dc.DrawCircle(x, y, radius)
	dc.Fill()
	dc.SetRGBA(0.5, 0.5, 0.5, 0.8)
	dc.SetLineWidth(2)
	dc.DrawCircle(x, y, radius)
	dc.Stroke()
	dc.SetFontFace(basicfont.Face7x13)
	dc.SetRGBA(1, 1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.1f%s", value, unit), x, y, 0.5, 0.5)
}

func (g *Generator) Cleanup() {
	os.RemoveAll(g.tempDir)
}
