// Global variables
let selectedActivity = null;
let activityMap = null;

// DOM elements
const authBtn = document.getElementById('authBtn');
const authStatus = document.getElementById('authStatus');
const activitiesSection = document.getElementById('activitiesSection');
const activitiesGrid = document.getElementById('activitiesGrid');
const activityDetail = document.getElementById('activityDetail');
const activityInfo = document.getElementById('activityInfo');
const mapContainer = document.getElementById('mapContainer');
const videoSection = document.getElementById('videoSection');
const videoInput = document.getElementById('videoInput');
const videoInfo = document.getElementById('videoInfo');
const processBtn = document.getElementById('processBtn');
const progress = document.getElementById('progress');
const progressBar = document.getElementById('progressBar');
const progressText = document.getElementById('progressText');
const result = document.getElementById('result');

// Event listeners
document.addEventListener('DOMContentLoaded', initApp);
authBtn.addEventListener('click', authenticateStrava);
videoInput.addEventListener('change', handleVideoUpload);
processBtn.addEventListener('click', processVideo);

function initApp() {
    console.log('Strava Add Overlay iniciado');
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
    
    // CORREÇÃO: Verifica se 'activities' é nulo ou se o tamanho é zero.
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
    
    const date = formatDate(activity.start_date);
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
        // Remove seleção anterior
        document.querySelectorAll('.activity-card.selected').forEach(el => {
            el.classList.remove('selected');
        });
        
        // Adiciona seleção atual
        cardElement.classList.add('selected');
        selectedActivity = activity;
        
        // Carrega detalhes
        const detail = await window.go.main.App.GetActivityDetail(activity.id);
        displayActivityDetail(detail);
        displayMap(activity);
        
        // Mostra seções
        activityDetail.classList.remove('hidden');
        videoSection.classList.remove('hidden');
        
    } catch (error) {
        showMessage(result, `Erro ao carregar detalhes: ${error}`, 'error');
    }
}

function displayActivityDetail(detail) {
    const date = formatDate(detail.start_date);
    const time = formatTime(detail.start_date);
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
    // Remove mapa anterior
    if (activityMap) {
        activityMap.remove();
    }
    
    if (activity.start_latlng && activity.start_latlng.length === 2) {
        activityMap = L.map('mapContainer').setView(
            [activity.start_latlng[0], activity.start_latlng[1]], 
            13
        );
        
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors'
        }).addTo(activityMap);
        
        // Marcador de início
        L.marker([activity.start_latlng[0], activity.start_latlng[1]])
            .addTo(activityMap)
            .bindPopup('🏁 Início da atividade');
        
        // Marcador de fim
        if (activity.end_latlng && activity.end_latlng.length === 2) {
            L.marker([activity.end_latlng[0], activity.end_latlng[1]])
                .addTo(activityMap)
                .bindPopup('🏆 Fim da atividade');
        }
    }
}

function handleVideoUpload(event) {
    const file = event.target.files[0];
    if (!file) {
        videoInfo.innerHTML = '';
        processBtn.disabled = true;
        return;
    }
    
    const size = (file.size / (1024 * 1024)).toFixed(2);
    const type = file.type;
    
    videoInfo.innerHTML = `
        <h4>Vídeo Selecionado</h4>
        <p><strong>Nome:</strong> ${file.name}</p>
        <p><strong>Tamanho:</strong> ${size} MB</p>
        <p><strong>Tipo:</strong> ${type}</p>
        <p><strong>Modificado:</strong> ${formatDate(file.lastModified)}</p>
    `;
    
    processBtn.disabled = false;
}

async function processVideo() {
    if (!selectedActivity || !videoInput.files[0]) {
        showMessage(result, 'Selecione uma atividade e um vídeo', 'error');
        return;
    }
    
    try {
        processBtn.disabled = true;
        processBtn.textContent = 'Processando...';
        progress.classList.remove('hidden');
        result.innerHTML = '';
        
        // Simula progresso
        simulateProgress();
        
        const videoPath = videoInput.files[0].path || videoInput.files[0].name;
        const outputPath = await window.go.main.App.ProcessVideoOverlay(
            selectedActivity.id,
            videoPath
        );
        
        // Finaliza progresso
        updateProgress(100);
        
        showMessage(result, `
            Vídeo processado com sucesso!<br>
            <strong>Local:</strong> ${outputPath}
        `, 'success');
        
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
    let progress = 0;
    const interval = setInterval(() => {
        progress += Math.random() * 15;
        if (progress > 90) {
            progress = 90;
            clearInterval(interval);
        }
        updateProgress(progress);
    }, 800);
}

function updateProgress(value) {
    progressBar.style.width = `${value}%`;
    progressText.textContent = `${Math.round(value)}%`;
}

function showMessage(container, message, type) {
    container.innerHTML = `<div class="${type}">${message}</div>`;
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('pt-BR');
}

function formatTime(dateString) {
    const date = new Date(dateString);
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