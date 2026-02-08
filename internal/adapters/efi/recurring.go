package efi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// CreateRecurrence cria uma nova autorização de recorrência PIX Automático.
// O devedor receberá um QR Code para autorizar no app do banco.
func (c *Client) CreateRecurrence(ctx context.Context, req CreateRecurrenceRequest) (*Recurrence, error) {
	if req.Contract == "" {
		return nil, fmt.Errorf("contrato (contract) é obrigatório")
	}
	if req.Debtor.CPF == "" && req.Debtor.CNPJ == "" {
		return nil, fmt.Errorf("CPF ou CNPJ do devedor é obrigatório")
	}
	if req.Amount == "" {
		return nil, fmt.Errorf("valor é obrigatório")
	}
	if req.Periodicity == "" {
		return nil, fmt.Errorf("periodicidade é obrigatória")
	}

	// Monta o payload
	payload := map[string]interface{}{
		"contrato":      req.Contract,
		"devedor":       buildDebtorPayload(req.Debtor),
		"objeto":        req.Object,
		"dataInicial":   req.StartDate,
		"dataFinal":     req.EndDate,
		"periodicidade": req.Periodicity,
		"valorRec":      req.Amount,
	}

	if req.Description != "" {
		payload["descricao"] = req.Description
	}
	if req.DueDay > 0 && req.DueDay <= 28 {
		payload["diaVencimento"] = req.DueDay
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, "/v2/rec", payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar recorrência: %w", err)
	}

	var recurrence Recurrence
	if err := json.Unmarshal(respBody, &recurrence); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &recurrence, nil
}

// buildDebtorPayload cria o payload do devedor no formato da API
func buildDebtorPayload(d PixDevedor) map[string]interface{} {
	payload := map[string]interface{}{
		"nome": d.Nome,
	}
	if d.CPF != "" {
		payload["cpf"] = d.CPF
	}
	if d.CNPJ != "" {
		payload["cnpj"] = d.CNPJ
	}
	if d.Email != "" {
		payload["email"] = d.Email
	}
	return payload
}

// GetRecurrence consulta uma recorrência pelo ID
func (c *Client) GetRecurrence(ctx context.Context, idRec string) (*Recurrence, error) {
	if idRec == "" {
		return nil, fmt.Errorf("idRec é obrigatório")
	}

	path := fmt.Sprintf("/v2/rec/%s", idRec)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar recorrência: %w", err)
	}

	var recurrence Recurrence
	if err := json.Unmarshal(respBody, &recurrence); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &recurrence, nil
}

// UpdateRecurrence atualiza uma recorrência existente (valor, data final, etc)
func (c *Client) UpdateRecurrence(ctx context.Context, idRec string, req UpdateRecurrenceRequest) (*Recurrence, error) {
	if idRec == "" {
		return nil, fmt.Errorf("idRec é obrigatório")
	}

	path := fmt.Sprintf("/v2/rec/%s", idRec)

	payload := make(map[string]interface{})
	if req.Amount != "" {
		payload["valorRec"] = req.Amount
	}
	if req.EndDate != "" {
		payload["dataFinal"] = req.EndDate
	}
	if req.Status != "" {
		payload["status"] = req.Status
	}

	respBody, err := c.doRequest(ctx, http.MethodPatch, path, payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao atualizar recorrência: %w", err)
	}

	var recurrence Recurrence
	if err := json.Unmarshal(respBody, &recurrence); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &recurrence, nil
}

// CancelRecurrence cancela uma recorrência ativa
func (c *Client) CancelRecurrence(ctx context.Context, idRec string) error {
	if idRec == "" {
		return fmt.Errorf("idRec é obrigatório")
	}

	path := fmt.Sprintf("/v2/rec/%s", idRec)

	payload := map[string]interface{}{
		"status": "CANCELADA",
	}

	_, err := c.doRequest(ctx, http.MethodPatch, path, payload)
	if err != nil {
		return fmt.Errorf("erro ao cancelar recorrência: %w", err)
	}

	return nil
}

// ListRecurrences lista recorrências em um período
func (c *Client) ListRecurrences(ctx context.Context, startDate, endDate time.Time) (*RecurrenceListResponse, error) {
	path := fmt.Sprintf("/v2/rec?inicio=%s&fim=%s",
		startDate.Format("2006-01-02T15:04:05Z"),
		endDate.Format("2006-01-02T15:04:05Z"),
	)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao listar recorrências: %w", err)
	}

	var result RecurrenceListResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// IsRecurrenceApproved retorna true se a recorrência foi aprovada
func (c *Client) IsRecurrenceApproved(ctx context.Context, idRec string) (bool, error) {
	rec, err := c.GetRecurrence(ctx, idRec)
	if err != nil {
		return false, err
	}
	return rec.Status == RecurrenceStatusApproved, nil
}

// WaitForRecurrenceApproval aguarda a aprovação da recorrência pelo pagador
func (c *Client) WaitForRecurrenceApproval(ctx context.Context, idRec string, pollInterval, timeout time.Duration) (*Recurrence, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		rec, err := c.GetRecurrence(ctx, idRec)
		if err != nil {
			return nil, err
		}

		switch rec.Status {
		case RecurrenceStatusApproved:
			return rec, nil
		case RecurrenceStatusRejected:
			return rec, fmt.Errorf("recorrência foi rejeitada pelo pagador")
		case RecurrenceStatusCancelled:
			return rec, fmt.Errorf("recorrência foi cancelada")
		case RecurrenceStatusExpired:
			return rec, fmt.Errorf("recorrência expirou")
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}

	return nil, fmt.Errorf("timeout aguardando aprovação da recorrência")
}

// GetRecurrencePayments lista os pagamentos de uma recorrência
func (c *Client) GetRecurrencePayments(ctx context.Context, idRec string) ([]PixPayment, error) {
	if idRec == "" {
		return nil, fmt.Errorf("idRec é obrigatório")
	}

	path := fmt.Sprintf("/v2/rec/%s/pix", idRec)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar pagamentos: %w", err)
	}

	var result struct {
		Pix []PixPayment `json:"pix"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return result.Pix, nil
}
