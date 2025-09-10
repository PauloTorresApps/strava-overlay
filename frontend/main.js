console.log('🚀 main.js carregando...');

// Global variables
let selectedActivity = null;
let activityMap = null;
let videoStartMarker = null;
let selectedVideoPath = "";
let tileCache = new Map();
let mapBounds = null;
let manualSyncTime = "";
let activityPolyline = null;
let currentMarkerDensity = 'medium';
let currentGPSMarkersGroup = null;

// --- VARIÁVEIS PARA PAGINAÇÃO ---
let allActivities = []; // Todas as atividades carregadas
let currentPage = 1;
let isLoadingMore = false;
let hasMorePages = true;
let showOnlyGPS = true; // Filtro para mostrar apenas atividades com GPS

// --- VARIÁVEIS PARA CONTROLE DE AUTENTICAÇÃO ---
let isAuthenticated = false;
let isCheckingAuth = false;

// DOM elements
let authBtn, authStatus, activitiesSection, activitiesGrid;
let activityDetail, activityInfo, mapContainer, videoSection;
let selectVideoBtn, videoInfo, processBtn, progress;
let progressBar, progressText, result;
let loadMoreBtn, prevPageBtn, nextPageBtn, currentPageSpan;
let totalActivitiesSpan, gpsActivitiesSpan, filterGPSCheckbox;

// Event listeners
document.addEventListener('DOMContentLoaded', function() {
    console.log('📄 DOM carregado, inicializando app...');
    initializeElements();
    initApp();
});

function initializeElements() {
    // Inicializa elementos DOM de forma segura
    authBtn = document.getElementById('authBtn');
    authStatus = document.getElementById('authStatus');
    activitiesSection = document.getElementById('activitiesSection');
    activitiesGrid = document.getElementById('activitiesGrid');
    activityDetail = document.getElementById('activityDetail');
    activityInfo = document.getElementById('activityInfo');
    mapContainer = document.getElementById('mapContainer');
    videoSection = document.getElementById('videoSection');
    selectVideoBtn = document.getElementById('selectVideoBtn');
    videoInfo = document.getElementById('videoInfo');
    processBtn = document.getElementById('processBtn');
    progress = document.getElementById('progress');
    progressBar = document.getElementById('progressBar');
    progressText = document.getElementById('progressText');
    result = document.getElementById('result');
    
    // Elementos de paginação
    loadMoreBtn = document.getElementById('loadMoreBtn');
    prevPageBtn = document.getElementById('prevPageBtn');
    nextPageBtn = document.getElementById('nextPageBtn');
    currentPageSpan = document.getElementById('currentPage');
    totalActivitiesSpan = document.getElementById('totalActivities');
    gpsActivitiesSpan = document.getElementById('gpsActivities');
    filterGPSCheckbox = document.getElementById('filterGPS');
    
    // Adiciona event listeners de forma segura
    if (authBtn) authBtn.addEventListener('click', authenticateStrava);
    if (selectVideoBtn) selectVideoBtn.addEventListener('click', selectVideo);
    if (processBtn) processBtn.addEventListener('click', processVideo);
    
    // Event listeners para paginação
    if (loadMoreBtn) loadMoreBtn.addEventListener('click', loadMoreActivities);
    if (prevPageBtn) prevPageBtn.addEventListener('click', () => changePage(-1));
    if (nextPageBtn) nextPageBtn.addEventListener('click', () => changePage(1));
    if (filterGPSCheckbox) filterGPSCheckbox.addEventListener('change', handleFilterChange);
    
    console.log('✅ Elementos DOM inicializados');
}

// --- FUNÇÃO PRINCIPAL ---
async function initApp() {
    console.log('🚀 Strava Add Overlay iniciado');
    preloadMapResources();
    
    // Verifica autenticação automaticamente na inicialização
    setTimeout(checkAuthenticationOnStartup, 500);
}

// --- VERIFICAÇÃO DE AUTENTICAÇÃO NA INICIALIZAÇÃO ---
async function checkAuthenticationOnStartup() {
    if (isCheckingAuth) {
        console.log('⏳ Já verificando autenticação...');
        return;
    }
    
    console.log('🔍 Iniciando verificação de autenticação...');
    isCheckingAuth = true;
    
    try {
        // Atualiza UI de forma segura
        safeUpdateStatus('checking', 'Verificando conexão...');
        safeShowMessage('🔍 Verificando credenciais salvas...', 'info');
        
        if (authBtn) {
            authBtn.disabled = true;
            authBtn.textContent = 'Verificando...';
        }
        
        // Verifica se backend está disponível
        if (!window.go?.main?.App?.CheckAuthenticationStatus) {
            throw new Error('Backend não disponível');
        }
        
        console.log('📡 Chamando backend...');
        const response = await window.go.main.App.CheckAuthenticationStatus();
        console.log('📡 Resposta recebida:', response);
        
        if (response?.is_authenticated) {
            handleAuthSuccess(response);
        } else {
            handleAuthFailure(response?.error);
        }
        
    } catch (error) {
        console.error('❌ Erro na verificação:', error);
        handleAuthError(error);
    } finally {
        isCheckingAuth = false;
    }
}

