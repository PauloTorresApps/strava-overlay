
/**
 * NOVA FUNÇÃO: Recupera chave de API do ConfigService
 */
function getAPIKeyFromConfig(provider) {
    if (!window.configService || !window.configService.initialized) {
        console.warn('⚠️ ConfigService não inicializado ainda');
        return null;
    }
    
    return window.configService.getAPIKey(provider);
}

/**
 * NOVA FUNÇÃO: Atualiza MAP_PROVIDERS com chaves do backend
 */
function updateMapProvidersWithKeys() {
    console.log('🔧 Atualizando MAP_PROVIDERS com chaves do backend...');
    
    // Thunderforest
    const thunderforestKey = getAPIKeyFromConfig('thunderforest');
   
    if (thunderforestKey) {
        // Adiciona novos provedores Thunderforest
        MAP_PROVIDERS.thunderforest_outdoors = {
            name: 'Thunderforest Outdoors',
            url: `https://tile.thunderforest.com/outdoors/{z}/{x}/{y}.png?apikey=${thunderforestKey}`,
            attribution: '© Thunderforest © OpenStreetMap contributors',
            darkFilter: false
        };
        
        MAP_PROVIDERS.thunderforest_landscape = {
            name: 'Thunderforest Landscape', 
            url: `https://tile.thunderforest.com/landscape/{z}/{x}/{y}.png?apikey=${thunderforestKey}`,
            attribution: '© Thunderforest © OpenStreetMap contributors',
            darkFilter: false
        };
        
        console.log('✅ Provedores Thunderforest adicionados');
    }
    
    // Mapbox
    const mapboxToken = getAPIKeyFromConfig('mapbox_public');
    if (mapboxToken) {
        console.log('🔑 Token Mapbox encontrado, adicionando provedores...');
        
        // Atualiza provedores Mapbox existentes substituindo {accessToken}
        Object.keys(MAP_PROVIDERS).forEach(key => {
            if (MAP_PROVIDERS[key].provider === 'mapbox') {
                MAP_PROVIDERS[key].url = MAP_PROVIDERS[key].url.replace('{accessToken}', mapboxToken);
            }
        });
        
        console.log('✅ Tokens Mapbox aplicados aos provedores existentes');
    }
    
    console.log(`🗺️ MAP_PROVIDERS atualizado com ${Object.keys(MAP_PROVIDERS).length} provedores`);
}


/**
 * SUBSTITUA a função displayMap existente por esta versão com seletor de mapas
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

        // Carrega preferência salva do usuário
        loadMapPreference();

        // Inicializa o mapa Leaflet
        activityMap = L.map('mapContainer').setView([0, 0], 2);
        
        console.log("🗺️ Mapa criado, adicionando camada de tiles...");
        
        // Adiciona camada de tiles baseada na preferência
        addTileLayer(currentMapProvider);
        
        // Adiciona controle de seleção de camadas
        addLayerSelector();

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

/**
 * Configurações dos diferentes tipos de mapa disponíveis - ATUALIZADO COM MAPBOX
 */
