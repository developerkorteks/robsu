package service

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nabilulilalbab/bottele/config"
	"github.com/nabilulilalbab/bottele/dto"
	"github.com/nabilulilalbab/bottele/models"
)

// Reference to variables from topup_service
// These will be linked at runtime through import

// SendMessageToAdmin mengirim pesan ke admin
func SendMessageToAdmin(bot *tgbotapi.BotAPI, message string, fromUser *tgbotapi.User) error {
	adminChatID := config.GetAdminChatID()
	if adminChatID == 0 {
		return fmt.Errorf("admin chat ID tidak dikonfigurasi")
	}

	// Format pesan untuk admin
	adminMessage := fmt.Sprintf(`📩 *Pesan dari User*

👤 *User:* %s (@%s)
🆔 *User ID:* %d
🕐 *Waktu:* %s

💬 *Pesan:*
%s`,
		getUserDisplayName(fromUser),
		fromUser.UserName,
		fromUser.ID,
		time.Now().Format("02/01/2006 15:04:05"),
		message)

	msg := tgbotapi.NewMessage(adminChatID, adminMessage)
	msg.ParseMode = "Markdown"

	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending message to admin: %v", err)
		return err
	}

	return nil
}

// SendAdminNotification mengirim notifikasi ke admin
func SendAdminNotification(bot *tgbotapi.BotAPI, notification string) error {
	adminChatID := config.GetAdminChatID()
	if adminChatID == 0 {
		return fmt.Errorf("admin chat ID tidak dikonfigurasi")
	}

	msg := tgbotapi.NewMessage(adminChatID, notification)
	msg.ParseMode = "Markdown"

	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Error sending notification to admin: %v", err)
		return err
	}

	return nil
}

// BroadcastMessage mengirim pesan broadcast (hanya admin yang bisa)
func BroadcastMessage(bot *tgbotapi.BotAPI, message string, userIDs []int64) error {
	successCount := 0
	failCount := 0

	for _, userID := range userIDs {
		msg := tgbotapi.NewMessage(userID, message)
		msg.ParseMode = "Markdown"

		_, err := bot.Send(msg)
		if err != nil {
			log.Printf("Error sending broadcast to user %d: %v", userID, err)
			failCount++
		} else {
			successCount++
		}
	}

	// Kirim laporan ke admin
	report := fmt.Sprintf(`📊 *Laporan Broadcast*

✅ Berhasil: %d user
❌ Gagal: %d user
📊 Total: %d user`, successCount, failCount, len(userIDs))

	adminChatID := config.GetAdminChatID()
	if adminChatID != 0 {
		reportMsg := tgbotapi.NewMessage(adminChatID, report)
		reportMsg.ParseMode = "Markdown"
		bot.Send(reportMsg)
	}

	return nil
}

// GetAllUserIDs mendapatkan semua user ID yang pernah berinteraksi dengan bot
func GetAllUserIDs() []int64 {
	var userIDs []int64
	userMap := make(map[int64]bool)

	// Get from in-memory transactions
	TxMutex.RLock()
	for _, tx := range Transactions {
		if !userMap[tx.UserID] {
			userIDs = append(userIDs, tx.UserID)
			userMap[tx.UserID] = true
		}
	}
	TxMutex.RUnlock()

	// Also get from active users in database
	activeUserIDs := GetAllUserIDsFromData()
	for _, userID := range activeUserIDs {
		if !userMap[userID] {
			userIDs = append(userIDs, userID)
			userMap[userID] = true
		}
	}

	// Get users from database transactions and balances
	var dbUsers []models.User
	if err := config.DB.Find(&dbUsers).Error; err == nil {
		for _, user := range dbUsers {
			if !userMap[user.ChatID] {
				userIDs = append(userIDs, user.ChatID)
				userMap[user.ChatID] = true
			}
		}
	}

	return userIDs
}

// GetAllTransactions mendapatkan semua transaksi untuk statistik
func GetAllTransactions() []*dto.Transaction {
	var transactions []*dto.Transaction

	// Get all transactions from in-memory storage
	TxMutex.RLock()
	for _, tx := range Transactions {
		transactions = append(transactions, tx)
	}
	TxMutex.RUnlock()

	return transactions
}

// GetUserStats mendapatkan statistik user untuk admin
func GetUserStats() string {
	// Get transaction statistics
	allTransactions := GetAllTransactions()
	totalTransactions := len(allTransactions)
	confirmedCount := 0
	rejectedCount := 0
	pendingCount := 0
	expiredCount := 0
	totalRevenue := int64(0)

	for _, tx := range allTransactions {
		switch tx.Status {
		case "confirmed":
			confirmedCount++
			totalRevenue += tx.Amount
		case "rejected":
			rejectedCount++
		case "pending":
			pendingCount++
		case "expired":
			expiredCount++
		}
	}

	// Get user count from transactions
	userIDs := GetAllUserIDs()
	totalUsers := len(userIDs)

	return fmt.Sprintf(`📊 *Statistik Bot GRN Store*

👥 *User Statistics:*
• Total User: %d
• User Aktif: %d

💰 *Transaction Statistics:*
• Total Transaksi: %d
• ✅ Confirmed: %d
• ⏳ Pending: %d
• ❌ Rejected: %d
• ⏰ Expired: %d

💵 *Revenue Statistics:*
• Total Revenue: %s
• Rata-rata per Transaksi: %s

📈 *Status:* Real-time data`,
		totalUsers,
		totalUsers,
		totalTransactions,
		confirmedCount,
		pendingCount,
		rejectedCount,
		expiredCount,
		formatPrice(totalRevenue),
		formatPrice(getAverageTransaction(totalRevenue, confirmedCount)))
}

func getAverageTransaction(total int64, count int) int64 {
	if count == 0 {
		return 0
	}
	return total / int64(count)
}

func formatPrice(price int64) string {
	if price == 0 {
		return "Rp 0"
	}

	// Convert to string and add thousand separators
	priceStr := fmt.Sprintf("%d", price)
	result := "Rp "

	// Add dots every 3 digits from right
	for i, digit := range priceStr {
		if i > 0 && (len(priceStr)-i)%3 == 0 {
			result += "."
		}
		result += string(digit)
	}

	return result
}

func getUserDisplayName(user *tgbotapi.User) string {
	if user.FirstName != "" && user.LastName != "" {
		return user.FirstName + " " + user.LastName
	} else if user.FirstName != "" {
		return user.FirstName
	} else if user.UserName != "" {
		return "@" + user.UserName
	}
	return "Unknown User"
}
