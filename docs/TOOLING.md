# Tooling e Setup - BlackBelt Services

## Ferramentas Essenciais

### 1. Efí Bank SDK (PIX Automático)

SDK oficial para integração com PIX Automático.

#### Instalação

```bash
go get github.com/efipay/sdk-go-apis-efi
```

#### Configuração

```bash
# .env
EFI_CLIENT_ID=Client_Id_xxx
EFI_CLIENT_SECRET=Client_Secret_xxx
EFI_PIX_KEY=pix@blackbelt.com.br
EFI_CERTIFICATE_PATH=./certs/efi.pem
EFI_SANDBOX=true
```

#### Obter Certificado

1. Acessar [Efí Bank Portal](https://app.sejaefi.com.br/)
2. Menu → API → Minhas Aplicações
3. Criar aplicação de produção
4. Baixar certificado `.p12`
5. Converter para `.pem`:
   ```bash
   openssl pkcs12 -in certificado.p12 -out certs/efi.pem -nodes
   ```

#### Uso Básico

```go
package main

import (
    "fmt"
    efi "github.com/efipay/sdk-go-apis-efi/sdk"
)

func main() {
    client := efi.NewClient(efi.Config{
        ClientID:     os.Getenv("EFI_CLIENT_ID"),
        ClientSecret: os.Getenv("EFI_CLIENT_SECRET"),
        Certificate:  os.Getenv("EFI_CERTIFICATE_PATH"),
        Sandbox:      os.Getenv("EFI_SANDBOX") == "true",
    })
    
    // Criar cobrança PIX
    body := map[string]interface{}{
        "calendario": map[string]interface{}{
            "expiracao": 3600,
        },
        "devedor": map[string]string{
            "cpf":  "12345678900",
            "nome": "João da Silva",
        },
        "valor": map[string]string{
            "original": "99.00",
        },
        "chave": os.Getenv("EFI_PIX_KEY"),
    }
    
    resp, err := client.PixCreateCharge(body)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("QR Code: %s\n", resp["pixCopiaECola"])
    fmt.Printf("txid: %s\n", resp["txid"])
}
```

#### Testar Webhook (Sandbox)

```bash
# Simular pagamento PIX recebido
curl -X POST http://localhost:8080/api/webhooks/pix \
  -H "Content-Type: application/json" \
  -d '{
    "evento": "pix_auto_pagamento_recebido",
    "txid": "xxx123",
    "valor": "99.00"
  }'
```

---

### 2. Stripe CLI

Para desenvolvimento local e testes de webhooks Stripe.

#### Instalação

**Linux/WSL:**
```bash
curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg
echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | sudo tee -a /etc/apt/sources.list.d/stripe.list
sudo apt update
sudo apt install stripe
```

**macOS:**
```bash
brew install stripe/stripe-cli/stripe
```

**Windows:**
```powershell
scoop bucket add stripe https://github.com/stripe/scoop-stripe-cli.git
scoop install stripe
```

#### Configuração

```bash
# Login (abre browser)
stripe login

# Webhook forwarding
stripe listen --forward-to localhost:8080/api/webhooks/stripe

# Output:
# > Ready! Your webhook signing secret is whsec_... (add to .env)
```

#### Uso

```bash
# Trigger eventos manualmente
stripe trigger checkout.session.completed
stripe trigger invoice.paid
stripe trigger customer.subscription.deleted

# Ver logs
stripe logs tail

# Criar produto/price
stripe products create --name="BlackBelt Pro" --description="Plano Pro"
stripe prices create --product=prod_xxx --unit-amount=19900 --currency=brl --recurring[interval]=month
```

---

### 3. golang-migrate

Gerenciamento de migrations de banco de dados.

#### Instalação

```bash
# Go install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Linux binary
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

#### Uso

```bash
# Criar migration
migrate create -ext sql -dir migrations -seq add_subscriptions

# Rodar migrations
migrate -path ./migrations -database "$DATABASE_URL" up

# Rollback
migrate -path ./migrations -database "$DATABASE_URL" down 1

# Ver versão
migrate -path ./migrations -database "$DATABASE_URL" version

# Forçar versão (se erro)
migrate -path ./migrations -database "$DATABASE_URL" force 2
```

---

### 4. Supabase CLI

Para desenvolvimento local e sync com projeto Supabase.

#### Instalação

```bash
# npm
npm install -g supabase

# Homebrew
brew install supabase/tap/supabase
```

#### Uso

```bash
# Login
supabase login

# Inicializar projeto
supabase init

# Linkar com projeto remoto
supabase link --project-ref your-project-ref

# Gerar tipos TypeScript (para mobile)
supabase gen types typescript --project-id your-project-id > types/supabase.ts

# Rodar local
supabase start

# Pull migrations do remoto
supabase db pull

# Push migrations para remoto
supabase db push
```

---

### 5. Docker Compose

Para desenvolvimento local com todos os serviços.

#### docker-compose.yml

```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - ENV=development
      - DATABASE_URL=postgresql://postgres:postgres@db:5432/blackbelt?sslmode=disable
      - SUPABASE_URL=${SUPABASE_URL}
      - SUPABASE_SERVICE_KEY=${SUPABASE_SERVICE_KEY}
      - EFI_CLIENT_ID=${EFI_CLIENT_ID}
      - EFI_CLIENT_SECRET=${EFI_CLIENT_SECRET}
      - EFI_PIX_KEY=${EFI_PIX_KEY}
      - EFI_CERTIFICATE_PATH=/app/certs/efi.pem
      - EFI_SANDBOX=true
      - STRIPE_SECRET_KEY=${STRIPE_SECRET_KEY}
      - STRIPE_WEBHOOK_SECRET=${STRIPE_WEBHOOK_SECRET}
    volumes:
      - ./certs:/app/certs:ro
    depends_on:
      - db

  db:
    image: postgres:15-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: blackbelt
    volumes:
      - postgres_data:/var/lib/postgresql/data

  migrate:
    image: migrate/migrate
    volumes:
      - ./migrations:/migrations
    command: ["-path", "/migrations", "-database", "postgresql://postgres:postgres@db:5432/blackbelt?sslmode=disable", "up"]
    depends_on:
      - db

volumes:
  postgres_data:
```

#### Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api ./cmd/api

FROM alpine:3.19

RUN apk --no-cache add ca-certificates

COPY --from=builder /bin/api /bin/api

EXPOSE 8080

CMD ["/bin/api"]
```

---

### 6. Air (Hot Reload)

Recarregar automaticamente durante desenvolvimento.

#### Instalação

```bash
go install github.com/cosmtrek/air@latest
```

#### Configuração (.air.toml)

```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/main ./cmd/api"
bin = "tmp/main"
include_ext = ["go", "tpl", "tmpl", "html"]
exclude_dir = ["assets", "tmp", "vendor", "docs"]
delay = 1000

[log]
time = false

[color]
main = "magenta"
watcher = "cyan"
build = "yellow"
runner = "green"
```

#### Uso

```bash
air
```

---

### 7. golangci-lint

Linting e análise estática.

#### Instalação

```bash
# Go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Homebrew
brew install golangci-lint
```

#### Configuração (.golangci.yml)

```yaml
linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - gofmt
    - goimports
    - misspell

linters-settings:
  gofmt:
    simplify: true

run:
  timeout: 5m
  skip-dirs:
    - vendor
    - docs
```

#### Uso

```bash
golangci-lint run ./...
```

---

## Environment Variables

### .env.example

```bash
# Server
PORT=8080
ENV=development

# Supabase
SUPABASE_URL=https://xxx.supabase.co
SUPABASE_SERVICE_KEY=eyJ...
SUPABASE_JWT_SECRET=your-jwt-secret

# Database (if using local postgres instead of Supabase)
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/blackbelt?sslmode=disable

# Efí Bank (PIX Automático)
EFI_CLIENT_ID=Client_Id_xxx
EFI_CLIENT_SECRET=Client_Secret_xxx
EFI_PIX_KEY=pix@blackbelt.com.br
EFI_CERTIFICATE_PATH=./certs/efi.pem
EFI_SANDBOX=true

# Stripe
STRIPE_SECRET_KEY=sk_test_xxx
STRIPE_WEBHOOK_SECRET=whsec_xxx

# Email (Resend)
RESEND_API_KEY=re_xxx

# Security
API_RATE_LIMIT=100
```

---

## Makefile

```makefile
.PHONY: build run test lint migrate-up migrate-down docker-up docker-down stripe-listen

# Development
run:
	air

build:
	go build -o bin/api ./cmd/api

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

# Migrations
migrate-up:
	migrate -path ./migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path ./migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f api

# Stripe
stripe-listen:
	stripe listen --forward-to localhost:8080/api/webhooks/stripe

# Supabase
supabase-types:
	supabase gen types typescript --project-id $(SUPABASE_PROJECT_ID) > ../black-belt-app/src/types/supabase.ts

# Clean
clean:
	rm -rf bin/ tmp/
```

---

## Scripts

### scripts/setup.sh

```bash
#!/bin/bash
set -e

echo "🚀 Setting up BlackBelt Services..."

# Check dependencies
command -v go >/dev/null 2>&1 || { echo "Go is required but not installed."; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "Docker is required but not installed."; exit 1; }

# Copy env file
if [ ! -f .env ]; then
    cp .env.example .env
    echo "📝 Created .env file. Please fill in your credentials."
fi

# Create certs directory
mkdir -p certs
echo "📁 Created certs/ directory. Add your Efí Bank certificate here."

# Install tools
echo "📦 Installing Go tools..."
go install github.com/cosmtrek/air@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Download dependencies
echo "📦 Downloading Go modules..."
go mod download

echo "✅ Setup complete! Run 'make run' to start the server."
```

### scripts/seed.sh

```bash
#!/bin/bash

# Seed subscription plans
psql "$DATABASE_URL" << 'EOF'
INSERT INTO subscription_plans (name, slug, price_monthly, max_students, max_professors, features) VALUES
('Starter', 'starter', 9900, 50, 2, '["checkin", "schedule", "profiles"]'),
('Pro', 'pro', 19900, 200, 5, '["checkin", "schedule", "profiles", "analytics", "store"]'),
('Business', 'business', 39900, NULL, NULL, '["checkin", "schedule", "profiles", "analytics", "store", "api", "multi_location", "priority_support"]')
ON CONFLICT (slug) DO NOTHING;
EOF

echo "✅ Seed complete!"
```

---

## Referências

- [Efí Bank SDK Go](https://github.com/efipay/sdk-go-apis-efi)
- [Efí Bank Docs](https://dev.efipay.com.br/)
- [Stripe Go SDK](https://github.com/stripe/stripe-go)
- [Stripe Docs](https://stripe.com/docs)
- [Chi Router](https://github.com/go-chi/chi)
- [golang-migrate](https://github.com/golang-migrate/migrate)

---

*Atualizado: 2026-02-08*
