package efi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateSplitConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  SplitConfig
		wantErr bool
	}{
		{
			name: "valid split with beneficiary",
			config: SplitConfig{
				Description: "Test split",
				MyPart: SplitPart{
					Type:  SplitTypePercentage,
					Value: "70.00",
				},
				Transfers: []SplitPart{
					{
						Type:  SplitTypePercentage,
						Value: "30.00",
						Beneficiary: &Beneficiary{
							CPF:  "12345678901",
							Name: "Partner",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "transfer without beneficiary",
			config: SplitConfig{
				Description: "Test split",
				MyPart: SplitPart{
					Type:  SplitTypePercentage,
					Value: "70.00",
				},
				Transfers: []SplitPart{
					{
						Type:  SplitTypePercentage,
						Value: "30.00",
						// Missing beneficiary
					},
				},
			},
			wantErr: true,
		},
		{
			name: "beneficiary without CPF or CNPJ",
			config: SplitConfig{
				Description: "Test split",
				MyPart: SplitPart{
					Type:  SplitTypePercentage,
					Value: "70.00",
				},
				Transfers: []SplitPart{
					{
						Type:  SplitTypePercentage,
						Value: "30.00",
						Beneficiary: &Beneficiary{
							Name: "Partner",
							// Missing CPF and CNPJ
						},
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSplitConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSplitConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestQuickSplitConfig(t *testing.T) {
	partner := Beneficiary{
		CPF:  "12345678901",
		Name: "Partner",
	}

	config := QuickSplitConfig("Test split", 70.0, partner)

	if config.Description != "Test split" {
		t.Errorf("Description = %v, want %v", config.Description, "Test split")
	}
	if config.MyPart.Value != "70.00" {
		t.Errorf("MyPart.Value = %v, want %v", config.MyPart.Value, "70.00")
	}
	if len(config.Transfers) != 1 {
		t.Fatalf("Transfers count = %v, want %v", len(config.Transfers), 1)
	}
	if config.Transfers[0].Value != "30.00" {
		t.Errorf("Transfer value = %v, want %v", config.Transfers[0].Value, "30.00")
	}
}

func TestGymPartnerSplitConfig(t *testing.T) {
	config := GymPartnerSplitConfig(70.0, "12345678901", "Academia Parceira")

	if config.MyPart.Value != "70.00" {
		t.Errorf("MyPart.Value = %v, want 70.00", config.MyPart.Value)
	}
	if len(config.Transfers) != 1 {
		t.Fatalf("Transfers count = %v, want 1", len(config.Transfers))
	}
	if config.Transfers[0].Beneficiary.CPF != "12345678901" {
		t.Errorf("Beneficiary CPF = %v, want 12345678901", config.Transfers[0].Beneficiary.CPF)
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "ErrNotFound sentinel",
			err:  ErrNotFound,
			want: true,
		},
		{
			name: "API error with 404",
			err:  &APIError{Status: 404, Mensagem: "Not found"},
			want: true,
		},
		{
			name: "API error with 400",
			err:  &APIError{Status: 400, Mensagem: "Bad request"},
			want: false,
		},
		{
			name: "other error",
			err:  ErrUnauthorized,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractPixKeyFromURL(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/webhooks/efi/chave123", "chave123"},
		{"/webhooks/efi/", ""},
		{"/webhooks/other/key", ""},
		{"/something/else", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := ExtractPixKeyFromURL(tt.path); got != tt.want {
				t.Errorf("ExtractPixKeyFromURL(%v) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRecurrenceStatusError(t *testing.T) {
	err := NewRecurrenceStatusError("rec123", RecurrenceStatusRejected, "User rejected")

	if !err.IsRejected() {
		t.Error("Expected IsRejected() to be true")
	}
	if err.IsCancelled() {
		t.Error("Expected IsCancelled() to be false")
	}
	if err.IsExpired() {
		t.Error("Expected IsExpired() to be false")
	}

	expected := "recorrência rec123 está REJEITADA: User rejected"
	if err.Error() != expected {
		t.Errorf("Error() = %v, want %v", err.Error(), expected)
	}
}

func TestWebhookHandler_HandleEfiWebhook(t *testing.T) {
	var receivedPix PixPayment
	var receivedRec RecurrenceEvent

	handler := NewWebhookHandler()
	handler.SkipSignatureValidation = true
	handler.OnPixPayment = func(ctx context.Context, pix PixPayment) error {
		receivedPix = pix
		return nil
	}
	handler.OnRecurrenceUpdate = func(ctx context.Context, event RecurrenceEvent) error {
		receivedRec = event
		return nil
	}

	// Test PIX payment webhook
	t.Run("valid pix payment", func(t *testing.T) {
		payload := WebhookEvent{
			Pix: []PixPayment{
				{
					EndToEndID: "E123456789",
					TxID:       "tx123",
					Value:      "100.00",
					Payer:      PixDevedor{Nome: "John Doe", CPF: "12345678901"},
				},
			},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/webhooks/efi", nil)
		req.Body = &testReadCloser{data: body}
		w := httptest.NewRecorder()

		handler.HandleEfiWebhook(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if receivedPix.EndToEndID != "E123456789" {
			t.Errorf("Expected EndToEndID E123456789, got %s", receivedPix.EndToEndID)
		}
	})

	// Test recurrence event webhook
	t.Run("valid recurrence event", func(t *testing.T) {
		payload := WebhookEvent{
			Rec: &RecurrenceEvent{
				ID:       "rec123",
				Contract: "contract456",
				Status:   RecurrenceStatusApproved,
			},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/webhooks/efi", nil)
		req.Body = &testReadCloser{data: body}
		w := httptest.NewRecorder()

		handler.HandleEfiWebhook(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if receivedRec.ID != "rec123" {
			t.Errorf("Expected recurrence ID rec123, got %s", receivedRec.ID)
		}
	})

	// Test wrong method
	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/webhooks/efi", nil)
		w := httptest.NewRecorder()

		handler.HandleEfiWebhook(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

type testReadCloser struct {
	data []byte
	pos  int
}

func (r *testReadCloser) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *testReadCloser) Close() error {
	return nil
}

func TestPeriodicityConstants(t *testing.T) {
	tests := []struct {
		periodicity Periodicity
		expected    string
	}{
		{PeriodicityWeekly, "SEMANAL"},
		{PeriodicityBiweekly, "QUINZENAL"},
		{PeriodicityMonthly, "MENSAL"},
		{PeriodicityQuarterly, "TRIMESTRAL"},
		{PeriodicityYearly, "ANUAL"},
	}

	for _, tt := range tests {
		if string(tt.periodicity) != tt.expected {
			t.Errorf("Periodicity = %v, want %v", tt.periodicity, tt.expected)
		}
	}
}

func TestRecurrenceStatusConstants(t *testing.T) {
	tests := []struct {
		status   RecurrenceStatus
		expected string
	}{
		{RecurrenceStatusCreated, "CRIADA"},
		{RecurrenceStatusApproved, "APROVADA"},
		{RecurrenceStatusRejected, "REJEITADA"},
		{RecurrenceStatusCancelled, "CANCELADA"},
		{RecurrenceStatusExpired, "EXPIRADA"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("Status = %v, want %v", tt.status, tt.expected)
		}
	}
}
