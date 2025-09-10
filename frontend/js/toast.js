// frontend/js/toast.js - Sistema de notificações moderno

class ToastManager {
    constructor() {
        this.container = this.createContainer();
        this.toasts = new Map();
        this.autoCloseDelay = 5000; // 5 segundos
    }

    createContainer() {
        let container = document.getElementById('toastContainer');
        if (!container) {
            container = document.createElement('div');
            container.id = 'toastContainer';
            container.className = 'toast-container';
            document.body.appendChild(container);
        }
        return container;
    }

    show(message, type = 'info', options = {}) {
        const toastId = Date.now() + Math.random();
        const toast = this.createToast(message, type, options, toastId);
        
        this.container.appendChild(toast);
        this.toasts.set(toastId, toast);

        // Animação de entrada
        requestAnimationFrame(() => {
            toast.style.transform = 'translateX(0)';
            toast.style.opacity = '1';
        });

        // Auto-close
        if (!options.persistent) {
            setTimeout(() => {
                this.remove(toastId);
            }, options.duration || this.autoCloseDelay);
        }

        return toastId;
    }

    createToast(message, type, options, toastId) {
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.style.transform = 'translateX(100%)';
        toast.style.opacity = '0';
        toast.style.transition = 'all 0.3s ease';

        const icon = this.getIcon(type);
        const closeButton = options.closable !== false ? this.createCloseButton(toastId) : '';

        toast.innerHTML = `
            <div style="display: flex; align-items: flex-start; gap: 0.75rem;">
                <div style="flex-shrink: 0; margin-top: 0.125rem;">
                    ${icon}
                </div>
                <div style="flex: 1; min-width: 0;">
                    <div style="font-weight: 500; margin-bottom: 0.25rem; color: var(--text-primary);">
                        ${options.title || this.getDefaultTitle(type)}
                    </div>
                    <div style="font-size: 0.9rem; color: var(--text-secondary); line-height: 1.4;">
                        ${message}
                    </div>
                    ${options.actions ? this.createActions(options.actions, toastId) : ''}
                </div>
                ${closeButton}
            </div>
            ${options.progress ? this.createProgressBar() : ''}
        `;

        return toast;
    }

    getIcon(type) {
        const icons = {
            success: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" style="color: var(--accent-green);">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
            </svg>`,
            error: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" style="color: var(--accent-red);">
                <path d="M12 2C6.47 2 2 6.47 2 12s4.47 10 10 10 10-4.47 10-10S17.53 2 12 2zm5 13.59L15.59 17 12 13.41 8.41 17 7 15.59 10.59 12 7 8.41 8.41 7 12 10.59 15.59 7 17 8.41 13.41 12 17 15.59z"/>
            </svg>`,
            warning: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" style="color: var(--accent-orange);">
                <path d="M1 21h22L12 2 1 21zm12-3h-2v-2h2v2zm0-4h-2v-4h2v4z"/>
            </svg>`,
            info: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" style="color: var(--accent-blue);">
                <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/>
            </svg>`,
            loading: `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor" style="color: var(--accent-blue); animation: spin 1s linear infinite;">
                <path d="M12 4V2A10 10 0 0 0 2 12h2a8 8 0 0 1 8-8z"/>
            </svg>`
        };
        return icons[type] || icons.info;
    }

    getDefaultTitle(type) {
        const titles = {
            success: 'Sucesso',
            error: 'Erro',
            warning: 'Atenção',
            info: 'Informação',
            loading: 'Processando'
        };
        return titles[type] || 'Notificação';
    }

    createCloseButton(toastId) {
        return `
            <button onclick="toastManager.remove('${toastId}')" 
                    style="background: none; border: none; color: var(--text-secondary); cursor: pointer; padding: 0.25rem; margin: -0.25rem -0.25rem -0.25rem 0;">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                    <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
                </svg>
            </button>
        `;
    }

    createActions(actions, toastId) {
        return `
            <div style="margin-top: 0.75rem; display: flex; gap: 0.5rem;">
                ${actions.map(action => `
                    <button onclick="${action.handler}; toastManager.remove('${toastId}')" 
                            class="btn btn-${action.type || 'secondary'}" 
                            style="padding: 0.5rem 1rem; font-size: 0.8rem;">
                        ${action.text}
                    </button>
                `).join('')}
            </div>
        `;
    }

