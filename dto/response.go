package dto

import (
	"encoding/json"
	"fmt"
)

type ApiResponse struct {
	StatusCode int       `json:"statusCode"`
	Message    string    `json:"message"`
	Success    bool      `json:"success"`
	Data       []Package `json:"data"`
}

type Package struct {
	PackageCode             string            `json:"package_code"`
	PackageName             string            `json:"package_name"`
	PackageNameAliasShort   string            `json:"package_name_alias_short"`
	PackageDescription      string            `json:"package_description"`
	Price                   int64             `json:"package_harga_int"`
	PriceFormatted          string            `json:"package_harga"`
	HaveDailyLimit          bool              `json:"have_daily_limit"`
	DailyLimitDetails       DailyLimitDetails `json:"daily_limit_details"`
	NoNeedLogin             bool              `json:"no_need_login"`
	CanMultiTrx             bool              `json:"can_multi_trx"`
	CanScheduledTrx         bool              `json:"can_scheduled_trx"`
	HaveCutOffTime          bool              `json:"have_cut_off_time"`
	CutOffTime              CutOffTime        `json:"cut_off_time"`
	NeedCheckStock          bool              `json:"need_check_stock"`
	IsShowPaymentMethod     bool              `json:"is_show_payment_method"`
	AvailablePaymentMethods []PaymentMethod   `json:"available_payment_methods"`
}

type DailyLimitDetails struct {
	MaxDailyTransactionLimit     int `json:"max_daily_transaction_limit"`
	CurrentDailyTransactionCount int `json:"current_daily_transaction_count"`
}

type CutOffTime struct {
	ProhibitedHourStarttime string `json:"prohibited_hour_starttime"`
	ProhibitedHourEndtime   string `json:"prohibited_hour_endtime"`
}

type PaymentMethod struct {
	Order                    int    `json:"order"`
	PaymentMethod            string `json:"payment_method"`
	PaymentMethodDisplayName string `json:"payment_method_display_name"`
	Desc                     string `json:"desc"`
}

type OTPResponse struct {
	StatusCode int     `json:"statusCode"`
	Message    string  `json:"message"`
	Success    bool    `json:"success"`
	Data       OTPData `json:"data"`
}

type OTPData struct {
	AuthID      string `json:"auth_id"`
	CanResendIn int    `json:"can_resend_in"`
}

type OTPVerifyResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Success    bool   `json:"success"`
}

type TopUpResponse struct {
	StatusCode int       `json:"statusCode"`
	Message    string    `json:"message"`
	Success    bool      `json:"success"`
	Data       TopUpData `json:"data"`
}

type TopUpData struct {
	TransactionID string `json:"transaction_id"`
	QRISCode      string `json:"qris_code"`
	Amount        int64  `json:"amount"`
	ExpiredAt     string `json:"expired_at"`
}

type Transaction struct {
	ID         string `json:"id"`
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Amount     int64  `json:"amount"`
	Status     string `json:"status"` // pending, confirmed, rejected, expired
	QRISCode   string `json:"qris_code"`
	CreatedAt  string `json:"created_at"`
	ApprovedBy int64  `json:"approved_by,omitempty"`
	ApprovedAt string `json:"approved_at,omitempty"`
	ExpiredAt  string `json:"expired_at"`
}

type UserBalance struct {
	UserID  int64 `json:"user_id"`
	Balance int64 `json:"balance"`
}

type BalanceResponse struct {
	StatusCode int         `json:"statusCode"`
	Message    string      `json:"message"`
	Success    bool        `json:"success"`
	Data       UserBalance `json:"data"`
}

type OTPVerifyLoginResponse struct {
	StatusCode int                `json:"statusCode"`
	Message    string             `json:"message"`
	Success    bool               `json:"success"`
	Data       OTPVerifyLoginData `json:"data"`
}

type OTPVerifyLoginData struct {
	AccessToken string `json:"access_token"`
}

type PurchaseResponse struct {
	StatusCode int          `json:"statusCode"`
	Message    string       `json:"message"`
	Success    bool         `json:"success"`
	Data       PurchaseData `json:"data"` // Handle via custom unmarshaling
}

