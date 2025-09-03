// Global variables
let selectedActivity = null;
let activityMap = null;
let videoStartMarker = null;
let selectedVideoPath = "";
let tileCache = new Map(); // Cache de tiles
let mapBounds = null; // Cache dos bounds da atividade
let manualSyncTime = ""; // Armazena o tempo de início selecionado manualmente
let activityPolyline = null; // Referência à linha do trajeto no mapa

// DOM elements
const authBtn = document.getElementById('authBtn');
const authStatus = document.getElementById('authStatus');
const activitiesSection = document.getElementById('activitiesSection');
const activitiesGrid = document.getElementById('activitiesGrid');
const activityDetail = document.getElementById('activityDetail');
const activityInfo = document.getElementById('activityInfo');
const mapContainer = document.getElementById('mapContainer');
const videoSection = document.getElementById('videoSection');
const selectVideoBtn = document.getElementById('selectVideoBtn');
const videoInfo = document.getElementById('videoInfo');
const processBtn = document.getElementById('processBtn');
const progress = document.getElementById('progress');
const progressBar = document.getElementById('progressBar');
const progressText = document.getElementById('progressText');
const result = document.getElementById('result');

// Event listeners
document.addEventListener('DOMContentLoaded', initApp);
authBtn.addEventListener('click', authenticateStrava);
selectVideoBtn.addEventListener('click', selectVideo);
processBtn.addEventListener('click', processVideo);

function initApp() {
    console.log('Strava Add Overlay iniciado');
    preloadMapResources();
}

// Pré-carrega recursos do mapa
function preloadMapResources() {
    // Pre-cache comum tiles do OpenStreetMap
    const commonTiles = [
        'https://a.tile.openstreetmap.org/10/512/512.png',
        'https://b.tile.openstreetmap.org/10/513/512.png',
        'https://c.tile.openstreetmap.org/10/512/513.png'
    ];
    
    commonTiles.forEach(url => {
        const img = new Image();
        img.crossOrigin = 'anonymous';
        img.src = url;
    });
}

async function authenticateStrava() {
    try {
        authBtn.disabled = true;
        authBtn.textContent = 'Conectando...';
        authStatus.innerHTML = '';
        
        await window.go.main.App.AuthenticateStrava();
        
        showMessage(authStatus, 'Conectado com sucesso ao Strava!', 'success');
        authBtn.style.display = 'none';
        
        await loadActivities();
        
    } catch (error) {
        showMessage(authStatus, `Erro na autenticação: ${error}`, 'error');
        authBtn.disabled = false;
        authBtn.textContent = 'Autenticar com Strava';
    }
}

async function loadActivities() {
    try {
        const activities = await window.go.main.App.GetActivities();
        displayActivities(activities);
        activitiesSection.classList.remove('hidden');
        
    } catch (error) {
        showMessage(authStatus, `Erro ao carregar atividades: ${error}`, 'error');
    }
}

function displayActivities(activities) {
    activitiesGrid.innerHTML = '';
    
    if (!activities || activities.length === 0) {
        activitiesGrid.innerHTML = '<p>Nenhuma atividade com GPS encontrada.</p>';
        return;
    }
    
    activities.forEach(activity => {
        const card = createActivityCard(activity);
        activitiesGrid.appendChild(card);
    });
}

function createActivityCard(activity) {
    const card = document.createElement('div');
    card.className = 'activity-card';
    card.onclick = () => selectActivity(activity, card);
    
    const date = formatDate(new Date(activity.start_date));
    const distance = (activity.distance / 1000).toFixed(2);
    const duration = formatDuration(activity.moving_time);
    const maxSpeed = activity.max_speed ? (activity.max_speed * 3.6).toFixed(1) : 'N/A';
    
    card.innerHTML = `
        <h3>${activity.name}</h3>
        <p><strong>Tipo:</strong> ${translateActivityType(activity.type)}</p>
        <p><strong>Data:</strong> ${date}</p>
        <p><strong>Distância:</strong> ${distance} km</p>
        <p><strong>Duração:</strong> ${duration}</p>
        <p><strong>Vel. Máx:</strong> ${maxSpeed} km/h</p>
    `;
    
    return card;
}

