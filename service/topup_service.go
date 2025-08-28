package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/nabilulilalbab/bottele/config"
	"github.com/nabilulilalbab/bottele/dto"
	"github.com/nabilulilalbab/bottele/models"
	"gorm.io/gorm"
)

// In-memory storage untuk demo (dalam production gunakan database)
var (
	Transactions = make(map[string]*dto.Transaction)
	userBalances = make(map[int64]*dto.UserBalance)
	activeUsers  = make(map[int64]bool) // Track all users who interacted with bot
	TxMutex      sync.RWMutex
	balMutex     sync.RWMutex
	userMutex    sync.RWMutex
)

// CreateTopUpTransaction membuat transaksi top-up baru dengan QRIS dinamis
func CreateTopUpTransaction(userID int64, username string, amount int64) (*dto.TopUpResponse, error) {
	// Check cooldown to prevent spam
	if err := CheckUserActionCooldown(userID, 30); err != nil {
		return nil, err
	}

	// Acquire transaction lock to prevent race conditions
	lock := AcquireTransactionLock(userID)
	defer ReleaseTransactionLock(lock)

	// Set action time after acquiring lock
	SetUserActionTime(userID)

	// Generate transaction ID
	transactionID := fmt.Sprintf("TXN_%d_%d", userID, time.Now().Unix())

	// Generate QRIS dinamis
	qrisCode, err := GenerateDynamicQRIS(amount)
	if err != nil {
		NotifyAdminError(userID, "Topup QRIS", fmt.Sprintf("Failed to generate QRIS: %v", err))
		return nil, fmt.Errorf("terjadi kesalahan sistem, silakan coba lagi")
	}

	// Set expired time (30 menit dari sekarang)
	expiredAt := time.Now().Add(30 * time.Minute)

	// Create transaction
	transaction := &dto.Transaction{
		ID:        transactionID,
		UserID:    userID,
		Username:  username,
		Amount:    amount,
		Status:    "pending",
		QRISCode:  qrisCode,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		ExpiredAt: expiredAt.Format("2006-01-02 15:04:05"),
	}

	// Store transaction
	TxMutex.Lock()
	Transactions[transactionID] = transaction
	TxMutex.Unlock()

	// Sync to database
	if err := SyncTransactionToDatabase(transaction); err != nil {
		log.Printf("Warning: Failed to sync transaction to database: %v", err)
	}

	// Debug log
	log.Printf("Transaction created: ID=%s, UserID=%d, Amount=%d", transactionID, userID, amount)

	// Notify admin about topup request
	NotifyAdminTopupApproval(userID, amount, "QRIS")

	// Return response
	response := &dto.TopUpResponse{
		StatusCode: 200,
		Message:    "QRIS berhasil dibuat. Silakan lakukan pembayaran dalam 30 menit.",
		Success:    true,
		Data: dto.TopUpData{
			TransactionID: transactionID,
			QRISCode:      qrisCode,
			Amount:        amount,
			ExpiredAt:     expiredAt.Format("2006-01-02 15:04:05"),
		},
	}

	return response, nil
}

// GetPendingTransactions mendapatkan semua transaksi pending
func GetPendingTransactions() []*dto.Transaction {
	TxMutex.RLock()
	defer TxMutex.RUnlock()

	var pending []*dto.Transaction
	now := time.Now()

	for _, tx := range Transactions {
		// Check if expired
		expiredTime, err := time.Parse("2006-01-02 15:04:05", tx.ExpiredAt)
		if err == nil && now.After(expiredTime) && tx.Status == "pending" {
			tx.Status = "expired"
		}

		if tx.Status == "pending" {
			pending = append(pending, tx)
		}
	}

	return pending
}

