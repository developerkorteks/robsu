package models

import (
	"time"

	"gorm.io/gorm"
)

// User model untuk menyimpan data user dan session
type User struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ChatID          int64     `gorm:"uniqueIndex;not null" json:"chat_id"`
	PhoneNumber     string    `gorm:"not null" json:"phone_number"`
	AccessToken     string    `json:"access_token"`
	TokenExpiresAt  *time.Time `json:"token_expires_at"`
	IsVerified      bool      `gorm:"default:false" json:"is_verified"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Transaction model untuk top-up transactions
type Transaction struct {
	ID          string     `gorm:"primaryKey" json:"id"`
	UserID      int64      `gorm:"not null" json:"user_id"`
	Username    string     `gorm:"not null" json:"username"`
	Amount      int64      `gorm:"not null" json:"amount"`
	Status      string     `gorm:"default:pending" json:"status"`
	QRISCode    string     `json:"qris_code"`
	CreatedAt   time.Time  `json:"created_at"`
	ApprovedBy  *int64     `json:"approved_by"`
	ApprovedAt  *time.Time `json:"approved_at"`
	ExpiredAt   time.Time  `json:"expired_at"`
	User        User       `gorm:"foreignKey:UserID;references:ChatID" json:"user"`
}

// UserBalance model untuk saldo user
type UserBalance struct {
	UserID    int64     `gorm:"primaryKey" json:"user_id"`
	Balance   int64     `gorm:"default:0" json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID;references:ChatID" json:"user"`
}

// PurchaseTransaction model untuk transaksi pembelian
type PurchaseTransaction struct {
	ID           string    `gorm:"primaryKey" json:"id"`
	UserID       int64     `gorm:"not null" json:"user_id"`
	PackageCode  string    `gorm:"not null" json:"package_code"`
	PackageName  string    `gorm:"not null" json:"package_name"`
	PaymentMethod string   `gorm:"not null" json:"payment_method"`
	PhoneNumber  string    `gorm:"not null" json:"phone_number"`
	Price        int64     `gorm:"not null" json:"price"`
	Status       string    `gorm:"default:pending" json:"status"`
	ResponseData string    `json:"response_data"` // JSON response from API
	CreatedAt    time.Time `json:"created_at"`
	User         User      `gorm:"foreignKey:UserID;references:ChatID" json:"user"`
}

// ActiveUser model untuk tracking user interactions
type ActiveUser struct {
	UserID          int64     `gorm:"primaryKey" json:"user_id"`
	LastInteraction time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"last_interaction"`
}

// OTPSession model untuk tracking OTP sessions
type OTPSession struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      int64     `gorm:"not null" json:"user_id"`
	PhoneNumber string    `gorm:"not null" json:"phone_number"`
	AuthID      string    `gorm:"not null" json:"auth_id"`
	Status      string    `gorm:"default:pending" json:"status"` // pending, verified, expired
	CreatedAt   time.Time `json:"created_at"`
	ExpiredAt   time.Time `json:"expired_at"`
}

// AutoMigrate runs database migrations
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&Transaction{},
		&UserBalance{},
		&PurchaseTransaction{},
		&ActiveUser{},
		&OTPSession{},
	)
}