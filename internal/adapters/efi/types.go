// Package efi implementa o adaptador para a API Efí Bank (antiga Gerencianet)
package efi

import "time"

// TokenResponse representa a resposta do endpoint de autenticação OAuth2
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// PixCalendario define o calendário de uma cobrança PIX
type PixCalendario struct {
	Criacao   string `json:"criacao,omitempty"`
	Expiracao int    `json:"expiracao"` // Tempo em segundos até expirar
}

// PixDevedor representa os dados do devedor/pagador
type PixDevedor struct {
	CPF   string `json:"cpf,omitempty"`
	CNPJ  string `json:"cnpj,omitempty"`
	Nome  string `json:"nome,omitempty"`
	Email string `json:"email,omitempty"`
}

// Debtor é um alias para PixDevedor (compatibilidade com PIX Automático)
type Debtor = PixDevedor

// Payer é um alias para PixDevedor (compatibilidade com webhooks)
type Payer = PixDevedor

// PixValor representa o valor da cobrança
type PixValor struct {
	Original string `json:"original"` // Valor como string com 2 casas decimais (ex: "100.00")
}

// PixInfoAdicional representa informações adicionais do PIX
type PixInfoAdicional struct {
	Nome  string `json:"nome"`
	Valor string `json:"valor"`
}

// PixCobRequest representa uma requisição para criar cobrança PIX imediata
type PixCobRequest struct {
	Calendario     PixCalendario      `json:"calendario"`
	Devedor        *PixDevedor        `json:"devedor,omitempty"`
	Valor          PixValor           `json:"valor"`
	Chave          string             `json:"chave"` // Chave PIX do recebedor
	SolicitacaoPag string             `json:"solicitacaoPagador,omitempty"`
	InfoAdicionais []PixInfoAdicional `json:"infoAdicionais,omitempty"`
}

// PixCobResponse representa a resposta de uma cobrança PIX criada
type PixCobResponse struct {
	Calendario PixCalendario `json:"calendario"`
	TxID       string        `json:"txid"`
	Revisao    int           `json:"revisao"`
	Location   string        `json:"loc,omitempty"`
	Status     string        `json:"status"` // ATIVA, CONCLUIDA, REMOVIDA_PELO_USUARIO_RECEBEDOR, REMOVIDA_PELO_PSP
	Devedor    *PixDevedor   `json:"devedor,omitempty"`
	Valor      PixValor      `json:"valor"`
	Chave      string        `json:"chave"`
	PixCopiaECola string     `json:"pixCopiaECola,omitempty"`
}

// PixWebhook representa os dados de um webhook configurado
type PixWebhook struct {
	WebhookURL string    `json:"webhookUrl"`
	Chave      string    `json:"chave"`
	CriadoEm   time.Time `json:"criacao,omitempty"`
}

// PixWebhookPayload representa o payload recebido em um webhook PIX
type PixWebhookPayload struct {
	Pix []PixRecebido `json:"pix"`
}

// PixRecebido representa um PIX recebido notificado via webhook
type PixRecebido struct {
	EndToEndID string    `json:"endToEndId"` // ID único da transação no SPI
	TxID       string    `json:"txid"`       // ID da cobrança
	Chave      string    `json:"chave"`      // Chave PIX que recebeu
	Valor      string    `json:"valor"`      // Valor recebido
	Horario    time.Time `json:"horario"`    // Momento do recebimento
	Pagador    struct {
		CPF  string `json:"cpf,omitempty"`
		CNPJ string `json:"cnpj,omitempty"`
		Nome string `json:"nome"`
	} `json:"pagador"`
	InfoPagador string `json:"infoPagador,omitempty"`
}

// PixDevolucao representa uma devolução de PIX
type PixDevolucao struct {
	ID     string `json:"id"`
	RTrId  string `json:"rtrId"` // ID de retorno
	Valor  string `json:"valor"`
	Status string `json:"status"` // EM_PROCESSAMENTO, DEVOLVIDO, NAO_REALIZADO
	Motivo string `json:"motivo,omitempty"`
}

// PixDevolucaoRequest representa a requisição de devolução
type PixDevolucaoRequest struct {
	Valor string `json:"valor"` // Valor a devolver
}

