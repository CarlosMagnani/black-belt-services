package domain

import "time"

// PaymentStatus representa o estado de um pagamento
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusConfirmed PaymentStatus = "confirmed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusExpired   PaymentStatus = "expired"
)

// PaymentMethod representa o método de pagamento
type PaymentMethod string

const (
	PaymentMethodPix    PaymentMethod = "pix"
	PaymentMethodBoleto PaymentMethod = "boleto"
	PaymentMethodCard   PaymentMethod = "card"
)

// Payment representa um pagamento
type Payment struct {
	ID             string        `json:"id"`
	SubscriptionID string        `json:"subscription_id"`
	UserID         string        `json:"user_id"`
	Amount         int64         `json:"amount"` // Valor em centavos
	Status         PaymentStatus `json:"status"`
	Method         PaymentMethod `json:"method"`
	ExternalID     string        `json:"external_id"` // ID na Efí Bank (txid para PIX)
	PixCode        string        `json:"pix_code,omitempty"`
	PixQRCode      string        `json:"pix_qr_code,omitempty"`
	ExpiresAt      *time.Time    `json:"expires_at,omitempty"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// IsPaid verifica se o pagamento foi confirmado
func (p *Payment) IsPaid() bool {
	return p.Status == PaymentStatusConfirmed
}

// IsExpired verifica se o pagamento expirou
func (p *Payment) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// AmountInReais retorna o valor em reais
func (p *Payment) AmountInReais() float64 {
	return float64(p.Amount) / 100
}

// NewPayment cria um novo pagamento pendente
func NewPayment(subscriptionID, userID string, amountInCents int64, method PaymentMethod) *Payment {
	now := time.Now()
	return &Payment{
		SubscriptionID: subscriptionID,
		UserID:         userID,
		Amount:         amountInCents,
		Status:         PaymentStatusPending,
		Method:         method,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// Confirm marca o pagamento como confirmado
func (p *Payment) Confirm() {
	now := time.Now()
	p.Status = PaymentStatusConfirmed
	p.PaidAt = &now
	p.UpdatedAt = now
}

// Fail marca o pagamento como falho
func (p *Payment) Fail() {
	p.Status = PaymentStatusFailed
	p.UpdatedAt = time.Now()
}

// Refund marca o pagamento como reembolsado
func (p *Payment) Refund() {
	p.Status = PaymentStatusRefunded
	p.UpdatedAt = time.Now()
}
