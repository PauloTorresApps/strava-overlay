console.log('🎨 ui.js carregando...');

/**
 * Exibe uma mensagem na tela em um container específico.
 * @param {HTMLElement} container - O elemento DOM onde a mensagem será exibida.
 * @param {string} message - O conteúdo da mensagem. Pode incluir HTML.
 * @param {'success'|'error'|'info'} type - O tipo de mensagem para estilização.
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
 * @param {number} value - O valor do progresso (0 a 100).
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
 * @param {boolean} isLoading - Indica se as atividades estão sendo carregadas.
 */
function updateLoadMoreButton(isLoading) {
    if (!loadMoreBtn) return;

    if (isLoading) {
        loadMoreBtn.disabled = true;
        loadMoreBtn.textContent = 'Carregando...';
    } else {
        loadMoreBtn.disabled = !hasMorePages;
        loadMoreBtn.textContent = hasMorePages ? 'Carregar Mais Atividades' : 'Todas as atividades foram carregadas';
    }
}
