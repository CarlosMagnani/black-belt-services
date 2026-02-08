# Checklist de Segurança - BlackBelt Services

## 1. Validação de Webhooks

### PIX Automático (Efí Bank)

#### Verificar mTLS (Mutual TLS)
A Efí Bank usa certificado mTLS para autenticar webhooks.

```go
func (h *WebhookHandler) HandlePixWebhook(w http.ResponseWriter, r *http.Request) {
    // 1. Verificar certificado do cliente (mTLS)
    if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
        http.Error(w, "Certificate required", 401)
        return
    }
    
    // 2. Validar que o certificado é da Efí Bank
    cert := r.TLS.PeerCertificates[0]
    if !isValidEfiCertificate(cert) {
        http.Error(w, "Invalid certificate", 401)
        return
    }
    
    // 3. Processar webhook...
}

func isValidEfiCertificate(cert *x509.Certificate) bool {
    // Verificar CN ou fingerprint do certificado Efí
    return strings.Contains(cert.Subject.CommonName, "efipay") ||
           strings.Contains(cert.Issuer.CommonName, "Gerencianet")
}
```

#### Alternativa: Validar IP de origem

```go
var efiAllowedIPs = []string{
    "34.193.116.226",
    "52.7.213.171",
    // IPs da Efí Bank (verificar documentação atual)
}

func isEfiIP(r *http.Request) bool {
    ip := r.Header.Get("X-Forwarded-For")
    if ip == "" {
        ip, _, _ = net.SplitHostPort(r.RemoteAddr)
    }
    
    for _, allowed := range efiAllowedIPs {
        if ip == allowed {
            return true
        }
    }
    return false
}
```

### Stripe

#### Validar assinatura

```go
import (
    "github.com/stripe/stripe-go/v84/webhook"
)

func (h *WebhookHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
    payload, _ := io.ReadAll(r.Body)
    signature := r.Header.Get("Stripe-Signature")
    
    event, err := webhook.ConstructEvent(
        payload,
        signature,
        h.config.StripeWebhookSecret,
    )
    if err != nil {
        h.logger.Warn().Err(err).Msg("Invalid Stripe signature")
        http.Error(w, "Invalid signature", 400)
        return
    }
    
    // Processar evento...
}
```

### Checklist

- [ ] PIX: Validar certificado mTLS ou IP de origem
- [ ] Stripe: Usar `webhook.ConstructEvent()` do SDK
- [ ] Armazenar secrets de forma segura (env vars, não hardcoded)
- [ ] Usar secrets diferentes por ambiente (dev/staging/prod)
- [ ] Logar tentativas de validação falhas
- [ ] Rate limit no endpoint de webhook

---

## 2. Idempotência

### Por que é crítico?
- Previne processamento duplicado de webhooks
- Permite retries seguros
- Garante exactly-once semantics

### Implementação com Redis

```go
func (s *WebhookService) ProcessEvent(ctx context.Context, event WebhookEvent) error {
    key := fmt.Sprintf("webhook:%s", event.EventID)
    
    // Tentar adquirir lock
    acquired, err := s.redis.SetNX(ctx, key, "processing", 24*time.Hour).Result()
    if err != nil {
        return err
    }
    
    if !acquired {
        s.logger.Info().Str("event_id", event.EventID).Msg("Event already processed")
        return nil // Já processado, retorna sucesso
    }
    
    // Processar evento
    if err := s.handleEvent(ctx, event); err != nil {
        // Falhou: remover lock para permitir retry
        s.redis.Del(ctx, key)
        return err
    }
    
    // Sucesso: atualizar valor
    s.redis.Set(ctx, key, "processed", 24*time.Hour)
    return nil
}
```

### Implementação com PostgreSQL

