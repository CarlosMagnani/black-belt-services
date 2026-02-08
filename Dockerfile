# ===========================================
# BlackBelt Backend - Multi-stage Dockerfile
# ===========================================

# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Instala dependências para compilação com CGO (necessário para alguns pacotes)
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copia arquivos de dependências primeiro (melhor cache)
COPY go.mod go.sum ./
RUN go mod download

# Copia o código fonte
COPY . .

# Compila o binário
# CGO_ENABLED=0 para binário estático
# -ldflags="-s -w" para reduzir tamanho
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /blackbelt-api ./cmd/api

# Stage 2: Runtime
FROM alpine:3.19

# Instala certificados CA (necessário para HTTPS/mTLS)
RUN apk --no-cache add ca-certificates tzdata

# Cria usuário não-root para segurança
RUN adduser -D -g '' appuser

WORKDIR /app

# Copia o binário compilado
COPY --from=builder /blackbelt-api .

# Cria diretório para certificados
RUN mkdir -p /app/certs && chown -R appuser:appuser /app

# Troca para usuário não-root
USER appuser

# Porta padrão da API
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Comando de entrada
ENTRYPOINT ["./blackbelt-api"]