type PurchaseData struct {
	DeeplinkData         DeeplinkData    `json:"deeplink_data"`
	HaveDeeplink         bool            `json:"have_deeplink"`
	IsQRIS               bool            `json:"is_qris"`
	MSISDN               string          `json:"msisdn"`
	PackageCode          string          `json:"package_code"`
	PackageName          string          `json:"package_name"`
	PackageProcessingFee int64           `json:"package_processing_fee"`
	Price                int64           `json:"price"`
	QRISData             json.RawMessage `json:"qris_data"`
	TrxID                string          `json:"trx_id"`
}

type DeeplinkData struct {
	DeeplinkURL   string `json:"deeplink_url"`
	PaymentMethod string `json:"payment_method"`
}

type QRISData struct {
	PaymentExpiredAt int64  `json:"payment_expired_at"`
	QRCode           string `json:"qr_code"`
	RemainingTime    int64  `json:"remaining_time"`
}

// GetQRISData safely extracts QRIS data from the raw JSON
func (pd *PurchaseData) GetQRISData() QRISData {
	var qrisData QRISData
	
	// Try to unmarshal as object first
	if err := json.Unmarshal(pd.QRISData, &qrisData); err == nil {
		return qrisData
	}
	
	// Try to unmarshal as array
	var qrisArray []QRISData
	if err := json.Unmarshal(pd.QRISData, &qrisArray); err == nil && len(qrisArray) > 0 {
		return qrisArray[0]
	}
	
	// Return empty struct if both fail
	return QRISData{}
}

// Custom unmarshaling for PurchaseResponse to handle array data
func (pr *PurchaseResponse) UnmarshalJSON(data []byte) error {
	// First, unmarshal the basic fields without the Data field
	type Alias struct {
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
		Success    bool   `json:"success"`
	}
	
	aux := &struct {
		Data json.RawMessage `json:"data"`
		*Alias
	}{
		Alias: &Alias{},
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set the basic fields
	pr.StatusCode = aux.StatusCode
	pr.Message = aux.Message
	pr.Success = aux.Success

	// Try to unmarshal data as array first
	var purchaseDataArray []PurchaseData
	if err := json.Unmarshal(aux.Data, &purchaseDataArray); err == nil {
		if len(purchaseDataArray) > 0 {
			pr.Data = purchaseDataArray[0]
		} else {
			// Empty array case - set empty data
			pr.Data = PurchaseData{}
		}
		return nil
	}

	// If that fails, try as single object
	var purchaseData PurchaseData
	if err := json.Unmarshal(aux.Data, &purchaseData); err == nil {
		pr.Data = purchaseData
		return nil
	}

	return fmt.Errorf("cannot unmarshal data field as either array or object")
}

type TransactionCheckResponse struct {
	StatusCode int                  `json:"statusCode"`
	Message    string               `json:"message"`
	Success    bool                 `json:"success"`
	Data       TransactionCheckData `json:"data"`
}

type TransactionCheckData struct {
	AdminFee                 int64    `json:"admin_fee"`
	BalanceAfterTransaction  int64    `json:"balance_after_transaction"`
	BalanceBeforeTransaction int64    `json:"balance_before_transaction"`
	Channel                  string   `json:"channel"`
	ChannelTransactionCode   string   `json:"channel_transaction_code"`
	Code                     string   `json:"code"`
	DeeplinkURL              string   `json:"deeplink_url"`
	DestinationMSISDN        string   `json:"destination_msisdn"`
	HasParsialRefund         bool     `json:"has_parsial_refund"`
	HaveDeeplink             bool     `json:"have_deeplink"`
	IsQRIS                   bool     `json:"is_qris"`
	IsRefunded               int      `json:"is_refunded"`
	Name                     string   `json:"name"`
	Price                    int64    `json:"price"`
	QRISData                 QRISData `json:"qris_data"`
	RC                       string   `json:"rc"`
	RCMessage                string   `json:"rc_message"`
	RefundAmount             int64    `json:"refund_amount"`
	RefundReason             string   `json:"refund_reason"`
	SNAndInfo                string   `json:"sn_and_info"`
	SNOnly                   string   `json:"sn_only"`
	Status                   int      `json:"status"`
	TimeDate                 string   `json:"time_date"`
	TotalPrice               int64    `json:"total_price"`
	TrxID                    string   `json:"trx_id"`
}
