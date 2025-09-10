console.log('📹 video.js carregando...');

/**
 * Abre o seletor de arquivos de vídeo e busca o ponto de início automático.
 */
async function selectVideo() {
    try {
        const path = await window.go.main.App.SelectVideoFile();
        if (!path) return;

        selectedVideoPath = path;
        manualSyncTime = ""; // Reseta ao selecionar novo vídeo

        const fileName = path.split(/[\\/]/).pop();
        if (videoInfo) {
            videoInfo.innerHTML = `<h4>Vídeo Selecionado:</h4><p>${fileName}</p>`;
        }
        if (processBtn) processBtn.disabled = false;

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
        if (progress) progress.classList.remove('hidden');
        
        showMessage(result, '', ''); // Limpa mensagens anteriores
        simulateProgress();

        // Passa o tempo manual (pode ser uma string vazia) para o backend
        const outputPath = await window.go.main.App.ProcessVideoOverlay(selectedActivity.id, selectedVideoPath, manualSyncTime);
        
        updateProgress(100);
        showMessage(result, `Vídeo processado com sucesso!<br><strong>Local:</strong> ${outputPath}`, 'success');
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
