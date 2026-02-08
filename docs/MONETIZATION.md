# Modelo de Monetização — BlackBelt

## Resumo Executivo

**Modelo: B2B2C (Business-to-Business-to-Consumer)**

BlackBelt cobra **assinatura mensal da ACADEMIA** (não do aluno). A academia gerencia a cobrança dos seus alunos por conta própria.

### Receita BlackBelt
- Assinatura mensal fixa por academia
- Planos por tier (número de alunos, funcionalidades)
- **Sem taxa por transação de aluno**

### Custo para Academia
- Assinatura BlackBelt (mensal)
- Taxa do gateway (Stripe/PIX) — embutida no preço para o cliente final

---

## Opções de Cobrança Recorrente

### 1. PIX Automático (Recomendado 🇧🇷)

**Lançamento:** 16 de junho de 2025 (Banco Central)

**Vantagens:**
- ✅ **Não compromete limite do cartão** — débito direto da conta
- ✅ Sem convênio bancário necessário (diferente do débito automático tradicional)
- ✅ Taxas menores que cartão de crédito (~0.5-1% vs 2-3%)
- ✅ Liquidação instantânea (D+0)
- ✅ Cliente autoriza uma vez, débitos automáticos seguem
- ✅ Ideal para academias e SaaS no Brasil

**Funcionamento:**
1. Academia gera cobrança com QR Code ou notificação
2. Cliente autoriza no app do banco (uma única vez)
3. Cobranças seguintes são automáticas na data definida
4. Cliente pode cancelar a qualquer momento

**Retentativas:**
- 2 tentativas no dia do vencimento
- 3 tentativas adicionais nos dias seguintes
- Juros/multa incluídos na próxima cobrança se atraso

**Provedores com API:**
- Efí Bank (Gerencianet) — SDK Go disponível
- PagBank/PagSeguro
- Transfeera
- Vindi
- PagBrasil

### 2. Stripe Billing (Internacional)

**Vantagens:**
- ✅ Suporte global (cartões internacionais)
- ✅ SDK Go oficial (`stripe-go/v84`)
- ✅ Stripe Tax para impostos automáticos
- ✅ Customer Portal pronto
- ✅ Webhooks robustos

**Desvantagens:**
- ❌ Compromete limite do cartão (parcelamento tradicional)
- ❌ Taxas mais altas (~3.5% + R$0.40)
- ❌ Chargebacks mais comuns

**Quando usar:**
- Plano internacional (academias fora do Brasil)
- Cliente prefere cartão de crédito
- Fallback se PIX Automático falhar

---

## Arquitetura Proposta

```
┌─────────────────────────────────────────────────────────────┐
│                     BlackBelt Backend (Go)                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────────┐    ┌─────────────────────────────┐    │
│  │ Subscription    │    │     Payment Gateway         │    │
│  │   Service       │───▶│        Adapter              │    │
│  └─────────────────┘    └──────────┬──────────────────┘    │
│                                    │                        │
│                    ┌───────────────┴───────────────┐       │
│                    ▼                               ▼       │
│           ┌───────────────┐              ┌──────────────┐  │
│           │ PIX Automático│              │Stripe Billing│  │
│           │    Adapter    │              │   Adapter    │  │
│           └───────────────┘              └──────────────┘  │
│                    │                               │       │
└────────────────────┼───────────────────────────────┼───────┘
                     ▼                               ▼
            ┌───────────────┐              ┌──────────────┐
            │  Efí Bank /   │              │    Stripe    │
            │  PagBank API  │              │     API      │
            └───────────────┘              └──────────────┘
```

### Interface (Port)

```go
// internal/ports/billing.go

type SubscriptionService interface {
    // Criar assinatura para academia
    CreateSubscription(ctx context.Context, academyID string, planID string) (*Subscription, error)
    
    // Cancelar assinatura
    CancelSubscription(ctx context.Context, subscriptionID string) error
    
    // Listar assinaturas ativas
    ListActiveSubscriptions(ctx context.Context) ([]Subscription, error)
    
    // Processar webhook de pagamento
    HandlePaymentWebhook(ctx context.Context, payload []byte) error
}

type PaymentGateway interface {
    // Criar cobrança recorrente
    CreateRecurringCharge(ctx context.Context, req ChargeRequest) (*Charge, error)
    
    // Verificar status
    GetChargeStatus(ctx context.Context, chargeID string) (ChargeStatus, error)
    
    // Cancelar recorrência
    CancelRecurrence(ctx context.Context, subscriptionID string) error
}

type Subscription struct {
    ID          string
    AcademyID   string
    PlanID      string
    Status      SubscriptionStatus
    Gateway     string // "pix_auto" | "stripe"
    GatewaySubID string
    CurrentPeriodStart time.Time
    CurrentPeriodEnd   time.Time
    CreatedAt   time.Time
}

type ChargeRequest struct {
    CustomerID  string
    Amount      int64  // centavos
    Description string
    DueDate     time.Time
    Recurrence  RecurrenceConfig
}

type RecurrenceConfig struct {
    Interval    string // "monthly" | "yearly"
    MaxRetries  int
}
```

---

## Planos Sugeridos

| Plano       | Alunos | Preço/mês | Funcionalidades                    |
|-------------|--------|-----------|-------------------------------------|
| **Starter** | até 50 | R$ 99     | Check-in, Grade, Perfil            |
| **Pro**     | até 200| R$ 199    | + Loja, Relatórios                 |
| **Business**| ilimitado| R$ 399  | + API, Multi-unidade, Suporte VIP  |

---

## Fluxo de Onboarding Academia

```
1. Academia faz signup no BlackBelt
2. Trial de 14 dias (sem cartão)
3. Fim do trial → escolhe plano
4. Opção de pagamento:
   - PIX Automático (recomendado)
   - Cartão de crédito (Stripe)
5. Autoriza cobrança recorrente
6. Academia ativa!
```

---

## Próximos Passos

1. [ ] Escolher provedor PIX Automático (Efí Bank tem melhor SDK)
2. [ ] Implementar `PixAutomaticoAdapter` 
3. [ ] Implementar `StripeBillingAdapter`
4. [ ] Tabela `subscriptions` no Supabase
5. [ ] Webhooks para status de pagamento
6. [ ] Customer portal para gerenciar assinatura

---

## Referências

- [PIX Automático - Banco Central](https://www.bcb.gov.br/estabilidadefinanceira/pix-automatico)
- [Instrução Normativa BCB nº 513/2024](https://www.bcb.gov.br/estabilidadefinanceira/exibenormativo?tipo=Instru%C3%A7%C3%A3o%20Normativa%20BCB&numero=513)
- [API PIX - Bacen GitHub](https://github.com/bacen/pix-api)
- [Efí Bank - Guia Técnico](https://sejaefi.com.br/blog/pix-automatico-guia-tecnico)
- [Stripe Billing](https://stripe.com/docs/billing)

---

*Atualizado: 2026-02-08*