function handleAuthSuccess(response) {
    console.log('✅ Autenticação bem-sucedida');
    isAuthenticated = true;
    
    safeUpdateStatus('connected', 'Conectado ao Strava');
    safeShowMessage(`✅ ${response.message}`, 'success');
    
    if (authBtn) {
        authBtn.style.display = 'none';
    }
    
    // Carrega primeira página de atividades
    loadActivitiesPage(1);
}

function handleAuthFailure(error) {
    console.log('❌ Autenticação necessária:', error);
    isAuthenticated = false;
    
    safeUpdateStatus('error', 'Autenticação necessária');
    safeShowMessage('Clique no botão abaixo para conectar ao Strava', 'info');
    
    if (authBtn) {
        authBtn.disabled = false;
        authBtn.textContent = 'Autenticar com Strava';
        authBtn.style.display = 'block';
    }
}

function handleAuthError(error) {
    console.error('❌ Erro na verificação:', error);
    isAuthenticated = false;
    
    safeUpdateStatus('error', 'Erro na verificação');
    safeShowMessage('Erro na verificação. Clique para autenticar manualmente.', 'error');
    
    if (authBtn) {
        authBtn.disabled = false;
        authBtn.textContent = 'Autenticar com Strava';
        authBtn.style.display = 'block';
    }
}

// --- NOVA FUNÇÃO: Carrega uma página específica de atividades ---
async function loadActivitiesPage(page) {
    if (isLoadingMore) {
        console.log('⏳ Já carregando atividades...');
        return;
    }
    
    console.log(`📋 Carregando página ${page} de atividades...`);
    isLoadingMore = true;
    
    try {
        // Atualiza UI
        safeUpdateStatus('connected', `Carregando página ${page}...`);
        updateLoadMoreButton(true);
        
        // Chama a API com paginação
        const response = await window.go.main.App.GetActivitiesPage(page);
        
        if (!response) {
            throw new Error('Resposta vazia do servidor');
        }
        
        console.log(`📋 Página ${page}: ${response.activities?.length || 0} atividades recebidas`);
        
        // Atualiza variáveis globais
        currentPage = page;
        hasMorePages = response.has_more;
        
        // Se for a primeira página, limpa a lista
        if (page === 1) {
            allActivities = [];
        }
        
        // Adiciona novas atividades à lista
        if (response.activities && response.activities.length > 0) {
            allActivities = allActivities.concat(response.activities);
        }
        
        // Atualiza a exibição
        displayActivities(getFilteredActivities());
        updatePaginationControls();
        updateStatistics();
        
        // Mostra a seção de atividades
        if (activitiesSection) {
            activitiesSection.classList.remove('hidden');
        }
        
        // Atualiza status
        const totalGPS = allActivities.filter(a => a.has_gps).length;
        safeUpdateStatus('connected', `${allActivities.length} atividades carregadas`);
        
        // Limpa mensagem após um tempo
        setTimeout(() => safeShowMessage('', ''), 3000);
        
    } catch (error) {
        console.error('❌ Erro ao carregar atividades:', error);
        safeUpdateStatus('error', 'Erro ao carregar atividades');
        safeShowMessage(`Erro: ${error}`, 'error');
    } finally {
        isLoadingMore = false;
        updateLoadMoreButton(false);
    }
}

// --- NOVA FUNÇÃO: Carrega mais atividades (próxima página) ---
async function loadMoreActivities() {
    if (!hasMorePages || isLoadingMore) {
        return;
    }
    
    await loadActivitiesPage(currentPage + 1);
}

// --- NOVA FUNÇÃO: Muda de página ---
async function changePage(direction) {
    const newPage = currentPage + direction;
    if (newPage < 1) return;
    
    await loadActivitiesPage(newPage);
}

// --- NOVA FUNÇÃO: Filtra atividades baseado nas configurações ---
function getFilteredActivities() {
    if (!showOnlyGPS) {
        return allActivities;
    }
    
    return allActivities.filter(activity => activity.has_gps);
}

// --- NOVA FUNÇÃO: Lida com mudança no filtro ---
function handleFilterChange(event) {
    showOnlyGPS = event.target.checked;
    displayActivities(getFilteredActivities());
    updateStatistics();
}

