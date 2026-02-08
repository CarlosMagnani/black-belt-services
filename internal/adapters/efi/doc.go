// Package efi implementa o adaptador para a API PIX da Efí Bank (antiga Gerencianet).
//
// Este pacote implementa:
//   - PIX imediato (cobranças)
//   - PIX Automático (pagamentos recorrentes)
//   - Split de pagamentos (distribuição automática)
//   - Abertura de contas (API de parceiros)
//   - Tratamento de webhooks
//
// # Autenticação
//
// A API Efí usa OAuth2 com mTLS (mutual TLS). Você precisa:
//   - Client ID e Client Secret (do painel Efí)
//   - Certificado .pem ou .p12 (gerado no painel Efí)
//
// # Início Rápido
//
// Criar o cliente:
//
//	client, err := efi.NewClient(cfg, "sua-chave-pix")
//
// Criar uma autorização de recorrência PIX:
//
//	rec, err := client.CreateRecurrence(ctx, efi.CreateRecurrenceRequest{
//	    Contract:    "assinatura-123",
//	    Debtor:      efi.PixDevedor{CPF: "12345678901", Nome: "João Silva"},
//	    Object:      "Mensalidade Academia",
//	    StartDate:   "2024-02-01",
//	    EndDate:     "2025-02-01",
//	    Periodicity: efi.PeriodicityMonthly,
//	    Amount:      "99.90",
//	})
//
// O devedor receberá um QR Code (rec.QRCode) para autorizar no app do banco.
//
// # Tratamento de Webhooks
//
// Configure um handler de webhook:
//
//	handler := efi.NewWebhookHandler()
//	handler.OnPixPayment = func(ctx context.Context, pix efi.PixPayment) error {
//	    // Pagamento recebido - atualizar status da assinatura
//	    return nil
//	}
//	handler.OnRecurrenceUpdate = func(ctx context.Context, event efi.RecurrenceEvent) error {
//	    // Status da recorrência mudou (aprovada, rejeitada, cancelada)
//	    return nil
//	}
//	http.Handle("/webhooks/efi", handler)
//
// # Split de Pagamentos
//
// Distribuir pagamentos entre múltiplas partes:
//
//	config, err := client.CreateSplitConfig(ctx, efi.SplitConfig{
//	    Description: "Split academia",
//	    Immediate:   true,
//	    MyPart: efi.SplitPart{
//	        Type:  efi.SplitTypePercentage,
//	        Value: "70.00",
//	    },
//	    Transfers: []efi.SplitPart{
//	        {
//	            Type:  efi.SplitTypePercentage,
//	            Value: "30.00",
//	            Beneficiary: &efi.Beneficiary{
//	                CPF:  "98765432101",
//	                Name: "Academia Parceira",
//	            },
//	        },
//	    },
//	})
//
// Depois vincule o split às cobranças:
//
//	err = client.LinkSplitToCharge(ctx, txid, config.ID)
//
// # Tratamento de Erros
//
// O pacote fornece erros tipados para condições comuns:
//
//	if efi.IsNotFound(err) {
//	    // Recurso não existe
//	}
//	if efi.IsDuplicateRecurrence(err) {
//	    // Já existe uma recorrência com este contrato
//	}
//
// # Documentação da API
//
// Para mais detalhes, consulte a documentação oficial:
// https://dev.efipay.com.br
package efi
