package domain

import "time"

// PaymentStatus representa o estado de um pagamento (alinhado com enum SQL payment_status)
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// ValidPaymentStatuses lista todos os status válidos
var ValidPaymentStatuses = []PaymentStatus{
	PaymentStatusPending,
	PaymentStatusProcessing,
	PaymentStatusSucceeded,
	PaymentStatusFailed,
	PaymentStatusRefunded,
}

// IsValid verifica se o status é válido
func (s PaymentStatus) IsValid() bool {
	for _, v := range ValidPaymentStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// PaymentHistory representa um registro de pagamento no histórico
// Alinhado com tabela SQL: public.payment_history
type PaymentHistory struct {
	ID             string `json:"id"`
	SubscriptionID string `json:"subscription_id"`
	AcademyID      string `json:"academy_id"` // CHANGED: era UserID

	// Amount (centavos)
	Amount   int    `json:"amount"`
	Currency string `json:"currency"` // default "BRL"

	// Gateway info
	PaymentGateway   PaymentGateway `json:"payment_gateway"`
	GatewayPaymentID *string        `json:"gateway_payment_id,omitempty"` // PIX txid ou Stripe payment_intent_id
	GatewayChargeID  *string        `json:"gateway_charge_id,omitempty"`  // ID da cobrança recorrente
	GatewayInvoiceID *string        `json:"gateway_invoice_id,omitempty"` // Stripe invoice_id

	// Status
	Status PaymentStatus `json:"status"`

	// Details
	PaymentMethod *string `json:"payment_method,omitempty"` // "pix" | "card"
	FailureReason *string `json:"failure_reason,omitempty"`
	FailureCode   *string `json:"failure_code,omitempty"`

	// Period covered
	PeriodStart *time.Time `json:"period_start,omitempty"`
	PeriodEnd   *time.Time `json:"period_end,omitempty"`

	// Timestamps
	PaidAt    *time.Time `json:"paid_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsPaid verifica se o pagamento foi confirmado
func (p *PaymentHistory) IsPaid() bool {
	return p.Status == PaymentStatusSucceeded
}

// AmountInReais retorna o valor em reais
func (p *PaymentHistory) AmountInReais() float64 {
	return float64(p.Amount) / 100
}

// NewPaymentHistory cria um novo registro de pagamento pendente
func NewPaymentHistory(subscriptionID, academyID string, amountInCents int, gateway PaymentGateway) *PaymentHistory {
	return &PaymentHistory{
		SubscriptionID: subscriptionID,
		AcademyID:      academyID,
		Amount:         amountInCents,
		Currency:       "BRL",
		PaymentGateway: gateway,
		Status:         PaymentStatusPending,
		CreatedAt:      time.Now(),
	}
}

// MarkProcessing marca o pagamento como em processamento
func (p *PaymentHistory) MarkProcessing() {
	p.Status = PaymentStatusProcessing
}

// Succeed marca o pagamento como bem-sucedido
func (p *PaymentHistory) Succeed() {
	now := time.Now()
	p.Status = PaymentStatusSucceeded
	p.PaidAt = &now
}

// Fail marca o pagamento como falho
func (p *PaymentHistory) Fail(reason, code string) {
	p.Status = PaymentStatusFailed
	p.FailureReason = &reason
	p.FailureCode = &code
}

// Refund marca o pagamento como reembolsado
func (p *PaymentHistory) Refund() {
	p.Status = PaymentStatusRefunded
}
