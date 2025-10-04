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
 * Simula um progresso para feedback visual durante o processamento.
 */
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