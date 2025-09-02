package gps

import (
	"fmt"
	"math"
	"time"
)

type GPSPoint struct {
	Time     time.Time
	Lat      float64
	Lng      float64
	Velocity float64
	Altitude float64
	Bearing  float64
	GForce   float64
}

type GPSProcessor struct {
	points []GPSPoint
}

func NewGPSProcessor() *GPSProcessor {
	return &GPSProcessor{}
}

func (gp *GPSProcessor) ProcessStreamData(timeData, latlngData, velocityData, altitudeData []interface{}, startTime time.Time) error {
	if len(timeData) != len(latlngData) {
		return fmt.Errorf("data length mismatch")
	}

	gp.points = make([]GPSPoint, len(timeData))

	for i := 0; i < len(timeData); i++ {
		timeOffset := timeData[i].(float64)
		latlng := latlngData[i].([]interface{})

		point := GPSPoint{
			Time: startTime.Add(time.Duration(timeOffset) * time.Second),
			Lat:  latlng[0].(float64),
			Lng:  latlng[1].(float64),
		}

		if velocityData != nil && i < len(velocityData) && velocityData[i] != nil {
			point.Velocity = velocityData[i].(float64)
		}

		if altitudeData != nil && i < len(altitudeData) && altitudeData[i] != nil {
			point.Altitude = altitudeData[i].(float64)
		}

		if i > 0 {
			point.Bearing = gp.calculateBearing(gp.points[i-1], point)
			point.GForce = gp.calculateGForce(gp.points[i-1], point)
		}

		gp.points[i] = point
	}

	return nil
}

// GetPointsForTimeRange coleta todos os pontos GPS dentro de um intervalo de tempo.
// Aprimorada para garantir cobertura completa do intervalo.
func (gp *GPSProcessor) GetPointsForTimeRange(startTime, endTime time.Time) []GPSPoint {
	var result []GPSPoint

	// Primeiro, encontra o ponto mais próximo ao início
	startPoint, startFound := gp.GetPointForTime(startTime)
	if !startFound {
		return result
	}

	// Encontra o ponto mais próximo ao fim
	endPoint, endFound := gp.GetPointForTime(endTime)
	if !endFound {
		return result
	}

	// Coleta todos os pontos entre o ponto inicial e final (inclusive)
	collecting := false
	for _, point := range gp.points {
		// Inicia coleta quando atinge o ponto inicial
		if point.Time.Equal(startPoint.Time) {
			collecting = true
		}

		if collecting {
			result = append(result, point)
		}

		// Para coleta quando atinge o ponto final
		if point.Time.Equal(endPoint.Time) {
			break
		}
	}

	// Log de debug para verificar a sincronização
	fmt.Printf("DEBUG: Vídeo inicia em %s, termina em %s\n", startTime.Format("15:04:05"), endTime.Format("15:04:05"))
	fmt.Printf("DEBUG: GPS inicia em %s, termina em %s\n", startPoint.Time.Format("15:04:05"), endPoint.Time.Format("15:04:05"))
	fmt.Printf("DEBUG: Coletados %d pontos GPS para o vídeo\n", len(result))

	return result
}

func (gp *GPSProcessor) calculateBearing(from, to GPSPoint) float64 {
	lat1 := from.Lat * math.Pi / 180
	lat2 := to.Lat * math.Pi / 180
	deltaLng := (to.Lng - from.Lng) * math.Pi / 180

	y := math.Sin(deltaLng) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(deltaLng)

	bearing := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(bearing+360, 360)
}

func (gp *GPSProcessor) calculateGForce(from, to GPSPoint) float64 {
	if from.Time.Equal(to.Time) {
		return 0
	}

	deltaV := to.Velocity - from.Velocity
	deltaT := to.Time.Sub(from.Time).Seconds()

	if deltaT == 0 {
		return 0
	}

	acceleration := deltaV / deltaT
	return acceleration / 9.81
}

// GetPointForTime encontra o ponto GPS mais próximo a um tempo específico.
// Em caso de empate, prefere o ponto com tempo maior (posterior).
func (gp *GPSProcessor) GetPointForTime(targetTime time.Time) (GPSPoint, bool) {
	if len(gp.points) == 0 {
		return GPSPoint{}, false
	}

	var closestPoint GPSPoint
	var minDiff time.Duration
	found := false

	for _, point := range gp.points {
		diff := point.Time.Sub(targetTime)
		if diff < 0 {
			diff = -diff // valor absoluto
		}

		if !found || diff < minDiff || (diff == minDiff && point.Time.After(closestPoint.Time)) {
			closestPoint = point
			minDiff = diff
			found = true
		}
	}

	return closestPoint, found
}
