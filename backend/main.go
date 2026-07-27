package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/jiabaoliu101/aiswitch/backend/config"
	"github.com/jiabaoliu101/aiswitch/backend/database"
	"github.com/jiabaoliu101/aiswitch/backend/handlers"
	"github.com/jiabaoliu101/aiswitch/backend/middleware"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize JWT secret
	middleware.SetJWTSecret(cfg.JWTSecret)
	handlers.SetCardCodePepper(cfg.CardCodePepper)

	// Connect to database
	if err := database.Connect(cfg.DatabaseURL); err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer database.Close()
	if err := database.Migrate(); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}
	handlers.StartLeaseExpirySweep()

	// Setup router
	r := gin.Default()
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "aiswitch-api"})
	})

	api := r.Group("/api/v1")
	{
		// ── Public routes ──
		auth := api.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
			auth.POST("/refresh", handlers.RefreshToken)
		}

		// ── Protected routes ──
		protected := api.Group("")
		protected.Use(middleware.AuthRequired())
		{
			protected.POST("/auth/logout", handlers.Logout)

			// User
			protected.GET("/user/me", handlers.GetMe)

			// Subscriptions
			protected.GET("/subscriptions/plans", handlers.GetSubscriptionPlans)
			protected.GET("/subscriptions/current", handlers.GetCurrentSubscription)
			protected.POST("/subscriptions/order", handlers.CreateOrder)
			protected.GET("/subscriptions/status/:order_id", handlers.CheckOrderStatus)
			protected.POST("/redemption-codes/redeem", handlers.RedeemCode)

			// My accounts
			protected.GET("/my-accounts", handlers.GetMyAccounts)
			protected.GET("/my-accounts/:id", handlers.GetMyAccount)

			// Client checkin
			protected.POST("/client/checkin", handlers.Checkin)

			// Admin routes
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminRequired())
			{
				admin.GET("/users", handlers.AdminListUsers)
				admin.PUT("/users/:id", handlers.AdminUpdateUser)

				admin.GET("/accounts", handlers.AdminListAccounts)
				admin.POST("/accounts", handlers.AdminAddAccount)
				admin.POST("/accounts/bulk", handlers.AdminBulkAddAccounts)
				admin.PUT("/accounts/:id", handlers.AdminUpdateAccount)
				admin.DELETE("/accounts/:id", handlers.AdminDeleteAccount)

				admin.GET("/assignments", handlers.AdminListAssignments)
				admin.POST("/assignments", handlers.AdminAssignAccount)
				admin.DELETE("/assignments/:id", handlers.AdminRevokeAssignment)

				admin.GET("/payments", handlers.AdminListPayments)
				admin.GET("/redemption-codes", handlers.AdminListRedemptionCodes)
				admin.POST("/redemption-codes", handlers.AdminGenerateRedemptionCodes)
				admin.DELETE("/redemption-codes/:id", handlers.AdminDisableRedemptionCode)
				admin.GET("/dashboard", handlers.AdminDashboard)
			}
		}
	}

	log.Printf("AISwitch API server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