// --- NOVA FUNÇÃO: Atualiza estatísticas ---
function updateStatistics() {
    const totalCount = allActivities.length;
    const gpsCount = allActivities.filter(a => a.has_gps).length;
    
    if (totalActivitiesSpan) {
        totalActivitiesSpan.textContent = `${totalCount} atividades carregadas`;
    }
    
    if (gpsActivitiesSpan) {
        gpsActivitiesSpan.textContent = `${gpsCount} com GPS`;
    }
}

// --- NOVA FUNÇÃO: Atualiza controles de paginação ---
function updatePaginationControls() {
    // Mostra ou esconde o botão "Carregar Mais"
    if (loadMoreBtn) {
        loadMoreBtn.style.display = hasMorePages ? 'block' : 'none';
    }
    
    // Atualiza controles de página (se você quiser usar navegação por páginas)
    const paginationControls = document.getElementById('paginationControls');
    if (paginationControls) {
        // Por enquanto, vamos manter oculto e usar apenas "Carregar Mais"
        paginationControls.style.display = 'none';
    }
    
    if (currentPageSpan) {
        currentPageSpan.textContent = currentPage;
    }
    
    if (prevPageBtn) {
        prevPageBtn.disabled = currentPage <= 1;
    }
    
    if (nextPageBtn) {
        nextPageBtn.disabled = !hasMorePages;
    }
}

// --- NOVA FUNÇÃO: Atualiza botão "Carregar Mais" ---
function updateLoadMoreButton(isLoading) {
    if (!loadMoreBtn) return;
    
    const loadMoreText = document.getElementById('loadMoreText');
    
    if (isLoading) {
        loadMoreBtn.disabled = true;
        if (loadMoreText) {
            loadMoreText.innerHTML = '<div class="loading-more"><div class="spinner"></div>Carregando...</div>';
        }
    } else {
        loadMoreBtn.disabled = !hasMorePages;
        if (loadMoreText) {
            if (hasMorePages) {
                loadMoreText.textContent = 'Carregar Mais Atividades';
            } else {
                loadMoreText.textContent = 'Todas as atividades foram carregadas';
            }
        }
    }
}

// --- FUNÇÃO MODIFICADA: Exibe atividades com indicador de GPS ---
function displayActivities(activities) {
    if (!activitiesGrid) return;
    
    activitiesGrid.innerHTML = '';
    
    if (!activities || activities.length === 0) {
        activitiesGrid.innerHTML = '<p>Nenhuma atividade encontrada com os filtros aplicados.</p>';
        return;
    }
    
    activities.forEach(activity => {
        const card = createActivityCard(activity);
        activitiesGrid.appendChild(card);
    });
}

// --- FUNÇÃO MODIFICADA: Cria card com indicador de GPS ---
function createActivityCard(activity) {
    const card = document.createElement('div');
    card.className = 'activity-card';
    
    // Adiciona classe especial para atividades sem GPS
    if (!activity.has_gps) {
        card.className += ' no-gps';
    }
    
    // Só permite seleção se tiver GPS
    if (activity.has_gps) {
        card.onclick = () => selectActivity(activity, card);
    } else {
        card.style.cursor = 'not-allowed';
        card.title = 'Esta atividade não possui dados GPS';
    }
    
    const date = formatDate(new Date(activity.start_date));
    const distance = (activity.distance / 1000).toFixed(2);
    const duration = formatDuration(activity.moving_time);
    const maxSpeed = activity.max_speed ? (activity.max_speed * 3.6).toFixed(1) : 'N/A';
    
    // Badge de GPS
    const gpsBadge = activity.has_gps 
        ? '<span class="gps-badge">GPS</span>' 
        : '<span class="gps-badge no-gps-badge">Sem GPS</span>';
    
    card.innerHTML = `
        <h3>${activity.name} ${gpsBadge}</h3>
        <p><strong>Tipo:</strong> ${translateActivityType(activity.type)}</p>
        <p><strong>Data:</strong> ${date}</p>
        <p><strong>Distância:</strong> ${distance} km</p>
        <p><strong>Duração:</strong> ${duration}</p>
        ${activity.has_gps ? `<p><strong>Vel. Máx:</strong> ${maxSpeed} km/h</p>` : ''}
    `;
    
    return card;
}

// FUNÇÕES AUXILIARES SEGURAS
function safeUpdateStatus(status, message) {
    console.log(`🔄 Status: ${status} - ${message}`);
    
    try {
        const indicator = document.getElementById('statusIndicator');
        const text = document.getElementById('connectionText');
        const authLoading = document.getElementById('authLoading');
        const autoConnectInfo = document.getElementById('autoConnectInfo');
        
        if (indicator) {
            indicator.className = `status-indicator ${status}`;
        }
        
        if (text) {
            text.textContent = message;
        }
        
        if (authLoading && (status === 'connected' || status === 'error')) {
            authLoading.style.display = 'none';
        }
        
        if (autoConnectInfo && status === 'connected') {
            autoConnectInfo.style.display = 'none';
        }
        
    } catch (error) {
        console.error('❌ Erro ao atualizar status:', error);
    }
}

