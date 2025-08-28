package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Global bot instance for admin notifications
var BotInstance *tgbotapi.BotAPI

func GetBotToken() string {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN tidak ditemukan di environment variable")
	}
	return token
}

func GetAdminChatID() int64 {
	chatIDStr := os.Getenv("ADMIN_CHAT_ID")
	if chatIDStr == "" {
		log.Println("Warning: ADMIN_CHAT_ID tidak ditemukan di environment variable")
		return 0
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		log.Printf("Error parsing ADMIN_CHAT_ID: %v", err)
		return 0
	}

	return chatID
}

func GetAdminUsername() string {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		log.Println("Warning: ADMIN_USERNAME tidak ditemukan di environment variable")
		return ""
	}
	return strings.TrimPrefix(username, "@")
}

func IsAdmin(chatID int64) bool {
	adminChatID := GetAdminChatID()
	return adminChatID != 0 && chatID == adminChatID
}

func GetAdminTelegramID() int64 {
	return GetAdminChatID() // Same as admin chat ID
}

func GetAdminWhatsAppNumber() string {
	phone := os.Getenv("ADMIN_WHATSAPP")
	if phone == "" {
		log.Println("Warning: ADMIN_WHATSAPP tidak ditemukan di environment variable")
		return ""
	}
	return phone
}
