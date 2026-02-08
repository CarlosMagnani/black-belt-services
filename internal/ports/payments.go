// Package ports define as interfaces (portas) para adaptadores externos
// Seguindo o padrão Hexagonal Architecture / Ports & Adapters
package ports

import (
	"context"

	"github.com/magnani/black-belt-app/backend/internal/domain"
)

// PixChargeRequest representa uma requisição para criar cobrança PIX
type PixChargeRequest struct {
	TxID        string // Identificador único da transação (opcional, será gerado se vazio)
	Amount      int64  // Valor em centavos
	Description string // Descrição da cobrança
	ExpiresIn   int    // Tempo de expiração em segundos (ex: 3600 para 1 hora)

	// Dados do pagador (opcional para PIX imediato)
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

// WebhookEvent representa um evento de webhook recebido
type WebhookEvent struct {
	Type      string                 // Tipo do evento (pix, cobranca, etc)
	Timestamp string                 // Momento do evento
	Data      map[string]interface{} // Dados do evento
}

// PaymentProvider define a interface para provedores de pagamento
// Qualquer gateway (Efí, PagSeguro, Stripe) deve implementar esta interface
type PaymentProvider interface {
	// CreatePixCharge cria uma nova cobrança PIX
	CreatePixCharge(ctx context.Context, req *PixChargeRequest) (*PixChargeResponse, error)

	// GetPixCharge consulta uma cobrança PIX pelo txid
	GetPixCharge(ctx context.Context, txid string) (*PixChargeResponse, error)

	// CancelPixCharge cancela uma cobrança PIX pendente
	CancelPixCharge(ctx context.Context, txid string) error

	// RefundPix solicita devolução de um PIX recebido
	RefundPix(ctx context.Context, e2eID string, amount int64) error

	// RegisterWebhook registra a URL de webhook para receber notificações
	RegisterWebhook(ctx context.Context, pixKey string, webhookURL string) error

	// ParseWebhookEvent processa o payload de um webhook e retorna o evento
	ParseWebhookEvent(payload []byte, signature string) (*WebhookEvent, error)
}

// SubscriptionService define operações de assinatura
type SubscriptionService interface {
	// Create cria uma nova assinatura
	Create(ctx context.Context, userID, planID string) (*domain.Subscription, error)

	// Cancel cancela uma assinatura ativa
	Cancel(ctx context.Context, subscriptionID string) error

	// GetByUser obtém a assinatura ativa de um usuário
	GetByUser(ctx context.Context, userID string) (*domain.Subscription, error)

	// ProcessPayment processa um pagamento recebido
	ProcessPayment(ctx context.Context, externalID string) error
}
