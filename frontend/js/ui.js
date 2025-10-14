console.log('🎨 ui.js carregando...');

/**
 * Exibe uma mensagem na tela em um container específico.
 */
function showMessage(container, message, type) {
    try {
        if (container) {
            container.innerHTML = message ? `<div class="${type}">${message}</div>` : '';
        }
    } catch (error) {
        console.error('❌ Erro ao mostrar mensagem:', error);
    }
}

/**
 * Atualiza a barra de progresso.
 */
function updateProgress(value) {
    const val = Math.round(value);
    if (progressBar) {
        progressBar.style.width = `${val}%`;
    }
    if (progressText) {
        progressText.textContent = `${val}%`;
    }
}

/**
 * NÃO usar mais simulateProgress - agora temos progresso real!
 * Mantida apenas para compatibilidade, mas não deve ser chamada.
 */
function simulateProgress() {
    console.warn('⚠️ simulateProgress() está deprecated - usando progresso real agora');
}

/**
 * Atualiza o estado do botão "Carregar Mais".
 */
function updateLoadMoreButton(isLoading) {
    if (!loadMoreBtn) return;

    if (isLoading) {
        loadMoreBtn.disabled = true;
        loadMoreBtn.textContent = window.t('activities.loading', 'Carregando...');
    } else {
        loadMoreBtn.disabled = !hasMorePages;
        loadMoreBtn.textContent = hasMorePages 
            ? window.t('activities.loadMore', 'Carregar Mais Atividades') 
            : window.t('activities.allLoaded', 'Todas as atividades foram carregadas');
    }
}

/**
 * Limpa mensagem de progresso detalhado
 */
function clearProgressMessage() {
    const messageDiv = document.getElementById('progressMessage');
    if (messageDiv) {
        messageDiv.remove();
    }
}