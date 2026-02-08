// Package domain contém as entidades de domínio da aplicação
package domain

import "time"

// PlanInterval representa o intervalo de cobrança do plano
type PlanInterval string

const (
	PlanIntervalMonthly PlanInterval = "monthly"
	PlanIntervalYearly  PlanInterval = "yearly"
)

// Plan representa um plano de assinatura
type Plan struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Price       int64        `json:"price"`       // Valor em centavos
	Interval    PlanInterval `json:"interval"`    // monthly ou yearly
	TrialDays   int          `json:"trial_days"`  // Dias de teste grátis
	Active      bool         `json:"active"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// PriceInReais retorna o preço formatado em reais
func (p *Plan) PriceInReais() float64 {
	return float64(p.Price) / 100
}

// NewPlan cria um novo plano com valores padrão
func NewPlan(name string, priceInCents int64, interval PlanInterval) *Plan {
	now := time.Now()
	return &Plan{
		Name:      name,
		Price:     priceInCents,
		Interval:  interval,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