```go
func (s *WebhookService) ProcessEvent(ctx context.Context, event WebhookEvent) error {
    // Tentar inserir (unique constraint previne duplicatas)
    _, err := s.db.ExecContext(ctx, `
        INSERT INTO webhook_events (event_id, gateway, event_type, payload, status)
        VALUES ($1, $2, $3, $4, 'processing')
        ON CONFLICT (event_id) DO NOTHING
    `, event.EventID, event.Gateway, event.Type, event.Payload)
    
    if err != nil {
        return err
    }
    
    // Verificar se inseriu (se não, já existe)
    var status string
    s.db.QueryRowContext(ctx, `
        SELECT status FROM webhook_events WHERE event_id = $1
    `, event.EventID).Scan(&status)
    
    if status != "processing" {
        return nil // Já processado
    }
    
    // Processar...
}
```

### Checklist

- [ ] Webhook events salvos no banco antes de processar
- [ ] Unique constraint no event_id
- [ ] Status tracking (pending → processing → processed/failed)
- [ ] TTL para limpeza de eventos antigos
- [ ] Retry count para limitar tentativas

---

## 3. Autenticação JWT (Supabase)

### Validar token em cada request

```go
package middleware

import (
    "github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Extrair token do header
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Missing authorization header", 401)
                return
            }
            
            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            
            // 2. Validar e parsear token
            token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
                if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                    return nil, fmt.Errorf("unexpected signing method")
                }
                return []byte(jwtSecret), nil
            })
            
            if err != nil || !token.Valid {
                http.Error(w, "Invalid token", 401)
                return
            }
            
            // 3. Extrair claims
            claims, ok := token.Claims.(jwt.MapClaims)
            if !ok {
                http.Error(w, "Invalid claims", 401)
                return
            }
            
            // 4. Verificar expiração
            exp, _ := claims.GetExpirationTime()
            if exp != nil && exp.Before(time.Now()) {
                http.Error(w, "Token expired", 401)
                return
            }
            
            // 5. Adicionar user_id ao context
            userID := claims["sub"].(string)
            ctx := context.WithValue(r.Context(), "user_id", userID)
            
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### Verificar role/permissões

```go
func RequireRole(roles ...string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            userRole := r.Context().Value("role").(string)
            
            for _, role := range roles {
                if userRole == role {
                    next.ServeHTTP(w, r)
                    return
                }
            }
            
            http.Error(w, "Forbidden", 403)
        })
    }
}

// Uso
r.With(RequireRole("owner")).Post("/api/subscriptions", handler.CreateSubscription)
```

### Checklist

- [ ] Todos os endpoints protegidos (exceto webhooks e health)
- [ ] JWT secret armazenado de forma segura
- [ ] Verificar expiração do token
- [ ] Verificar role antes de ações sensíveis
- [ ] Logar tentativas de acesso não autorizado

---

## 4. Rate Limiting

### Por IP

```go
import (
    "github.com/go-chi/httprate"
)

func RateLimitMiddleware(requestsPerMinute int) func(http.Handler) http.Handler {
    return httprate.LimitByIP(requestsPerMinute, time.Minute)
}

