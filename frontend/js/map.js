console.log('🗺️ map.js carregando (versão corrigida)...');

/**
 * Inicializa e exibe o mapa para uma atividade específica.
 * @param {object} activity - Os dados da atividade.
 */
async function displayMap(activity) {
    console.log("🗺️ Inicializando mapa para a atividade:", activity.name);

    try {
        // Limpa mapa anterior se existir
        if (activityMap) {
            console.log("🧹 Removendo mapa anterior...");
            activityMap.remove();
            activityMap = null;
        }
        
        // Limpa marcadores anteriores
        if (videoStartMarker) {
            videoStartMarker = null;
        }
        if (activityPolyline) {
            activityPolyline = null;
        }
        
        // Reseta a sincronização
        manualSyncTime = "";

        // Verifica se o container do mapa existe
        const mapContainer = document.getElementById('mapContainer');
        if (!mapContainer) {
            throw new Error('Container do mapa não encontrado');
        }

        // Limpa o container
        mapContainer.innerHTML = '';
        
        console.log("📍 Criando novo mapa...");
        
        // Aguarda um pouco para garantir que o DOM esteja pronto
        await new Promise(resolve => setTimeout(resolve, 100));

        // Inicializa o mapa Leaflet
        activityMap = L.map('mapContainer').setView([0, 0], 2);
        
        console.log("🗺️ Mapa criado, adicionando tiles...");
        
        // Adiciona camada de tiles
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors',
            maxZoom: 18
        }).addTo(activityMap);

        console.log("📊 Carregando dados GPS...");
        
        // Carrega e exibe a trajetória
        await loadInterpolatedTrajectory(activity);
        
        console.log("✅ Mapa inicializado com sucesso!");

    } catch (error) {
        console.error("❌ ERRO AO EXIBIR O MAPA:", error);
        const mapContainer = document.getElementById('mapContainer');
        if (mapContainer) {
            mapContainer.innerHTML = `
                <div style="display: flex; align-items: center; justify-content: center; height: 100%; color: var(--error-text); flex-direction: column; gap: 10px;">
                    <div style="font-size: 1.2rem;">❌ Erro ao carregar o mapa</div>
                    <div style="font-size: 0.9rem; opacity: 0.8;">${error.message}</div>
                    <button onclick="displayMap(selectedActivity)" style="margin-top: 10px;">Tentar Novamente</button>
                </div>
            `;
        }
    }
}
/**
 * Carrega e exibe a trajetória interpolada com gradiente de velocidade.
 * @param {object} activity - A atividade para a qual carregar a trajetória.
 */
async function loadInterpolatedTrajectory(activity) {
    try {
        console.log("📈 Carregando trajeto detalhado...");
        showMessage(result, 'Carregando trajeto detalhado...', 'info');

        const fullTrajectory = await window.go.main.App.GetFullGPSTrajectory(activity.id);

        if (!fullTrajectory || fullTrajectory.length === 0) {
            console.log("⚠️ Sem dados de trajeto completo, usando trajeto simplificado");
            loadFallbackTrajectory(activity);
            return;
        }

        console.log(`✅ Trajeto completo carregado: ${fullTrajectory.length} pontos interpolados`);

        // Cria a trajetória principal COM GRADIENTE REAL
        createSpeedGradientTrajectory(fullTrajectory);

        // Adiciona marcadores de início e fim
        const startPoint = fullTrajectory[0];
        const endPoint = fullTrajectory[fullTrajectory.length - 1];

        console.log("📍 Adicionando marcadores de início e fim...");
        
        L.marker([startPoint.lat, startPoint.lng]).addTo(activityMap)
            .bindPopup('🏁 Início da atividade')
            .openPopup();
            
        L.marker([endPoint.lat, endPoint.lng]).addTo(activityMap)
            .bindPopup('🏆 Fim da atividade');

        // Ajusta a visualização para mostrar toda a trajetória
        const bounds = L.latLngBounds(fullTrajectory.map(p => [p.lat, p.lng]));
        activityMap.fitBounds(bounds, { padding: [20, 20] });
        
        // Adiciona legenda de velocidade
        addSpeedLegend();
        
        console.log("🎯 Mapa ajustado para mostrar toda a trajetória");
        
        showMessage(result, `✅ Trajeto colorido carregado: ${fullTrajectory.length} pontos GPS`, 'success');

    } catch (error) {
        console.error("❌ Erro ao carregar trajeto:", error);
        showMessage(result, `Erro ao carregar trajeto: ${error.message}`, 'error');
        loadFallbackTrajectory(activity);
    }
}

/**
 * Cria a polilinha no mapa colorida pela velocidade - VERSÃO CORRIGIDA.
 * @param {Array} trajectoryPoints - Os pontos da trajetória.
 */
