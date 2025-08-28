package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nabilulilalbab/bottele/dto"
	"github.com/nabilulilalbab/bottele/service"
)

// TopUp Approval Request
type TopUpApprovalRequest struct {
	TransactionID string `json:"transaction_id" binding:"required"`
	Status        string `json:"status" binding:"required"` // "approved" or "rejected"
	AdminNote     string `json:"admin_note"`
}

// Get pending top up transactions
func GetPendingTopUps(c *gin.Context) {
	// Get pending transactions from the same in-memory storage used by bot
	pendingTransactions := service.GetPendingTransactions()

	// Convert to API response format
	var apiTransactions []gin.H
	for _, tx := range pendingTransactions {
		apiTransactions = append(apiTransactions, gin.H{
			"id":         tx.ID,
			"user_id":    tx.UserID,
			"username":   tx.Username,
			"amount":     tx.Amount,
			"status":     tx.Status,
			"qris_code":  tx.QRISCode,
			"created_at": tx.CreatedAt,
			"expired_at": tx.ExpiredAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    apiTransactions,
		"count":   len(apiTransactions),
	})
}

// Get specific transaction details
func GetTransactionDetail(c *gin.Context) {
	transactionID := c.Param("id")

	// Get transaction from in-memory storage
	service.TxMutex.RLock()
	transaction, exists := service.Transactions[transactionID]
	service.TxMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Transaction not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":          transaction.ID,
			"user_id":     transaction.UserID,
			"username":    transaction.Username,
			"amount":      transaction.Amount,
			"status":      transaction.Status,
			"qris_code":   transaction.QRISCode,
			"created_at":  transaction.CreatedAt,
			"expired_at":  transaction.ExpiredAt,
			"approved_by": transaction.ApprovedBy,
			"approved_at": transaction.ApprovedAt,
		},
	})
}

// Approve or reject top up transaction
func ProcessTopUpApproval(c *gin.Context) {
	var req TopUpApprovalRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Validate status
	if req.Status != "approved" && req.Status != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Status must be 'approved' or 'rejected'",
		})
		return
	}

	// Use the existing service functions that bot uses
	var err error
	if req.Status == "approved" {
		// Use the same ConfirmTopUp function that bot uses
		// We'll use admin ID 0 to indicate API approval
		err = service.ConfirmTopUp(req.TransactionID, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to approve transaction: " + err.Error(),
			})
			return
		}

		// Send additional notification with admin note if provided
		if req.AdminNote != "" {
			service.TxMutex.RLock()
			transaction, exists := service.Transactions[req.TransactionID]
			service.TxMutex.RUnlock()

			if exists {
				noteMessage := fmt.Sprintf("📝 Catatan Admin: %s", req.AdminNote)
				service.NotifyAdminError(transaction.UserID, "Admin Note", noteMessage)
			}
		}

	} else if req.Status == "rejected" {
		// Use the same RejectTopUp function that bot uses
		err = service.RejectTopUp(req.TransactionID, 0)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to reject transaction: " + err.Error(),
			})
			return
		}

		// Send rejection notification with admin note
		service.TxMutex.RLock()
		transaction, exists := service.Transactions[req.TransactionID]
		service.TxMutex.RUnlock()

		if exists {
			rejectionMessage := fmt.Sprintf("❌ Top up Anda sebesar Rp %s ditolak.", formatAmount(transaction.Amount))
			if req.AdminNote != "" {
				rejectionMessage += fmt.Sprintf(" Alasan: %s", req.AdminNote)
			}
			service.NotifyAdminError(transaction.UserID, "Top Up Rejected", rejectionMessage)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Transaction " + req.Status + " successfully",
		"data": gin.H{
			"transaction_id": req.TransactionID,
			"status":         req.Status,
			"admin_note":     req.AdminNote,
			"processed_at":   time.Now(),
		},
	})
}

