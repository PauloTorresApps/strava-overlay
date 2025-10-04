// frontend/js/i18n.js - Sistema de Internacionalização
console.log('🌍 i18n.js carregando...');

class I18nService {
    constructor() {
        this.currentLocale = null;
        this.translations = {};
        this.availableLocales = [
            { code: 'pt-BR', name: 'Português (BR)', flag: '🇧🇷' },
            { code: 'en-US', name: 'English (US)', flag: '🇺🇸' },
            { code: 'es-ES', name: 'Español', flag: '🇪🇸' },
            { code: 'zh-CN', name: '中文', flag: '🇨🇳' }
        ];
        this.defaultLocale = 'pt-BR';
    }

    /**
     * Inicializa o sistema i18n
     */
    async initialize() {
        console.log('🌍 Inicializando sistema i18n...');
        
        // Detecta idioma do sistema
        const systemLocale = this.detectSystemLocale();
        console.log(`🔍 Idioma do sistema detectado: ${systemLocale}`);
        
        // Carrega idioma salvo ou usa o detectado
        const savedLocale = localStorage.getItem('app_locale');
        const localeToUse = savedLocale || systemLocale || this.defaultLocale;
        
        console.log(`📌 Usando idioma: ${localeToUse}`);
        
        // Carrega traduções
        await this.loadLocale(localeToUse);
        
        // Cria seletor de idioma no header
        this.createLanguageSelector();
        
        console.log('✅ Sistema i18n inicializado');
    }

    /**
     * Detecta o idioma do sistema operacional
     */
    detectSystemLocale() {
        const browserLang = navigator.language || navigator.userLanguage;
        console.log(`🌐 Idioma do navegador: ${browserLang}`);
        
        // Mapeia idiomas do navegador para nossos locales
        const langMap = {
            'pt': 'pt-BR',
            'pt-BR': 'pt-BR',
            'pt-PT': 'pt-BR',
            'en': 'en-US',
            'en-US': 'en-US',
            'en-GB': 'en-US',
            'es': 'es-ES',
            'es-ES': 'es-ES',
            'es-MX': 'es-ES',
            'zh': 'zh-CN',
            'zh-CN': 'zh-CN',
            'zh-TW': 'zh-CN'
        };
        
        // Tenta correspondência exata primeiro
        if (langMap[browserLang]) {
            return langMap[browserLang];
        }
        
        // Tenta correspondência do código de idioma
        const langCode = browserLang.split('-')[0];
        return langMap[langCode] || this.defaultLocale;
    }

    /**
     * Carrega arquivo de tradução para um locale
     */
    async loadLocale(locale) {
        try {
            const response = await fetch(`locales/${locale}.json`);
            if (!response.ok) {
                throw new Error(`Falha ao carregar ${locale}`);
            }
            
            this.translations = await response.json();
            this.currentLocale = locale;
            
            console.log(`✅ Traduções carregadas para ${locale}`);
            
            // Atualiza a interface
            this.updateUI();
            
            return true;
        } catch (error) {
            console.error(`❌ Erro ao carregar locale ${locale}:`, error);
            
            // Fallback para idioma padrão
            if (locale !== this.defaultLocale) {
                console.log(`🔄 Tentando fallback para ${this.defaultLocale}`);
                return this.loadLocale(this.defaultLocale);
            }
            
            return false;
        }
    }

    /**
     * Obtém tradução por chave (suporta caminhos aninhados)
     */
    t(key, fallback = key) {
        const keys = key.split('.');
        let value = this.translations;
        
        for (const k of keys) {
            if (value && typeof value === 'object' && k in value) {
                value = value[k];
            } else {
                console.warn(`⚠️ Tradução não encontrada: ${key}`);
                return fallback;
            }
        }
        
        return value || fallback;
    }

    /**
     * Troca o idioma
     */
    async changeLocale(locale) {
        if (locale === this.currentLocale) return;
        
        console.log(`🔄 Mudando idioma para ${locale}...`);
        
        const success = await this.loadLocale(locale);
        if (success) {
            localStorage.setItem('app_locale', locale);
            console.log(`✅ Idioma alterado para ${locale}`);
            
            // Dispara evento customizado para outros componentes reagirem
            window.dispatchEvent(new CustomEvent('localeChanged', { 
                detail: { locale } 
            }));
        }
    }