function createSpeedGradientTrajectory(trajectoryPoints) {
    console.log(`🎨 Criando trajeto com gradiente real de velocidade: ${trajectoryPoints.length} pontos...`);
    
    if (trajectoryPoints.length < 2) {
        console.warn('⚠️ Pontos insuficientes para criar trajeto');
        return;
    }

    // Grupo para armazenar todos os segmentos
    const trajectoryGroup = L.layerGroup().addTo(activityMap);
    
    // Cria segmentos coloridos individualmente
    for (let i = 0; i < trajectoryPoints.length - 1; i++) {
        const currentPoint = trajectoryPoints[i];
        const nextPoint = trajectoryPoints[i + 1];
        
        // Velocidade do segmento atual (em km/h)
        const segmentSpeed = currentPoint.velocity * 3.6;
        
        // Coordenadas do segmento
        const segmentCoords = [
            [currentPoint.lat, currentPoint.lng],
            [nextPoint.lat, nextPoint.lng]
        ];
        
        // Cria polilinha individual para este segmento
        const segmentLine = L.polyline(segmentCoords, {
            color: getSpeedColor(segmentSpeed),
            weight: 4,
            opacity: 0.8,
            smoothFactor: 1.0
        });
        
        // Adiciona popup com info do segmento
        segmentLine.bindPopup(`
            <div style="font-size: 12px;">
                <strong>📍 Segmento ${i + 1}</strong><br>
                🏃 Velocidade: ${segmentSpeed.toFixed(1)} km/h<br>
                ⏰ Tempo: ${new Date(currentPoint.time).toLocaleTimeString('pt-BR')}<br>
                📏 Altitude: ${currentPoint.altitude.toFixed(0)}m
            </div>
        `);
        
        // Adiciona handler de clique para sincronização
        segmentLine.on('click', (e) => {
            handleSegmentClick(e, currentPoint);
        });
        
        // Adiciona ao grupo
        trajectoryGroup.addLayer(segmentLine);
    }
    
    // Armazena referência global
    activityPolyline = trajectoryGroup;
    
    // Calcula estatísticas para log
    const speeds = trajectoryPoints.map(p => p.velocity * 3.6);
    const avgSpeed = speeds.reduce((sum, speed) => sum + speed, 0) / speeds.length;
    const maxSpeed = Math.max(...speeds);
    const minSpeed = Math.min(...speeds);
    
    console.log(`✅ Trajeto criado com gradiente de velocidade:`);
    console.log(`   📊 Velocidade média: ${avgSpeed.toFixed(1)} km/h`);
    console.log(`   🚀 Velocidade máxima: ${maxSpeed.toFixed(1)} km/h`);
    console.log(`   🐌 Velocidade mínima: ${minSpeed.toFixed(1)} km/h`);
    console.log(`   🎨 ${trajectoryPoints.length - 1} segmentos coloridos`);
}

/**
 * Handler de clique otimizado para segmentos individuais.
 * @param {L.LeafletMouseEvent} e - Evento de clique
 * @param {object} point - Ponto GPS do segmento
 */
function handleSegmentClick(e, point) {
    console.log(`🖱️ Clique no segmento: ${point.time}`);
    
    manualSyncTime = point.time;
    updateVideoStartMarker(point.lat, point.lng, '▶️ Início Manual do Vídeo');
    
    const timeStr = new Date(point.time).toLocaleTimeString('pt-BR');
    const speedStr = (point.velocity * 3.6).toFixed(1);
    
    showMessage(result, `🎯 Sincronização: ${timeStr} (${speedStr} km/h)`, 'success');
}

/**
 * Lida com o clique na trajetória para sincronização manual.
 * @param {L.LeafletMouseEvent} e - O evento de clique do Leaflet.
 * @param {Array} trajectoryPoints - Os pontos da trajetória para encontrar o mais próximo.
 */
function handleTrajectoryClick(e, trajectoryPoints) {
    console.log("🖱️ Clique no trajeto detectado, buscando ponto mais próximo...");
    
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
        console.log(`✅ Ponto mais próximo encontrado: ${closestPoint.time} (${minDistance.toFixed(2)}m de distância)`);
        manualSyncTime = closestPoint.time;
        updateVideoStartMarker(closestPoint.lat, closestPoint.lng, '▶️ Início Manual do Vídeo');
        
        const timeStr = new Date(closestPoint.time).toLocaleTimeString('pt-BR');
        const speedStr = (closestPoint.velocity * 3.6).toFixed(1);
        showMessage(result, `🎯 Sincronização definida: ${timeStr} (${speedStr} km/h)`, 'success');
    } else {
        console.log("❌ Nenhum ponto encontrado próximo ao clique");
        showMessage(result, 'Não foi possível encontrar um ponto GPS próximo', 'error');
    }
}

