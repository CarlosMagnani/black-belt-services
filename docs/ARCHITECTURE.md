# BlackBelt Services - Backend Architecture

## Visão Geral

Backend em Go para **BlackBelt** — SaaS de gestão para academias de Jiu-Jitsu.

**Modelo:** B2B2C (assinatura mensal da academia)

**Gateways de Pagamento:**
- **PIX Automático** (Efí Bank) — principal para Brasil
- **Stripe Billing** — fallback/internacional

## Stack Tecnológica

| Componente | Tecnologia |
|------------|------------|
| **Linguagem** | Go 1.22+ |
| **Framework Web** | Chi (leve e idiomático) |
| **Database** | PostgreSQL 15+ (via Supabase) |
| **Auth** | Supabase Auth (JWT validation) |
| **PIX** | Efí Bank SDK |
| **Cartão** | Stripe Billing |
| **Migrations** | golang-migrate |
| **Testing** | testify, go-sqlmock |
| **Linting** | golangci-lint |
| **API Docs** | Swagger/OpenAPI |

## Princípios Arquiteturais

### 1. Clean Architecture (Ports & Adapters)
- Separação clara entre domínio, casos de uso e infraestrutura
- Inversão de dependências via interfaces (ports)
- Adapters implementam interfaces

### 2. API-First
- API REST padronizada
- Versionamento via path (`/api/v1/...`)
- Documentação OpenAPI

### 3. Observabilidade
- Logs estruturados (zerolog)
- Métricas (Prometheus format)
- Health checks

### 4. Segurança
- JWT validation (Supabase tokens)
- HTTPS obrigatório
- Webhook signature validation
- Rate limiting
- Input validation

## Camadas da Aplicação

```
┌─────────────────────────────────────────┐
│  Transport Layer (HTTP/REST)            │
│  - Handlers, Middlewares, Validators   │
└─────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│  Service Layer (Business Logic)         │
│  - SubscriptionService, AcademyService  │
└─────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│  Ports (Interfaces)                     │
│  - PaymentGateway, Repository, Auth     │
└─────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│  Adapters (Implementations)             │
│  - PixAutoAdapter, StripeAdapter        │
│  - SupabaseRepo, SupabaseAuth           │
└─────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│  Infrastructure                         │
│  - Efí Bank API, Stripe API, Supabase   │
└─────────────────────────────────────────┘
```

## Modelo de Pagamento: Assinatura Recorrente

### PIX Automático (Padrão Brasil) 🇧🇷

**Vantagens:**
- ✅ Não compromete limite do cartão
- ✅ Taxa ~0.5-1% (vs 3.5% cartão)
- ✅ Liquidação D+0 (instantâneo)
- ✅ Sem convênio bancário

**Fluxo:**
```
Academia escolhe plano
       ↓
Backend gera QR Code (Efí Bank)
       ↓
Dono escaneia e autoriza no banco
       ↓
Webhook confirma → subscription ativa
       ↓
Cobranças mensais automáticas
```

### Stripe Billing (Fallback/Internacional)

**Quando usar:**
- Cliente prefere cartão
- Academia fora do Brasil
- PIX Automático não disponível

**Fluxo:**
```
Academia escolhe plano
       ↓
Redirect para Stripe Checkout
       ↓
Pagamento com cartão
       ↓
Webhook confirma → subscription ativa
       ↓
Renovações automáticas via Stripe
```

## Componentes Principais

### 1. Subscription Service
- Criar/cancelar assinaturas
- Escolher gateway (PIX ou Stripe)
- Processar webhooks de ambos gateways
- Gerenciar status (trialing, active, past_due, canceled)

### 2. Academy Service
- CRUD de academias
- Associar subscription
- Verificar limites do plano (alunos, features)

### 3. Auth Middleware
- Validar JWT do Supabase
- Extrair user_id e claims
- Verificar permissões (owner, professor, student)

### 4. Webhook Handler
- Receber eventos PIX (Efí Bank)
- Receber eventos Stripe
- Idempotência
- Logging/auditoria

## Estrutura de Pastas

