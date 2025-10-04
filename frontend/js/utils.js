console.log('🛠️ utils.js carregando...');

/**
 * Formata um objeto Date para o formato de data local usando i18n
 */
function formatDate(date) {
    if (window.i18n && window.i18n.currentLocale) {
        return window.i18n.formatDate(date);
    }
    return date.toLocaleDateString('pt-BR');
}

/**
 * Formata um objeto Date para o formato de hora local usando i18n
 */
function formatTime(date) {
    if (window.i18n && window.i18n.currentLocale) {
        return window.i18n.formatTime(date);
    }
    return date.toLocaleTimeString('pt-BR', {
        hour: '2-digit',
        minute: '2-digit'
    });
}

/**
 * Converte segundos em uma string de duração (ex: "1h 30m" ou "45m 10s")
 */
function formatDuration(seconds) {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;

    if (hours > 0) {
        return `${hours}h ${minutes}m`;
    }
    return `${minutes}m ${secs}s`;
}

/**
 * Traduz o tipo de atividade usando i18n
 */
function translateActivityType(type) {
    if (window.i18n && window.i18n.currentLocale) {
        return window.i18n.translateActivityType(type);
    }
    
    // Fallback para português
    const translations = {
        'Ride': 'Ciclismo',
        'Run': 'Corrida',
        'Hike': 'Caminhada',
        'Walk': 'Caminhada',
        'Swimming': 'Natação',
        'Workout': 'Treino',
        'WeightTraining': 'Musculação',
        'VirtualRide': 'Ciclismo Virtual',
        'VirtualRun': 'Corrida Virtual',
        'EBikeRide': 'E-Bike',
    };
    return translations[type] || type;
}

/**
 * Retorna o ícone apropriado para cada tipo de atividade
 */
function getActivityIcon(type) {
    const icons = {
        // Ciclismo
        'Ride': '🚴',
        'VirtualRide': '🚴',
        'EBikeRide': '⚡🚴',
        'Handcycle': '🦽',
        'Velomobile': '🚴',
        
        // Corrida
        'Run': '🏃',
        'VirtualRun': '🏃',
        'TrailRun': '🏃‍♂️',
        
        // Caminhada
        'Walk': '🚶',
        'Hike': '🥾',
        
        // Natação
        'Swim': '🏊',
        'Swimming': '🏊',
        
        // Academia
        'WeightTraining': '🏋️',
        'Workout': '💪',
        'CrossFit': '🏋️‍♂️',
        
        // Esportes de inverno
        'Ski': '⛷️',
        'AlpineSki': '⛷️',
        'BackcountrySki': '⛷️',
        'NordicSki': '⛷️',
        'Snowboard': '🏂',
        'Snowshoe': '❄️',
        'IceSkate': '⛸️',
        
        // Esportes aquáticos
        'Rowing': '🚣',
        'Kayaking': '🛶',
        'Canoeing': '🛶',
        'StandUpPaddling': '🏄',
        'Surfing': '🏄',
        'Kitesurf': '🪁',
        'Windsurf': '🏄',
        'Sail': '⛵',
        
        // Escalada
        'RockClimbing': '🧗',
        'Climbing': '🧗',
        
        // Ioga e alongamento
        'Yoga': '🧘',
        'Pilates': '🤸',
        
        // Outros esportes
        'Golf': '⛳',
        'Soccer': '⚽',
        'Basketball': '🏀',
        'Tennis': '🎾',
        'Badminton': '🏸',
        'TableTennis': '🏓',
        'Squash': '🎾',
        'Volleyball': '🏐',
        'Cricket': '🏏',
        'Hockey': '🏒',
        'Rugby': '🏈',
        'Football': '🏈',
        'MartialArts': '🥋',
        'Boxing': '🥊',
        
        // Patinação
        'InlineSkate': '🛼',
        'RollerSki': '🛼',
        'Skateboard': '🛹',
        
        // Atividades motorizadas
        'EMountainBikeRide': '⚡🚵',
        'Elliptical': '🏃‍♀️',
        'StairStepper': '🪜',
        
        // Atividades de cadeira de rodas
        'WheelchairRun': '🦽',
        'WheelchairWalk': '🦽',
        
        // Default
        'default': '🏃‍♂️'
    };
    
    return icons[type] || icons['default'];
}