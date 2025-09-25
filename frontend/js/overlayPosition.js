// frontend/js/overlayPosition.js
console.log('📍 overlayPosition.js carregando...');

// Variável global para armazenar a posição selecionada
let selectedOverlayPosition = 'bottom-left'; // Padrão atual

/**
 * Inicializa o controle de posição do overlay
 */
function initOverlayPositionControl() {
    const positionButtons = document.querySelectorAll('.position-btn');
    
    positionButtons.forEach(button => {
        button.addEventListener('click', handlePositionSelection);
    });
    
    console.log('✅ Controle de posição do overlay inicializado');
}

/**
 * Manipula a seleção de posição
 */
function handlePositionSelection(event) {
    const button = event.currentTarget;
    const position = button.dataset.position;
    
    // Remove seleção anterior
    document.querySelectorAll('.position-btn').forEach(btn => {
        btn.classList.remove('selected');
    });
    
    // Adiciona seleção ao botão clicado
    button.classList.add('selected');
    
    // Atualiza a posição global
    selectedOverlayPosition = position;
    
    console.log(`📍 Posição do overlay selecionada: ${position}`);
    
    // Feedback visual adicional
    showMessage(result, `Overlay será posicionado no ${getPositionLabel(position)}`, 'info');
}

/**
 * Retorna o label amigável para a posição
 */
function getPositionLabel(position) {
    const labels = {
        'top-left': 'canto superior esquerdo',
        'top-right': 'canto superior direito',
        'bottom-left': 'canto inferior esquerdo',
        'bottom-right': 'canto inferior direito'
    };
    return labels[position] || position;
}

/**
 * Mostra o controle de posição quando um vídeo é selecionado
 */
function showOverlayPositionControl() {
    const control = document.getElementById('overlayPositionControl');
    if (control) {
        control.classList.remove('hidden');
        console.log('📍 Controle de posição exibido');
    }
}

/**
 * Esconde o controle de posição
 */
function hideOverlayPositionControl() {
    const control = document.getElementById('overlayPositionControl');
    if (control) {
        control.classList.add('hidden');
    }
}

/**
 * Retorna a posição selecionada para o processamento
 */
function getSelectedOverlayPosition() {
    return selectedOverlayPosition;
}

// Adiciona ao escopo global para acesso em outros módulos
window.overlayPosition = {
    init: initOverlayPositionControl,
    show: showOverlayPositionControl,
    hide: hideOverlayPositionControl,
    getPosition: getSelectedOverlayPosition,
    setPosition: (position) => {
        selectedOverlayPosition = position;
        // Atualiza UI
        document.querySelectorAll('.position-btn').forEach(btn => {
            btn.classList.toggle('selected', btn.dataset.position === position);
        });
    }
};

console.log('✅ overlayPosition.js carregado');