// Get all transactions with filters
func GetAllTransactions(c *gin.Context) {
	status := c.Query("status")     // pending, completed, rejected
	userIDStr := c.Query("user_id") // filter by user
	limit := parseIntDefault(c.DefaultQuery("limit", "50"), 50)
	offset := parseIntDefault(c.DefaultQuery("offset", "0"), 0)

	// Get all transactions from in-memory storage
	service.TxMutex.RLock()
	allTransactions := make([]*dto.Transaction, 0, len(service.Transactions))
	for _, tx := range service.Transactions {
		allTransactions = append(allTransactions, tx)
	}
	service.TxMutex.RUnlock()

	// Apply filters
	var filteredTransactions []*dto.Transaction
	for _, tx := range allTransactions {
		// Filter by status
		if status != "" && tx.Status != status {
			continue
		}

		// Filter by user ID
		if userIDStr != "" {
			userID, err := strconv.ParseInt(userIDStr, 10, 64)
			if err != nil || tx.UserID != userID {
				continue
			}
		}

		filteredTransactions = append(filteredTransactions, tx)
	}

	// Sort by created_at DESC (newest first)
	// Simple sort implementation
	for i := 0; i < len(filteredTransactions)-1; i++ {
		for j := i + 1; j < len(filteredTransactions); j++ {
			time1, _ := time.Parse("2006-01-02 15:04:05", filteredTransactions[i].CreatedAt)
			time2, _ := time.Parse("2006-01-02 15:04:05", filteredTransactions[j].CreatedAt)
			if time1.Before(time2) {
				filteredTransactions[i], filteredTransactions[j] = filteredTransactions[j], filteredTransactions[i]
			}
		}
	}

	total := len(filteredTransactions)

	// Apply pagination
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	paginatedTransactions := filteredTransactions[start:end]

	// Convert to API response format
	var apiTransactions []gin.H
	for _, tx := range paginatedTransactions {
		apiTransactions = append(apiTransactions, gin.H{
			"id":          tx.ID,
			"user_id":     tx.UserID,
			"username":    tx.Username,
			"amount":      tx.Amount,
			"status":      tx.Status,
			"qris_code":   tx.QRISCode,
			"created_at":  tx.CreatedAt,
			"expired_at":  tx.ExpiredAt,
			"approved_by": tx.ApprovedBy,
			"approved_at": tx.ApprovedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    apiTransactions,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// Bulk approve multiple transactions
func BulkApproveTransactions(c *gin.Context) {
	var req struct {
		TransactionIDs []string `json:"transaction_ids" binding:"required"`
		AdminNote      string   `json:"admin_note"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	var results []gin.H
	successCount := 0
	failCount := 0

	for _, transactionID := range req.TransactionIDs {
		// Use the same ConfirmTopUp function that bot uses
		err := service.ConfirmTopUp(transactionID, 0) // 0 indicates API approval

		if err != nil {
			results = append(results, gin.H{
				"transaction_id": transactionID,
				"status":         "failed",
				"error":          err.Error(),
			})
			failCount++
			continue
		}

		// Send additional notification with admin note if provided
		if req.AdminNote != "" {
			service.TxMutex.RLock()
			transaction, exists := service.Transactions[transactionID]
			service.TxMutex.RUnlock()

			if exists {
				noteMessage := fmt.Sprintf("📝 Catatan Admin: %s", req.AdminNote)
				service.NotifyAdminError(transaction.UserID, "Admin Note", noteMessage)
			}
		}

		results = append(results, gin.H{
			"transaction_id": transactionID,
			"status":         "success",
		})
		successCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Bulk approval completed",
		"success_count": successCount,
		"fail_count":    failCount,
		"results":       results,
	})
}

// Helper functions
func formatAmount(amount int64) string {
	if amount < 1000 {
		return strconv.FormatInt(amount, 10)
	}

	// Convert to string
	str := strconv.FormatInt(amount, 10)

	// Add thousand separators
	result := ""
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result += "."
		}
		result += string(digit)
	}

	return result
}

func parseIntDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}

	return val
}

// CreateTopUpTransaction creates a new topup transaction via API
func CreateTopUpTransaction(c *gin.Context) {
	var req struct {
		UserID   int64  `json:"user_id" binding:"required"`
		Username string `json:"username" binding:"required"`
		Amount   int64  `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	// Validate amount
	if req.Amount < 10000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Minimal top up adalah Rp 10.000",
		})
		return
	}

	if req.Amount > 1000000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Maksimal top up adalah Rp 1.000.000",
		})
		return
	}

	// Create topup transaction using the same service function
	topUpResp, err := service.CreateTopUpTransaction(req.UserID, req.Username, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Top up transaction created successfully",
		"data": gin.H{
			"transaction_id": topUpResp.Data.TransactionID,
			"qris_code":      topUpResp.Data.QRISCode,
			"amount":         topUpResp.Data.Amount,
			"expired_at":     topUpResp.Data.ExpiredAt,
		},
	})
}

// GetUserBalance gets user balance via API
func GetUserBalance(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid user ID",
		})
		return
	}

	// Get user balance using the same service function
	balance := service.GetUserBalance(userID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"user_id": balance.UserID,
			"balance": balance.Balance,
		},
	})
}