function safeShowMessage(message, type) {
    try {
        if (authStatus) {
            authStatus.innerHTML = message ? `<div class="${type}">${message}</div>` : '';
        }
    } catch (error) {
        console.error('❌ Erro ao mostrar mensagem:', error);
    }
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

// --- FUNÇÃO DE AUTENTICAÇÃO MANUAL ---
async function authenticateStrava() {
    if (isCheckingAuth) {
        console.log('⏳ Verificação em andamento...');
        return;
    }
    
    try {
        if (authBtn) {
            authBtn.disabled = true;
            authBtn.textContent = 'Conectando...';
        }
        
        safeUpdateStatus('checking', 'Autenticando...');
        
        console.log('🔐 Iniciando autenticação manual...');
        await window.go.main.App.AuthenticateStrava();
        
        isAuthenticated = true;
        safeUpdateStatus('connected', 'Conectado');
        safeShowMessage('Conectado com sucesso!', 'success');
        
        if (authBtn) {
            authBtn.style.display = 'none';
        }
        
        console.log('📋 Carregando atividades após autenticação manual...');
        loadActivitiesPage(1);
        
    } catch (error) {
        console.error('❌ Erro na autenticação:', error);
        isAuthenticated = false;
        safeUpdateStatus('error', 'Falha na autenticação');
        safeShowMessage(`Erro: ${error}`, 'error');
        
        if (authBtn) {
            authBtn.disabled = false;
            authBtn.textContent = 'Autenticar com Strava';
        }
    }
}

// Função para selecionar uma atividade
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
        
        if (activityDetail) activityDetail.classList.remove('hidden');
        if (videoSection) videoSection.classList.remove('hidden');
        
    } catch (error) {
        showMessage(result, `Erro ao carregar detalhes: ${error}`, 'error');
    }
}

// Exibe detalhes da atividade
function displayActivityDetail(detail) {
    if (!activityInfo) return;
    
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
// FUNÇÕES DE MAPA
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

        // Inicializa mapa básico primeiro
        activityMap = L.map('mapContainer');
        
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors'
        }).addTo(activityMap);

        console.log("Mapa inicializado, carregando dados GPS...");

        // Carrega dados GPS interpolados primeiro
        await loadInterpolatedTrajectory(activity);
        
    } catch (error) {
        console.error("ERRO AO EXIBIR O MAPA:", error);
        if (mapContainer) {
            mapContainer.innerHTML = `<div class="error">Erro ao carregar o mapa: ${error.message}</div>`;
        }
    }
}

async function loadInterpolatedTrajectory(activity) {
    try {
        showMessage(result, 'Carregando trajeto detalhado...', 'info');

        // 1. CARREGA TRAJETO COMPLETO
        console.log("Carregando trajeto completo...");
        const fullTrajectory = await window.go.main.App.GetFullGPSTrajectory(activity.id);
        
        if (!fullTrajectory || fullTrajectory.length === 0) {
            console.log("Sem dados de trajeto completo, usando trajeto simplificado");
            loadFallbackTrajectory(activity);
            return;
        }

        console.log(`✅ Trajeto completo: ${fullTrajectory.length} pontos interpolados`);

        // 2. CRIA TRAJETO PRINCIPAL
        await createSpeedGradientTrajectory(fullTrajectory);

        // 3. CARREGA MARCADORES COM DENSIDADE PADRÃO
        const markerCount = await loadGPSMarkersWithDensity(activity.id, 'medium');

        // 4. MARCADORES DE INÍCIO E FIM
        const startPoint = fullTrajectory[0];
        const endPoint = fullTrajectory[fullTrajectory.length - 1];
        
        L.marker([startPoint.lat, startPoint.lng], {
            icon: createCustomIcon('🏁', '#28a745')
        }).addTo(activityMap).bindPopup(`
            <strong>🏁 INÍCIO</strong><br>
            ⏰ ${new Date(startPoint.time).toLocaleTimeString('pt-BR')}<br>
            🏃 ${(startPoint.velocity * 3.6).toFixed(1)} km/h
        `);
        
        L.marker([endPoint.lat, endPoint.lng], {
            icon: createCustomIcon('🏆', '#dc3545')  
        }).addTo(activityMap).bindPopup(`
            <strong>🏆 FIM</strong><br>
            ⏰ ${new Date(endPoint.time).toLocaleTimeString('pt-BR')}<br>
            🏃 ${(endPoint.velocity * 3.6).toFixed(1)} km/h
        `);

        // 5. FIT BOUNDS
        const bounds = L.latLngBounds(fullTrajectory.map(p => [p.lat, p.lng]));
        activityMap.fitBounds(bounds, { padding: [20, 20] });

        // 6. CONTROLES APRIMORADOS
        addAdvancedTrajectoryControls(activity.id);

        showMessage(result, `✅ Trajeto: ${fullTrajectory.length} pontos | Marcadores: ${markerCount}`, 'success');
        setTimeout(() => showMessage(result, '', ''), 4000);

    } catch (error) {
        console.error("Erro ao carregar trajeto:", error);
        showMessage(result, `Erro: ${error}`, 'error');
        loadFallbackTrajectory(activity);
    }
}

