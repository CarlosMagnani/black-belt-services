package efi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// CreateSplitConfig cria uma nova configuração de split de pagamento.
// Split permite distribuir pagamentos automaticamente entre beneficiários.
func (c *Client) CreateSplitConfig(ctx context.Context, config SplitConfig) (*SplitConfigResponse, error) {
	if config.Description == "" {
		return nil, fmt.Errorf("descrição é obrigatória")
	}
	if config.MyPart.Value == "" {
		return nil, fmt.Errorf("valor da minha parte é obrigatório")
	}

	// Valida a configuração
	if err := ValidateSplitConfig(config); err != nil {
		return nil, err
	}

	// Monta o payload
	payload := map[string]interface{}{
		"descricao": config.Description,
		"imediato":  config.Immediate,
		"minhaParte": map[string]interface{}{
			"tipo":  config.MyPart.Type,
			"valor": config.MyPart.Value,
		},
		"repasses": buildTransfersPayload(config.Transfers),
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, "/v2/gn/split/config", payload)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar configuração de split: %w", err)
	}

	var result SplitConfigResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// buildTransfersPayload constrói o array de repasses para a API
func buildTransfersPayload(transfers []SplitPart) []map[string]interface{} {
	result := make([]map[string]interface{}, len(transfers))
	for i, t := range transfers {
		transfer := map[string]interface{}{
			"tipo":  t.Type,
			"valor": t.Value,
		}
		if t.Beneficiary != nil {
			beneficiary := make(map[string]interface{})
			if t.Beneficiary.CPF != "" {
				beneficiary["cpf"] = t.Beneficiary.CPF
			}
			if t.Beneficiary.CNPJ != "" {
				beneficiary["cnpj"] = t.Beneficiary.CNPJ
			}
			if t.Beneficiary.Bank != "" {
				beneficiary["banco"] = t.Beneficiary.Bank
			}
			if t.Beneficiary.Name != "" {
				beneficiary["nome"] = t.Beneficiary.Name
			}
			transfer["favorecido"] = beneficiary
		}
		result[i] = transfer
	}
	return result
}

// GetSplitConfig consulta uma configuração de split pelo ID
func (c *Client) GetSplitConfig(ctx context.Context, configID string) (*SplitConfigResponse, error) {
	if configID == "" {
		return nil, fmt.Errorf("config_id é obrigatório")
	}

	path := fmt.Sprintf("/v2/gn/split/config/%s", configID)

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar configuração de split: %w", err)
	}

	var result SplitConfigResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return &result, nil
}

// LinkSplitToCharge vincula uma configuração de split a uma cobrança PIX
func (c *Client) LinkSplitToCharge(ctx context.Context, txid, splitConfigID string) error {
	if txid == "" {
		return fmt.Errorf("txid é obrigatório")
	}
	if splitConfigID == "" {
		return fmt.Errorf("split_config_id é obrigatório")
	}

	path := fmt.Sprintf("/v2/gn/split/cob/%s/vinculo/%s", txid, splitConfigID)

	_, err := c.doRequest(ctx, http.MethodPut, path, nil)
	if err != nil {
		return fmt.Errorf("erro ao vincular split à cobrança: %w", err)
	}

	return nil
}

// UnlinkSplitFromCharge remove uma configuração de split de uma cobrança
func (c *Client) UnlinkSplitFromCharge(ctx context.Context, txid, splitConfigID string) error {
	if txid == "" {
		return fmt.Errorf("txid é obrigatório")
	}
	if splitConfigID == "" {
		return fmt.Errorf("split_config_id é obrigatório")
	}

	path := fmt.Sprintf("/v2/gn/split/cob/%s/vinculo/%s", txid, splitConfigID)

	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("erro ao desvincular split da cobrança: %w", err)
	}

	return nil
}

// DeleteSplitConfig deleta uma configuração de split
func (c *Client) DeleteSplitConfig(ctx context.Context, configID string) error {
	if configID == "" {
		return fmt.Errorf("config_id é obrigatório")
	}

	path := fmt.Sprintf("/v2/gn/split/config/%s", configID)

	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("erro ao deletar configuração de split: %w", err)
	}

	return nil
}

// ValidateSplitConfig valida se uma configuração de split está correta
func ValidateSplitConfig(config SplitConfig) error {
	for i, transfer := range config.Transfers {
		if transfer.Beneficiary == nil {
			return fmt.Errorf("repasse[%d]: beneficiário é obrigatório", i)
		}
		if transfer.Beneficiary.CPF == "" && transfer.Beneficiary.CNPJ == "" {
			return fmt.Errorf("repasse[%d]: CPF ou CNPJ do beneficiário é obrigatório", i)
		}
	}

	return nil
}

// QuickSplitConfig é um helper para criar uma configuração simples de split.
// myPercentage: a porcentagem que você mantém (0-100)
// partner: o beneficiário que recebe o resto
func QuickSplitConfig(description string, myPercentage float64, partner Beneficiary) SplitConfig {
	partnerPercentage := 100 - myPercentage

	return SplitConfig{
		Description: description,
		Immediate:   true,
		MyPart: SplitPart{
			Type:  SplitTypePercentage,
			Value: fmt.Sprintf("%.2f", myPercentage),
		},
		Transfers: []SplitPart{
			{
				Type:        SplitTypePercentage,
				Value:       fmt.Sprintf("%.2f", partnerPercentage),
				Beneficiary: &partner,
			},
		},
	}
}

// GymPartnerSplitConfig cria um split típico para academias parceiras.
// Exemplo: 70% para a academia principal, 30% para a academia parceira.
func GymPartnerSplitConfig(mainGymPercent float64, partnerCPFOrCNPJ, partnerName string) SplitConfig {
	partner := Beneficiary{Name: partnerName}
	if len(partnerCPFOrCNPJ) == 11 {
		partner.CPF = partnerCPFOrCNPJ
	} else {
		partner.CNPJ = partnerCPFOrCNPJ
	}

	return QuickSplitConfig(
		fmt.Sprintf("Split %s (%.0f%%)", partnerName, 100-mainGymPercent),
		mainGymPercent,
		partner,
	)
}
