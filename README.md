# Black Belt Services

Backend API em Go para marketplace de serviços de artes marciais (DojoFlow), com integração Stripe Connect para split payments entre plataforma e instrutores.

## 🎯 Visão Geral

Sistema de pagamentos que permite:
- ✅ Onboarding de instrutores via Stripe Connect
- 💳 Processamento de pagamentos (cartão, PIX, boleto)
- 💰 Split automático de receita (plataforma + instrutor)
- 🔔 Webhooks para eventos em tempo real
- 📊 Reconciliação financeira
- 🔒 Segurança e conformidade PCI-DSS

## 📚 Documentação

Toda a documentação está em [`/docs`](./docs/):

- **[ARCHITECTURE.md](./docs/ARCHITECTURE.md)** - Visão geral da arquitetura
- **[FOLDER_STRUCTURE.md](./docs/FOLDER_STRUCTURE.md)** - Estrutura de pastas e módulos
- **[DATA_MODEL.md](./docs/DATA_MODEL.md)** - Modelo de dados e migrations
- **[FLOWS.md](./docs/FLOWS.md)** - Fluxos detalhados (onboarding, payment, split, webhooks)
- **[SECURITY_CHECKLIST.md](./docs/SECURITY_CHECKLIST.md)** - Segurança e confiabilidade
- **[TOOLING.md](./docs/TOOLING.md)** - Setup de ferramentas e ambiente

## 🚀 Quick Start

### Pré-requisitos

- Go 1.21+
- Docker & Docker Compose
- Stripe CLI
- PostgreSQL 15+ (via Docker)
- Redis (via Docker)

### Setup

```bash
# 1. Clone o repositório
git clone https://github.com/seu-user/black-belt-services.git
cd black-belt-services

# 2. Copiar .env.example
cp .env.example .env
# Editar .env com suas credenciais Stripe

# 3. Instalar ferramentas de desenvolvimento
make install-tools

# 4. Subir banco de dados e Redis
make docker-up

# 5. Rodar migrations
make migrate-up

# 6. Iniciar servidor (com hot reload)
make dev
```

Em outro terminal:
```bash
# 7. Stripe webhook forwarding
make stripe-listen
```

### Verificar

```bash
# Health check
curl http://localhost:8080/health
```

## 🏗️ Estrutura do Projeto

```
black-belt-services/
├── cmd/                    # Entry points
│   ├── api/               # API server
│   └── migrate/           # Migrations CLI
├── internal/              # Código privado
│   ├── config/            # Configuração
│   ├── domain/            # Entidades de domínio
│   ├── repository/        # Interfaces de persistência
│   ├── storage/           # Implementações (Postgres, Redis)
│   ├── service/           # Lógica de negócio
│   │   ├── connect/       # Stripe Connect
│   │   ├── payment/       # Pagamentos
│   │   ├── transfer/      # Transfers/Splits
│   │   ├── webhook/       # Webhooks
│   │   └── order/         # Pedidos
│   ├── handler/           # HTTP handlers
│   ├── stripe/            # Stripe client wrapper
│   └── server/            # HTTP server
├── migrations/            # Database migrations
├── scripts/               # Scripts utilitários
├── test/                  # Integration tests
└── docs/                  # Documentação
```

## 🔑 Principais Features

### 1. Stripe Connect - Onboarding de Instrutores

```go
// POST /v1/connect/accounts
{
  "user_id": "uuid",
  "type": "express",
  "country": "BR",
  "email": "instrutor@example.com"
}

// POST /v1/connect/accounts/:id/link
{
  "return_url": "https://app.com/connect/return",
  "refresh_url": "https://app.com/connect/retry"
}
```

### 2. Pagamentos

```go
// POST /v1/orders
{
  "instructor_user_id": "uuid",
  "amount_total": 10000,  // R$100,00
  "description": "Aula de Jiu-Jitsu"
}

// POST /v1/payments
{
  "order_id": "uuid",
  "payment_method_types": ["card"]
}
```

### 3. Split Automático

Após pagamento confirmado:
- 85% para o instrutor → Transfer para Connected Account
- 15% para a plataforma → Fica no balance

### 4. Webhooks

```
POST /v1/webhooks/stripe
```

