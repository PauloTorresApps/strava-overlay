console.log('🛠️ utils.js carregando...');

/**
 * Formata um objeto Date para o formato de data local (dd/mm/aaaa).
 * @param {Date} date - O objeto Date a ser formatado.
 * @returns {string} A data formatada.
 */
function formatDate(date) {
    return date.toLocaleDateString('pt-BR');
}

/**
 * Formata um objeto Date para o formato de hora local (hh:mm).
 * @param {Date} date - O objeto Date a ser formatado.
 * @returns {string} A hora formatada.
 */
function formatTime(date) {
    return date.toLocaleTimeString('pt-BR', {
        hour: '2-digit',
        minute: '2-digit'
    });
}

/**
 * Converte segundos em uma string de duração (ex: "1h 30m" ou "45m 10s").
 * @param {number} seconds - A duração total em segundos.
 * @returns {string} A duração formatada.
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
 * Traduz o tipo de atividade do inglês para o português.
 * @param {string} type - O tipo de atividade (ex: "Ride", "Run").
 * @returns {string} O tipo de atividade traduzido.
 */
function translateActivityType(type) {
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
        // Adicione outras traduções conforme necessário
    };
    return translations[type] || type;
}

/**
 * Retorna o ícone apropriado para cada tipo de atividade
 * @param {string} type - O tipo de atividade
 * @returns {string} O emoji/ícone correspondente
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