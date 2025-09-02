package overlay

import (
	"fmt"
	"image/color"
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
		width:   200,
		height:  200,
		tempDir: tempDir,
	}
}

func (g *Generator) GenerateOverlaySequence(points []gps.GPSPoint, frameRate float64) ([]string, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("no GPS points provided")
	}

	// Calcula velocidade máxima para escala do velocímetro
	maxSpeed := 0.0
	for _, point := range points {
		speed := point.Velocity * 3.6 // m/s para km/h
		if speed > maxSpeed {
			maxSpeed = speed
		}
	}
	maxSpeedScale := math.Ceil(maxSpeed/10) * 10

	var imagePaths []string

	// Gera uma imagem para cada ponto GPS
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

	dc.SetRGBA(0, 0, 0, 0)
	dc.Clear()

	centerX := float64(g.width) / 2
	centerY := float64(g.height) / 2
	radius := 80.0

	// Círculo do velocímetro
	dc.SetColor(color.RGBA{255, 255, 255, 200})
	dc.SetLineWidth(3)
	dc.DrawCircle(centerX, centerY, radius)
	dc.Stroke()

	// Marcações de velocidade
	steps := 8
	for i := 0; i < steps; i++ {
		angle := float64(i)*2*math.Pi/float64(steps) - math.Pi/2
		x1 := centerX + (radius-10)*math.Cos(angle)
		y1 := centerY + (radius-10)*math.Sin(angle)
		x2 := centerX + radius*math.Cos(angle)
		y2 := centerY + radius*math.Sin(angle)

		dc.SetLineWidth(2)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}

	// Ponteiro de velocidade
	currentSpeed := point.Velocity * 3.6
	speedAngle := (currentSpeed/maxSpeed)*3*math.Pi/2 - math.Pi/2
	speedX := centerX + (radius-15)*math.Cos(speedAngle)
	speedY := centerY + (radius-15)*math.Sin(speedAngle)

	dc.SetColor(color.RGBA{0, 255, 0, 255})
	dc.SetLineWidth(3)
	dc.DrawLine(centerX, centerY, speedX, speedY)
	dc.Stroke()

	// Bússola (agulha vermelha)
	compassAngle := point.Bearing*math.Pi/180 - math.Pi/2
	needleLength := radius * 0.6
	needleX := centerX + needleLength*math.Cos(compassAngle)
	needleY := centerY + needleLength*math.Sin(compassAngle)

	dc.SetColor(color.RGBA{255, 0, 0, 255})
	dc.SetLineWidth(4)
	dc.DrawLine(centerX, centerY, needleX, needleY)
	dc.Stroke()

	// Texto com fonte básica
	dc.SetFontFace(basicfont.Face7x13)
	dc.SetColor(color.RGBA{255, 255, 255, 255})

	// Velocidade digital
	speedText := fmt.Sprintf("%.0f km/h", currentSpeed)
	dc.DrawStringAnchored(speedText, centerX, centerY+35, 0.5, 0.5)

	// Força G
	gForceText := fmt.Sprintf("G: %.2f", point.GForce)
	dc.DrawStringAnchored(gForceText, centerX-radius-20, centerY-radius+10, 0.5, 0.5)

	// Altimetria
	altitudeText := fmt.Sprintf("Alt: %.0fm", point.Altitude)
	dc.DrawStringAnchored(altitudeText, centerX+radius+20, centerY+radius-10, 0.5, 0.5)

	return dc.SavePNG(outputPath)
}

func (g *Generator) Cleanup() {
	os.RemoveAll(g.tempDir)
}