// ConfirmTopUp mengkonfirmasi top-up oleh admin
func ConfirmTopUp(transactionID string, adminID int64) error {
	TxMutex.Lock()
	defer TxMutex.Unlock()

	// Debug log
	log.Printf("Attempting to confirm transaction: %s", transactionID)
	log.Printf("Available transactions: %d", len(Transactions))
	for id := range Transactions {
		log.Printf("  Available ID: %s", id)
	}

	// Get transaction
	tx, exists := Transactions[transactionID]
	if !exists {
		log.Printf("Transaction not found: %s", transactionID)
		return fmt.Errorf("transaksi tidak ditemukan")
	}

	if tx.Status != "pending" {
		return fmt.Errorf("transaksi sudah diproses atau expired")
	}

	// Check if expired
	expiredTime, err := time.Parse("2006-01-02 15:04:05", tx.ExpiredAt)
	if err == nil && time.Now().After(expiredTime) {
		tx.Status = "expired"
		return fmt.Errorf("transaksi sudah expired")
	}

	// Update transaction
	tx.Status = "confirmed"
	tx.ApprovedBy = adminID
	tx.ApprovedAt = time.Now().Format("2006-01-02 15:04:05")

	// Sync to database
	if err := SyncTransactionToDatabase(tx); err != nil {
		log.Printf("Warning: Failed to sync transaction to database: %v", err)
	}

	// Update user balance using AddUserBalance function
	err = AddUserBalance(tx.UserID, tx.Amount)
	if err != nil {
		log.Printf("Error adding balance for user %d: %v", tx.UserID, err)
		NotifyAdminError(tx.UserID, "Balance Update", fmt.Sprintf("Failed to add balance for topup %s: %v", transactionID, err))
		return fmt.Errorf("gagal menambah saldo user")
	}

	// Notify user about successful topup
	NotifyUserTopupSuccess(tx.UserID, tx.Amount, transactionID)

	return nil
}

// RejectTopUp menolak top-up oleh admin
func RejectTopUp(transactionID string, adminID int64) error {
	TxMutex.Lock()
	defer TxMutex.Unlock()

	// Get transaction
	tx, exists := Transactions[transactionID]
	if !exists {
		return fmt.Errorf("transaksi tidak ditemukan")
	}

	if tx.Status != "pending" {
		return fmt.Errorf("transaksi sudah diproses atau expired")
	}

	// Update transaction
	tx.Status = "rejected"
	tx.ApprovedBy = adminID
	tx.ApprovedAt = time.Now().Format("2006-01-02 15:04:05")

	// Sync to database
	if err := SyncTransactionToDatabase(tx); err != nil {
		log.Printf("Warning: Failed to sync transaction to database: %v", err)
	}

	return nil
}

// GetUserBalance mendapatkan saldo user dari database
func GetUserBalance(userID int64) *dto.UserBalance {
	var userBalance models.UserBalance
	err := config.DB.Where("user_id = ?", userID).First(&userBalance).Error
	if err != nil {
		// Create new balance record if not exists
		userBalance = models.UserBalance{
			UserID:    userID,
			Balance:   0,
			UpdatedAt: time.Now(),
		}
		config.DB.Create(&userBalance)
	}

	return &dto.UserBalance{
		UserID:  userBalance.UserID,
		Balance: userBalance.Balance,
	}
}

// DeductUserBalance memotong saldo user dengan atomic operation
func DeductUserBalance(userID int64, amount int64) error {
	balMutex.Lock()
	defer balMutex.Unlock()

	// Use database transaction for atomic operation
	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var userBalance models.UserBalance
	err := tx.Where("user_id = ?", userID).First(&userBalance).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("user balance not found")
	}

	if userBalance.Balance < amount {
		tx.Rollback()
		return fmt.Errorf("insufficient balance")
	}

	// Deduct balance atomically
	result := tx.Model(&userBalance).Where("user_id = ? AND balance >= ?", userID, amount).
		Update("balance", gorm.Expr("balance - ?", amount))

	if result.Error != nil {
		tx.Rollback()
		return result.Error
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return fmt.Errorf("insufficient balance or concurrent modification")
	}

	return tx.Commit().Error
}

// AddUserBalance menambah saldo user (untuk testing dan topup confirmation)
func AddUserBalance(userID int64, amount int64) error {
	var userBalance models.UserBalance
	err := config.DB.Where("user_id = ?", userID).First(&userBalance).Error
	if err != nil {
		// Create new balance record if not exists
		userBalance = models.UserBalance{
			UserID:    userID,
			Balance:   amount,
			UpdatedAt: time.Now(),
		}
		return config.DB.Create(&userBalance).Error
	}

	// Add to existing balance
	userBalance.Balance += amount
	userBalance.UpdatedAt = time.Now()

	return config.DB.Save(&userBalance).Error
}

