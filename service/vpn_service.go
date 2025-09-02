package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/nabilulilalbab/bottele/config"
	"github.com/nabilulilalbab/bottele/models"
	"gorm.io/gorm"
)

const (
	VPN_API_BASE_URL = "http://128.199.227.169:37849/api/v1/vpn"
	VPN_AUTH_URL     = "http://128.199.227.169:37849/api/v1/auth/login"
	VPN_USERNAME     = "admin"
	VPN_PASSWORD     = "db4bb47cd788"
	VPN_PRICE_PER_DAY = 266.666666667 // 8000 / 30 hari
	VPN_MIN_BALANCE   = 10000          // Minimal saldo 10000
	VPN_TIMEOUT       = 60             // Timeout 60 detik
)

var (
	currentVPNToken string
	tokenExpiry     time.Time
)

// VPN API Request/Response structures
type VPNCreateRequest struct {
	Days     int    `json:"days"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
	Username string `json:"username"`
}

type VPNExtendRequest struct {
	Days int `json:"days"`
}

type VPNCreateResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    VPNUserData `json:"data"`
}

type VPNUserData struct {
	Protocol string                 `json:"protocol"`
	Server   string                 `json:"server"`
	Port     int                    `json:"port"`
	Username string                 `json:"username"`
	Password string                 `json:"password,omitempty"`
	UUID     string                 `json:"uuid,omitempty"`
	Config   map[string]interface{} `json:"config"`
}

type VPNExtendResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type VPNLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type VPNLoginResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    VPNAuthData `json:"data"`
}

type VPNAuthData struct {
	Token     string `json:"token"`
	Username  string `json:"username"`
	ExpiresAt string `json:"expires_at"`
}

// CalculateVPNPrice menghitung harga VPN berdasarkan jumlah hari
func CalculateVPNPrice(days int) int64 {
	return int64(float64(days) * VPN_PRICE_PER_DAY)
}

// getVPNToken mendapatkan token VPN yang valid
func getVPNToken() (string, error) {
	// Check if current token is still valid (with 5 minute buffer)
	if currentVPNToken != "" && time.Now().Add(5*time.Minute).Before(tokenExpiry) {
		return currentVPNToken, nil
	}
	
	log.Printf("Getting new VPN token...")
	
	// Login to get new token
	reqBody := VPNLoginRequest{
		Username: VPN_USERNAME,
		Password: VPN_PASSWORD,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("error marshaling login request: %v", err)
	}
	
	req, err := http.NewRequest("POST", VPN_AUTH_URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating login request: %v", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("accept", "application/json")
	
	client := &http.Client{
		Timeout: VPN_TIMEOUT * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making login request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var loginResp VPNLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("error decoding login response: %v", err)
	}
	
	if !loginResp.Success {
		return "", fmt.Errorf("login failed: %s", loginResp.Message)
	}
	
	// Parse expiry time
	expiryTime, err := time.Parse("2006-01-02T15:04:05.999999999Z07:00", loginResp.Data.ExpiresAt)
	if err != nil {
		log.Printf("Warning: Could not parse expiry time: %v", err)
		// Set expiry to 1 hour from now as fallback
		expiryTime = time.Now().Add(1 * time.Hour)
	}
	
	// Update global variables
	currentVPNToken = loginResp.Data.Token
	tokenExpiry = expiryTime
	
	log.Printf("VPN token refreshed, expires at: %s", tokenExpiry.Format("2006-01-02 15:04:05"))
	
	return currentVPNToken, nil
}

// CreateVPNUser membuat user VPN baru
func CreateVPNUser(userID int64, username, email, password, protocol string, days int) (*models.VPNTransaction, error) {
	db := config.DB
	
	// Validasi input
	if days <= 0 {
		return nil, fmt.Errorf("jumlah hari harus lebih dari 0")
	}
	
	if protocol != "ssh" && protocol != "trojan" && protocol != "vless" && protocol != "vmess" {
		return nil, fmt.Errorf("protocol tidak valid. Pilih: ssh, trojan, vless, atau vmess")
	}
	
	// Hitung harga
	price := CalculateVPNPrice(days)
	
	// Cek saldo user
	balance := GetUserBalance(userID)
	if balance.Balance < VPN_MIN_BALANCE {
		return nil, fmt.Errorf("saldo minimal untuk VPN adalah Rp %d", VPN_MIN_BALANCE)
	}
	
	if balance.Balance < price {
		return nil, fmt.Errorf("saldo tidak mencukupi. Dibutuhkan: Rp %d, Saldo Anda: Rp %d", 
			price, balance.Balance)
	}
	
	// Generate unique VPN username
	vpnUsername := fmt.Sprintf("grn_%d_%d", userID, time.Now().Unix())
	
	// Create transaction record
	txID := generateTransactionID()
	vpnTx := &models.VPNTransaction{
		ID:       txID,
		UserID:   userID,
		Username: vpnUsername,
		Email:    email,
		Password: password,
		Protocol: protocol,
		Days:     days,
		Price:    price,
		Status:   "pending",
	}
	
	if err := db.Create(vpnTx).Error; err != nil {
		return nil, fmt.Errorf("gagal menyimpan transaksi VPN: %v", err)
	}
	
	// Call VPN API
	apiResp, err := callVPNCreateAPI(vpnUsername, email, password, protocol, days)
	if err != nil {
		// Update transaction status to failed
		vpnTx.Status = "failed"
		db.Save(vpnTx)
		return nil, fmt.Errorf("gagal membuat VPN: %v", err)
	}
	
	if !apiResp.Success {
		// Update transaction status to failed
		vpnTx.Status = "failed"
		db.Save(vpnTx)
		return nil, fmt.Errorf("gagal membuat VPN: %s", apiResp.Message)
	}
	
	// Deduct balance
	err = DeductUserBalance(userID, price)
	if err != nil {
		log.Printf("Error deducting balance for VPN user %d: %v", userID, err)
		// Try to delete VPN user if balance deduction fails (optional)
		// We'll keep the VPN but mark transaction as failed
		vpnTx.Status = "failed"
		db.Save(vpnTx)
		return nil, fmt.Errorf("gagal memotong saldo: %v", err)
	}
	
	// Save VPN user data
	configData, _ := json.Marshal(apiResp.Data.Config)
	vpnUser := &models.VPNUser{
		UserID:      userID,
		VPNUsername: vpnUsername,
		Protocol:    protocol,
		Server:      apiResp.Data.Server,
		Port:        apiResp.Data.Port,
		Password:    apiResp.Data.Password,
		UUID:        apiResp.Data.UUID,
		ConfigData:  string(configData),
		ExpiredAt:   time.Now().AddDate(0, 0, days),
	}
	
	if err := db.Create(vpnUser).Error; err != nil {
		log.Printf("Error saving VPN user data: %v", err)
		// Continue anyway, transaction is successful
	}
	
	// Update transaction status
	responseData, _ := json.Marshal(apiResp)
	vpnTx.Status = "success"
	vpnTx.ResponseData = string(responseData)
	db.Save(vpnTx)
	
	return vpnTx, nil
}

// ExtendVPNUser memperpanjang masa aktif VPN user
func ExtendVPNUser(userID int64, vpnUsername string, days int) error {
	db := config.DB
	
	// Validasi input
	if days <= 0 {
		return fmt.Errorf("jumlah hari harus lebih dari 0")
	}
	
	// Cek apakah VPN user exists dan milik user ini
	var vpnUser models.VPNUser
	if err := db.Where("user_id = ? AND vpn_username = ?", userID, vpnUsername).First(&vpnUser).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("VPN user tidak ditemukan")
		}
		return fmt.Errorf("gagal mengecek VPN user: %v", err)
	}
	
	// Hitung harga
	price := CalculateVPNPrice(days)
	
	// Cek saldo user
	balance := GetUserBalance(userID)
	if balance.Balance < price {
		return fmt.Errorf("saldo tidak mencukupi. Dibutuhkan: Rp %d, Saldo Anda: Rp %d", 
			price, balance.Balance)
	}
	
	// Call VPN extend API
	apiResp, err := callVPNExtendAPI(vpnUsername, vpnUser.Protocol, days)
	if err != nil {
		return fmt.Errorf("gagal memperpanjang VPN: %v", err)
	}
	
	if !apiResp.Success {
		return fmt.Errorf("gagal memperpanjang VPN: %s", apiResp.Message)
	}
	
	// Deduct balance
	err = DeductUserBalance(userID, price)
	if err != nil {
		log.Printf("Error deducting balance for VPN extend user %d: %v", userID, err)
		return fmt.Errorf("gagal memotong saldo: %v", err)
	}
	
	// Update VPN user expiry
	vpnUser.ExpiredAt = vpnUser.ExpiredAt.AddDate(0, 0, days)
	if err := db.Save(&vpnUser).Error; err != nil {
		log.Printf("Error updating VPN user expiry: %v", err)
		// Continue anyway, extension is successful
	}
	
	// Create transaction record for extension
	txID := generateTransactionID()
	vpnTx := &models.VPNTransaction{
		ID:       txID,
		UserID:   userID,
		Username: vpnUsername,
		Email:    "extend",
		Password: "extend",
		Protocol: vpnUser.Protocol,
		Days:     days,
		Price:    price,
		Status:   "success",
	}
	
	if err := db.Create(vpnTx).Error; err != nil {
		log.Printf("Error saving VPN extend transaction: %v", err)
		// Continue anyway
	}
	
	return nil
}

// GetUserVPNs mendapatkan daftar VPN user
func GetUserVPNs(userID int64) ([]models.VPNUser, error) {
	db := config.DB
	
	var vpnUsers []models.VPNUser
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&vpnUsers).Error; err != nil {
		return nil, fmt.Errorf("gagal mengambil data VPN: %v", err)
	}
	
	return vpnUsers, nil
}

// GetVPNTransactionHistory mendapatkan riwayat transaksi VPN user
func GetVPNTransactionHistory(userID int64) ([]models.VPNTransaction, error) {
	db := config.DB
	
	var transactions []models.VPNTransaction
	if err := db.Where("user_id = ?", userID).Order("created_at DESC").Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("gagal mengambil riwayat VPN: %v", err)
	}
	
	return transactions, nil
}

// callVPNCreateAPI calls the VPN creation API with retry mechanism
func callVPNCreateAPI(username, email, password, protocol string, days int) (*VPNCreateResponse, error) {
	// Try up to 3 times for EOF errors
	for attempt := 1; attempt <= 3; attempt++ {
		log.Printf("VPN API Attempt %d for protocol %s", attempt, protocol)
		
		resp, err := makeVPNCreateRequest(username, email, password, protocol, days)
		if err != nil {
			if attempt < 3 && (err.Error() == "EOF" || err.Error() == "unexpected EOF") {
				log.Printf("EOF error on attempt %d, retrying...", attempt)
				time.Sleep(time.Duration(attempt) * time.Second) // Progressive delay
				continue
			}
			return nil, err
		}
		return resp, nil
	}
	return nil, fmt.Errorf("failed after 3 attempts")
}

// makeVPNCreateRequest makes the actual HTTP request
func makeVPNCreateRequest(username, email, password, protocol string, days int) (*VPNCreateResponse, error) {
	// Get valid token
	token, err := getVPNToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get VPN token: %v", err)
	}
	
	url := fmt.Sprintf("%s/%s/create", VPN_API_BASE_URL, protocol)
	
	reqBody := VPNCreateRequest{
		Days:     days,
		Email:    email,
		Password: password,
		Protocol: protocol,
		Username: username,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}
	
	log.Printf("VPN API Request to %s: %s", url, string(jsonData))
	
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	
	// Set headers sesuai dokumentasi dengan token dinamis
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	req.Header.Set("accept", "application/json")
	
	// Increase timeout dan disable keep-alive
	client := &http.Client{
		Timeout: VPN_TIMEOUT * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("VPN API Error: %v", err)
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()
	
	log.Printf("VPN API Response Status: %d", resp.StatusCode)
	
	// Check status code (201 untuk create, 200 untuk success)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		// Read response body for error details
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var apiResp VPNCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("VPN API Decode Error: %v", err)
		return nil, fmt.Errorf("error decoding response: %v", err)
	}
	
	log.Printf("VPN API Response: %+v", apiResp)
	
	return &apiResp, nil
}

// callVPNExtendAPI calls the VPN extend API
func callVPNExtendAPI(username, protocol string, days int) (*VPNExtendResponse, error) {
	// Get valid token
	token, err := getVPNToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get VPN token: %v", err)
	}
	
	url := fmt.Sprintf("%s/%s/users/%s/extend", VPN_API_BASE_URL, protocol, username)
	
	reqBody := VPNExtendRequest{
		Days: days,
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}
	
	log.Printf("VPN Extend API Request to %s: %s", url, string(jsonData))
	
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	
	// Set headers sesuai dokumentasi dengan token dinamis
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)
	req.Header.Set("accept", "application/json")
	
	// Increase timeout dan disable keep-alive
	client := &http.Client{
		Timeout: VPN_TIMEOUT * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}
	
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("VPN Extend API Error: %v", err)
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()
	
	log.Printf("VPN Extend API Response Status: %d", resp.StatusCode)
	
	// Check status code (200 untuk extend)
	if resp.StatusCode != 200 {
		// Read response body for error details
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var apiResp VPNExtendResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		log.Printf("VPN Extend API Decode Error: %v", err)
		return nil, fmt.Errorf("error decoding response: %v", err)
	}
	
	log.Printf("VPN Extend API Response: %+v", apiResp)
	
	return &apiResp, nil
}

// generateTransactionID generates a unique transaction ID
func generateTransactionID() string {
	return fmt.Sprintf("VPN_%d", time.Now().UnixNano())
}