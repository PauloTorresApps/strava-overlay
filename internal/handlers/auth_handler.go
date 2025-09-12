package handlers

import (
	"context"
	"fmt"
	"log"

	"strava-overlay/internal/auth"
	"strava-overlay/internal/strava"
)

// AuthStatus representa o status da autenticação
type AuthStatus struct {
	IsAuthenticated bool   `json:"is_authenticated"`
	Message         string `json:"message"`
	Error           string `json:"error,omitempty"`
}

// AuthHandler gerencia todas as operações relacionadas à autenticação
type AuthHandler struct {
	stravaAuth      *auth.StravaAuth
	setStravaClient func(*strava.Client) // Callback para definir o cliente no app principal
}

// NewAuthHandler cria um novo handler de autenticação
func NewAuthHandler(stravaAuth *auth.StravaAuth, setStravaClient func(*strava.Client)) *AuthHandler {
	return &AuthHandler{
		stravaAuth:      stravaAuth,
		setStravaClient: setStravaClient,
	}
}

// CheckAuthenticationStatus verifica automaticamente se há um token válido
func (h *AuthHandler) CheckAuthenticationStatus(ctx context.Context) AuthStatus {
	log.Printf("🔍 Verificando status de autenticação...")

	token, err := h.stravaAuth.GetValidToken(ctx)
	if err != nil {
		log.Printf("❌ Falha na verificação de token: %v", err)
		return AuthStatus{
			IsAuthenticated: false,
			Message:         "Autenticação necessária",
			Error:           err.Error(),
		}
	}

	// Se chegou até aqui, o token é válido
	client := strava.NewClient(token)
	h.setStravaClient(client)

	log.Printf("✅ Token válido encontrado - Cliente Strava inicializado")

	return AuthStatus{
		IsAuthenticated: true,
		Message:         "Conectado automaticamente ao Strava",
	}
}

// AuthenticateStrava handles Strava authentication (para autenticação manual)
func (h *AuthHandler) AuthenticateStrava(ctx context.Context) error {
	log.Printf("🔐 Iniciando autenticação manual do Strava...")

	token, err := h.stravaAuth.GetValidToken(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	client := strava.NewClient(token)
	h.setStravaClient(client)

	log.Printf("✅ Autenticação manual concluída com sucesso")
	return nil
}