// GetTransactionByUserID mendapatkan transaksi berdasarkan user ID
func GetTransactionByUserID(userID int64) *dto.Transaction {
	TxMutex.RLock()
	defer TxMutex.RUnlock()

	// Find latest pending transaction for user
	var latestTx *dto.Transaction
	var latestTime time.Time

	for _, tx := range Transactions {
		if tx.UserID == userID && tx.Status == "pending" {
			createdTime, err := time.Parse("2006-01-02 15:04:05", tx.CreatedAt)
			if err == nil && (latestTx == nil || createdTime.After(latestTime)) {
				latestTx = tx
				latestTime = createdTime
			}
		}
	}

	return latestTx
}

// SendWhatsAppNotification mengirim notifikasi WhatsApp ke admin
func SendWhatsAppNotification(message string) error {
	whatsappURL := "http://128.199.109.211:25120/send-message"
	adminNumber := "6285150588080"

	payload := map[string]string{
		"number":  adminNumber,
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling WhatsApp payload: %v", err)
		return err
	}

	// Send HTTP request to WhatsApp API
	req, err := http.NewRequest("POST", whatsappURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating WhatsApp request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending WhatsApp notification: %v", err)
		return err
	}
	defer resp.Body.Close()

	log.Printf("WhatsApp notification sent successfully to %s", adminNumber)
	return nil
}

// AddActiveUser menambahkan user ke daftar active users
func AddActiveUser(userID int64) {
	userMutex.Lock()
	activeUsers[userID] = true
	userMutex.Unlock()
}

// GetAllUserIDsFromData mendapatkan semua user ID dari data
func GetAllUserIDsFromData() []int64 {
	var userIDs []int64

	// Get from active users (all users who ever interacted)
	userMutex.RLock()
	for userID := range activeUsers {
		userIDs = append(userIDs, userID)
	}
	userMutex.RUnlock()

	return userIDs
}

// SyncTransactionToDatabase menyinkronkan transaksi dari in-memory ke database
func SyncTransactionToDatabase(tx *dto.Transaction) error {
	// Parse times
	createdAt, err := time.Parse("2006-01-02 15:04:05", tx.CreatedAt)
	if err != nil {
		return fmt.Errorf("invalid created_at format: %v", err)
	}

	expiredAt, err := time.Parse("2006-01-02 15:04:05", tx.ExpiredAt)
	if err != nil {
		return fmt.Errorf("invalid expired_at format: %v", err)
	}

	// Create database transaction model
	dbTx := models.Transaction{
		ID:        tx.ID,
		UserID:    tx.UserID,
		Username:  tx.Username,
		Amount:    tx.Amount,
		Status:    tx.Status,
		QRISCode:  tx.QRISCode,
		CreatedAt: createdAt,
		ExpiredAt: expiredAt,
	}

	// Set approved fields if available
	if tx.ApprovedBy != 0 {
		dbTx.ApprovedBy = &tx.ApprovedBy
	}

	if tx.ApprovedAt != "" {
		approvedAt, err := time.Parse("2006-01-02 15:04:05", tx.ApprovedAt)
		if err == nil {
			dbTx.ApprovedAt = &approvedAt
		}
	}

	// Save to database (upsert)
	return config.DB.Save(&dbTx).Error
}

// LoadTransactionsFromDatabase memuat transaksi dari database ke in-memory
func LoadTransactionsFromDatabase() error {
	var dbTransactions []models.Transaction
	err := config.DB.Find(&dbTransactions).Error
	if err != nil {
		return err
	}

	TxMutex.Lock()
	defer TxMutex.Unlock()

	// Clear existing in-memory transactions
	Transactions = make(map[string]*dto.Transaction)

	// Load from database
	for _, dbTx := range dbTransactions {
		tx := &dto.Transaction{
			ID:        dbTx.ID,
			UserID:    dbTx.UserID,
			Username:  dbTx.Username,
			Amount:    dbTx.Amount,
			Status:    dbTx.Status,
			QRISCode:  dbTx.QRISCode,
			CreatedAt: dbTx.CreatedAt.Format("2006-01-02 15:04:05"),
			ExpiredAt: dbTx.ExpiredAt.Format("2006-01-02 15:04:05"),
		}

		if dbTx.ApprovedBy != nil {
			tx.ApprovedBy = *dbTx.ApprovedBy
		}

		if dbTx.ApprovedAt != nil {
			tx.ApprovedAt = dbTx.ApprovedAt.Format("2006-01-02 15:04:05")
		}

		Transactions[tx.ID] = tx
	}

	log.Printf("Loaded %d transactions from database", len(Transactions))
	return nil
}
