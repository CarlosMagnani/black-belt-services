package domain

import "time"

// SubscriptionStatus representa o estado de uma assinatura
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusPending   SubscriptionStatus = "pending"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusExpired   SubscriptionStatus = "expired"
	SubscriptionStatusTrial     SubscriptionStatus = "trial"
)

// Subscription representa uma assinatura de um aluno a um plano
type Subscription struct {
	ID                 string             `json:"id"`
	UserID             string             `json:"user_id"`
	PlanID             string             `json:"plan_id"`
	Status             SubscriptionStatus `json:"status"`
	ExternalID         string             `json:"external_id"`         // ID na Efí Bank
	CurrentPeriodStart time.Time          `json:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end"`
	TrialEnd           *time.Time         `json:"trial_end,omitempty"`
	CancelledAt        *time.Time         `json:"cancelled_at,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// IsActive verifica se a assinatura está ativa ou em período de teste
func (s *Subscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive || s.Status == SubscriptionStatusTrial
}

// IsInTrial verifica se está no período de teste
func (s *Subscription) IsInTrial() bool {
	if s.TrialEnd == nil {
		return false
	}
	return time.Now().Before(*s.TrialEnd)
}

// DaysUntilExpiration retorna dias até expiração do período atual
func (s *Subscription) DaysUntilExpiration() int {
	if s.CurrentPeriodEnd.IsZero() {
		return 0
	}
	duration := time.Until(s.CurrentPeriodEnd)
	return int(duration.Hours() / 24)
}

// NewSubscription cria uma nova assinatura pendente
func NewSubscription(userID, planID string) *Subscription {
	now := time.Now()
	return &Subscription{
		UserID:    userID,
		PlanID:    planID,
		Status:    SubscriptionStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Activate ativa a assinatura definindo o período
func (s *Subscription) Activate(periodStart, periodEnd time.Time) {
	s.Status = SubscriptionStatusActive
	s.CurrentPeriodStart = periodStart
	s.CurrentPeriodEnd = periodEnd
	s.UpdatedAt = time.Now()
}

// Cancel cancela a assinatura
func (s *Subscription) Cancel() {
	now := time.Now()
	s.Status = SubscriptionStatusCancelled
	s.CancelledAt = &now
	s.UpdatedAt = now
}
