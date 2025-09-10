package gps

import (
	"fmt"
	"log"
	"math"
	"sync"
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
	points    []GPSPoint
	pointsMap map[int64]GPSPoint // Cache por timestamp para busca rápida
	mutex     sync.RWMutex
	cached    bool
}

func NewGPSProcessor() *GPSProcessor {
	return &GPSProcessor{}
}

func (gp *GPSProcessor) ProcessStreamDataOptimized(timeData, latlngData, velocityData, altitudeData []interface{}, startTime time.Time) error {
	gp.mutex.Lock()
	defer gp.mutex.Unlock()

	if len(timeData) != len(latlngData) {
		return fmt.Errorf("data length mismatch: time=%d, latlng=%d", len(timeData), len(latlngData))
	}

	// Pré-aloca slices para melhor performance
	rawPoints := make([]GPSPoint, 0, len(timeData))
	gp.pointsMap = make(map[int64]GPSPoint, len(timeData)*2) // Estima para pontos interpolados

	// Worker pool para processamento paralelo
	numWorkers := 4
	jobs := make(chan int, len(timeData))
	results := make(chan GPSPoint, len(timeData))
	var wg sync.WaitGroup

	// Inicia workers
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				point := gp.processGPSPoint(i, timeData, latlngData, velocityData, altitudeData, startTime)
				if point.Lat != 0 && point.Lng != 0 {
					results <- point
				}
			}
		}()
	}

	// Envia jobs
	go func() {
		for i := 0; i < len(timeData); i++ {
			jobs <- i
		}
		close(jobs)
	}()

	// Coleta resultados
	go func() {
		wg.Wait()
		close(results)
	}()

	// Ordena pontos por timestamp
	pointMap := make(map[time.Time]GPSPoint)
	for point := range results {
		pointMap[point.Time] = point
	}

	// Converte para slice ordenado
	for _, point := range pointMap {
		rawPoints = append(rawPoints, point)
	}

	if len(rawPoints) == 0 {
		return fmt.Errorf("nenhum ponto GPS válido encontrado")
	}

	// Calcula bearing e G-force sequencialmente (depende da ordem)
	gp.calculateDerivedValues(rawPoints)

	// Interpola pontos
	gp.points = gp.interpolatePointsOptimized(rawPoints)

	// Cria cache de busca rápida
	gp.buildTimeCache()
	gp.cached = true

	log.Printf("DEBUG: %d pontos GPS processados, %d após interpolação", len(rawPoints), len(gp.points))
	return nil
}

func (gp *GPSProcessor) processGPSPoint(i int, timeData, latlngData, velocityData, altitudeData []interface{}, startTime time.Time) GPSPoint {
	timeOffset, ok := timeData[i].(float64)
	if !ok {
		return GPSPoint{}
	}

	latlngInterface, ok := latlngData[i].([]interface{})
	if !ok || len(latlngInterface) != 2 {
		return GPSPoint{}
	}

	lat, ok1 := latlngInterface[0].(float64)
	lng, ok2 := latlngInterface[1].(float64)

	if !ok1 || !ok2 || !gp.isValidCoordinate(lat, lng) {
		return GPSPoint{}
	}

	point := GPSPoint{
		Time: startTime.Add(time.Duration(timeOffset) * time.Second),
		Lat:  lat,
		Lng:  lng,
	}

	// Adiciona dados opcionais
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

	return point
}

func (gp *GPSProcessor) isValidCoordinate(lat, lng float64) bool {
	return !(lat == 0 && lng == 0) &&
		lat >= -90 && lat <= 90 &&
		lng >= -180 && lng <= 180 &&
		!math.IsNaN(lat) && !math.IsNaN(lng)
}

func (gp *GPSProcessor) buildTimeCache() {
	for _, point := range gp.points {
		timestamp := point.Time.Unix()
		gp.pointsMap[timestamp] = point
	}
}

// GetPointForTimeOptimized - busca otimizada usando cache
func (gp *GPSProcessor) GetPointForTimeOptimized(targetTime time.Time) (GPSPoint, bool) {
	gp.mutex.RLock()
	defer gp.mutex.RUnlock()

	if !gp.cached {
		return gp.GetPointForTime(targetTime) // Fallback para método original
	}

	targetTimestamp := targetTime.Unix()

	// Busca exata primeiro
	if point, exists := gp.pointsMap[targetTimestamp]; exists {
		return point, true
	}

	// Busca aproximada (±5 segundos)
	for offset := int64(1); offset <= 5; offset++ {
		if point, exists := gp.pointsMap[targetTimestamp+offset]; exists {
			return point, true
		}
		if point, exists := gp.pointsMap[targetTimestamp-offset]; exists {
			return point, true
		}
	}

	// Fallback para busca linear
	return gp.GetPointForTime(targetTime)
}

// Spatial index para busca geográfica otimizada
type SpatialIndex struct {
	cells    map[string][]int // grid_key -> point_indices
	points   []GPSPoint
	cellSize float64
}

func (gp *GPSProcessor) buildSpatialIndex() *SpatialIndex {
	si := &SpatialIndex{
		cells:    make(map[string][]int),
		points:   gp.points,
		cellSize: 0.001, // ~100m resolution
	}

	for i, point := range gp.points {
		key := si.getCellKey(point.Lat, point.Lng)
		si.cells[key] = append(si.cells[key], i)
	}

	return si
}

