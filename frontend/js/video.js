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

let isProcessing = false;
let progressUnsubscribe = null;

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
        isProcessing = true;
        
        // Atualiza botões
        if (processBtn) {
            processBtn.disabled = true;
            processBtn.style.display = 'none';
        }
        
        showCancelButton();
        
        if (progress) progress.classList.remove('hidden');
        updateProgress(0);
        showMessage(result, '', '');

        // Escuta eventos de progresso
        progressUnsubscribe = window.runtime.EventsOn('video:progress', (data) => {
            console.log('📊 Progresso:', data);
            updateDetailedProgress(data.stage, data.progress, data.message);
        });
        
        // Escuta conclusão para notificação
        const completionUnsubscribe = window.runtime.EventsOn('video:completed', (data) => {
            if (data.success) {
                const fileName = data.outputPath.split(/[\\/]/).pop();
                window.go.main.App.SendNotification(
                    '✅ Vídeo Processado!',
                    `Arquivo pronto: ${fileName}`
                );
            } else {
                window.go.main.App.SendNotification(
                    '❌ Erro no Processamento',
                    data.error
                );
            }
            completionUnsubscribe();
        });

        const overlayPosition = window.overlayPosition ? window.overlayPosition.getPosition() : 'bottom-left';
        console.log(`📍 Processando vídeo com overlay na posição: ${overlayPosition}`);

        const outputPath = await window.go.main.App.ProcessVideoOverlay(
            selectedActivity.id, 
            selectedVideoPath, 
            manualSyncTime,
            overlayPosition
        );
        
        if (progressUnsubscribe) {
            progressUnsubscribe();
            progressUnsubscribe = null;
        }
        
        updateProgress(100);
        showMessage(result, `Vídeo processado com sucesso!<br><strong>Local:</strong> ${outputPath}`, 'success');
        
        if (window.overlayPosition) {
            window.overlayPosition.hide();
        }
    } catch (error) {
        if (progressUnsubscribe) {
            progressUnsubscribe();
            progressUnsubscribe = null;
        }
        
        const errorMsg = error.toString();
        if (errorMsg.includes('cancelado')) {
            window.go.main.App.SendNotification('⚠️ Cancelado', 'Processamento cancelado pelo usuário');
            showMessage(result, '⚠️ Processamento cancelado pelo usuário', 'info');
        } else {
            window.go.main.App.SendNotification('❌ Erro', errorMsg);
            showMessage(result, `Erro no processamento: ${error}`, 'error');
        }
        updateProgress(0);
    } finally {
        isProcessing = false;
        hideCancelButton();
        
        if (processBtn) {
            processBtn.disabled = false;
            processBtn.style.display = 'inline-block';
            processBtn.textContent = 'Processar com Overlay';
        }
        
        setTimeout(() => {
            if (progress) progress.classList.add('hidden');
            clearProgressMessage();
            updateProgress(0);
        }, 5000);
    }
}

/**
 * Cancela o processamento em andamento
 */
async function cancelProcessing() {
    if (!isProcessing) return;
    
    try {
        const confirmed = confirm('Deseja realmente cancelar o processamento?');
        if (!confirmed) return;
        
        await window.go.main.App.CancelVideoProcessing();
        console.log('🛑 Cancelamento solicitado');
    } catch (error) {
        console.error('Erro ao cancelar:', error);
        showMessage(result, `Erro ao cancelar: ${error}`, 'error');
    }
}

/**
 * Mostra o botão de cancelar
 */
function showCancelButton() {
    let cancelBtn = document.getElementById('cancelProcessBtn');
    
    if (!cancelBtn) {
        cancelBtn = document.createElement('button');
        cancelBtn.id = 'cancelProcessBtn';
        cancelBtn.textContent = '🛑 Cancelar Processamento';
        cancelBtn.style.cssText = `
            background-color: #dc3545;
            margin-left: 10px;
        `;
        cancelBtn.onclick = cancelProcessing;
        
        processBtn.parentNode.insertBefore(cancelBtn, processBtn.nextSibling);
    }
    
    cancelBtn.style.display = 'inline-block';
}

/**
 * Esconde o botão de cancelar
 */
function hideCancelButton() {
    const cancelBtn = document.getElementById('cancelProcessBtn');
    if (cancelBtn) {
        cancelBtn.style.display = 'none';
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

/**
 * Limpa mensagem de progresso detalhado
 */
function clearProgressMessage() {
    const messageDiv = document.getElementById('progressMessage');
    if (messageDiv) {
        messageDiv.remove();
    }
}

// Listener global para notificações
if (window.runtime) {
    window.runtime.EventsOn('system:notification', (data) => {
        console.log('🔔 Notificação:', data.title, '-', data.message);
        
        // Mostra toast visual
        if (window.toastManager) {
            window.toastManager.success(data.message, {
                title: data.title,
                duration: 10000
            });
        } else {
            // Fallback se toast não existir
            alert(`${data.title}\n\n${data.message}`);
        }
    });
}