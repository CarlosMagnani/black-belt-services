package efi

import (
	"errors"
	"fmt"
	"net/http"
)

// Códigos de erro comuns da API Efí
const (
	ErrCodeInvalidToken     = "invalid_token"
	ErrCodeExpiredToken     = "expired_token"
	ErrCodeInvalidRequest   = "invalid_request"
	ErrCodeInvalidGrant     = "invalid_grant"
	ErrCodeNotFound         = "not_found"
	ErrCodeConflict         = "conflict"
	ErrCodeInvalidValue     = "valor_invalido"
	ErrCodeInvalidCPF       = "cpf_invalido"
	ErrCodeInvalidCNPJ      = "cnpj_invalido"
	ErrCodeRecurrenceExists = "recorrencia_duplicada"
)

// Erros sentinela para condições comuns
var (
	// ErrNotFound indica que o recurso não foi encontrado
	ErrNotFound = errors.New("efi: recurso não encontrado")

	// ErrUnauthorized indica falha de autenticação
	ErrUnauthorized = errors.New("efi: não autorizado")

	// ErrInvalidRequest indica requisição inválida
	ErrInvalidRequest = errors.New("efi: requisição inválida")

	// ErrRecurrenceRejected indica que a recorrência foi rejeitada pelo pagador
	ErrRecurrenceRejected = errors.New("efi: recorrência rejeitada")

	// ErrRecurrenceCancelled indica que a recorrência foi cancelada
	ErrRecurrenceCancelled = errors.New("efi: recorrência cancelada")

	// ErrRecurrenceExpired indica que a autorização de recorrência expirou
	ErrRecurrenceExpired = errors.New("efi: recorrência expirada")

	// ErrDuplicateRecurrence indica que já existe uma recorrência com este contrato
	ErrDuplicateRecurrence = errors.New("efi: recorrência duplicada")

	// ErrRateLimited indica rate limiting
	ErrRateLimited = errors.New("efi: rate limit atingido")

	// ErrServerError indica erro interno do servidor Efí
	ErrServerError = errors.New("efi: erro do servidor")
)

// IsNotFound retorna true se o erro indica que o recurso não foi encontrado
func IsNotFound(err error) bool {
	if errors.Is(err, ErrNotFound) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status == http.StatusNotFound
	}
	return false
}

// IsUnauthorized retorna true se o erro indica falha de autenticação
func IsUnauthorized(err error) bool {
	if errors.Is(err, ErrUnauthorized) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status == http.StatusUnauthorized
	}
	return false
}

// IsRateLimited retorna true se o erro indica rate limiting
func IsRateLimited(err error) bool {
	if errors.Is(err, ErrRateLimited) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status == http.StatusTooManyRequests
	}
	return false
}

// IsServerError retorna true se o erro é do servidor (5xx)
func IsServerError(err error) bool {
	if errors.Is(err, ErrServerError) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Status >= 500
	}
	return false
}

// IsDuplicateRecurrence retorna true se o erro indica recorrência duplicada
func IsDuplicateRecurrence(err error) bool {
	if errors.Is(err, ErrDuplicateRecurrence) {
		return true
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.Nome == ErrCodeRecurrenceExists || apiErr.Status == http.StatusConflict
	}
	return false
}

// ClassifyError converte um erro da API para um erro sentinela quando apropriado
func ClassifyError(err error) error {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return err
	}

	switch apiErr.Status {
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, apiErr.Error())
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", ErrUnauthorized, apiErr.Error())
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", ErrRateLimited, apiErr.Error())
	case http.StatusConflict:
		return fmt.Errorf("%w: %s", ErrDuplicateRecurrence, apiErr.Error())
	}

	if apiErr.Status >= 500 {
		return fmt.Errorf("%w: %s", ErrServerError, apiErr.Error())
	}

	switch apiErr.Nome {
	case ErrCodeRecurrenceExists:
		return fmt.Errorf("%w: %s", ErrDuplicateRecurrence, apiErr.Error())
	}

	return err
}

// RecurrenceStatusError representa um erro relacionado ao status da recorrência
type RecurrenceStatusError struct {
	RecurrenceID string
	Status       RecurrenceStatus
	Reason       string
}

func (e *RecurrenceStatusError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("recorrência %s está %s: %s", e.RecurrenceID, e.Status, e.Reason)
	}
	return fmt.Sprintf("recorrência %s está %s", e.RecurrenceID, e.Status)
}

// NewRecurrenceStatusError cria um novo RecurrenceStatusError
func NewRecurrenceStatusError(id string, status RecurrenceStatus, reason string) *RecurrenceStatusError {
	return &RecurrenceStatusError{
		RecurrenceID: id,
		Status:       status,
		Reason:       reason,
	}
}

// IsRejected retorna true se a recorrência foi rejeitada
func (e *RecurrenceStatusError) IsRejected() bool {
	return e.Status == RecurrenceStatusRejected
}

// IsCancelled retorna true se a recorrência foi cancelada
func (e *RecurrenceStatusError) IsCancelled() bool {
	return e.Status == RecurrenceStatusCancelled
}

// IsExpired retorna true se a recorrência expirou
func (e *RecurrenceStatusError) IsExpired() bool {
	return e.Status == RecurrenceStatusExpired
}

// ValidationError representa um erro de validação com detalhes do campo
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("erro de validação no campo '%s': %s", e.Field, e.Message)
}

// NewValidationError cria um novo ValidationError
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// WrapAPIError envolve um erro com contexto adicional
func WrapAPIError(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("efi %s: %w", operation, err)
}
