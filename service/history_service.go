package service

import (
	"github.com/nabilulilalbab/bottele/config"
	"github.com/nabilulilalbab/bottele/models"
)

// GetUserPurchaseHistory gets user's purchase history from database
func GetUserPurchaseHistory(userID int64) ([]models.PurchaseTransaction, error) {
	var history []models.PurchaseTransaction
	
	err := config.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&history).Error
	
	return history, err
}

// GetPurchaseTransactionByID gets specific purchase transaction
func GetPurchaseTransactionByID(transactionID string) (*models.PurchaseTransaction, error) {
	var transaction models.PurchaseTransaction
	
	err := config.DB.Where("id = ?", transactionID).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	
	return &transaction, nil
}

// UpdatePurchaseTransactionStatus updates transaction status
func UpdatePurchaseTransactionStatus(transactionID, status string) error {
	return config.DB.Model(&models.PurchaseTransaction{}).
		Where("id = ?", transactionID).
		Update("status", status).Error
}