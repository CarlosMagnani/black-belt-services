# Fluxos - BlackBelt Services

## 1. Onboarding de Academia + Trial

### Objetivo
Cadastrar nova academia com 20 dias de trial gratuito.

### Fluxo Completo

```
┌─────────────────────────────────────────────────────┐
│ 1. Dono faz signup no app (Supabase Auth)          │
│    role = 'owner'                                   │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 2. Dono preenche dados da academia                 │
│    - Nome, descrição                                │
│    - Logo (opcional)                                │
│    - Endereço                                       │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 3. Frontend: POST /api/academies                   │
│    {                                                 │
│      "name": "Academia Gracie",                     │
│      "description": "Jiu-Jitsu tradicional",       │
│      "address": {...}                               │
│    }                                                 │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 4. Backend: Cria academia                          │
│    - Gera invite_code único                         │
│    - Gera slug URL-friendly                         │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 5. Trigger: Cria subscription (trial)              │
│    INSERT INTO subscriptions (                      │
│      academy_id,                                    │
│      plan_id = 'starter',                           │
│      status = 'trialing',                           │
│      trial_end_date = NOW() + 20 days              │
│    )                                                │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 6. Academia ativa! 🎉                              │
│    - 20 dias de trial                               │
│    - Todas as features disponíveis                  │
│    - Banner "X dias restantes"                      │
└─────────────────────────────────────────────────────┘
```

---

## 2. Assinatura via PIX Automático 🇧🇷

### Objetivo
Converter trial em assinatura paga usando PIX Automático (não compromete limite do cartão).

### Fluxo Completo

```
┌─────────────────────────────────────────────────────┐
│ 1. Dono acessa tela de Upgrade                     │
│    - Trial expirando ou já expirou                  │
│    - Vê planos disponíveis                          │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 2. Dono seleciona plano e "Pagar com PIX"          │
│    - Starter: R$99/mês                              │
│    - Pro: R$199/mês                                 │
│    - Business: R$399/mês                            │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 3. Frontend: POST /api/subscriptions/pix-auto      │
│    {                                                 │
│      "plan_id": "uuid-do-plano",                    │
│      "cpf": "123.456.789-00"                        │
│    }                                                 │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 4. Backend: Cria cobrança no Efí Bank              │
│    efi.PixCreateCharge({                            │
│      valor: 9900,                                   │
│      devedor: {cpf, nome},                          │
│      chave: "pix@blackbelt.com",                   │
│      calendario: {expiracao: 3600}                  │
│    })                                               │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 5. Efí Bank retorna QR Code                        │
│    {                                                 │
│      "txid": "xxx",                                 │
│      "pixCopiaECola": "00020126...",               │
│      "qrcode": "data:image/png;base64,..."         │
│    }                                                │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 6. Frontend: Exibe QR Code                         │
│    - QR Code para escanear                          │
│    - Código PIX Copia e Cola                        │
│    - Timer de expiração (1 hora)                    │
│    - Polling para verificar status                  │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 7. Dono escaneia QR no app do banco               │
│    - Abre app do banco (Nubank, Itaú, etc)         │
│    - Escaneia QR Code                               │
│    - Autoriza pagamento recorrente                  │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 8. Webhook: pix.authorized                         │
│    POST /api/webhooks/pix                           │
│    {                                                 │
│      "evento": "pix_auto_autorizado",               │
│      "txid": "xxx",                                 │
│      "recorrencia_id": "yyy"                        │
│    }                                                │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 9. Backend: Atualiza subscription                  │
│    UPDATE subscriptions SET                         │
│      status = 'active',                             │
│      payment_gateway = 'pix_auto',                  │
│      pix_recurrence_id = 'yyy',                     │
│      current_period_start = NOW(),                  │
│      current_period_end = NOW() + 1 month           │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 10. Assinatura ativa! 💚                           │
│     - Cobranças automáticas todo mês                │
│     - Dono recebe notificação no banco              │
│     - Sem comprometer limite do cartão              │
└─────────────────────────────────────────────────────┘
```