const MAP_PROVIDERS = {
    osm: {
        name: 'OpenStreetMap',
        url: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
        attribution: '© OpenStreetMap contributors',
        darkFilter: false
    },
    osmDark: {
        name: 'OpenStreetMap Dark',
        url: 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',
        attribution: '© OpenStreetMap contributors',
        darkFilter: true
    },
    satellite: {
        name: 'Satélite (Esri)',
        url: 'https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}',
        attribution: '© Esri, Maxar, Earthstar Geographics',
        darkFilter: false
    },
    terrain: {
        name: 'Terreno',
        url: 'https://{s}.tile.opentopomap.org/{z}/{x}/{y}.png',
        attribution: '© OpenTopoMap (CC-BY-SA)',
        darkFilter: false
    },
    cartodb_dark: {
        name: 'CartoDB Dark',
        url: 'https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png',
        attribution: '© OpenStreetMap © CartoDB',
        darkFilter: false
    },
    cyclemap: {
        name: 'Ciclovias',
        url: 'https://tile.thunderforest.com/cycle/{z}/{x}/{y}.png?apikey=API_KEY_AQUI',
        attribution: '© Thunderforest © OpenStreetMap contributors',
        darkFilter: false,
        requiresApiKey: false
    },
    // === MAPBOX STYLES ===
    mapbox_streets: {
        name: 'Mapbox Streets',
        url: 'https://api.mapbox.com/styles/v1/mapbox/streets-v12/tiles/{z}/{x}/{y}?access_token={accessToken}',
        attribution: '© Mapbox © OpenStreetMap contributors',
        darkFilter: false,
        requiresApiKey: true,
        provider: 'mapbox'
    },
    mapbox_satellite: {
        name: 'Mapbox Satellite',
        url: 'https://api.mapbox.com/styles/v1/mapbox/satellite-v9/tiles/{z}/{x}/{y}?access_token={accessToken}',
        attribution: '© Mapbox © OpenStreetMap contributors',
        darkFilter: false,
        requiresApiKey: true,
        provider: 'mapbox'
    },
    mapbox_satellite_streets: {
        name: 'Mapbox Satellite Streets',
        url: 'https://api.mapbox.com/styles/v1/mapbox/satellite-streets-v12/tiles/{z}/{x}/{y}?access_token={accessToken}',
        attribution: '© Mapbox © OpenStreetMap contributors',
        darkFilter: false,
        requiresApiKey: true,
        provider: 'mapbox'
    },
    mapbox_outdoors: {
        name: 'Mapbox Outdoors',
        url: 'https://api.mapbox.com/styles/v1/mapbox/outdoors-v12/tiles/{z}/{x}/{y}?access_token={accessToken}',
        attribution: '© Mapbox © OpenStreetMap contributors',
        darkFilter: false,
        requiresApiKey: true,
        provider: 'mapbox'
    },
    mapbox_dark: {
        name: 'Mapbox Dark',
        url: 'https://api.mapbox.com/styles/v1/mapbox/dark-v11/tiles/{z}/{x}/{y}?access_token={accessToken}',
        attribution: '© Mapbox © OpenStreetMap contributors',
        darkFilter: false,
        requiresApiKey: true,
        provider: 'mapbox'
    },
    mapbox_light: {
        name: 'Mapbox Light',
        url: 'https://api.mapbox.com/styles/v1/mapbox/light-v11/tiles/{z}/{x}/{y}?access_token={accessToken}',
        attribution: '© Mapbox © OpenStreetMap contributors',
        darkFilter: false,
        requiresApiKey: true,
        provider: 'mapbox'
    }
};

let currentMapProvider = 'osm'; // Padrão
let currentTileLayer = null;
let mapboxAccessToken = null;

/**
 * Configuração do token do Mapbox
 */
function setMapboxToken(token) {
    mapboxAccessToken = token;
    // Salva o token no localStorage para uso futuro
    localStorage.setItem('mapbox_token', token);
    console.log('🗺️ Token do Mapbox configurado');
}

/**
 * Carrega o token do Mapbox do localStorage
 */
function loadMapboxToken() {
    const savedToken = localStorage.getItem('mapbox_token');
    if (savedToken) {
        mapboxAccessToken = savedToken;
        console.log('🗺️ Token do Mapbox carregado do localStorage');
    }
    return savedToken;
}

/**
 * Inicializa o mapa com seletor de camadas
 */
function initializeMapWithLayerControl(activity) {
    console.log("🗺️ Inicializando mapa com controle de camadas...");

    // Cria o mapa
    activityMap = L.map('mapContainer').setView([0, 0], 2);
    
    // Adiciona camada inicial
    addTileLayer(currentMapProvider);
    
    // Adiciona controle de camadas
    addLayerSelector();
    
    console.log("✅ Mapa inicializado com seletor de camadas");
}

/**
 * Adiciona camada de tiles ao mapa com suporte ao Mapbox
 */
