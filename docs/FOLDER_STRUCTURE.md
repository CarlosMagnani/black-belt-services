# Estrutura de Pastas - BlackBelt Services

## Árvore Completa

```
black-belt-services/
├── cmd/
│   ├── api/                    # API server principal
│   │   └── main.go
│   ├── migrate/                # CLI para migrations
│   │   └── main.go
│   └── cron/                   # Jobs agendados (trial check, etc)
│       └── main.go
│
├── internal/                   # Código privado da aplicação
│   ├── config/                 # Configuração e env vars
│   │   ├── config.go
│   │   ├── efi.go              # Config Efí Bank (PIX)
│   │   └── stripe.go           # Config Stripe
│   │
│   ├── domain/                 # Entidades de domínio
│   │   ├── academy.go
│   │   ├── subscription.go
│   │   ├── plan.go
│   │   ├── payment.go
│   │   ├── webhook_event.go
│   │   └── errors.go           # Domain errors
│   │
│   ├── ports/                  # Interfaces (Clean Architecture)
│   │   ├── billing.go          # PaymentGateway interface
│   │   ├── repository.go       # Repository interfaces
│   │   ├── auth.go             # Auth interface
│   │   └── notifier.go         # Email/Push notifications
│   │
│   ├── adapters/               # Implementações de ports
│   │   ├── pix_automatico.go   # Efí Bank adapter
│   │   ├── stripe_billing.go   # Stripe adapter
│   │   ├── supabase_repo.go    # Supabase client
│   │   ├── supabase_auth.go    # JWT validation
│   │   └── resend_notifier.go  # Email via Resend
│   │
│   ├── service/                # Lógica de negócio
│   │   ├── academy_service.go
│   │   ├── subscription_service.go
│   │   └── webhook_service.go
│   │
│   ├── handlers/               # HTTP handlers
│   │   ├── health.go
│   │   ├── academy.go
│   │   ├── subscription.go
│   │   └── webhook.go
│   │
│   ├── middleware/             # HTTP middlewares
│   │   ├── auth.go             # JWT validation
│   │   ├── cors.go
│   │   ├── logging.go
│   │   ├── ratelimit.go
│   │   └── recovery.go
│   │
│   └── server/                 # HTTP server setup
│       ├── server.go
│       └── router.go
│
├── pkg/                        # Código reutilizável
│   ├── logger/
│   │   └── logger.go           # zerolog wrapper
│   ├── validator/
│   │   └── validator.go        # Input validation
│   └── response/
│       └── response.go         # JSON response helpers
│
├── migrations/                 # SQL migrations
│   ├── 000001_init_schema.up.sql
│   ├── 000001_init_schema.down.sql
│   └── ...
│
├── docs/                       # Documentação
│   ├── EXECUTIVE_SUMMARY.md
│   ├── ARCHITECTURE.md
│   ├── DATA_MODEL.md
│   ├── FLOWS.md
│   ├── MONETIZATION.md
│   ├── FOLDER_STRUCTURE.md
│   ├── SECURITY_CHECKLIST.md
│   └── TOOLING.md
│
├── certs/                      # Certificados (gitignore!)
│   └── efi.pem                 # Certificado Efí Bank
│
├── scripts/                    # Scripts auxiliares
│   ├── setup.sh                # Setup inicial
│   └── seed.sh                 # Seed de dados
│
├── .env.example                # Template de env vars
├── .gitignore
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## Detalhamento por Camada

### 1. `cmd/` - Entrypoints

Cada subpasta é um binário separado.

**api/main.go:**
```go
package main

import (
    "black-belt-services/internal/config"
    "black-belt-services/internal/server"
)

func main() {
    cfg := config.Load()
    srv := server.New(cfg)
    srv.Run()
}
```

**cron/main.go:**
```go
package main

// Job que roda diariamente para verificar trials expirados
func main() {
    cfg := config.Load()
    svc := service.NewSubscriptionService(cfg)
    svc.CheckExpiredTrials(context.Background())
}
```

---

### 2. `internal/domain/` - Entidades

Entidades de negócio puras, sem dependências externas.

**subscription.go:**
```go
package domain

import "time"

type SubscriptionStatus string

const (
    StatusTrialing SubscriptionStatus = "trialing"
    StatusActive   SubscriptionStatus = "active"
    StatusPastDue  SubscriptionStatus = "past_due"
    StatusCanceled SubscriptionStatus = "canceled"
    StatusExpired  SubscriptionStatus = "expired"
)

type PaymentGateway string