/**
 * Retorna uma cor baseada na velocidade em km/h - CORES MAIS CONTRASTANTES.
 * @param {number} speedKmh - A velocidade em km/h.
 * @returns {string} O código hexadecimal da cor.
 */
function getSpeedColor(speedKmh) {
    // Velocidades muito baixas ou parada (0-3 km/h) - CINZA ESCURO
    if (speedKmh <= 3) return '#6c757d';
    
    // Muito lento (3-8 km/h) - AZUL
    if (speedKmh <= 8) return '#0d6efd';
    
    // Lento (8-15 km/h) - VERDE
    if (speedKmh <= 15) return '#198754';
    
    // Moderado (15-25 km/h) - AMARELO/LARANJA
    if (speedKmh <= 25) return '#fd7e14';
    
    // Rápido (25-40 km/h) - LARANJA ESCURO
    if (speedKmh <= 40) return '#d63384';
    
    // Muito rápido (40+ km/h) - VERMELHO
    return '#dc3545';
}

/**
 * Versão alternativa com gradiente suave usando bibliotecas externas.
 * @param {Array} trajectoryPoints - Os pontos da trajetória.
 */
function createSmoothSpeedGradient(trajectoryPoints) {
    console.log(`🌈 Criando trajeto com gradiente suave: ${trajectoryPoints.length} pontos...`);
    
    // Cria um canvas para gerar o gradiente
    const canvas = document.createElement('canvas');
    const ctx = canvas.getContext('2d');
    canvas.width = trajectoryPoints.length;
    canvas.height = 1;
    
    // Cria gradiente baseado nas velocidades
    const gradient = ctx.createLinearGradient(0, 0, canvas.width, 0);
    
    trajectoryPoints.forEach((point, index) => {
        const speed = point.velocity * 3.6;
        const position = index / (trajectoryPoints.length - 1);
        gradient.addColorStop(position, getSpeedColor(speed));
    });
    
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    
    // Para uma implementação completa, seria necessário usar bibliotecas como:
    // - Leaflet.hotline
    // - Leaflet.multicolor-polyline
    // Por enquanto, usamos a versão por segmentos acima
    
    console.log('ℹ️ Para gradiente suave completo, considere usar Leaflet.hotline');
}

/**
 * Cria legenda de velocidade no mapa.
 */
function addSpeedLegend() {
    if (!activityMap) return;
    
    const legend = L.control({ position: 'bottomright' });
    
    legend.onAdd = function() {
        const div = L.DomUtil.create('div', 'speed-legend');
        div.innerHTML = `
            <div style="background: rgba(13,17,23,0.9); padding: 10px; border-radius: 5px; font-size: 12px; color: white; border: 1px solid #30363d;">
                <div style="font-weight: bold; margin-bottom: 5px;">🏃 Velocidade</div>
                <div style="display: flex; align-items: center; margin: 2px 0;">
                    <div style="width: 15px; height: 3px; background: #6c757d; margin-right: 5px;"></div>
                    Parado (0-3 km/h)
                </div>
                <div style="display: flex; align-items: center; margin: 2px 0;">
                    <div style="width: 15px; height: 3px; background: #0d6efd; margin-right: 5px;"></div>
                    Muito lento (3-8 km/h)
                </div>
                <div style="display: flex; align-items: center; margin: 2px 0;">
                    <div style="width: 15px; height: 3px; background: #198754; margin-right: 5px;"></div>
                    Lento (8-15 km/h)
                </div>
                <div style="display: flex; align-items: center; margin: 2px 0;">
                    <div style="width: 15px; height: 3px; background: #fd7e14; margin-right: 5px;"></div>
                    Moderado (15-25 km/h)
                </div>
                <div style="display: flex; align-items: center; margin: 2px 0;">
                    <div style="width: 15px; height: 3px; background: #d63384; margin-right: 5px;"></div>
                    Rápido (25-40 km/h)
                </div>
                <div style="display: flex; align-items: center; margin: 2px 0;">
                    <div style="width: 15px; height: 3px; background: #dc3545; margin-right: 5px;"></div>
                    Muito rápido (40+ km/h)
                </div>
            </div>
        `;
        return div;
    };
    
    legend.addTo(activityMap);
}
/**
 * Carrega uma trajetória simplificada como fallback.
 * @param {object} activity - Os dados da atividade contendo o `summary_polyline`.
 */