function addTileLayer(providerKey) {
    const provider = MAP_PROVIDERS[providerKey];
    
    if (!provider) {
        console.error('❌ Provedor de mapa inválido:', providerKey);
        return;
    }
    
    // Verifica se é um provedor Mapbox e se temos token
    if (provider.provider === 'mapbox') {
        if (!mapboxAccessToken) {
            console.warn('⚠️ Token do Mapbox não configurado. Tentando carregar...');
            loadMapboxToken();
            
            if (!mapboxAccessToken) {
                showMapboxTokenModal();
                return;
            }
        }
    }
    
    // Remove camada anterior se existir
    if (currentTileLayer) {
        activityMap.removeLayer(currentTileLayer);
    }
    
    // Prepara URL com token se necessário
    let tileUrl = provider.url;
    if (provider.provider === 'mapbox' && mapboxAccessToken) {
        tileUrl = tileUrl.replace('{accessToken}', mapboxAccessToken);
    }
    
    // Cria nova camada
    const tileOptions = {
        attribution: provider.attribution,
        maxZoom: 18,
        tileSize: 512,
        zoomOffset: -1
    };
    
    // Ajusta configurações específicas do Mapbox
    if (provider.provider === 'mapbox') {
        tileOptions.tileSize = 512;
        tileOptions.zoomOffset = -1;
    }
    
    currentTileLayer = L.tileLayer(tileUrl, tileOptions);
    
    // Adiciona ao mapa
    currentTileLayer.addTo(activityMap);
    
    // Aplica filtro dark se necessário
    applyMapTheme(provider.darkFilter);
    
    currentMapProvider = providerKey;
    console.log(`🗺️ Camada alterada para: ${provider.name}`);
}

/**
 * Aplica tema dark ao mapa
 */
function applyMapTheme(useDarkFilter) {
    const tilePane = document.querySelector('.leaflet-tile-pane');
    
    if (tilePane) {
        if (useDarkFilter) {
            tilePane.style.filter = 'invert(1) hue-rotate(180deg) brightness(95%) contrast(90%)';
        } else {
            tilePane.style.filter = 'none';
        }
    }
}

/**
 * Adiciona seletor de camadas ao mapa com categorias organizadas
 */
