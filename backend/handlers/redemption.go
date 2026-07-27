package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/jiabaoliu101/aiswitch/backend/database"
	"github.com/jiabaoliu101/aiswitch/backend/models"
)

var cardCodePepper []byte

func SetCardCodePepper(pepper string) { cardCodePepper = []byte(pepper) }

func normalizeCardCode(code string) string {
	return strings.ToUpper(strings.NewReplacer("-", "", " ", "", "\t", "", "\n", "").Replace(code))
}

func cardCodeHash(code string) string {
	h := hmac.New(sha256.New, cardCodePepper)
	_, _ = h.Write([]byte(normalizeCardCode(code)))
	return hex.EncodeToString(h.Sum(nil))
}

func newCardCode() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	chars := make([]byte, 20)
	for i := range chars {
		n, err := rand.Int(rand.Reader, bigInt(len(alphabet)))
		if err != nil {
			return "", err
		}
		chars[i] = alphabet[n.Int64()]
	}
	return fmt.Sprintf("AS-%s-%s-%s-%s", chars[:5], chars[5:10], chars[10:15], chars[15:20]), nil
}

func bigInt(n int) *big.Int { return big.NewInt(int64(n)) }

type redemptionCodeEntry struct {
	ID         string     `json:"id"`
	PlanName   string     `json:"plan_name"`
	Status     string     `json:"status"`
	ExpiresAt  *time.Time `json:"expires_at"`
	RedeemedAt *time.Time `json:"redeemed_at"`
	RedeemedBy string     `json:"redeemed_by"`
	CreatedAt  time.Time  `json:"created_at"`
	BatchName  string     `json:"batch_name"`
	Note       string     `json:"note"`
}

func AdminGenerateRedemptionCodes(c *gin.Context) {
	var req models.GenerateRedemptionCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Expiry must be in the future"})
		return
	}
	var exists bool
	if err := database.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM subscription_plans WHERE id = $1 AND is_active = true)`, req.PlanID).Scan(&exists); err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found or inactive"})
		return
	}
	codes := make([]string, 0, req.Count)
	for len(codes) < req.Count {
		code, err := newCardCode()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate card code"})
			return
		}
		_, err = database.DB.Exec(`INSERT INTO redemption_codes (code_hash, plan_id, expires_at, created_by, batch_name, note) VALUES ($1, $2, $3, $4, $5, $6)`, cardCodeHash(code), req.PlanID, req.ExpiresAt, c.GetString("user_id"), strings.TrimSpace(req.BatchName), strings.TrimSpace(req.Note))
		if err != nil {
			continue
		}
		codes = append(codes, code)
	}
	c.JSON(http.StatusCreated, gin.H{"codes": codes, "message": "Card codes are shown only once; export them now."})
}

func AdminListRedemptionCodes(c *gin.Context) {
	rows, err := database.DB.Query(`SELECT rc.id, sp.name, rc.status, rc.expires_at, rc.redeemed_at, COALESCE(u.email, ''), rc.created_at, rc.batch_name, rc.note FROM redemption_codes rc JOIN subscription_plans sp ON sp.id = rc.plan_id LEFT JOIN users u ON u.id = rc.redeemed_by ORDER BY rc.created_at DESC LIMIT 500`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()
	result := []redemptionCodeEntry{}
	for rows.Next() {
		var item redemptionCodeEntry
		if err := rows.Scan(&item.ID, &item.PlanName, &item.Status, &item.ExpiresAt, &item.RedeemedAt, &item.RedeemedBy, &item.CreatedAt, &item.BatchName, &item.Note); err == nil {
			result = append(result, item)
		}
	}
	c.JSON(http.StatusOK, result)
}

func AdminDisableRedemptionCode(c *gin.Context) {
	result, err := database.DB.Exec(`UPDATE redemption_codes SET status = 'disabled' WHERE id = $1 AND status = 'active'`, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Card code cannot be disabled"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Card code disabled"})
}

func RedeemCode(c *gin.Context) {
	var req models.RedeemCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	var id, planID, status string
	var expiresAt sql.NullTime
	err = tx.QueryRow(`SELECT id, plan_id, status, expires_at FROM redemption_codes WHERE code_hash = $1 FOR UPDATE`, cardCodeHash(req.Code)).Scan(&id, &planID, &status, &expiresAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid card code"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if status != "active" {
		c.JSON(http.StatusConflict, gin.H{"error": "Card code has already been used or disabled"})
		return
	}
	if expiresAt.Valid && !expiresAt.Time.After(time.Now()) {
		c.JSON(http.StatusConflict, gin.H{"error": "Card code has expired"})
		return
	}
	if err := grantSubscriptionTx(tx, c.GetString("user_id"), planID); err != nil {
		if errors.Is(err, ErrInsufficientInventory) {
			c.JSON(http.StatusConflict, gin.H{"error": "This plan is temporarily out of stock. Please contact support."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate subscription"})
		return
	}
	if _, err := tx.Exec(`UPDATE redemption_codes SET status = 'redeemed', redeemed_by = $1, redeemed_at = NOW() WHERE id = $2`, c.GetString("user_id"), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to redeem card code"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to redeem card code"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Card code redeemed"})
}