### Renovação Automática (mensal)

```
┌─────────────────────────────────────────────────────┐
│ 1. Data de vencimento chega                        │
│    current_period_end = hoje                        │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 2. Efí Bank tenta cobrança automática              │
│    (baseado na recorrencia_id)                      │
└──────┬──────────────────────────────────────────────┘
       │
       ├─── SUCESSO ─────────────────────┐
       │                                  │
       ▼                                  ▼
┌──────────────────┐           ┌──────────────────────┐
│ FALHA            │           │ Webhook: pix.paid    │
│ (saldo insuf.)   │           └──────────────────────┘
└──────┬───────────┘                      │
       │                                  ▼
       ▼                        ┌──────────────────────┐
┌──────────────────┐           │ Backend: Renova      │
│ Retry em 24h     │           │ current_period_end   │
│ (até 3x)         │           │ += 1 month           │
└──────┬───────────┘           └──────────────────────┘
       │
       ├─── 3 falhas ───────────────────┐
       │                                 │
       ▼                                 ▼
┌──────────────────┐           ┌──────────────────────┐
│ status =         │           │ Email: "Atualize seu │
│ 'past_due'       │           │ método de pagamento" │
└──────────────────┘           └──────────────────────┘
```

---

## 3. Assinatura via Stripe (Fallback/Internacional)

### Objetivo
Alternativa para quem prefere cartão de crédito ou está fora do Brasil.

### Fluxo Completo

```
┌─────────────────────────────────────────────────────┐
│ 1. Dono acessa tela de Upgrade                     │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 2. Dono seleciona plano e "Pagar com Cartão"       │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 3. Frontend: POST /api/subscriptions/stripe        │
│    {                                                 │
│      "plan_id": "uuid-do-plano",                    │
│      "success_url": "blackbelt://payment-success",  │
│      "cancel_url": "blackbelt://payment-cancel"     │
│    }                                                 │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 4. Backend: Cria Checkout Session no Stripe        │
│    stripe.checkout.sessions.Create({                │
│      mode: "subscription",                          │
│      line_items: [{price: "price_xxx", qty: 1}],   │
│      success_url,                                   │
│      cancel_url,                                    │
│      client_reference_id: academy_id                │
│    })                                               │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 5. Stripe retorna Checkout URL                     │
│    {                                                 │
│      "url": "https://checkout.stripe.com/c/pay/..."│
│    }                                                │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 6. Frontend: Redireciona para Stripe               │
│    Linking.openURL(checkoutUrl)                     │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 7. Dono preenche cartão no Stripe Checkout         │
│    - Número do cartão                               │
│    - Validade, CVV                                  │
│    - 3D Secure (se necessário)                      │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 8. Webhook: checkout.session.completed             │
│    POST /api/webhooks/stripe                        │
│    {                                                 │
│      "type": "checkout.session.completed",          │
│      "data": {                                      │
│        "object": {                                  │
│          "client_reference_id": "academy_id",       │
│          "customer": "cus_xxx",                     │
│          "subscription": "sub_xxx"                  │
│        }                                            │
│      }                                              │
│    }                                                │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 9. Backend: Atualiza subscription                  │
│    UPDATE subscriptions SET                         │
│      status = 'active',                             │
│      payment_gateway = 'stripe',                    │
│      stripe_customer_id = 'cus_xxx',                │
│      stripe_subscription_id = 'sub_xxx',            │
│      current_period_start = NOW(),                  │
│      current_period_end = NOW() + 1 month           │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 10. Deep link retorna para app                     │
│     blackbelt://payment-success                     │
│     Tela de sucesso! 🎉                            │
└─────────────────────────────────────────────────────┘
```

---

## 4. Cancelamento de Assinatura

### Fluxo

