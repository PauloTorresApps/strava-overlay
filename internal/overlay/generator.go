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

	// Fundo transparente
	dc.SetRGBA(0, 0, 0, 0)
	dc.Clear()

	centerX := float64(g.width) / 2
	centerY := float64(g.height) / 2
	radius := 70.0
	currentSpeed := point.Velocity * 3.6

	// === VELOCÍMETRO CIRCULAR ===
	// Fundo do velocímetro
	dc.SetRGBA(0.1, 0.1, 0.1, 0.9)
	dc.DrawCircle(centerX, centerY, radius)
	dc.Fill()

	// Marcações de velocidade (5 em 5 km/h)
	// Velocidade 0 em 225° (7:30h), crescendo no sentido horário
	dc.SetFontFace(basicfont.Face7x13)
	steps := int(maxSpeed / 5)
	startAngle := 225 * math.Pi / 180 // 225° em radianos
	totalArc := 3 * math.Pi / 2       // 270° de arco

	for i := 0; i <= steps; i++ {
		speed := float64(i) * 5
		angle := startAngle + (speed/maxSpeed)*totalArc

		// Linha de marcação
		x1 := centerX + (radius-15)*math.Cos(angle)
		y1 := centerY + (radius-15)*math.Sin(angle)
		x2 := centerX + (radius-5)*math.Cos(angle)
		y2 := centerY + (radius-5)*math.Sin(angle)

		dc.SetRGBA(0.4, 0.4, 0.4, 1)
		dc.SetLineWidth(1.5)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()

		// Números a cada 10 km/h
		if i%2 == 0 {
			textX := centerX + (radius-25)*math.Cos(angle)
			textY := centerY + (radius-25)*math.Sin(angle)
			dc.SetRGBA(0.8, 0.8, 0.8, 1)
			dc.DrawStringAnchored(fmt.Sprintf("%.0f", speed), textX, textY, 0.5, 0.5)
		}
	}

	// Pontos cardeais
	cardinals := []struct {
		text  string
		angle float64
	}{
		{"N", -math.Pi / 2}, {"E", 0}, {"S", math.Pi / 2}, {"W", math.Pi},
	}
	for _, card := range cardinals {
		x := centerX + (radius+15)*math.Cos(card.angle)
		y := centerY + (radius+15)*math.Sin(card.angle)
		dc.SetRGBA(0, 1, 1, 1) // Azul neon
		dc.DrawStringAnchored(card.text, x, y, 0.5, 0.5)
	}

	// Borda de progresso (cor baseada na velocidade)
	progressAngle := (currentSpeed / maxSpeed) * totalArc
	steps_progress := int(progressAngle * 50 / math.Pi)

	for i := 0; i < steps_progress; i++ {
		angle := startAngle + float64(i)*progressAngle/float64(steps_progress)

		// Cor quente/fria baseada na velocidade
		ratio := currentSpeed / maxSpeed
		r := ratio
		g := 1 - ratio
		b := 0.2

		dc.SetRGBA(r, g, b, 1)
		dc.SetLineWidth(4)

		x1 := centerX + (radius+2)*math.Cos(angle)
		y1 := centerY + (radius+2)*math.Sin(angle)
		x2 := centerX + (radius+8)*math.Cos(angle)
		y2 := centerY + (radius+8)*math.Sin(angle)
		dc.DrawLine(x1, y1, x2, y2)
		dc.Stroke()
	}

	// Borda externa
	dc.SetRGBA(0.3, 0.3, 0.3, 1)
	dc.SetLineWidth(2)
	dc.DrawCircle(centerX, centerY, radius)
	dc.Stroke()

	// === AGULHA DA BÚSSOLA (TRIANGULAR VERMELHA) ===
	compassAngle := point.Bearing*math.Pi/180 - math.Pi/2
	needleLength := 25.0
	needleWidth := 8.0

	// Ponta da agulha
	tipX := centerX + needleLength*math.Cos(compassAngle)
	tipY := centerY + needleLength*math.Sin(compassAngle)

	// Base da agulha (lados do triângulo)
	leftX := centerX + needleWidth*math.Cos(compassAngle+math.Pi/2)
	leftY := centerY + needleWidth*math.Sin(compassAngle+math.Pi/2)
	rightX := centerX + needleWidth*math.Cos(compassAngle-math.Pi/2)
	rightY := centerY + needleWidth*math.Sin(compassAngle-math.Pi/2)

	// Desenha triângulo vermelho neon
	dc.SetRGBA(1, 0.1, 0.1, 1) // Vermelho neon
	dc.MoveTo(tipX, tipY)
	dc.LineTo(leftX, leftY)
	dc.LineTo(rightX, rightY)
	dc.ClosePath()
	dc.Fill()

	// Centro da bússola
	dc.SetRGBA(1, 0.1, 0.1, 1)
	dc.DrawCircle(centerX, centerY, 4)
	dc.Fill()

	// === DISPLAY DIGITAL DE VELOCIDADE ===
	dc.SetRGBA(0, 0, 0, 0.9)
	dc.DrawRoundedRectangle(centerX-25, centerY+35, 50, 20, 5)
	dc.Fill()

	dc.SetRGBA(0, 1, 0, 1) // Verde neon
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(centerX-25, centerY+35, 50, 20, 5)
	dc.Stroke()

	dc.SetRGBA(0, 1, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.0f km/h", currentSpeed), centerX, centerY+45, 0.5, 0.5)

	// === G-FORCE (superior esquerdo) ===
	gX := centerX - radius - 30
	gY := centerY - radius + 20

	dc.SetRGBA(0.1, 0.1, 0.1, 0.9)
	dc.DrawCircle(gX, gY, 18)
	dc.Fill()

	dc.SetRGBA(1, 1, 0, 1) // Amarelo neon
	dc.SetLineWidth(2)
	dc.DrawCircle(gX, gY, 18)
	dc.Stroke()

	dc.SetRGBA(1, 1, 0, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.1f G", math.Abs(point.GForce)), gX, gY, 0.5, 0.5)

	// === ALTÍMETRO (inferior esquerdo) ===
	altX := centerX - radius - 30
	altY := centerY + radius - 20

	dc.SetRGBA(0.1, 0.1, 0.1, 0.9)
	dc.DrawRoundedRectangle(altX-25, altY-12, 50, 24, 5)
	dc.Fill()

	dc.SetRGBA(0, 1, 1, 1) // Azul ciano neon
	dc.SetLineWidth(2)
	dc.DrawRoundedRectangle(altX-25, altY-12, 50, 24, 5)
	dc.Stroke()

	dc.SetRGBA(0, 1, 1, 1)
	dc.DrawStringAnchored(fmt.Sprintf("%.0fm", point.Altitude), altX, altY, 0.5, 0.5)

	return dc.SavePNG(outputPath)
}

func (g *Generator) Cleanup() {
	os.RemoveAll(g.tempDir)
}
