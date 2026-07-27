package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jiabaoliu101/aiswitch/backend/database"
	"github.com/jiabaoliu101/aiswitch/backend/models"
)

// ── Subscription Handlers ──

func GetSubscriptionPlans(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, name, duration_days, max_accounts, price_cents, is_active
		 FROM subscription_plans WHERE is_active = true ORDER BY price_cents`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var plans []models.SubscriptionPlan
	for rows.Next() {
		var p models.SubscriptionPlan
		if err := rows.Scan(&p.ID, &p.Name, &p.DurationDays, &p.MaxAccounts, &p.PriceCents, &p.IsActive); err != nil {
			continue
		}
		plans = append(plans, p)
	}
	c.JSON(http.StatusOK, plans)
}

func GetCurrentSubscription(c *gin.Context) {
	userID := c.GetString("user_id")
	var sub struct {
		ID          string    `json:"id"`
		PlanName    string    `json:"plan_name"`
		MaxAccounts int       `json:"max_accounts"`
		Status      string    `json:"status"`
		ExpiresAt   time.Time `json:"expires_at"`
		AssignedCount int     `json:"assigned_count"`
	}
	err := database.DB.QueryRow(
		`SELECT us.id, sp.name, sp.max_accounts, us.status, us.expires_at
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON us.plan_id = sp.id
		 WHERE us.user_id = $1 AND us.status = 'active'
		 ORDER BY us.expires_at DESC LIMIT 1`,
		userID,
	).Scan(&sub.ID, &sub.PlanName, &sub.MaxAccounts, &sub.Status, &sub.ExpiresAt)
	if err != nil {
		// No active subscription
		c.JSON(http.StatusOK, gin.H{
			"has_subscription": false,
			"subscription":     nil,
		})
		return
	}

	// Count assigned accounts
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM account_assignments WHERE user_id = $1 AND status = 'active'`,
		userID,
	).Scan(&sub.AssignedCount)

	c.JSON(http.StatusOK, gin.H{
		"has_subscription": true,
		"subscription":     sub,
	})
}

func CreateOrder(c *gin.Context) {
	var req models.OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")

	// Get plan price
	var priceCents int
	err := database.DB.QueryRow(
		`SELECT price_cents FROM subscription_plans WHERE id = $1 AND is_active = true`, req.PlanID,
	).Scan(&priceCents)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found or inactive"})
		return
	}

	// Create payment record (pending)
	var orderID string
	err = database.DB.QueryRow(
		`INSERT INTO payments (user_id, amount_cents, currency, status) VALUES ($1, $2, 'CNY', 'pending') RETURNING id`,
		userID, priceCents,
	).Scan(&orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	// TODO: Integrate with real payment gateway (PayJS/XorPay)
	// For now, return a mock QR code URL
	qrCodeURL := "https://api.payjs.cn/native?mchid=demo&total_fee=" + itoa(priceCents) + "&out_trade_no=" + orderID

	c.JSON(http.StatusOK, models.OrderResponse{
		OrderID:   orderID,
		QRCodeURL: qrCodeURL,
		Amount:    priceCents,
	})
}

func CheckOrderStatus(c *gin.Context) {
	orderID := c.Param("order_id")
	userID := c.GetString("user_id")

	var status string
	var planID string
	err := database.DB.QueryRow(
		`SELECT p.status,
		 (SELECT id FROM subscription_plans WHERE price_cents = p.amount_cents LIMIT 1) as plan_id
		 FROM payments p WHERE p.id = $1 AND p.user_id = $2`,
		orderID, userID,
	).Scan(&status, &planID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// If payment completed and no active subscription, auto-activate
	if status == "completed" && planID != "" {
	var existingSubID string
		err := database.DB.QueryRow(
			`SELECT id FROM user_subscriptions WHERE user_id = $1 AND status = 'active' AND expires_at > NOW() LIMIT 1`,
			userID,
		).Scan(&existingSubID)
		if err != nil {
			// No active subscription - create one and assign account
			activateSubscription(userID, planID)
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

// activateSubscription creates a subscription and assigns an available account
func activateSubscription(userID, planID string) {
	// Get plan details
	var durationDays, maxAccounts int
	database.DB.QueryRow(
		`SELECT duration_days, max_accounts FROM subscription_plans WHERE id = $1`,
		planID,
	).Scan(&durationDays, &maxAccounts)

	// Create subscription
	database.DB.Exec(
		`INSERT INTO user_subscriptions (user_id, plan_id, status, expires_at)
		 VALUES ($1, $2, 'active', NOW() + $3 * INTERVAL '1 day')`,
		userID, planID, durationDays,
	)

	// Auto-assign accounts up to max_accounts
	for i := 0; i < maxAccounts; i++ {
		var accountID string
		err := database.DB.QueryRow(
			`SELECT id FROM account_pool
			 WHERE platform = 'codex' AND status = 'available'
			 ORDER BY created_at LIMIT 1`,
		).Scan(&accountID)
		if err != nil {
			break // No more available accounts
		}
		database.DB.Exec(
			`UPDATE account_pool SET status = 'assigned', updated_at = NOW() WHERE id = $1`,
			accountID,
		)
		database.DB.Exec(
			`INSERT INTO account_assignments (user_id, account_id, status) VALUES ($1, $2, 'active')`,
			userID, accountID,
		)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}