// Uso
r.Use(RateLimitMiddleware(100))
```

### Por usuário autenticado

```go
func RateLimitByUser(requestsPerMinute int) func(http.Handler) http.Handler {
    return httprate.Limit(
        requestsPerMinute,
        time.Minute,
        httprate.WithKeyFuncs(func(r *http.Request) (string, error) {
            userID := r.Context().Value("user_id")
            if userID == nil {
                return httprate.KeyByIP(r)
            }
            return userID.(string), nil
        }),
    )
}
```

### Limites recomendados

| Endpoint | Limite | Justificativa |
|----------|--------|---------------|
| `/api/auth/*` | 10/min | Prevenir brute force |
| `/api/subscriptions/*` | 20/min | Operações sensíveis |
| `/api/webhooks/*` | 200/min | Webhooks legítimos |
| `/api/*` (geral) | 100/min | Uso normal |

---

## 5. Input Validation

### Usar go-playground/validator

```go
import (
    "github.com/go-playground/validator/v10"
)

var validate = validator.New()

type CreateSubscriptionRequest struct {
    PlanID string `json:"plan_id" validate:"required,uuid"`
    CPF    string `json:"cpf" validate:"required,len=11,numeric"`
}

func (h *Handler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
    var req CreateSubscriptionRequest
    
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", 400)
        return
    }
    
    if err := validate.Struct(req); err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    
    // Processar...
}
```

### Validar CPF

```go
func isValidCPF(cpf string) bool {
    // Remove caracteres não numéricos
    cpf = regexp.MustCompile(`\D`).ReplaceAllString(cpf, "")
    
    if len(cpf) != 11 {
        return false
    }
    
    // Verificar se todos os dígitos são iguais
    allSame := true
    for i := 1; i < len(cpf); i++ {
        if cpf[i] != cpf[0] {
            allSame = false
            break
        }
    }
    if allSame {
        return false
    }
    
    // Calcular dígitos verificadores
    // ... (implementar algoritmo de validação)
    
    return true
}
```

### Checklist

- [ ] Validar todos os inputs antes de processar
- [ ] Usar tipos específicos (UUID, não string)
- [ ] Sanitizar inputs de texto
- [ ] Limitar tamanho de payloads (body size limit)
- [ ] Validar CPF para PIX

---

## 6. Secrets Management

### Nunca commitar secrets

**.gitignore:**
```
.env
.env.*
!.env.example
certs/
*.pem
*.p12
```

### Usar env vars

```go
type Config struct {
    SupabaseURL        string `env:"SUPABASE_URL,required"`
    SupabaseServiceKey string `env:"SUPABASE_SERVICE_KEY,required"`
    EfiClientID        string `env:"EFI_CLIENT_ID,required"`
    EfiClientSecret    string `env:"EFI_CLIENT_SECRET,required"`
    StripeSecretKey    string `env:"STRIPE_SECRET_KEY,required"`
}

func Load() (*Config, error) {
    var cfg Config
    if err := env.Parse(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

### Checklist

- [ ] Nenhum secret em código ou repo
- [ ] .env no .gitignore
- [ ] Secrets diferentes por ambiente
- [ ] Certificados PIX em diretório seguro
- [ ] Rotação periódica de secrets

---

## 7. Logging de Segurança

### O que logar

```go
// Logar tentativas de autenticação falhas
logger.Warn().
    Str("ip", r.RemoteAddr).
    Str("user_agent", r.UserAgent()).
    Msg("Authentication failed")

// Logar webhooks inválidos
logger.Warn().
    Str("gateway", "stripe").
    Str("ip", r.RemoteAddr).
    Msg("Invalid webhook signature")

// Logar operações sensíveis
logger.Info().
    Str("user_id", userID).
    Str("action", "subscription_created").
    Str("gateway", "pix_auto").
    Msg("Subscription created")
```

### O que NÃO logar

- ❌ Tokens JWT completos
- ❌ Secrets/senhas
- ❌ Números completos de cartão
- ❌ CPF completo (mascarar: `123.***.**9-00`)

---

## 8. HTTPS e Headers

### Forçar HTTPS

```go
func HTTPSRedirect(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("X-Forwarded-Proto") == "http" {
            url := "https://" + r.Host + r.URL.String()
            http.Redirect(w, r, url, http.StatusMovedPermanently)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Security Headers

```go
func SecurityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        
        next.ServeHTTP(w, r)
    })
}
```

---

## 9. Checklist Final de Deploy

### Antes de ir para produção

- [ ] Remover `EFI_SANDBOX=true`
- [ ] Usar `STRIPE_SECRET_KEY=sk_live_*` (não sk_test)
- [ ] Configurar webhook endpoints no Efí Bank e Stripe
- [ ] Testar webhooks em staging primeiro
- [ ] Verificar rate limits
- [ ] Configurar alertas para erros de webhook
- [ ] Backup de certificados PIX
- [ ] Documentar processo de rotação de secrets
- [ ] Revisar logs para não expor dados sensíveis
- [ ] Testar fluxo completo (trial → PIX → renovação)

---

*Atualizado: 2026-02-08*