// Função para criar trajeto com gradiente de velocidade
async function createSpeedGradientTrajectory(fullTrajectoryPoints) {
    console.log(`🎨 Criando trajeto colorido com ${fullTrajectoryPoints.length} pontos...`);
    
    // Cria um trajeto único com todos os pontos, colorido pela velocidade média
    const allLatLngs = fullTrajectoryPoints.map(p => [p.lat, p.lng]);
    const avgSpeed = fullTrajectoryPoints.reduce((sum, p) => sum + (p.velocity * 3.6), 0) / fullTrajectoryPoints.length;
    
    activityPolyline = L.polyline(allLatLngs, {
        color: getSpeedColor(avgSpeed),
        weight: 4,
        opacity: 0.8,
        smoothFactor: 1.0
    }).addTo(activityMap);

    // Handler de clique para sincronização
    activityPolyline.on('click', (e) => handleTrajectoryClickOptimized(e, fullTrajectoryPoints));

    activityPolyline.bindPopup(`
        <div style="font-size: 12px;">
            <strong>📊 Trajeto Completo</strong><br>
            🏃 Velocidade média: ${avgSpeed.toFixed(1)} km/h<br>
            📏 ${fullTrajectoryPoints.length} pontos GPS<br>
            ⏱️ ${new Date(fullTrajectoryPoints[0].time).toLocaleTimeString('pt-BR')} - 
                 ${new Date(fullTrajectoryPoints[fullTrajectoryPoints.length-1].time).toLocaleTimeString('pt-BR')}
        </div>
    `);

    console.log(`✅ Trajeto principal criado (${allLatLngs.length} coordenadas)`);
}

// Handler de clique otimizado para trajeto completo
async function handleTrajectoryClickOptimized(e, fullTrajectoryPoints) {
    console.log("🖱️ Clique no trajeto detectado, buscando ponto mais próximo...");
    
    const clickLatLng = e.latlng;
    let closestPoint = null;
    let minDistance = Infinity;
    
    // Busca otimizada
    if (fullTrajectoryPoints.length > 1000) {
        // Para trajetos grandes, faz amostragem primeiro
        const sampleStep = Math.ceil(fullTrajectoryPoints.length / 200);
        const sampledPoints = fullTrajectoryPoints.filter((_, index) => index % sampleStep === 0);
        
        // Encontra região aproximada
        sampledPoints.forEach(point => {
            const distance = clickLatLng.distanceTo([point.lat, point.lng]);
            if (distance < minDistance) {
                minDistance = distance;
                closestPoint = point;
            }
        });
        
        // Refina busca na região próxima
        const closestIndex = fullTrajectoryPoints.findIndex(p => p.time === closestPoint.time);
        const searchRange = Math.min(100, Math.floor(fullTrajectoryPoints.length / 20));
        const startIdx = Math.max(0, closestIndex - searchRange);
        const endIdx = Math.min(fullTrajectoryPoints.length - 1, closestIndex + searchRange);
        
        minDistance = Infinity;
        for (let i = startIdx; i <= endIdx; i++) {
            const point = fullTrajectoryPoints[i];
            const distance = clickLatLng.distanceTo([point.lat, point.lng]);
            if (distance < minDistance) {
                minDistance = distance;
                closestPoint = point;
            }
        }
    } else {
        // Para trajetos menores, busca linear simples
        fullTrajectoryPoints.forEach(point => {
            const distance = clickLatLng.distanceTo([point.lat, point.lng]);
            if (distance < minDistance) {
                minDistance = distance;
                closestPoint = point;
            }
        });
    }
    
    if (closestPoint) {
        console.log(`✅ Ponto mais próximo: ${closestPoint.time} (${minDistance.toFixed(2)}m de distância)`);
        manualSyncTime = closestPoint.time;
        updateVideoStartMarker(closestPoint.lat, closestPoint.lng, '▶️ Início Manual do Vídeo');
        
        const timeStr = new Date(closestPoint.time).toLocaleTimeString('pt-BR');
        const speedStr = (closestPoint.velocity * 3.6).toFixed(1);
        showMessage(result, `🎯 Sincronização: ${timeStr} (${speedStr} km/h)`, 'success');
    }
}

