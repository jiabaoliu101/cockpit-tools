package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jiabaoliu101/aiswitch/backend/database"
	"github.com/jiabaoliu101/aiswitch/backend/models"
)

type featureDefinition struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	Category string `json:"category"`
}

var featureCatalog = []featureDefinition{
	{Key: "feature.local_switch", Name: "本地换号", Category: "核心功能"},
	{Key: "feature.api_relay", Name: "中转站", Category: "核心功能"},
	{Key: "platform.codex", Name: "Codex", Category: "平台"},
	{Key: "platform.codex_api_service", Name: "Codex API 服务", Category: "平台"},
	{Key: "platform.claude_manager", Name: "Claude", Category: "平台"},
	{Key: "platform.antigravity", Name: "Antigravity", Category: "平台"},
	{Key: "platform.antigravity_ide", Name: "Antigravity IDE", Category: "平台"},
	{Key: "platform.zed", Name: "Zed", Category: "平台"},
	{Key: "platform.github-copilot", Name: "GitHub Copilot", Category: "平台"},
	{Key: "platform.windsurf", Name: "Windsurf", Category: "平台"},
	{Key: "platform.kiro", Name: "Kiro", Category: "平台"},
	{Key: "platform.cursor", Name: "Cursor", Category: "平台"},
	{Key: "platform.grok", Name: "Grok", Category: "平台"},
	{Key: "platform.codebuddy", Name: "CodeBuddy", Category: "平台"},
	{Key: "platform.codebuddy_cn", Name: "CodeBuddy CN", Category: "平台"},
	{Key: "platform.qoder", Name: "Qoder", Category: "平台"},
	{Key: "platform.zcode", Name: "Zcode", Category: "平台"},
	{Key: "platform.trae", Name: "Trae", Category: "平台"},
	{Key: "platform.trae_solo", Name: "Trae Solo", Category: "平台"},
	{Key: "platform.trae_cn", Name: "Trae CN", Category: "平台"},
	{Key: "platform.trae_solo_cn", Name: "Trae Solo CN", Category: "平台"},
	{Key: "platform.workbuddy", Name: "WorkBuddy", Category: "平台"},
}

func allFeatures(enabled bool) map[string]bool {
	result := make(map[string]bool, len(featureCatalog))
	for _, feature := range featureCatalog {
		result[feature.Key] = enabled
	}
	return result
}

func decodeFeatures(raw []byte) map[string]bool {
	result := map[string]bool{}
	_ = json.Unmarshal(raw, &result)
	return result
}

func resolveEntitlements(userID string) (models.EffectiveEntitlements, error) {
	result := models.EffectiveEntitlements{Features: allFeatures(false)}

	var isAdmin bool
	if err := database.DB.QueryRow(
		`SELECT is_admin FROM users WHERE id = $1 AND is_active = true`, userID,
	).Scan(&isAdmin); err != nil {
		return result, err
	}
	result.IsAdmin = isAdmin
	if isAdmin {
		result.HasSubscription = true
		result.MaxAccounts = 999
		result.MaxDevices = 999
		result.Features = allFeatures(true)
		return result, nil
	}

	var rawFeatures []byte
	var expiresAt time.Time
	err := database.DB.QueryRow(
		`SELECT sp.id, sp.name, sp.max_accounts, sp.max_devices, sp.entitlements, us.expires_at
		 FROM user_subscriptions us
		 JOIN subscription_plans sp ON sp.id = us.plan_id
		 WHERE us.user_id = $1 AND us.status = 'active' AND us.expires_at > NOW()
		   AND sp.is_active = true
		 ORDER BY us.expires_at DESC LIMIT 1`,
		userID,
	).Scan(&result.PlanID, &result.PlanName, &result.MaxAccounts, &result.MaxDevices, &rawFeatures, &expiresAt)
	if err == sql.ErrNoRows {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.HasSubscription = true
	result.ExpiresAt = &expiresAt
	for key, enabled := range decodeFeatures(rawFeatures) {
		result.Features[key] = enabled
	}

	var overrideRaw []byte
	var maxAccounts, maxDevices sql.NullInt64
	err = database.DB.QueryRow(
		`SELECT entitlements, max_accounts_override, max_devices_override
		 FROM user_entitlement_overrides WHERE user_id = $1`,
		userID,
	).Scan(&overrideRaw, &maxAccounts, &maxDevices)
	if err != nil && err != sql.ErrNoRows {
		return result, err
	}
	if err == nil {
		for key, enabled := range decodeFeatures(overrideRaw) {
			result.Features[key] = enabled
		}
		if maxAccounts.Valid {
			result.MaxAccounts = int(maxAccounts.Int64)
		}
		if maxDevices.Valid {
			result.MaxDevices = int(maxDevices.Int64)
		}
	}
	return result, nil
}

func GetEntitlements(c *gin.Context) {
	result, err := resolveEntitlements(c.GetString("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve entitlements"})
		return
	}
	c.JSON(http.StatusOK, result)
}

