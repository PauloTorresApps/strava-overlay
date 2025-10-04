console.log('🚴 activities.js carregando...');

/**
 * Carrega uma página específica de atividades
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
        showMessage(result, window.t('errors.loadFailed', 'Erro ao carregar') + `: ${error}`, 'error');
    } finally {
        isLoadingMore = false;
        updateLoadMoreButton(false);
    }
}

/**
 * Carrega a próxima página de atividades
 */
function loadMoreActivities() {
    if (!hasMorePages || isLoadingMore) return;
    loadActivitiesPage(currentPage + 1);
}

/**
 * Filtra as atividades com base no checkbox
 */
function getFilteredActivities() {
    return showOnlyGPS ? allActivities.filter(activity => activity.has_gps) : allActivities;
}

/**
 * Manipula a mudança no filtro de GPS
 */
function handleFilterChange(event) {
    showOnlyGPS = event.target.checked;
    displayActivities(getFilteredActivities());
    updateStatistics();
}

/**
 * Atualiza as estatísticas de atividades
 */
function updateStatistics() {
    const totalCount = allActivities.length;
    const gpsCount = allActivities.filter(a => a.has_gps).length;

    if (totalActivitiesSpan) {
        totalActivitiesSpan.textContent = `${totalCount} ${window.t('activities.stats.total', 'atividades carregadas')}`;
    }
    if (gpsActivitiesSpan) {
        gpsActivitiesSpan.textContent = `${gpsCount} ${window.t('activities.stats.withGPS', 'com GPS')}`;
    }
}

/**
 * Renderiza a lista de atividades
 */
function displayActivities(activities) {
    if (!activitiesGrid) return;
    activitiesGrid.innerHTML = '';

    if (!activities || activities.length === 0) {
        activitiesGrid.innerHTML = `<p>${window.t('activities.noActivities', 'Nenhuma atividade encontrada com os filtros aplicados.')}</p>`;
        return;
    }
    
    activities.forEach(activity => {
        const card = createActivityCard(activity);
        activitiesGrid.appendChild(card);
    });
}

/**
 * Cria um card de atividade com i18n
 */
function createActivityCard(activity) {
    const card = document.createElement('div');
    card.className = 'activity-card';
    
    if (!activity.has_gps) {
        card.classList.add('no-gps');
        card.title = window.t('activities.noGPS', 'Esta atividade não possui dados GPS');
    }

    if (activity.has_gps) {
        card.onclick = () => selectActivity(activity, card);
    } else {
        card.style.cursor = 'not-allowed';
    }

    const activityDate = new Date(activity.start_date);
    const dateStr = formatDate(activityDate);
    const timeStr = formatTime(activityDate);

    const gpsBadge = activity.has_gps 
        ? '<span class="gps-badge has-gps">GPS</span>' 
        : `<span class="gps-badge no-gps">${window.t('activities.noGPS', 'Sem GPS')}</span>`;
    
    const activityIcon = getActivityIcon(activity.type);

    card.innerHTML = `
        <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px;">
            <h3 style="margin: 0; font-size: 1.1rem; line-height: 1.3; flex: 1;">${activity.name}</h3>
            ${gpsBadge}
        </div>
        <div style="color: var(--secondary-text); font-size: 0.9rem;">
            <div style="margin-bottom: 4px;">
                <strong style="color: var(--accent-color);">${activityIcon} ${translateActivityType(activity.type)}</strong>
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
 * Seleciona uma atividade
 */
async function selectActivity(activity, cardElement) {
    try {
        document.querySelectorAll('.activity-card.selected').forEach(el => el.classList.remove('selected'));
        cardElement.classList.add('selected');
        selectedActivity = activity;

        const detail = await window.go.main.App.GetActivityDetail(activity.id);
        selectedActivity.detail = detail;
        
        displayActivityDetailWithI18n(detail);
        await displayMap(activity);

        if (activityDetail) activityDetail.classList.remove('hidden');
        if (videoSection) videoSection.classList.remove('hidden');

    } catch (error) {
        showMessage(result, window.t('errors.loadFailed', 'Erro ao carregar') + `: ${error}`, 'error');
    }
}

/**
 * Atualiza a lista de atividades
 */
async function refreshActivities() {
    if (isLoadingMore) return;
    
    console.log('🔄 Atualizando lista de atividades...');
    
    if (refreshActivitiesBtn) {
        refreshActivitiesBtn.disabled = true;
        refreshActivitiesBtn.innerHTML = `⏳ ${window.t('activities.loading', 'Carregando...')}`;
    }
    
    try {
        allActivities = [];
        currentPage = 1;
        hasMorePages = true;
        
        if (activitiesGrid) {
            activitiesGrid.innerHTML = `<p>${window.t('activities.loading', 'Carregando atividades...')}</p>`;
        }
        
        await loadActivitiesPage(1);
        
        showMessage(result, window.t('messages.success', '✅ Lista de atividades atualizada'), 'success');
        
    } catch (error) {
        console.error('❌ Erro ao atualizar atividades:', error);
        showMessage(result, window.t('errors.loadFailed', 'Erro ao atualizar') + `: ${error.message}`, 'error');
    } finally {
        if (refreshActivitiesBtn) {
            refreshActivitiesBtn.disabled = false;
            refreshActivitiesBtn.innerHTML = `🔄 ${window.t('activities.refresh', 'Atualizar Lista')}`;
        }
    }
}