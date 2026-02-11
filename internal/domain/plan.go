// Package domain contém as entidades de domínio da aplicação
package domain

import (
	"encoding/json"
	"time"
)

// SubscriptionPlan representa um plano de assinatura oferecido pela BlackBelt
// Alinhado com tabela SQL: public.subscription_plans
type SubscriptionPlan struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"` // starter, pro, business
	Description string `json:"description,omitempty"`

	// Pricing (centavos)
	PriceMonthly int  `json:"price_monthly"`
	PriceYearly  *int `json:"price_yearly,omitempty"`
	Currency     string `json:"currency"` // default "BRL"

	// Limits
	MaxStudents   *int `json:"max_students,omitempty"`   // NULL = unlimited
	MaxProfessors *int `json:"max_professors,omitempty"` // NULL = unlimited
	MaxLocations  int  `json:"max_locations"`

	// Features (JSON array)
	Features json.RawMessage `json:"features"`

	// Status
	IsActive bool `json:"is_active"`

	// Stripe integration
	StripePriceIDMonthly *string `json:"stripe_price_id_monthly,omitempty"`
	StripePriceIDYearly  *string `json:"stripe_price_id_yearly,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PriceMonthlyInReais retorna o preço mensal formatado em reais
func (p *SubscriptionPlan) PriceMonthlyInReais() float64 {
	return float64(p.PriceMonthly) / 100
}

// PriceYearlyInReais retorna o preço anual formatado em reais (0 se não disponível)
func (p *SubscriptionPlan) PriceYearlyInReais() float64 {
	if p.PriceYearly == nil {
		return 0
	}
	return float64(*p.PriceYearly) / 100
}

// HasYearlyOption verifica se o plano tem opção anual
func (p *SubscriptionPlan) HasYearlyOption() bool {
	return p.PriceYearly != nil
}

// IsUnlimitedStudents verifica se o plano tem alunos ilimitados
func (p *SubscriptionPlan) IsUnlimitedStudents() bool {
	return p.MaxStudents == nil
}

// NewSubscriptionPlan cria um novo plano com valores padrão
func NewSubscriptionPlan(name, slug string, priceMonthlyInCents int) *SubscriptionPlan {
	now := time.Now()
	return &SubscriptionPlan{
		Name:         name,
		Slug:         slug,
		PriceMonthly: priceMonthlyInCents,
		Currency:     "BRL",
		MaxLocations: 1,
		Features:     json.RawMessage("[]"),
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
