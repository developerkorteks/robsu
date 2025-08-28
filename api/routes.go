package api

import (
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all API routes
func SetupRoutes(router *gin.Engine) {
	// API group
	api := router.Group("/api")

	// Admin approval endpoints
	admin := api.Group("/admin")
	{
		// Get pending top up transactions
		admin.GET("/topups/pending", GetPendingTopUps)

		// Get all transactions with filters
		admin.GET("/transactions", GetAllTransactions)

		// Get specific transaction detail
		admin.GET("/transactions/:id", GetTransactionDetail)

		// Approve or reject single transaction
		admin.POST("/topups/approve", ProcessTopUpApproval)

		// Bulk approve multiple transactions
		admin.POST("/topups/bulk-approve", BulkApproveTransactions)
	}

	// Public endpoints for external integration
	public := api.Group("/public")
	{
		// Create topup transaction (for external systems)
		public.POST("/topups/create", CreateTopUpTransaction)

		// Get user balance
		public.GET("/users/:user_id/balance", GetUserBalance)
	}

	// Health check endpoint
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "GRN Store API is running",
		})
	})
}
