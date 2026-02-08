package efi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// AccountsClient é um cliente específico para a API de abertura de contas.
// Esta API está em URL diferente e requer autorização especial de parceiro.
type AccountsClient struct {
	baseURL      string
	httpClient   *http.Client
	tokenManager *TokenManager
}

// accountsBaseURL define a URL base da API de abertura de contas
func (c *Client) accountsBaseURL() string {
	if strings.Contains(c.baseURL, "pix-h") || strings.Contains(c.baseURL, "sandbox") {
		return AccountsURLSandbox
	}
	return AccountsURLProd
}

// accountsClient retorna um cliente de contas usando o mesmo token manager
func (c *Client) accountsClient() *AccountsClient {
	return &AccountsClient{
		baseURL:      c.accountsBaseURL(),
		httpClient:   c.httpClient,
		tokenManager: c.tokenManager,
	}
}

// NewAccountsClient cria um cliente para a API de abertura de contas.
// Requer credenciais de parceiro com acesso à API restrita.
func NewAccountsClient(baseURL string, httpClient *http.Client, tokenManager *TokenManager) *AccountsClient {
	return &AccountsClient{
		baseURL:      baseURL,
		httpClient:   httpClient,
		tokenManager: tokenManager,
	}
}

// CreateAccount cria uma nova conta digital (API restrita - requer autorização especial).
// Este endpoint só está disponível para parceiros com contratos especiais.
func (c *Client) CreateAccount(ctx context.Context, req CreateAccountRequest) (*Account, error) {
	return c.accountsClient().CreateAccount(ctx, req)
}

// GetAccountStatus consulta o status de uma conta
func (c *Client) GetAccountStatus(ctx context.Context, accountID string) (*AccountStatus, error) {
	return c.accountsClient().GetAccountStatus(ctx, accountID)
}

// doRequest executa uma requisição HTTP autenticada
func (c *AccountsClient) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	return doAuthenticatedRequest(ctx, c.tokenManager, c.httpClient, c.baseURL, method, path, body)
}

// CreateAccount cria uma nova conta digital (API restrita - requer autorização especial).
// Este endpoint só está disponível para parceiros com contratos especiais.
func (c *AccountsClient) CreateAccount(ctx context.Context, req CreateAccountRequest) (*Account, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("nome é obrigatório")
	}
	if req.Email == "" {
		return nil, fmt.Errorf("email é obrigatório")
	}
	if req.CPF == "" && req.CNPJ == "" {
		return nil, fmt.Errorf("CPF ou CNPJ é obrigatório")
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, "/v1/conta-simplificada", req)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar conta: %w", err)
	}

	var account Account
	if err := json.Unmarshal(respBody, &account); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &account, nil
}

// GetAccountStatus consulta o status de uma conta
func (c *AccountsClient) GetAccountStatus(ctx context.Context, accountID string) (*AccountStatus, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account_id é obrigatório")
	}

	path := fmt.Sprintf("/v1/conta-simplificada/%s", accountID)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar status da conta: %w", err)
	}

	var status AccountStatus
	if err := json.Unmarshal(respBody, &status); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &status, nil
}

// ListAccounts lista todas as contas criadas (se disponível)
func (c *AccountsClient) ListAccounts(ctx context.Context) ([]Account, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/v1/conta-simplificada", nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar contas: %w", err)
	}

	var result struct {
		Accounts []Account `json:"contas"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return result.Accounts, nil
}

// UpdateAccountStatus atualiza o status de uma conta (ativar, bloquear)
func (c *AccountsClient) UpdateAccountStatus(ctx context.Context, accountID string, newStatus string) error {
	if accountID == "" {
		return fmt.Errorf("account_id é obrigatório")
	}

	path := fmt.Sprintf("/v1/conta-simplificada/%s", accountID)

	payload := map[string]string{
		"status": newStatus,
	}

	_, err := c.doRequest(ctx, http.MethodPatch, path, payload)
	if err != nil {
		return fmt.Errorf("erro ao atualizar status da conta: %w", err)
	}

	return nil
}

// doAuthenticatedRequest é uma função helper que executa requisições autenticadas
func doAuthenticatedRequest(ctx context.Context, tokenManager *TokenManager, httpClient *http.Client, baseURL, method, path string, body interface{}) ([]byte, error) {
	token, err := tokenManager.GetToken()
	if err != nil {
		return nil, err
	}

	var reqBody []byte
	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("erro ao serializar body: %w", err)
		}
	}

	url := fmt.Sprintf("%s%s", baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	if reqBody != nil {
		req.Body = readCloser{bytes: reqBody}
		req.ContentLength = int64(len(reqBody))
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBody := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			respBody = append(respBody, buf[:n]...)
		}
		if readErr != nil {
			break
		}
	}

	if resp.StatusCode == http.StatusUnauthorized {
		tokenManager.Invalidate()
		return nil, fmt.Errorf("token inválido ou expirado")
	}

	if resp.StatusCode >= 400 {
		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil {
			return nil, &apiErr
		}
		return nil, fmt.Errorf("erro da API: status %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// readCloser é um io.ReadCloser simples para bytes
type readCloser struct {
	bytes []byte
	pos   int
}

func (r readCloser) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.bytes) {
		return 0, nil
	}
	n = copy(p, r.bytes[r.pos:])
	r.pos += n
	return n, nil
}

func (r readCloser) Close() error {
	return nil
}
