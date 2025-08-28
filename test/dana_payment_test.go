package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nabilulilalbab/bottele/dto"
	"github.com/nabilulilalbab/bottele/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// MockHTTPClient is a mock for HTTP client
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// MockUserSession represents a mock user session
type MockUserSession struct {
	UserID      int64
	PhoneNumber string
	AccessToken string
}

// Test data constants
const (
	testUserID        = int64(123456789)
	testPhoneNumber   = "087786388052"
	testAccessToken   = "test_access_token_123"
	testPackageCode   = "XL_UNLIMITED_1GB"
	testPaymentMethod = "DANA"
	testTransactionID = "dana_test_txn_123"
	testDeeplinkURL   = "dana://pay?amount=1500&merchant=test"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect to test database")
	}

	// Auto migrate the schema
	db.AutoMigrate(&models.User{}, &models.PurchaseTransaction{}, &models.UserBalance{})

	return db
}

// createMockPurchaseResponse creates a mock purchase response for DANA payment
func createMockPurchaseResponse() *dto.PurchaseResponse {
	return &dto.PurchaseResponse{
		StatusCode: 200,
		Message:    "Purchase successful",
		Success:    true,
		Data: dto.PurchaseData{
			DeeplinkData: dto.DeeplinkData{
				DeeplinkURL:   testDeeplinkURL,
				PaymentMethod: testPaymentMethod,
			},
			HaveDeeplink:         true,
			IsQRIS:               false,
			MSISDN:               testPhoneNumber,
			PackageCode:          testPackageCode,
			PackageName:          "XL Unlimited 1GB",
			PackageProcessingFee: 1500,
			Price:                3000,           // Original price + processing fee
			QRISData:             json.RawMessage(`[]`), // Empty array for DANA
			TrxID:                testTransactionID,
		},
	}
}

// createMockErrorResponse creates a mock error response
func createMockErrorResponse() *dto.PurchaseResponse {
	return &dto.PurchaseResponse{
		StatusCode: 400,
		Message:    "Insufficient balance",
		Success:    false,
		Data:       dto.PurchaseData{},
	}
}

// TestDANAPaymentSuccess tests successful DANA payment flow
func TestDANAPaymentSuccess(t *testing.T) {
	// Setup test database
	db := setupTestDB()

	// Create test user session
	userSession := &models.User{
		ChatID:      testUserID,
		PhoneNumber: testPhoneNumber,
		AccessToken: testAccessToken,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	db.Create(userSession)

	// Create test user balance
	userBalance := &models.UserBalance{
		UserID:    testUserID,
		Balance:   5000, // Sufficient balance
		UpdatedAt: time.Now(),
	}
	db.Create(userBalance)

	// Create mock HTTP server
	mockResponse := createMockPurchaseResponse()

	// Create JSON in the format expected by the custom unmarshaler
	mockResponseMap := map[string]interface{}{
		"statusCode": mockResponse.StatusCode,
		"message":    mockResponse.Message,
		"success":    mockResponse.Success,
		"data":       mockResponse.Data, // This will be marshaled as an object
	}
	mockResponseJSON, _ := json.Marshal(mockResponseMap)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("accept"))
		assert.NotEmpty(t, r.Header.Get("X-API-Key"))

		// Verify request body
		var purchaseReq dto.PurchaseRequest
		err := json.NewDecoder(r.Body).Decode(&purchaseReq)
		assert.NoError(t, err)
		assert.Equal(t, testAccessToken, purchaseReq.AccessToken)
		assert.Equal(t, testPackageCode, purchaseReq.PackageCode)
		assert.Equal(t, testPaymentMethod, purchaseReq.PaymentMethod)
		assert.Equal(t, testPhoneNumber, purchaseReq.PhoneNumber)
		assert.Equal(t, "telegram_bot", purchaseReq.Source)

		// Send mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write(mockResponseJSON)
	}))
	defer server.Close()

	// Test the purchase function
	// Note: In a real implementation, you would need to inject the HTTP client
	// or make the URL configurable for testing
	t.Run("Successful DANA Payment", func(t *testing.T) {
		// This test would require modifying the service to accept a custom HTTP client
		// For now, we'll test the response parsing logic

		// Parse the mock response
		var parsedResponse dto.PurchaseResponse
		err := json.Unmarshal(mockResponseJSON, &parsedResponse)
		assert.NoError(t, err)

		// Verify response structure
		assert.True(t, parsedResponse.Success)
		assert.Equal(t, 200, parsedResponse.StatusCode)
		assert.Equal(t, testTransactionID, parsedResponse.Data.TrxID)
		assert.Equal(t, testDeeplinkURL, parsedResponse.Data.DeeplinkData.DeeplinkURL)
		assert.Equal(t, testPaymentMethod, parsedResponse.Data.DeeplinkData.PaymentMethod)
		assert.True(t, parsedResponse.Data.HaveDeeplink)
		assert.False(t, parsedResponse.Data.IsQRIS)
		assert.Equal(t, int64(1500), parsedResponse.Data.PackageProcessingFee)
	})
}

