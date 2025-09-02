// Global variables
let selectedActivity = null;
let activityMap = null;
let videoStartMarker = null;
let selectedVideoPath = "";
let tileCache = new Map(); // Cache de tiles
let mapBounds = null; // Cache dos bounds da atividade

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

// Tile layer otimizado com cache
function createOptimizedTileLayer() {
    return L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        maxZoom: 19,
        tileSize: 256,
        crossOrigin: true,
        // Cache por 1 hora
        cacheTimeout: 3600000,
        // Carrega tiles extras para suavizar navegação
        keepBuffer: 4,
        // Otimizações de performance
        updateWhenIdle: false,
        updateWhenZooming: false,
        reuseTiles: true
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

function displayMap(activity) {
    console.log("Inicializando mapa para a atividade:", activity.name);
    
    try {
        // Remove mapa anterior se existir
        if (activityMap) {
            activityMap.remove();
            activityMap = null;
        }
        
        // Remove marcador do vídeo anterior
        if (videoStartMarker) {
            videoStartMarker = null;
        }
        
        if (activity.map && activity.map.summary_polyline) {
            // Inicializa o mapa
            activityMap = L.map('mapContainer');
            
            // Adiciona camada de tiles
            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
                attribution: '© OpenStreetMap contributors'
            }).addTo(activityMap);
            
            // Decodifica e adiciona a rota
            const latlngs = L.Polyline.fromEncoded(activity.map.summary_polyline).getLatLngs();
            const polyline = L.polyline(latlngs, { color: '#f85149', weight: 3 }).addTo(activityMap);
            
            // Ajusta visualização para mostrar toda a rota
            activityMap.fitBounds(polyline.getBounds());
            
            // Adiciona marcadores de início e fim
            L.marker(latlngs[0]).addTo(activityMap).bindPopup('🏁 Início');
            L.marker(latlngs[latlngs.length - 1]).addTo(activityMap).bindPopup('🏆 Fim');
            
            console.log("Mapa inicializado com sucesso");
            
        } else if (activity.start_latlng && activity.start_latlng.length === 2) {
            // Fallback para atividades sem polyline
            activityMap = L.map('mapContainer').setView([activity.start_latlng[0], activity.start_latlng[1]], 13);
            L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', { 
                attribution: '© OpenStreetMap contributors' 
            }).addTo(activityMap);
            L.marker([activity.start_latlng[0], activity.start_latlng[1]]).addTo(activityMap)
                .bindPopup('🏁 Início da atividade');
        } else {
            console.log("Nenhum dado de mapa disponível para esta atividade.");
            document.getElementById('mapContainer').innerHTML = `<div class="error">Mapa não disponível para esta atividade.</div>`;
            return;
        }
        
    } catch (error) {
        console.error("ERRO AO EXIBIR O MAPA:", error);
        document.getElementById('mapContainer').innerHTML = `<div class="error">Erro ao carregar o mapa: ${error.message}</div>`;
    }
}

async function selectVideo() {
    try {
        const path = await window.go.main.App.SelectVideoFile();
        if (!path) {
            return;
        }

        selectedVideoPath = path;

        const fileName = path.split(/[\\/]/).pop();
        videoInfo.innerHTML = `
            <h4>Vídeo Selecionado</h4>
            <p><strong>Arquivo:</strong> ${fileName}</p>
            <p><strong>Caminho:</strong> ${path}</p>
        `;
        processBtn.disabled = false;

        // Busca o ponto GPS correspondente ao início do vídeo
        console.log("Buscando ponto GPS para sincronização...");
        const point = await window.go.main.App.GetGPSPointForVideoTime(selectedActivity.id, path);
        
        if (point && point.lat && point.lng) {
            console.log(`Ponto GPS encontrado: ${point.lat}, ${point.lng}`);
            
            // Remove marcador anterior se existir
            if (videoStartMarker) {
                videoStartMarker.remove();
                videoStartMarker = null;
            }

            // Verifica se o mapa existe
            if (!activityMap) {
                console.error("Mapa não está inicializado");
                return;
            }

            // Cria ícone azul para o início do vídeo
            const blueIcon = new L.Icon({
                iconUrl: 'https://raw.githubusercontent.com/pointhi/leaflet-color-markers/master/img/marker-icon-2x-blue.png',
                shadowUrl: 'https://cdnjs.cloudflare.com/ajax/libs/leaflet/0.7.7/images/marker-shadow.png',
                iconSize: [25, 41], 
                iconAnchor: [12, 41], 
                popupAnchor: [1, -34], 
                shadowSize: [41, 41]
            });

            // Adiciona marcador do início do vídeo
            videoStartMarker = L.marker([point.lat, point.lng], { icon: blueIcon })
                .addTo(activityMap)
                .bindPopup('▶️ Início do Vídeo')
                .openPopup();

            console.log("Marcador adicionado, ajustando visualização...");

            // Aguarda um momento e então ajusta a visualização
            setTimeout(() => {
                try {
                    // Force o mapa a recalcular seu tamanho
                    activityMap.invalidateSize();
                    
                    // Centraliza no ponto do vídeo com zoom alto
                    activityMap.setView([point.lat, point.lng], 16);
                    
                    console.log("Mapa centralizado no ponto do vídeo");
                } catch (error) {
                    console.error("Erro ao centralizar mapa:", error);
                }
            }, 200);

        } else {
            console.warn("Nenhum ponto GPS encontrado para o horário do vídeo");
            showMessage(result, 'Não foi possível encontrar dados GPS para o horário do vídeo', 'error');
        }

    } catch (error) {
        console.error("Erro ao selecionar o vídeo:", error);
        showMessage(result, `Erro ao selecionar vídeo: ${error}`, 'error');
    }
}

// Função para aguardar o mapa estar pronto
function waitForMapReady() {
    return new Promise((resolve) => {
        if (!activityMap) {
            resolve(false);
            return;
        }
        
        // Aguarda o mapa estar completamente carregado
        activityMap.whenReady(() => {
            setTimeout(resolve, 100); // Buffer adicional
        });
    });
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
        const outputPath = await window.go.main.App.ProcessVideoOverlay(selectedActivity.id, selectedVideoPath);
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