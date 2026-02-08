# Executive Summary - black-belt-services

## 📋 Resumo Executivo

Backend API em Go para **BlackBelt** — SaaS de gestão para academias de Jiu-Jitsu.

**Modelo de negócio: B2B2C**
- BlackBelt cobra **assinatura mensal da ACADEMIA** (não do aluno)
- Academia gerencia cobrança dos próprios alunos
- Sem split payment — assinatura simples

---

## 🎯 Objetivos do Sistema

1. **Gestão de Academias** — CRUD de academias, professores, alunos
2. **Assinaturas** — Cobrar mensalidade das academias via PIX Automático ou Stripe
3. **Autenticação** — Integração com Supabase Auth
4. **Webhooks** — Processar eventos de pagamento em tempo real
5. **API REST** — Servir o app mobile (Expo/React Native)

---

## 🏗️ Arquitetura

### Modelo de Pagamento: Assinatura Recorrente

```
Academia assina BlackBelt (R$99-399/mês)
    ↓
[PIX Automático ou Stripe Billing]
    ↓
BlackBelt recebe 100% da assinatura
    ↓
Academia usa o app para gerenciar seus alunos
```

**Por que este modelo?**
- Simplicidade — sem split complexo
- Previsibilidade — receita recorrente (MRR)
- Valor claro — academia paga pelo software, não por transação
- Menor churn — não depende de volume de alunos

### Opções de Cobrança

| Gateway        | Vantagem                           | Taxa        |
|----------------|-------------------------------------|-------------|
| PIX Automático | Não compromete limite cartão       | ~0.5-1%     |
| Stripe Billing | Cartões internacionais, portal     | ~3.5% + R$0.40 |

**Recomendado:** PIX Automático para Brasil (lançado 16/06/2025)

---

## 📊 Modelo de Dados

### Tabelas Principais

1. **`users`** — Usuários (profiles sincronizados com Supabase Auth)
2. **`academies`** — Academias cadastradas
3. **`subscriptions`** — Assinaturas das academias
4. **`subscription_plans`** — Planos disponíveis (Starter, Pro, Business)
5. **`payments`** — Histórico de cobranças
6. **`webhook_events`** — Eventos recebidos (PIX/Stripe)

### Relacionamentos

```
users → academies (owner)
academies → subscriptions → subscription_plans
subscriptions → payments
webhook_events → (atualiza subscriptions, payments)
```

---

## 🔄 Fluxos Principais

### 1. Onboarding de Academia

```
1. Dono cria conta (Supabase Auth)
2. Preenche dados da academia
3. Trial de 14 dias (sem cartão)
4. Fim do trial → escolhe plano
5. Autoriza PIX Automático ou cadastra cartão
6. Assinatura ativa!
```

**Tempo estimado:** 5 minutos

### 2. Renovação de Assinatura

```
1. Data de vencimento chega
2. Gateway tenta cobrança automática
3. Sucesso → subscription.renewed (webhook)
4. Falha → até 3 retentativas
5. Todas falham → subscription.past_due
6. 7 dias sem pagamento → subscription.canceled
```

### 3. Upgrade/Downgrade

```
1. Academia acessa portal de assinatura
2. Escolhe novo plano
3. Proration calculado automaticamente
4. Próxima cobrança ajustada
```

---

## 🛠️ Stack Técnica

| Componente      | Tecnologia                    |
|-----------------|-------------------------------|
| **Linguagem**   | Go 1.22+                      |
| **Framework**   | Chi (router HTTP)             |
| **Database**    | Supabase (PostgreSQL + RLS)   |
| **Auth**        | Supabase Auth (JWT)           |
| **Pagamentos**  | PIX Automático + Stripe       |
| **Deploy**      | Docker + Railway/Fly.io       |

---

## 📁 Estrutura do Projeto

```
black-belt-services/
├── cmd/
│   └── api/
│       └── main.go              # Entrypoint
├── internal/
│   ├── domain/                  # Entidades de negócio
│   │   ├── academy.go
│   │   ├── subscription.go
│   │   └── user.go
│   ├── ports/                   # Interfaces (ports)
│   │   ├── billing.go
│   │   ├── repository.go
│   │   └── auth.go
│   ├── adapters/                # Implementações (adapters)
│   │   ├── pix_automatico.go
│   │   ├── stripe_billing.go
│   │   └── supabase_repo.go
│   ├── handlers/                # HTTP handlers
│   │   ├── academy.go
│   │   ├── subscription.go
│   │   └── webhook.go
│   └── service/                 # Lógica de negócio
│       ├── academy_service.go
│       └── subscription_service.go
├── pkg/
│   └── middleware/              # Middlewares HTTP
├── docs/                        # Documentação
└── docker-compose.yml
```

---

## 🔐 Segurança

1. **JWT Validation** — Tokens Supabase validados em cada request
2. **RLS no Supabase** — Row-Level Security para isolamento de dados
3. **Webhook Signatures** — Validar assinatura Stripe/PIX
4. **Rate Limiting** — Proteger endpoints públicos
5. **Secrets Management** — ENV vars, nunca hardcoded

---

## 📈 Planos de Assinatura

| Plano       | Alunos   | Preço/mês | Funcionalidades                    |
|-------------|----------|-----------|-------------------------------------|
| **Starter** | até 50   | R$ 99     | Check-in, Grade, Perfil            |
| **Pro**     | até 200  | R$ 199    | + Loja, Relatórios                 |
| **Business**| ilimitado| R$ 399    | + API, Multi-unidade, Suporte VIP  |

---

## 🚀 Roadmap Backend

### Fase 1: MVP (Sprint 1-2)
- [ ] Setup projeto Go + Chi
- [ ] Integração Supabase (auth + db)
- [ ] CRUD Academies
- [ ] Endpoints básicos

### Fase 2: Billing (Sprint 3-4)
- [ ] Integração PIX Automático (Efí Bank)
- [ ] Stripe Billing como fallback
- [ ] Webhooks de pagamento
- [ ] Portal de assinatura

### Fase 3: Escala (Sprint 5+)
- [ ] Cache Redis
- [ ] Background jobs
- [ ] Métricas/Observability
- [ ] Multi-tenancy otimizado

---

## 📚 Documentos Relacionados

- [ARCHITECTURE.md](./ARCHITECTURE.md) — Detalhes técnicos
- [DATA_MODEL.md](./DATA_MODEL.md) — Schema do banco
- [FLOWS.md](./FLOWS.md) — Diagramas de fluxo
- [MONETIZATION.md](./MONETIZATION.md) — Modelo de monetização detalhado
- [SECURITY_CHECKLIST.md](./SECURITY_CHECKLIST.md) — Checklist de segurança

---

*Atualizado: 2026-02-08*