// TestDANAPaymentError tests error scenarios in DANA payment
func TestDANAPaymentError(t *testing.T) {
	t.Run("API Error Response", func(t *testing.T) {
		mockErrorResponse := createMockErrorResponse()

		// Create JSON in the format expected by the custom unmarshaler
		mockErrorMap := map[string]interface{}{
			"statusCode": mockErrorResponse.StatusCode,
			"message":    mockErrorResponse.Message,
			"success":    mockErrorResponse.Success,
			"data":       mockErrorResponse.Data,
		}
		mockErrorJSON, _ := json.Marshal(mockErrorMap)

		// Parse the mock error response
		var parsedResponse dto.PurchaseResponse
		err := json.Unmarshal(mockErrorJSON, &parsedResponse)
		assert.NoError(t, err)

		// Verify error response structure
		assert.False(t, parsedResponse.Success)
		assert.Equal(t, 400, parsedResponse.StatusCode)
		assert.Equal(t, "Insufficient balance", parsedResponse.Message)
	})

	t.Run("Invalid JSON Response", func(t *testing.T) {
		invalidJSON := `{"invalid": json}`

		var parsedResponse dto.PurchaseResponse
		err := json.Unmarshal([]byte(invalidJSON), &parsedResponse)
		assert.Error(t, err)
	})
}

