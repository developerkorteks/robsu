package service

import (
	"encoding/json"
	"net/http"

	"github.com/nabilulilalbab/bottele/dto"
)

// di service/package_service.go
type PackageAlias struct {
	Name  string
	Price int64
}

const apiURL = "https://grnstore.domcloud.dev/api/user/products?limit=100"

func FetchPackages() ([]dto.Package, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("accept", "application/json")
	req.Header.Set("X-API-Key", "nadia-admin-2024-secure-key")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp dto.ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}
