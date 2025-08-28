package dto

type GetPackageRequest struct {
	Limit int `json:"limit"`
	Page  int `json:"page"`
}

type OTPRequest struct {
	PhoneNumber string `json:"phone_number"`
}

type OTPVerifyRequest struct {
	AuthID string `json:"auth_id"`
	Code   string `json:"code"`
}

type TopUpRequest struct {
	UserID int64 `json:"user_id"`
	Amount int64 `json:"amount"`
}

type ConfirmTopUpRequest struct {
	TransactionID string `json:"transaction_id"`
	AdminID       int64  `json:"admin_id"`
}

type OTPVerifyLoginRequest struct {
	PhoneNumber string `json:"phone_number"`
	OTPCode     string `json:"otp_code"`
}

type PurchaseRequest struct {
	AccessToken   string `json:"access_token"`
	PackageCode   string `json:"package_code"`
	PaymentMethod string `json:"payment_method"`
	PhoneNumber   string `json:"phone_number"`
	Source        string `json:"source"`
}

type SearchRequest struct {
	Query         string `json:"query"`
	MinPrice      int64  `json:"min_price"`
	MaxPrice      int64  `json:"max_price"`
	PaymentMethod string `json:"payment_method"`
}