const (
    GatewayPixAuto PaymentGateway = "pix_auto"
    GatewayStripe  PaymentGateway = "stripe"
)

type Subscription struct {
    ID                   string
    AcademyID            string
    PlanID               string
    Status               SubscriptionStatus
    PaymentGateway       PaymentGateway
    
    // Trial
    TrialStartDate       time.Time
    TrialEndDate         time.Time
    
    // PIX Automático
    PixAuthorizationID   string
    PixRecurrenceID      string
    PixCustomerCPF       string
    
    // Stripe
    StripeCustomerID     string
    StripeSubscriptionID string
    
    // Período atual
    CurrentPeriodStart   time.Time
    CurrentPeriodEnd     time.Time
    
    // Cancelamento
    CanceledAt           *time.Time
    CancelReason         string
    
    CreatedAt            time.Time
    UpdatedAt            time.Time
}

func (s *Subscription) IsActive() bool {
    return s.Status == StatusActive || s.Status == StatusTrialing
}

func (s *Subscription) IsTrialExpired() bool {
    return s.Status == StatusTrialing && time.Now().After(s.TrialEndDate)
}
```

---

### 3. `internal/ports/` - Interfaces

Contratos que os adapters devem implementar.

**billing.go:**
```go
package ports

import (
    "context"
    "black-belt-services/internal/domain"
)

// PaymentGateway abstrai PIX Automático e Stripe
type PaymentGateway interface {
    CreateRecurringCharge(ctx context.Context, req ChargeRequest) (*Charge, error)
    GetChargeStatus(ctx context.Context, chargeID string) (*ChargeStatus, error)
    CancelRecurrence(ctx context.Context, recurrenceID string) error
    HandleWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
}

type ChargeRequest struct {
    CustomerID    string
    CustomerName  string
    CustomerCPF   string  // PIX only
    CustomerEmail string
    Amount        int64   // centavos
    PlanName      string
    Interval      string  // "monthly" | "yearly"
}

type Charge struct {
    ID               string
    Gateway          string
    Status           string
    QRCode           string // PIX only
    AuthorizationURL string // PIX only
    CheckoutURL      string // Stripe only
    ExpiresAt        time.Time
}
```

**repository.go:**
```go
package ports

type SubscriptionRepository interface {
    Create(ctx context.Context, sub *domain.Subscription) error
    Update(ctx context.Context, sub *domain.Subscription) error
    FindByID(ctx context.Context, id string) (*domain.Subscription, error)
    FindByAcademyID(ctx context.Context, academyID string) (*domain.Subscription, error)
    FindByGatewayID(ctx context.Context, gatewayID string) (*domain.Subscription, error)
    FindExpiredTrials(ctx context.Context) ([]*domain.Subscription, error)
    FindTrialsExpiringSoon(ctx context.Context, days int) ([]*domain.Subscription, error)
}

type AcademyRepository interface {
    Create(ctx context.Context, academy *domain.Academy) error
    Update(ctx context.Context, academy *domain.Academy) error
    FindByID(ctx context.Context, id string) (*domain.Academy, error)
    FindByOwnerID(ctx context.Context, ownerID string) (*domain.Academy, error)
    FindByInviteCode(ctx context.Context, code string) (*domain.Academy, error)
}
```

---

### 4. `internal/adapters/` - Implementações

**pix_automatico.go:**
```go
package adapters

import (
    "context"
    efi "github.com/efipay/sdk-go-apis-efi/sdk"
    "black-belt-services/internal/ports"
)

type PixAutomaticoAdapter struct {
    client *efi.Client
    pixKey string
}

func NewPixAutomaticoAdapter(cfg *config.EfiConfig) *PixAutomaticoAdapter {
    client := efi.NewClient(efi.Config{
        ClientID:     cfg.ClientID,
        ClientSecret: cfg.ClientSecret,
        Certificate:  cfg.CertificatePath,
        Sandbox:      cfg.Sandbox,
    })
    
    return &PixAutomaticoAdapter{
        client: client,
        pixKey: cfg.PixKey,
    }
}

func (p *PixAutomaticoAdapter) CreateRecurringCharge(ctx context.Context, req ports.ChargeRequest) (*ports.Charge, error) {
    body := map[string]interface{}{
        "calendario": map[string]interface{}{
            "expiracao": 3600,
        },
        "devedor": map[string]string{
            "cpf":  req.CustomerCPF,
            "nome": req.CustomerName,
        },
        "valor": map[string]string{
            "original": fmt.Sprintf("%.2f", float64(req.Amount)/100),
        },
        "chave": p.pixKey,
    }
    
    resp, err := p.client.PixCreateCharge(body)
    if err != nil {
        return nil, err
    }
    
    return &ports.Charge{
        ID:               resp["txid"].(string),
        Gateway:          "pix_auto",
        Status:           "pending_authorization",
        QRCode:           resp["pixCopiaECola"].(string),
        AuthorizationURL: resp["linkVisualizacao"].(string),
    }, nil
}
```

**stripe_billing.go:**
```go
package adapters

