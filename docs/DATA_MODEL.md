# Modelo de Dados - BlackBelt Services

## Diagrama de Relacionamentos

```
┌─────────────┐       ┌──────────────────────┐
│   profiles  │──────▶│      academies       │
└─────────────┘       └──────────────────────┘
                               │
                               │ (1:1)
                               ▼
                      ┌──────────────────────┐
                      │    subscriptions     │
                      └──────────────────────┘
                               │
                               │ (N:1)
                               ▼
                      ┌──────────────────────┐
                      │ subscription_plans   │
                      └──────────────────────┘

┌──────────────────────┐
│   payment_history    │ ◀── (logs de pagamentos)
└──────────────────────┘

┌──────────────────────┐
│   webhook_events     │ ◀── (auditoria de webhooks)
└──────────────────────┘
```

---

## Tabelas

### 1. `profiles`

Perfis de usuários (sincronizado com Supabase Auth).

> **Nota:** Esta tabela já existe no projeto mobile. O backend apenas lê.

```sql
CREATE TABLE profiles (
    id UUID PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    full_name TEXT,
    avatar_url TEXT,
    role TEXT NOT NULL CHECK (role IN ('owner', 'professor', 'student')),
    
    -- Belt info (students)
    belt_color TEXT,
    belt_degrees INTEGER DEFAULT 0,
    
    -- Academy relation
    academy_id UUID REFERENCES academies(id),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

### 2. `academies`

Academias cadastradas na plataforma.

```sql
CREATE TABLE academies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Basic info
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL, -- URL-friendly name
    description TEXT,
    
    -- Owner
    owner_id UUID NOT NULL REFERENCES profiles(id),
    
    -- Contact
    phone TEXT,
    email TEXT,
    website TEXT,
    
    -- Address
    address_street TEXT,
    address_city TEXT,
    address_state TEXT,
    address_zip TEXT,
    address_country TEXT DEFAULT 'BR',
    
    -- Branding
    logo_url TEXT,
    cover_url TEXT,
    primary_color TEXT DEFAULT '#000000',
    
    -- Invite code
    invite_code TEXT UNIQUE NOT NULL,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_academies_owner ON academies(owner_id);
CREATE INDEX idx_academies_invite_code ON academies(invite_code);
CREATE INDEX idx_academies_slug ON academies(slug);
```

---

### 3. `subscription_plans`

Planos de assinatura disponíveis.

```sql
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Identification
    name TEXT NOT NULL,              -- "Starter", "Pro", "Business"
    slug TEXT UNIQUE NOT NULL,       -- "starter", "pro", "business"
    
    -- Pricing (centavos)
    price_monthly INTEGER NOT NULL,  -- R$ 99,00 = 9900
    price_yearly INTEGER,            -- Desconto anual (opcional)
    currency TEXT DEFAULT 'BRL',
    
    -- Limits
    max_students INTEGER,            -- NULL = unlimited
    max_professors INTEGER,
    max_locations INTEGER DEFAULT 1,
    
    -- Features (JSON array)
    features JSONB NOT NULL DEFAULT '[]',
    
    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default plans
INSERT INTO subscription_plans (name, slug, price_monthly, max_students, max_professors, features) VALUES
('Starter', 'starter', 9900, 50, 2, '["checkin", "schedule", "profiles"]'),
('Pro', 'pro', 19900, 200, 5, '["checkin", "schedule", "profiles", "analytics", "store"]'),
('Business', 'business', 39900, NULL, NULL, '["checkin", "schedule", "profiles", "analytics", "store", "api", "multi_location", "priority_support"]');
```

---

### 4. `subscriptions`

Assinaturas das academias.

```sql
CREATE TYPE subscription_status AS ENUM (
    'trialing',     -- Em período de trial
    'active',       -- Pagamento em dia
    'past_due',     -- Pagamento atrasado (grace period)
    'canceled',     -- Cancelada
    'expired'       -- Trial expirado sem conversão
);

