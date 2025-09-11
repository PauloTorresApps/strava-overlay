console.log('🚴 activities.js carregando...');

/**
 * Carrega uma página específica de atividades do backend.
 * @param {number} page - O número da página a ser carregada.
 */
async function loadActivitiesPage(page) {
    if (isLoadingMore) return;

    console.log(`📋 Carregando página ${page} de atividades...`);
    isLoadingMore = true;
    updateLoadMoreButton(true);

    try {
        const response = await window.go.main.App.GetActivitiesPage(page);
        if (!response) throw new Error('Resposta vazia do servidor');

        currentPage = page;
        hasMorePages = response.has_more;
        
        if (page === 1) allActivities = [];
        if (response.activities?.length > 0) {
            allActivities = allActivities.concat(response.activities);
        }

        displayActivities(getFilteredActivities());
        updateStatistics();

    } catch (error) {
        console.error('❌ Erro ao carregar atividades:', error);
        showMessage(result, `Erro: ${error}`, 'error');
    } finally {
        isLoadingMore = false;
        updateLoadMoreButton(false);
    }
}

/**
 * Carrega a próxima página de atividades.
 */
function loadMoreActivities() {
    if (!hasMorePages || isLoadingMore) return;
    loadActivitiesPage(currentPage + 1);
}

/**
 * Filtra as atividades com base na configuração do checkbox.
 * @returns {Array} A lista de atividades filtrada.
 */
function getFilteredActivities() {
    return showOnlyGPS ? allActivities.filter(activity => activity.has_gps) : allActivities;
}

/**
 * Manipula a mudança no filtro de GPS.
 * @param {Event} event - O evento de mudança do checkbox.
 */
function handleFilterChange(event) {
    showOnlyGPS = event.target.checked;
    displayActivities(getFilteredActivities());
    updateStatistics();
}

/**
 * Atualiza as estatísticas de atividades (total e com GPS).
 */
function updateStatistics() {
    const totalCount = allActivities.length;
    const gpsCount = allActivities.filter(a => a.has_gps).length;

    if (totalActivitiesSpan) totalActivitiesSpan.textContent = `${totalCount} atividades carregadas`;
    if (gpsActivitiesSpan) gpsActivitiesSpan.textContent = `${gpsCount} com GPS`;
}

/**
 * Renderiza a lista de atividades na tela.
 * @param {Array} activities - A lista de atividades para exibir.
 */
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

/**
 * Cria um elemento de card para uma atividade - VERSÃO SIMPLIFICADA.
 * @param {object} activity - Os dados da atividade.
 * @returns {HTMLElement} O elemento do card criado.
 */
function createActivityCard(activity) {
    const card = document.createElement('div');
    card.className = 'activity-card';
    
    // Adiciona classe especial para atividades sem GPS
    if (!activity.has_gps) {
        card.classList.add('no-gps');
        card.title = 'Esta atividade não possui dados GPS';
    }

    // Só permite seleção se tiver GPS
    if (activity.has_gps) {
        card.onclick = () => selectActivity(activity, card);
    } else {
        card.style.cursor = 'not-allowed';
    }

    // Formata data e hora
    const activityDate = new Date(activity.start_date);
    const dateStr = formatDate(activityDate);
    const timeStr = formatTime(activityDate);

    // Badge de GPS
    const gpsBadge = activity.has_gps 
        ? '<span class="gps-badge has-gps">GPS</span>' 
        : '<span class="gps-badge no-gps">Sem GPS</span>';

    // HTML simplificado - nome, tipo, data/hora e badge GPS
    card.innerHTML = `
        <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px;">
            <h3 style="margin: 0; font-size: 1.1rem; line-height: 1.3; flex: 1;">${activity.name}</h3>
            ${gpsBadge}
        </div>
        <div style="color: var(--secondary-text); font-size: 0.9rem;">
            <div style="margin-bottom: 4px;">
                <strong style="color: var(--accent-color);">🚴 ${translateActivityType(activity.type)}</strong>
            </div>
            <div style="margin-bottom: 4px;">
                <strong style="color: var(--primary-text);">📅 ${dateStr}</strong>
            </div>
            <div>
                <strong style="color: var(--primary-text);">🕒 ${timeStr}</strong>
            </div>
        </div>
    `;
    
    return card;
}

/**
 * Seleciona uma atividade, busca detalhes e exibe no mapa.
 * @param {object} activity - A atividade selecionada.
 * @param {HTMLElement} cardElement - O elemento do card clicado.
 */
async function selectActivity(activity, cardElement) {
    try {
        document.querySelectorAll('.activity-card.selected').forEach(el => el.classList.remove('selected'));
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
