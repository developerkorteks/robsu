package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const whatsappAPIURL = "http://128.199.109.211:25120/send-message"

// SendWhatsAppMessage sends a message via WhatsApp API
func SendWhatsAppMessage(phoneNumber, message string) error {
	payload := map[string]string{
		"number":  phoneNumber,
		"message": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling WhatsApp payload: %v", err)
	}

	req, err := http.NewRequest("POST", whatsappAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating WhatsApp request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending WhatsApp message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("WhatsApp API returned status %d", resp.StatusCode)
	}

	log.Printf("WhatsApp message sent successfully to %s", phoneNumber)
	return nil
}
