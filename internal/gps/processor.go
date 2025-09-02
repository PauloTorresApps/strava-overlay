package gps

import (
	"fmt"
	"log"
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
		return fmt.Errorf("data length mismatch: time=%d, latlng=%d", len(timeData), len(latlngData))
	}

	log.Printf("DEBUG: Processando %d pontos GPS", len(timeData))
	rawPoints := make([]GPSPoint, 0, len(timeData))

	validPoints := 0
	for i := 0; i < len(timeData); i++ {
		timeOffset, ok := timeData[i].(float64)
		if !ok {
			continue
		}

		latlngInterface, ok := latlngData[i].([]interface{})
		if !ok || len(latlngInterface) != 2 {
			continue
		}

		lat, ok1 := latlngInterface[0].(float64)
		lng, ok2 := latlngInterface[1].(float64)

		if !ok1 || !ok2 || lat == 0 && lng == 0 ||
			lat < -90 || lat > 90 || lng < -180 || lng > 180 ||
			math.IsNaN(lat) || math.IsNaN(lng) {
			continue
		}

		point := GPSPoint{
			Time: startTime.Add(time.Duration(timeOffset) * time.Second),
			Lat:  lat,
			Lng:  lng,
		}

		if velocityData != nil && i < len(velocityData) && velocityData[i] != nil {
			if vel, ok := velocityData[i].(float64); ok && !math.IsNaN(vel) {
				point.Velocity = vel
			}
		}

		if altitudeData != nil && i < len(altitudeData) && altitudeData[i] != nil {
			if alt, ok := altitudeData[i].(float64); ok && !math.IsNaN(alt) {
				point.Altitude = alt
			}
		}

		if len(rawPoints) > 0 {
			prevPoint := rawPoints[len(rawPoints)-1]
			point.Bearing = gp.calculateBearing(prevPoint, point)
			point.GForce = gp.calculateGForce(prevPoint, point)
		}

		rawPoints = append(rawPoints, point)
		validPoints++
	}

	if validPoints == 0 {
		return fmt.Errorf("nenhum ponto GPS válido encontrado")
	}

	// Interpola os pontos para criar uma transição suave
	gp.points = gp.interpolatePoints(rawPoints)
	log.Printf("DEBUG: %d pontos GPS válidos processados, resultando em %d pontos após interpolação", validPoints, len(gp.points))

	return nil
}

func (gp *GPSProcessor) interpolatePoints(points []GPSPoint) []GPSPoint {
	if len(points) < 2 {
		return points
	}

	var interpolated []GPSPoint
	for i := 0; i < len(points)-1; i++ {
		p1 := points[i]
		p2 := points[i+1]

		t1 := p1.Time
		t2 := p2.Time
		duration := t2.Sub(t1)

		// Adiciona o ponto inicial do intervalo
		interpolated = append(interpolated, p1)

		// Se o intervalo for maior que 1 segundo, interpola
		if duration.Seconds() > 1.0 {
			// Gera os pontos intermediários a cada segundo
			for t := time.Second; t < duration; t += time.Second {
				ratio := float64(t) / float64(duration)

				lat := p1.Lat + ratio*(p2.Lat-p1.Lat)
				lng := p1.Lng + ratio*(p2.Lng-p1.Lng)
				altitude := p1.Altitude + ratio*(p2.Altitude-p1.Altitude)
				velocity := p1.Velocity + ratio*(p2.Velocity-p1.Velocity)

				// Para Bearing e GForce, usamos o valor do ponto anterior para simplicidade
				bearing := p1.Bearing
				gForce := p1.GForce

				newPoint := GPSPoint{
					Time:     t1.Add(t),
					Lat:      lat,
					Lng:      lng,
					Altitude: altitude,
					Velocity: velocity,
					Bearing:  bearing,
					GForce:   gForce,
				}
				interpolated = append(interpolated, newPoint)
			}
		}
	}

	// Adiciona o último ponto
	interpolated = append(interpolated, points[len(points)-1])
	return interpolated
}

func (gp *GPSProcessor) GetPointsForTimeRange(startTime, endTime time.Time) []GPSPoint {
	var result []GPSPoint

	startIdx := -1
	endIdx := -1

	// Encontra os índices de início e fim baseados no tempo
	for i, point := range gp.points {
		if startIdx == -1 && (point.Time.Equal(startTime) || point.Time.After(startTime)) {
			startIdx = i
		}
		if endIdx == -1 && (point.Time.Equal(endTime) || point.Time.After(endTime)) {
			endIdx = i
			break // para a busca após encontrar o ponto final
		}
	}

	if startIdx != -1 && endIdx != -1 {
		result = gp.points[startIdx : endIdx+1]
	}

	fmt.Printf("DEBUG: Coletados %d pontos GPS para o vídeo (de %s a %s)\n", len(result), startTime.Format("15:04:05"), endTime.Format("15:04:05"))
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

func (gp *GPSProcessor) GetPointForTime(targetTime time.Time) (GPSPoint, bool) {
	if len(gp.points) == 0 {
		return GPSPoint{}, false
	}

	var closestPoint GPSPoint
	minDiff := time.Duration(math.MaxInt64)
	found := false

	for _, point := range gp.points {
		diff := point.Time.Sub(targetTime)
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			closestPoint = point
			found = true
		}
	}

	if found {
		log.Printf("DEBUG: Ponto GPS mais próximo encontrado: %s (diferença: %v)", closestPoint.Time.Format("15:04:05"), minDiff)
	}

	return closestPoint, found
}
