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

func (gp *GPSProcessor) GetPointsForTimeRange(startTime, endTime time.Time) []GPSPoint {
	var result []GPSPoint

	for _, point := range gp.points {
		if (point.Time.Equal(startTime) || point.Time.After(startTime)) &&
			(point.Time.Equal(endTime) || point.Time.Before(endTime)) {
			result = append(result, point)
		}
	}

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

// GetPointForTime encontra o primeiro ponto GPS em ou após um tempo específico.
func (gp *GPSProcessor) GetPointForTime(targetTime time.Time) (GPSPoint, bool) {
	for _, point := range gp.points {
		if !point.Time.Before(targetTime) {
			return point, true
		}
	}
	if len(gp.points) > 0 {
		return gp.points[len(gp.points)-1], true
	}
	return GPSPoint{}, false
}
