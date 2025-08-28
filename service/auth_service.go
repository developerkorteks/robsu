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
	otpVerifyLoginURL = "https://grnstore.domcloud.dev/api/otp/verify"
)

// VerifyOTPAndLogin verifies OTP and gets access token
func VerifyOTPAndLogin(phoneNumber, otpCode string, userID int64) (*dto.OTPVerifyLoginResponse, error) {
	// Create verify request
	verifyReq := dto.OTPVerifyLoginRequest{
		PhoneNumber: phoneNumber,
		OTPCode:     otpCode,
	}

	jsonData, err := json.Marshal(verifyReq)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", otpVerifyLoginURL, bytes.NewBuffer(jsonData))
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

	var verifyResp dto.OTPVerifyLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("gagal decode response: %v", err)
	}

	if !verifyResp.Success {
		return nil, fmt.Errorf("verifikasi OTP gagal: %s", verifyResp.Message)
	}

	// Save user session to database
	err = SaveUserSession(userID, phoneNumber, verifyResp.Data.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("gagal menyimpan session: %v", err)
	}

	return &verifyResp, nil
}

// SaveUserSession saves user session with access token
func SaveUserSession(chatID int64, phoneNumber, accessToken string) error {
	// Token expires in 1 hour
	expiresAt := time.Now().Add(1 * time.Hour)

	user := models.User{
		ChatID:         chatID,
		PhoneNumber:    phoneNumber,
		AccessToken:    accessToken,
		TokenExpiresAt: &expiresAt,
		IsVerified:     true,
	}

	// Upsert user
	result := config.DB.Where("chat_id = ?", chatID).First(&models.User{})
	if result.Error != nil {
		// Create new user
		return config.DB.Create(&user).Error
	} else {
		// Update existing user
		return config.DB.Model(&models.User{}).Where("chat_id = ?", chatID).Updates(map[string]interface{}{
			"phone_number":     phoneNumber,
			"access_token":     accessToken,
			"token_expires_at": expiresAt,
			"is_verified":      true,
			"updated_at":       time.Now(),
		}).Error
	}
}

// GetUserSession gets user session from database
func GetUserSession(chatID int64) (*models.User, error) {
	var user models.User
	err := config.DB.Where("chat_id = ?", chatID).First(&user).Error
	if err != nil {
		return nil, err
	}

	// Check if token is expired
	if user.TokenExpiresAt != nil && time.Now().After(*user.TokenExpiresAt) {
		// Token expired, clear it
		config.DB.Model(&user).Updates(map[string]interface{}{
			"access_token":     "",
			"token_expires_at": nil,
			"is_verified":      false,
		})
		return nil, fmt.Errorf("token expired")
	}

	return &user, nil
}

// IsUserLoggedIn checks if user has valid session
func IsUserLoggedIn(chatID int64) bool {
	user, err := GetUserSession(chatID)
	return err == nil && user.AccessToken != "" && user.IsVerified
}

// ClearUserSession clears user session (logout)
func ClearUserSession(chatID int64) error {
	return config.DB.Model(&models.User{}).Where("chat_id = ?", chatID).Updates(map[string]interface{}{
		"access_token":     "",
		"token_expires_at": nil,
		"is_verified":      false,
		"updated_at":       time.Now(),
	}).Error
}

// AddActiveUserToDB adds user to active users in database
func AddActiveUserToDB(userID int64) error {
	activeUser := models.ActiveUser{
		UserID:          userID,
		LastInteraction: time.Now(),
	}

	// Upsert active user
	return config.DB.Save(&activeUser).Error
}

// GetAllUserIDsFromDB gets all user IDs from database
func GetAllUserIDsFromDB() ([]int64, error) {
	var activeUsers []models.ActiveUser
	err := config.DB.Find(&activeUsers).Error
	if err != nil {
		return nil, err
	}

	var userIDs []int64
	for _, user := range activeUsers {
		userIDs = append(userIDs, user.UserID)
	}

	return userIDs, nil
}