// Funções auxiliares do mapa
function getSpeedColor(speedKmh) {
    if (speedKmh > 40) return '#dc3545'; // Vermelho - muito rápido
    if (speedKmh > 25) return '#fd7e14'; // Laranja - rápido  
    if (speedKmh > 15) return '#ffc107'; // Amarelo - moderado
    if (speedKmh > 8) return '#28a745';  // Verde - lento
    return '#6c757d'; // Cinza - muito lento/parado
}

function createCustomIcon(emoji, color) {
    return L.divIcon({
        html: `<div style="
            background-color: ${color}; 
            border-radius: 50%; 
            width: 30px; 
            height: 30px; 
            display: flex; 
            align-items: center; 
            justify-content: center; 
            font-size: 14px;
            border: 2px solid white;
            box-shadow: 0 2px 5px rgba(0,0,0,0.3);
        ">${emoji}</div>`,
        iconSize: [30, 30],
        iconAnchor: [15, 15]
    });
}

// Adiciona controles avançados com densidade de marcadores
function addAdvancedTrajectoryControls(activityId) {
    // Controle de densidade dos marcadores
    const densityControl = L.control({ position: 'topright' });
    
    densityControl.onAdd = function() {
        const div = L.DomUtil.create('div', 'density-control');
        div.innerHTML = `
            <div style="
                background: rgba(22, 27, 34, 0.95);
                border: 1px solid #30363d;
                border-radius: 8px;
                padding: 10px;
                margin-bottom: 10px;
                min-width: 180px;
            ">
                <div style="font-weight: bold; margin-bottom: 8px; color: #c9d1d9; font-size: 13px;">
                    📍 Densidade de Marcadores
                </div>
                <select id="density-selector" style="
                    width: 100%;
                    padding: 6px;
                    border: 1px solid #30363d;
                    border-radius: 4px;
                    background: #161b22;
                    color: #c9d1d9;
                    font-size: 12px;
                ">
                    <option value="low">📊 Baixa (60s)</option>
                    <option value="medium" selected>📊 Média (30s + eventos)</option>
                    <option value="high">📊 Alta (15s)</option>
                    <option value="ultra_high">📊 Ultra (5s)</option>
                </select>
                <div style="
                    font-size: 10px; 
                    color: #8b949e; 
                    margin-top: 4px;
                    text-align: center;
                ">
                    Atual: <span id="current-density">Média</span>
                </div>
            </div>
        `;
        
        // Previne propagação de eventos do mapa
        L.DomEvent.disableClickPropagation(div);
        
        return div;
    };
    
    densityControl.addTo(activityMap);
    
    // Handler para mudança de densidade
    setTimeout(() => {
        const selector = document.getElementById('density-selector');
        const currentLabel = document.getElementById('current-density');
        
        if (selector) {
            selector.addEventListener('change', async (e) => {
                const newDensity = e.target.value;
                console.log(`🔄 Alterando densidade para: ${newDensity}`);
                
                showMessage(result, `🔄 Carregando marcadores (${getDensityLabel(newDensity)})...`, 'info');
                
                const count = await loadGPSMarkersWithDensity(activityId, newDensity);
                if (currentLabel) {
                    currentLabel.textContent = getDensityLabel(newDensity);
                }
                
                console.log(`✅ Densidade alterada: ${count} marcadores`);
            });
        }
    }, 100);
    
    // Legenda de velocidade
    addTrajectoryControls();
}

// Adiciona controles de visualização
function addTrajectoryControls() {
    // Legenda de velocidade
    const legend = L.control({ position: 'bottomleft' });
    
    legend.onAdd = function() {
        const div = L.DomUtil.create('div', 'speed-legend');
        div.innerHTML = `
            <div style="font-weight: bold; margin-bottom: 5px;">🏃 Velocidade</div>
            <div class="speed-legend-item">
                <div class="speed-legend-color" style="background: #dc3545;"></div>
                > 40 km/h
            </div>
            <div class="speed-legend-item">
                <div class="speed-legend-color" style="background: #fd7e14;"></div>
                25-40 km/h  
            </div>
            <div class="speed-legend-item">
                <div class="speed-legend-color" style="background: #ffc107;"></div>
                15-25 km/h
            </div>
            <div class="speed-legend-item">
                <div class="speed-legend-color" style="background: #28a745;"></div>
                8-15 km/h
            </div>
            <div class="speed-legend-item">
                <div class="speed-legend-color" style="background: #6c757d;"></div>
                < 8 km/h
            </div>
        `;
        return div;
    };
    
    legend.addTo(activityMap);
}