var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{8,128}$`)

func CheckinDevice(c *gin.Context) {
	userID := c.GetString("user_id")
	var req struct {
		DeviceID   string `json:"device_id" binding:"required"`
		DeviceName string `json:"device_name" binding:"max=255"`
		Platform   string `json:"platform" binding:"max=50"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !deviceIDPattern.MatchString(req.DeviceID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device"})
		return
	}
	entitlements, err := resolveEntitlements(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve entitlements"})
		return
	}
	if !entitlements.HasSubscription {
		c.JSON(http.StatusForbidden, gin.H{"error": "Active subscription required"})
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}
	defer tx.Rollback()
	var lockedUserID string
	if err := tx.QueryRow(`SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&lockedUserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}

	var exists bool
	_ = tx.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM user_devices WHERE user_id = $1 AND device_id = $2 AND revoked_at IS NULL)`,
		userID, req.DeviceID,
	).Scan(&exists)
	if !exists {
		var count int
		if err := tx.QueryRow(
			`SELECT COUNT(*) FROM user_devices WHERE user_id = $1 AND revoked_at IS NULL`, userID,
		).Scan(&count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check device limit"})
			return
		}
		if count >= entitlements.MaxDevices {
			c.JSON(http.StatusConflict, gin.H{"error": "Device limit reached", "code": "device_limit_reached"})
			return
		}
	}

	_, err = tx.Exec(
		`INSERT INTO user_devices (user_id, device_id, device_name, platform)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, device_id) DO UPDATE
		 SET device_name = EXCLUDED.device_name, platform = EXCLUDED.platform,
		     last_seen_at = NOW(), revoked_at = NULL`,
		userID, req.DeviceID, strings.TrimSpace(req.DeviceName), strings.TrimSpace(req.Platform),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"registered": true, "entitlements": entitlements})
}

func AdminFeatureCatalog(c *gin.Context) {
	c.JSON(http.StatusOK, featureCatalog)
}

type adminPlanRequest struct {
	Name         string          `json:"name" binding:"required,max=100"`
	DurationDays int             `json:"duration_days" binding:"required,min=1"`
	MaxAccounts  int             `json:"max_accounts" binding:"min=0"`
	MaxDevices   int             `json:"max_devices" binding:"required,min=1,max=999"`
	PriceCents   int             `json:"price_cents" binding:"min=0"`
	IsActive     bool            `json:"is_active"`
	Entitlements map[string]bool `json:"entitlements"`
}

func listPlans(includeInactive bool) ([]models.SubscriptionPlan, error) {
	query := `SELECT id, name, duration_days, max_accounts, max_devices, entitlements, price_cents, is_active
		FROM subscription_plans`
	if !includeInactive {
		query += ` WHERE is_active = true`
	}
	query += ` ORDER BY price_cents, created_at`
	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.SubscriptionPlan{}
	for rows.Next() {
		var plan models.SubscriptionPlan
		var raw []byte
		if err := rows.Scan(&plan.ID, &plan.Name, &plan.DurationDays, &plan.MaxAccounts, &plan.MaxDevices, &raw, &plan.PriceCents, &plan.IsActive); err != nil {
			return nil, err
		}
		plan.Entitlements = decodeFeatures(raw)
		result = append(result, plan)
	}
	return result, rows.Err()
}

