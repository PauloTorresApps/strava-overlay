console.log('🚀 app.js carregando...');

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
    authBtn = document.getElementById('authBtn');
    statusDiv = document.getElementById('status');
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
    loadMoreBtn = document.getElementById('loadMoreBtn');
    filterGPSCheckbox = document.getElementById('filterGPS');
    totalActivitiesSpan = document.getElementById('totalActivities');
    gpsActivitiesSpan = document.getElementById('gpsActivities');
    console.log('✅ Elementos DOM inicializados');
}

/**
 * Adiciona os event listeners aos elementos do DOM.
 */
function addEventListeners() {
    if (authBtn) authBtn.addEventListener('click', authenticateStrava);
    if (selectVideoBtn) selectVideoBtn.addEventListener('click', selectVideo);
    if (processBtn) processBtn.addEventListener('click', processVideo);
    if (loadMoreBtn) loadMoreBtn.addEventListener('click', loadMoreActivities);
    if (filterGPSCheckbox) filterGPSCheckbox.addEventListener('change', handleFilterChange);
    console.log('✅ Event listeners adicionados');
}

// --- Ponto de Entrada ---
// Garante que o DOM esteja pronto antes de executar o script principal.
document.addEventListener('DOMContentLoaded', initApp);
