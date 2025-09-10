console.log('📦 globals.js carregando...');

// --- Variáveis de Estado da Aplicação ---
let selectedActivity = null;
let selectedVideoPath = "";
let manualSyncTime = "";
let isAuthenticated = false;
let isCheckingAuth = false;

// --- Variáveis de Paginação ---
let allActivities = []; // Armazena todas as atividades carregadas
let currentPage = 1;
let isLoadingMore = false;
let hasMorePages = true;
let showOnlyGPS = true; // Filtro padrão

// --- Variáveis de Mapa (Leaflet) ---
let activityMap = null;
let videoStartMarker = null;
let activityPolyline = null;
let currentMarkerDensity = 'medium';
let currentGPSMarkersGroup = null;

// --- Elementos DOM (serão inicializados em app.js) ---
let authBtn, statusDiv, activitiesSection, activitiesGrid;
let activityDetail, activityInfo, mapContainer, videoSection;
let selectVideoBtn, videoInfo, processBtn, progress;
let progressBar, progressText, result;
let loadMoreBtn, filterGPSCheckbox;
let totalActivitiesSpan, gpsActivitiesSpan;