func (si *SpatialIndex) getCellKey(lat, lng float64) string {
	cellLat := math.Floor(lat / si.cellSize)
	cellLng := math.Floor(lng / si.cellSize)
	return fmt.Sprintf("%.0f,%.0f", cellLat, cellLng)
}

func (si *SpatialIndex) findNearestPoint(targetLat, targetLng float64) (GPSPoint, bool) {
	key := si.getCellKey(targetLat, targetLng)

	// Busca na célula atual e adjacentes
	searchKeys := []string{key}
	for dlat := -1; dlat <= 1; dlat++ {
		for dlng := -1; dlng <= 1; dlng++ {
			if dlat == 0 && dlng == 0 {
				continue
			}
			adjKey := fmt.Sprintf("%.0f,%.0f",
				math.Floor(targetLat/si.cellSize)+float64(dlat),
				math.Floor(targetLng/si.cellSize)+float64(dlng))
			searchKeys = append(searchKeys, adjKey)
		}
	}

	var closestPoint GPSPoint
	minDist := math.MaxFloat64
	found := false

	for _, searchKey := range searchKeys {
		if indices, exists := si.cells[searchKey]; exists {
			for _, idx := range indices {
				point := si.points[idx]
				dist := haversineDistance(GPSPoint{Lat: targetLat, Lng: targetLng}, point)
				if dist < minDist {
					minDist = dist
					closestPoint = point
					found = true
				}
			}
		}
	}

	return closestPoint, found
}

// calculateDerivedValues calcula bearing e G-force para os pontos
func (gp *GPSProcessor) calculateDerivedValues(points []GPSPoint) {
	for i := 1; i < len(points); i++ {
		prevPoint := points[i-1]
		currentPoint := &points[i]

		currentPoint.Bearing = gp.calculateBearing(prevPoint, *currentPoint)
		currentPoint.GForce = gp.calculateGForce(prevPoint, *currentPoint)
	}

	// O primeiro ponto herda os valores do segundo (se existir)
	if len(points) > 1 {
		points[0].Bearing = points[1].Bearing
		points[0].GForce = points[1].GForce
	}
}

// interpolatePointsOptimized - versão otimizada da interpolação
func (gp *GPSProcessor) interpolatePointsOptimized(points []GPSPoint) []GPSPoint {
	if len(points) < 2 {
		return points
	}

	var interpolated []GPSPoint

	// Pre-aloca com estimativa de tamanho
	estimatedSize := len(points) * 2 // Estimativa conservadora
	interpolated = make([]GPSPoint, 0, estimatedSize)

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
			// Calcula quantos pontos intermediários são necessários
			numInterpolated := int(duration.Seconds()) - 1
			if numInterpolated > 60 { // Limita para evitar muitos pontos
				numInterpolated = 60
			}

			for j := 1; j <= numInterpolated; j++ {
				ratio := float64(j) / float64(numInterpolated+1)

				lat := p1.Lat + ratio*(p2.Lat-p1.Lat)
				lng := p1.Lng + ratio*(p2.Lng-p1.Lng)
				altitude := p1.Altitude + ratio*(p2.Altitude-p1.Altitude)
				velocity := p1.Velocity + ratio*(p2.Velocity-p1.Velocity)

				// Interpolação linear para bearing (considerando wraparound)
				bearing := gp.interpolateBearing(p1.Bearing, p2.Bearing, ratio)
				gForce := p1.GForce + ratio*(p2.GForce-p1.GForce)

				newPoint := GPSPoint{
					Time:     t1.Add(time.Duration(float64(duration) * ratio)),
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

// interpolateBearing interpola bearing considerando o wraparound de 0-360 graus
func (gp *GPSProcessor) interpolateBearing(bearing1, bearing2, ratio float64) float64 {
	diff := bearing2 - bearing1

	// Ajusta para o caminho mais curto
	if diff > 180 {
		diff -= 360
	} else if diff < -180 {
		diff += 360
	}

	result := bearing1 + ratio*diff

	// Normaliza para 0-360
	if result < 0 {
		result += 360
	} else if result >= 360 {
		result -= 360
	}

	return result
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

// haversineDistance calcula a distância entre dois pontos GPS.
func haversineDistance(p1, p2 GPSPoint) float64 {
	const R = 6371 // Raio da Terra em km
	lat1Rad := p1.Lat * math.Pi / 180
	lon1Rad := p1.Lng * math.Pi / 180
	lat2Rad := p2.Lat * math.Pi / 180
	lon2Rad := p2.Lng * math.Pi / 180

	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// GetPointForCoords encontra o ponto GPS mais próximo de uma coordenada específica.
func (gp *GPSProcessor) GetPointForCoords(targetLat, targetLng float64) (GPSPoint, bool) {
	if len(gp.points) == 0 {
		return GPSPoint{}, false
	}

	var closestPoint GPSPoint
	minDist := math.MaxFloat64
	found := false

	targetPoint := GPSPoint{Lat: targetLat, Lng: targetLng}

	for _, point := range gp.points {
		dist := haversineDistance(targetPoint, point)
		if dist < minDist {
			minDist = dist
			closestPoint = point
			found = true
		}
	}

	if found {
		log.Printf("DEBUG: Ponto GPS mais próximo do clique encontrado: %s (distância: %.2f m)", closestPoint.Time.Format("15:04:05"), minDist*1000)
	}

	return closestPoint, found
}

// GetAllPoints retorna todos os pontos GPS processados e interpolados.
func (gp *GPSProcessor) GetAllPoints() []GPSPoint {
	return gp.points
}
