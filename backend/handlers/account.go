package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jiabaoliu101/aiswitch/backend/database"
	"github.com/jiabaoliu101/aiswitch/backend/models"
)

// ── User Account Handlers ──

// GetMyAccounts returns the assigned Codex accounts for the current user
func GetMyAccounts(c *gin.Context) {
	userID := c.GetString("user_id")

	rows, err := database.DB.Query(
		`SELECT ap.id, ap.platform, ap.email, ap.access_token_encrypted, ap.token_expiry
		 FROM account_pool ap
		 JOIN account_assignments aa ON ap.id = aa.account_id
		 WHERE aa.user_id = $1 AND aa.status = 'active' AND ap.status = 'assigned'`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var accounts []models.UserAccountResponse
	for rows.Next() {
		var a models.UserAccountResponse
		if err := rows.Scan(&a.ID, &a.Platform, &a.Email, &a.AccessToken, &a.TokenExpiry); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}

	c.JSON(http.StatusOK, accounts)
}

// GetMyAccount returns a specific account detail
func GetMyAccount(c *gin.Context) {
	userID := c.GetString("user_id")
	accountID := c.Param("id")

	var a models.UserAccountResponse
	err := database.DB.QueryRow(
		`SELECT ap.id, ap.platform, ap.email, ap.access_token_encrypted, ap.token_expiry
		 FROM account_pool ap
		 JOIN account_assignments aa ON ap.id = aa.account_id
		 WHERE ap.id = $1 AND aa.user_id = $2 AND aa.status = 'active'`,
		accountID, userID,
	).Scan(&a.ID, &a.Platform, &a.Email, &a.AccessToken, &a.TokenExpiry)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}
	c.JSON(http.StatusOK, a)
}

// ── Client Checkin ──

// Checkin validates the subscription and returns fresh account data
func Checkin(c *gin.Context) {
	userID := c.GetString("user_id")

	// Check subscription status
	var hasActiveSub bool
	var expiresAt *time.Time
	database.DB.QueryRow(
		`SELECT EXISTS(
			SELECT 1 FROM user_subscriptions WHERE user_id = $1 AND status = 'active' AND expires_at > NOW()
		)`, userID,
	).Scan(&hasActiveSub)
	if hasActiveSub {
		database.DB.QueryRow(
			`SELECT expires_at FROM user_subscriptions WHERE user_id = $1 AND status = 'active' ORDER BY expires_at DESC LIMIT 1`,
			userID,
		).Scan(&expiresAt)
	}

	// Fetch assigned accounts
	rows, err := database.DB.Query(
		`SELECT ap.id, ap.platform, ap.email, ap.access_token_encrypted, ap.token_expiry
		 FROM account_pool ap
		 JOIN account_assignments aa ON ap.id = aa.account_id
		 WHERE aa.user_id = $1 AND aa.status = 'active'`,
		userID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var accounts []models.UserAccountResponse
	for rows.Next() {
		var a models.UserAccountResponse
		rows.Scan(&a.ID, &a.Platform, &a.Email, &a.AccessToken, &a.TokenExpiry)
		accounts = append(accounts, a)
	}

	if accounts == nil {
		accounts = []models.UserAccountResponse{}
	}

	c.JSON(http.StatusOK, gin.H{
		"subscription_active": hasActiveSub,
		"subscription_expires_at": expiresAt,
		"accounts": accounts,
	})
}
