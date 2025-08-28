package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nabilulilalbab/bottele/dto"
)

const (
	otpRequestURL = "https://grnstore.domcloud.dev/api/otp/request"
	otpVerifyURL  = "https://grnstore.domcloud.dev/api/otp/verify"
	apiKey        = "nadia-admin-2024-secure-key"
)

func RequestOTP(phoneNumber string) (*dto.OTPResponse, error) {
	otpReq := dto.OTPRequest{
		PhoneNumber: phoneNumber,
	}

	jsonData, err := json.Marshal(otpReq)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", otpRequestURL, bytes.NewBuffer(jsonData))
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

	var otpResp dto.OTPResponse
	if err := json.NewDecoder(resp.Body).Decode(&otpResp); err != nil {
		return nil, fmt.Errorf("gagal decode response: %v", err)
	}

	if !otpResp.Success {
		return nil, fmt.Errorf("gagal request OTP: %s", otpResp.Message)
	}

	return &otpResp, nil
}

func VerifyOTP(authID, code string) (*dto.OTPVerifyResponse, error) {
	verifyReq := dto.OTPVerifyRequest{
		AuthID: authID,
		Code:   code,
	}

	jsonData, err := json.Marshal(verifyReq)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", otpVerifyURL, bytes.NewBuffer(jsonData))
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

	var verifyResp dto.OTPVerifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return nil, fmt.Errorf("gagal decode response: %v", err)
	}

	return &verifyResp, nil
}