async function selectActivity(activity, cardElement) {
    try {
        document.querySelectorAll('.activity-card.selected').forEach(el => {
            el.classList.remove('selected');
        });
        
        cardElement.classList.add('selected');
        selectedActivity = activity;
        
        const detail = await window.go.main.App.GetActivityDetail(activity.id);
        displayActivityDetail(detail);
        await displayMap(activity);
        
        activityDetail.classList.remove('hidden');
        videoSection.classList.remove('hidden');
        
    } catch (error) {
        showMessage(result, `Erro ao carregar detalhes: ${error}`, 'error');
    }
}

function displayActivityDetail(detail) {
    const startDate = new Date(detail.start_date);
    const date = formatDate(startDate);
    const time = formatTime(startDate);
    const distance = (detail.distance / 1000).toFixed(2);
    const duration = formatDuration(detail.moving_time);
    const elevation = detail.total_elevation_gain ? detail.total_elevation_gain.toFixed(0) : 'N/A';
    const maxSpeed = detail.max_speed ? (detail.max_speed * 3.6).toFixed(1) : 'N/A';
    const calories = detail.calories ? detail.calories.toFixed(0) : 'N/A';
    
    activityInfo.innerHTML = `
        <div class="info-grid">
            <div class="info-item">
                <h4>Informações Básicas</h4>
                <p><strong>Nome:</strong> ${detail.name}</p>
                <p><strong>Tipo:</strong> ${translateActivityType(detail.type)}</p>
                <p><strong>Data:</strong> ${date}</p>
                <p><strong>Horário:</strong> ${time}</p>
            </div>
            <div class="info-item">
                <h4>Desempenho</h4>
                <p><strong>Distância:</strong> ${distance} km</p>
                <p><strong>Duração:</strong> ${duration}</p>
                <p><strong>Vel. Máxima:</strong> ${maxSpeed} km/h</p>
                <p><strong>Calorias:</strong> ${calories}</p>
            </div>
            <div class="info-item">
                <h4>Elevação</h4>
                <p><strong>Ganho Total:</strong> ${elevation} m</p>
            </div>
        </div>
    `;
}

// ========================================
// 2. SUBSTITUIR NO frontend/main.js
// ========================================

async function displayMap(activity) {
    console.log("Inicializando mapa para a atividade:", activity.name);
    
    try {
        if (activityMap) {
            activityMap.remove();
            activityMap = null;
        }
        if (videoStartMarker) videoStartMarker = null;
        if (activityPolyline) activityPolyline = null;
        manualSyncTime = ""; // Reseta a sincronização manual
        
        if (activity.map && activity.map.summary_polyline) {
            activityMap = L.map('mapContainer');
            
            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '© OpenStreetMap contributors'
            }).addTo(activityMap);
            
            const latlngs = L.Polyline.fromEncoded(activity.map.summary_polyline).getLatLngs();
            activityPolyline = L.polyline(latlngs, { color: '#f85149', weight: 3 }).addTo(activityMap);
            
            activityPolyline.on('click', handleMapClick);

            // Fit bounds primeiro
            activityMap.fitBounds(activityPolyline.getBounds());
            
            // Adiciona marcadores de início e fim
            L.marker(latlngs[0]).addTo(activityMap).bindPopup('🏁 Início');
            L.marker(latlngs[latlngs.length - 1]).addTo(activityMap).bindPopup('🏆 Fim');
            
            console.log("Mapa inicializado com sucesso");

            // AGORA adiciona os pontos GPS depois que o mapa está pronto
            setTimeout(async () => {
                await addGPSMarkersToMap(activity.id);
            }, 500); // Pequeno delay para garantir que o mapa está renderizado
            
        } else if (activity.start_latlng && activity.start_latlng.length === 2) {
            activityMap = L.map('mapContainer').setView([activity.start_latlng[0], activity.start_latlng[1]], 13);
            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', { 
                attribution: '© OpenStreetMap contributors' 
            }).addTo(activityMap);
            L.marker([activity.start_latlng[0], activity.start_latlng[1]]).addTo(activityMap)
                .bindPopup('🏁 Início da atividade');
        } else {
            document.getElementById('mapContainer').innerHTML = `<div class="error">Mapa não disponível para esta atividade.</div>`;
            return;
        }
        
    } catch (error) {
        console.error("ERRO AO EXIBIR O MAPA:", error);
        document.getElementById('mapContainer').innerHTML = `<div class="error">Erro ao carregar o mapa: ${error.message}</div>`;
    }
}

