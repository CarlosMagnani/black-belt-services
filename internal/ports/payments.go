// Package ports define as interfaces (portas) para adaptadores externos
// Seguindo o padrão Hexagonal Architecture / Ports & Adapters
package ports

import (
	"context"
	"encoding/json"

	"github.com/magnani/black-belt-app/backend/internal/domain"
)

// ──────────────────────────────────────────────
// PIX (Efí Bank) types
// ──────────────────────────────────────────────

// PixChargeRequest representa uma requisição para criar cobrança PIX
type PixChargeRequest struct {
	TxID        string // Identificador único da transação (opcional, será gerado se vazio)
	Amount      int64  // Valor em centavos
	Description string // Descrição da cobrança
	ExpiresIn   int    // Tempo de expiração em segundos (ex: 3600 para 1 hora)

	// Dados do pagador
	PayerName     string
	PayerDocument string // CPF ou CNPJ
}

// PixChargeResponse representa a resposta de uma cobrança PIX criada
type PixChargeResponse struct {
	TxID      string // Identificador da transação
	Location  string // Location do payload
	PixCode   string // Código PIX copia e cola
	QRCodeURL string // URL da imagem do QR Code (se disponível)
	ExpiresAt string // Data/hora de expiração
}

// PixRecurrenceSetupRequest configura PIX Automático recorrente
type PixRecurrenceSetupRequest struct {
	AcademyID    string
	CustomerCPF  string
	CustomerName string
	Amount       int64  // Valor em centavos
	Description  string
}

// PixRecurrenceSetupResponse resposta da configuração de recorrência
type PixRecurrenceSetupResponse struct {
	AuthorizationID string // ID da autorização
	RecurrenceID    string // ID da recorrência configurada
}

// ──────────────────────────────────────────────
// Webhook types (inline — para parsing de payloads)
// ──────────────────────────────────────────────

// IncomingWebhookEvent dados brutos recebidos de um webhook
type IncomingWebhookEvent struct {
	Gateway   string          // "pix_auto" | "stripe"
	EventID   string          // ID do evento no gateway
	EventType string          // Tipo do evento
	Payload   json.RawMessage // Payload completo
	Headers   json.RawMessage // Headers do request
	Signature string          // Assinatura para validação
}

// ──────────────────────────────────────────────
// Provider interfaces
// ──────────────────────────────────────────────

// PixProvider define a interface para o gateway PIX (Efí Bank)
type PixProvider interface {
	// CreatePixCharge cria uma nova cobrança PIX imediata
	CreatePixCharge(ctx context.Context, req *PixChargeRequest) (*PixChargeResponse, error)

	// GetPixCharge consulta uma cobrança PIX pelo txid
	GetPixCharge(ctx context.Context, txid string) (*PixChargeResponse, error)

	// CancelPixCharge cancela uma cobrança PIX pendente
	CancelPixCharge(ctx context.Context, txid string) error

	// RefundPix solicita devolução de um PIX recebido
	RefundPix(ctx context.Context, e2eID string, amount int64) error

	// SetupRecurrence configura PIX Automático recorrente
	SetupRecurrence(ctx context.Context, req *PixRecurrenceSetupRequest) (*PixRecurrenceSetupResponse, error)

	// CancelRecurrence cancela PIX Automático
	CancelRecurrence(ctx context.Context, authorizationID string) error

	// RegisterWebhook registra a URL de webhook para receber notificações PIX
	RegisterWebhook(ctx context.Context, pixKey string, webhookURL string) error

	// ValidateWebhookSignature valida a assinatura de um webhook PIX
	ValidateWebhookSignature(payload []byte, signature string) bool

	// ParseWebhookEvent processa o payload de um webhook e retorna o evento parseado
	ParseWebhookEvent(payload []byte) (*IncomingWebhookEvent, error)
}

