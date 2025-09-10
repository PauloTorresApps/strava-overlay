console.log('🔒 auth.js carregando...');

/**
 * Inicia a verificação de autenticação ao carregar a página.
 */
async function checkAuthenticationOnStartup() {
    if (isCheckingAuth) {
        console.log('⏳ Já verificando autenticação...');
        return;
    }

    console.log('🔍 Iniciando verificação de autenticação...');
    isCheckingAuth = true;

    try {
        showMessage(statusDiv, '🔍 Verificando credenciais salvas...', 'info');
        if (authBtn) {
            authBtn.disabled = true;
            authBtn.textContent = 'Verificando...';
        }

        const response = await window.go.main.App.CheckAuthenticationStatus();
        console.log('📡 Resposta do backend:', response);

        if (response?.is_authenticated) {
            handleAuthSuccess(response);
        } else {
            handleAuthFailure(response?.error);
        }

    } catch (error) {
        console.error('❌ Erro na verificação:', error);
        handleAuthError(error);
    } finally {
        isCheckingAuth = false;
    }
}

/**
 * Lida com o sucesso da autenticação.
 * @param {object} response - A resposta do backend.
 */
function handleAuthSuccess(response) {
    console.log('✅ Autenticação bem-sucedida');
    isAuthenticated = true;

    showMessage(statusDiv, `✅ ${response.message}`, 'success');
    if (authBtn) authBtn.style.display = 'none';
    if (activitiesSection) activitiesSection.classList.remove('hidden');

    loadActivitiesPage(1); // Carrega a primeira página de atividades
}

/**
 * Lida com a falha na autenticação (necessário autenticar).
 * @param {string} error - A mensagem de erro opcional.
 */
function handleAuthFailure(error) {
    console.log('❌ Autenticação necessária:', error);
    isAuthenticated = false;

    showMessage(statusDiv, 'Clique no botão para conectar ao Strava', 'info');
    if (authBtn) {
        authBtn.disabled = false;
        authBtn.textContent = 'Autenticar com Strava';
        authBtn.style.display = 'block';
    }
}

/**
 * Lida com erros durante o processo de autenticação.
 * @param {Error} error - O objeto de erro.
 */
function handleAuthError(error) {
    console.error('❌ Erro na verificação:', error);
    isAuthenticated = false;

    showMessage(statusDiv, 'Erro na verificação. Clique para autenticar manualmente.', 'error');
    if (authBtn) {
        authBtn.disabled = false;
        authBtn.textContent = 'Autenticar com Strava';
        authBtn.style.display = 'block';
    }
}

/**
 * Inicia o fluxo de autenticação manual via backend.
 */
async function authenticateStrava() {
    if (isCheckingAuth) return;

    try {
        if (authBtn) {
            authBtn.disabled = true;
            authBtn.textContent = 'Conectando...';
        }
        showMessage(statusDiv, 'Autenticando...', 'info');

        await window.go.main.App.AuthenticateStrava();

        isAuthenticated = true;
        showMessage(statusDiv, 'Conectado com sucesso!', 'success');
        if (authBtn) authBtn.style.display = 'none';
        if (activitiesSection) activitiesSection.classList.remove('hidden');

        loadActivitiesPage(1);

    } catch (error) {
        console.error('❌ Erro na autenticação manual:', error);
        isAuthenticated = false;
        showMessage(statusDiv, `Erro: ${error}`, 'error');
        if (authBtn) {
            authBtn.disabled = false;
            authBtn.textContent = 'Autenticar com Strava';
        }
    }
}