function addLayerSelector() {
    const layerControl = L.control({ position: 'topright' });    
    layerControl.onAdd = function() {
        const div = L.DomUtil.create('div', 'map-layer-control');
        
        // NOVA: Atualiza provedores antes de construir o seletor
        updateMapProvidersWithKeys();
        
        // Constrói opções organizadas por categoria
        let optionsHTML = '';
        
        // Categoria Esportes
        optionsHTML += '<optgroup label="🚴 Esportes">';
        if (MAP_PROVIDERS.cyclemap) optionsHTML += `<option value="cyclemap">${MAP_PROVIDERS.cyclemap.name}</option>`;
        if (MAP_PROVIDERS.mapbox_outdoors) optionsHTML += `<option value="mapbox_outdoors">${MAP_PROVIDERS.mapbox_outdoors.name}</option>`;
        if (MAP_PROVIDERS.thunderforest_outdoors) optionsHTML += `<option value="thunderforest_outdoors">${MAP_PROVIDERS.thunderforest_outdoors.name}</option>`;
        optionsHTML += '</optgroup>';
        
        // Categoria Satélite
        optionsHTML += '<optgroup label="🛰️ Satélite">';
        if (MAP_PROVIDERS.satellite) optionsHTML += `<option value="satellite">${MAP_PROVIDERS.satellite.name}</option>`;
        if (MAP_PROVIDERS.mapbox_satellite) optionsHTML += `<option value="mapbox_satellite">${MAP_PROVIDERS.mapbox_satellite.name}</option>`;
        if (MAP_PROVIDERS.mapbox_satellite_streets) optionsHTML += `<option value="mapbox_satellite_streets">${MAP_PROVIDERS.mapbox_satellite_streets.name}</option>`;
        optionsHTML += '</optgroup>';
        
        // Categoria Ruas
        optionsHTML += '<optgroup label="🌐 Ruas e Navegação">';
        if (MAP_PROVIDERS.osm) optionsHTML += `<option value="osm">${MAP_PROVIDERS.osm.name}</option>`;
        if (MAP_PROVIDERS.mapbox_streets) optionsHTML += `<option value="mapbox_streets">${MAP_PROVIDERS.mapbox_streets.name}</option>`;
        optionsHTML += '</optgroup>';
        
        // Categoria Escuro
        optionsHTML += '<optgroup label="🌙 Modo Escuro">';
        if (MAP_PROVIDERS.osmDark) optionsHTML += `<option value="osmDark">${MAP_PROVIDERS.osmDark.name}</option>`;
        if (MAP_PROVIDERS.mapbox_dark) optionsHTML += `<option value="mapbox_dark">${MAP_PROVIDERS.mapbox_dark.name}</option>`;
        if (MAP_PROVIDERS.cartodb_dark) optionsHTML += `<option value="cartodb_dark">${MAP_PROVIDERS.cartodb_dark.name}</option>`;
        optionsHTML += '</optgroup>';
        
        // Categoria Terreno
        optionsHTML += '<optgroup label="🏔️ Terreno">';
        if (MAP_PROVIDERS.terrain) optionsHTML += `<option value="terrain">${MAP_PROVIDERS.terrain.name}</option>`;
        if (MAP_PROVIDERS.mapbox_light) optionsHTML += `<option value="mapbox_light">${MAP_PROVIDERS.mapbox_light.name}</option>`;
        if (MAP_PROVIDERS.thunderforest_landscape) optionsHTML += `<option value="thunderforest_landscape">${MAP_PROVIDERS.thunderforest_landscape.name}</option>`;
        optionsHTML += '</optgroup>';
        
        div.innerHTML = `
            <div style="
                background: rgba(22, 27, 34, 0.95);
                border: 1px solid var(--border-color);
                border-radius: 8px;
                padding: 12px;
                backdrop-filter: blur(10px);
                min-width: 200px;
                max-height: 400px;
                overflow-y: auto;
            ">
                <div style="
                    font-weight: bold; 
                    margin-bottom: 10px; 
                    color: var(--primary-text); 
                    font-size: 14px;
                    text-align: center;
                ">
                    🗺️ Provedor de Mapa
                </div>
                
                <select id="mapTypeSelector" style="
                    width: 100%;
                    padding: 8px;
                    border: 1px solid var(--border-color);
                    border-radius: 4px;
                    background: var(--container-bg);
                    color: var(--font-select-color);
                    font-size: 14px;
                    margin-bottom: 10px;
                ">
                    ${optionsHTML}
                </select>
                
                <div style="
                    font-size: 12px; 
                    color: var(--success-text);
                    margin-bottom: 8px;
                    text-align: center;
                ">
                    Atual: <span id="currentMapType">${MAP_PROVIDERS[currentMapProvider]?.name || 'OpenStreetMap'}</span>
                </div>
                
                <div style="
                    border-top: 1px solid var(--border-color);
                    padding-top: 8px;
                    margin-top: 8px;
                    font-size: 11px;
                    color: var(--secondary-text);
                    text-align: center;
                ">
                    ${Object.keys(MAP_PROVIDERS).length} provedores disponíveis
                </div>
            </div>
        `;
        
        // Previne propagação de eventos do mapa
        L.DomEvent.disableClickPropagation(div);
        
        return div;
    };
    
    layerControl.addTo(activityMap);
    
    // Adiciona event listeners após um delay
    setTimeout(() => {
        const selector = document.getElementById('mapTypeSelector');
        const currentTypeLabel = document.getElementById('currentMapType');
        
        if (selector) {
            selector.value = currentMapProvider;
            
            selector.addEventListener('change', (e) => {
                const newProvider = e.target.value;
                addTileLayer(newProvider);
                
                if (currentTypeLabel) {
                    currentTypeLabel.textContent = MAP_PROVIDERS[newProvider]?.name || newProvider;
                }
                
                // Salva preferência
                localStorage.setItem('preferredMapType', newProvider);
            });
        }
    }, 100);
}

