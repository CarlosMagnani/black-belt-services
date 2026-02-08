package efi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

// WebhookHandler processa webhooks recebidos da Efí Bank
type WebhookHandler struct {
	// OnPixPayment é chamado quando um pagamento PIX é recebido
	OnPixPayment func(ctx context.Context, pix PixPayment) error

	// OnRecurrenceUpdate é chamado quando o status de uma recorrência muda
	OnRecurrenceUpdate func(ctx context.Context, event RecurrenceEvent) error

	// OnError é chamado quando ocorre um erro durante o processamento
	OnError func(ctx context.Context, err error)

	// WebhookSecret é o secret para validar assinaturas (opcional)
	WebhookSecret string

	// SkipSignatureValidation desabilita validação de assinatura (apenas para testes)
	SkipSignatureValidation bool
}

// NewWebhookHandler cria um novo handler de webhook
func NewWebhookHandler() *WebhookHandler {
	return &WebhookHandler{}
}

// HandleEfiWebhook é o handler HTTP para webhooks da Efí
// Monte em POST /webhooks/efi
func (h *WebhookHandler) HandleEfiWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Apenas aceita POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Lê o body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Valida assinatura se configurada
	if h.WebhookSecret != "" && !h.SkipSignatureValidation {
		signature := r.Header.Get("X-Signature")
		if signature == "" {
			signature = r.Header.Get("x-signature")
		}
		if !h.validateSignature(body, signature) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse do evento
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Processa o evento
	if err := h.processEvent(ctx, event); err != nil {
		log.Printf("Erro ao processar webhook: %v", err)
		if h.OnError != nil {
			h.OnError(ctx, err)
		}
		// Retorna 200 para evitar retries da Efí
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

// validateSignature valida a assinatura do webhook usando HMAC-SHA256
func (h *WebhookHandler) validateSignature(body []byte, signature string) bool {
	if signature == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.WebhookSecret))
	mac.Write(body)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSig))
}

// processEvent roteia o evento para o handler apropriado
func (h *WebhookHandler) processEvent(ctx context.Context, event WebhookEvent) error {
	// Processa pagamentos PIX
	for _, pix := range event.Pix {
		if err := h.ProcessPixPayment(ctx, pix); err != nil {
			return err
		}
	}

	// Processa eventos de recorrência
	if event.Rec != nil {
		if err := h.ProcessRecurrenceUpdate(ctx, *event.Rec); err != nil {
			return err
		}
	}

	return nil
}

// ProcessPixPayment processa uma notificação de pagamento PIX
func (h *WebhookHandler) ProcessPixPayment(ctx context.Context, pix PixPayment) error {
	log.Printf("PIX recebido: e2e=%s txid=%s valor=%s", pix.EndToEndID, pix.TxID, pix.Value)

	if h.OnPixPayment == nil {
		return nil
	}

	return h.OnPixPayment(ctx, pix)
}

// ProcessRecurrenceUpdate processa uma notificação de mudança de status de recorrência
func (h *WebhookHandler) ProcessRecurrenceUpdate(ctx context.Context, event RecurrenceEvent) error {
	log.Printf("Recorrência atualizada: id=%s status=%s", event.ID, event.Status)

	if h.OnRecurrenceUpdate == nil {
		return nil
	}

	return h.OnRecurrenceUpdate(ctx, event)
}

// WebhookConfig contém configuração de webhook registrado
type WebhookConfig struct {
	URL      string `json:"webhookUrl"`
	ChavePix string `json:"chave"` // Chave PIX associada ao webhook
}

// GetWebhook consulta o webhook registrado para uma chave PIX
func (c *Client) GetWebhook(ctx context.Context, pixKey string) (*WebhookConfig, error) {
	if pixKey == "" {
		return nil, nil
	}

	path := "/v2/webhook/" + pixKey

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		// 404 significa que não há webhook registrado
		if apiErr, ok := err.(*APIError); ok && apiErr.Status == 404 {
			return nil, nil
		}
		return nil, err
	}

	var config WebhookConfig
	if err := json.Unmarshal(respBody, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// DeleteWebhook remove um webhook registrado
func (c *Client) DeleteWebhook(ctx context.Context, pixKey string) error {
	if pixKey == "" {
		return nil
	}

	path := "/v2/webhook/" + pixKey

	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// ListWebhooks lista todos os webhooks registrados
func (c *Client) ListWebhooks(ctx context.Context) ([]WebhookConfig, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/v2/webhook", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Webhooks []WebhookConfig `json:"webhooks"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return result.Webhooks, nil
}

// ExtractPixKeyFromURL extrai a chave PIX de uma URL de webhook
func ExtractPixKeyFromURL(path string) string {
	// Espera formato: /webhooks/efi/{pixKey}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "webhooks" && parts[1] == "efi" {
		return parts[2]
	}
	return ""
}
