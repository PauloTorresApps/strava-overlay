console.log('🚀 app.js carregando (versão corrigida)...');

/**
 * Função de inicialização principal da aplicação.
 * É chamada quando o DOM está completamente carregado.
 */
function initApp() {
    console.log('🚀 Strava Add Overlay iniciado');
    initializeDOMElements();
    addEventListeners();
    
    // Verifica a autenticação automaticamente na inicialização
    setTimeout(checkAuthenticationOnStartup, 500);
}

/**
 * Mapeia as variáveis globais para os elementos do DOM.
 */
function initializeDOMElements() {
    // Elementos do header (removido authStatus que não existe mais)
    statusIndicator = document.getElementById('statusIndicator');
    statusText = document.getElementById('statusText');
    authBtn = document.getElementById('authBtn');
    
    // Seções principais
    activitiesSection = document.getElementById('activitiesSection');
    activitiesGrid = document.getElementById('activitiesGrid');
    activityDetail = document.getElementById('activityDetail');
    activityInfo = document.getElementById('activityInfo');
    mapContainer = document.getElementById('mapContainer');
    videoSection = document.getElementById('videoSection');
    
    // Controles de vídeo
    selectVideoBtn = document.getElementById('selectVideoBtn');
    videoInfo = document.getElementById('videoInfo');
    processBtn = document.getElementById('processBtn');
    progress = document.getElementById('progress');
    progressBar = document.getElementById('progressBar');
    progressText = document.getElementById('progressText');
    result = document.getElementById('result');
    
    // Controles de atividades
    loadMoreBtn = document.getElementById('loadMoreBtn');
    filterGPSCheckbox = document.getElementById('filterGPS');
    totalActivitiesSpan = document.getElementById('totalActivities');
    gpsActivitiesSpan = document.getElementById('gpsActivities');
    
    // Debug: verificar elementos críticos
    const criticalElements = {
        mapContainer,
        activitiesSection,
        activitiesGrid,
        selectVideoBtn,
        processBtn
    };
    
    const missing = Object.entries(criticalElements)
        .filter(([name, element]) => !element)
        .map(([name]) => name);
        
    if (missing.length > 0) {
        console.warn('⚠️ Elementos DOM faltando:', missing);
    } else {
        console.log('✅ Todos os elementos DOM críticos encontrados');
    }
}

/**
 * Adiciona os event listeners aos elementos do DOM.
 */
function addEventListeners() {
    // Event listeners para funcionalidades de vídeo
    if (selectVideoBtn) selectVideoBtn.addEventListener('click', selectVideo);
    if (processBtn) processBtn.addEventListener('click', processVideo);
    
    // Event listeners para atividades
    if (loadMoreBtn) loadMoreBtn.addEventListener('click', loadMoreActivities);
    if (filterGPSCheckbox) filterGPSCheckbox.addEventListener('change', handleFilterChange);
    
    // Event listener para redimensionamento da janela (importante para o mapa)
    window.addEventListener('resize', debounce(() => {
        if (activityMap) {
            console.log('🔄 Redimensionamento detectado, invalidando mapa...');
            setTimeout(() => {
                activityMap.invalidateSize();
            }, 100);
        }
    }, 250));
    
    console.log('✅ Event listeners adicionados');
}

/**
 * Seleciona uma atividade e carrega seus detalhes.
 * @param {object} activity - A atividade selecionada.
 * @param {HTMLElement} cardElement - O elemento do card clicado.
 */
