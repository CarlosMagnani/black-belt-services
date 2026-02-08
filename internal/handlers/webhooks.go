// Package handlers contém os handlers HTTP da aplicação
package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/magnani/black-belt-app/backend/internal/ports"
)

// WebhookHandler gerencia webhooks recebidos de provedores de pagamento
type WebhookHandler struct {
	paymentProvider ports.PaymentProvider
	webhookSecret   string
	eventHandlers   map[string]WebhookEventHandler
}

// WebhookEventHandler é uma função que processa um tipo específico de evento
type WebhookEventHandler func(event *ports.WebhookEvent) error

// NewWebhookHandler cria um novo handler de webhooks
func NewWebhookHandler(provider ports.PaymentProvider, secret string) *WebhookHandler {
	return &WebhookHandler{
		paymentProvider: provider,
		webhookSecret:   secret,
		eventHandlers:   make(map[string]WebhookEventHandler),
	}
}

// RegisterHandler registra um handler para um tipo de evento
func (wh *WebhookHandler) RegisterHandler(eventType string, handler WebhookEventHandler) {
	wh.eventHandlers[eventType] = handler
}

// HandleEfiWebhook processa webhooks da Efí Bank
// Endpoint: POST /api/webhooks/efi
func (wh *WebhookHandler) HandleEfiWebhook(w http.ResponseWriter, r *http.Request) {
	// Apenas POST é permitido
	if r.Method != http.MethodPost {
		http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		return
	}

	// Lê o body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Webhook] Erro ao ler body: %v", err)
		http.Error(w, "Erro ao ler requisição", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log do webhook recebido (útil para debug)
	log.Printf("[Webhook] Recebido: %s", string(body))

	// Obtém a assinatura do header (se existir)
	signature := r.Header.Get("X-Webhook-Signature")

	// Valida e parseia o webhook
	event, err := wh.paymentProvider.ParseWebhookEvent(body, signature)
	if err != nil {
		log.Printf("[Webhook] Erro ao processar: %v", err)
		http.Error(w, "Erro ao processar webhook", http.StatusBadRequest)
		return
	}

	// Roteia para o handler apropriado
	if handler, ok := wh.eventHandlers[event.Type]; ok {
		if err := handler(event); err != nil {
			log.Printf("[Webhook] Erro no handler '%s': %v", event.Type, err)
			// Retornamos 200 mesmo assim para evitar retentativas
			// O erro já foi logado e pode ser tratado posteriormente
		}
	} else {
		log.Printf("[Webhook] Tipo de evento não tratado: %s", event.Type)
	}

	// Retorna 200 OK para confirmar recebimento
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received"})
}

// HandlePixReceived é um exemplo de handler para pagamentos PIX recebidos
func HandlePixReceived(event *ports.WebhookEvent) error {
	txid, ok := event.Data["txid"].(string)
	if !ok {
		log.Printf("[PIX] Evento sem txid válido")
		return nil
	}

	valor, _ := event.Data["valor"].(string)
	e2eID, _ := event.Data["endToEndId"].(string)

	log.Printf("[PIX] Pagamento recebido! TxID: %s, Valor: R$ %s, E2E: %s", txid, valor, e2eID)

	// TODO: Implementar lógica de negócio
	// 1. Buscar a cobrança pelo txid
	// 2. Atualizar status do pagamento no banco
	// 3. Ativar/renovar assinatura do usuário
	// 4. Enviar confirmação por email/push

	return nil
}

// HealthCheck endpoint para verificar se o servidor está funcionando
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "blackbelt-api",
	})
}
