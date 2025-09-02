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
	gp.points = make([]GPSPoint, 0, len(timeData))

	validPoints := 0
	for i := 0; i < len(timeData); i++ {
		timeOffset, ok := timeData[i].(float64)
		if !ok {
			log.Printf("DEBUG: Ignorando ponto %d - timeOffset inválido", i)
			continue
		}

		latlngInterface, ok := latlngData[i].([]interface{})
		if !ok || len(latlngInterface) != 2 {
			log.Printf("DEBUG: Ignorando ponto %d - latlng inválido", i)
			continue
		}

		lat, ok1 := latlngInterface[0].(float64)
		lng, ok2 := latlngInterface[1].(float64)

		if !ok1 || !ok2 || lat == 0 && lng == 0 ||
			lat < -90 || lat > 90 || lng < -180 || lng > 180 ||
			math.IsNaN(lat) || math.IsNaN(lng) {
			log.Printf("DEBUG: Ignorando ponto %d - coordenadas inválidas: lat=%.6f, lng=%.6f", i, lat, lng)
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

		if len(gp.points) > 0 {
			prevPoint := gp.points[len(gp.points)-1]
			point.Bearing = gp.calculateBearing(prevPoint, point)
			point.GForce = gp.calculateGForce(prevPoint, point)
		}

		gp.points = append(gp.points, point)
		validPoints++
	}

	if validPoints == 0 {
		return fmt.Errorf("nenhum ponto GPS válido encontrado")
	}

	log.Printf("DEBUG: %d pontos GPS válidos processados de %d totais", validPoints, len(timeData))
	return nil
}

func (gp *GPSProcessor) GetPointsForTimeRange(startTime, endTime time.Time) []GPSPoint {
	var result []GPSPoint

	startPoint, startFound := gp.GetPointForTime(startTime)
	if !startFound {
		return result
	}

	endPoint, endFound := gp.GetPointForTime(endTime)
	if !endFound {
		return result
	}

	collecting := false
	for _, point := range gp.points {
		if point.Time.Equal(startPoint.Time) {
			collecting = true
		}

		if collecting {
			result = append(result, point)
		}

		if point.Time.Equal(endPoint.Time) {
			break
		}
	}

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

func (gp *GPSProcessor) GetPointForTime(targetTime time.Time) (GPSPoint, bool) {
	if len(gp.points) == 0 {
		log.Printf("DEBUG: Nenhum ponto GPS disponível")
		return GPSPoint{}, false
	}

	var closestPoint GPSPoint
	var minDiff time.Duration
	found := false

	log.Printf("DEBUG: Buscando ponto GPS para %s entre %d pontos",
		targetTime.Format("15:04:05"), len(gp.points))

	for i, point := range gp.points {
		if point.Lat == 0 && point.Lng == 0 {
			continue
		}
		if point.Lat < -90 || point.Lat > 90 || point.Lng < -180 || point.Lng > 180 {
			continue
		}

		diff := point.Time.Sub(targetTime)
		if diff < 0 {
			diff = -diff
		}

		if !found || diff < minDiff || (diff == minDiff && point.Time.After(closestPoint.Time)) {
			closestPoint = point
			minDiff = diff
			found = true
		}

		if i < 3 || i >= len(gp.points)-3 {
			log.Printf("DEBUG: Ponto %d: %s - Lat: %.6f, Lng: %.6f",
				i, point.Time.Format("15:04:05"), point.Lat, point.Lng)
		}
	}

	if found {
		log.Printf("DEBUG: Ponto GPS encontrado: %s (diferença: %v) - Lat: %.6f, Lng: %.6f",
			closestPoint.Time.Format("15:04:05"), minDiff, closestPoint.Lat, closestPoint.Lng)
	} else {
		log.Printf("DEBUG: Nenhum ponto GPS válido encontrado")
	}

	return closestPoint, found
}
