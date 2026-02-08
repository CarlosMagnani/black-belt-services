package efi

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/pkcs12"

	"github.com/magnani/black-belt-app/backend/internal/config"
	"github.com/magnani/black-belt-app/backend/internal/ports"
)

// Client implementa ports.PaymentProvider para a API Efí Bank
type Client struct {
	baseURL      string
	pixKey       string // Chave PIX do recebedor
	httpClient   *http.Client
	tokenManager *TokenManager
}

// NewClient cria um novo cliente Efí com mTLS configurado
func NewClient(cfg *config.EfiConfig, pixKey string) (*Client, error) {
	// Carrega o certificado para mTLS
	tlsConfig, err := loadCertificate(cfg.CertificatePath, cfg.CertificatePassword)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar certificado: %w", err)
	}

	// Configura o cliente HTTP com mTLS
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Cria o gerenciador de tokens
	tokenManager := NewTokenManager(cfg.ClientID, cfg.ClientSecret, cfg.PixURL, httpClient)

	return &Client{
		baseURL:      cfg.PixURL,
		pixKey:       pixKey,
		httpClient:   httpClient,
		tokenManager: tokenManager,
	}, nil
}

// loadCertificate carrega um certificado .p12 ou .pem para mTLS
func loadCertificate(certPath, password string) (*tls.Config, error) {
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler certificado: %w", err)
	}

	// Tenta decodificar como PKCS12 (.p12)
	privateKey, certificate, err := pkcs12.Decode(certData, password)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar certificado PKCS12: %w", err)
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{certificate.Raw},
		PrivateKey:  privateKey,
	}

	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		MinVersion:   tls.VersionTLS12,
	}, nil
}