    /**
     * Cria o seletor de idioma no header
     */
    createLanguageSelector() {
        const header = document.querySelector('.header-right');
        if (!header) {
            console.warn('⚠️ Header não encontrado');
            return;
        }

        // Remove seletor existente se houver
        const existing = document.getElementById('languageSelector');
        if (existing) existing.remove();

        const container = document.createElement('div');
        container.id = 'languageSelector';
        container.style.cssText = `
            position: relative;
            margin-left: 15px;
        `;

        const button = document.createElement('button');
        button.className = 'language-selector-btn';
        button.innerHTML = this.getCurrentFlag();
        button.style.cssText = `
            background: var(--container-bg);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            padding: 8px 12px;
            cursor: pointer;
            font-size: 1.2rem;
            transition: all 0.2s;
            display: flex;
            align-items: center;
            gap: 8px;
        `;
        button.title = 'Idioma / Language / Idioma / 语言';

        const dropdown = document.createElement('div');
        dropdown.className = 'language-dropdown';
        dropdown.style.cssText = `
            display: none;
            position: absolute;
            top: calc(100% + 5px);
            right: 0;
            background: var(--container-bg);
            border: 1px solid var(--border-color);
            border-radius: 6px;
            min-width: 200px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            z-index: 1000;
        `;

        this.availableLocales.forEach(locale => {
            const item = document.createElement('div');
            item.className = 'language-item';
            item.innerHTML = `
                <span style="margin-right: 8px;">${locale.flag}</span>
                <span>${locale.name}</span>
                ${locale.code === this.currentLocale ? '<span style="margin-left: auto; color: var(--accent-color);">✓</span>' : ''}
            `;
            item.style.cssText = `
                padding: 10px 15px;
                cursor: pointer;
                display: flex;
                align-items: center;
                transition: background 0.2s;
                color: var(--primary-text);
            `;
            item.addEventListener('mouseenter', () => {
                item.style.background = 'var(--accent-color)';
                item.style.color = 'white';
            });
            item.addEventListener('mouseleave', () => {
                item.style.background = 'transparent';
                item.style.color = 'var(--primary-text)';
            });
            item.addEventListener('click', () => {
                this.changeLocale(locale.code);
                dropdown.style.display = 'none';
            });
            dropdown.appendChild(item);
        });

        button.addEventListener('click', (e) => {
            e.stopPropagation();
            dropdown.style.display = dropdown.style.display === 'none' ? 'block' : 'none';
        });

        // Fecha dropdown ao clicar fora
        document.addEventListener('click', () => {
            dropdown.style.display = 'none';
        });

        container.appendChild(button);
        container.appendChild(dropdown);
        header.appendChild(container);

        console.log('✅ Seletor de idioma criado');
    }

    /**
     * Retorna a bandeira do idioma atual
     */
    getCurrentFlag() {
        const locale = this.availableLocales.find(l => l.code === this.currentLocale);
        return locale ? locale.flag : '🌍';
    }

    /**
     * Atualiza todos os elementos com atributo data-i18n
     */
    updateUI() {
        console.log('🔄 Atualizando interface com traduções...');
        
        // Atualiza elementos com data-i18n
        document.querySelectorAll('[data-i18n]').forEach(element => {
            const key = element.getAttribute('data-i18n');
            const translation = this.t(key);
            
            if (element.tagName === 'INPUT' && element.hasAttribute('placeholder')) {
                element.placeholder = translation;
            } else {
                element.textContent = translation;
            }
        });

        // Atualiza título da página
        document.title = this.t('app.title', 'Strava Video Overlay');
        
        // Atualiza seletor de idioma se existir
        const langBtn = document.querySelector('.language-selector-btn');
        if (langBtn) {
            langBtn.innerHTML = this.getCurrentFlag();
        }

        console.log('✅ Interface atualizada');
    }

    /**
     * Formata data de acordo com o locale
     */
    formatDate(date) {
        const localeMap = {
            'pt-BR': 'pt-BR',
            'en-US': 'en-US',
            'es-ES': 'es-ES',
            'zh-CN': 'zh-CN'
        };
        
        return date.toLocaleDateString(localeMap[this.currentLocale] || 'pt-BR');
    }

    /**
     * Formata hora de acordo com o locale
     */
    formatTime(date) {
        const localeMap = {
            'pt-BR': 'pt-BR',
            'en-US': 'en-US',
            'es-ES': 'es-ES',
            'zh-CN': 'zh-CN'
        };
        
        return date.toLocaleTimeString(localeMap[this.currentLocale] || 'pt-BR', {
            hour: '2-digit',
            minute: '2-digit'
        });
    }

    /**
     * Traduz tipo de atividade
     */
    translateActivityType(type) {
        return this.t(`activities.types.${type}`, type);
    }
}

// Instância global
window.i18n = new I18nService();

// Helper para tradução rápida
window.t = (key, fallback) => window.i18n.t(key, fallback);

console.log('✅ i18n.js carregado');