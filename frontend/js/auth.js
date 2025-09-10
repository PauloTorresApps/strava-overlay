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
        updateHeaderStatus('checking', 'Verificando conexão...');
        hideAuthButton();

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

    updateHeaderStatus('connected', 'Conectado ao Strava');
    hideAuthButton();
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

    updateHeaderStatus('error', 'Não conectado');
    showAuthButton();
}

/**
 * Lida com erros durante o processo de autenticação.
 * @param {Error} error - O objeto de erro.
 */
function handleAuthError(error) {
    console.error('❌ Erro na verificação:', error);
    isAuthenticated = false;

    updateHeaderStatus('error', 'Erro na conexão');
    showAuthButton();
}

/**
 * Inicia o fluxo de autenticação manual via backend.
 */
async function authenticateStrava() {
    if (isCheckingAuth) return;

    try {
        isCheckingAuth = true;
        updateHeaderStatus('checking', 'Conectando...');
        hideAuthButton();

        await window.go.main.App.AuthenticateStrava();

        isAuthenticated = true;
        updateHeaderStatus('connected', 'Conectado ao Strava');
        if (activitiesSection) activitiesSection.classList.remove('hidden');

        loadActivitiesPage(1);

    } catch (error) {
        console.error('❌ Erro na autenticação manual:', error);
        isAuthenticated = false;
        updateHeaderStatus('error', 'Falha na autenticação');
        showAuthButton();
    } finally {
        isCheckingAuth = false;
    }
}

// === FUNÇÕES PARA MANIPULAR O HEADER ===

/**
 * Atualiza o status no header.
 * @param {string} status - 'checking', 'connected', ou 'error'
 * @param {string} text - Texto a exibir
 */
function updateHeaderStatus(status, text) {
    const statusIndicator = document.getElementById('statusIndicator');
    const statusDot = document.getElementById('statusDot');
    const statusText = document.getElementById('statusText');

    if (statusIndicator && statusDot && statusText) {
        // Remove classes existentes
        statusIndicator.className = 'status-indicator';
        statusDot.className = 'status-dot';
        
        // Adiciona nova classe
        statusIndicator.classList.add(status);
        statusDot.classList.add(status);
        
        // Atualiza texto
        statusText.textContent = text;
    }
}

/**
 * Mostra o botão de autenticação.
 */
function showAuthButton() {
    const authBtn = document.getElementById('authBtn');
    if (authBtn) {
        authBtn.classList.add('show');
        authBtn.onclick = authenticateStrava;
    }
}

/**
 * Esconde o botão de autenticação.
 */
function hideAuthButton() {
    const authBtn = document.getElementById('authBtn');
    if (authBtn) {
        authBtn.classList.remove('show');
        authBtn.onclick = null;
    }
}