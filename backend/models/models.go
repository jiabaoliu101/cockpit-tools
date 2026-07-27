package models

import "time"

// User represents a registered user
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Nickname     string    `json:"nickname"`
	IsActive     bool      `json:"is_active"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
	LastLoginAt  time.Time `json:"last_login_at"`
}

// SubscriptionPlan defines a purchasable subscription tier
type SubscriptionPlan struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DurationDays int   `json:"duration_days"`
	MaxAccounts int    `json:"max_accounts"`
	PriceCents  int    `json:"price_cents"` // in CNY fen
	IsActive    bool   `json:"is_active"`
}

// UserSubscription links a user to an active plan
type UserSubscription struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	PlanID    string    `json:"plan_id"`
	Status    string    `json:"status"` // active, expired, cancelled
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// Payment records a payment transaction
type Payment struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	AmountCents  int        `json:"amount_cents"`
	Currency     string     `json:"currency"`
	PaymentMethod string    `json:"payment_method"`
	GatewayOrderID string   `json:"gateway_order_id"`
	Status       string     `json:"status"` // pending, completed, failed, refunded
	PaidAt       *time.Time `json:"paid_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

// AccountPoolEntry is a pre-imported IDE account in the pool
type AccountPoolEntry struct {
	ID                   string    `json:"id"`
	Platform             string    `json:"platform"`
	Email                string    `json:"email"`
	AccessTokenEncrypted  string    `json:"-"`
	RefreshTokenEncrypted string    `json:"-"`
	TokenExpiry          int64     `json:"token_expiry"`
	Metadata             string    `json:"metadata"` // JSON blob
	Status               string    `json:"status"` // available, assigned, expired, disabled
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// AccountAssignment links a user to a pool account
type AccountAssignment struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	AccountID  string    `json:"account_id"`
	Status     string    `json:"status"` // active, revoked
	AssignedAt time.Time `json:"assigned_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
}

// AuditLog records administrative and security events
type AuditLog struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Action    string    `json:"action"`
	Details   string    `json:"details"` // JSON blob
	IPAddress string    `json:"ip_address"`
	CreatedAt time.Time `json:"created_at"`
}

// ── Request/Response DTOs ──

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         User   `json:"user"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type OrderRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

type OrderResponse struct {
	OrderID   string `json:"order_id"`
	QRCodeURL string `json:"qr_code_url"`
	Amount    int    `json:"amount_cents"`
}

type UserAccountResponse struct {
	ID          string `json:"id"`
	Platform    string `json:"platform"`
	Email       string `json:"email"`
	AccessToken string `json:"access_token"`
	TokenExpiry int64  `json:"token_expiry"`
}

type DashboardStats struct {
	TotalUsers        int `json:"total_users"`
	ActiveSubscriptions int `json:"active_subscriptions"`
	AvailableAccounts int `json:"available_accounts"`
	AssignedAccounts  int `json:"assigned_accounts"`
	MonthlyRevenue    int `json:"monthly_revenue_cents"`
}
