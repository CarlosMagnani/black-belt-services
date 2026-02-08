// Package config gerencia as configurações do aplicativo
// carregando variáveis de ambiente do arquivo .env
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config armazena todas as configurações da aplicação
type Config struct {
	// Servidor
	Port string
	Env  string

	// Efí Bank
	Efi EfiConfig

	// Webhook
	Webhook WebhookConfig
}

// EfiConfig armazena configurações específicas da Efí Bank
type EfiConfig struct {
	ClientID            string
	ClientSecret        string
	CertificatePath     string
	CertificatePassword string
	Sandbox             bool
	PixURL              string
}

// WebhookConfig armazena configurações de webhook
type WebhookConfig struct {
	URL    string
	Secret string
}

// Load carrega as configurações do arquivo .env e variáveis de ambiente
// O arquivo .env é opcional - variáveis de ambiente têm prioridade
func Load() (*Config, error) {
	// Tenta carregar .env (ignora erro se não existir)
	_ = godotenv.Load()

	cfg := &Config{
		Port: getEnv("PORT", "8080"),
		Env:  getEnv("ENV", "development"),
		Efi: EfiConfig{
			ClientID:            getEnv("EFI_CLIENT_ID", ""),
			ClientSecret:        getEnv("EFI_CLIENT_SECRET", ""),
			CertificatePath:     getEnv("EFI_CERTIFICATE_PATH", ""),
			CertificatePassword: getEnv("EFI_CERTIFICATE_PASSWORD", ""),
			Sandbox:             getEnvBool("EFI_SANDBOX", true),
			PixURL:              getEnv("EFI_PIX_URL", "https://pix-h.api.efipay.com.br"),
		},
		Webhook: WebhookConfig{
			URL:    getEnv("WEBHOOK_URL", ""),
			Secret: getEnv("WEBHOOK_SECRET", ""),
		},
	}

	// Validação básica
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate verifica se as configurações obrigatórias estão presentes
func (c *Config) validate() error {
	if c.Efi.ClientID == "" {
		return fmt.Errorf("EFI_CLIENT_ID é obrigatório")
	}
	if c.Efi.ClientSecret == "" {
		return fmt.Errorf("EFI_CLIENT_SECRET é obrigatório")
	}
	if c.Efi.CertificatePath == "" {
		return fmt.Errorf("EFI_CERTIFICATE_PATH é obrigatório")
	}
	return nil
}

// IsDevelopment retorna true se estiver em ambiente de desenvolvimento
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// IsProduction retorna true se estiver em ambiente de produção
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// getEnv obtém uma variável de ambiente ou retorna o valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool obtém uma variável de ambiente como bool
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