```
┌─────────────────────────────────────────────────────┐
│ 1. Dono acessa Configurações → Assinatura          │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 2. Clica "Cancelar Assinatura"                     │
│    Modal: "Você perderá acesso em DD/MM/YYYY"       │
│    Input: Motivo (opcional)                         │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 3. Frontend: DELETE /api/subscriptions/:id         │
│    {                                                 │
│      "reason": "muito caro"                         │
│    }                                                 │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 4. Backend: Cancela no gateway                     │
│                                                     │
│    if gateway == 'pix_auto':                        │
│        efi.CancelRecurrence(recurrence_id)          │
│    else:                                            │
│        stripe.Subscriptions.Cancel(sub_id)          │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 5. Backend: Atualiza subscription                  │
│    UPDATE subscriptions SET                         │
│      status = 'canceled',                           │
│      canceled_at = NOW(),                           │
│      cancel_reason = 'muito caro'                   │
│    -- Mantém current_period_end (acesso até lá)     │
└──────┬──────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────┐
│ 6. Acesso mantido até fim do período pago          │
│    - Banner: "Sua assinatura termina em DD/MM"      │
│    - CTA: "Reativar assinatura"                     │
└──────┬──────────────────────────────────────────────┘
       │
       │ (current_period_end chega)
       ▼
┌─────────────────────────────────────────────────────┐
│ 7. Acesso bloqueado                                │
│    status = 'expired'                               │
│    Mostra apenas tela de reativação                 │
└─────────────────────────────────────────────────────┘
```

---

## 5. Fluxo de Webhooks

### Eventos PIX Automático (Efí Bank)

| Evento | Ação |
|--------|------|
| `pix_auto.autorizado` | Ativar subscription, salvar recurrence_id |
| `pix_auto.pagamento_recebido` | Renovar período, registrar em payment_history |
| `pix_auto.pagamento_falhou` | Marcar past_due, notificar dono |
| `pix_auto.cancelado` | Marcar canceled |

### Eventos Stripe

| Evento | Ação |
|--------|------|
| `checkout.session.completed` | Ativar subscription, salvar IDs |
| `invoice.paid` | Renovar período, registrar pagamento |
| `invoice.payment_failed` | Marcar past_due, notificar |
| `customer.subscription.deleted` | Marcar canceled |

### Implementação do Handler

```go
// handlers/webhook.go

func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    payload, _ := io.ReadAll(r.Body)
    gateway := chi.URLParam(r, "gateway") // "pix" ou "stripe"
    
    var event *ports.WebhookEvent
    var err error
    
    // 1. Validar e parsear (adapter específico)
    switch gateway {
    case "pix":
        event, err = h.pixAdapter.HandleWebhook(r.Context(), payload, r.Header.Get("X-Efi-Signature"))
    case "stripe":
        event, err = h.stripeAdapter.HandleWebhook(r.Context(), payload, r.Header.Get("Stripe-Signature"))
    }
    
    if err != nil {
        http.Error(w, "Invalid webhook", 400)
        return
    }
    
    // 2. Idempotency check
    exists, _ := h.webhookRepo.ExistsByEventID(r.Context(), event.EventID)
    if exists {
        w.WriteHeader(200) // Já processado, retorna OK
        return
    }
    
    // 3. Persistir evento
    h.webhookRepo.Create(r.Context(), &domain.WebhookEvent{
        Gateway:   gateway,
        EventID:   event.EventID,
        EventType: event.Type,
        Payload:   event.Payload,
        Status:    "pending",
    })
    
    // 4. Processar (async)
    go h.subscriptionService.ProcessPaymentEvent(context.Background(), event)
    
    // 5. Retornar 200 imediatamente
    w.WriteHeader(200)
}
```

### Processamento de Eventos