CREATE TYPE payment_gateway AS ENUM (
    'pix_auto',     -- PIX Automático (Efí Bank)
    'stripe'        -- Stripe Billing
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Relationships
    academy_id UUID UNIQUE NOT NULL REFERENCES academies(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    
    -- Status
    status subscription_status NOT NULL DEFAULT 'trialing',
    
    -- Trial
    trial_start_date TIMESTAMPTZ DEFAULT NOW(),
    trial_end_date TIMESTAMPTZ,  -- NOW() + 20 days
    
    -- Gateway info
    payment_gateway payment_gateway,
    
    -- PIX Automático fields
    pix_authorization_id TEXT,
    pix_recurrence_id TEXT,
    pix_customer_cpf TEXT,
    
    -- Stripe fields
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    stripe_price_id TEXT,
    
    -- Billing period
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    
    -- Cancellation
    canceled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_academy ON subscriptions(academy_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_trial_end ON subscriptions(trial_end_date);
CREATE INDEX idx_subscriptions_period_end ON subscriptions(current_period_end);
```

---

### 5. `payment_history`

Histórico de pagamentos (sucesso e falhas).

```sql
CREATE TYPE payment_status AS ENUM (
    'pending',
    'processing',
    'succeeded',
    'failed',
    'refunded'
);

CREATE TABLE payment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Relationships
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    
    -- Amount (centavos)
    amount INTEGER NOT NULL,
    currency TEXT DEFAULT 'BRL',
    
    -- Gateway info
    payment_gateway payment_gateway NOT NULL,
    gateway_payment_id TEXT,        -- ID no gateway (pix txid ou stripe pi_xxx)
    gateway_charge_id TEXT,         -- ID da cobrança recorrente
    
    -- Status
    status payment_status NOT NULL DEFAULT 'pending',
    
    -- Details
    payment_method TEXT,            -- "pix" | "card"
    failure_reason TEXT,
    
    -- Period covered
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    
    -- Timestamps
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payment_history_subscription ON payment_history(subscription_id);
CREATE INDEX idx_payment_history_status ON payment_history(status);
CREATE INDEX idx_payment_history_gateway_id ON payment_history(gateway_payment_id);
CREATE INDEX idx_payment_history_created ON payment_history(created_at DESC);
```

---

### 6. `webhook_events`

Eventos de webhook recebidos (auditoria).

```sql
CREATE TYPE webhook_status AS ENUM (
    'pending',
    'processing',
    'processed',
    'failed',
    'skipped'
);

CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Source
    gateway TEXT NOT NULL,          -- "pix_auto" | "stripe"
    
    -- Event info
    event_id TEXT UNIQUE NOT NULL,  -- ID do evento no gateway
    event_type TEXT NOT NULL,       -- Tipo do evento
    
    -- Payload
    payload JSONB NOT NULL,
    
    -- Processing
    status webhook_status NOT NULL DEFAULT 'pending',
    processed_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_webhook_events_event_id ON webhook_events(event_id);
CREATE INDEX idx_webhook_events_type ON webhook_events(event_type);
CREATE INDEX idx_webhook_events_status ON webhook_events(status);
CREATE INDEX idx_webhook_events_gateway ON webhook_events(gateway);
CREATE INDEX idx_webhook_events_created ON webhook_events(created_at DESC);
```

---

## Triggers e Functions

### Auto-update `updated_at`

```sql
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to tables
CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON academies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER set_updated_at
    BEFORE UPDATE ON subscription_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
```

### Create subscription on academy creation

```sql
CREATE OR REPLACE FUNCTION handle_new_academy()
RETURNS TRIGGER AS $$
DECLARE
    starter_plan_id UUID;
BEGIN
    -- Get starter plan
    SELECT id INTO starter_plan_id 
    FROM subscription_plans 
    WHERE slug = 'starter' 
    LIMIT 1;
    
    -- Create trial subscription
    INSERT INTO subscriptions (
        academy_id, 
        plan_id, 
        status, 
        trial_start_date,
        trial_end_date
    ) VALUES (
        NEW.id,
        starter_plan_id,
        'trialing',
        NOW(),
        NOW() + INTERVAL '20 days'
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

CREATE TRIGGER on_academy_created
    AFTER INSERT ON academies
    FOR EACH ROW
    EXECUTE FUNCTION handle_new_academy();
```

---

## Migrations

### Migration 1: Base Schema

**`000001_init_schema.up.sql`**

```sql
-- Extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Enums
CREATE TYPE subscription_status AS ENUM ('trialing', 'active', 'past_due', 'canceled', 'expired');
CREATE TYPE payment_gateway AS ENUM ('pix_auto', 'stripe');
CREATE TYPE payment_status AS ENUM ('pending', 'processing', 'succeeded', 'failed', 'refunded');
CREATE TYPE webhook_status AS ENUM ('pending', 'processing', 'processed', 'failed', 'skipped');

-- Tables (in order of dependencies)
-- 1. subscription_plans
CREATE TABLE subscription_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    price_monthly INTEGER NOT NULL,
    price_yearly INTEGER,
    currency TEXT DEFAULT 'BRL',
    max_students INTEGER,
    max_professors INTEGER,
    max_locations INTEGER DEFAULT 1,
    features JSONB NOT NULL DEFAULT '[]',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. academies (profiles já existe)
CREATE TABLE academies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT,
    owner_id UUID NOT NULL,
    phone TEXT,
    email TEXT,
    website TEXT,
    address_street TEXT,
    address_city TEXT,
    address_state TEXT,
    address_zip TEXT,
    address_country TEXT DEFAULT 'BR',
    logo_url TEXT,
    cover_url TEXT,
    primary_color TEXT DEFAULT '#000000',
    invite_code TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_academies_owner ON academies(owner_id);
CREATE INDEX idx_academies_invite_code ON academies(invite_code);

-- 3. subscriptions
CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    academy_id UUID UNIQUE NOT NULL REFERENCES academies(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES subscription_plans(id),
    status subscription_status NOT NULL DEFAULT 'trialing',
    trial_start_date TIMESTAMPTZ DEFAULT NOW(),
    trial_end_date TIMESTAMPTZ,
    payment_gateway payment_gateway,
    pix_authorization_id TEXT,
    pix_recurrence_id TEXT,
    pix_customer_cpf TEXT,
    stripe_customer_id TEXT,
    stripe_subscription_id TEXT,
    stripe_price_id TEXT,
    current_period_start TIMESTAMPTZ,
    current_period_end TIMESTAMPTZ,
    canceled_at TIMESTAMPTZ,
    cancel_reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_academy ON subscriptions(academy_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
CREATE INDEX idx_subscriptions_trial_end ON subscriptions(trial_end_date);

-- 4. payment_history
CREATE TABLE payment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    amount INTEGER NOT NULL,
    currency TEXT DEFAULT 'BRL',
    payment_gateway payment_gateway NOT NULL,
    gateway_payment_id TEXT,
    gateway_charge_id TEXT,
    status payment_status NOT NULL DEFAULT 'pending',
    payment_method TEXT,
    failure_reason TEXT,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_payment_history_subscription ON payment_history(subscription_id);
CREATE INDEX idx_payment_history_status ON payment_history(status);

-- 5. webhook_events
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gateway TEXT NOT NULL,
    event_id TEXT UNIQUE NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status webhook_status NOT NULL DEFAULT 'pending',
    processed_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_webhook_events_event_id ON webhook_events(event_id);
CREATE INDEX idx_webhook_events_type ON webhook_events(event_type);
CREATE INDEX idx_webhook_events_status ON webhook_events(status);

-- Insert default plans
INSERT INTO subscription_plans (name, slug, price_monthly, max_students, max_professors, features) VALUES
('Starter', 'starter', 9900, 50, 2, '["checkin", "schedule", "profiles"]'),
('Pro', 'pro', 19900, 200, 5, '["checkin", "schedule", "profiles", "analytics", "store"]'),
('Business', 'business', 39900, NULL, NULL, '["checkin", "schedule", "profiles", "analytics", "store", "api", "multi_location", "priority_support"]');
```

**`000001_init_schema.down.sql`**

```sql
DROP TABLE IF EXISTS webhook_events CASCADE;
DROP TABLE IF EXISTS payment_history CASCADE;
DROP TABLE IF EXISTS subscriptions CASCADE;
DROP TABLE IF EXISTS academies CASCADE;
DROP TABLE IF EXISTS subscription_plans CASCADE;

DROP TYPE IF EXISTS webhook_status;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS payment_gateway;
DROP TYPE IF EXISTS subscription_status;
```

---

## RLS Policies (Supabase)

```sql
-- Academies: owner pode tudo, membros podem ler
ALTER TABLE academies ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Owner can manage academy"
    ON academies FOR ALL
    USING (owner_id = auth.uid());

CREATE POLICY "Members can view academy"
    ON academies FOR SELECT
    USING (
        id IN (
            SELECT academy_id FROM profiles WHERE id = auth.uid()
        )
    );

-- Subscriptions: apenas owner pode ver
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Owner can view subscription"
    ON subscriptions FOR SELECT
    USING (
        academy_id IN (
            SELECT id FROM academies WHERE owner_id = auth.uid()
        )
    );

-- Payment history: apenas owner
ALTER TABLE payment_history ENABLE ROW LEVEL SECURITY;

CREATE POLICY "Owner can view payments"
    ON payment_history FOR SELECT
    USING (
        subscription_id IN (
            SELECT s.id FROM subscriptions s
            JOIN academies a ON a.id = s.academy_id
            WHERE a.owner_id = auth.uid()
        )
    );
```

---

## Queries Úteis

### Verificar trial expirando

```sql
-- Academias com trial expirando em 3 dias
SELECT 
    a.name,
    a.email,
    s.trial_end_date,
    s.trial_end_date - NOW() AS days_remaining
FROM subscriptions s
JOIN academies a ON a.id = s.academy_id
WHERE s.status = 'trialing'
  AND s.trial_end_date BETWEEN NOW() AND NOW() + INTERVAL '3 days';
```

### Verificar assinaturas vencidas

```sql
-- Assinaturas past_due há mais de 7 dias
SELECT 
    a.name,
    s.current_period_end,
    NOW() - s.current_period_end AS days_overdue
FROM subscriptions s
JOIN academies a ON a.id = s.academy_id
WHERE s.status = 'past_due'
  AND s.current_period_end < NOW() - INTERVAL '7 days';
```

### MRR (Monthly Recurring Revenue)

```sql
SELECT 
    SUM(sp.price_monthly) / 100.0 AS mrr_brl
FROM subscriptions s
JOIN subscription_plans sp ON sp.id = s.plan_id
WHERE s.status = 'active';
```

---

*Atualizado: 2026-02-08*
