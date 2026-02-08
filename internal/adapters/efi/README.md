# Efí Bank PIX Adapter

Adapter Go para integração com a API PIX da Efí Bank (antiga Gerencianet).

## Funcionalidades

- ✅ **PIX Automático** - Recorrência automática de pagamentos PIX
- ✅ **Split de Pagamento** - Distribuição automática entre beneficiários
- ✅ **Abertura de Contas** - API parceiros (restrita)
- ✅ **Webhooks** - Recebimento de notificações
- ✅ **Autenticação OAuth2 + mTLS**
- ✅ **Retry com backoff exponencial**
- ✅ **Logs estruturados**

## Instalação

```go
import "github.com/seu-projeto/backend/internal/adapters/efi"
```

## Configuração

### Credenciais

1. Acesse o [painel Efí](https://app.efipay.com.br)
2. Vá em API > Aplicações
3. Crie uma nova aplicação
4. Baixe o certificado (.pem ou .p12)

### Variáveis de Ambiente

```env
EFI_CLIENT_ID=seu-client-id
EFI_CLIENT_SECRET=seu-client-secret
EFI_CERT_PATH=/path/to/certificate.pem
EFI_SANDBOX=true
```

## Uso

### Criar Cliente

```go
client, err := efi.NewClient(efi.Config{
    ClientID:     os.Getenv("EFI_CLIENT_ID"),
    ClientSecret: os.Getenv("EFI_CLIENT_SECRET"),
    CertPath:     os.Getenv("EFI_CERT_PATH"),
    Sandbox:      os.Getenv("EFI_SANDBOX") == "true",
    Logger:       slog.Default(),
})
if err != nil {
    log.Fatal(err)
}
```

### PIX Automático (Recorrência)

```go
// Criar autorização de recorrência
rec, err := client.CreateRecurrence(ctx, efi.CreateRecurrenceRequest{
    Contract:    "assinatura-123",
    Debtor: efi.Debtor{
        CPF:   "12345678901",
        Name:  "João da Silva",
        Email: "joao@email.com",
    },
    Object:      "Plano Mensal Academia",
    StartDate:   "2024-02-01",
    EndDate:     "2025-02-01",
    Periodicity: efi.PeriodicityMonthly,
    Amount:      "149.90",
    DueDay:      10, // Dia do vencimento (1-28)
})

// O aluno deve autorizar no app do banco usando rec.QRCode
fmt.Println("QR Code:", rec.QRCode)
fmt.Println("Status:", rec.Status) // CRIADA (aguardando autorização)

// Consultar status
rec, err = client.GetRecurrence(ctx, rec.ID)
// Status: APROVADA (após autorização do aluno)

// Cancelar
err = client.CancelRecurrence(ctx, rec.ID)
```

### Split de Pagamento

```go
// Criar configuração de split (70% dono, 30% parceiro)
config, err := client.CreateSplitConfig(ctx, efi.SplitConfig{
    Description: "Split Academia Principal",
    Immediate:   true, // Split imediato (não D+1)
    MyPart: efi.SplitPart{
        Type:  string(efi.SplitTypePercentage),
        Value: "70.00",
    },
    Transfers: []efi.SplitPart{
        {
            Type:  string(efi.SplitTypePercentage),
            Value: "30.00",
            Beneficiary: &efi.Beneficiary{
                CPF:  "98765432101",
                Name: "Academia Parceira",
            },
        },
    },
})

// Vincular split a uma cobrança
err = client.LinkSplitToCharge(ctx, "txid-da-cobranca", config.ID)
```

### Webhooks

```go
// Criar handler
handler := efi.NewWebhookHandler(logger)
handler.WebhookSecret = "seu-secret" // Opcional

// Callback para pagamentos PIX
handler.OnPixPayment = func(ctx context.Context, pix efi.PixPayment) error {
    log.Printf("Pagamento recebido: %s - R$ %s", pix.EndToEndID, pix.Value)
    
    // Atualizar banco de dados
    return db.RegisterPayment(ctx, pix.TxID, pix.Value)
}

// Callback para mudanças de status de recorrência
handler.OnRecurrenceUpdate = func(ctx context.Context, event efi.RecurrenceEvent) error {
    log.Printf("Recorrência %s: %s", event.ID, event.Status)
    
    switch event.Status {
    case efi.RecurrenceStatusApproved:
        return db.ActivateSubscription(ctx, event.Contract)
    case efi.RecurrenceStatusCancelled:
        return db.CancelSubscription(ctx, event.Contract)
    }
    return nil
}

// Montar no router
http.Handle("POST /webhooks/efi", handler)

// Registrar URL do webhook na Efí
err = client.RegisterWebhook(ctx, "sua-chave-pix", "https://seu-dominio.com/webhooks/efi")
```

## Fluxo de Assinatura

```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│  Aluno  │     │ Backend │     │   Efí   │     │  Banco  │
└────┬────┘     └────┬────┘     └────┬────┘     └────┬────┘
     │               │               │               │
     │ 1. Escolhe    │               │               │
     │    plano      │               │               │
     │───────────────>               │               │
     │               │               │               │
     │               │ 2. Create     │               │
     │               │    Recurrence │               │
     │               │───────────────>               │
     │               │               │               │
     │               │ 3. QR Code    │               │
     │               │<───────────────               │
     │               │               │               │
     │ 4. QR Code    │               │               │
     │<───────────────               │               │
     │               │               │               │
     │ 5. Autoriza   │               │               │
     │────────────────────────────────────────────────>
     │               │               │               │
     │               │ 6. Webhook    │               │
     │               │    APROVADA   │               │
     │               │<───────────────               │
     │               │               │               │
     │ 7. Assinatura │               │               │
     │    ativa!     │               │               │
     │<───────────────               │               │
     │               │               │               │
     │               │ 8. A cada mês │               │
     │               │    Efí cobra  │               │
     │               │    automatico │               │
     │               │<───────────────>              │
     │               │               │               │
     │               │ 9. Webhook    │               │
     │               │    PIX pago   │               │
     │               │<───────────────               │
     │               │               │               │
```

## Tratamento de Erros

```go
if err := client.CreateRecurrence(ctx, req); err != nil {
    if efi.IsDuplicateRecurrence(err) {
        // Já existe uma recorrência com este contrato
    }
    if efi.IsNotFound(err) {
        // Recurso não encontrado
    }
    if efi.IsRateLimited(err) {
        // Rate limit - aguardar
    }
    if efi.IsServerError(err) {
        // Erro no servidor Efí - tentar novamente
    }
}
```

## Testes

```bash
go test ./internal/adapters/efi/...
```

## Referências

- [Documentação Efí](https://dev.efipay.com.br)
- [PIX Automático](https://dev.efipay.com.br/docs/api-pix/recorrencia)
- [Split de Pagamento](https://dev.efipay.com.br/docs/api-pix/split-de-pagamento)