// Carrega marcadores GPS com densidade
async function loadGPSMarkersWithDensity(activityId, density = 'medium') {
    try {
        console.log(`📍 Carregando marcadores GPS (densidade: ${density})...`);
        
        // Remove marcadores existentes
        if (currentGPSMarkersGroup && activityMap.hasLayer(currentGPSMarkersGroup)) {
            activityMap.removeLayer(currentGPSMarkersGroup);
        }
        
        // Carrega pontos com a densidade especificada
        let gpsMarkers;
        if (window.go.main.App.GetGPSPointsWithDensity) {
            // Se o método com densidade estiver disponível
            gpsMarkers = await window.go.main.App.GetGPSPointsWithDensity(activityId, density);
        } else {
            // Fallback para método padrão
            gpsMarkers = await window.go.main.App.GetAllGPSPoints(activityId);
        }

        if (!gpsMarkers || gpsMarkers.length === 0) {
            console.log("Nenhum marcador GPS encontrado");
            return 0;
        }

        console.log(`✅ ${gpsMarkers.length} marcadores carregados (densidade: ${density})`);

        // Cria grupo de marcadores
        currentGPSMarkersGroup = L.layerGroup();
        
        // Adiciona cada marcador
        gpsMarkers.forEach((point, index) => {
            const speed = point.velocity * 3.6;
            const color = getSpeedColor(speed);
            const time = new Date(point.time);
            
            // Tamanho baseado na densidade
            let radius = getDensityRadius(density);
            
            const marker = L.circleMarker([point.lat, point.lng], {
                radius: radius,
                fillColor: color,
                fillOpacity: 0.8,
                color: '#ffffff',
                weight: 1.5,
                opacity: 1
            });
            
            // Popup com informações detalhadas
            const timeStr = time.toLocaleTimeString('pt-BR');
            marker.bindPopup(`
                <div style="font-size: 12px;">
                    <strong>📍 Ponto GPS ${index + 1}</strong><br>
                    <strong>⏰ ${timeStr}</strong><br>
                    🏃 Velocidade: ${speed.toFixed(1)} km/h<br>
                    ⛰️ Altitude: ${point.altitude.toFixed(0)}m<br>
                    🧭 Direção: ${point.bearing.toFixed(0)}°<br>
                    📍 ${point.lat.toFixed(6)}, ${point.lng.toFixed(6)}<br>
                    <hr style="margin: 8px 0;">
                    <small><em>💡 Clique próximo no trajeto para sincronizar</em></small>
                </div>
            `);

            // Tooltip no hover
            marker.bindTooltip(`${timeStr} • ${speed.toFixed(1)} km/h`, {
                permanent: false,
                direction: 'top',
                offset: [0, -8],
                className: 'custom-tooltip'
            });
            
            currentGPSMarkersGroup.addLayer(marker);
        });
        
        // Adiciona ao mapa
        currentGPSMarkersGroup.addTo(activityMap);
        
        // Atualiza densidade atual
        currentMarkerDensity = density;
        
        // Feedback para usuário
        const densityLabel = getDensityLabel(density);
        showMessage(result, `📍 ${gpsMarkers.length} marcadores GPS (${densityLabel})`, 'success');
        setTimeout(() => showMessage(result, '', ''), 3000);
        
        return gpsMarkers.length;

    } catch (error) {
        console.error("Erro ao carregar marcadores GPS:", error);
        showMessage(result, `Erro ao carregar marcadores: ${error}`, 'error');
        return 0;
    }
}

// Funções auxiliares de densidade
function getDensityRadius(density) {
    switch(density) {
        case 'ultra_high': return 3;
        case 'high': return 4;
        case 'medium': return 4;
        case 'low': return 5;
        default: return 4;
    }
}

function getDensityLabel(density) {
    switch(density) {
        case 'ultra_high': return 'Ultra Alta';
        case 'high': return 'Alta';
        case 'medium': return 'Média';
        case 'low': return 'Baixa';
        default: return 'Média';
    }
}

// Fallback para trajeto simplificado
function loadFallbackTrajectory(activity) {
    console.log("Carregando trajeto simplificado (fallback)");
    
    if (activity.map && activity.map.summary_polyline) {
        const latlngs = L.Polyline.fromEncoded(activity.map.summary_polyline).getLatLngs();
        activityPolyline = L.polyline(latlngs, { color: '#f85149', weight: 3 }).addTo(activityMap);
        
        activityPolyline.on('click', handleMapClick);
        activityMap.fitBounds(activityPolyline.getBounds());
        
        L.marker(latlngs[0]).addTo(activityMap).bindPopup('🏁 Início');
        L.marker(latlngs[latlngs.length - 1]).addTo(activityMap).bindPopup('🏆 Fim');
        
        showMessage(result, 'Trajeto básico carregado (dados GPS limitados)', 'info');
    }
}

