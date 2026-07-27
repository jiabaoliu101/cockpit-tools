package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect(databaseURL string) error {
	var err error
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}
	log.Println("Connected to PostgreSQL database")
	return nil
}

func Migrate() error {
	schema := `
	CREATE EXTENSION IF NOT EXISTS pgcrypto;

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		nickname VARCHAR(100) DEFAULT '',
		is_active BOOLEAN DEFAULT TRUE,
		is_admin BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		last_login_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		token_hash VARCHAR(255) UNIQUE NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS subscription_plans (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) NOT NULL,
		duration_days INTEGER NOT NULL DEFAULT 30,
		max_accounts INTEGER NOT NULL DEFAULT 1,
		price_cents INTEGER NOT NULL DEFAULT 0,
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS user_subscriptions (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		plan_id UUID REFERENCES subscription_plans(id),
		status VARCHAR(20) NOT NULL DEFAULT 'active',
		starts_at TIMESTAMPTZ DEFAULT NOW(),
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS payments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		plan_id UUID REFERENCES subscription_plans(id),
		amount_cents INTEGER NOT NULL,
		currency VARCHAR(10) DEFAULT 'CNY',
		payment_method VARCHAR(50) DEFAULT '',
		gateway_order_id VARCHAR(255) DEFAULT '',
		status VARCHAR(20) DEFAULT 'pending',
		paid_at TIMESTAMPTZ,
		fulfilled_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS account_pool (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		platform VARCHAR(50) NOT NULL DEFAULT 'codex',
		email VARCHAR(255) DEFAULT '',
		access_token_encrypted TEXT NOT NULL DEFAULT '',
		refresh_token_encrypted TEXT NOT NULL DEFAULT '',
		token_expiry BIGINT DEFAULT 0,
		metadata JSONB DEFAULT '{}',
		status VARCHAR(20) DEFAULT 'available',
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS account_assignments (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		account_id UUID REFERENCES account_pool(id) ON DELETE CASCADE,
		status VARCHAR(20) DEFAULT 'active',
		assigned_at TIMESTAMPTZ DEFAULT NOW(),
		revoked_at TIMESTAMPTZ,
		UNIQUE(account_id)
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE SET NULL,
		action VARCHAR(100) NOT NULL,
		details JSONB DEFAULT '{}',
		ip_address VARCHAR(45) DEFAULT '',
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	-- Default subscription plans (prices in CNY fen: 1999 = ¥19.99)
	INSERT INTO subscription_plans (name, duration_days, max_accounts, price_cents) VALUES
		('Basic Monthly', 30, 1, 1999),
		('Pro Monthly', 30, 3, 4999),
		('Basic Yearly', 365, 1, 19999),
		('Pro Yearly', 365, 3, 49999)
	ON CONFLICT DO NOTHING;

	-- Additive changes for databases created before payment plans were bound to orders.
	ALTER TABLE payments ADD COLUMN IF NOT EXISTS plan_id UUID REFERENCES subscription_plans(id);
	ALTER TABLE payments ADD COLUMN IF NOT EXISTS fulfilled_at TIMESTAMPTZ;
	CREATE INDEX IF NOT EXISTS idx_payments_user_status ON payments(user_id, status);
	CREATE INDEX IF NOT EXISTS idx_account_pool_available ON account_pool(platform, status, created_at);
	CREATE INDEX IF NOT EXISTS idx_assignments_user_status ON account_assignments(user_id, status);
	`
	_, err := DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	log.Println("Database migration completed")
	return nil
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}