// NOVA FUNÇÃO para adicionar marcadores GPS
async function addGPSMarkersToMap(activityId) {
    try {
        console.log("Carregando pontos GPS para a atividade:", activityId);
        showMessage(result, 'Carregando pontos GPS...', 'info');

        const gpsPoints = await window.go.main.App.GetAllGPSPoints(activityId);
        
        if (!gpsPoints || gpsPoints.length === 0) {
            console.log("Nenhum ponto GPS encontrado");
            showMessage(result, '', ''); // Limpa mensagem
            return;
        }

        console.log(`Adicionando ${gpsPoints.length} marcadores GPS ao mapa`);

        // Cria um grupo de camadas para os marcadores GPS
        const gpsMarkersGroup = L.layerGroup().addTo(activityMap);

        // Adiciona cada ponto como um pequeno marcador circular
        gpsPoints.forEach((point, index) => {
            const speed = point.velocity * 3.6; // m/s para km/h
            
            // Cor baseada na velocidade
            let color = '#58a6ff'; // Azul padrão
            if (speed > 30) color = '#f85149'; // Vermelho para alta velocidade
            else if (speed > 15) color = '#ffa657'; // Laranja para média velocidade
            else if (speed > 5) color = '#56d364'; // Verde para baixa velocidade

            const marker = L.circleMarker([point.lat, point.lng], {
                radius: 4,
                fillColor: color,
                fillOpacity: 0.7,
                color: '#ffffff',
                weight: 1,
                opacity: 0.8
            });

            // Popup com informações do ponto
            const time = new Date(point.time).toLocaleTimeString('pt-BR');
            marker.bindPopup(`
                <div style="font-size: 12px;">
                    <strong>⏰ ${time}</strong><br>
                    📍 ${point.lat.toFixed(6)}, ${point.lng.toFixed(6)}<br>
                    🏃 ${speed.toFixed(1)} km/h<br>
                    ⛰️ ${point.altitude.toFixed(0)}m<br>
                    🧭 ${point.bearing.toFixed(0)}°
                </div>
            `);

            // Adiciona ao grupo
            gpsMarkersGroup.addLayer(marker);
        });

        // Controle de camadas para mostrar/ocultar marcadores
        const overlayMaps = {
            "📍 Pontos GPS": gpsMarkersGroup
        };
        
        L.control.layers(null, overlayMaps, { 
            position: 'topright',
            collapsed: false 
        }).addTo(activityMap);

        console.log(`${gpsPoints.length} marcadores GPS adicionados com sucesso`);
        showMessage(result, `${gpsPoints.length} pontos GPS carregados no mapa`, 'success');

        // Limpa a mensagem após 3 segundos
        setTimeout(() => {
            showMessage(result, '', '');
        }, 3000);

    } catch (error) {
        console.error("Erro ao carregar pontos GPS:", error);
        showMessage(result, `Erro ao carregar pontos GPS: ${error}`, 'error');
    }
}

// FUNÇÃO AUXILIAR para resetar marcadores quando necessário
function clearGPSMarkers() {
    if (activityMap) {
        activityMap.eachLayer((layer) => {
            if (layer instanceof L.CircleMarker && !(layer instanceof L.Marker)) {
                activityMap.removeLayer(layer);
            }
        });
    }
}

// A função addAllGpsPointsToMap foi removida e sua lógica integrada em displayMap

// Função para lidar com o clique no mapa para sincronização manual
async function handleMapClick(e) {
    if (!selectedActivity) return;

    try {
        console.log(`Clique no mapa detectado em: ${e.latlng.lat}, ${e.latlng.lng}`);
        showMessage(result, 'Ajustando ponto de sincronização...', 'info');

        const point = await window.go.main.App.GetGPSPointForMapClick(selectedActivity.id, e.latlng.lat, e.latlng.lng);
        
        if (point && point.lat && point.lng) {
            console.log(`Ponto de sincronização manual definido para: ${point.time}`);
            manualSyncTime = point.time; // Armazena o tempo manual
            updateVideoStartMarker(point.lat, point.lng, '▶️ Início Manual do Vídeo');
            showMessage(result, `Ponto de sincronização manual definido.`, 'success');
        } else {
            showMessage(result, `Não foi possível encontrar um ponto GPS próximo ao clique.`, 'error');
        }

    } catch (error) {
        console.error("Erro ao definir ponto de sincronização manual:", error);
        showMessage(result, `Erro ao ajustar sincronização: ${error}`, 'error');
    }
}

