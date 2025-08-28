package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nabilulilalbab/bottele/dto"
)

const (
	searchURL = "https://grnstore.domcloud.dev/api/user/products/search"
)

// SearchProducts searches for products based on criteria
func SearchProducts(query string, minPrice, maxPrice int64, paymentMethod string) (*dto.ApiResponse, error) {
	searchReq := dto.SearchRequest{
		Query:         query,
		MinPrice:      minPrice,
		MaxPrice:      maxPrice,
		PaymentMethod: paymentMethod,
	}

	jsonData, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("gagal marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(jsonData))
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

	var searchResp dto.ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("gagal decode response: %v", err)
	}

	if !searchResp.Success {
		return nil, fmt.Errorf("pencarian gagal: %s", searchResp.Message)
	}

	return &searchResp, nil
}