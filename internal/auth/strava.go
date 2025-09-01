package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

type StravaAuth struct {
	config     *oauth2.Config
	tokenFile  string
	httpServer *http.Server
}

type TokenData struct {
	Token     *oauth2.Token `json:"token"`
	ExpiresAt time.Time     `json:"expires_at"`
}

func NewStravaAuth(clientID, clientSecret string) *StravaAuth {
	homeDir, _ := os.UserHomeDir()
	tokenFile := filepath.Join(homeDir, ".strava-overlay", "token.json")

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://www.strava.com/oauth/authorize",
			TokenURL: "https://www.strava.com/oauth/token",
		},
		RedirectURL: "http://localhost:8080/callback",
		Scopes:      []string{"read,activity:read"},
	}

	return &StravaAuth{
		config:    config,
		tokenFile: tokenFile,
	}
}

func (sa *StravaAuth) GetValidToken(ctx context.Context) (*oauth2.Token, error) {
	if token, err := sa.loadToken(); err == nil {
		if token.Valid() {
			return token, nil
		}
		if refreshed, err := sa.config.TokenSource(ctx, token).Token(); err == nil {
			sa.saveToken(refreshed)
			return refreshed, nil
		}
	}
	return sa.authorizeUser(ctx)
}

func (sa *StravaAuth) authorizeUser(ctx context.Context) (*oauth2.Token, error) {
	state := sa.generateState()

	mux := http.NewServeMux()
	tokenChan := make(chan *oauth2.Token, 1)
	errChan := make(chan error, 1)

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state", http.StatusBadRequest)
			errChan <- fmt.Errorf("invalid state")
			return
		}

		code := r.URL.Query().Get("code")
		token, err := sa.config.Exchange(ctx, code)
		if err != nil {
			http.Error(w, "Token exchange failed", http.StatusInternalServerError)
			errChan <- err
			return
		}

		fmt.Fprintf(w, "Authorization successful! You can close this window.")
		tokenChan <- token
	})

	sa.httpServer = &http.Server{Addr: ":8080", Handler: mux}
	go sa.httpServer.ListenAndServe()

	authURL := sa.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
	fmt.Printf("Visit this URL to authorize: %s\n", authURL)

	// Abre a URL de autorização no navegador padrão do usuário
	err := browser.OpenURL(authURL)
	if err != nil {
		// Se não conseguir abrir o navegador, retorna o erro para que o usuário possa usar a URL do console
		return nil, fmt.Errorf("failed to open browser for authentication: %w. Please visit the URL printed in the console", err)
	}

	select {
	case token := <-tokenChan:
		sa.httpServer.Shutdown(ctx)
		sa.saveToken(token)
		return token, nil
	case err := <-errChan:
		sa.httpServer.Shutdown(ctx)
		return nil, err
	case <-ctx.Done():
		sa.httpServer.Shutdown(ctx)
		return nil, ctx.Err()
	}
}

func (sa *StravaAuth) generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func (sa *StravaAuth) loadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(sa.tokenFile)
	if err != nil {
		return nil, err
	}

	var tokenData TokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return nil, err
	}

	return tokenData.Token, nil
}

func (sa *StravaAuth) saveToken(token *oauth2.Token) error {
	os.MkdirAll(filepath.Dir(sa.tokenFile), 0755)

	tokenData := TokenData{
		Token:     token,
		ExpiresAt: token.Expiry,
	}

	data, err := json.MarshalIndent(tokenData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sa.tokenFile, data, 0600)
}
