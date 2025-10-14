console.log('📹 video.js carregando...');

let progressStages = {
    'init': { label: 'Inicializando', icon: '⚙️' },
    'metadata': { label: 'Metadados', icon: '📄' },
    'activity': { label: 'Atividade', icon: '🚴' },
    'sync': { label: 'Sincronização', icon: '🔄' },
    'gps': { label: 'Dados GPS', icon: '📍' },
    'overlay': { label: 'Gerando Overlays', icon: '🎨' },
    'output': { label: 'Preparando Saída', icon: '💾' },
    'encoding': { label: 'Codificando Vídeo', icon: '🎬' },
    'complete': { label: 'Concluído', icon: '✅' }
};

/**
 * Abre o seletor de arquivos de vídeo e busca o ponto de início automático.
 */
async function selectVideo() {
    try {
        const path = await window.go.main.App.SelectVideoFile();
        if (!path) return;

        selectedVideoPath = path;
        manualSyncTime = "";

        const fileName = path.split(/[\\/]/).pop();
        if (videoInfo) {
            videoInfo.innerHTML = `<h4>Vídeo Selecionado:</h4><p>${fileName}</p>`;
        }
        if (processBtn) processBtn.disabled = false;

        if (window.overlayPosition) {
            window.overlayPosition.show();
        }

        console.log("Buscando ponto GPS para sincronização automática...");
        const point = await window.go.main.App.GetGPSPointForVideoTime(selectedActivity.id, path);

        if (point?.lat && point.lng) {
            updateVideoStartMarker(point.lat, point.lng, '▶️ Início Automático (Clique no trajeto para ajustar)');
            showMessage(result, 'Ponto de início automático encontrado!', 'success');
        } else {
            showMessage(result, 'Não foi possível encontrar o ponto de início automático. Clique no mapa para definir manualmente.', 'info');
        }
    } catch (error) {
        showMessage(result, `Erro ao selecionar vídeo: ${error}`, 'error');
    }
}

/**
 * Envia a atividade e o vídeo para o backend para processamento do overlay.
 */
async function processVideo() {
    if (!selectedActivity || !selectedVideoPath) {
        showMessage(result, 'Selecione uma atividade e um vídeo primeiro.', 'error');
        return;
    }

    try {
        if (processBtn) {
            processBtn.disabled = true;
            processBtn.textContent = 'Processando...';
        }
        
        // Mostra barra de progresso
        if (progress) progress.classList.remove('hidden');
        updateProgress(0);
        showMessage(result, '', '');

        // Escuta eventos de progresso
        const unsubscribe = window.runtime.EventsOn('video:progress', (data) => {
            console.log('📊 Progresso:', data);
            updateDetailedProgress(data.stage, data.progress, data.message);
        });

        const overlayPosition = window.overlayPosition ? window.overlayPosition.getPosition() : 'bottom-left';
        console.log(`📍 Processando vídeo com overlay na posição: ${overlayPosition}`);

        const outputPath = await window.go.main.App.ProcessVideoOverlay(
            selectedActivity.id, 
            selectedVideoPath, 
            manualSyncTime,
            overlayPosition
        );
        
        // Remove listener de eventos
        unsubscribe();
        
        updateProgress(100);
        showMessage(result, `Vídeo processado com sucesso!<br><strong>Local:</strong> ${outputPath}`, 'success');
        
        if (window.overlayPosition) {
            window.overlayPosition.hide();
        }
    } catch (error) {
        updateProgress(0);
        showMessage(result, `Erro no processamento: ${error}`, 'error');
    } finally {
        if (processBtn) {
            processBtn.disabled = false;
            processBtn.textContent = 'Processar com Overlay';
        }
        setTimeout(() => {
            if (progress) progress.classList.add('hidden');
            updateProgress(0);
        }, 5000);
    }
}

/**
 * Atualiza a barra de progresso com detalhes do estágio atual
 */
function updateDetailedProgress(stage, progressValue, message) {
    const progressBar = document.getElementById('progressBar');
    const progressText = document.getElementById('progressText');
    
    if (progressBar) {
        progressBar.style.width = `${progressValue}%`;
    }
    
    if (progressText) {
        const stageInfo = progressStages[stage] || { label: stage, icon: '⚙️' };
        progressText.textContent = `${stageInfo.icon} ${stageInfo.label}: ${Math.round(progressValue)}%`;
    }
    
    // Atualiza mensagem detalhada
    if (message) {
        const progressContainer = document.getElementById('progress');
        let messageDiv = document.getElementById('progressMessage');
        
        if (!messageDiv) {
            messageDiv = document.createElement('div');
            messageDiv.id = 'progressMessage';
            messageDiv.style.cssText = `
                margin-top: 10px;
                padding: 8px 12px;
                background: var(--info-bg);
                border: 1px solid var(--info-border);
                border-radius: 4px;
                color: var(--info-text);
                font-size: 0.9rem;
                text-align: center;
            `;
            progressContainer.appendChild(messageDiv);
        }
        
        messageDiv.textContent = message;
    }
}