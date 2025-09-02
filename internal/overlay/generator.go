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
	// Resolução aumentada para uma imagem mais nítida
	return &Generator{
		width:   800,
		height:  800,
		tempDir: tempDir,
	}
}

func (g *Generator) GenerateOverlaySequence(points []gps.GPSPoint, frameRate float64) ([]string, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("no GPS points provided")
	}

	maxSpeed := 0.0
	for _, point := range points {
		speed := point.Velocity * 3.6 // m/s para km/h
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
		err := g.generateSpeedometerImage(point, maxSpeedScale, imagePath)
		if err != nil {
			return nil, fmt.Errorf("erro ao gerar overlay %d: %w", i, err)
		}
		imagePaths = append(imagePaths, imagePath)
	}

	fmt.Printf("DEBUG: Geradas %d imagens de overlay para %d pontos GPS\n", len(imagePaths), len(points))
	return imagePaths, nil
}

func (g *Generator) generateSpeedometerImage(point gps.GPSPoint, maxSpeed float64, outputPath string) error {
	dc := gg.NewContext(g.width, g.height)

	// Fundo transparente
	dc.SetRGBA(0, 0, 0, 0)
	dc.Clear()

	// Fator de escala para converter as dimensões originais para a nova resolução
	scaleFactor := float64(g.width) / 350.0

	centerX := float64(g.width) / 2
	centerY := float64(g.height) / 2
	radius := 85.0 * scaleFactor
	currentSpeed := point.Velocity * 3.6

	// === VELOCÍMETRO CIRCULAR ===
	dc.SetRGBA(0.1, 0.1, 0.1, 0.85)
	dc.DrawCircle(centerX, centerY, radius)
	dc.Fill()

	// Marcações de velocidade
	dc.SetFontFace(basicfont.Face7x13) // Nota: esta fonte não escala perfeitamente.
	steps := int(maxSpeed / 5)
	startAngle := 135 * math.Pi / 180
	totalArc := 270 * math.Pi / 180

	for i := 0; i <= steps; i++ {
		speed := float64(i) * 5
		angle := startAngle + (speed/maxSpeed)*totalArc

		x1 := centerX + (radius-(12*scaleFactor))*math.Cos(angle)
		y1 := centerY + (radius-(12*scaleFactor))*math.Sin(angle)
		x2 := centerX + (radius-(5*scaleFactor))*math.Cos(angle)
		y2 := centerY + (radius-(5*scaleFactor))*math.Sin(angle)

		dc.SetRGBA(0.5, 0.5, 0.5, 1)
		dc.SetLineWidth(2 * scaleFactor)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()

		if i%2 == 0 {
			textRadius := radius - (26 * scaleFactor)
			textX := centerX + textRadius*math.Cos(angle)
			textY := centerY + textRadius*math.Sin(angle)
			dc.SetRGBA(0.9, 0.9, 0.9, 1)
			dc.DrawStringAnchored(fmt.Sprintf("%.0f", speed), textX, textY, 0.5, 0.5)
		}
	}

	// Borda de progresso
	progressRatio := currentSpeed / maxSpeed
	if progressRatio > 1 {
		progressRatio = 1
	}
	progressAngle := startAngle + (progressRatio * totalArc)

	dc.SetLineWidth(6 * scaleFactor)
	r := progressRatio * 1.2
	greenVal := (1 - progressRatio) * 0.5
	b := (1 - progressRatio)
	dc.SetRGBA(r, greenVal, b, 1)
	dc.DrawArc(centerX, centerY, radius+(6*scaleFactor), startAngle, progressAngle)
	dc.Stroke()

	// Pontos cardeais
	cardinals := []struct {
		text  string
		angle float64
	}{
		{"N", -90}, {"E", 0}, {"S", 90}, {"W", 180},
	}
	for _, card := range cardinals {
		angleRad := card.angle * math.Pi / 180
		x := centerX + (radius+(20*scaleFactor))*math.Cos(angleRad)
		y := centerY + (radius+(20*scaleFactor))*math.Sin(angleRad)
		dc.SetRGBA(0.8, 0.8, 0.8, 1)
		dc.DrawStringAnchored(card.text, x, y, 0.5, 0.5)
	}

	// === AGULHA DA BÚSSOLA ===
	compassAngleRad := point.Bearing*math.Pi/180 - math.Pi/2
	needleLength := radius * 0.5
	needleWidth := radius * 0.1

	tipX := centerX + needleLength*math.Cos(compassAngleRad)
	tipY := centerY + needleLength*math.Sin(compassAngleRad)
	baseAngle1 := compassAngleRad + math.Pi*0.9
	baseAngle2 := compassAngleRad - math.Pi*0.9
	baseX1 := centerX + needleWidth*math.Cos(baseAngle1)
	baseY1 := centerY + needleWidth*math.Sin(baseAngle1)
	baseX2 := centerX + needleWidth*math.Cos(baseAngle2)
	baseY2 := centerY + needleWidth*math.Sin(baseAngle2)

	dc.SetRGBA(1, 0, 0, 0.9)
	dc.MoveTo(tipX, tipY)
	dc.LineTo(baseX1, baseY1)
	dc.LineTo(baseX2, baseY2)
	dc.ClosePath()
	dc.Fill()

	// Centro da bússola
	dc.SetRGBA(0.1, 0.1, 0.1, 1)
	dc.DrawCircle(centerX, centerY, 6*scaleFactor)
	dc.Fill()
	dc.SetRGBA(1, 0, 0, 0.9)
	dc.DrawCircle(centerX, centerY, 3*scaleFactor)
	dc.Fill()

	// === DISPLAY DIGITAL DE VELOCIDADE ===
	digitalY := centerY + radius*0.5
	dc.SetRGBA(0, 1, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.1f", currentSpeed), centerX, digitalY, 0.5, 0.5)
	dc.DrawStringAnchored("km/h", centerX, digitalY+(12*scaleFactor), 0.5, 0.5)

	// === G-FORCE ===
	gForceRadius := 25.0 * scaleFactor
	gX := centerX - radius - gForceRadius - (4 * scaleFactor)
	gY := centerY - radius*0.6

	dc.SetRGBA(0.1, 0.1, 0.1, 0.85)
	dc.DrawCircle(gX, gY, gForceRadius)
	dc.Fill()
	dc.SetRGBA(0, 1, 0, 1)
	dc.SetLineWidth(2 * scaleFactor)
	dc.DrawCircle(gX, gY, gForceRadius)
	dc.Stroke()
	dc.SetRGBA(0.9, 0.9, 0.9, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.1fG", math.Abs(point.GForce)), gX, gY, 0.5, 0.5)

	// === ALTÍMETRO ===
	altRadius := 25.0 * scaleFactor
	altX := centerX - radius - altRadius - (4 * scaleFactor)
	altY := centerY + radius*0.6

	dc.SetRGBA(0.1, 0.1, 0.1, 0.85)
	dc.DrawCircle(altX, altY, altRadius)
	dc.Fill()
	dc.SetRGBA(0, 1, 1, 1)
	dc.SetLineWidth(2 * scaleFactor)
	dc.DrawCircle(altX, altY, altRadius)
	dc.Stroke()
	dc.SetRGBA(0.9, 0.9, 0.9, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.0fm", point.Altitude), altX, altY, 0.5, 0.5)

	return dc.SavePNG(outputPath)
}

func (g *Generator) Cleanup() {
	os.RemoveAll(g.tempDir)
}