// Função auxiliar para criar/atualizar o marcador de início do vídeo
function updateVideoStartMarker(lat, lng, popupText) {
     if (!activityMap) {
        console.error("Mapa não está inicializado para atualizar o marcador");
        return;
    }

    if (videoStartMarker) {
        videoStartMarker.remove();
        videoStartMarker = null;
    }

    const blueIcon = new L.Icon({
        iconUrl: 'https://raw.githubusercontent.com/pointhi/leaflet-color-markers/master/img/marker-icon-2x-blue.png',
        shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/0.7.7/images/marker-shadow.png',
        iconSize: [25, 41], 
        iconAnchor: [12, 41], 
        popupAnchor: [1, -34], 
        shadowSize: [41, 41]
    });

    videoStartMarker = L.marker([lat, lng], { icon: blueIcon })
        .addTo(activityMap)
        .bindPopup(popupText)
        .openPopup();

    setTimeout(() => {
        try {
            activityMap.invalidateSize();
            // Usar setView para centralizar e aplicar zoom.
            activityMap.setView([lat, lng], 16);
            console.log("Mapa centralizado e com zoom no novo marcador de início.");
        } catch (error) {
            console.error("Erro ao centralizar mapa no marcador:", error);
        }
    }, 200);
}

async function selectVideo() {
    try {
        const path = await window.go.main.App.SelectVideoFile();
        if (!path) return;

        selectedVideoPath = path;
        manualSyncTime = ""; // Reseta ao selecionar um novo vídeo

        const fileName = path.split(/[\\/]/).pop();
        videoInfo.innerHTML = `
            <h4>Vídeo Selecionado</h4>
            <p><strong>Arquivo:</strong> ${fileName}</p>
        `;
        processBtn.disabled = false;

        console.log("Buscando ponto GPS para sincronização automática...");
        const point = await window.go.main.App.GetGPSPointForVideoTime(selectedActivity.id, path);
        
        if (point && point.lat && point.lng) {
            updateVideoStartMarker(point.lat, point.lng, '▶️ Início Automático (Clique no trajeto para ajustar)');
        } else {
            showMessage(result, 'Não foi possível encontrar dados GPS para o horário do vídeo. Clique no mapa para definir o início.', 'error');
        }
    } catch (error) {
        showMessage(result, `Erro ao selecionar vídeo: ${error}`, 'error');
    }
}

async function processVideo() {
    if (!selectedActivity || !selectedVideoPath) {
        showMessage(result, 'Selecione uma atividade e um vídeo', 'error');
        return;
    }
    try {
        processBtn.disabled = true;
        processBtn.textContent = 'Processando...';
        progress.classList.remove('hidden');
        result.innerHTML = '';
        simulateProgress();

        // Passa o tempo manual (pode ser uma string vazia) para o backend
        const outputPath = await window.go.main.App.ProcessVideoOverlay(selectedActivity.id, selectedVideoPath, manualSyncTime);
        
        updateProgress(100);
        showMessage(result, `Vídeo processado com sucesso!<br><strong>Local:</strong> ${outputPath}`, 'success');
    } catch (error) {
        showMessage(result, `Erro no processamento: ${error}`, 'error');
    } finally {
        processBtn.disabled = false;
        processBtn.textContent = 'Processar com Overlay';
        setTimeout(() => {
            progress.classList.add('hidden');
            updateProgress(0);
        }, 3000);
    }
}

function simulateProgress() {
    let currentProgress = 0;
    const interval = setInterval(() => {
        currentProgress += Math.random() * 15;
        if (currentProgress > 90) {
            currentProgress = 90;
            clearInterval(interval);
        }
        updateProgress(currentProgress);
    }, 800);
}

function updateProgress(value) {
    progressBar.style.width = `${value}%`;
    progressText.textContent = `${Math.round(value)}%`;
}

function showMessage(container, message, type) {
    container.innerHTML = `<div class="${type}">${message}</div>`;
}

function formatDate(date) {
    return date.toLocaleDateString('pt-BR');
}

function formatTime(date) {
    return date.toLocaleTimeString('pt-BR', { 
        hour: '2-digit', 
        minute: '2-digit' 
    });
}

function formatDuration(seconds) {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    
    if (hours > 0) {
        return `${hours}h ${minutes}m`;
    }
    return `${minutes}m ${secs}s`;
}

function translateActivityType(type) {
    const translations = {
        'Ride': 'Ciclismo',
        'Run': 'Corrida',
        'Hike': 'Caminhada',
        'Walk': 'Caminhada',
        'Swimming': 'Natação',
        'Workout': 'Treino',
        'WeightTraining': 'Musculação'
    };
    return translations[type] || type;
}