function loadFallbackTrajectory(activity) {
    console.log("🔄 Carregando trajeto simplificado (fallback)");
    
    try {
        if (activity.map && activity.map.summary_polyline) {
            const latlngs = L.Polyline.fromEncoded(activity.map.summary_polyline).getLatLngs();
            
            activityPolyline = L.polyline(latlngs, { 
                color: '#f85149', 
                weight: 3,
                opacity: 0.8 
            }).addTo(activityMap);
            
            // Handler de clique básico
            activityPolyline.on('click', handleMapClickBasic);
            
            // Ajusta visualização
            activityMap.fitBounds(activityPolyline.getBounds());
            
            // Marcadores simples
            if (latlngs.length > 0) {
                L.marker(latlngs[0]).addTo(activityMap).bindPopup('🏁 Início');
                L.marker(latlngs[latlngs.length - 1]).addTo(activityMap).bindPopup('🏆 Fim');
            }
            
            showMessage(result, 'Trajeto básico carregado (dados GPS limitados)', 'info');
            console.log("✅ Trajeto simplificado carregado com sucesso");
        } else {
            throw new Error('Nenhum dado de trajeto disponível');
        }
    } catch (error) {
        console.error("❌ Erro no fallback do trajeto:", error);
        showMessage(result, 'Erro: Nenhum dado GPS disponível para esta atividade', 'error');
    }
}

/**
 * Handler de clique básico para o mapa (fallback).
 * @param {L.LeafletMouseEvent} e - O evento de clique do Leaflet.
 */
async function handleMapClickBasic(e) {
    if (!selectedActivity) {
        console.log("❌ Nenhuma atividade selecionada");
        return;
    }

    try {
        console.log(`🖱️ Clique básico no mapa detectado em: ${e.latlng.lat}, ${e.latlng.lng}`);
        showMessage(result, 'Buscando ponto GPS mais próximo...', 'info');

        const point = await window.go.main.App.GetGPSPointForMapClick(selectedActivity.id, e.latlng.lat, e.latlng.lng);
        
        if (point && point.lat && point.lng) {
            console.log(`✅ Ponto de sincronização encontrado: ${point.time}`);
            manualSyncTime = point.time;
            updateVideoStartMarker(point.lat, point.lng, '▶️ Início Manual do Vídeo');
            
            const timeStr = new Date(point.time).toLocaleTimeString('pt-BR');
            showMessage(result, `🎯 Sincronização definida: ${timeStr}`, 'success');
        } else {
            console.log("❌ Nenhum ponto GPS encontrado");
            showMessage(result, 'Não foi possível encontrar um ponto GPS próximo', 'error');
        }

    } catch (error) {
        console.error("❌ Erro ao definir ponto de sincronização:", error);
        showMessage(result, `Erro: ${error.message}`, 'error');
    }
}

/**
 * Atualiza ou cria o marcador de início do vídeo no mapa.
 * @param {number} lat - Latitude do marcador.
 * @param {number} lng - Longitude do marcador.
 * @param {string} popupText - O texto para o popup do marcador.
 */
function updateVideoStartMarker(lat, lng, popupText) {
    if (!activityMap) {
        console.error("❌ Mapa não está inicializado para atualizar o marcador");
        return;
    }

    try {
        // Remove marcador anterior se existir
        if (videoStartMarker) {
            activityMap.removeLayer(videoStartMarker);
            videoStartMarker = null;
        }

        // Cria ícone customizado azul
        const blueIcon = new L.Icon({
            // iconUrl: 'https://raw.githubusercontent.com/pointhi/leaflet-color-markers/master/img/marker-icon-2x-blue.png',
            iconUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-icon-2x.png',
            // shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/0.7.7/images/marker-shadow.png',
            shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/1.9.4/images/marker-shadow.png',
            iconSize: [25, 41], 
            iconAnchor: [12, 41], 
            popupAnchor: [1, -34], 
            shadowSize: [41, 41]
        });

        // Cria novo marcador
        videoStartMarker = L.marker([lat, lng], { icon: blueIcon })
            .addTo(activityMap)
            .bindPopup(popupText)
            .openPopup();

        // Centraliza no marcador
        setTimeout(() => {
            if (activityMap) {
                activityMap.setView([lat, lng], Math.max(activityMap.getZoom(), 15));
                console.log("📍 Marcador de início do vídeo atualizado e centralizado");
            }
        }, 100);

    } catch (error) {
        console.error("❌ Erro ao atualizar marcador:", error);
    }
}

/**
 * Força a re-renderização do mapa (útil para problemas de layout).
 */
function invalidateMapSize() {
    if (activityMap) {
        setTimeout(() => {
            activityMap.invalidateSize();
            console.log("🔄 Tamanho do mapa revalidado");
        }, 100);
    }
}

/**
 * Função de debug para verificar estado do mapa.
 */
function debugMapState() {
    console.log("🐛 Estado do mapa:", {
        mapExists: !!activityMap,
        containerExists: !!document.getElementById('mapContainer'),
        selectedActivity: !!selectedActivity,
        polylineExists: !!activityPolyline,
        markerExists: !!videoStartMarker
    });
}

// Expõe funções para debug global
if (typeof window !== 'undefined') {
    window.debugMapState = debugMapState;
    window.invalidateMapSize = invalidateMapSize;
}