// StripeProvider define a interface para o gateway Stripe
type StripeProvider interface {
	// CreateCustomer cria um customer no Stripe
	CreateCustomer(ctx context.Context, academyID, email, name string) (customerID string, err error)

	// CreateSubscription cria uma subscription no Stripe
	CreateSubscription(ctx context.Context, customerID, priceID string) (subscriptionID string, clientSecret string, err error)

	// CancelSubscription cancela uma subscription no Stripe
	CancelSubscription(ctx context.Context, subscriptionID string, atPeriodEnd bool) error

	// ValidateWebhookSignature valida a assinatura de um webhook Stripe
	ValidateWebhookSignature(payload []byte, signature string) bool

	// ParseWebhookEvent processa o payload de um webhook Stripe
	ParseWebhookEvent(payload []byte) (*IncomingWebhookEvent, error)
}

// ──────────────────────────────────────────────
// Service interfaces
// ──────────────────────────────────────────────

// SubscriptionService define operações de assinatura (agora por academy, não por user)
type SubscriptionService interface {
	// CreateTrial cria uma nova assinatura em trial para uma academia
	CreateTrial(ctx context.Context, academyID, planID string) (*domain.Subscription, error)

	// Activate ativa uma assinatura após pagamento
	Activate(ctx context.Context, subscriptionID string, gateway domain.PaymentGateway) (*domain.Subscription, error)

	// Cancel cancela uma assinatura
	Cancel(ctx context.Context, subscriptionID string, reason string, atPeriodEnd bool) error

	// ChangePlan altera o plano de uma assinatura
	ChangePlan(ctx context.Context, subscriptionID, newPlanID string) (*domain.Subscription, error)

	// GetByAcademy obtém a assinatura de uma academia
	GetByAcademy(ctx context.Context, academyID string) (*domain.Subscription, error)

	// GetByStripeSubscriptionID obtém por ID da subscription no Stripe
	GetByStripeSubscriptionID(ctx context.Context, stripeSubID string) (*domain.Subscription, error)

	// ExpireTrials expira trials vencidos (job periódico)
	ExpireTrials(ctx context.Context) (int, error)
}

// PaymentService define operações de pagamento
type PaymentService interface {
	// RecordPayment registra um pagamento no histórico
	RecordPayment(ctx context.Context, payment *domain.PaymentHistory) error

	// GetByGatewayPaymentID busca pagamento pelo ID do gateway
	GetByGatewayPaymentID(ctx context.Context, gatewayPaymentID string) (*domain.PaymentHistory, error)

	// ListByAcademy lista histórico de pagamentos de uma academia
	ListByAcademy(ctx context.Context, academyID string, limit, offset int) ([]*domain.PaymentHistory, error)

	// ListBySubscription lista pagamentos de uma assinatura
	ListBySubscription(ctx context.Context, subscriptionID string) ([]*domain.PaymentHistory, error)
}

// PlanService define operações de planos
type PlanService interface {
	// ListActive lista todos os planos ativos
	ListActive(ctx context.Context) ([]*domain.SubscriptionPlan, error)

	// GetBySlug busca plano pelo slug
	GetBySlug(ctx context.Context, slug string) (*domain.SubscriptionPlan, error)

	// GetByID busca plano pelo ID
	GetByID(ctx context.Context, id string) (*domain.SubscriptionPlan, error)
}

// WebhookService define operações de webhook (auditoria e processamento)
type WebhookService interface {
	// Store armazena um evento de webhook recebido
	Store(ctx context.Context, event *domain.WebhookEvent) error

	// Process processa um evento de webhook (idempotente)
	Process(ctx context.Context, eventID string) error

	// GetByEventID busca webhook pelo event_id do gateway
	GetByEventID(ctx context.Context, eventID string) (*domain.WebhookEvent, error)

	// RetryFailed retenta webhooks falhados que estão prontos para retry
	RetryFailed(ctx context.Context) (processed int, err error)

	// ListPending lista webhooks pendentes de processamento
	ListPending(ctx context.Context, limit int) ([]*domain.WebhookEvent, error)
}