```go
// service/subscription_service.go

func (s *SubscriptionService) ProcessPaymentEvent(ctx context.Context, event *ports.WebhookEvent) error {
    switch event.Type {
    
    // ========== PIX AUTOMÁTICO ==========
    case "pix_auto.autorizado":
        return s.handlePixAuthorized(ctx, event)
        
    case "pix_auto.pagamento_recebido":
        return s.handlePaymentReceived(ctx, event)
        
    case "pix_auto.pagamento_falhou":
        return s.handlePaymentFailed(ctx, event)
    
    // ========== STRIPE ==========
    case "checkout.session.completed":
        return s.handleCheckoutCompleted(ctx, event)
        
    case "invoice.paid":
        return s.handleInvoicePaid(ctx, event)
        
    case "invoice.payment_failed":
        return s.handlePaymentFailed(ctx, event)
        
    case "customer.subscription.deleted":
        return s.handleSubscriptionCanceled(ctx, event)
    }
    
    return nil
}

func (s *SubscriptionService) handlePaymentReceived(ctx context.Context, event *ports.WebhookEvent) error {
    sub, err := s.repo.FindByGatewayID(ctx, event.SubscriptionID)
    if err != nil {
        return err
    }
    
    // Renovar período
    sub.CurrentPeriodStart = time.Now()
    sub.CurrentPeriodEnd = time.Now().AddDate(0, 1, 0) // +1 mês
    sub.Status = domain.StatusActive
    
    // Registrar pagamento
    payment := &domain.PaymentHistory{
        SubscriptionID:   sub.ID,
        Amount:           event.Amount,
        PaymentGateway:   sub.PaymentGateway,
        GatewayPaymentID: event.PaymentID,
        Status:           domain.PaymentSucceeded,
        PeriodStart:      sub.CurrentPeriodStart,
        PeriodEnd:        sub.CurrentPeriodEnd,
        PaidAt:           time.Now(),
    }
    
    s.paymentRepo.Create(ctx, payment)
    return s.repo.Update(ctx, sub)
}
```

---

## 6. Verificação de Trial (Cron)

### Job Diário

```go
// service/subscription_service.go

func (s *SubscriptionService) CheckExpiredTrials(ctx context.Context) error {
    // 1. Buscar trials expirados
    expired, err := s.repo.FindExpiredTrials(ctx)
    if err != nil {
        return err
    }
    
    for _, sub := range expired {
        // 2. Marcar como expirado
        sub.Status = domain.StatusExpired
        s.repo.Update(ctx, sub)
        
        // 3. Notificar dono
        academy, _ := s.academyRepo.FindByID(ctx, sub.AcademyID)
        s.notifier.SendTrialExpired(ctx, academy.Email, academy.Name)
    }
    
    // 4. Buscar trials expirando em 3 dias
    expiring, err := s.repo.FindTrialsExpiringSoon(ctx, 3)
    if err != nil {
        return err
    }
    
    for _, sub := range expiring {
        academy, _ := s.academyRepo.FindByID(ctx, sub.AcademyID)
        daysLeft := int(time.Until(sub.TrialEndDate).Hours() / 24)
        s.notifier.SendTrialExpiring(ctx, academy.Email, academy.Name, daysLeft)
    }
    
    return nil
}
```

### Query para Trials Expirados

```sql
-- Trials que expiraram
SELECT * FROM subscriptions
WHERE status = 'trialing'
  AND trial_end_date < NOW();

-- Trials expirando em 3 dias
SELECT * FROM subscriptions
WHERE status = 'trialing'
  AND trial_end_date BETWEEN NOW() AND NOW() + INTERVAL '3 days';
```

---

## Resumo dos Endpoints

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/api/academies` | Criar academia (+ trigger subscription) |
| GET | `/api/academies/:id` | Buscar academia |
| POST | `/api/subscriptions/pix-auto` | Iniciar assinatura PIX |
| POST | `/api/subscriptions/stripe` | Iniciar assinatura Stripe |
| DELETE | `/api/subscriptions/:id` | Cancelar assinatura |
| GET | `/api/subscriptions/current` | Buscar assinatura atual |
| POST | `/api/webhooks/pix` | Webhook Efí Bank |
| POST | `/api/webhooks/stripe` | Webhook Stripe |

---

*Atualizado: 2026-02-08*