func AdminListPlans(c *gin.Context) {
	plans, err := listPlans(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list plans"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

func AdminCreatePlan(c *gin.Context) {
	var req adminPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	raw, _ := json.Marshal(req.Entitlements)
	var id string
	err := database.DB.QueryRow(
		`INSERT INTO subscription_plans
		 (name, duration_days, max_accounts, max_devices, entitlements, entitlements_initialized, price_cents, is_active)
		 VALUES ($1, $2, $3, $4, $5, TRUE, $6, $7) RETURNING id`,
		strings.TrimSpace(req.Name), req.DurationDays, req.MaxAccounts, req.MaxDevices, raw, req.PriceCents, req.IsActive,
	).Scan(&id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to create plan"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func AdminUpdatePlan(c *gin.Context) {
	var req adminPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	raw, _ := json.Marshal(req.Entitlements)
	result, err := database.DB.Exec(
		`UPDATE subscription_plans
		 SET name = $1, duration_days = $2, max_accounts = $3, max_devices = $4,
		     entitlements = $5, entitlements_initialized = TRUE, price_cents = $6, is_active = $7
		 WHERE id = $8`,
		strings.TrimSpace(req.Name), req.DurationDays, req.MaxAccounts, req.MaxDevices, raw, req.PriceCents, req.IsActive, c.Param("id"),
	)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Failed to update plan"})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Plan updated"})
}

func AdminGetUserEntitlements(c *gin.Context) {
	userID := c.Param("id")
	effective, err := resolveEntitlements(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	override := map[string]bool{}
	var raw []byte
	var maxAccounts, maxDevices sql.NullInt64
	if err := database.DB.QueryRow(
		`SELECT entitlements, max_accounts_override, max_devices_override
		 FROM user_entitlement_overrides WHERE user_id = $1`, userID,
	).Scan(&raw, &maxAccounts, &maxDevices); err == nil {
		override = decodeFeatures(raw)
	}
	c.JSON(http.StatusOK, gin.H{
		"effective": effective,
		"override": gin.H{
			"entitlements": override,
			"max_accounts": nullableInt(maxAccounts),
			"max_devices":  nullableInt(maxDevices),
		},
	})
}

func nullableInt(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}

func AdminUpdateUserEntitlements(c *gin.Context) {
	var req struct {
		Entitlements map[string]bool `json:"entitlements"`
		MaxAccounts  *int            `json:"max_accounts"`
		MaxDevices   *int            `json:"max_devices"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.MaxAccounts != nil && *req.MaxAccounts < 0 || req.MaxDevices != nil && (*req.MaxDevices < 1 || *req.MaxDevices > 999) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}
	raw, _ := json.Marshal(req.Entitlements)
	_, err := database.DB.Exec(
		`INSERT INTO user_entitlement_overrides
		 (user_id, entitlements, max_accounts_override, max_devices_override, updated_by)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (user_id) DO UPDATE
		 SET entitlements = EXCLUDED.entitlements,
		     max_accounts_override = EXCLUDED.max_accounts_override,
		     max_devices_override = EXCLUDED.max_devices_override,
		     updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		c.Param("id"), raw, req.MaxAccounts, req.MaxDevices, c.GetString("user_id"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user entitlements"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User entitlements updated"})
}

func AdminListUserDevices(c *gin.Context) {
	rows, err := database.DB.Query(
		`SELECT id, device_id, device_name, platform, first_seen_at, last_seen_at, revoked_at
		 FROM user_devices WHERE user_id = $1 ORDER BY last_seen_at DESC`, c.Param("id"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list devices"})
		return
	}
	defer rows.Close()
	result := []gin.H{}
	for rows.Next() {
		var id, deviceID, deviceName, platform string
		var firstSeen, lastSeen time.Time
		var revokedAt sql.NullTime
		if err := rows.Scan(&id, &deviceID, &deviceName, &platform, &firstSeen, &lastSeen, &revokedAt); err != nil {
			continue
		}
		result = append(result, gin.H{
			"id": id, "device_id": deviceID, "device_name": deviceName, "platform": platform,
			"first_seen_at": firstSeen, "last_seen_at": lastSeen, "revoked_at": nullableTime(revokedAt),
		})
	}
	c.JSON(http.StatusOK, result)
}

func nullableTime(value sql.NullTime) any {
	if !value.Valid {
		return nil
	}
	return value.Time
}

func AdminRevokeUserDevice(c *gin.Context) {
	result, err := database.DB.Exec(
		`UPDATE user_devices SET revoked_at = NOW()
		 WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`,
		c.Param("device_id"), c.Param("id"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke device"})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Active device not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Device revoked"})
}

func featureForPlatform(platform string) string {
	normalized := strings.ToLower(strings.TrimSpace(platform))
	if normalized == "claude" {
		normalized = "claude_manager"
	}
	if normalized == "github_copilot" {
		normalized = "github-copilot"
	}
	return "platform." + normalized
}