/**
 * NOVA FUNÇÃO: Inicializa map.js com chaves do ConfigService
 */
function initializeMapWithConfigService() {
    console.log('🗺️ Inicializando map.js com ConfigService...');
    
    // Se ConfigService ainda não está pronto, aguarda
    if (!window.configService || !window.configService.initialized) {
        console.log('⏳ Aguardando ConfigService...');
        setTimeout(initializeMapWithConfigService, 500);
        return;
    }
    
    // Atualiza provedores com chaves do backend
    
    console.log('✅ map.js inicializado com ConfigService');
    window.MAP_PROVIDERS = MAP_PROVIDERS;


    updateMapProvidersWithKeys();
}

/**
 * Modal para configuração do token do Mapbox
 */
function showMapboxTokenModal() {
    // Remove modal existente se houver
    const existingModal = document.getElementById('mapboxModal');
    if (existingModal) {
        existingModal.remove();
    }
    
    const modal = document.createElement('div');
    modal.id = 'mapboxModal';
    modal.innerHTML = `
        <div style="
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.7);
            display: flex;
            align-items: center;
            justify-content: center;
            z-index: 10000;
        ">
            <div style="
                background: var(--container-bg);
                border: 1px solid var(--border-color);
                border-radius: 8px;
                padding: 24px;
                max-width: 500px;
                width: 90%;
            ">
                <h3 style="
                    margin: 0 0 16px 0;
                    color: var(--primary-text);
                    text-align: center;
                ">
                    🗺️ Configuração do Mapbox
                </h3>
                
                <p style="
                    color: var(--secondary-text);
                    margin-bottom: 16px;
                    line-height: 1.5;
                ">
                    Para usar os mapas do Mapbox, você precisa de um token de acesso. 
                    <a href="https://account.mapbox.com/access-tokens/" target="_blank" 
                       style="color: var(--accent-color);">
                        Obtenha seu token gratuito aqui
                    </a>
                </p>
                
                <div style="margin-bottom: 16px;">
                    <label style="
                        display: block;
                        margin-bottom: 8px;
                        color: var(--primary-text);
                        font-weight: 500;
                    ">
                        Token de Acesso do Mapbox:
                    </label>
                    <input type="text" id="mapboxTokenInput" placeholder="pk.eyJ1..." style="
                        width: 100%;
                        padding: 12px;
                        border: 1px solid var(--border-color);
                        border-radius: 4px;
                        background: var(--bg-color);
                        color: var(--primary-text);
                        font-family: monospace;
                    ">
                </div>
                
                <div style="
                    display: flex;
                    gap: 12px;
                    justify-content: flex-end;
                ">
                    <button id="cancelMapboxBtn" style="
                        padding: 10px 20px;
                        border: 1px solid var(--border-color);
                        border-radius: 4px;
                        background: transparent;
                        color: var(--secondary-text);
                        cursor: pointer;
                    ">
                        Cancelar
                    </button>
                    <button id="saveMapboxBtn" style="
                        padding: 10px 20px;
                        border: none;
                        border-radius: 4px;
                        background: var(--accent-color);
                        color: white;
                        cursor: pointer;
                        font-weight: 500;
                    ">
                        Salvar Token
                    </button>
                </div>
                
                <div style="
                    margin-top: 16px;
                    padding-top: 16px;
                    border-top: 1px solid var(--border-color);
                    font-size: 12px;
                    color: var(--secondary-text);
                    text-align: center;
                ">
                    💡 O Mapbox oferece 50.000 visualizações gratuitas por mês
                </div>
            </div>
        </div>
    `;
    
    document.body.appendChild(modal);
    
    // Preenche com token existente se houver
    const tokenInput = document.getElementById('mapboxTokenInput');
    if (mapboxAccessToken) {
        tokenInput.value = mapboxAccessToken;
    }
    
    // Event listeners
    document.getElementById('cancelMapboxBtn').addEventListener('click', () => {
        modal.remove();
    });
    
    document.getElementById('saveMapboxBtn').addEventListener('click', () => {
        const token = tokenInput.value.trim();
        if (token) {
            if (token.startsWith('pk.')) {
                setMapboxToken(token);
                modal.remove();
                // Mostra mensagem de sucesso
                showMessage(result, '✅ Token do Mapbox configurado com sucesso!', 'success');
            } else {
                alert('⚠️ Token inválido. O token do Mapbox deve começar com "pk."');
            }
        } else {
            alert('⚠️ Por favor, insira um token válido.');
        }
    });
    
    // Fecha modal clicando fora
    modal.addEventListener('click', (e) => {
        if (e.target === modal) {
            modal.remove();
        }
    });
    
    // Foca no input
    tokenInput.focus();
}


