console.log('📹 video.js carregando...');

let notificationPermission = 'default';

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

async function processVideo() {
    if (!selectedActivity || !selectedVideoPath) {
        showMessage(result, 'Selecione uma atividade e um vídeo primeiro.', 'error');
        return;
    }

    await requestNotificationPermission();

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

        const completionUnsubscribe = window.runtime.EventsOn('video:completed', (data) => {
            if (data.success) {
                sendDesktopNotification(
                    '✅ Vídeo Processado!',
                    `Seu vídeo está pronto:\n${data.outputPath.split(/[\\/]/).pop()}`,
                );
                
                // Tocar som (opcional)
                playNotificationSound();
            } else {
                sendDesktopNotification(
                    '❌ Erro no Processamento',
                    `Falha ao processar o vídeo: ${data.error}`
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
            showMessage(result, '⚠️ Processamento cancelado pelo usuário', 'info');
        } else {
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

function hideCancelButton() {
    const cancelBtn = document.getElementById('cancelProcessBtn');
    if (cancelBtn) {
        cancelBtn.style.display = 'none';
    }
}

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

// Função para solicitar permissão de notificação
async function requestNotificationPermission() {
    if (!("Notification" in window)) {
        console.log("Este navegador não suporta notificações desktop");
        return false;
    }
    
    if (Notification.permission === "granted") {
        notificationPermission = "granted";
        return true;
    }
    
    if (Notification.permission !== "denied") {
        const permission = await Notification.requestPermission();
        notificationPermission = permission;
        return permission === "granted";
    }
    
    return false;
}

// Função para enviar notificação
function sendDesktopNotification(title, body, icon = null) {
    if (notificationPermission !== "granted") {
        console.log("Permissão de notificação não concedida");
        return;
    }
    
    const notification = new Notification(title, {
        body: body,
        icon: icon || '/wails-logo.png', // ou use um ícone personalizado
        badge: '/wails-logo.png',
        requireInteraction: false,
        silent: false
    });
    
    // Auto-fechar após 10 segundos
    setTimeout(() => notification.close(), 10000);
    
    // Focar janela ao clicar
    notification.onclick = function() {
        window.focus();
        notification.close();
    };
}

function playNotificationSound() {
    try {
        const audio = new Audio('data:audio/wav;base64,UklGRnoGAABXQVZFZm10IBAAAAABAAEAQB8AAEAfAAABAAgAZGF0YQoGAACBhYqFbF1fdJivrJBhNjVgodDbq2EcBj+a2/LDciUFLIHO8tiJNwgZaLvt559NEAxQp+PwtmMcBjiR1/LMeSwFJHfH8N2QQAoUXrTp66hVFApGn+DyvmwhBjSL0vTPfC0GI3S68tycRQsRW67k7qZSEwlBm+HyvWwjBzKHz/TQfC8FI3K18tucQgwOWKrh7qhYFgpGnt/zu3AfBjGByvXTfzYHInCv8NygQQ0NXKfg6KlXEwg9nNrwwXQkBjJ+xvPWgTkHIG+q7NypPw0OXKXd5q1aFgo8l9XuwHgmBTJ8wvLZhj0HIG2m69yrPQ0OXKHh5q1bFQo7ldTvwHkoBDJ7wPLaiD4GIG2k6t2uOwwPXJ7h5KxbFgo6k9LvwHsoBTF4vvHajD4GH2yk6d6vPAwOW53g46xdGAg5kdDuvn0qBDJ3vO/Yjj4GHWmh5+CwOwwOWprf4qtdGAg4j8/tvoErBzF1uu7UkD0FHWec5N+xOgwOWpne4qpaGAg4js3svoIuBzByue3Skj4EHGWa49+yOQwOWZfb4alYFgg4jMrrvoQyBi9vt+zRkz4EG2OY4d+zOQwOWZXY36lXFgg4iszqvoU0Bi9us+rQlD4EG2GW3d6zOQwOWJPX3qlVFgg4iMrpvoY1BS9tse/PlT4EGl+U292zOQwNWJDW3qhUFgg3h8jovoY3Bi9tsO/Olz4EGl6R2+K0OgwNV47V3KdSFgg3hsfovoY4Bi5ssO/OmD4DGl2P2+K0OgwNV43U3KZRFgc2hcXnvoY5Bi5rsO/OmT4DGVyN2+O1OwwMVozT26ZQFgc2g8Tmvoc6BS5qsO/NmT4DGVuL2+O1OwwMVYvR26VPFgc1gsPmvog6BS5psO/Mmj4DGVqJ2+O1OwwMVYvR2qVQFQc1gcHmu4g6BS5psO/Mmj4DGVmH2+O1OwwMVYvR2qVQFQc1gL/mu4g6BS5psO/Mmj4DGVmH2+O1OwwMVYvR2qVQFQc1gL/mu4g6BS5psO/Mmj4DGVmH2+O1OwwMVYvR2qVQFQc1gL/mu4g6BS5psO/Mmj4D'); 
        audio.volume = 0.3;
        audio.play().catch(e => console.log('Som não pode ser tocado:', e));
    } catch (e) {
        console.log('Erro ao tocar som:', e);
    }
}