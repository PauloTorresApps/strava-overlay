console.log('🗺️ map.js carregando...');

/**
 * Inicializa e exibe o mapa para uma atividade específica.
 * @param {object} activity - Os dados da atividade.
 */
async function displayMap(activity) {
    console.log("Inicializando mapa para a atividade:", activity.name);

    try {
        if (activityMap) {
            activityMap.remove();
            activityMap = null;
        }
        manualSyncTime = ""; // Reseta a sincronização

        // Inicializa o mapa Leaflet
        activityMap = L.map('mapContainer');
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors'
        }).addTo(activityMap);

        await loadInterpolatedTrajectory(activity);

    } catch (error) {
        console.error("ERRO AO EXIBIR O MAPA:", error);
        if (mapContainer) {
            mapContainer.innerHTML = `<div class="error">Erro ao carregar o mapa: ${error.message}</div>`;
        }
    }
}

/**
 * Carrega e exibe a trajetória interpolada com gradiente de velocidade.
 * @param {object} activity - A atividade para a qual carregar a trajetória.
 */
async function loadInterpolatedTrajectory(activity) {
    try {
        showMessage(result, 'Carregando trajeto detalhado...', 'info');

        const fullTrajectory = await window.go.main.App.GetFullGPSTrajectory(activity.id);

        if (!fullTrajectory || fullTrajectory.length === 0) {
            loadFallbackTrajectory(activity);
            return;
        }

        createSpeedGradientTrajectory(fullTrajectory);

        const startPoint = fullTrajectory[0];
        const endPoint = fullTrajectory[fullTrajectory.length - 1];

        L.marker([startPoint.lat, startPoint.lng]).addTo(activityMap).bindPopup('🏁 Início');
        L.marker([endPoint.lat, endPoint.lng]).addTo(activityMap).bindPopup('🏆 Fim');

        const bounds = L.latLngBounds(fullTrajectory.map(p => [p.lat, p.lng]));
        activityMap.fitBounds(bounds, { padding: [20, 20] });
        
        showMessage(result, `✅ Trajeto com ${fullTrajectory.length} pontos carregado`, 'success');

    } catch (error) {
        console.error("Erro ao carregar trajeto:", error);
        showMessage(result, `Erro: ${error}`, 'error');
        loadFallbackTrajectory(activity);
    }
}

/**
 * Cria a polilinha no mapa colorida pela velocidade.
 * @param {Array} trajectoryPoints - Os pontos da trajetória.
 */
function createSpeedGradientTrajectory(trajectoryPoints) {
    const allLatLngs = trajectoryPoints.map(p => [p.lat, p.lng]);
    const avgSpeed = trajectoryPoints.reduce((sum, p) => sum + (p.velocity * 3.6), 0) / trajectoryPoints.length;

    activityPolyline = L.polyline(allLatLngs, {
        color: getSpeedColor(avgSpeed),
        weight: 4,
        opacity: 0.8
    }).addTo(activityMap);

    activityPolyline.on('click', (e) => handleTrajectoryClick(e, trajectoryPoints));
    activityPolyline.bindPopup(`Velocidade média: ${avgSpeed.toFixed(1)} km/h`);
}

/**
 * Lida com o clique na trajetória para sincronização manual.
 * @param {L.LeafletMouseEvent} e - O evento de clique do Leaflet.
 * @param {Array} trajectoryPoints - Os pontos da trajetória para encontrar o mais próximo.
 */
function handleTrajectoryClick(e, trajectoryPoints) {
    const clickLatLng = e.latlng;
    let closestPoint = null;
    let minDistance = Infinity;

    trajectoryPoints.forEach(point => {
        const distance = clickLatLng.distanceTo([point.lat, point.lng]);
        if (distance < minDistance) {
            minDistance = distance;
            closestPoint = point;
        }
    });

    if (closestPoint) {
        manualSyncTime = closestPoint.time;
        updateVideoStartMarker(closestPoint.lat, closestPoint.lng, '▶️ Início Manual do Vídeo');
        const timeStr = new Date(closestPoint.time).toLocaleTimeString('pt-BR');
        showMessage(result, `🎯 Sincronização manual definida para: ${timeStr}`, 'success');
    }
}

/**
 * Retorna uma cor baseada na velocidade em km/h.
 * @param {number} speedKmh - A velocidade em km/h.
 * @returns {string} O código hexadecimal da cor.
 */
function getSpeedColor(speedKmh) {
    if (speedKmh > 40) return '#dc3545'; // Vermelho
    if (speedKmh > 25) return '#fd7e14'; // Laranja
    if (speedKmh > 15) return '#ffc107'; // Amarelo
    if (speedKmh > 8) return '#28a745';  // Verde
    return '#6c757d';                   // Cinza
}

/**
 * Carrega uma trajetória simplificada como fallback.
 * @param {object} activity - Os dados da atividade contendo o `summary_polyline`.
 */
function loadFallbackTrajectory(activity) {
    if (activity.map?.summary_polyline) {
        const latlngs = L.Polyline.fromEncoded(activity.map.summary_polyline).getLatLngs();
        activityPolyline = L.polyline(latlngs, { color: '#f85149', weight: 3 }).addTo(activityMap);
        activityMap.fitBounds(activityPolyline.getBounds());
        showMessage(result, 'Trajeto básico carregado (dados detalhados indisponíveis)', 'info');
    }
}

/**
 * Atualiza ou cria o marcador de início do vídeo no mapa.
 * @param {number} lat - Latitude do marcador.
 * @param {number} lng - Longitude do marcador.
 * @param {string} popupText - O texto para o popup do marcador.
 */
function updateVideoStartMarker(lat, lng, popupText) {
    if (!activityMap) return;
    if (videoStartMarker) videoStartMarker.remove();

    const blueIcon = new L.Icon({
        iconUrl: 'https://raw.githubusercontent.com/pointhi/leaflet-color-markers/master/img/marker-icon-2x-blue.png',
        shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/0.7.7/images/marker-shadow.png',
        iconSize: [25, 41], iconAnchor: [12, 41], popupAnchor: [1, -34], shadowSize: [41, 41]
    });

    videoStartMarker = L.marker([lat, lng], { icon: blueIcon })
        .addTo(activityMap)
        .bindPopup(popupText)
        .openPopup();

    activityMap.setView([lat, lng], 16);
}