    createProgressBar() {
        return `
            <div style="margin-top: 0.75rem;">
                <div style="background: var(--bg-tertiary); height: 4px; border-radius: 2px; overflow: hidden;">
                    <div class="toast-progress" style="background: var(--accent-blue); height: 100%; width: 0%; transition: width 0.3s ease;"></div>
                </div>
            </div>
        `;
    }

    updateProgress(toastId, percentage) {
        const toast = this.toasts.get(toastId);
        if (toast) {
            const progressBar = toast.querySelector('.toast-progress');
            if (progressBar) {
                progressBar.style.width = `${percentage}%`;
            }
        }
    }

    remove(toastId) {
        const toast = this.toasts.get(toastId);
        if (toast) {
            toast.style.transform = 'translateX(100%)';
            toast.style.opacity = '0';
            
            setTimeout(() => {
                if (toast.parentNode) {
                    toast.parentNode.removeChild(toast);
                }
                this.toasts.delete(toastId);
            }, 300);
        }
    }

    clear() {
        this.toasts.forEach((toast, id) => {
            this.remove(id);
        });
    }

    // Métodos de conveniência
    success(message, options = {}) {
        return this.show(message, 'success', options);
    }

    error(message, options = {}) {
        return this.show(message, 'error', options);
    }

    warning(message, options = {}) {
        return this.show(message, 'warning', options);
    }

    info(message, options = {}) {
        return this.show(message, 'info', options);
    }

    loading(message, options = {}) {
        return this.show(message, 'loading', { persistent: true, closable: false, ...options });
    }

    // Métodos especiais para casos de uso específicos
    progressToast(message, options = {}) {
        return this.show(message, 'info', { 
            progress: true, 
            persistent: true, 
            closable: false,
            ...options 
        });
    }

    confirmToast(message, onConfirm, onCancel = null) {
        return this.show(message, 'warning', {
            title: 'Confirmação necessária',
            persistent: true,
            actions: [
                {
                    text: 'Confirmar',
                    type: 'primary',
                    handler: `(${onConfirm.toString()})()`
                },
                {
                    text: 'Cancelar',
                    type: 'secondary',
                    handler: onCancel ? `(${onCancel.toString()})()` : 'void(0)'
                }
            ]
        });
    }

    networkError(error, retryCallback = null) {
        const actions = retryCallback ? [
            {
                text: 'Tentar Novamente',
                type: 'primary',
                handler: `(${retryCallback.toString()})()`
            }
        ] : [];

        return this.show(
            `Erro de conexão: ${error.message || 'Verifique sua internet'}`,
            'error',
            {
                title: 'Problema de Conectividade',
                duration: 8000,
                actions
            }
        );
    }

    activityToast(activity, onSelect) {
        return this.show(
            `${activity.name} - ${(activity.distance / 1000).toFixed(1)}km`,
            'info',
            {
                title: 'Nova atividade encontrada',
                duration: 10000,
                actions: [
                    {
                        text: 'Selecionar',
                        type: 'primary',
                        handler: `(${onSelect.toString()})(${activity.id})`
                    }
                ]
            }
        );
    }
}

// Instância global
const toastManager = new ToastManager();

// Integração com o sistema existente
function showMessage(container, message, type) {
    // Mantém compatibilidade com o sistema antigo
    if (container && container.innerHTML !== undefined) {
        container.innerHTML = message ? `<div class="${type}">${message}</div>` : '';
    }
    
    // Adiciona toast moderno
    if (message && message.trim()) {
        const cleanMessage = message.replace(/<[^>]*>/g, ''); // Remove HTML tags
        toastManager.show(cleanMessage, type);
    }
}

// Sobrescreve função global para usar toasts
window.showGlobalMessage = function(message, type = 'info', options = {}) {
    return toastManager.show(message, type, options);
};

// Intercepta erros não tratados
window.addEventListener('error', (event) => {
    toastManager.error(
        'Erro inesperado na aplicação. Recarregue a página se persistir.',
        {
            title: 'Erro Interno',
            duration: 10000
        }
    );
});

// Intercepta promessas rejeitadas
window.addEventListener('unhandledrejection', (event) => {
    console.error('Unhandled promise rejection:', event.reason);
    toastManager.error(
        'Operação falhou inesperadamente. Tente novamente.',
        {
            title: 'Operação Falhada',
            duration: 8000
        }
    );
});

// Exporta para uso global
window.toastManager = toastManager;