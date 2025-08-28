package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/nabilulilalbab/bottele/config"
	"github.com/nabilulilalbab/bottele/dto"
	"github.com/nabilulilalbab/bottele/models"
)

const (
	purchaseURL         = "https://grnstore.domcloud.dev/api/purchase"
	transactionCheckURL = "https://grnstore.domcloud.dev/api/transaction/check"
)

// PurchaseProduct makes a purchase using access token
func PurchaseProduct(userID int64, packageCode, paymentMethod string) (*dto.PurchaseResponse, error) {
	// Check cooldown to prevent spam
	if err := CheckUserActionCooldown(userID, 10); err != nil {
		return nil, err
	}

	// Acquire transaction lock to prevent race conditions
	lock := AcquireTransactionLock(userID)
	defer ReleaseTransactionLock(lock)

	// Set action time after acquiring lock
	SetUserActionTime(userID)

	// Get user session
	user, err := GetUserSession(userID)
	if err != nil {
		NotifyAdminError(userID, "Purchase", fmt.Sprintf("User session error: %v", err))
		return nil, fmt.Errorf("sesi login tidak valid, silakan login ulang")
	}

	if user.AccessToken == "" {
		NotifyAdminError(userID, "Purchase", "No valid access token")
		return nil, fmt.Errorf("sesi login tidak valid, silakan login ulang")
	}

	// Create purchase request
	purchaseReq := dto.PurchaseRequest{
		AccessToken:   user.AccessToken,
		PackageCode:   packageCode,
		PaymentMethod: paymentMethod,
		PhoneNumber:   user.PhoneNumber,
		Source:        "telegram_bot",
	}

	jsonData, err := json.Marshal(purchaseReq)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", purchaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request: %v", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		NotifyAdminError(userID, "Purchase API", fmt.Sprintf("HTTP request failed: %v", err))
		return nil, fmt.Errorf("terjadi kesalahan sistem, silakan coba lagi")
	}
	defer resp.Body.Close()

	var purchaseResp dto.PurchaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&purchaseResp); err != nil {
		NotifyAdminError(userID, "Purchase API", fmt.Sprintf("Response decode error: %v", err))
		return nil, fmt.Errorf("terjadi kesalahan sistem, silakan coba lagi")
	}

	// Check if API returned error
	if !purchaseResp.Success {
		// Log API errors to admin but show user-friendly message
		NotifyAdminError(userID, "Purchase API", fmt.Sprintf("API returned error: %s", purchaseResp.Message))
		return nil, fmt.Errorf("pembelian tidak dapat diproses saat ini, silakan coba lagi nanti")
	}

	// Get package price from API (already includes +1500)
	packagePrice, err := GetPackagePrice(packageCode)
	if err != nil {
		fmt.Printf("Warning: failed to get package price: %v\n", err)
		packagePrice = 1500 // Default fallback
	}

	// Set the correct price (from API that already includes +1500)
	purchaseResp.Data.Price = packagePrice

	// Save purchase transaction to database
	err = SavePurchaseTransaction(userID, packageCode, paymentMethod, user.PhoneNumber, &purchaseResp)
	if err != nil {
		// Log error to admin but don't fail the purchase
		NotifyAdminError(userID, "Database", fmt.Sprintf("Failed to save purchase transaction: %v", err))
	}

	return &purchaseResp, nil
}

// SavePurchaseTransaction saves purchase transaction to database
func SavePurchaseTransaction(userID int64, packageCode, paymentMethod, phoneNumber string, response *dto.PurchaseResponse) error {
	responseData, _ := json.Marshal(response)

	transaction := models.PurchaseTransaction{
		ID:            response.Data.TrxID,
		UserID:        userID,
		PackageCode:   packageCode,
		PackageName:   response.Data.PackageName,
		PaymentMethod: paymentMethod,
		PhoneNumber:   phoneNumber,
		Price:         response.Data.Price, // Use the corrected price
		Status:        "pending",
		ResponseData:  string(responseData),
		CreatedAt:     time.Now(),
	}

	return config.DB.Create(&transaction).Error
}

// CheckTransactionStatus checks transaction status
func CheckTransactionStatus(transactionID string) (*dto.TransactionCheckResponse, error) {
	checkReq := map[string]string{
		"transaction_id": transactionID,
	}

	jsonData, err := json.Marshal(checkReq)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", transactionCheckURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("gagal membuat request: %v", err)
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal mengirim request: %v", err)
	}
	defer resp.Body.Close()

	var checkResp dto.TransactionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&checkResp); err != nil {
		return nil, fmt.Errorf("gagal decode response: %v", err)
	}

	// Update transaction status in database
	if checkResp.Success {
		status := "success"
		if checkResp.Data.Status != 1 {
			status = "failed"
		}

		config.DB.Model(&models.PurchaseTransaction{}).Where("id = ?", transactionID).Update("status", status)
	}

	return &checkResp, nil
}

// GetPackagePrice gets the original price of a package
func GetPackagePrice(packageCode string) (int64, error) {
	packages, err := FetchPackages()
	if err != nil {
		return 0, err
	}

	for _, pkg := range packages {
		if pkg.PackageCode == packageCode {
			return pkg.Price, nil
		}
	}

	return 0, fmt.Errorf("package not found")
}

// GetAvailablePaymentMethods gets available payment methods for a package
func GetAvailablePaymentMethods(packageCode string) ([]dto.PaymentMethod, error) {
	packages, err := FetchPackages()
	if err != nil {
		return nil, err
	}

	for _, pkg := range packages {
		if pkg.PackageCode == packageCode {
			return pkg.AvailablePaymentMethods, nil
		}
	}

	return nil, fmt.Errorf("package not found")
}

// GetPurchaseTransaction gets a purchase transaction from database by transaction ID
func GetPurchaseTransaction(transactionID string) (*models.PurchaseTransaction, error) {
	var transaction models.PurchaseTransaction
	err := config.DB.Where("id = ?", transactionID).First(&transaction).Error
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}