Eventos tratados:
- `account.updated` - Atualizar status do Connected Account
- `payment_intent.succeeded` - Confirmar pagamento
- `transfer.paid` - Confirmar transfer

## 🧪 Testes

```bash
# Unit tests
make test

# Integration tests
make test-integration

# Coverage report
make test-coverage
# Abre coverage.html no browser
```

## 🔒 Segurança

- ✅ Validação de assinatura de webhooks
- ✅ Idempotency keys (evita duplicação)
- ✅ Logs estruturados (zerolog)
- ✅ Rate limiting
- ✅ Input validation
- ✅ HTTPS obrigatório em produção
- ✅ Secrets gerenciados via env vars

Ver [SECURITY_CHECKLIST.md](./docs/SECURITY_CHECKLIST.md) para detalhes.

## 📊 Reconciliação

Job diário que compara:
- Pagamentos: DB vs. Stripe
- Transfers: DB vs. Stripe
- Saldos: Plataforma vs. Connected Accounts

Alerta sobre discrepâncias.

## 🛠️ Comandos Úteis

```bash
# Desenvolvimento
make dev              # Servidor com hot reload
make test             # Rodar testes
make lint             # Linter
make lint-fix         # Auto-fix linter

# Database
make migrate-up       # Aplicar migrations
make migrate-down     # Rollback
make migrate-create name=add_feature  # Nova migration

# Docker
make docker-up        # Subir Postgres + Redis
make docker-down      # Parar
make docker-logs      # Ver logs

# Stripe
make stripe-listen    # Webhook forwarding

# Limpeza
make clean            # Limpar build artifacts
```

## 📦 Dependências Principais

```go
github.com/gin-gonic/gin              // HTTP framework
github.com/stripe/stripe-go/v76       // Stripe SDK
github.com/jackc/pgx/v5               // PostgreSQL
github.com/redis/go-redis/v9          // Redis
github.com/golang-migrate/migrate/v4  // Migrations
github.com/rs/zerolog                 // Logging
github.com/golang-jwt/jwt/v5          // JWT
```

## 🌍 Variáveis de Ambiente

Ver [.env.example](./.env.example):

```bash
# Server
PORT=8080
ENV=development

# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/dojoflow_dev

# Redis
REDIS_URL=redis://localhost:6379/0

# Stripe
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...

# Security
JWT_SECRET=your-secret
API_RATE_LIMIT=100
```

## 🚢 Deploy

### Docker

```bash
# Build image
docker build -t black-belt-services .

# Run
docker run -p 8080:8080 --env-file .env black-belt-services
```

### Docker Compose (Produção)

```yaml
version: '3.8'
services:
  api:
    image: black-belt-services:latest
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
    depends_on:
      - postgres
      - redis
```

## 📈 Roadmap

- [ ] Implementar estrutura base (cmd, internal)
- [ ] Setup de migrations e seeds
- [ ] Implementar domínio e repositórios
- [ ] Criar services (Connect, Payment, Transfer, Webhook)
- [ ] Implementar handlers e middlewares
- [ ] Testes unitários (>80% coverage)
- [ ] Testes de integração
- [ ] Documentação OpenAPI/Swagger
- [ ] CI/CD pipeline
- [ ] Monitoring (Prometheus, Grafana)
- [ ] Deploy em staging
- [ ] Deploy em produção

## 🤝 Contribuindo

1. Fork o projeto
2. Crie uma branch (`git checkout -b feature/amazing-feature`)
3. Commit suas mudanças (`git commit -m 'Add amazing feature'`)
4. Push para a branch (`git push origin feature/amazing-feature`)
5. Abra um Pull Request

**Code Style:**
- Rodar `make lint` antes de commitar
- Cobertura de testes >80%
- Documentar funções públicas

## 📝 License

Este projeto está sob a licença MIT. Ver [LICENSE](./LICENSE).

## 🔗 Links Úteis

- [Stripe Connect Documentation](https://stripe.com/docs/connect)
- [Stripe API Reference](https://stripe.com/docs/api)
- [Stripe CLI](https://stripe.com/docs/stripe-cli)
- [Go Best Practices](https://github.com/golang-standards/project-layout)

---

**Desenvolvido com ❤️ para a comunidade de artes marciais**