// Handler de clique no mapa para sincronização manual
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

// Atualiza marcador de início do vídeo
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
            activityMap.setView([lat, lng], 16);
            console.log("Mapa centralizado e com zoom no novo marcador de início.");
        } catch (error) {
            console.error("Erro ao centralizar mapa no marcador:", error);
        }
    }, 200);
}

// ========================================
// FUNÇÕES DE VÍDEO
// ========================================

async function selectVideo() {
    try {
        const path = await window.go.main.App.SelectVideoFile();
        if (!path) return;

        selectedVideoPath = path;
        manualSyncTime = ""; // Reseta ao selecionar um novo vídeo

        const fileName = path.split(/[\\/]/).pop();
        if (videoInfo) {
            videoInfo.innerHTML = `
                <h4>Vídeo Selecionado</h4>
                <p><strong>Arquivo:</strong> ${fileName}</p>
            `;
        }
        
        if (processBtn) {
            processBtn.disabled = false;
        }

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
        if (processBtn) {
            processBtn.disabled = true;
            processBtn.textContent = 'Processando...';
        }
        
        if (progress) {
            progress.classList.remove('hidden');
        }
        
        showMessage(result, '', '');
        simulateProgress();

        // Passa o tempo manual (pode ser uma string vazia) para o backend
        const outputPath = await window.go.main.App.ProcessVideoOverlay(selectedActivity.id, selectedVideoPath, manualSyncTime);
        
        updateProgress(100);
        showMessage(result, `Vídeo processado com sucesso!<br><strong>Local:</strong> ${outputPath}`, 'success');
    } catch (error) {
        showMessage(result, `Erro no processamento: ${error}`, 'error');
    } finally {
        if (processBtn) {
            processBtn.disabled = false;
            processBtn.textContent = 'Processar com Overlay';
        }
        
        setTimeout(() => {
            if (progress) {
                progress.classList.add('hidden');
            }
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
    if (progressBar) {
        progressBar.style.width = `${value}%`;
    }
    if (progressText) {
        progressText.textContent = `${Math.round(value)}%`;
    }
}

// ========================================
// FUNÇÕES UTILITÁRIAS
// ========================================

function showMessage(container, message, type) {
    try {
        if (container) {
            container.innerHTML = message ? `<div class="${type}">${message}</div>` : '';
        }
    } catch (error) {
        console.error('❌ Erro ao mostrar mensagem:', error);
    }
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
        'WeightTraining': 'Musculação',
        'VirtualRide': 'Ciclismo Virtual',
        'VirtualRun': 'Corrida Virtual',
        'EBikeRide': 'E-Bike',
        'Velomobile': 'Velomobile',
        'AlpineSki': 'Esqui Alpino',
        'BackcountrySki': 'Esqui Backcountry',
        'Canoeing': 'Canoagem',
        'Crossfit': 'Crossfit',
        'Elliptical': 'Elíptico',
        'Golf': 'Golfe',
        'Handcycle': 'Handbike',
        'IceSkate': 'Patinação no Gelo',
        'InlineSkate': 'Patinação Inline',
        'Kayaking': 'Caiaque',
        'Kitesurf': 'Kitesurf',
        'NordicSki': 'Esqui Nórdico',
        'RockClimbing': 'Escalada',
        'RollerSki': 'Ski com Rodas',
        'Rowing': 'Remo',
        'Sail': 'Vela',
        'Skateboard': 'Skate',
        'Snowboard': 'Snowboard',
        'Snowshoe': 'Caminhada na Neve',
        'Soccer': 'Futebol',
        'StairStepper': 'Escada',
        'StandUpPaddling': 'Stand Up Paddle',
        'Surfing': 'Surf',
        'Tennis': 'Tênis',
        'Volleyball': 'Vôlei',
        'Wheelchair': 'Cadeira de Rodas',
        'Windsurf': 'Windsurf',
        'Yoga': 'Yoga'
    };
    return translations[type] || type;
}

// ========================================
// FUNÇÕES AUXILIARES DO MAPA
// ========================================

// Limpa controles antigos
function clearTrajectoryControls() {
    // Remove controles existentes se houver
    if (activityMap) {
        activityMap.eachLayer((layer) => {
            if (layer instanceof L.Control) {
                activityMap.removeControl(layer);
            }
        });
    }
}

// Reseta marcadores quando necessário
function clearGPSMarkers() {
    if (activityMap) {
        activityMap.eachLayer((layer) => {
            if (layer instanceof L.CircleMarker && !(layer instanceof L.Marker)) {
                activityMap.removeLayer(layer);
            }
        });
    }
}

// ========================================
// MENSAGEM FINAL DE CARREGAMENTO
// ========================================

console.log('✅ main.js carregado completamente - Versão com Paginação Completa');