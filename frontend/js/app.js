console.log('🚀 app.js carregando (versão com i18n)...');

/**
 * Função de inicialização principal da aplicação.
 */
async function initApp() {
    console.log('🚀 Strava Add Overlay iniciando');
    
    // Solicitar permissão de notificação logo ao iniciar
    if ("Notification" in window && Notification.permission === "default") {
        setTimeout(() => {
            Notification.requestPermission().then(permission => {
                console.log('🔔 Permissão de notificação:', permission);
            });
        }, 2000); // Aguarda 2s para não ser intrusivo
    }

    // 1. PRIMEIRO: Inicializa i18n
    try {
        console.log('🌍 Inicializando sistema de internacionalização...');
        await window.i18n.initialize();
        console.log('✅ Sistema i18n inicializado');
    } catch (error) {
        console.error('❌ Erro ao inicializar i18n:', error);
        console.log('🔄 Continuando sem i18n...');
    }
    
    // 2. Inicializa configurações
    try {
        console.log('⚙️ Inicializando configurações...');
        await window.initializeConfig();
        console.log('✅ Configurações inicializadas');
    } catch (error) {
        console.error('❌ Erro ao inicializar configurações:', error);
        console.log('🔄 Continuando com configurações padrão...');
    }
    
    // 3. Inicializa elementos DOM
    initializeDOMElements();
    
    // 4. Adiciona event listeners
    addEventListeners();

    // 5. Inicializa controle de posição
    if (window.overlayPosition) {
        window.overlayPosition.init();
        console.log('✅ Controle de posição inicializado');
    }
    
    // 6. Verifica autenticação
    setTimeout(checkAuthenticationOnStartup, 500);
    
    // 7. Escuta mudanças de idioma para atualizar UI dinâmica
    window.addEventListener('localeChanged', handleLocaleChange);
}

/**
 * Handler para mudanças de idioma
 */
function handleLocaleChange(event) {
    console.log(`🌍 Idioma alterado para: ${event.detail.locale}`);
    
    // Atualiza textos dinâmicos que não usam data-i18n
    updateDynamicTexts();
    
    // Re-renderiza atividades se estiverem carregadas
    if (allActivities && allActivities.length > 0) {
        displayActivities(getFilteredActivities());
    }
    
    // Atualiza detalhes da atividade se estiver visível
    if (selectedActivity && !activityDetail.classList.contains('hidden')) {
        // Re-renderiza os detalhes com novo idioma
        displayActivityDetailWithI18n(selectedActivity);
    }
    
    // Atualiza estatísticas
    updateStatistics();
    
    // Atualiza botão de carregar mais
    updateLoadMoreButton(isLoadingMore);
}

/**
 * Atualiza textos dinâmicos que não podem usar data-i18n
 */
function updateDynamicTexts() {
    // Atualiza título do documento
    document.title = window.t('app.title', 'Strava Video Overlay');
    
    // Atualiza placeholders se necessário
    const placeholders = document.querySelectorAll('[data-i18n-placeholder]');
    placeholders.forEach(element => {
        const key = element.getAttribute('data-i18n-placeholder');
        element.placeholder = window.t(key);
    });
}

/**
 * Mapeia as variáveis globais para os elementos do DOM
 */
