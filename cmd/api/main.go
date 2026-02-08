// Package main é o ponto de entrada da API BlackBelt
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/magnani/black-belt-app/backend/internal/adapters/efi"
	"github.com/magnani/black-belt-app/backend/internal/config"
	"github.com/magnani/black-belt-app/backend/internal/handlers"
)

func main() {
	log.Println("🥋 Iniciando BlackBelt API...")

	// Carrega configurações
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Erro ao carregar configurações: %v", err)
	}

	log.Printf("📦 Ambiente: %s", cfg.Env)
	log.Printf("🔐 Efí Sandbox: %v", cfg.Efi.Sandbox)

	// Inicializa o cliente Efí (comentado até ter certificado)
	// Para desenvolvimento, podemos pular esta etapa
	var efiClient *efi.Client
	if _, err := os.Stat(cfg.Efi.CertificatePath); err == nil {
		// Certificado existe, inicializa o cliente
		pixKey := os.Getenv("EFI_PIX_KEY") // Chave PIX do estabelecimento
		efiClient, err = efi.NewClient(&cfg.Efi, pixKey)
		if err != nil {
			log.Printf("⚠️  Aviso: Erro ao inicializar cliente Efí: %v", err)
		} else {
			log.Println("✅ Cliente Efí inicializado com sucesso")
		}
	} else {
		log.Printf("⚠️  Aviso: Certificado não encontrado em %s", cfg.Efi.CertificatePath)
		log.Println("   O cliente Efí não será inicializado")
	}

	// Configura o router
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", handlers.HealthCheck)
	mux.HandleFunc("/api/health", handlers.HealthCheck)

	// Webhook Efí (só registra se o cliente foi inicializado)
	if efiClient != nil {
		webhookHandler := handlers.NewWebhookHandler(efiClient, cfg.Webhook.Secret)
		webhookHandler.RegisterHandler("pix", handlers.HandlePixReceived)
		mux.HandleFunc("/api/webhooks/efi", webhookHandler.HandleEfiWebhook)
		log.Println("📨 Webhook endpoint registrado: /api/webhooks/efi")
	}

	// Inicia o servidor
	addr := ":" + cfg.Port
	log.Printf("🚀 Servidor rodando em http://localhost%s", addr)
	log.Printf("🏥 Health check: http://localhost%s/health", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("❌ Erro ao iniciar servidor: %v", err)
	}
}
