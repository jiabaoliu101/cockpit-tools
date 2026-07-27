package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/jiabaoliu101/aiswitch/backend/database"
	"github.com/jiabaoliu101/aiswitch/backend/models"
)

// ── Admin: Users ──

func AdminListUsers(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, email, nickname, is_active, is_admin, created_at, last_login_at
		 FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Nickname, &u.IsActive, &u.IsAdmin, &u.CreatedAt, &u.LastLoginAt); err != nil {
			continue
		}
		users = append(users, u)
	}
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, users)
}

func AdminUpdateUser(c *gin.Context) {
	userID := c.Param("id")
	var req struct {
		IsActive *bool   `json:"is_active"`
		IsAdmin  *bool   `json:"is_admin"`
		Nickname *string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IsActive != nil {
		if _, err := database.DB.Exec(`UPDATE users SET is_active = $1 WHERE id = $2`, *req.IsActive, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
			return
		}
		if !*req.IsActive {
			// Existing access tokens are rejected by AuthRequired; revoke refresh tokens too.
			if _, err := database.DB.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke user sessions"})
				return
			}
		}
	}
	if req.IsAdmin != nil {
		database.DB.Exec(`UPDATE users SET is_admin = $1 WHERE id = $2`, *req.IsAdmin, userID)
	}
	if req.Nickname != nil {
		database.DB.Exec(`UPDATE users SET nickname = $1 WHERE id = $2`, *req.Nickname, userID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

// ── Admin: Account Pool ──

type poolEntry struct {
	ID          string `json:"id"`
	Platform    string `json:"platform"`
	Email       string `json:"email"`
	TokenExpiry int64  `json:"token_expiry"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func AdminListAccounts(c *gin.Context) {
	status := c.DefaultQuery("status", "")
	query := `SELECT id, platform, email, token_expiry, status,
		to_char(created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at,
		to_char(updated_at, 'YYYY-MM-DD HH24:MI:SS') as updated_at
		FROM account_pool`
	var args []any

	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at`

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var result []poolEntry
	for rows.Next() {
		var e poolEntry
		if err := rows.Scan(&e.ID, &e.Platform, &e.Email, &e.TokenExpiry, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			continue
		}
		result = append(result, e)
	}
	if result == nil {
		result = []poolEntry{}
	}
	c.JSON(http.StatusOK, result)
}

func AdminAddAccount(c *gin.Context) {
	var req struct {
		Platform     string `json:"platform" binding:"required"`
		Email        string `json:"email"`
		AccessToken  string `json:"access_token" binding:"required"`
		RefreshToken string `json:"refresh_token"`
		TokenExpiry  int64  `json:"token_expiry"`
		Metadata     string `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metadataJSON := req.Metadata
	if metadataJSON == "" {
		metadataJSON = "{}"
	}

	id := uuid.New().String()
	_, err := database.DB.Exec(
		`INSERT INTO account_pool (id, platform, email, access_token_encrypted, refresh_token_encrypted, token_expiry, metadata, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'staging')`,
		id, req.Platform, req.Email, req.AccessToken, req.RefreshToken, req.TokenExpiry, metadataJSON,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add account: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Account added to pool"})
}

func AdminBulkAddAccounts(c *gin.Context) {
	var req []struct {
		Platform     string `json:"platform"`
		Email        string `json:"email"`
		AccessToken  string `json:"access_token" binding:"required"`
		RefreshToken string `json:"refresh_token"`
		TokenExpiry  int64  `json:"token_expiry"`
		Metadata     string `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	added := 0
	for _, acc := range req {
		metadataJSON := acc.Metadata
		if metadataJSON == "" {
			metadataJSON = "{}"
		}
		id := uuid.New().String()
		_, err := database.DB.Exec(
			`INSERT INTO account_pool (id, platform, email, access_token_encrypted, refresh_token_encrypted, token_expiry, metadata, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, 'staging')`,
			id, acc.Platform, acc.Email, acc.AccessToken, acc.RefreshToken, acc.TokenExpiry, metadataJSON,
		)
		if err == nil {
			added++
		}
	}

	c.JSON(http.StatusCreated, gin.H{"added": added, "total": len(req)})
}

func AdminUpdateAccount(c *gin.Context) {
	accountID := c.Param("id")
	var req struct {
		AccessToken  *string `json:"access_token"`
		RefreshToken *string `json:"refresh_token"`
		TokenExpiry  *int64  `json:"token_expiry"`
		Status       *string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.AccessToken != nil {
		database.DB.Exec(`UPDATE account_pool SET access_token_encrypted = $1, updated_at = NOW() WHERE id = $2`, *req.AccessToken, accountID)
	}
	if req.RefreshToken != nil {
		database.DB.Exec(`UPDATE account_pool SET refresh_token_encrypted = $1, updated_at = NOW() WHERE id = $2`, *req.RefreshToken, accountID)
	}
	if req.TokenExpiry != nil {
		database.DB.Exec(`UPDATE account_pool SET token_expiry = $1, updated_at = NOW() WHERE id = $2`, *req.TokenExpiry, accountID)
	}
	if req.Status != nil {
		database.DB.Exec(`UPDATE account_pool SET status = $1, updated_at = NOW() WHERE id = $2`, *req.Status, accountID)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account updated"})
}

func AdminDeleteAccount(c *gin.Context) {
	accountID := c.Param("id")
	database.DB.Exec(`DELETE FROM account_pool WHERE id = $1`, accountID)
	c.JSON(http.StatusOK, gin.H{"message": "Account removed from pool"})
}

// ── Admin: Assignments ──

type assignmentEntry struct {
	ID         string `json:"id"`
	UserEmail  string `json:"user_email"`
	AccountEmail string `json:"account_email"`
	Status     string `json:"status"`
	AssignedAt string `json:"assigned_at"`
}

func AdminListAssignments(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT aa.id, u.email, ap.email, aa.status,
			to_char(aa.assigned_at, 'YYYY-MM-DD HH24:MI:SS') as assigned_at
		 FROM account_assignments aa
		 JOIN users u ON aa.user_id = u.id
		 JOIN account_pool ap ON aa.account_id = ap.id
		 ORDER BY aa.assigned_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var result []assignmentEntry
	for rows.Next() {
		var e assignmentEntry
		if err := rows.Scan(&e.ID, &e.UserEmail, &e.AccountEmail, &e.Status, &e.AssignedAt); err != nil {
			continue
		}
		result = append(result, e)
	}
	if result == nil {
		result = []assignmentEntry{}
	}
	c.JSON(http.StatusOK, result)
}

func AdminAssignAccount(c *gin.Context) {
	var req struct {
		UserID    string `json:"user_id" binding:"required"`
		AccountID string `json:"account_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check account is available
	var status string
	err := database.DB.QueryRow(
		`SELECT status FROM account_pool WHERE id = $1`, req.AccountID,
	).Scan(&status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}
	if status != "ready" {
		c.JSON(http.StatusConflict, gin.H{"error": "Account is not ready for assignment"})
		return
	}

	// Assign
	database.DB.Exec(`UPDATE account_pool SET status = 'assigned', updated_at = NOW() WHERE id = $1`, req.AccountID)
	database.DB.Exec(
		`INSERT INTO account_assignments (user_id, account_id, status) VALUES ($1, $2, 'active')`,
		req.UserID, req.AccountID,
	)

	c.JSON(http.StatusCreated, gin.H{"message": "Account assigned"})
}

func AdminRevokeAssignment(c *gin.Context) {
	assignmentID := c.Param("id")

	// A local credential may still be present on the subscriber's device. Do not
	// put it straight back into stock; it must be revoked and verified first.
	var accountID string
	database.DB.QueryRow(
		`SELECT account_id FROM account_assignments WHERE id = $1`, assignmentID,
	).Scan(&accountID)

	if accountID != "" {
		database.DB.Exec(`UPDATE account_pool SET status = 'revoking', updated_at = NOW() WHERE id = $1`, accountID)
	}
	database.DB.Exec(
		`UPDATE account_assignments SET status = 'revoked', revoked_at = NOW() WHERE id = $1`,
		assignmentID,
	)

	c.JSON(http.StatusOK, gin.H{"message": "Assignment revoked"})
}

// ── Admin: Dashboard ──

func AdminDashboard(c *gin.Context) {
	var stats models.DashboardStats

	database.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&stats.TotalUsers)
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM user_subscriptions WHERE status = 'active' AND expires_at > NOW()`,
	).Scan(&stats.ActiveSubscriptions)
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM account_pool WHERE status = 'ready'`,
	).Scan(&stats.AvailableAccounts)
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM account_pool WHERE status = 'assigned'`,
	).Scan(&stats.AssignedAccounts)
	database.DB.QueryRow(
		`SELECT COALESCE(SUM(amount_cents), 0) FROM payments WHERE status = 'completed'
		 AND paid_at > NOW() - INTERVAL '30 days'`,
	).Scan(&stats.MonthlyRevenue)

	c.JSON(http.StatusOK, stats)
}

// ── Admin: Payments ──

type paymentEntry struct {
	ID          string  `json:"id"`
	UserEmail   string  `json:"user_email"`
	AmountCents int     `json:"amount_cents"`
	Status      string  `json:"status"`
	PaidAt      *time.Time `json:"paid_at"`
	CreatedAt   string  `json:"created_at"`
}

func AdminListPayments(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT p.id, u.email, p.amount_cents, p.status, p.paid_at,
			to_char(p.created_at, 'YYYY-MM-DD HH24:MI:SS') as created_at
		 FROM payments p
		 JOIN users u ON p.user_id = u.id
		 ORDER BY p.created_at DESC LIMIT 200`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var result []paymentEntry
	for rows.Next() {
		var e paymentEntry
		if err := rows.Scan(&e.ID, &e.UserEmail, &e.AmountCents, &e.Status, &e.PaidAt, &e.CreatedAt); err != nil {
			continue
		}
		result = append(result, e)
	}
	if result == nil {
		result = []paymentEntry{}
	}
	c.JSON(http.StatusOK, result)
}
