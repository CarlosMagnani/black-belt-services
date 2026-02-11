package domain

import (
	"encoding/json"
	"time"
)

// SubscriptionStatus representa o estado de uma assinatura (alinhado com enum SQL subscription_status)
type SubscriptionStatus string

const (
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"  // Em período de trial
	SubscriptionStatusActive   SubscriptionStatus = "active"    // Pagamento em dia
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"  // Pagamento atrasado (grace period)
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"  // Cancelada
	SubscriptionStatusExpired  SubscriptionStatus = "expired"   // Trial expirado sem conversão
)

// ValidSubscriptionStatuses lista todos os status válidos
var ValidSubscriptionStatuses = []SubscriptionStatus{
	SubscriptionStatusTrialing,
	SubscriptionStatusActive,
	SubscriptionStatusPastDue,
	SubscriptionStatusCanceled,
	SubscriptionStatusExpired,
}

// IsValid verifica se o status é válido
func (s SubscriptionStatus) IsValid() bool {
	for _, v := range ValidSubscriptionStatuses {
		if s == v {
			return true
		}
	}
	return false
}

// PaymentGateway representa o gateway de pagamento (alinhado com enum SQL payment_gateway)
type PaymentGateway string

const (
	PaymentGatewayPixAuto PaymentGateway = "pix_auto" // PIX Automático (Efí Bank)
	PaymentGatewayStripe  PaymentGateway = "stripe"   // Stripe Billing
)

// Subscription representa uma assinatura de uma academia ao BlackBelt (B2B)
// Alinhado com tabela SQL: public.subscriptions
type Subscription struct {
	ID        string             `json:"id"`
	AcademyID string             `json:"academy_id"` // CHANGED: era UserID
	PlanID    string             `json:"plan_id"`
	Status    SubscriptionStatus `json:"status"`

	// Trial
	TrialStartDate *time.Time `json:"trial_start_date,omitempty"`
	TrialEndDate   *time.Time `json:"trial_end_date,omitempty"`

	// Gateway info
	PaymentGateway *PaymentGateway `json:"payment_gateway,omitempty"`

	// PIX Automático fields (Efí Bank)
	PixAuthorizationID *string `json:"pix_authorization_id,omitempty"`
	PixRecurrenceID    *string `json:"pix_recurrence_id,omitempty"`
	PixCustomerCPF     *string `json:"pix_customer_cpf,omitempty"`
	PixCustomerName    *string `json:"pix_customer_name,omitempty"`

	// Stripe fields
	StripeCustomerID     *string `json:"stripe_customer_id,omitempty"`
	StripeSubscriptionID *string `json:"stripe_subscription_id,omitempty"`
	StripePriceID        *string `json:"stripe_price_id,omitempty"`

	// Billing period
	CurrentPeriodStart *time.Time `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *time.Time `json:"current_period_end,omitempty"`

	// Cancellation
	CanceledAt        *time.Time `json:"canceled_at,omitempty"`
	CancelAtPeriodEnd bool       `json:"cancel_at_period_end"`
	CancelReason      *string    `json:"cancel_reason,omitempty"`

	// Metadata
	Metadata json.RawMessage `json:"metadata,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IsActive verifica se a assinatura está ativa ou em período de teste
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrialing
}

// IsInTrial verifica se está no período de teste
func (s *Subscription) IsInTrial() bool {
	if s.TrialEndDate == nil {
		return false
	}
	return s.Status == SubscriptionStatusTrialing && time.Now().Before(*s.TrialEndDate)
}

// IsPastDue verifica se o pagamento está atrasado
func (s *Subscription) IsPastDue() bool {
	return s.Status == SubscriptionStatusPastDue
}

// DaysUntilExpiration retorna dias até expiração do período atual
func (s *Subscription) DaysUntilExpiration() int {
	if s.CurrentPeriodEnd == nil || s.CurrentPeriodEnd.IsZero() {
		return 0
	}
	duration := time.Until(*s.CurrentPeriodEnd)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}

// DaysUntilTrialEnd retorna dias até o fim do trial
func (s *Subscription) DaysUntilTrialEnd() int {
	if s.TrialEndDate == nil {
		return 0
	}
	duration := time.Until(*s.TrialEndDate)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}

// NewTrialSubscription cria uma nova assinatura em trial para uma academia
func NewTrialSubscription(academyID, planID string, trialDays int) *Subscription {
	now := time.Now()
	trialEnd := now.AddDate(0, 0, trialDays)
	periodEnd := now.AddDate(0, 1, 0)
	return &Subscription{
		AcademyID:          academyID,
		PlanID:             planID,
		Status:             SubscriptionStatusTrialing,
		TrialStartDate:     &now,
		TrialEndDate:       &trialEnd,
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &periodEnd,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// Activate ativa a assinatura definindo o período e gateway
func (s *Subscription) Activate(gateway PaymentGateway, periodStart, periodEnd time.Time) {
	s.Status = SubscriptionStatusActive
	s.PaymentGateway = &gateway
	s.CurrentPeriodStart = &periodStart
	s.CurrentPeriodEnd = &periodEnd
	s.UpdatedAt = time.Now()
}

// MarkPastDue marca a assinatura como pagamento atrasado
func (s *Subscription) MarkPastDue() {
	s.Status = SubscriptionStatusPastDue
	s.UpdatedAt = time.Now()
}

// Cancel cancela a assinatura
func (s *Subscription) Cancel(reason string, atPeriodEnd bool) {
	now := time.Now()
	if atPeriodEnd {
		s.CancelAtPeriodEnd = true
		s.CancelReason = &reason
	} else {
		s.Status = SubscriptionStatusCanceled
		s.CanceledAt = &now
		s.CancelReason = &reason
	}
	s.UpdatedAt = now
}

// Expire marca a assinatura como expirada (trial sem conversão)
func (s *Subscription) Expire() {
	s.Status = SubscriptionStatusExpired
	s.UpdatedAt = time.Now()
}
