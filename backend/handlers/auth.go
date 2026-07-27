package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/jiabaoliu101/aiswitch/backend/database"
	"github.com/jiabaoliu101/aiswitch/backend/middleware"
	"github.com/jiabaoliu101/aiswitch/backend/models"
)

// ── Helpers ──

func generateTokens(userID string, isAdmin bool) (string, string, error) {
	// Access token (15 min)
	accessClaims := jwt.MapClaims{
		"sub":     userID,
		"is_admin": isAdmin,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"iat":     time.Now().Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(middleware.JWTSecret)
	if err != nil {
		return "", "", err
	}

	// Refresh token (7 days)
	refreshToken := uuid.New().String()
	refreshHash, err := bcrypt.GenerateFromPassword([]byte(refreshToken), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	_, err = database.DB.Exec(
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, string(refreshHash), time.Now().Add(7*24*time.Hour),
	)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// ── Auth Handlers ──

func Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var user models.User
	err = database.DB.QueryRow(
		`INSERT INTO users (email, password_hash, nickname) VALUES ($1, $2, $3)
		 RETURNING id, email, nickname, is_active, is_admin, created_at, last_login_at`,
		req.Email, string(hashedPassword), req.Nickname,
	).Scan(&user.ID, &user.Email, &user.Nickname, &user.IsActive, &user.IsAdmin, &user.CreatedAt, &user.LastLoginAt)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	accessToken, refreshToken, err := generateTokens(user.ID, user.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusCreated, models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := database.DB.QueryRow(
		`SELECT id, email, password_hash, nickname, is_active, is_admin, created_at, last_login_at
		 FROM users WHERE email = $1`,
		req.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Nickname, &user.IsActive, &user.IsAdmin, &user.CreatedAt, &user.LastLoginAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is disabled"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Update last login
	database.DB.Exec(`UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID)

	accessToken, refreshToken, err := generateTokens(user.ID, user.IsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
	})
}

func RefreshToken(c *gin.Context) {
	var req models.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find valid refresh tokens for this raw token value
	// (We iterate and compare because bcrypt hashes are unique each time)
	rows, err := database.DB.Query(
		`SELECT rt.id, rt.user_id, rt.token_hash, u.is_admin
		 FROM refresh_tokens rt JOIN users u ON rt.user_id = u.id
		 WHERE rt.expires_at > NOW()`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var foundUserID string
	var foundIsAdmin bool
	var foundTokenID string

	for rows.Next() {
		var tokenID, userID, tokenHash string
		var isAdmin bool
		if err := rows.Scan(&tokenID, &userID, &tokenHash, &isAdmin); err != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(tokenHash), []byte(req.RefreshToken)) == nil {
			foundUserID = userID
			foundIsAdmin = isAdmin
			foundTokenID = tokenID
			break
		}
	}

	if foundUserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Delete the used refresh token
	database.DB.Exec(`DELETE FROM refresh_tokens WHERE id = $1`, foundTokenID)

	accessToken, refreshToken, err := generateTokens(foundUserID, foundIsAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func Logout(c *gin.Context) {
	userID := c.GetString("user_id")
	database.DB.Exec(`DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// ── User Handlers ──

func GetMe(c *gin.Context) {
	userID := c.GetString("user_id")
	var user models.User
	err := database.DB.QueryRow(
		`SELECT id, email, nickname, is_active, is_admin, created_at, last_login_at
		 FROM users WHERE id = $1`, userID,
	).Scan(&user.ID, &user.Email, &user.Nickname, &user.IsActive, &user.IsAdmin, &user.CreatedAt, &user.LastLoginAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}