/**
 * Carrega preferência salva do usuário e token do Mapbox
 */
function loadMapPreference() {
    const saved = localStorage.getItem('preferredMapType');
    if (saved && MAP_PROVIDERS[saved]) {
        currentMapProvider = saved;
        console.log(`📦 Preferência de mapa carregada: ${MAP_PROVIDERS[saved].name}`);
    }
    
    // Carrega token do Mapbox
    loadMapboxToken();
}
/**
 * Versão atualizada da função displayMap para usar o novo sistema
 */
async function displayMapWithLayerControl(activity) {
    console.log("🗺️ Inicializando mapa para a atividade:", activity.name);

    try {
        // Limpa mapa anterior
        if (activityMap) {
            console.log("🧹 Removendo mapa anterior...");
            activityMap.remove();
            activityMap = null;
        }
        
        // Limpa marcadores
        if (videoStartMarker) videoStartMarker = null;
        if (activityPolyline) activityPolyline = null;
        manualSyncTime = "";

        // Verifica container
        const mapContainer = document.getElementById('mapContainer');
        if (!mapContainer) {
            throw new Error('Container do mapa não encontrado');
        }

        mapContainer.innerHTML = '';
        
        // Carrega preferência do usuário
        loadMapPreference();
        
        console.log("📍 Criando novo mapa com controle de camadas...");
        
        await new Promise(resolve => setTimeout(resolve, 100));

        // Inicializa mapa com sistema de camadas
        initializeMapWithLayerControl(activity);
        
        console.log("📊 Carregando dados GPS...");
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
                    <button onclick="displayMapWithLayerControl(selectedActivity)" style="margin-top: 10px;">Tentar Novamente</button>
                </div>
            `;
        }
    }
}

// Função para testar se o token do Mapbox está funcionando
function testMapboxToken() {
    if (!mapboxAccessToken) {
        console.log('❌ Nenhum token do Mapbox configurado');
        return false;
    }
    
    // Testa fazendo uma requisição simples
    const testUrl = `https://api.mapbox.com/styles/v1/mapbox/streets-v12?access_token=${mapboxAccessToken}`;
    
    fetch(testUrl)
        .then(response => {
            if (response.ok) {
                console.log('✅ Token do Mapbox válido');
                return true;
            } else {
                console.log('❌ Token do Mapbox inválido');
                return false;
            }
        })
        .catch(error => {
            console.error('❌ Erro ao testar token do Mapbox:', error);
            return false;
        });
}

/**
 * Função de conveniência para alterar mapa via código
 */
function changeMapType(providerKey) {
    if (MAP_PROVIDERS[providerKey] && activityMap) {
        addTileLayer(providerKey);
        
        // Atualiza seletor se existir
        const selector = document.getElementById('mapTypeSelector');
        if (selector) {
            selector.value = providerKey;
        }
        
        const currentTypeLabel = document.getElementById('currentMapType');
        if (currentTypeLabel) {
            currentTypeLabel.textContent = MAP_PROVIDERS[providerKey].name;
        }
    }
}

// Expõe funções globalmente para fácil uso
window.changeMapType = changeMapType;
window.setMapboxToken = setMapboxToken;
window.showMapboxTokenModal = showMapboxTokenModal;
window.testMapboxToken = testMapboxToken;

if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initializeMapWithConfigService);
} else {
    initializeMapWithConfigService();
}