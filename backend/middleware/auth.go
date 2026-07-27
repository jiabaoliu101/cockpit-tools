package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/jiabaoliu101/aiswitch/backend/database"
)

var JWTSecret []byte

func SetJWTSecret(secret string) {
	JWTSecret = []byte(secret)
}

// AuthRequired validates the JWT access token and injects user claims
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		userID, ok := claims["sub"].(string)
		if !ok || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token subject"})
			c.Abort()
			return
		}

		var isActive bool
		if err := database.DB.QueryRow(`SELECT is_active FROM users WHERE id = $1`, userID).Scan(&isActive); err != nil || !isActive {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled or no longer exists"})
			c.Abort()
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

// AdminRequired ensures the user is an admin
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		var isAdmin bool
		if err := database.DB.QueryRow(`SELECT is_admin FROM users WHERE id = $1 AND is_active = true`, userID).Scan(&isAdmin); err != nil || !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