import (
    "context"
    "github.com/stripe/stripe-go/v84"
    "github.com/stripe/stripe-go/v84/checkout/session"
)

type StripeBillingAdapter struct {
    webhookSecret string
}

func NewStripeBillingAdapter(cfg *config.StripeConfig) *StripeBillingAdapter {
    stripe.Key = cfg.SecretKey
    return &StripeBillingAdapter{
        webhookSecret: cfg.WebhookSecret,
    }
}

func (s *StripeBillingAdapter) CreateRecurringCharge(ctx context.Context, req ports.ChargeRequest) (*ports.Charge, error) {
    params := &stripe.CheckoutSessionParams{
        Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
        LineItems: []*stripe.CheckoutSessionLineItemParams{
            {
                Price:    stripe.String(req.StripePriceID),
                Quantity: stripe.Int64(1),
            },
        },
        SuccessURL: stripe.String(req.SuccessURL),
        CancelURL:  stripe.String(req.CancelURL),
        CustomerEmail: stripe.String(req.CustomerEmail),
        ClientReferenceID: stripe.String(req.AcademyID),
    }
    
    sess, err := session.New(params)
    if err != nil {
        return nil, err
    }
    
    return &ports.Charge{
        ID:          sess.ID,
        Gateway:     "stripe",
        Status:      "pending",
        CheckoutURL: sess.URL,
    }, nil
}
```

---

### 5. `internal/service/` - Lógica de Negócio

**subscription_service.go:**
```go
package service

type SubscriptionService struct {
    repo       ports.SubscriptionRepository
    academyRepo ports.AcademyRepository
    pixAdapter ports.PaymentGateway
    stripeAdapter ports.PaymentGateway
    notifier   ports.Notifier
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, req CreateSubscriptionRequest) (*Charge, error) {
    // 1. Buscar academia
    academy, err := s.academyRepo.FindByID(ctx, req.AcademyID)
    if err != nil {
        return nil, err
    }
    
    // 2. Verificar se já tem subscription ativa
    existing, _ := s.repo.FindByAcademyID(ctx, req.AcademyID)
    if existing != nil && existing.IsActive() {
        return nil, ErrSubscriptionAlreadyActive
    }
    
    // 3. Escolher gateway
    var gateway ports.PaymentGateway
    if req.Gateway == "pix_auto" {
        gateway = s.pixAdapter
    } else {
        gateway = s.stripeAdapter
    }
    
    // 4. Criar cobrança
    charge, err := gateway.CreateRecurringCharge(ctx, ports.ChargeRequest{
        CustomerName:  academy.Name,
        CustomerEmail: academy.Email,
        CustomerCPF:   req.CPF, // PIX only
        Amount:        req.Plan.PriceMonthly,
        PlanName:      req.Plan.Name,
    })
    if err != nil {
        return nil, err
    }
    
    return charge, nil
}
```

---

### 6. `internal/handlers/` - HTTP

**subscription.go:**
```go
package handlers

func (h *SubscriptionHandler) CreatePixSubscription(w http.ResponseWriter, r *http.Request) {
    var req struct {
        PlanID string `json:"plan_id"`
        CPF    string `json:"cpf"`
    }
    json.NewDecoder(r.Body).Decode(&req)
    
    // Get academy from auth context
    academyID := r.Context().Value("academy_id").(string)
    
    charge, err := h.subscriptionService.CreateSubscription(r.Context(), CreateSubscriptionRequest{
        AcademyID: academyID,
        PlanID:    req.PlanID,
        Gateway:   "pix_auto",
        CPF:       req.CPF,
    })
    if err != nil {
        response.Error(w, err)
        return
    }
    
    response.JSON(w, 200, charge)
}
```

---

## Makefile

```makefile
.PHONY: build run test migrate-up migrate-down docker-up

# Build
build:
	go build -o bin/api ./cmd/api

# Run
run:
	go run ./cmd/api

# Test
test:
	go test -v ./...

# Migrations
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

# Stripe CLI (webhook forwarding)
stripe-listen:
	stripe listen --forward-to localhost:8080/api/webhooks/stripe
```

---

*Atualizado: 2026-02-08*