async function selectActivity(activity, cardElement) {
    try {
        console.log('🎯 Selecionando atividade:', activity.name);
        
        // Remove seleção anterior
        document.querySelectorAll('.activity-card.selected').forEach(el => {
            el.classList.remove('selected');
        });
        
        // Marca nova seleção
        cardElement.classList.add('selected');
        selectedActivity = activity;

        // Carrega detalhes da atividade
        console.log('📊 Carregando detalhes da atividade...');
        const detail = await window.go.main.App.GetActivityDetail(activity.id);
        displayActivityDetail(detail);
        
        // Mostra seções
        if (activityDetail) activityDetail.classList.remove('hidden');
        if (videoSection) videoSection.classList.remove('hidden');
        
        // Carrega mapa - com delay para garantir que a seção esteja visível
        console.log('🗺️ Preparando para carregar mapa...');
        setTimeout(async () => {
            try {
                await displayMap(activity);
            } catch (error) {
                console.error('❌ Erro ao carregar mapa:', error);
                showMessage(result, `Erro ao carregar mapa: ${error.message}`, 'error');
            }
        }, 100);

    } catch (error) {
        console.error('❌ Erro ao selecionar atividade:', error);
        showMessage(result, `Erro ao carregar detalhes: ${error.message}`, 'error');
    }
}

/**
 * Exibe os detalhes de uma atividade na seção de informações.
 * @param {object} detail - Os dados detalhados da atividade.
 */
function displayActivityDetail(detail) {
    if (!activityInfo) return;

    const startDate = new Date(detail.start_date);
    const distance = (detail.distance / 1000).toFixed(2);
    const elevation = detail.total_elevation_gain ? detail.total_elevation_gain.toFixed(0) : 'N/A';
    const maxSpeed = detail.max_speed ? (detail.max_speed * 3.6).toFixed(1) : 'N/A';
    const calories = detail.calories ? detail.calories.toFixed(0) : 'N/A';

    activityInfo.innerHTML = `
        <div class="info-grid">
            <div class="info-item">
                <h4>Informações Básicas</h4>
                <p><strong>Nome:</strong> ${detail.name}</p>
                <p><strong>Tipo:</strong> ${translateActivityType(detail.type)}</p>
                <p><strong>Data:</strong> ${formatDate(startDate)}</p>
                <p><strong>Horário:</strong> ${formatTime(startDate)}</p>
            </div>
            <div class="info-item">
                <h4>Desempenho</h4>
                <p><strong>Distância:</strong> ${distance} km</p>
                <p><strong>Duração:</strong> ${formatDuration(detail.moving_time)}</p>
                <p><strong>Vel. Máxima:</strong> ${maxSpeed} km/h</p>
                <p><strong>Calorias:</strong> ${calories}</p>
                <p><strong>Ganho de Elevação:</strong> ${elevation} m</p>
            </div>
        </div>
    `;
}

/**
 * Função debounce para evitar execuções excessivas.
 * @param {Function} func - Função a ser executada
 * @param {number} wait - Tempo de espera em ms
 * @returns {Function} Função com debounce aplicado
 */
function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

/**
 * Força a atualização do layout do mapa quando necessário.
 */
function forceMapUpdate() {
    if (activityMap && mapContainer) {
        console.log('🔄 Forçando atualização do mapa...');
        
        // Aguarda um pouco e então invalida o tamanho
        setTimeout(() => {
            try {
                activityMap.invalidateSize();
                console.log('✅ Mapa atualizado');
            } catch (error) {
                console.error('❌ Erro ao atualizar mapa:', error);
            }
        }, 150);
    }
}

/**
 * Observador de mudanças de visibilidade para corrigir problemas do mapa.
 */
function setupMapVisibilityObserver() {
    if (!mapContainer) return;
    
    const observer = new IntersectionObserver((entries) => {
        entries.forEach(entry => {
            if (entry.isIntersecting && activityMap) {
                console.log('👁️ Container do mapa tornou-se visível, atualizando...');
                forceMapUpdate();
            }
        });
    });
    
    observer.observe(mapContainer);
}

// --- Ponto de Entrada ---
document.addEventListener('DOMContentLoaded', () => {
    initApp();
    
    // Configura observador do mapa após inicialização
    setTimeout(setupMapVisibilityObserver, 1000);
});

// Adiciona handlers globais para depuração
window.addEventListener('error', (event) => {
    console.error('❌ Erro global:', event.error);
});

// Expõe funções úteis para debug
if (typeof window !== 'undefined') {
    window.forceMapUpdate = forceMapUpdate;
    window.selectedActivity = selectedActivity;
}