function initializeDOMElements() {
    // Elementos do header
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
    refreshActivitiesBtn = document.getElementById('refreshActivitiesBtn');
    
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
 * Adiciona os event listeners aos elementos do DOM
 */
function addEventListeners() {
    if (selectVideoBtn) selectVideoBtn.addEventListener('click', selectVideo);
    if (processBtn) processBtn.addEventListener('click', processVideo);
    if (loadMoreBtn) loadMoreBtn.addEventListener('click', loadMoreActivities);
    if (filterGPSCheckbox) filterGPSCheckbox.addEventListener('change', handleFilterChange);
    if (refreshActivitiesBtn) refreshActivitiesBtn.addEventListener('click', refreshActivities);
    
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
 * Exibe detalhes da atividade com internacionalização
 */
function displayActivityDetailWithI18n(activity) {
    if (!activityInfo || !activity) return;

    const detail = activity.detail || activity; // Suporta tanto Activity quanto ActivityDetail
    const startDate = new Date(detail.start_date);
    const distance = (detail.distance / 1000).toFixed(2);
    const elevation = detail.total_elevation_gain ? detail.total_elevation_gain.toFixed(0) : 'N/A';
    const maxSpeed = detail.max_speed ? (detail.max_speed * 3.6).toFixed(1) : 'N/A';
    const calories = detail.calories ? detail.calories.toFixed(0) : 'N/A';

    activityInfo.innerHTML = `
        <div class="info-grid">
            <div class="info-item">
                <h4>${window.t('activityDetail.basicInfo.title', 'Informações Básicas')}</h4>
                <p><strong>${window.t('activityDetail.basicInfo.name', 'Nome')}:</strong> ${detail.name}</p>
                <p><strong>${window.t('activityDetail.basicInfo.type', 'Tipo')}:</strong> ${translateActivityType(detail.type)}</p>
                <p><strong>${window.t('activityDetail.basicInfo.date', 'Data')}:</strong> ${formatDate(startDate)}</p>
                <p><strong>${window.t('activityDetail.basicInfo.time', 'Horário')}:</strong> ${formatTime(startDate)}</p>
            </div>
            <div class="info-item">
                <h4>${window.t('activityDetail.performance.title', 'Desempenho')}</h4>
                <p><strong>${window.t('activityDetail.performance.distance', 'Distância')}:</strong> ${distance} km</p>
                <p><strong>${window.t('activityDetail.performance.duration', 'Duração')}:</strong> ${formatDuration(detail.moving_time)}</p>
                <p><strong>${window.t('activityDetail.performance.maxSpeed', 'Vel. Máxima')}:</strong> ${maxSpeed} km/h</p>
                <p><strong>${window.t('activityDetail.performance.calories', 'Calorias')}:</strong> ${calories}</p>
                <p><strong>${window.t('activityDetail.performance.elevation', 'Ganho de Elevação')}:</strong> ${elevation} m</p>
            </div>
        </div>
    `;
}

/**
 * Seleciona uma atividade e carrega seus detalhes
 */
async function selectActivity(activity, cardElement) {
    try {
        console.log('🎯 Selecionando atividade:', activity.name);
        
        document.querySelectorAll('.activity-card.selected').forEach(el => {
            el.classList.remove('selected');
        });
        
        cardElement.classList.add('selected');
        selectedActivity = activity;

        console.log('📊 Carregando detalhes da atividade...');
        const detail = await window.go.main.App.GetActivityDetail(activity.id);
        selectedActivity.detail = detail; // Armazena detail para uso posterior
        
        displayActivityDetailWithI18n(detail);
        
        if (activityDetail) activityDetail.classList.remove('hidden');
        if (videoSection) videoSection.classList.remove('hidden');
        
        console.log('🗺️ Preparando para carregar mapa...');
        setTimeout(async () => {
            try {
                await displayMap(activity);
            } catch (error) {
                console.error('❌ Erro ao carregar mapa:', error);
                showMessage(result, window.t('errors.loadFailed', 'Erro ao carregar') + `: ${error.message}`, 'error');
            }
        }, 100);

    } catch (error) {
        console.error('❌ Erro ao selecionar atividade:', error);
        showMessage(result, window.t('errors.loadFailed', 'Erro ao carregar') + `: ${error.message}`, 'error');
    }
}

/**
 * Função debounce
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
 * Força a atualização do layout do mapa
 */
function forceMapUpdate() {
    if (activityMap && mapContainer) {
        console.log('🔄 Forçando atualização do mapa...');
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
 * Observador de mudanças de visibilidade
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

/**
 * Mostra informações de configuração (debug)
 */
function showConfigInfo() {
    if (window.configService && window.configService.initialized) {
        const config = window.configService.getConfig();
        console.group('📋 Informações de Configuração');
        console.log('Versão da App:', config.app_version);
        console.log('Ambiente:', config.environment);
        console.log('Provedores disponíveis:', window.configService.getAvailableProviders());
        console.log('Thunderforest disponível:', window.configService.isProviderAvailable('thunderforest'));
        console.log('Mapbox disponível:', window.configService.isProviderAvailable('mapbox'));
        console.groupEnd();
    }
    
    if (window.i18n && window.i18n.currentLocale) {
        console.group('🌍 Informações de i18n');
        console.log('Idioma atual:', window.i18n.currentLocale);
        console.log('Idiomas disponíveis:', window.i18n.availableLocales.map(l => l.code));
        console.groupEnd();
    }
}

/**
 * Configura listeners para eventos de progresso
 */
function setupProgressListeners() {
    // O listener já está configurado em processVideo()
    // mas podemos adicionar logs globais aqui se necessário
    console.log('✅ Sistema de progresso em tempo real configurado');
}

// Chamar após initApp
document.addEventListener('DOMContentLoaded', () => {
    initApp();
    setupProgressListeners();
    setTimeout(setupMapVisibilityObserver, 1000);
    setTimeout(showConfigInfo, 2000);
});

// Handlers globais para depuração
window.addEventListener('error', (event) => {
    console.error('❌ Erro global:', event.error);
});

// Expõe funções úteis para debug
if (typeof window !== 'undefined') {
    window.forceMapUpdate = forceMapUpdate;
    window.selectedActivity = selectedActivity;
    window.showConfigInfo = showConfigInfo;
}