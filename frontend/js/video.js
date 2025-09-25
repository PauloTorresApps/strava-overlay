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

        // NOVO: Mostra o controle de posição do overlay
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
        if (progress) progress.classList.remove('hidden');
        
        showMessage(result, '', ''); // Limpa mensagens anteriores
        simulateProgress();

        // NOVO: Pega a posição selecionada do overlay
        const overlayPosition = window.overlayPosition ? window.overlayPosition.getPosition() : 'bottom-left';
        console.log(`📍 Processando vídeo com overlay na posição: ${overlayPosition}`);

        // MODIFICADO: Passa a posição do overlay como 4º parâmetro
        const outputPath = await window.go.main.App.ProcessVideoOverlay(
            selectedActivity.id, 
            selectedVideoPath, 
            manualSyncTime,
            overlayPosition // NOVO parâmetro
        );
        
        updateProgress(100);
        showMessage(result, `Vídeo processado com sucesso!<br><strong>Local:</strong> ${outputPath}`, 'success');
        
        // NOVO: Esconde o controle após processamento bem-sucedido
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