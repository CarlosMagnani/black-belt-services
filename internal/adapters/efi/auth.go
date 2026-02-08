package efi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// TokenManager gerencia tokens OAuth2 com refresh automático
// É thread-safe e cacheia o token até próximo da expiração
type TokenManager struct {
	clientID     string
	clientSecret string
	baseURL      string
	httpClient   *http.Client

	mu          sync.RWMutex
	token       string
	expiresAt   time.Time
	refreshLead time.Duration // Tempo antes da expiração para fazer refresh
}

// NewTokenManager cria um novo gerenciador de tokens
func NewTokenManager(clientID, clientSecret, baseURL string, httpClient *http.Client) *TokenManager {
	return &TokenManager{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		httpClient:   httpClient,
		refreshLead:  60 * time.Second, // Renova 1 minuto antes de expirar
	}
}

// GetToken retorna um token válido, renovando se necessário
func (tm *TokenManager) GetToken() (string, error) {
	tm.mu.RLock()
	// Verifica se o token atual ainda é válido
	if tm.token != "" && time.Now().Add(tm.refreshLead).Before(tm.expiresAt) {
		token := tm.token
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// Token expirado ou inexistente, precisa renovar
	return tm.refresh()
}

// refresh obtém um novo token da API
func (tm *TokenManager) refresh() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check: outra goroutine pode ter renovado enquanto esperávamos o lock
	if tm.token != "" && time.Now().Add(tm.refreshLead).Before(tm.expiresAt) {
		return tm.token, nil
	}

	// Prepara a requisição de autenticação
	authURL := fmt.Sprintf("%s/oauth/token", tm.baseURL)

	body := strings.NewReader("grant_type=client_credentials")
	req, err := http.NewRequest(http.MethodPost, authURL, body)
	if err != nil {
		return "", fmt.Errorf("erro ao criar requisição de auth: %w", err)
	}

	// Basic Auth com client_id:client_secret
	credentials := base64.StdEncoding.EncodeToString(
		[]byte(fmt.Sprintf("%s:%s", tm.clientID, tm.clientSecret)),
	)
	req.Header.Set("Authorization", "Basic "+credentials)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Faz a requisição
	resp, err := tm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro na requisição de auth: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erro ao ler resposta de auth: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Mensagem != "" {
			return "", fmt.Errorf("erro de autenticação: %s", apiErr.Mensagem)
		}
		return "", fmt.Errorf("erro de autenticação: status %d - %s", resp.StatusCode, string(respBody))
	}

	// Parse da resposta
	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("erro ao decodificar token: %w", err)
	}

	// Atualiza o cache
	tm.token = tokenResp.AccessToken
	tm.expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return tm.token, nil
}

// Invalidate força a renovação do token na próxima chamada
// Útil quando recebemos erro 401
func (tm *TokenManager) Invalidate() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.token = ""
	tm.expiresAt = time.Time{}
}