```
black-belt-services/
├── cmd/
│   └── api/
│       └── main.go                 # Entrypoint
├── internal/
│   ├── domain/                     # Entidades de negócio
│   │   ├── academy.go
│   │   ├── subscription.go
│   │   ├── plan.go
│   │   └── user.go
│   ├── ports/                      # Interfaces
│   │   ├── billing.go              # PaymentGateway interface
│   │   ├── repository.go           # Repository interfaces
│   │   └── auth.go                 # Auth interface
│   ├── adapters/                   # Implementações
│   │   ├── pix_automatico.go       # Efí Bank
│   │   ├── stripe_billing.go       # Stripe
│   │   ├── supabase_repo.go        # Supabase client
│   │   └── supabase_auth.go        # JWT validation
│   ├── service/                    # Lógica de negócio
│   │   ├── subscription_service.go
│   │   └── academy_service.go
│   └── handlers/                   # HTTP handlers
│       ├── subscription.go
│       ├── academy.go
│       └── webhook.go
├── pkg/
│   └── middleware/                 # Middlewares HTTP
│       ├── auth.go
│       ├── cors.go
│       └── ratelimit.go
├── migrations/                     # SQL migrations
├── docs/                           # Documentação
├── docker-compose.yml
├── Dockerfile
└── go.mod
```

## Interfaces (Ports)

### PaymentGateway

```go
// internal/ports/billing.go

type PaymentGateway interface {
    // Criar cobrança recorrente
    CreateRecurringCharge(ctx context.Context, req ChargeRequest) (*Charge, error)
    
    // Verificar status da cobrança
    GetChargeStatus(ctx context.Context, chargeID string) (*ChargeStatus, error)
    
    // Cancelar recorrência
    CancelRecurrence(ctx context.Context, subscriptionID string) error
    
    // Processar webhook
    HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
}

type ChargeRequest struct {
    CustomerID   string
    CustomerName string
    CustomerCPF  string // PIX
    CustomerEmail string
    Amount       int64  // centavos
    PlanName     string
    Recurrence   RecurrenceConfig
}

type RecurrenceConfig struct {
    Interval string // "monthly" | "yearly"
}

type Charge struct {
    ID              string
    Gateway         string // "pix_auto" | "stripe"
    Status          string
    QRCode          string // PIX only
    AuthorizationURL string // PIX only
    CheckoutURL     string // Stripe only
    ExpiresAt       time.Time
}

type WebhookEvent struct {
    Type          string
    SubscriptionID string
    Status        string
    Payload       json.RawMessage
}
```

### SubscriptionService

```go
// internal/ports/billing.go

type SubscriptionService interface {
    // Criar assinatura (escolhe gateway)
    CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*Subscription, error)
    
    // Buscar assinatura da academia
    GetByAcademyID(ctx context.Context, academyID string) (*Subscription, error)
    
    // Cancelar
    CancelSubscription(ctx context.Context, subscriptionID string) error
    
    // Processar evento de pagamento
    ProcessPaymentEvent(ctx context.Context, event *WebhookEvent) error
}

type CreateSubscriptionRequest struct {
    AcademyID   string
    PlanID      string
    Gateway     string // "pix_auto" | "stripe"
    CustomerCPF string // required for PIX
}
```

## Configuração

```env
# Server
PORT=8080
ENV=development|staging|production

# Supabase
SUPABASE_URL=https://xxx.supabase.co
SUPABASE_SERVICE_KEY=eyJ...
SUPABASE_JWT_SECRET=your-jwt-secret

# Efí Bank (PIX Automático)
EFI_CLIENT_ID=Client_Id_xxx
EFI_CLIENT_SECRET=Client_Secret_xxx
EFI_PIX_KEY=chave-pix@blackbelt.com
EFI_CERTIFICATE_PATH=./certs/efi.pem
EFI_SANDBOX=true

# Stripe
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx

# Security
API_RATE_LIMIT=100
```

## Health Check

```go
// GET /health
{
    "status": "ok",
    "version": "1.0.0",
    "database": "connected",
    "efi_bank": "connected",
    "stripe": "connected"
}
```

## Deployment

- **Container:** Docker
- **Platform:** Railway / Fly.io / Render
- **Migrations:** golang-migrate via init container
- **Secrets:** Environment variables
- **Health:** `/health` endpoint
- **Graceful Shutdown:** SIGTERM handling

## Próximos Passos

1. [x] Documentação de arquitetura
2. [ ] Setup projeto Go + Chi
3. [ ] Integração Supabase Auth
4. [ ] Implementar PaymentGateway interface
5. [ ] Adapter PIX Automático (Efí Bank)
6. [ ] Adapter Stripe Billing
7. [ ] Subscription Service
8. [ ] Webhook handlers
9. [ ] Testes unitários + integração
10. [ ] Deploy em staging

---

*Atualizado: 2026-02-08*
