package domain

import (
	"encoding/json"
	"time"
)

// WebhookStatus representa o estado de processamento de um webhook (alinhado com enum SQL webhook_status)
type WebhookStatus string

const (
	WebhookStatusPending    WebhookStatus = "pending"
	WebhookStatusProcessing WebhookStatus = "processing"
	WebhookStatusProcessed  WebhookStatus = "processed"
	WebhookStatusFailed     WebhookStatus = "failed"
	WebhookStatusSkipped    WebhookStatus = "skipped"
)

// ValidWebhookStatuses lista todos os status válidos
var ValidWebhookStatuses = []WebhookStatus{
	WebhookStatusPending,
	WebhookStatusProcessing,
	WebhookStatusProcessed,
	WebhookStatusFailed,
	WebhookStatusSkipped,
}

// IsValid verifica se o status é válido
func (s WebhookStatus) IsValid() bool {
	for _, v := range ValidWebhookStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// MaxWebhookRetries define o número máximo de tentativas
const MaxWebhookRetries = 5

// WebhookEvent representa um evento de webhook recebido para auditoria
// Alinhado com tabela SQL: public.webhook_events
type WebhookEvent struct {
	ID string `json:"id"`

	// Source
	Gateway string `json:"gateway"` // "pix_auto" | "stripe"

	// Event info
	EventID   string `json:"event_id"`   // ID do evento no gateway (UNIQUE)
	EventType string `json:"event_type"` // Tipo do evento

	// Payload
	Payload json.RawMessage `json:"payload"`
	Headers json.RawMessage `json:"headers,omitempty"` // Request headers (para validação)

	// Processing
	Status       WebhookStatus `json:"status"`
	ProcessedAt  *time.Time    `json:"processed_at,omitempty"`
	ErrorMessage *string       `json:"error_message,omitempty"`
	RetryCount   int           `json:"retry_count"`
	NextRetryAt  *time.Time    `json:"next_retry_at,omitempty"`

	// Timestamps
	ReceivedAt time.Time `json:"received_at"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewWebhookEvent cria um novo evento de webhook pendente
func NewWebhookEvent(gateway, eventID, eventType string, payload, headers json.RawMessage) *WebhookEvent {
	now := time.Now()
	return &WebhookEvent{
		Gateway:    gateway,
		EventID:    eventID,
		EventType:  eventType,
		Payload:    payload,
		Headers:    headers,
		Status:     WebhookStatusPending,
		RetryCount: 0,
		ReceivedAt: now,
		CreatedAt:  now,
	}
}

// MarkProcessing marca o webhook como em processamento
func (w *WebhookEvent) MarkProcessing() {
	w.Status = WebhookStatusProcessing
}

// MarkProcessed marca o webhook como processado com sucesso
func (w *WebhookEvent) MarkProcessed() {
	now := time.Now()
	w.Status = WebhookStatusProcessed
	w.ProcessedAt = &now
}

// MarkFailed marca o webhook como falho e agenda retry com backoff exponencial
func (w *WebhookEvent) MarkFailed(errMsg string) {
	w.Status = WebhookStatusFailed
	w.ErrorMessage = &errMsg
	w.RetryCount++

	if w.RetryCount <= MaxWebhookRetries {
		// Backoff exponencial: 1min, 2min, 4min, 8min, 16min
		backoff := time.Duration(1<<uint(w.RetryCount-1)) * time.Minute
		nextRetry := time.Now().Add(backoff)
		w.NextRetryAt = &nextRetry
	}
}

// MarkSkipped marca o webhook como pulado (evento duplicado ou irrelevante)
func (w *WebhookEvent) MarkSkipped() {
	w.Status = WebhookStatusSkipped
}

// CanRetry verifica se o webhook pode ser retentado
func (w *WebhookEvent) CanRetry() bool {
	return w.Status == WebhookStatusFailed && w.RetryCount <= MaxWebhookRetries
}

// IsRetryDue verifica se já é hora de retentar
func (w *WebhookEvent) IsRetryDue() bool {
	if !w.CanRetry() || w.NextRetryAt == nil {
		return false
	}
	return time.Now().After(*w.NextRetryAt)
}