// APIError representa um erro retornado pela API Efí
type APIError struct {
	Nome     string `json:"nome"`
	Mensagem string `json:"mensagem"`
	Type     string `json:"type,omitempty"`
	Title    string `json:"title,omitempty"`
	Status   int    `json:"status,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// Error implementa a interface error
func (e *APIError) Error() string {
	if e.Mensagem != "" {
		return e.Mensagem
	}
	if e.Detail != "" {
		return e.Detail
	}
	return e.Nome
}

// PixLocation representa um location (payload do QR Code)
type PixLocation struct {
	ID       int    `json:"id"`
	Location string `json:"location"`
	TipoCob  string `json:"tipoCob"`
	CriadoEm string `json:"criacao"`
}

// QRCodeResponse representa a resposta do endpoint de QR Code
type QRCodeResponse struct {
	QRCode       string `json:"qrcode"`        // Imagem em base64
	ImagemQRCode string `json:"imagemQrcode"`  // URL da imagem
}

// ==================== PIX Automático (Recorrência) ====================

// Periodicity define a frequência da recorrência
type Periodicity string

const (
	PeriodicityWeekly     Periodicity = "SEMANAL"
	PeriodicityBiweekly   Periodicity = "QUINZENAL"
	PeriodicityMonthly    Periodicity = "MENSAL"
	PeriodicityBimonthly  Periodicity = "BIMESTRAL"
	PeriodicityQuarterly  Periodicity = "TRIMESTRAL"
	PeriodicitySemiannual Periodicity = "SEMESTRAL"
	PeriodicityYearly     Periodicity = "ANUAL"
)

// RecurrenceStatus define os status possíveis de uma recorrência
type RecurrenceStatus string

const (
	RecurrenceStatusCreated   RecurrenceStatus = "CRIADA"
	RecurrenceStatusApproved  RecurrenceStatus = "APROVADA"
	RecurrenceStatusRejected  RecurrenceStatus = "REJEITADA"
	RecurrenceStatusCancelled RecurrenceStatus = "CANCELADA"
	RecurrenceStatusExpired   RecurrenceStatus = "EXPIRADA"
)

// CreateRecurrenceRequest é a requisição para criar uma recorrência PIX
type CreateRecurrenceRequest struct {
	Contract    string      `json:"contrato"`           // Identificador único do contrato
	Debtor      PixDevedor  `json:"devedor"`            // Dados do devedor
	Object      string      `json:"objeto"`             // Descrição do objeto
	StartDate   string      `json:"dataInicial"`        // Data inicial (YYYY-MM-DD)
	EndDate     string      `json:"dataFinal"`          // Data final (YYYY-MM-DD)
	Periodicity Periodicity `json:"periodicidade"`      // Frequência
	Amount      string      `json:"valorRec"`           // Valor (ex: "100.00")
	Description string      `json:"descricao,omitempty"`
	DueDay      int         `json:"diaVencimento,omitempty"` // Dia do vencimento (1-28)
}

// UpdateRecurrenceRequest é a requisição para atualizar uma recorrência
type UpdateRecurrenceRequest struct {
	Amount  string `json:"valorRec,omitempty"`
	EndDate string `json:"dataFinal,omitempty"`
	Status  string `json:"status,omitempty"`
}

// Recurrence representa uma autorização de recorrência PIX
type Recurrence struct {
	ID           string           `json:"idRec"`
	Contract     string           `json:"contrato"`
	Status       RecurrenceStatus `json:"status"`
	QRCode       string           `json:"pixCopiaECola"`
	Location     string           `json:"location"`
	TxID         string           `json:"txid,omitempty"`
	Amount       string           `json:"valorRec"`
	Periodicity  Periodicity      `json:"periodicidade"`
	StartDate    string           `json:"dataInicial"`
	EndDate      string           `json:"dataFinal"`
	NextDueDate  string           `json:"proximoVencimento,omitempty"`
	CreatedAt    string           `json:"criacao"`
	Debtor       PixDevedor       `json:"devedor"`
}

// RecurrenceListResponse é a resposta de listagem de recorrências
type RecurrenceListResponse struct {
	Recurrences []Recurrence `json:"recorrencias"`
	Total       int          `json:"total"`
}

// RecurrenceEvent representa um evento de mudança de status de recorrência
type RecurrenceEvent struct {
	ID        string           `json:"idRec"`
	Contract  string           `json:"contrato"`
	Status    RecurrenceStatus `json:"status"`
	Timestamp string           `json:"timestamp"`
	Reason    string           `json:"motivo,omitempty"`
}

// ==================== Split de Pagamento ====================

// SplitType define o tipo de cálculo do split
type SplitType string

const (
	SplitTypePercentage SplitType = "porcentagem"
	SplitTypeFixed      SplitType = "valor"
)

// Beneficiary representa um beneficiário do split
type Beneficiary struct {
	CPF  string `json:"cpf,omitempty"`
	CNPJ string `json:"cnpj,omitempty"`
	Bank string `json:"banco,omitempty"`
	Name string `json:"nome,omitempty"`
}

// SplitPart representa uma parte em uma configuração de split
type SplitPart struct {
	Type        SplitType    `json:"tipo"`
	Value       string       `json:"valor"`
	Beneficiary *Beneficiary `json:"favorecido,omitempty"`
}

// SplitConfig é a configuração de split de pagamento
type SplitConfig struct {
	Description string      `json:"descricao"`
	Immediate   bool        `json:"imediato"` // true = split imediato, false = D+1
	MyPart      SplitPart   `json:"minhaParte"`
	Transfers   []SplitPart `json:"repasses"`
}

// SplitConfigResponse é a resposta após criar um split config
type SplitConfigResponse struct {
	ID          string    `json:"id"`
	Description string    `json:"descricao"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"criacao"`
}

// ==================== Abertura de Contas (API Restrita) ====================

// Address representa um endereço físico
type Address struct {
	Street       string `json:"logradouro"`
	Number       string `json:"numero"`
	Complement   string `json:"complemento,omitempty"`
	Neighborhood string `json:"bairro"`
	City         string `json:"cidade"`
	State        string `json:"uf"`
	ZipCode      string `json:"cep"`
}

// CreateAccountRequest é a requisição para criar uma nova conta
type CreateAccountRequest struct {
	CPF       string  `json:"cpf,omitempty"`
	CNPJ      string  `json:"cnpj,omitempty"`
	Name      string  `json:"nome"`
	Email     string  `json:"email"`
	BirthDate string  `json:"dataNascimento,omitempty"`
	Phone     string  `json:"telefone,omitempty"`
	Address   Address `json:"endereco"`
}

// Account representa uma conta criada
type Account struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"criacao"`
}

