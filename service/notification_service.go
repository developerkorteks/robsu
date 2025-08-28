package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nabilulilalbab/bottele/config"
)

var (
	notificationMutex sync.Mutex
	lastNotification  = make(map[string]time.Time)
)

// NotifyAdminError sends error notification to admin via Telegram
func NotifyAdminError(userID int64, operation, details string) {
	notificationMutex.Lock()
	defer notificationMutex.Unlock()

	// Prevent spam - only send same error once per minute
	key := fmt.Sprintf("error_%d_%s", userID, operation)
	if lastTime, exists := lastNotification[key]; exists {
		if time.Since(lastTime) < time.Minute {
			return
		}
	}
	lastNotification[key] = time.Now()

	message := fmt.Sprintf(
		"🚨 *SYSTEM ERROR ALERT*\n\n"+
			"⏰ Time: %s\n"+
			"👤 User ID: %d\n"+
			"🔧 Operation: %s\n"+
			"❌ Error: %s\n\n"+
			"🔍 Action Required: Please investigate immediately.",
		time.Now().Format("2006-01-02 15:04:05"),
		userID,
		operation,
		details,
	)

	sendToTelegramAdmin(message)
}

// NotifyAdminApprovalNeeded sends approval notification to admin
func NotifyAdminApprovalNeeded(userID int64, operation, details string) {
	notificationMutex.Lock()
	defer notificationMutex.Unlock()

	// Prevent spam - only send same approval once per 5 minutes
	key := fmt.Sprintf("approval_%d_%s", userID, operation)
	if lastTime, exists := lastNotification[key]; exists {
		if time.Since(lastTime) < 5*time.Minute {
			return
		}
	}
	lastNotification[key] = time.Now()

	message := fmt.Sprintf(
		"⚠️ *APPROVAL REQUIRED*\n\n"+
			"⏰ Time: %s\n"+
			"👤 User ID: %d\n"+
			"🔧 Operation: %s\n"+
			"📝 Details: %s\n\n"+
			"✅ Please review and approve if necessary.",
		time.Now().Format("2006-01-02 15:04:05"),
		userID,
		operation,
		details,
	)

	// Send to both Telegram and WhatsApp for approval
	sendToTelegramAdmin(message)
	sendToWhatsAppAdmin(message)
}

// NotifyAdminTopupApproval sends topup approval notification
func NotifyAdminTopupApproval(userID int64, amount int64, method string) {
	message := fmt.Sprintf(
		"💰 *TOPUP APPROVAL NEEDED*\n\n"+
			"⏰ Time: %s\n"+
			"👤 User ID: %d\n"+
			"💵 Amount: Rp %s\n"+
			"💳 Method: %s\n\n"+
			"✅ Please approve this topup request.",
		time.Now().Format("2006-01-02 15:04:05"),
		userID,
		formatRupiah(amount),
		method,
	)

	// Send to both Telegram and WhatsApp for topup approval
	sendToTelegramAdmin(message)
	sendToWhatsAppAdmin(message)
}

// sendToTelegramAdmin sends message to admin via Telegram
func sendToTelegramAdmin(message string) {
	if config.BotInstance == nil {
		log.Printf("Bot instance not available for admin notification")
		return
	}

	adminID := config.GetAdminTelegramID()
	if adminID == 0 {
		log.Printf("Admin Telegram ID not configured")
		return
	}

	msg := tgbotapi.NewMessage(adminID, message)
	msg.ParseMode = "Markdown"

	if _, err := config.BotInstance.Send(msg); err != nil {
		log.Printf("Failed to send admin notification to Telegram: %v", err)
	}
}

// sendToWhatsAppAdmin sends message to admin via WhatsApp (only for approvals)
func sendToWhatsAppAdmin(message string) {
	adminPhone := config.GetAdminWhatsAppNumber()
	if adminPhone == "" {
		log.Printf("Admin WhatsApp number not configured")
		return
	}

	// Convert markdown to plain text for WhatsApp
	plainMessage := convertMarkdownToPlain(message)

	if err := SendWhatsAppMessage(adminPhone, plainMessage); err != nil {
		log.Printf("Failed to send admin notification to WhatsApp: %v", err)
	}
}

// convertMarkdownToPlain converts markdown formatting to plain text
func convertMarkdownToPlain(text string) string {
	// Simple conversion - remove markdown formatting
	result := text
	result = fmt.Sprintf("> grnstore: %s", result)
	return result
}

// formatRupiah formats number to Rupiah currency
func formatRupiah(amount int64) string {
	if amount < 1000 {
		return fmt.Sprintf("%d", amount)
	}

	str := fmt.Sprintf("%d", amount)
	n := len(str)
	result := ""

	for i, char := range str {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(char)
	}

	return result
}

// NotifyUserTopupSuccess sends notification to user when topup is approved
func NotifyUserTopupSuccess(userID int64, amount int64, transactionID string) {
	if config.BotInstance == nil {
		log.Printf("Bot instance not available for user notification")
		return
	}

	balance := GetUserBalance(userID)
	text := fmt.Sprintf(`✅ *Top-Up Berhasil!*

💰 *Nominal:* %s
🆔 *Transaction ID:* `+"`%s`"+`
💳 *Saldo Terkini:* %s

Terima kasih telah menggunakan layanan kami! 🙏`,
		formatRupiah(amount),
		transactionID,
		formatRupiah(balance.Balance))

	msg := tgbotapi.NewMessage(userID, text)
	msg.ParseMode = "Markdown"

	if _, err := config.BotInstance.Send(msg); err != nil {
		log.Printf("Failed to notify user %d about topup success: %v", userID, err)
	}
}
