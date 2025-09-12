// frontend/js/config.js - Serviço de configuração para acessar variáveis de ambiente
console.log('⚙️ config.js carregando...');

/**
 * Classe para gerenciar configurações do frontend vindas do backend
 */
class ConfigService {
    constructor() {
        this.config = null;
        this.apiKeys = null;
        this.mapProviders = null;
        this.loading = false;
        this.initialized = false;
    }

    /**
     * Inicializa o serviço carregando configurações do backend
     */
    async initialize() {
        if (this.loading) {
            console.log('⏳ ConfigService já está carregando...');
            return;
        }

        if (this.initialized) {
            console.log('✅ ConfigService já inicializado');
            return;
        }

        this.loading = true;

        try {
            console.log('🔄 Carregando configurações do backend...');
            
            // Carrega configurações gerais
            this.config = await window.go.main.App.GetFrontendConfig();
            console.log('✅ Configurações gerais carregadas:', this.config);

            // Carrega chaves de API seguras
            this.apiKeys = await window.go.main.App.GetSecureAPIKeys();
            console.log('🔑 Chaves de API carregadas:', Object.keys(this.apiKeys));

            // Carrega configuração de provedores de mapa
            this.mapProviders = await window.go.main.App.GetMapProviderConfig();
            console.log('🗺️ Provedores de mapa carregados:', Object.keys(this.mapProviders));

            this.initialized = true;
            console.log('🎉 ConfigService inicializado com sucesso!');

        } catch (error) {
            console.error('❌ Erro ao carregar configurações:', error);
            // Define configurações padrão como fallback
            this.setDefaultConfig();
        } finally {
            this.loading = false;
        }
    }

    /**
     * Define configurações padrão caso o carregamento falhe
     */
    setDefaultConfig() {
        console.log('🔄 Usando configurações padrão...');
        
        this.config = {
            app_version: '1.0.0',
            environment: 'development',
            default_map_provider: 'osm',
            available_providers: ['osm', 'osmDark', 'satellite', 'terrain', 'cartodb_dark']
        };

        this.apiKeys = {};
        
        this.mapProviders = {
            openstreetmap: { enabled: true }
        };
    }

    /**
     * Retorna uma chave de API específica
     */
    getAPIKey(provider) {
        if (!this.initialized) {
            console.warn('⚠️ ConfigService não inicializado ainda');
            return null;
        }

        const key = this.apiKeys[provider];
        if (!key) {
            console.log(`ℹ️ Chave de API não encontrada para: ${provider}`);
            return null;
        }

        return key;
    }

    /**
     * Retorna configuração de um provedor de mapa
     */
    getMapProviderConfig(provider) {
        if (!this.initialized) {
            console.warn('⚠️ ConfigService não inicializado ainda');
            return null;
        }

        return this.mapProviders[provider] || null;
    }

    /**
     * Verifica se um provedor está disponível
     */
    isProviderAvailable(provider) {
        const config = this.getMapProviderConfig(provider);
        return config && config.enabled === true;
    }

    /**
     * Retorna lista de provedores disponíveis
     */
    getAvailableProviders() {
        if (!this.initialized) {
            return ['osm']; // Fallback básico
        }

        return this.config.available_providers || ['osm'];
    }

    /**
     * Retorna configuração completa
     */
    getConfig() {
        return this.config;
    }

    /**
     * Retorna versão da aplicação
     */
    getAppVersion() {
        return this.config?.app_version || '1.0.0';
    }

    /**
     * Retorna ambiente (development, production)
     */
    getEnvironment() {
        return this.config?.environment || 'development';
    }

    /**
     * Constrói URL do Thunderforest com API key
     */
    getThunderforestURL(style, z, x, y) {
        const apiKey = this.getAPIKey('thunderforest');
        if (!apiKey) {
            console.warn('⚠️ Chave Thunderforest não disponível');
            return null;
        }

        return `https://tile.thunderforest.com/${style}/${z}/${x}/${y}.png?apikey=${apiKey}`;
    }

    /**
     * Constrói URL do Mapbox com token público
     */
    getMapboxURL(styleId, z, x, y) {
        const publicToken = this.getAPIKey('mapbox_public');
        if (!publicToken) {
            console.warn('⚠️ Token público Mapbox não disponível');
            return null;
        }

        return `https://api.mapbox.com/styles/v1/mapbox/${styleId}/tiles/${z}/${x}/${y}?access_token=${publicToken}`;
    }

    /**
     * Retorna provedores de mapa atualizados com chaves de API
     */
    getUpdatedMapProviders() {
        const providers = { ...MAP_PROVIDERS }; // Copia provedores base

        // Atualiza Thunderforest se disponível
        if (this.isProviderAvailable('thunderforest')) {
            const thunderforestKey = this.getAPIKey('thunderforest');
            
            providers.thunderforest_cycle = {
                name: 'Thunderforest Cycle',
                url: `https://tile.thunderforest.com/cycle/{z}/{x}/{y}.png?apikey=${thunderforestKey}`,
                attribution: '© Thunderforest © OpenStreetMap contributors',
                darkFilter: false
            };

            providers.thunderforest_outdoors = {
                name: 'Thunderforest Outdoors',
                url: `https://tile.thunderforest.com/outdoors/{z}/{x}/{y}.png?apikey=${thunderforestKey}`,
                attribution: '© Thunderforest © OpenStreetMap contributors',
                darkFilter: false
            };
        }

        // Atualiza Mapbox se disponível
        if (this.isProviderAvailable('mapbox')) {
            const mapboxToken = this.getAPIKey('mapbox_public');
            
            // Substitui URLs placeholder por URLs reais
            Object.keys(providers).forEach(key => {
                if (providers[key].provider === 'mapbox') {
                    providers[key].url = providers[key].url.replace('{accessToken}', mapboxToken);
                }
            });
        }

        return providers;
    }

    /**
     * Debug - Mostra todas as configurações
     */
    debugConfig() {
        console.group('🐛 ConfigService Debug');
        console.log('Inicializado:', this.initialized);
        console.log('Config:', this.config);
        console.log('API Keys:', this.apiKeys);
        console.log('Map Providers:', this.mapProviders);
        console.groupEnd();
    }
}

// Instância global do serviço de configuração
window.configService = new ConfigService();

/**
 * Função de conveniência para obter chaves de API
 */
window.getAPIKey = function(provider) {
    return window.configService.getAPIKey(provider);
};

/**
 * Função para inicializar configurações - deve ser chamada na inicialização da app
 */
window.initializeConfig = async function() {
    await window.configService.initialize();
    
    // Atualiza provedores de mapa com as novas chaves
    if (typeof MAP_PROVIDERS !== 'undefined') {
        const updatedProviders = window.configService.getUpdatedMapProviders();
        Object.assign(MAP_PROVIDERS, updatedProviders);
        console.log('🗺️ Provedores de mapa atualizados com chaves de API');
    }
};

console.log('✅ ConfigService carregado e pronto para uso');