// AccountStatus representa o status de uma conta
type AccountStatus struct {
	ID      string `json:"id"`
	Status  string `json:"status"` // PENDENTE, ATIVO, BLOQUEADO, CANCELADO
	Message string `json:"mensagem,omitempty"`
	Details string `json:"detalhes,omitempty"`
}

// ==================== Webhook Types ====================

// WebhookEventType define o tipo de evento de webhook
type WebhookEventType string

const (
	WebhookEventPix          WebhookEventType = "pix"
	WebhookEventRecurrence   WebhookEventType = "rec"
	WebhookEventRecApproved  WebhookEventType = "rec_aprovada"
	WebhookEventRecRejected  WebhookEventType = "rec_rejeitada"
	WebhookEventRecCancelled WebhookEventType = "rec_cancelada"
)

// PixPayment representa um pagamento PIX para webhook
type PixPayment struct {
	EndToEndID   string     `json:"endToEndId"`
	TxID         string     `json:"txid"`
	Value        string     `json:"valor"`
	Payer        PixDevedor `json:"pagador"`
	PaymentTime  string     `json:"horario"`
	Info         string     `json:"infoPagador,omitempty"`
	RecurrenceID string     `json:"idRec,omitempty"` // Se veio de recorrência
}

// WebhookEvent representa o payload recebido em um webhook da Efí
// (compatível com a estrutura oficial)
type WebhookEvent struct {
	Type      WebhookEventType `json:"tipo"`
	Timestamp string           `json:"timestamp"`
	Pix       []PixPayment     `json:"pix,omitempty"`
	Rec       *RecurrenceEvent `json:"rec,omitempty"`
}