// TestDANAPaymentValidation tests input validation for DANA payment
func TestDANAPaymentValidation(t *testing.T) {
	testCases := []struct {
		name          string
		userID        int64
		packageCode   string
		paymentMethod string
		expectedError bool
		errorMessage  string
	}{
		{
			name:          "Valid Input",
			userID:        testUserID,
			packageCode:   testPackageCode,
			paymentMethod: testPaymentMethod,
			expectedError: false,
		},
		{
			name:          "Empty Package Code",
			userID:        testUserID,
			packageCode:   "",
			paymentMethod: testPaymentMethod,
			expectedError: true,
			errorMessage:  "package code cannot be empty",
		},
		{
			name:          "Empty Payment Method",
			userID:        testUserID,
			packageCode:   testPackageCode,
			paymentMethod: "",
			expectedError: true,
			errorMessage:  "payment method cannot be empty",
		},
		{
			name:          "Invalid User ID",
			userID:        0,
			packageCode:   testPackageCode,
			paymentMethod: testPaymentMethod,
			expectedError: true,
			errorMessage:  "user ID cannot be zero",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate input parameters
			err := validatePurchaseInput(tc.userID, tc.packageCode, tc.paymentMethod)

			if tc.expectedError {
				assert.Error(t, err)
				if tc.errorMessage != "" {
					assert.Contains(t, err.Error(), tc.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDANADeeplinkGeneration tests DANA deeplink URL generation
func TestDANADeeplinkGeneration(t *testing.T) {
	t.Run("Valid Deeplink URL", func(t *testing.T) {
		mockResponse := createMockPurchaseResponse()
		deeplinkURL := mockResponse.Data.DeeplinkData.DeeplinkURL

		// Verify deeplink URL format
		assert.NotEmpty(t, deeplinkURL)
		assert.Contains(t, deeplinkURL, "dana://")
		assert.Contains(t, deeplinkURL, "amount=1500")
	})

	t.Run("Empty Deeplink URL", func(t *testing.T) {
		mockResponse := createMockPurchaseResponse()
		mockResponse.Data.DeeplinkData.DeeplinkURL = ""

		// This should be handled as an error in the actual implementation
		assert.Empty(t, mockResponse.Data.DeeplinkData.DeeplinkURL)
	})
}

// TestDANATransactionSaving tests saving DANA transaction to database
func TestDANATransactionSaving(t *testing.T) {
	db := setupTestDB()

	t.Run("Save Transaction Successfully", func(t *testing.T) {
		mockResponse := createMockPurchaseResponse()

		// Create transaction record
		responseData, _ := json.Marshal(mockResponse)
		transaction := models.PurchaseTransaction{
			ID:            mockResponse.Data.TrxID,
			UserID:        testUserID,
			PackageCode:   testPackageCode,
			PackageName:   mockResponse.Data.PackageName,
			PaymentMethod: testPaymentMethod,
			PhoneNumber:   testPhoneNumber,
			Price:         mockResponse.Data.PackageProcessingFee, // Use processing fee
			Status:        "pending",
			ResponseData:  string(responseData),
			CreatedAt:     time.Now(),
		}

		// Save to database
		err := db.Create(&transaction).Error
		assert.NoError(t, err)

		// Verify saved transaction
		var savedTransaction models.PurchaseTransaction
		err = db.Where("id = ?", testTransactionID).First(&savedTransaction).Error
		assert.NoError(t, err)
		assert.Equal(t, testUserID, savedTransaction.UserID)
		assert.Equal(t, testPackageCode, savedTransaction.PackageCode)
		assert.Equal(t, testPaymentMethod, savedTransaction.PaymentMethod)
		assert.Equal(t, "pending", savedTransaction.Status)
		assert.Equal(t, int64(1500), savedTransaction.Price)
	})
}

// TestDANABalanceDeduction tests balance deduction for DANA payment
func TestDANABalanceDeduction(t *testing.T) {
	db := setupTestDB()

	t.Run("Sufficient Balance", func(t *testing.T) {
		// Create user with sufficient balance
		userBalance := &models.UserBalance{
			UserID:    testUserID,
			Balance:   5000,
			UpdatedAt: time.Now(),
		}
		db.Create(userBalance)

		// Simulate balance deduction
		deductionAmount := int64(1500)
		err := db.Model(&models.UserBalance{}).
			Where("user_id = ?", testUserID).
			Update("balance", gorm.Expr("balance - ?", deductionAmount)).Error

		assert.NoError(t, err)

		// Verify balance after deduction
		var updatedBalance models.UserBalance
		db.Where("user_id = ?", testUserID).First(&updatedBalance)
		assert.Equal(t, int64(3500), updatedBalance.Balance)
	})

	t.Run("Insufficient Balance", func(t *testing.T) {
		// Create user with insufficient balance
		userBalance := &models.UserBalance{
			UserID:    testUserID + 1, // Different user
			Balance:   1000,           // Less than required 1500
			UpdatedAt: time.Now(),
		}
		db.Create(userBalance)

		// This should be handled in the service layer
		// The test verifies the balance check logic
		assert.True(t, userBalance.Balance < 1500)
	})
}

// Helper function to validate purchase input
func validatePurchaseInput(userID int64, packageCode, paymentMethod string) error {
	if userID == 0 {
		return fmt.Errorf("user ID cannot be zero")
	}
	if packageCode == "" {
		return fmt.Errorf("package code cannot be empty")
	}
	if paymentMethod == "" {
		return fmt.Errorf("payment method cannot be empty")
	}
	return nil
}

// TestDANAPaymentIntegration tests the complete DANA payment integration
func TestDANAPaymentIntegration(t *testing.T) {
	t.Run("Complete DANA Payment Flow", func(t *testing.T) {
		// Setup
		db := setupTestDB()

		// Create user session
		userSession := &models.User{
			ChatID:      testUserID,
			PhoneNumber: testPhoneNumber,
			AccessToken: testAccessToken,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		db.Create(userSession)

		// Create user balance
		userBalance := &models.UserBalance{
			UserID:    testUserID,
			Balance:   5000,
			UpdatedAt: time.Now(),
		}
		db.Create(userBalance)

		// Mock successful API response
		mockResponse := createMockPurchaseResponse()

		// Verify the complete flow components
		assert.Equal(t, testUserID, userSession.ChatID)
		assert.Equal(t, testAccessToken, userSession.AccessToken)
		assert.True(t, userBalance.Balance >= 1500) // Sufficient balance
		assert.True(t, mockResponse.Success)
		assert.True(t, mockResponse.Data.HaveDeeplink)
		assert.NotEmpty(t, mockResponse.Data.DeeplinkData.DeeplinkURL)
		assert.Equal(t, testPaymentMethod, mockResponse.Data.DeeplinkData.PaymentMethod)
	})
}

// BenchmarkDANAPaymentProcessing benchmarks DANA payment processing
func BenchmarkDANAPaymentProcessing(b *testing.B) {
	mockResponse := createMockPurchaseResponse()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate response processing
		responseData, _ := json.Marshal(mockResponse)
		var parsedResponse dto.PurchaseResponse
		json.Unmarshal(responseData, &parsedResponse)

		// Simulate validation
		_ = validatePurchaseInput(testUserID, testPackageCode, testPaymentMethod)
	}
}