// doRequest executa uma requisição HTTP autenticada
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Obtém token válido
	token, err := c.tokenManager.GetToken()
	if err != nil {
		return nil, err
	}

	// Prepara o body se houver
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("erro ao serializar body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	// Cria a requisição
	url := fmt.Sprintf("%s%s", c.baseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Executa
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição HTTP: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// Trata erros de autenticação
	if resp.StatusCode == http.StatusUnauthorized {
		c.tokenManager.Invalidate()
		return nil, fmt.Errorf("token inválido ou expirado")
	}

	// Trata erros da API
	if resp.StatusCode >= 400 {
		var apiErr APIError
		if json.Unmarshal(respBody, &apiErr) == nil {
			return nil, &apiErr
		}
		return nil, fmt.Errorf("erro da API: status %d - %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// CreatePixCharge cria uma nova cobrança PIX imediata
func (c *Client) CreatePixCharge(ctx context.Context, req *ports.PixChargeRequest) (*ports.PixChargeResponse, error) {
	// Monta o request para a API Efí
	efiReq := PixCobRequest{
		Calendario: PixCalendario{
			Expiracao: req.ExpiresIn,
		},
		Valor: PixValor{
			Original: fmt.Sprintf("%.2f", float64(req.Amount)/100),
		},
		Chave:          c.pixKey,
		SolicitacaoPag: req.Description,
	}

	// Adiciona dados do pagador se informados
	if req.PayerName != "" || req.PayerDocument != "" {
		efiReq.Devedor = &PixDevedor{
			Nome: req.PayerName,
		}
		if len(req.PayerDocument) == 11 {
			efiReq.Devedor.CPF = req.PayerDocument
		} else if len(req.PayerDocument) == 14 {
			efiReq.Devedor.CNPJ = req.PayerDocument
		}
	}

	// Define o endpoint baseado se temos txid ou não
	var path string
	var method string
	if req.TxID != "" {
		path = fmt.Sprintf("/v2/cob/%s", req.TxID)
		method = http.MethodPut
	} else {
		path = "/v2/cob"
		method = http.MethodPost
	}

	// Faz a requisição
	respBody, err := c.doRequest(ctx, method, path, efiReq)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cobrança PIX: %w", err)
	}

	// Parse da resposta
	var efiResp PixCobResponse
	if err := json.Unmarshal(respBody, &efiResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &ports.PixChargeResponse{
		TxID:      efiResp.TxID,
		Location:  efiResp.Location,
		PixCode:   efiResp.PixCopiaECola,
		ExpiresAt: efiResp.Calendario.Criacao,
	}, nil
}

// GetPixCharge consulta uma cobrança PIX pelo txid
func (c *Client) GetPixCharge(ctx context.Context, txid string) (*ports.PixChargeResponse, error) {
	path := fmt.Sprintf("/v2/cob/%s", txid)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar cobrança: %w", err)
	}

	var efiResp PixCobResponse
	if err := json.Unmarshal(respBody, &efiResp); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &ports.PixChargeResponse{
		TxID:      efiResp.TxID,
		Location:  efiResp.Location,
		PixCode:   efiResp.PixCopiaECola,
		ExpiresAt: efiResp.Calendario.Criacao,
	}, nil
}

// CancelPixCharge cancela uma cobrança PIX pendente
func (c *Client) CancelPixCharge(ctx context.Context, txid string) error {
	path := fmt.Sprintf("/v2/cob/%s", txid)

	// Para cancelar, enviamos PATCH com status REMOVIDA_PELO_USUARIO_RECEBEDOR
	patchData := map[string]string{"status": "REMOVIDA_PELO_USUARIO_RECEBEDOR"}

	_, err := c.doRequest(ctx, http.MethodPatch, path, patchData)
	if err != nil {
		return fmt.Errorf("erro ao cancelar cobrança: %w", err)
	}

	return nil
}

// RefundPix solicita devolução de um PIX recebido
func (c *Client) RefundPix(ctx context.Context, e2eID string, amount int64) error {
	// Gera um ID único para a devolução
	refundID := fmt.Sprintf("dev%d", time.Now().UnixNano())
	path := fmt.Sprintf("/v2/pix/%s/devolucao/%s", e2eID, refundID)

	devReq := PixDevolucaoRequest{
		Valor: fmt.Sprintf("%.2f", float64(amount)/100),
	}

	_, err := c.doRequest(ctx, http.MethodPut, path, devReq)
	if err != nil {
		return fmt.Errorf("erro ao solicitar devolução: %w", err)
	}

	return nil
}

// RegisterWebhook registra a URL de webhook para uma chave PIX
func (c *Client) RegisterWebhook(ctx context.Context, pixKey string, webhookURL string) error {
	path := fmt.Sprintf("/v2/webhook/%s", pixKey)

	webhookReq := map[string]string{"webhookUrl": webhookURL}

	_, err := c.doRequest(ctx, http.MethodPut, path, webhookReq)
	if err != nil {
		return fmt.Errorf("erro ao registrar webhook: %w", err)
	}

	return nil
}

// ParseWebhookEvent processa o payload de um webhook e retorna o evento estruturado
func (c *Client) ParseWebhookEvent(payload []byte, signature string) (*ports.WebhookEvent, error) {
	// TODO: Implementar validação de assinatura quando Efí disponibilizar
	// Por enquanto, apenas faz o parse do payload

	var webhookPayload PixWebhookPayload
	if err := json.Unmarshal(payload, &webhookPayload); err != nil {
		return nil, fmt.Errorf("erro ao decodificar webhook: %w", err)
	}

	// Converte para o formato genérico
	event := &ports.WebhookEvent{
		Type:      "pix",
		Timestamp: time.Now().Format(time.RFC3339),
		Data:      make(map[string]interface{}),
	}

	if len(webhookPayload.Pix) > 0 {
		pix := webhookPayload.Pix[0]
		event.Data["txid"] = pix.TxID
		event.Data["endToEndId"] = pix.EndToEndID
		event.Data["valor"] = pix.Valor
		event.Data["horario"] = pix.Horario
		event.Data["pagador"] = pix.Pagador
	}

	return event, nil
}

// Garante que Client implementa PaymentProvider
var _ ports.PaymentProvider = (*Client)(nil)
