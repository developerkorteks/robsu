package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nabilulilalbab/bottele/config"
	"github.com/nabilulilalbab/bottele/dto"
	"github.com/nabilulilalbab/bottele/models"
	"github.com/nabilulilalbab/bottele/service"
)

// Import dari topup_service untuk akses ke transactions map

const pageSize = 10

var packageMap = map[string]service.PackageAlias{} // alias untuk caching

func HandleUpdate(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered: %v", r)
		}
	}()

	// Track user interaction
	var userID int64
	if update.Message != nil {
		userID = update.Message.Chat.ID
		handleMessage(bot, update.Message)
	}

	if update.CallbackQuery != nil {
		userID = update.CallbackQuery.Message.Chat.ID
		handleCallbackQuery(bot, update.CallbackQuery)
	}

	// Add user to active users list in database
	if userID != 0 {
		err := service.AddActiveUserToDB(userID)
		if err != nil {
			log.Printf("Error adding active user %d: %v", userID, err)
		}
	}
}

func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	userState := getUserState(chatID)

	if message.IsCommand() {
		switch message.Command() {
		case "start":
			handleStart(bot, chatID)
		case "menu":
			showMainMenu(bot, chatID)
		case "products":
			sendProductList(bot, chatID, 0)
		case "help":
			showHelp(bot, chatID)
		case "rules":
			sendRulesMessage(bot, chatID)
		case "admin":
			handleAdminCommand(bot, message)
		case "stats":
			handleStatsCommand(bot, chatID)
		case "pending":
			if !config.IsAdmin(chatID) {
				sendErrorMessage(bot, chatID, "❌ Perintah tidak dikenal. Ketik /menu untuk melihat menu utama.")
				return
			}
			handlePendingCommand(bot, message)
		case "confirm":
			if !config.IsAdmin(chatID) {
				sendErrorMessage(bot, chatID, "❌ Perintah tidak dikenal. Ketik /menu untuk melihat menu utama.")
				return
			}
			handleConfirmCommand(bot, message)
		case "debug":
			if !config.IsAdmin(chatID) {
				sendErrorMessage(bot, chatID, "❌ Perintah tidak dikenal. Ketik /menu untuk melihat menu utama.")
				return
			}
			handleDebugCommand(bot, message)
		case "reject":
			if !config.IsAdmin(chatID) {
				sendErrorMessage(bot, chatID, "❌ Perintah tidak dikenal. Ketik /menu untuk melihat menu utama.")
				return
			}
			handleRejectCommand(bot, message)
		case "balance":
			handleBalanceCommand(bot, chatID)
		case "search":
			handleSearchCommand(bot, message)
		case "history":
			handleHistoryCommand(bot, chatID)
		case "broadcast":
			if !config.IsAdmin(chatID) {
				sendErrorMessage(bot, chatID, "❌ Perintah tidak dikenal. Ketik /menu untuk melihat menu utama.")
				return
			}
			handleBroadcastCommand(bot, message)
		default:
			sendErrorMessage(bot, chatID, "❌ Perintah tidak dikenal. Ketik /menu untuk melihat menu utama.")
		}
		return
	}

	// Handle text messages based on user state
	userState.mu.RLock()
	state := userState.State
	userState.mu.RUnlock()

	switch state {
	case "waiting_phone":
		handlePhoneInput(bot, chatID, message.Text)
	case "waiting_otp":
		handleOTPInput(bot, chatID, message.Text)
	case "waiting_admin_message":
		handleAdminMessageInput(bot, chatID, message.Text, message.From)
	case "waiting_topup_amount":
		handleTopUpAmountInput(bot, chatID, message.Text, message.From)
	case "waiting_broadcast_message":
		handleBroadcastMessageInput(bot, chatID, message.Text)
	case "waiting_search_query":
		handleSearchQueryInput(bot, chatID, message.Text)
	case "waiting_vpn_email":
		handleVPNEmailInput(bot, chatID, message.Text)
	case "waiting_vpn_password":
		handleVPNPasswordInput(bot, chatID, message.Text)
	case "waiting_vpn_days":
		handleVPNDaysInput(bot, chatID, message.Text)
	case "waiting_vpn_extend_days":
		handleVPNExtendDaysInput(bot, chatID, message.Text)
	default:
		showMainMenu(bot, chatID)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, cq *tgbotapi.CallbackQuery) {
	data := cq.Data
	chatID := cq.Message.Chat.ID

	// Answer callback query to remove loading state
	bot.Request(tgbotapi.NewCallback(cq.ID, ""))

	if strings.HasPrefix(data, "buy:") {
		productCode := strings.TrimPrefix(data, "buy:")
		handleBuyProduct(bot, chatID, productCode)
	} else if strings.HasPrefix(data, "detail:") {
		productCode := strings.TrimPrefix(data, "detail:")
		handleProductDetail(bot, chatID, productCode)
	} else if strings.HasPrefix(data, "page:") {
		pageStr := strings.TrimPrefix(data, "page:")
		page, _ := strconv.Atoi(pageStr)
		editProductList(bot, cq.Message, page)
	} else if data == "verify_phone" {
		handleVerifyPhone(bot, chatID)
	} else if data == "main_menu" {
		showMainMenu(bot, chatID)
	} else if data == "products" {
		sendProductList(bot, chatID, 0)
	} else if data == "help" {
		showHelp(bot, chatID)
	} else if data == "balance" {
		handleBalanceCommand(bot, chatID)
	} else if data == "rules" {
		sendRulesMessage(bot, chatID)
	} else if data == "history" {
		handleHistoryCommandNew(bot, chatID)
	} else if data == "contact_admin" {
		handleContactAdmin(bot, chatID)
	} else if data == "proceed_payment" {
		// Debug: Check state before calling handleProceedPayment
		debugState := getUserState(chatID)
		debugState.mu.RLock()
		log.Printf("DEBUG CALLBACK proceed_payment - User %d: phone='%s', productCode='%s'",
			chatID, debugState.PhoneNumber, debugState.ProductCode)
		debugState.mu.RUnlock()

		handleProceedPayment(bot, chatID)
	} else if data == "admin_stats" {
		handleStatsCommand(bot, chatID)
	} else if data == "admin_panel" {
		handleAdminCommand(bot, &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}})
	} else if data == "topup" {
		handleTopUpRequest(bot, chatID)
	} else if data == "check_balance" {
		handleBalanceCommand(bot, chatID)
	} else if data == "admin_pending" {
		handlePendingCommand(bot, &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: chatID}})
	} else if data == "admin_broadcast" {
		handleBroadcastRequest(bot, chatID)
	} else if strings.HasPrefix(data, "send_broadcast:") {
		broadcastMessage := strings.TrimPrefix(data, "send_broadcast:")
		handleSendBroadcast(bot, chatID, broadcastMessage)
	} else if strings.HasPrefix(data, "topup:") {
		amountStr := strings.TrimPrefix(data, "topup:")
		if amountStr == "custom" {
			setUserState(chatID, "waiting_topup_amount")
			text := `╔══════════════════════════╗
║     💳 *TOP UP SALDO*    ║
╚══════════════════════════╝

💰 *Masukkan Nominal Custom*

Silakan ketik nominal top up yang Anda inginkan.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 *KETENTUAN:*
• 💵 Minimum: Rp 10.000
• 💎 Maximum: Rp 1.000.000
• ⚠️ Hanya angka (tanpa titik/koma)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 *CONTOH INPUT:*
• Untuk Rp 50.000 → ketik: *50000*
• Untuk Rp 100.000 → ketik: *100000*
• Untuk Rp 250.000 → ketik: *250000*

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚡ *Pembayaran via QRIS - Aman & Cepat*

🔤 *Ketik nominal sekarang:*`

			msg := tgbotapi.NewMessage(chatID, text)
			msg.ParseMode = "Markdown"
			if _, err := bot.Send(msg); err != nil {
				log.Printf("Error sending custom topup message: %v", err)
			}
		} else {
			_, err := strconv.ParseInt(amountStr, 10, 64)
			if err != nil {
				sendErrorMessage(bot, chatID, "❌ Nominal tidak valid.")
				return
			}
			handleTopUpAmountInput(bot, chatID, amountStr, &tgbotapi.User{ID: chatID})
		}
	} else if data == "logout" {
		handleLogout(bot, chatID)
	} else if strings.HasPrefix(data, "pay:") {
		parts := strings.Split(strings.TrimPrefix(data, "pay:"), ":")
		if len(parts) == 2 {
			handlePayment(bot, chatID, parts[0], parts[1])
		}
	} else if data == "search_products" {
		handleSearchRequest(bot, chatID)
	} else if data == "history" {
		handleHistoryCommand(bot, chatID)
	} else if strings.HasPrefix(data, "check:") {
		transactionID := strings.TrimPrefix(data, "check:")
		handleCheckTransaction(bot, chatID, transactionID)
	} else if strings.HasPrefix(data, "search_page:") {
		parts := strings.Split(strings.TrimPrefix(data, "search_page:"), ":")
		if len(parts) == 2 {
			query := parts[0]
			page, _ := strconv.Atoi(parts[1])
			// Re-search and display page
			searchResp, err := service.SearchProducts(query, 0, 1000000, "")
			if err == nil {
				displaySearchResults(bot, chatID, query, searchResp.Data, page)
			}
		}
	} else if strings.HasPrefix(data, "history_page:") {
		pageStr := strings.TrimPrefix(data, "history_page:")
		page, _ := strconv.Atoi(pageStr)
		if service.IsUserLoggedIn(chatID) {
			history, err := service.GetUserPurchaseHistory(chatID)
			if err == nil {
				displayPurchaseHistory(bot, chatID, history, page)
			}
		}
	} else if strings.HasPrefix(data, "history_detail:") {
		transactionID := strings.TrimPrefix(data, "history_detail:")
		handleTransactionDetail(bot, chatID, transactionID)
	} else if strings.HasPrefix(data, "approve_tx:") {
		transactionID := strings.TrimPrefix(data, "approve_tx:")
		handleApproveTransaction(bot, chatID, transactionID)
	} else if strings.HasPrefix(data, "reject_tx:") {
		transactionID := strings.TrimPrefix(data, "reject_tx:")
		handleRejectTransaction(bot, chatID, transactionID)
	} else if data == "vpn_menu" {
		handleVPNMenu(bot, chatID)
	} else if strings.HasPrefix(data, "vpn_create:") {
		protocol := strings.TrimPrefix(data, "vpn_create:")
		handleVPNCreateStart(bot, chatID, protocol)
	} else if data == "vpn_list" {
		handleVPNList(bot, chatID)
	} else if data == "vpn_history" {
		handleVPNHistory(bot, chatID)
	} else if strings.HasPrefix(data, "vpn_extend:") {
		vpnUsername := strings.TrimPrefix(data, "vpn_extend:")
		handleVPNExtendStart(bot, chatID, vpnUsername)
	} else if strings.HasPrefix(data, "vpn_detail:") {
		vpnUsername := strings.TrimPrefix(data, "vpn_detail:")
		handleVPNDetail(bot, chatID, vpnUsername)
	} else if strings.HasPrefix(data, "vpn_days:") {
		daysStr := strings.TrimPrefix(data, "vpn_days:")
		handleVPNDaysInput(bot, chatID, daysStr)
	} else if strings.HasPrefix(data, "vpn_confirm:") {
		daysStr := strings.TrimPrefix(data, "vpn_confirm:")
		handleVPNConfirm(bot, chatID, daysStr)
	} else if strings.HasPrefix(data, "vpn_extend_days:") {
		daysStr := strings.TrimPrefix(data, "vpn_extend_days:")
		handleVPNExtendDaysInput(bot, chatID, daysStr)
	}
}

func handleStart(bot *tgbotapi.BotAPI, chatID int64) {
	clearUserState(chatID)

	text := "```\n" +
		"╔══════════════════════════╗\n" +
		"║       🌟 GRN STORE 🌟      ║\n" +
		"║   Premium Digital Store   ║\n" +
		"╚══════════════════════════╝\n\n" +
		"🎯 SELAMAT DATANG!\n" +
		"Terima kasih telah memilih GRN Store!\n\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"🛍️ LAYANAN KAMI:\n" +
		"• 📶 Jual Kuota Internet All Operator\n" +
		"• 💳 Top Up Saldo\n" +
		"• 🌐 VPN Premium (SSHWS, Trojan, Vmess, Vless)\n" +
		"   ➝ Rp8.000 / bulan\n" +
		"   ➝ Server SG tersedia\n\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"📋 ALUR PEMBELIAN KUOTA:\n" +
		"1️⃣ Top Up saldo di bot\n" +
		"2️⃣ Pilih paket kuota yang ingin dibeli\n" +
		"3️⃣ Lakukan Verifikasi OTP (wajib)\n" +
		"4️⃣ Lanjutkan pembayaran\n" +
		"   (beberapa paket ada tambahan via DANA/QRIS – baca deskripsi)\n" +
		"5️⃣ Kuota akan diproses otomatis\n\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"⚠️ PERATURAN:\n" +
		"- 🚫 Tidak boleh spam & neko²\n" +
		"- ❗ Jika ada error segera lapor admin\n" +
		"- ⏳ Jika bot lemot, mohon sabar (mungkin sedang banyak pengguna)\n\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"💬 Grup VPN Server SG:\n" +
		"👉 https://chat.whatsapp.com/IeIXOndIoFr0apnlKzghUC\n" +
		"```"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛍️ Mulai Belanja", "main_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Cek Saldo", "balance"),
			tgbotapi.NewInlineKeyboardButtonData("💳 Top Up Saldo", "topup"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📜 Riwayat", "history"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Peraturan", "rules"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Bantuan", "help"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending start message: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

func showMainMenu(bot *tgbotapi.BotAPI, chatID int64) {
	text := `🏪 *GRN Store - Menu Utama*

Pilih layanan yang Anda butuhkan:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Lihat Produk", "products"),
			tgbotapi.NewInlineKeyboardButtonData("🔍 Cari Produk", "search_products"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Verifikasi Nomor", "verify_phone"),
			tgbotapi.NewInlineKeyboardButtonData("📋 History", "history"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Top Up Saldo", "topup"),
			tgbotapi.NewInlineKeyboardButtonData("💳 Cek Saldo", "check_balance"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔐 VPN Premium", "vpn_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Bantuan", "help"),
			tgbotapi.NewInlineKeyboardButtonData("👨‍💼 Hubungi Admin", "contact_admin"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending main menu: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

func showHelp(bot *tgbotapi.BotAPI, chatID int64) {
	text := `ℹ️ *Bantuan - GRN Store*

*Cara Berbelanja:*
1️⃣ Verifikasi nomor HP Anda terlebih dahulu
2️⃣ Pilih produk paket data yang diinginkan
3️⃣ Lakukan pembayaran sesuai instruksi
4️⃣ Paket data akan otomatis masuk ke nomor Anda

*Perintah Bot:*
• /start - Kembali ke menu utama
• /menu - Tampilkan menu utama
• /products - Lihat daftar produk
• /help - Bantuan

*Dukungan Pelanggan:*
Jika mengalami kendala, silakan hubungi admin kami.

*Jam Operasional:*
🕐 24 jam setiap hari`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨‍💼 Hubungi Admin", "contact_admin"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali ke Menu", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending help: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

func sendErrorMessage(bot *tgbotapi.BotAPI, chatID int64, message string) {
	msg := tgbotapi.NewMessage(chatID, message)
	bot.Send(msg)
}

func sendProductList(bot *tgbotapi.BotAPI, chatID int64, page int) {
	packages, err := service.FetchPackages()
	if err != nil {
		log.Printf("Error fetching packages: %v", err)
		sendErrorMessage(bot, chatID, "❌ Maaf, gagal memuat daftar produk. Silakan coba lagi nanti.")
		return
	}

	if len(packages) == 0 {
		sendErrorMessage(bot, chatID, "📭 Maaf, saat ini tidak ada produk yang tersedia.")
		return
	}

	total := len(packages)
	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	text := fmt.Sprintf(`📱 *Daftar Paket Data GRN Store*

Halaman %d dari %d | Total: %d produk

Pilih paket data yang Anda inginkan:`, page+1, (total+pageSize-1)/pageSize, total)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range packages[start:end] {
		packageMap[p.PackageCode] = service.PackageAlias{Name: p.PackageName, Price: p.Price}

		// Format harga dengan pemisah ribuan
		priceStr := formatPrice(p.Price)

		// Use short name if available, otherwise use full name
		displayName := p.PackageNameAliasShort
		if displayName == "" {
			displayName = p.PackageName
		}

		// Truncate long names for button display
		if len(displayName) > 50 {
			displayName = displayName[:47] + "..."
		}

		btnText := fmt.Sprintf("📦 %s - %s", displayName, priceStr)

		// Create row with product button and detail button
		productBtn := tgbotapi.NewInlineKeyboardButtonData(btnText, "detail:"+p.PackageCode)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(productBtn))
	}

	// Navigation buttons
	var navButtons []tgbotapi.InlineKeyboardButton
	if start > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Sebelumnya", fmt.Sprintf("page:%d", page-1)))
	}
	if end < total {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Selanjutnya ➡️", fmt.Sprintf("page:%d", page+1)))
	}
	if len(navButtons) > 0 {
		rows = append(rows, navButtons)
	}

	// Back to main menu button
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
	))

	kb := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = kb

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending product list: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan saat menampilkan produk.")
	}
}

func editProductList(bot *tgbotapi.BotAPI, message *tgbotapi.Message, page int) {
	packages, err := service.FetchPackages()
	if err != nil {
		log.Printf("Error fetching packages: %v", err)
		return
	}

	total := len(packages)
	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	text := fmt.Sprintf(`📱 *Daftar Paket Data GRN Store*

Halaman %d dari %d | Total: %d produk

Pilih paket data yang Anda inginkan:`, page+1, (total+pageSize-1)/pageSize, total)

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range packages[start:end] {
		packageMap[p.PackageCode] = service.PackageAlias{Name: p.PackageName, Price: p.Price}

		priceStr := formatPrice(p.Price)

		// Use short name if available, otherwise use full name
		displayName := p.PackageNameAliasShort
		if displayName == "" {
			displayName = p.PackageName
		}

		// Truncate long names for button display
		if len(displayName) > 50 {
			displayName = displayName[:47] + "..."
		}

		btnText := fmt.Sprintf("📦 %s - %s", displayName, priceStr)

		// Create row with product button
		productBtn := tgbotapi.NewInlineKeyboardButtonData(btnText, "detail:"+p.PackageCode)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(productBtn))
	}

	var navButtons []tgbotapi.InlineKeyboardButton
	if start > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Sebelumnya", fmt.Sprintf("page:%d", page-1)))
	}
	if end < total {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Selanjutnya ➡️", fmt.Sprintf("page:%d", page+1)))
	}
	if len(navButtons) > 0 {
		rows = append(rows, navButtons)
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
	))

	kb := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	editMsg := tgbotapi.NewEditMessageText(message.Chat.ID, message.MessageID, text)
	editMsg.ParseMode = "Markdown"
	editMsg.ReplyMarkup = &kb

	if _, err := bot.Send(editMsg); err != nil {
		log.Printf("Error editing product list: %v", err)
	}
}

func formatPrice(price int64) string {
	priceStr := fmt.Sprintf("Rp %d", price)
	// Simple thousand separator
	if price >= 1000 {
		priceStr = fmt.Sprintf("Rp %d.%03d", price/1000, price%1000)
		if price >= 1000000 {
			millions := price / 1000000
			thousands := (price % 1000000) / 1000
			hundreds := price % 1000
			priceStr = fmt.Sprintf("Rp %d.%03d.%03d", millions, thousands, hundreds)
		}
	}
	return priceStr
}

func handleBuyProduct(bot *tgbotapi.BotAPI, chatID int64, productCode string) {
	// Check if user is logged in
	if !service.IsUserLoggedIn(chatID) {
		text := `🔒 *Verifikasi Diperlukan*

Untuk membeli paket data, Anda perlu memverifikasi nomor HP terlebih dahulu.

Silakan verifikasi nomor HP Anda untuk melanjutkan pembelian.`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📞 Verifikasi Sekarang", "verify_phone"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali ke Produk", "products"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending verification required message: %v", err)
		}
		return
	}

	// Get user session to get phone number
	userSession, err := service.GetUserSession(chatID)
	if err != nil || userSession.PhoneNumber == "" {
		sendErrorMessage(bot, chatID, "❌ Nomor HP tidak ditemukan. Silakan login ulang.")
		return
	}

	// User is verified, proceed with purchase
	p, ok := packageMap[productCode]
	if !ok {
		sendErrorMessage(bot, chatID, "❌ Produk tidak ditemukan. Silakan pilih produk lain.")
		return
	}

	// Store selected product and phone number in user state
	setUserData(chatID, userSession.PhoneNumber, "", productCode)

	// Debug: Verify data was stored
	verifyState := getUserState(chatID)
	verifyState.mu.RLock()
	log.Printf("DEBUG handleBuyProduct AFTER setUserData - User %d: phone='%s', productCode='%s'",
		chatID, verifyState.PhoneNumber, verifyState.ProductCode)
	verifyState.mu.RUnlock()

	priceStr := formatPrice(p.Price)
	text := fmt.Sprintf(`✅ *Produk Dipilih*

📦 *Produk:* %s
💰 *Harga:* %s
📱 *Nomor:* %s

Silakan lanjutkan ke pembayaran untuk menyelesaikan pembelian.`, p.Name, priceStr, userSession.PhoneNumber)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💳 Lanjut Pembayaran", "proceed_payment"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Pilih Produk Lain", "products"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending product selection: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

func handleVerifyPhone(bot *tgbotapi.BotAPI, chatID int64) {
	setUserState(chatID, "waiting_phone")

	text := `📞 *Verifikasi Nomor HP*

Silakan masukkan nomor HP Anda yang akan digunakan untuk menerima paket data.

*Format yang benar:*
• 08xxxxxxxxxx
• +628xxxxxxxxxx
• 628xxxxxxxxxx

*Contoh:* 087817739901

Ketik nomor HP Anda:`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending phone verification request: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

func handlePhoneInput(bot *tgbotapi.BotAPI, chatID int64, phoneNumber string) {
	// Validate phone number format
	phoneNumber = strings.TrimSpace(phoneNumber)

	// Normalize phone number
	normalizedPhone := normalizePhoneNumber(phoneNumber)
	if !isValidPhoneNumber(normalizedPhone) {
		text := `❌ *Format Nomor Tidak Valid*

Silakan masukkan nomor HP dengan format yang benar:

*Format yang benar:*
• 08xxxxxxxxxx
• +628xxxxxxxxxx  
• 628xxxxxxxxxx

*Contoh:* 087817739901

Ketik nomor HP Anda:`

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid phone message: %v", err)
		}
		return
	}

	// Send OTP request
	otpResp, err := service.RequestOTP(normalizedPhone)
	if err != nil {
		log.Printf("Error requesting OTP: %v", err)
		sendErrorMessage(bot, chatID, "❌ Maaf, gagal mengirim kode OTP. Silakan coba lagi nanti.")
		setUserState(chatID, "start")
		return
	}

	// Store user data and update state
	setUserData(chatID, normalizedPhone, otpResp.Data.AuthID, "")
	setUserState(chatID, "waiting_otp")

	text := fmt.Sprintf(`✅ *Kode OTP Terkirim*

Kode OTP telah dikirim ke nomor: *%s*

%s

Silakan masukkan kode OTP yang Anda terima:

⏰ Kode dapat dikirim ulang dalam %d detik`, normalizedPhone, otpResp.Message, otpResp.Data.CanResendIn)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending OTP sent message: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

func handleOTPInput(bot *tgbotapi.BotAPI, chatID int64, otpCode string) {
	userState := getUserState(chatID)

	userState.mu.RLock()
	phoneNumber := userState.PhoneNumber
	userState.mu.RUnlock()

	if phoneNumber == "" {
		sendErrorMessage(bot, chatID, "❌ Sesi verifikasi tidak valid. Silakan mulai ulang verifikasi.")
		setUserState(chatID, "start")
		return
	}

	// Validate OTP format
	otpCode = strings.TrimSpace(otpCode)
	if !regexp.MustCompile(`^\d{4,6}$`).MatchString(otpCode) {
		text := `❌ *Format Kode OTP Tidak Valid*

Silakan masukkan kode OTP yang benar (4-6 digit angka).

*Contoh:* 123456

Masukkan kode OTP Anda:`

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid OTP message: %v", err)
		}
		return
	}

	// Verify OTP and get access token
	verifyResp, err := service.VerifyOTPAndLogin(phoneNumber, otpCode, chatID)
	if err != nil {
		log.Printf("Error verifying OTP: %v", err)
		sendErrorMessage(bot, chatID, "❌ Maaf, gagal memverifikasi kode OTP. Silakan coba lagi.")
		return
	}

	if !verifyResp.Success {
		text := fmt.Sprintf(`❌ *Kode OTP Salah*

%s

Silakan masukkan kode OTP yang benar:`, verifyResp.Message)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending wrong OTP message: %v", err)
		}
		return
	}

	// OTP verified successfully and logged in
	setUserState(chatID, "verified")

	text := fmt.Sprintf(`✅ *Login Berhasil!*

Nomor HP *%s* telah berhasil diverifikasi dan Anda sudah login.

🔑 *Access Token:* Aktif selama 1 jam
⏰ *Berlaku sampai:* %s

Sekarang Anda dapat membeli paket data dengan aman. Silakan pilih produk yang Anda inginkan.`,
		phoneNumber,
		time.Now().Add(1*time.Hour).Format("15:04 WIB"))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Lihat Produk", "products"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔓 Logout", "logout"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending login success message: %v", err)
		sendErrorMessage(bot, chatID, "Login berhasil! Silakan pilih produk yang Anda inginkan.")
	}
}

func normalizePhoneNumber(phone string) string {
	// Remove all non-digit characters except +
	re := regexp.MustCompile(`[^\d+]`)
	phone = re.ReplaceAllString(phone, "")

	// Handle different formats
	if strings.HasPrefix(phone, "+62") {
		return "0" + phone[3:]
	} else if strings.HasPrefix(phone, "62") && len(phone) > 10 {
		return "0" + phone[2:]
	} else if strings.HasPrefix(phone, "0") {
		return phone
	}

	return phone
}

func isValidPhoneNumber(phone string) bool {
	// Indonesian phone number validation
	// Should start with 08 and have 10-13 digits total
	re := regexp.MustCompile(`^08\d{8,11}$`)
	return re.MatchString(phone)
}

// Admin Functions

func handleContactAdmin(bot *tgbotapi.BotAPI, chatID int64) {
	setUserState(chatID, "waiting_admin_message")

	text := `👨‍💼 *Hubungi Admin GRN Store*

Silakan ketik pesan Anda untuk admin. Admin akan merespons secepat mungkin.

*Contoh pesan:*
• Pertanyaan tentang produk
• Keluhan atau masalah
• Saran dan masukan
• Bantuan teknis

Ketik pesan Anda:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending contact admin message: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

func handleAdminMessageInput(bot *tgbotapi.BotAPI, chatID int64, message string, user *tgbotapi.User) {
	// Send message to admin
	err := service.SendMessageToAdmin(bot, message, user)
	if err != nil {
		log.Printf("Error sending message to admin: %v", err)
		sendErrorMessage(bot, chatID, "❌ Maaf, gagal mengirim pesan ke admin. Silakan coba lagi nanti.")
		setUserState(chatID, "start")
		return
	}

	// Reset user state
	setUserState(chatID, "start")

	text := `✅ *Pesan Terkirim!*

Pesan Anda telah berhasil dikirim ke admin GRN Store.

Admin akan merespons pesan Anda secepat mungkin. Terima kasih atas kepercayaan Anda!`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending message sent confirmation: %v", err)
		sendErrorMessage(bot, chatID, "Pesan berhasil dikirim ke admin!")
	}
}

func handleAdminCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Check if user is admin
	if !config.IsAdmin(chatID) {
		sendErrorMessage(bot, chatID, "❌ Anda tidak memiliki akses admin.")
		return
	}

	text := `👨‍💼 *Panel Admin GRN Store*

Selamat datang, Admin! Pilih menu admin yang Anda butuhkan:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Statistik Bot", "admin_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Pending Top-Up", "admin_pending"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📢 Broadcast Message", "admin_broadcast"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending admin panel: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan.")
	}
}

func handleStatsCommand(bot *tgbotapi.BotAPI, chatID int64) {
	// Check if user is admin
	if !config.IsAdmin(chatID) {
		sendErrorMessage(bot, chatID, "❌ Anda tidak memiliki akses admin.")
		return
	}

	stats := service.GetUserStats()

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "admin_stats"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Panel Admin", "admin_panel"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, stats)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending stats: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan.")
	}
}

func handleProceedPayment(bot *tgbotapi.BotAPI, chatID int64) {
	userState := getUserState(chatID)

	userState.mu.RLock()
	productCode := userState.ProductCode
	phoneNumber := userState.PhoneNumber
	state := userState.State
	authID := userState.AuthID
	userState.mu.RUnlock()

	log.Printf("DEBUG handleProceedPayment - User %d: state='%s', productCode='%s', phoneNumber='%s', authID='%s'", chatID, state, productCode, phoneNumber, authID)

	if productCode == "" {
		log.Printf("ERROR: No product selected for user %d", chatID)
		sendErrorMessage(bot, chatID, "❌ Tidak ada produk yang dipilih. Silakan pilih produk terlebih dahulu.")
		return
	}

	p, ok := packageMap[productCode]
	if !ok {
		sendErrorMessage(bot, chatID, "❌ Produk tidak ditemukan. Silakan pilih produk lain.")
		return
	}

	// Get available payment methods for this product
	paymentMethods, err := service.GetAvailablePaymentMethods(productCode)
	if err != nil {
		log.Printf("Error getting payment methods for product %s: %v", productCode, err)
		sendErrorMessage(bot, chatID, "❌ Gagal memuat metode pembayaran. Silakan coba lagi.")
		return
	}

	if len(paymentMethods) == 0 {
		sendErrorMessage(bot, chatID, "❌ Tidak ada metode pembayaran yang tersedia untuk produk ini.")
		return
	}

	// Notify admin about new order
	adminNotification := fmt.Sprintf(`🛒 *Pesanan Baru!*

👤 *Customer:* %d
📱 *Nomor:* %s
📦 *Produk:* %s
💰 *Harga:* %s

⏰ *Waktu:* %s`, chatID, phoneNumber, p.Name, formatPrice(p.Price), "Sekarang")

	service.SendAdminNotification(bot, adminNotification)

	// Display payment method selection
	text := fmt.Sprintf(`💳 *Pilih Metode Pembayaran*

📦 *Produk:* %s
💰 *Harga:* %s
📱 *Nomor:* %s

Silakan pilih metode pembayaran yang Anda inginkan:`, p.Name, formatPrice(p.Price), phoneNumber)

	var rows [][]tgbotapi.InlineKeyboardButton

	// Add payment method buttons
	for _, pm := range paymentMethods {
		btnText := fmt.Sprintf("💳 %s", pm.PaymentMethodDisplayName)
		callbackData := fmt.Sprintf("pay:%s:%s", productCode, pm.PaymentMethod)
		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, callbackData)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// Add back buttons
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Pilih Produk Lain", "products"),
		tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
	))

	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending payment methods: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

// Top-Up Functions

func handleTopUpRequest(bot *tgbotapi.BotAPI, chatID int64) {
	setUserState(chatID, "waiting_topup_amount")

	text := `╔══════════════════════════╗
║     💳 *TOP UP SALDO*    ║
╚══════════════════════════╝

💰 *Masukkan Nominal Custom*

Silakan ketik nominal top up yang Anda inginkan.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 *KETENTUAN:*
• 💵 Minimum: Rp 10.000
• 💎 Maximum: Rp 1.000.000
• ⚠️ Hanya angka (tanpa titik/koma)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 *CONTOH INPUT:*
• Untuk Rp 50.000 → ketik: *50000*
• Untuk Rp 100.000 → ketik: *100000*
• Untuk Rp 250.000 → ketik: *250000*

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚡ *Pembayaran via QRIS - Aman & Cepat*

🔤 *Ketik nominal sekarang:*`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending top up request: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan. Silakan coba lagi.")
	}
}

func handleTopUpAmountInput(bot *tgbotapi.BotAPI, chatID int64, amountStr string, user *tgbotapi.User) {
	// Parse amount
	amount, err := strconv.ParseInt(strings.TrimSpace(amountStr), 10, 64)
	if err != nil {
		text := `❌ *Format Nominal Tidak Valid*

Silakan masukkan nominal dengan format yang benar (hanya angka).

*Contoh:* 50000

Ketik nominal top up:`

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid amount message: %v", err)
		}
		return
	}

	// Validate amount
	if amount < 10000 {
		sendErrorMessage(bot, chatID, "❌ Minimal top up adalah Rp 10.000")
		return
	}
	if amount > 1000000 {
		sendErrorMessage(bot, chatID, "❌ Maksimal top up adalah Rp 1.000.000")
		return
	}

	// Get username
	username := getUserDisplayName(user)

	// Create top up transaction
	topUpResp, err := service.CreateTopUpTransaction(chatID, username, amount)
	if err != nil {
		log.Printf("Error creating top up transaction: %v", err)
		// Show user-friendly error message (admin already notified by service)
		sendErrorMessage(bot, chatID, fmt.Sprintf("❌ %s", err.Error()))
		setUserState(chatID, "start")
		return
	}

	// Reset user state
	setUserState(chatID, "start")

	// Generate QR code
	qrBytes, err := service.GenerateQRCodeBytes(topUpResp.Data.QRISCode)
	if err != nil {
		log.Printf("Error generating QR code: %v", err)
		service.NotifyAdminError(chatID, "QR Code Generation", fmt.Sprintf("Failed to generate QR code for topup: %v", err))
		sendErrorMessage(bot, chatID, "❌ Terjadi kesalahan sistem, silakan coba lagi")
		return
	}

	// Send QR code
	photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
		Name:  "qris.png",
		Bytes: qrBytes,
	})

	text := fmt.Sprintf(`💰 *QRIS Top Up - GRN Store*

💳 *Nominal:* %s
🆔 *Transaction ID:* `+"`%s`"+`
⏰ *Berlaku sampai:* %s

*Cara Pembayaran:*
1️⃣ Scan QR code di atas dengan aplikasi e-wallet
2️⃣ Pastikan nominal sesuai: %s
3️⃣ Lakukan pembayaran
4️⃣ Tunggu konfirmasi dari admin

⚠️ *Penting:*
• QR code berlaku selama 30 menit
• Jangan transfer dengan nominal berbeda
• Hubungi admin jika ada kendala

Admin akan mengkonfirmasi pembayaran Anda secepat mungkin.`,
		formatPrice(amount),
		topUpResp.Data.TransactionID,
		topUpResp.Data.ExpiredAt,
		formatPrice(amount))

	photoMsg.Caption = text
	photoMsg.ParseMode = "Markdown"

	if _, err := bot.Send(photoMsg); err != nil {
		log.Printf("Error sending QR code: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan saat mengirim QR code.")
	}

	// Notify admin about new top up request
	adminNotification := fmt.Sprintf(`💰 *Top Up Request Baru!*

👤 *User:* %s (%d)
💳 *Nominal:* %s
🆔 *Transaction ID:* `+"`%s`"+`
⏰ *Expired:* %s

Menunggu pembayaran dari user.
Gunakan /confirm %s untuk approve setelah pembayaran diterima.`,
		username, chatID, formatPrice(amount), topUpResp.Data.TransactionID, topUpResp.Data.ExpiredAt, topUpResp.Data.TransactionID)

	service.SendAdminNotification(bot, adminNotification)

	// Send WhatsApp notification to admin about new topup request
	whatsappMsg := fmt.Sprintf(`🔔 TOPUP REQUEST BARU

User: %s (%d)
Nominal: %s
Transaction ID: %s
Status: Menunggu Pembayaran

Silakan cek pembayaran dan konfirmasi jika sudah diterima.`,
		username, chatID, formatPrice(amount), topUpResp.Data.TransactionID)

	service.SendWhatsAppNotification(whatsappMsg)
}

func handleBalanceCommand(bot *tgbotapi.BotAPI, chatID int64) {
	balance := service.GetUserBalance(chatID)

	text := fmt.Sprintf(`💳 *Saldo Anda*

💰 *Saldo saat ini:* %s

Gunakan saldo untuk membeli paket data di GRN Store.`, formatPrice(balance.Balance))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Top Up Saldo", "topup"),
			tgbotapi.NewInlineKeyboardButtonData("📱 Lihat Produk", "products"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending balance info: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan.")
	}
}

// Admin Top-Up Management Functions

func handlePendingCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Check if user is admin
	if !config.IsAdmin(chatID) {
		sendErrorMessage(bot, chatID, "❌ Anda tidak memiliki akses admin.")
		return
	}

	pendingTxs := service.GetPendingTransactions()

	if len(pendingTxs) == 0 {
		text := `📋 *Pending Top-Up Transactions*

Tidak ada transaksi pending saat ini.

Semua transaksi sudah diproses atau expired.`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "admin_pending"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Panel Admin", "admin_panel"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending empty pending list: %v", err)
		}
		return
	}

	text := `📋 *Pending Top-Up Transactions*

Daftar transaksi yang menunggu konfirmasi:

`

	// Create keyboard with approve/reject buttons for each transaction
	var keyboardRows [][]tgbotapi.InlineKeyboardButton

	for i, tx := range pendingTxs {
		text += fmt.Sprintf(`%d. *%s* (ID: %d)
   💳 Nominal: %s
   🆔 ID: `+"`%s`"+`
   ⏰ Expired: %s
   
`, i+1, tx.Username, tx.UserID, formatPrice(tx.Amount), tx.ID, tx.ExpiredAt)

		// Add approve/reject buttons for each transaction
		keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("✅ Approve #%d", i+1), fmt.Sprintf("approve_tx:%s", tx.ID)),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("❌ Reject #%d", i+1), fmt.Sprintf("reject_tx:%s", tx.ID)),
		))
	}

	text += `*Klik tombol di bawah untuk approve/reject transaksi:*`

	// Add control buttons
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "admin_pending"),
	))
	keyboardRows = append(keyboardRows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Panel Admin", "admin_panel"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending pending list: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan.")
	}
}

func handleConfirmCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Check if user is admin
	if !config.IsAdmin(chatID) {
		sendErrorMessage(bot, chatID, "❌ Anda tidak memiliki akses admin.")
		return
	}

	// Parse command arguments
	args := strings.Fields(message.Text)
	if len(args) != 2 {
		sendErrorMessage(bot, chatID, "❌ Format salah. Gunakan: /confirm <transaction_id>")
		return
	}

	transactionID := args[1]

	// Confirm transaction
	err := service.ConfirmTopUp(transactionID, chatID)
	if err != nil {
		log.Printf("Error confirming top up: %v", err)
		service.NotifyAdminError(chatID, "Topup Confirmation", fmt.Sprintf("Failed to confirm topup %s: %v", transactionID, err))
		sendErrorMessage(bot, chatID, fmt.Sprintf("❌ Gagal konfirmasi: %s", err.Error()))
		return
	}

	// Get transaction details for notification
	var confirmedTx *dto.Transaction

	// Find in all transactions (including confirmed ones)
	service.TxMutex.RLock()
	if tx, exists := service.Transactions[transactionID]; exists {
		confirmedTx = tx
	}
	service.TxMutex.RUnlock()

	if confirmedTx == nil {
		sendErrorMessage(bot, chatID, "❌ Transaksi tidak ditemukan.")
		return
	}

	// Get updated balance
	balance := service.GetUserBalance(confirmedTx.UserID)

	// Send confirmation to admin
	adminText := fmt.Sprintf(`✅ *Top-Up Berhasil Dikonfirmasi*

👤 *User:* %s (%d)
💳 *Nominal:* %s
🆔 *Transaction ID:* %s
💰 *Saldo User Sekarang:* %s

Notifikasi telah dikirim ke user.`,
		confirmedTx.Username,
		confirmedTx.UserID,
		formatPrice(confirmedTx.Amount),
		transactionID,
		formatPrice(balance.Balance))

	msg := tgbotapi.NewMessage(chatID, adminText)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending admin confirmation: %v", err)
	}

	// Send notification to user
	userText := fmt.Sprintf(`✅ *Top-Up Berhasil!*

💳 *Nominal:* %s
💰 *Saldo Anda sekarang:* %s

Terima kasih! Saldo Anda telah berhasil ditambahkan.
Sekarang Anda dapat membeli paket data di GRN Store.`,
		formatPrice(confirmedTx.Amount),
		formatPrice(balance.Balance))

	userKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Lihat Produk", "products"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	userMsg := tgbotapi.NewMessage(confirmedTx.UserID, userText)
	userMsg.ParseMode = "Markdown"
	userMsg.ReplyMarkup = userKeyboard

	if _, err := bot.Send(userMsg); err != nil {
		log.Printf("Error sending user notification: %v", err)
	}

	// Send WhatsApp notification to admin
	whatsappMsg := fmt.Sprintf(`✅ TOPUP BERHASIL DIKONFIRMASI

User: %s (%d)
Nominal: %s
Saldo Sekarang: %s
Transaction ID: %s

Saldo user telah berhasil ditambahkan.`,
		confirmedTx.Username,
		confirmedTx.UserID,
		formatPrice(confirmedTx.Amount),
		formatPrice(balance.Balance),
		transactionID)

	service.SendWhatsAppNotification(whatsappMsg)
}

func handleRejectCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Check if user is admin
	if !config.IsAdmin(chatID) {
		sendErrorMessage(bot, chatID, "❌ Anda tidak memiliki akses admin.")
		return
	}

	// Parse command arguments
	args := strings.Fields(message.Text)
	if len(args) != 2 {
		sendErrorMessage(bot, chatID, "❌ Format salah. Gunakan: /reject <transaction_id>")
		return
	}

	transactionID := args[1]

	// Get transaction details before rejection
	service.TxMutex.RLock()
	rejectedTx, exists := service.Transactions[transactionID]
	service.TxMutex.RUnlock()

	if !exists {
		sendErrorMessage(bot, chatID, "❌ Transaksi tidak ditemukan.")
		return
	}

	// Reject transaction
	err := service.RejectTopUp(transactionID, chatID)
	if err != nil {
		log.Printf("Error rejecting top up: %v", err)
		sendErrorMessage(bot, chatID, fmt.Sprintf("❌ Gagal menolak: %s", err.Error()))
		return
	}

	// Send confirmation to admin
	adminText := fmt.Sprintf(`❌ *Top-Up Ditolak*

👤 *User:* %s (%d)
💳 *Nominal:* %s
🆔 *Transaction ID:* %s

Transaksi telah ditolak dan user akan diberitahu.`,
		rejectedTx.Username,
		rejectedTx.UserID,
		formatPrice(rejectedTx.Amount),
		transactionID)

	msg := tgbotapi.NewMessage(chatID, adminText)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending admin rejection confirmation: %v", err)
	}

	// Send notification to user
	userText := fmt.Sprintf(`❌ *Top-Up Ditolak*

💳 *Nominal:* %s
🆔 *Transaction ID:* %s

Maaf, transaksi top-up Anda ditolak oleh admin.
Silakan hubungi admin untuk informasi lebih lanjut atau coba lagi.`,
		formatPrice(rejectedTx.Amount),
		transactionID)

	userKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("👨‍💼 Hubungi Admin", "contact_admin"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Top Up Lagi", "topup"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	userMsg := tgbotapi.NewMessage(rejectedTx.UserID, userText)
	userMsg.ParseMode = "Markdown"
	userMsg.ReplyMarkup = userKeyboard

	if _, err := bot.Send(userMsg); err != nil {
		log.Printf("Error sending user rejection notification: %v", err)
	}

	// Send WhatsApp notification for rejected topup
	whatsappMsg := fmt.Sprintf(`❌ TOPUP DITOLAK

User: %s (%d)
Nominal: %s
Transaction ID: %s

Transaksi topup telah ditolak oleh admin.`,
		rejectedTx.Username,
		rejectedTx.UserID,
		formatPrice(rejectedTx.Amount),
		transactionID)

	service.SendWhatsAppNotification(whatsappMsg)
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

func handleDebugCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Check if user is admin
	if !config.IsAdmin(chatID) {
		sendErrorMessage(bot, chatID, "❌ Anda tidak memiliki akses admin.")
		return
	}

	service.TxMutex.RLock()
	totalTx := len(service.Transactions)

	text := fmt.Sprintf(`🔍 *Debug Info*

📊 *Total Transactions:* %d

*Transaction IDs:*
`, totalTx)

	if totalTx == 0 {
		text += "Tidak ada transaksi dalam memory.\n"
	} else {
		for id, tx := range service.Transactions {
			text += fmt.Sprintf("• `%s`\n  User: %s (%d)\n  Amount: %s\n  Status: %s\n\n",
				id, tx.Username, tx.UserID, formatPrice(tx.Amount), tx.Status)
		}
	}
	service.TxMutex.RUnlock()

	text += `*Note:* Data disimpan in-memory, akan hilang saat bot restart.`

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending debug info: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan.")
	}
}

// Broadcast Functions

func handleBroadcastCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Check if user is admin
	if !config.IsAdmin(chatID) {
		sendErrorMessage(bot, chatID, "❌ Anda tidak memiliki akses admin.")
		return
	}

	// Parse command arguments
	args := strings.Fields(message.Text)
	if len(args) < 2 {
		sendErrorMessage(bot, chatID, "❌ Format salah. Gunakan: /broadcast <pesan>")
		return
	}

	// Get broadcast message
	broadcastMessage := strings.Join(args[1:], " ")

	// Get all user IDs
	userIDs := service.GetAllUserIDs()

	if len(userIDs) == 0 {
		sendErrorMessage(bot, chatID, "❌ Tidak ada user untuk broadcast.")
		return
	}

	// Send broadcast
	err := service.BroadcastMessage(bot, broadcastMessage, userIDs)
	if err != nil {
		log.Printf("Error broadcasting message: %v", err)
		sendErrorMessage(bot, chatID, "❌ Gagal mengirim broadcast.")
		return
	}

	// Confirmation message already sent by BroadcastMessage function
}

func handleBroadcastRequest(bot *tgbotapi.BotAPI, chatID int64) {
	setUserState(chatID, "waiting_broadcast_message")

	userIDs := service.GetAllUserIDs()

	text := fmt.Sprintf(`📢 *Broadcast Message*

Anda akan mengirim pesan ke *%d user* yang pernah berinteraksi dengan bot.

Silakan ketik pesan yang ingin Anda broadcast:

*Tips:*
• Gunakan format Markdown untuk formatting
• Pesan akan dikirim ke semua user
• Pastikan pesan sudah benar sebelum mengirim

*Contoh:*
🎉 *Promo Spesial GRN Store!*
Dapatkan bonus 20%% untuk top-up hari ini!`, len(userIDs))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "admin_panel"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending broadcast request: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan.")
	}
}

func handleBroadcastMessageInput(bot *tgbotapi.BotAPI, chatID int64, message string) {
	// Reset user state
	setUserState(chatID, "start")

	// Get all user IDs
	userIDs := service.GetAllUserIDs()

	if len(userIDs) == 0 {
		sendErrorMessage(bot, chatID, "❌ Tidak ada user untuk broadcast.")
		return
	}

	// Confirm broadcast
	text := fmt.Sprintf(`📢 *Konfirmasi Broadcast*

*Pesan yang akan dikirim:*
%s

*Target:* %d user

Apakah Anda yakin ingin mengirim broadcast ini?`, message, len(userIDs))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Kirim Sekarang", fmt.Sprintf("send_broadcast:%s", message)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "admin_panel"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending broadcast confirmation: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan.")
	}
}

func handleSendBroadcast(bot *tgbotapi.BotAPI, chatID int64, message string) {
	// Get all user IDs
	userIDs := service.GetAllUserIDs()

	if len(userIDs) == 0 {
		sendErrorMessage(bot, chatID, "❌ Tidak ada user untuk broadcast.")
		return
	}

	// Send broadcast
	err := service.BroadcastMessage(bot, message, userIDs)
	if err != nil {
		log.Printf("Error broadcasting message: %v", err)
		sendErrorMessage(bot, chatID, "❌ Gagal mengirim broadcast.")
		return
	}

	// Send confirmation to admin
	confirmText := fmt.Sprintf(`✅ *Broadcast Berhasil Dikirim*

*Pesan:* %s
*Target:* %d user

Laporan detail akan dikirim setelah semua pesan terkirim.`, message, len(userIDs))

	msg := tgbotapi.NewMessage(chatID, confirmText)
	msg.ParseMode = "Markdown"

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending broadcast confirmation: %v", err)
	}
}

// Product Detail Functions

func handleProductDetail(bot *tgbotapi.BotAPI, chatID int64, productCode string) {
	packages, err := service.FetchPackages()
	if err != nil {
		log.Printf("Error fetching packages: %v", err)
		sendErrorMessage(bot, chatID, "❌ Maaf, gagal memuat detail produk.")
		return
	}

	// Find the product
	var selectedProduct *dto.Package
	for _, p := range packages {
		if p.PackageCode == productCode {
			selectedProduct = &p
			break
		}
	}

	if selectedProduct == nil {
		sendErrorMessage(bot, chatID, "❌ Produk tidak ditemukan.")
		return
	}

	// Format product detail
	text := formatProductDetail(selectedProduct)

	// Create keyboard with buy option
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🛒 Beli Sekarang", "buy:"+productCode),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Kembali ke Daftar", "products"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending product detail: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan saat menampilkan detail produk.")
	}
}

func formatProductDetail(p *dto.Package) string {
	text := fmt.Sprintf(`📦 *Detail Produk - GRN Store*

🏷️ *Nama:* %s

💰 *Harga:* %s

📝 *Deskripsi:*
%s

`, p.PackageName, formatPrice(p.Price), p.PackageDescription)

	// Add features
	text += "✨ *Fitur:*\n"

	if p.CanMultiTrx {
		text += "• ✅ Multi Transaction\n"
	}
	if p.CanScheduledTrx {
		text += "• ⏰ Scheduled Transaction\n"
	}
	if p.NoNeedLogin {
		text += "• 🔓 No Login Required\n"
	}

	// Add daily limit info
	if p.HaveDailyLimit {
		text += fmt.Sprintf("\n📊 *Limit Harian:*\n• Max: %d transaksi\n• Terpakai: %d transaksi\n",
			p.DailyLimitDetails.MaxDailyTransactionLimit,
			p.DailyLimitDetails.CurrentDailyTransactionCount)
	}

	// Add cut off time if exists
	if p.HaveCutOffTime {
		text += fmt.Sprintf("\n⏰ *Jam Operasional:*\n• Tidak tersedia: %s - %s\n",
			p.CutOffTime.ProhibitedHourStarttime,
			p.CutOffTime.ProhibitedHourEndtime)
	}

	// Add payment methods
	if p.IsShowPaymentMethod && len(p.AvailablePaymentMethods) > 0 {
		text += "\n💳 *Metode Pembayaran:*\n"
		for _, pm := range p.AvailablePaymentMethods {
			text += fmt.Sprintf("• %s\n", pm.PaymentMethodDisplayName)
		}
	}

	return text
}

// Authentication Functions

func handleLogout(bot *tgbotapi.BotAPI, chatID int64) {
	err := service.ClearUserSession(chatID)
	if err != nil {
		log.Printf("Error clearing user session: %v", err)
		sendErrorMessage(bot, chatID, "❌ Maaf, terjadi kesalahan saat logout.")
		return
	}

	text := `🔓 *Logout Berhasil*

Anda telah berhasil logout dari sistem.

Untuk membeli produk lagi, Anda perlu login ulang dengan verifikasi OTP.`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📞 Login Lagi", "verify_phone"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending logout message: %v", err)
		sendErrorMessage(bot, chatID, "Logout berhasil!")
	}
}

// Payment Functions

func handlePayment(bot *tgbotapi.BotAPI, chatID int64, productCode, paymentMethod string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL ERROR in handlePayment: %v", r)
			service.NotifyAdminError(chatID, "Payment System", fmt.Sprintf("Critical error: %v", r))
			sendErrorMessage(bot, chatID, "❌ Terjadi kesalahan sistem. Tim teknis telah diberitahu. Silakan hubungi admin.")
		}
	}()

	// Check if user is logged in
	if !service.IsUserLoggedIn(chatID) {
		text := `🔒 *Session Expired*

Session login Anda telah berakhir. Silakan login ulang untuk melanjutkan pembelian.`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📞 Login Ulang", "verify_phone"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending session expired message: %v", err)
		}
		return
	}

	// Validate payment method
	if paymentMethod == "" {
		log.Printf("Empty payment method for user %d", chatID)
		sendErrorMessage(bot, chatID, "❌ Metode pembayaran tidak valid. Silakan pilih ulang.")
		return
	}

	// Get product price for balance validation
	packages, err := service.FetchPackages()
	if err != nil {
		log.Printf("Error fetching packages for balance check: %v", err)
		service.NotifyAdminError(chatID, "Package Fetch", fmt.Sprintf("Failed to fetch packages: %v", err))
		sendErrorMessage(bot, chatID, "❌ Gagal memuat data produk. Silakan coba lagi.")
		return
	}

	var packagePrice int64
	var productName string
	for _, pkg := range packages {
		if pkg.PackageCode == productCode {
			packagePrice = pkg.Price // Already includes +1500 from API
			productName = pkg.PackageName
			break
		}
	}

	if productName == "" {
		log.Printf("Product not found for code %s", productCode)
		sendErrorMessage(bot, chatID, "❌ Produk tidak ditemukan. Silakan pilih produk lain.")
		return
	}

	// Check user balance against package price (already includes +1500 from API)
	balance := service.GetUserBalance(chatID)
	if balance.Balance < packagePrice {
		text := fmt.Sprintf(`❌ *Saldo Tidak Mencukupi*

📦 *Produk:* %s
💰 *Harga:* %s
💳 *Saldo Anda:* %s
💸 *Kurang:* %s

Silakan top up saldo terlebih dahulu.`,
			productName,
			formatPrice(packagePrice),
			formatPrice(balance.Balance),
			formatPrice(packagePrice-balance.Balance))

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💰 Top Up Saldo", "topup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending insufficient balance message: %v", err)
		}
		return
	}

	// Send processing message
	processingMsg := tgbotapi.NewMessage(chatID, "⏳ Memproses pembayaran, mohon tunggu...")
	sentMsg, err := bot.Send(processingMsg)
	if err != nil {
		log.Printf("Error sending processing message: %v", err)
	}

	// Make purchase
	purchaseResp, err := service.PurchaseProduct(chatID, productCode, paymentMethod)
	if err != nil {
		log.Printf("Error making purchase for user %d: %v", chatID, err)

		// Delete processing message
		if sentMsg.MessageID != 0 {
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
			bot.Send(deleteMsg)
		}

		// Show user-friendly error message (admin already notified by service)
		sendErrorMessage(bot, chatID, fmt.Sprintf("❌ %s", err.Error()))
		return
	}

	// Delete processing message
	if sentMsg.MessageID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
		bot.Send(deleteMsg)
	}

	if !purchaseResp.Success {
		log.Printf("Purchase failed for user %d: %s", chatID, purchaseResp.Message)
		sendErrorMessage(bot, chatID, fmt.Sprintf("❌ Pembelian gagal: %s", purchaseResp.Message))
		return
	}

	// Validate response data
	if purchaseResp.Data.TrxID == "" {
		log.Printf("Empty transaction ID in purchase response for user %d", chatID)
		service.NotifyAdminError(chatID, "Purchase Validation", "Empty transaction ID in response")
		sendErrorMessage(bot, chatID, "❌ Transaksi tidak valid. Silakan hubungi admin.")
		return
	}

	// Handle different payment methods based on response
	qrisData := purchaseResp.Data.GetQRISData()
	if purchaseResp.Data.IsQRIS && qrisData.QRCode != "" {
		handleQRISPayment(bot, chatID, purchaseResp)
	} else if purchaseResp.Data.HaveDeeplink && purchaseResp.Data.DeeplinkData.DeeplinkURL != "" {
		handleDeeplinkPayment(bot, chatID, purchaseResp)
	} else {
		// Direct payment (BALANCE or instant)
		handleDirectPayment(bot, chatID, purchaseResp)
	}
}

func handleDirectPayment(bot *tgbotapi.BotAPI, chatID int64, purchaseResp *dto.PurchaseResponse) {
	// Deduct user balance for all payment methods - use full price (original + 1500)
	err := service.DeductUserBalance(chatID, purchaseResp.Data.Price)
	if err != nil {
		log.Printf("Error deducting balance for user %d: %v", chatID, err)
		service.NotifyAdminError(chatID, "Balance Deduction", fmt.Sprintf("Failed to deduct balance for transaction %s: %v", purchaseResp.Data.TrxID, err))
		sendErrorMessage(bot, chatID, "❌ Gagal memotong saldo. Silakan hubungi admin.")
		return
	}

	// Get updated balance
	balance := service.GetUserBalance(chatID)

	text := fmt.Sprintf(`✅ *Pembelian Berhasil!*

📦 *Produk:* %s
💰 *Harga:* %s
💳 *Metode:* %s
🆔 *Transaction ID:* %s
💰 *Saldo Tersisa:* %s

%s

Paket data akan segera aktif di nomor Anda.`,
		purchaseResp.Data.PackageName,
		formatPrice(purchaseResp.Data.PackageProcessingFee),
		purchaseResp.Data.DeeplinkData.PaymentMethod,
		purchaseResp.Data.TrxID,
		formatPrice(balance.Balance),
		purchaseResp.Message)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📱 Lihat Produk Lain", "products"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending purchase success message: %v", err)
	}

	// Send WhatsApp notification for successful transaction
	user, _ := service.GetUserSession(chatID)
	username := "Unknown"
	if user != nil {
		username = user.PhoneNumber
	}

	whatsappMsg := fmt.Sprintf(`✅ TRANSAKSI BERHASIL

User: %s (%d)
Produk: %s
Harga: %s
Metode: %s
Transaction ID: %s
Saldo Tersisa: %s

Paket data telah berhasil diproses.`,
		username, chatID,
		purchaseResp.Data.PackageName,
		formatPrice(purchaseResp.Data.PackageProcessingFee),
		purchaseResp.Data.DeeplinkData.PaymentMethod,
		purchaseResp.Data.TrxID,
		formatPrice(balance.Balance))

	service.SendWhatsAppNotification(whatsappMsg)
}

func handleQRISPayment(bot *tgbotapi.BotAPI, chatID int64, purchaseResp *dto.PurchaseResponse) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL ERROR in handleQRISPayment: %v", r)
			service.NotifyAdminError(chatID, "QRIS Payment", fmt.Sprintf("QRIS Payment Error for user %d: %v", chatID, r))
			sendErrorMessage(bot, chatID, "❌ Terjadi kesalahan sistem. Tim teknis telah diberitahu. Silakan hubungi admin.")
		}
	}()

	// Deduct user balance for QRIS payment - use full price (original + 1500)
	err := service.DeductUserBalance(chatID, purchaseResp.Data.Price)
	if err != nil {
		log.Printf("Error deducting balance for user %d: %v", chatID, err)
		service.NotifyAdminError(chatID, "Balance Deduction", fmt.Sprintf("Failed to deduct balance for transaction %s: %v", purchaseResp.Data.TrxID, err))
		sendErrorMessage(bot, chatID, "❌ Gagal memotong saldo. Silakan hubungi admin.")
		return
	}

	qrisData := purchaseResp.Data.GetQRISData()
	if qrisData.QRCode == "" {
		log.Printf("No QRIS data available for transaction %s", purchaseResp.Data.TrxID)
		service.NotifyAdminError(chatID, "QRIS Payment", fmt.Sprintf("No QRIS data for transaction %s", purchaseResp.Data.TrxID))
		sendErrorMessage(bot, chatID, "❌ Data QRIS tidak tersedia. Silakan hubungi admin.")
		return
	}

	// Generate QR code image from QRIS string
	qrBytes, err := service.GenerateQRCodeBytes(qrisData.QRCode)
	if err != nil {
		log.Printf("Error generating QR code for user %d: %v", chatID, err)
		service.NotifyAdminError(chatID, "QR Code Generation", fmt.Sprintf("Failed to generate QR code for transaction %s: %v", purchaseResp.Data.TrxID, err))
		sendErrorMessage(bot, chatID, "❌ Gagal membuat QR code. Silakan hubungi admin.")
		return
	}

	// Send QR code image
	photoMsg := tgbotapi.NewPhoto(chatID, tgbotapi.FileBytes{
		Name:  "payment_qris.png",
		Bytes: qrBytes,
	})

	text := fmt.Sprintf(`💳 *Pembayaran QRIS*

📦 *Produk:* %s
🆔 *Transaction ID:* %s
⏰ *Berlaku sampai:* %d detik

*Cara Pembayaran:*
1️⃣ Scan QR code di atas dengan aplikasi e-wallet
2️⃣ Lakukan pembayaran sesuai nominal yang tertera
3️⃣ Paket akan otomatis aktif setelah pembayaran berhasil

⚠️ *Penting:* QR code akan expired dalam %d detik. Segera lakukan pembayaran!`,
		purchaseResp.Data.PackageName,
		purchaseResp.Data.TrxID,
		qrisData.RemainingTime,
		qrisData.RemainingTime)

	photoMsg.Caption = text
	photoMsg.ParseMode = "Markdown"

	// Add check status button
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Cek Status Pembayaran", fmt.Sprintf("check:%s", purchaseResp.Data.TrxID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)
	photoMsg.ReplyMarkup = keyboard

	if _, err := bot.Send(photoMsg); err != nil {
		log.Printf("Error sending QRIS payment to user %d: %v", chatID, err)
		service.NotifyAdminError(chatID, "Message Sending", fmt.Sprintf("Failed to send QRIS message: %v", err))
		sendErrorMessage(bot, chatID, "❌ Gagal mengirim QR code. Silakan hubungi admin.")
		return
	}

	// Send WhatsApp notification for QRIS payment initiated
	user, _ := service.GetUserSession(chatID)
	username := "Unknown"
	if user != nil {
		username = user.PhoneNumber
	}

	whatsappMsg := fmt.Sprintf(`💳 PEMBAYARAN QRIS DIMULAI

User: %s (%d)
Produk: %s
Harga: %s
Transaction ID: %s
Status: Menunggu Pembayaran QRIS

User sedang melakukan pembayaran via QRIS.`,
		username, chatID,
		purchaseResp.Data.PackageName,
		formatPrice(purchaseResp.Data.PackageProcessingFee),
		purchaseResp.Data.TrxID)

	service.SendWhatsAppNotification(whatsappMsg)
}

func handleDeeplinkPayment(bot *tgbotapi.BotAPI, chatID int64, purchaseResp *dto.PurchaseResponse) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL ERROR in handleDeeplinkPayment: %v", r)
			service.NotifyAdminError(chatID, "Deeplink Payment", fmt.Sprintf("Deeplink Payment Error for user %d: %v", chatID, r))
			sendErrorMessage(bot, chatID, "❌ Terjadi kesalahan sistem. Tim teknis telah diberitahu. Silakan hubungi admin.")
		}
	}()

	// Deduct user balance for deeplink payment - use full price (original + 1500)
	err := service.DeductUserBalance(chatID, purchaseResp.Data.Price)
	if err != nil {
		log.Printf("Error deducting balance for user %d: %v", chatID, err)
		service.NotifyAdminError(chatID, "Balance Deduction", fmt.Sprintf("Failed to deduct balance for transaction %s: %v", purchaseResp.Data.TrxID, err))
		sendErrorMessage(bot, chatID, "❌ Gagal memotong saldo. Silakan hubungi admin.")
		return
	}

	if purchaseResp.Data.DeeplinkData.DeeplinkURL == "" {
		log.Printf("Empty deeplink URL for transaction %s", purchaseResp.Data.TrxID)
		service.NotifyAdminError(chatID, "Deeplink Payment", fmt.Sprintf("Empty deeplink URL for transaction %s", purchaseResp.Data.TrxID))
		sendErrorMessage(bot, chatID, "❌ Link pembayaran tidak tersedia. Silakan hubungi admin.")
		return
	}

	paymentMethod := purchaseResp.Data.DeeplinkData.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = "E-Wallet"
	}

	text := fmt.Sprintf(`💳 *Pembayaran %s*

📦 *Produk:* %s
🆔 *Transaction ID:* %s

*Cara Pembayaran:*
1️⃣ Klik tombol "Bayar dengan %s" di bawah
2️⃣ Aplikasi %s akan terbuka otomatis
3️⃣ Konfirmasi pembayaran di aplikasi
4️⃣ Kembali ke bot dan cek status pembayaran

⚠️ *Penting:* Pastikan Anda memiliki saldo yang cukup di aplikasi %s.`,
		paymentMethod,
		purchaseResp.Data.PackageName,
		purchaseResp.Data.TrxID,
		paymentMethod,
		paymentMethod,
		paymentMethod)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(
				fmt.Sprintf("💳 Bayar dengan %s", paymentMethod),
				purchaseResp.Data.DeeplinkData.DeeplinkURL,
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Cek Status Pembayaran", fmt.Sprintf("check:%s", purchaseResp.Data.TrxID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending deeplink payment to user %d: %v", chatID, err)
		service.NotifyAdminError(chatID, "Message Sending", fmt.Sprintf("Failed to send deeplink message: %v", err))
		sendErrorMessage(bot, chatID, "❌ Gagal mengirim link pembayaran. Silakan hubungi admin.")
		return
	}

	// Don't send WhatsApp notification for successful transactions
	// Only send to admin via Telegram for errors and approvals
}

// Search Functions

func handleSearchRequest(bot *tgbotapi.BotAPI, chatID int64) {
	setUserState(chatID, "waiting_search_query")

	text := `🔍 *Cari Produk - GRN Store*

Silakan ketik kata kunci untuk mencari produk yang Anda inginkan.

*Contoh pencarian:*
• "masa aktif" - untuk paket masa aktif
• "kuota" - untuk paket data/kuota
• "axis" - untuk produk AXIS
• "xl" - untuk produk XL

*Tips:*
• Gunakan kata kunci yang spesifik
• Bisa menggunakan nama operator
• Bisa mencari berdasarkan jenis paket

Ketik kata kunci pencarian:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending search request: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan.")
	}
}

func handleSearchCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	args := strings.Fields(message.Text)

	if len(args) < 2 {
		handleSearchRequest(bot, chatID)
		return
	}

	query := strings.Join(args[1:], " ")
	handleSearchQueryInput(bot, chatID, query)
}

func handleSearchQueryInput(bot *tgbotapi.BotAPI, chatID int64, query string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL ERROR in handleSearchQueryInput: %v", r)
			service.NotifyAdminError(chatID, "Search System", fmt.Sprintf("Critical error: %v", r))
			sendErrorMessage(bot, chatID, "❌ Terjadi kesalahan sistem. Tim teknis telah diberitahu.")
		}
	}()

	setUserState(chatID, "start")

	query = strings.TrimSpace(query)
	if query == "" {
		sendErrorMessage(bot, chatID, "❌ Kata kunci pencarian tidak boleh kosong.")
		return
	}

	// Search products with default parameters
	searchResp, err := service.SearchProducts(query, 0, 1000000, "")
	if err != nil {
		log.Printf("Error searching products for user %d: %v", chatID, err)
		service.NotifyAdminError(chatID, "Search API", fmt.Sprintf("Search failed for query '%s': %v", query, err))
		sendErrorMessage(bot, chatID, "❌ Maaf, pencarian gagal. Silakan coba lagi atau hubungi admin.")
		return
	}

	if len(searchResp.Data) == 0 {
		text := fmt.Sprintf(`🔍 *Hasil Pencarian*

Kata kunci: "%s"

❌ Tidak ditemukan produk yang sesuai.

*Saran:*
• Coba kata kunci yang berbeda
• Gunakan kata kunci yang lebih umum
• Lihat semua produk di menu utama`, query)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📱 Lihat Semua Produk", "products"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔍 Cari Lagi", "search_products"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending search results: %v", err)
		}
		return
	}

	// Display search results
	displaySearchResults(bot, chatID, query, searchResp.Data, 0)
}

func displaySearchResults(bot *tgbotapi.BotAPI, chatID int64, query string, products []dto.Package, page int) {
	pageSize := 5 // Smaller page size for search results
	total := len(products)
	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	text := fmt.Sprintf(`🔍 *Hasil Pencarian*

Kata kunci: "%s"
Ditemukan: %d produk
Halaman: %d dari %d

`, query, total, page+1, (total+pageSize-1)/pageSize)

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, p := range products[start:end] {
		// Store in packageMap for later use
		packageMap[p.PackageCode] = service.PackageAlias{Name: p.PackageName, Price: p.Price}

		displayName := p.PackageNameAliasShort
		if displayName == "" {
			displayName = p.PackageName
		}

		if len(displayName) > 45 {
			displayName = displayName[:42] + "..."
		}

		btnText := fmt.Sprintf("%d. %s - %s", start+i+1, displayName, formatPrice(p.Price))
		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, "detail:"+p.PackageCode)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// Navigation buttons
	var navButtons []tgbotapi.InlineKeyboardButton
	if start > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Sebelumnya", fmt.Sprintf("search_page:%s:%d", query, page-1)))
	}
	if end < total {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Selanjutnya ➡️", fmt.Sprintf("search_page:%s:%d", query, page+1)))
	}
	if len(navButtons) > 0 {
		rows = append(rows, navButtons)
	}

	// Action buttons
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔍 Cari Lagi", "search_products"),
		tgbotapi.NewInlineKeyboardButtonData("📱 Semua Produk", "products"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
	))

	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending search results: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan saat menampilkan hasil pencarian.")
	}
}

// History Functions

func handleHistoryCommand(bot *tgbotapi.BotAPI, chatID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL ERROR in handleHistoryCommand: %v", r)
			service.NotifyAdminError(chatID, "History System", fmt.Sprintf("Critical error: %v", r))
			sendErrorMessage(bot, chatID, "❌ Terjadi kesalahan sistem. Tim teknis telah diberitahu.")
		}
	}()

	// Check if user is logged in
	if !service.IsUserLoggedIn(chatID) {
		text := `🔒 *Login Diperlukan*

Untuk melihat history transaksi, Anda perlu login terlebih dahulu.

Silakan verifikasi nomor HP Anda untuk login.`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📞 Login Sekarang", "verify_phone"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending login required message: %v", err)
		}
		return
	}

	// Get user's purchase history from database
	history, err := service.GetUserPurchaseHistory(chatID)
	if err != nil {
		log.Printf("Error getting purchase history for user %d: %v", chatID, err)
		service.NotifyAdminError(chatID, "Database", fmt.Sprintf("Failed to get purchase history: %v", err))
		sendErrorMessage(bot, chatID, "❌ Maaf, gagal memuat history transaksi. Silakan coba lagi atau hubungi admin.")
		return
	}

	if len(history) == 0 {
		text := `📋 *History Transaksi*

❌ Belum ada transaksi yang tercatat.

Mulai berbelanja sekarang untuk melihat history transaksi Anda.`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📱 Lihat Produk", "products"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending empty history: %v", err)
		}
		return
	}

	// Display history
	displayPurchaseHistory(bot, chatID, history, 0)
}

func displayPurchaseHistory(bot *tgbotapi.BotAPI, chatID int64, history []models.PurchaseTransaction, page int) {
	pageSize := 5
	total := len(history)
	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	text := fmt.Sprintf(`📋 *History Transaksi*

Total: %d transaksi
Halaman: %d dari %d

`, total, page+1, (total+pageSize-1)/pageSize)

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, tx := range history[start:end] {
		statusIcon := getStatusIcon(tx.Status)

		// Recalculate price to ensure consistency (in case old transactions have wrong price)
		var displayPrice int64
		if packagePrice, err := service.GetPackagePrice(tx.PackageCode); err == nil {
			displayPrice = packagePrice // Use current API price (includes +1500)
		} else {
			// Fallback to stored price if package lookup fails
			displayPrice = tx.Price
		}

		btnText := fmt.Sprintf("%d. %s %s - %s",
			start+i+1,
			statusIcon,
			tx.PackageName,
			formatPrice(displayPrice))

		if len(btnText) > 60 {
			btnText = btnText[:57] + "..."
		}

		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("history_detail:%s", tx.ID))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// Navigation buttons
	var navButtons []tgbotapi.InlineKeyboardButton
	if start > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Sebelumnya", fmt.Sprintf("history_page:%d", page-1)))
	}
	if end < total {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Selanjutnya ➡️", fmt.Sprintf("history_page:%d", page+1)))
	}
	if len(navButtons) > 0 {
		rows = append(rows, navButtons)
	}

	// Action buttons
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔄 Refresh", "history"),
		tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
	))

	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending purchase history: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan saat menampilkan history.")
	}
}

func getStatusIcon(status string) string {
	switch status {
	case "success":
		return "✅"
	case "pending":
		return "⏳"
	case "failed":
		return "❌"
	default:
		return "❓"
	}
}

// Transaction Check Functions

func handleCheckTransaction(bot *tgbotapi.BotAPI, chatID int64, transactionID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL ERROR in handleCheckTransaction: %v", r)
			service.NotifyAdminError(chatID, "Transaction Check", fmt.Sprintf("Critical error: %v", r))
			sendErrorMessage(bot, chatID, "❌ Terjadi kesalahan sistem. Tim teknis telah diberitahu.")
		}
	}()

	// Check transaction status via API
	checkResp, err := service.CheckTransactionStatus(transactionID)
	if err != nil {
		log.Printf("Error checking transaction %s for user %d: %v", transactionID, chatID, err)
		service.NotifyAdminError(chatID, "Transaction Check API", fmt.Sprintf("Failed to check transaction %s: %v", transactionID, err))
		sendErrorMessage(bot, chatID, "❌ Maaf, gagal mengecek status transaksi. Silakan coba lagi atau hubungi admin.")
		return
	}

	if !checkResp.Success {
		sendErrorMessage(bot, chatID, fmt.Sprintf("❌ Gagal mengecek transaksi: %s", checkResp.Message))
		return
	}

	// Format transaction status
	data := checkResp.Data

	// Get transaction from our database - use the stored price (from API that includes +1500)
	dbTransaction, err := service.GetPurchaseTransaction(transactionID)
	var displayPrice int64
	if err != nil {
		log.Printf("Warning: Could not get transaction from database: %v", err)
		// Fallback: Try to get package price from API (already includes +1500)
		if packagePrice, err := service.GetPackagePrice(data.Code); err == nil {
			displayPrice = packagePrice
		} else {
			// Last resort: default to 1500
			displayPrice = 1500
		}
		log.Printf("Using fallback price of %d for transaction %s", displayPrice, transactionID)
	} else {
		// Use stored price directly - it's from API that already includes +1500
		displayPrice = dbTransaction.Price
	}
	var statusText string

	if data.Status == 1 && data.RC == "00" {
		statusText = "✅ *BERHASIL*"
	} else {
		statusText = "❌ *GAGAL*"
	}

	text := fmt.Sprintf(`🔍 *Status Transaksi*

%s

🆔 *Transaction ID:* %s
📦 *Produk:* %s
💰 *Harga:* %s
📱 *Nomor Tujuan:* %s
⏰ *Waktu:* %s

📊 *Detail Status:*
• Response Code: %s
• Message: %s

`, statusText,
		data.TrxID,
		data.Name,
		formatPrice(displayPrice),
		data.DestinationMSISDN,
		data.TimeDate,
		data.RC,
		data.RCMessage)

	if data.Status == 1 && data.RC == "00" {
		text += `✅ *Transaksi berhasil!* Paket data telah aktif di nomor Anda.`
	} else {
		text += `❌ *Transaksi gagal.* Silakan hubungi admin jika ada masalah.`
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Cek Lagi", fmt.Sprintf("check:%s", transactionID)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 History", "history"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending transaction status: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan saat menampilkan status.")
		return
	}

	// Send WhatsApp notification for transaction status check
	user, _ := service.GetUserSession(chatID)
	username := "Unknown"
	if user != nil {
		username = user.PhoneNumber
	}

	var whatsappStatus string
	if data.Status == 1 && data.RC == "00" {
		whatsappStatus = "BERHASIL"
		// Only send notification for successful transactions to avoid spam
		whatsappMsg := fmt.Sprintf(`✅ TRANSAKSI BERHASIL

User: %s (%d)
Transaction ID: %s
Produk: %s
Harga: %s
Nomor Tujuan: %s
Status: %s

Paket data telah berhasil diaktivasi.`,
			username, chatID,
			data.TrxID,
			data.Name,
			formatPrice(data.TotalPrice),
			data.DestinationMSISDN,
			whatsappStatus)

		service.SendWhatsAppNotification(whatsappMsg)
	}
}

func handleTransactionDetail(bot *tgbotapi.BotAPI, chatID int64, transactionID string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("CRITICAL ERROR in handleTransactionDetail: %v", r)
			service.NotifyAdminError(chatID, "Transaction Detail", fmt.Sprintf("Critical error: %v", r))
			sendErrorMessage(bot, chatID, "❌ Terjadi kesalahan sistem. Tim teknis telah diberitahu.")
		}
	}()

	// Get transaction from database
	transaction, err := service.GetPurchaseTransactionByID(transactionID)
	if err != nil {
		log.Printf("Error getting transaction detail %s for user %d: %v", transactionID, chatID, err)
		sendErrorMessage(bot, chatID, "❌ Transaksi tidak ditemukan.")
		return
	}

	// Check if transaction belongs to user
	if transaction.UserID != chatID {
		log.Printf("Unauthorized access to transaction %s by user %d", transactionID, chatID)
		sendErrorMessage(bot, chatID, "❌ Akses tidak diizinkan.")
		return
	}

	statusIcon := getStatusIcon(transaction.Status)
	statusText := strings.ToUpper(transaction.Status)

	// Recalculate price to ensure consistency (in case old transactions have wrong price)
	var displayPrice int64
	if packagePrice, err := service.GetPackagePrice(transaction.PackageCode); err == nil {
		displayPrice = packagePrice // Use current API price (includes +1500)
	} else {
		// Fallback to stored price if package lookup fails
		displayPrice = transaction.Price
	}

	text := fmt.Sprintf(`📋 *Detail Transaksi*

%s *Status:* %s

🆔 *Transaction ID:* %s
📦 *Produk:* %s
💰 *Harga:* %s
💳 *Metode:* %s
📱 *Nomor:* %s
⏰ *Waktu:* %s

`, statusIcon, statusText,
		transaction.ID,
		transaction.PackageName,
		formatPrice(displayPrice),
		transaction.PaymentMethod,
		transaction.PhoneNumber,
		transaction.CreatedAt.Format("2006-01-02 15:04:05"))

	var keyboard tgbotapi.InlineKeyboardMarkup
	if transaction.Status == "pending" {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Cek Status Terbaru", fmt.Sprintf("check:%s", transactionID)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Kembali ke History", "history"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Kembali ke History", "history"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending transaction detail: %v", err)
		sendErrorMessage(bot, chatID, "Maaf, terjadi kesalahan saat menampilkan detail.")
	}
}

// New professional functions
func sendRulesMessage(bot *tgbotapi.BotAPI, chatID int64) {
	text := `╔══════════════════════════╗
║    📋 *PERATURAN BOT*    ║
╚══════════════════════════╝

ᴘᴇʀᴀᴛᴜʀᴀɴ ʙᴏᴛ
1. ✧ ᴅɪʟᴀʀᴀɴɢ sᴘᴀᴍ ʙᴏᴛ
2. ✧ ʙᴏᴛ ᴅɪᴀᴍ? ᴄᴏʙᴀ ʟᴀɢɪ sᴇᴛᴇʟᴀʜ ᴅᴇʟᴀʏ.  
3. ✧ ᴘᴀsᴛɪᴋᴀɴ ɴᴏᴍᴏʀ / ɪᴅ sᴜᴅᴀʜ ʙᴇɴᴀʀ.  
4. ✧ ᴅᴏʀ ɪɴᴛᴇʀɴᴇᴛ ᴛᴀɴᴘᴀ ɢᴀʀᴀɴsɪ.  
5. ✧ ᴍᴇɴᴊᴜᴀʟ VPN ʙᴜᴋᴀɴ ᴄᴏɴꜰɪɢ.  
6. ✧ ᴠɪʀᴛᴇx / ʙᴜɢ ᴅɪʟᴀʀᴀɴɢ
7. ✧ ᴛᴇʟᴘᴏɴ ʙᴏᴛ = ʙʟᴏᴋɪʀ ᴘᴇʀᴍᴀɴᴇɴ.  
8. ✧ ᴇʀʀᴏʀ? ʟᴀᴘᴏʀ ᴏᴡɴᴇʀ.  
9. ✧ ʙᴏᴛ ʟᴀᴍʙᴀᴛ? ᴊᴀɴɢᴀɴ sᴘᴀᴍ.  
10. ✧ ᴏʀᴅᴇʀ VPN / ᴘʀᴏᴅᴜᴋ ʟᴀɪɴ: ʜᴜʙᴜɴɢɪ ᴏᴡɴᴇʀ

━━━━━━━━━━━━━━━━━━━━━━━━━━━━

⚠️ *PENTING:*
• Dengan menggunakan bot ini, Anda setuju dengan semua peraturan di atas
• Pelanggaran dapat mengakibatkan pemblokiran permanen
• Untuk pertanyaan lebih lanjut, hubungi admin

🏪 *GRN Store - Terpercaya & Profesional*`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			tgbotapi.NewInlineKeyboardButtonData("❓ Bantuan", "help"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending rules message: %v", err)
	}
}

func handleTopUpCommand(bot *tgbotapi.BotAPI, chatID int64) {
	text := `╔══════════════════════════╗
║     💳 *TOP UP SALDO*    ║
╚══════════════════════════╝

💰 *Pilih Nominal Top Up:*

🔥 *PAKET HEMAT:*
• Rp 10.000 - Untuk pembelian kecil
• Rp 25.000 - Paket populer ⭐
• Rp 50.000 - Hemat lebih banyak

💎 *PAKET PREMIUM:*
• Rp 100.000 - Bonus ekstra
• Rp 250.000 - Super hemat
• Rp 500.000 - Untuk reseller

━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📋 *Cara Top Up:*
1️⃣ Pilih nominal di bawah
2️⃣ Scan QRIS yang muncul
3️⃣ Bayar sesuai nominal
4️⃣ Saldo otomatis masuk 1-5 menit

⚡ *Pembayaran via QRIS - Aman & Cepat*`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Rp 10.000", "topup:10000"),
			tgbotapi.NewInlineKeyboardButtonData("💰 Rp 25.000", "topup:25000"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Rp 50.000", "topup:50000"),
			tgbotapi.NewInlineKeyboardButtonData("💰 Rp 100.000", "topup:100000"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💰 Rp 250.000", "topup:250000"),
			tgbotapi.NewInlineKeyboardButtonData("💰 Rp 500.000", "topup:500000"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Nominal Lain", "topup:custom"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending topup message: %v", err)
	}
}

func handleHistoryCommandNew(bot *tgbotapi.BotAPI, chatID int64) {
	history, err := service.GetUserPurchaseHistory(chatID)
	if err != nil {
		log.Printf("Error getting purchase history: %v", err)
		sendErrorMessage(bot, chatID, "❌ Gagal mengambil riwayat transaksi.")
		return
	}

	if len(history) == 0 {
		text := `╔══════════════════════════╗
║    📜 *RIWAYAT KOSONG*   ║
╚══════════════════════════╝

🔍 *Belum Ada Transaksi*

Anda belum melakukan transaksi apapun.
Mulai berbelanja sekarang untuk melihat riwayat transaksi Anda!

🛍️ *Yuk mulai belanja!*`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🛍️ Mulai Belanja", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending empty history message: %v", err)
		}
		return
	}

	handleHistoryCommand(bot, chatID)
}

// handleApproveTransaction handles transaction approval via inline button
func handleApproveTransaction(bot *tgbotapi.BotAPI, chatID int64, transactionID string) {
	// Check if user is admin
	if !config.IsAdmin(chatID) {
		bot.Request(tgbotapi.NewCallback("", "❌ Anda tidak memiliki akses admin."))
		return
	}

	// Confirm transaction
	err := service.ConfirmTopUp(transactionID, chatID)
	if err != nil {
		log.Printf("Error confirming top up: %v", err)
		bot.Request(tgbotapi.NewCallback("", "❌ Gagal approve transaksi."))
		return
	}

	// Send success message
	text := fmt.Sprintf(`✅ *Transaksi Berhasil Di-Approve*

🆔 *Transaction ID:* `+"`%s`"+`
👤 *Approved by:* Admin
⏰ *Waktu:* %s

User telah mendapat notifikasi dan saldo telah ditambahkan.`,
		transactionID,
		time.Now().Format("02/01/2006 15:04:05"))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Lihat Pending", "admin_pending"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Panel Admin", "admin_panel"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending approve confirmation: %v", err)
	}
}

// handleRejectTransaction handles transaction rejection via inline button
func handleRejectTransaction(bot *tgbotapi.BotAPI, chatID int64, transactionID string) {
	// Check if user is admin
	if !config.IsAdmin(chatID) {
		bot.Request(tgbotapi.NewCallback("", "❌ Anda tidak memiliki akses admin."))
		return
	}

	// Reject transaction
	err := service.RejectTopUp(transactionID, chatID)
	if err != nil {
		log.Printf("Error rejecting top up: %v", err)
		bot.Request(tgbotapi.NewCallback("", "❌ Gagal reject transaksi."))
		return
	}

	// Send success message
	text := fmt.Sprintf(`❌ *Transaksi Berhasil Di-Reject*

🆔 *Transaction ID:* `+"`%s`"+`
👤 *Rejected by:* Admin
⏰ *Waktu:* %s

User telah mendapat notifikasi penolakan.`,
		transactionID,
		time.Now().Format("02/01/2006 15:04:05"))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Lihat Pending", "admin_pending"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Panel Admin", "admin_panel"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending reject confirmation: %v", err)
	}
}

// VPN Functions

func handleVPNMenu(bot *tgbotapi.BotAPI, chatID int64) {
	// Check if user has minimum balance
	balance := service.GetUserBalance(chatID)
	if balance.Balance < 10000 {
		text := fmt.Sprintf(`🔐 *VPN Premium - GRN Store*

❌ *Saldo Tidak Mencukupi*

Untuk menggunakan layanan VPN, Anda memerlukan minimal saldo Rp 10.000.

💳 *Saldo Anda saat ini:* %s
💰 *Minimal saldo:* Rp 10.000
💸 *Kurang:* %s

Silakan top up saldo terlebih dahulu.`, 
			formatPrice(balance.Balance), 
			formatPrice(10000-balance.Balance))

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💰 Top Up Saldo", "topup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending VPN insufficient balance: %v", err)
		}
		return
	}

	text := `🔐 *VPN Premium - GRN Store*

🌟 *Server Singapore - Kualitas Terbaik*
💰 *Harga:* Rp 8.000/bulan (fleksibel per hari)
📊 *Perhitungan:* Rp 266.67/hari
💳 *Saldo Anda:* ` + formatPrice(balance.Balance) + `

🔒 *Protokol Tersedia:*
• SSH/SSL - Stabil & Cepat
• Trojan - Anti Blokir
• VLESS - Modern & Efisien
• VMESS - Fleksibel & Aman

✨ *Fitur Unggulan:*
• 🌐 Server Singapore Premium
• ⚡ Koneksi Super Cepat
• 🔒 Enkripsi Tingkat Militer
• 📱 Support Semua Device
• 🎯 Anti Lag Gaming
• 📺 Streaming Lancar

💡 *Fleksibilitas Pembayaran:*
Beli sesuai kebutuhan - 1 hari, 7 hari, 30 hari, atau custom!`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔑 SSH/SSL", "vpn_create:ssh"),
			tgbotapi.NewInlineKeyboardButtonData("🛡️ Trojan", "vpn_create:trojan"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚡ VLESS", "vpn_create:vless"),
			tgbotapi.NewInlineKeyboardButtonData("🔐 VMESS", "vpn_create:vmess"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 VPN Saya", "vpn_list"),
			tgbotapi.NewInlineKeyboardButtonData("📜 Riwayat VPN", "vpn_history"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN menu: %v", err)
	}
}

func handleVPNCreateStart(bot *tgbotapi.BotAPI, chatID int64, protocol string) {
	// Store protocol in user state
	setUserVPNData(chatID, protocol, "", "", "")
	setUserState(chatID, "waiting_vpn_email")

	protocolName := map[string]string{
		"ssh":    "SSH/SSL",
		"trojan": "Trojan",
		"vless":  "VLESS",
		"vmess":  "VMESS",
	}

	text := fmt.Sprintf(`🔐 *Buat VPN %s*

📧 *Langkah 1: Email*

Masukkan email untuk akun VPN Anda.
Email ini akan digunakan untuk identifikasi akun.

*Contoh:* user@gmail.com

⚠️ *Catatan:* Email tidak perlu valid/aktif, hanya untuk identifikasi.

Ketik email Anda:`, protocolName[protocol])

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "vpn_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN create start: %v", err)
	}
}

func handleVPNEmailInput(bot *tgbotapi.BotAPI, chatID int64, email string) {
	email = strings.TrimSpace(email)
	
	// Basic email validation
	if !strings.Contains(email, "@") || len(email) < 5 {
		text := `❌ *Format Email Tidak Valid*

Silakan masukkan email dengan format yang benar.

*Contoh:* user@gmail.com

Ketik email Anda:`

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid email message: %v", err)
		}
		return
	}

	// Store email and move to password
	setUserVPNData(chatID, "", email, "", "")
	setUserState(chatID, "waiting_vpn_password")

	text := `🔐 *Langkah 2: Password*

Masukkan password untuk akun VPN Anda.

*Syarat Password:*
• Minimal 6 karakter
• Boleh kombinasi huruf, angka, simbol
• Mudah diingat untuk Anda

*Contoh:* mypass123

Ketik password Anda:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "vpn_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN password request: %v", err)
	}
}

func handleVPNPasswordInput(bot *tgbotapi.BotAPI, chatID int64, password string) {
	password = strings.TrimSpace(password)
	
	// Basic password validation
	if len(password) < 6 {
		text := `❌ *Password Terlalu Pendek*

Password minimal 6 karakter.

*Contoh:* mypass123

Ketik password Anda:`

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid password message: %v", err)
		}
		return
	}

	// Store password and move to days
	setUserVPNData(chatID, "", "", password, "")
	setUserState(chatID, "waiting_vpn_days")

	text := `📅 *Langkah 3: Durasi*

Berapa hari VPN yang ingin Anda beli?

💰 *Perhitungan Harga:*
• 1 hari = Rp 267
• 7 hari = Rp 1.867  
• 15 hari = Rp 4.000
• 30 hari = Rp 8.000

*Contoh Input:*
• Ketik: 1 (untuk 1 hari)
• Ketik: 7 (untuk 1 minggu)
• Ketik: 30 (untuk 1 bulan)

Ketik jumlah hari:`

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1 hari", "vpn_days:1"),
			tgbotapi.NewInlineKeyboardButtonData("7 hari", "vpn_days:7"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("15 hari", "vpn_days:15"),
			tgbotapi.NewInlineKeyboardButtonData("30 hari", "vpn_days:30"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "vpn_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN days request: %v", err)
	}
}

func handleVPNDaysInput(bot *tgbotapi.BotAPI, chatID int64, daysStr string) {
	days, err := strconv.Atoi(strings.TrimSpace(daysStr))
	if err != nil || days <= 0 {
		text := `❌ *Input Tidak Valid*

Silakan masukkan angka yang valid untuk jumlah hari.

*Contoh:* 1, 7, 15, 30

Ketik jumlah hari:`

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid days message: %v", err)
		}
		return
	}

	if days > 365 {
		text := `❌ *Maksimal 365 Hari*

Untuk keamanan, maksimal pembelian VPN adalah 365 hari.

Ketik jumlah hari (1-365):`

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending max days message: %v", err)
		}
		return
	}

	// Get user state
	userState := getUserState(chatID)
	userState.mu.RLock()
	protocol := userState.VPNProtocol
	email := userState.VPNEmail
	password := userState.VPNPassword
	userState.mu.RUnlock()

	if protocol == "" || email == "" || password == "" {
		sendErrorMessage(bot, chatID, "❌ Data tidak lengkap. Silakan mulai ulang.")
		setUserState(chatID, "start")
		return
	}

	// Calculate price
	price := service.CalculateVPNPrice(days)
	
	// Check balance
	balance := service.GetUserBalance(chatID)
	if balance.Balance < price {
		text := fmt.Sprintf(`❌ *Saldo Tidak Mencukupi*

💰 *Harga VPN %d hari:* %s
💳 *Saldo Anda:* %s
💸 *Kurang:* %s

Silakan top up saldo terlebih dahulu.`, 
			days, formatPrice(price), formatPrice(balance.Balance), formatPrice(price-balance.Balance))

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💰 Top Up Saldo", "topup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Menu VPN", "vpn_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending VPN insufficient balance: %v", err)
		}
		return
	}

	// Show confirmation
	protocolName := map[string]string{
		"ssh":    "SSH/SSL",
		"trojan": "Trojan",
		"vless":  "VLESS",
		"vmess":  "VMESS",
	}

	text := fmt.Sprintf(`✅ *Konfirmasi Pembelian VPN*

🔐 *Protokol:* %s
📧 *Email:* %s
🔑 *Password:* %s
📅 *Durasi:* %d hari
💰 *Harga:* %s
💳 *Saldo Tersisa:* %s

Apakah Anda yakin ingin membeli VPN ini?`, 
		protocolName[protocol], email, password, days, formatPrice(price), formatPrice(balance.Balance-price))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Ya, Beli Sekarang", fmt.Sprintf("vpn_confirm:%d", days)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "vpn_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN confirmation: %v", err)
	}
}

func handleVPNList(bot *tgbotapi.BotAPI, chatID int64) {
	vpnUsers, err := service.GetUserVPNs(chatID)
	if err != nil {
		log.Printf("Error getting user VPNs: %v", err)
		sendErrorMessage(bot, chatID, "❌ Gagal mengambil data VPN Anda.")
		return
	}

	if len(vpnUsers) == 0 {
		text := `📋 *VPN Saya*

❌ Anda belum memiliki VPN aktif.

Buat VPN pertama Anda sekarang!`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔐 Buat VPN", "vpn_menu"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending empty VPN list: %v", err)
		}
		return
	}

	text := fmt.Sprintf(`📋 *VPN Saya* (%d VPN)

`, len(vpnUsers))

	var rows [][]tgbotapi.InlineKeyboardButton
	for i, vpn := range vpnUsers {
		status := "🟢 Aktif"
		if time.Now().After(vpn.ExpiredAt) {
			status = "🔴 Expired"
		}

		btnText := fmt.Sprintf("%d. %s %s - %s", i+1, strings.ToUpper(vpn.Protocol), vpn.VPNUsername, status)
		if len(btnText) > 60 {
			btnText = btnText[:57] + "..."
		}

		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("vpn_detail:%s", vpn.VPNUsername))
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// Add control buttons
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔐 Buat VPN Baru", "vpn_menu"),
	))
	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
	))

	keyboard := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN list: %v", err)
	}
}

func handleVPNHistory(bot *tgbotapi.BotAPI, chatID int64) {
	transactions, err := service.GetVPNTransactionHistory(chatID)
	if err != nil {
		log.Printf("Error getting VPN history: %v", err)
		sendErrorMessage(bot, chatID, "❌ Gagal mengambil riwayat VPN.")
		return
	}

	if len(transactions) == 0 {
		text := `📜 *Riwayat VPN*

❌ Belum ada transaksi VPN.

Buat VPN pertama Anda sekarang!`

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔐 Buat VPN", "vpn_menu"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending empty VPN history: %v", err)
		}
		return
	}

	text := fmt.Sprintf(`📜 *Riwayat VPN* (%d transaksi)

`, len(transactions))

	for i, tx := range transactions {
		if i >= 10 { // Limit to 10 recent transactions
			break
		}

		statusIcon := getStatusIcon(tx.Status)
		action := "Buat"
		if tx.Email == "extend" {
			action = "Perpanjang"
		}

		text += fmt.Sprintf(`%d. %s %s %s
   📅 %d hari - %s
   💰 %s - %s

`, i+1, statusIcon, action, strings.ToUpper(tx.Protocol), tx.Days, formatPrice(tx.Price), tx.CreatedAt.Format("02/01/06"))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 VPN Saya", "vpn_list"),
			tgbotapi.NewInlineKeyboardButtonData("🔐 Buat VPN", "vpn_menu"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN history: %v", err)
	}
}

func handleVPNConfirm(bot *tgbotapi.BotAPI, chatID int64, daysStr string) {
	days, err := strconv.Atoi(daysStr)
	if err != nil {
		sendErrorMessage(bot, chatID, "❌ Data tidak valid. Silakan mulai ulang.")
		return
	}

	// Get user state
	userState := getUserState(chatID)
	userState.mu.RLock()
	protocol := userState.VPNProtocol
	email := userState.VPNEmail
	password := userState.VPNPassword
	userState.mu.RUnlock()

	if protocol == "" || email == "" || password == "" {
		sendErrorMessage(bot, chatID, "❌ Data tidak lengkap. Silakan mulai ulang.")
		setUserState(chatID, "start")
		return
	}

	// Send processing message
	processingMsg := tgbotapi.NewMessage(chatID, "⏳ Sedang membuat VPN Anda, mohon tunggu...")
	sentMsg, err := bot.Send(processingMsg)
	if err != nil {
		log.Printf("Error sending processing message: %v", err)
	}

	// Create VPN
	vpnTx, err := service.CreateVPNUser(chatID, "", email, password, protocol, days)
	if err != nil {
		// Delete processing message
		if sentMsg.MessageID != 0 {
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
			bot.Send(deleteMsg)
		}

		sendErrorMessage(bot, chatID, fmt.Sprintf("❌ %s", err.Error()))
		setUserState(chatID, "start")
		return
	}

	// Delete processing message
	if sentMsg.MessageID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
		bot.Send(deleteMsg)
	}

	// Reset user state
	setUserState(chatID, "start")

	// Get updated balance
	balance := service.GetUserBalance(chatID)

	// Parse response data to show complete config
	var responseData map[string]interface{}
	configText := ""
	if vpnTx.ResponseData != "" {
		json.Unmarshal([]byte(vpnTx.ResponseData), &responseData)
		if data, ok := responseData["data"].(map[string]interface{}); ok {
			configText = formatVPNConfig(protocol, data)
		}
	}

	// Send success message
	text := fmt.Sprintf(`✅ *VPN Berhasil Dibuat!*

🔐 *Protokol:* %s
👤 *Username:* %s
🔑 *Password:* %s
📅 *Durasi:* %d hari
💰 *Harga:* %s
💳 *Saldo Tersisa:* %s

%s

🎉 VPN Anda sudah aktif dan siap digunakan!`, 
		strings.ToUpper(protocol), vpnTx.Username, password, days, 
		formatPrice(vpnTx.Price), formatPrice(balance.Balance), configText)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Lihat VPN Saya", "vpn_list"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔐 Buat VPN Lagi", "vpn_menu"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN success message: %v", err)
	}

	// Send WhatsApp notification
	whatsappMsg := fmt.Sprintf(`🔐 VPN BARU DIBUAT

User: %d
Protokol: %s
Username: %s
Durasi: %d hari
Harga: %s
Saldo Tersisa: %s

VPN berhasil dibuat dan aktif.`,
		chatID, strings.ToUpper(protocol), vpnTx.Username, days,
		formatPrice(vpnTx.Price), formatPrice(balance.Balance))

	service.SendWhatsAppNotification(whatsappMsg)
}

func handleVPNDetail(bot *tgbotapi.BotAPI, chatID int64, vpnUsername string) {
	// Get VPN details from database
	vpnUsers, err := service.GetUserVPNs(chatID)
	if err != nil {
		sendErrorMessage(bot, chatID, "❌ Gagal mengambil data VPN.")
		return
	}

	var selectedVPN *models.VPNUser
	for _, vpn := range vpnUsers {
		if vpn.VPNUsername == vpnUsername {
			selectedVPN = &vpn
			break
		}
	}

	if selectedVPN == nil {
		sendErrorMessage(bot, chatID, "❌ VPN tidak ditemukan.")
		return
	}

	// Parse config data
	var config map[string]interface{}
	if selectedVPN.ConfigData != "" {
		json.Unmarshal([]byte(selectedVPN.ConfigData), &config)
	}

	status := "🟢 Aktif"
	daysLeft := int(time.Until(selectedVPN.ExpiredAt).Hours() / 24)
	if time.Now().After(selectedVPN.ExpiredAt) {
		status = "🔴 Expired"
		daysLeft = 0
	}

	text := fmt.Sprintf(`🔐 *Detail VPN %s*

📊 *Status:* %s
👤 *Username:* %s
🔑 *Password:* %s
🌐 *Server:* %s
🔌 *Port:* %d
📅 *Expired:* %s
⏰ *Sisa:* %d hari

`, strings.ToUpper(selectedVPN.Protocol), status, selectedVPN.VPNUsername, 
		selectedVPN.Password, selectedVPN.Server, selectedVPN.Port,
		selectedVPN.ExpiredAt.Format("02/01/2006 15:04"), daysLeft)

	// Add complete protocol-specific config
	if config != nil {
		text += formatVPNConfigFromDB(selectedVPN.Protocol, config, selectedVPN.UUID)
	}

	var keyboard tgbotapi.InlineKeyboardMarkup
	if status == "🟢 Aktif" {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("⏰ Perpanjang", fmt.Sprintf("vpn_extend:%s", vpnUsername)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Kembali ke List", "vpn_list"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)
	} else {
		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Perpanjang", fmt.Sprintf("vpn_extend:%s", vpnUsername)),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📋 Kembali ke List", "vpn_list"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
			),
		)
	}

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN detail: %v", err)
	}
}

func handleVPNExtendStart(bot *tgbotapi.BotAPI, chatID int64, vpnUsername string) {
	// Store VPN username in state
	setUserVPNData(chatID, "", "", "", vpnUsername)
	setUserState(chatID, "waiting_vpn_extend_days")

	text := fmt.Sprintf(`⏰ *Perpanjang VPN*

👤 *VPN Username:* %s

Berapa hari ingin diperpanjang?

💰 *Perhitungan Harga:*
• 1 hari = Rp 267
• 7 hari = Rp 1.867  
• 15 hari = Rp 4.000
• 30 hari = Rp 8.000

*Contoh Input:*
• Ketik: 7 (untuk 1 minggu)
• Ketik: 30 (untuk 1 bulan)

Ketik jumlah hari:`, vpnUsername)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("7 hari", "vpn_extend_days:7"),
			tgbotapi.NewInlineKeyboardButtonData("15 hari", "vpn_extend_days:15"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("30 hari", "vpn_extend_days:30"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Batal", "vpn_list"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN extend start: %v", err)
	}
}

func handleVPNExtendDaysInput(bot *tgbotapi.BotAPI, chatID int64, daysStr string) {
	days, err := strconv.Atoi(strings.TrimSpace(daysStr))
	if err != nil || days <= 0 {
		text := `❌ *Input Tidak Valid*

Silakan masukkan angka yang valid untuk jumlah hari.

*Contoh:* 7, 15, 30

Ketik jumlah hari:`

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending invalid extend days: %v", err)
		}
		return
	}

	if days > 365 {
		sendErrorMessage(bot, chatID, "❌ Maksimal perpanjangan adalah 365 hari.")
		return
	}

	// Get VPN username from state
	userState := getUserState(chatID)
	userState.mu.RLock()
	vpnUsername := userState.VPNUsername
	userState.mu.RUnlock()

	if vpnUsername == "" {
		sendErrorMessage(bot, chatID, "❌ Data tidak lengkap. Silakan mulai ulang.")
		setUserState(chatID, "start")
		return
	}

	// Calculate price
	price := service.CalculateVPNPrice(days)
	
	// Check balance
	balance := service.GetUserBalance(chatID)
	if balance.Balance < price {
		text := fmt.Sprintf(`❌ *Saldo Tidak Mencukupi*

💰 *Harga perpanjangan %d hari:* %s
💳 *Saldo Anda:* %s
💸 *Kurang:* %s

Silakan top up saldo terlebih dahulu.`, 
			days, formatPrice(price), formatPrice(balance.Balance), formatPrice(price-balance.Balance))

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💰 Top Up Saldo", "topup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Menu VPN", "vpn_menu"),
			),
		)

		msg := tgbotapi.NewMessage(chatID, text)
		msg.ParseMode = "Markdown"
		msg.ReplyMarkup = keyboard

		if _, err := bot.Send(msg); err != nil {
			log.Printf("Error sending VPN extend insufficient balance: %v", err)
		}
		return
	}

	// Send processing message
	processingMsg := tgbotapi.NewMessage(chatID, "⏳ Sedang memperpanjang VPN Anda, mohon tunggu...")
	sentMsg, err := bot.Send(processingMsg)
	if err != nil {
		log.Printf("Error sending processing message: %v", err)
	}

	// Extend VPN
	err = service.ExtendVPNUser(chatID, vpnUsername, days)
	if err != nil {
		// Delete processing message
		if sentMsg.MessageID != 0 {
			deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
			bot.Send(deleteMsg)
		}

		sendErrorMessage(bot, chatID, fmt.Sprintf("❌ %s", err.Error()))
		setUserState(chatID, "start")
		return
	}

	// Delete processing message
	if sentMsg.MessageID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(chatID, sentMsg.MessageID)
		bot.Send(deleteMsg)
	}

	// Reset user state
	setUserState(chatID, "start")

	// Get updated balance
	balance = service.GetUserBalance(chatID)

	// Send success message
	text := fmt.Sprintf(`✅ *VPN Berhasil Diperpanjang!*

👤 *Username:* %s
📅 *Diperpanjang:* %d hari
💰 *Harga:* %s
💳 *Saldo Tersisa:* %s

🎉 VPN Anda telah diperpanjang dan masih aktif!`, 
		vpnUsername, days, formatPrice(price), formatPrice(balance.Balance))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Lihat Detail", fmt.Sprintf("vpn_detail:%s", vpnUsername)),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 VPN Saya", "vpn_list"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Menu Utama", "main_menu"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = keyboard

	if _, err := bot.Send(msg); err != nil {
		log.Printf("Error sending VPN extend success: %v", err)
	}

	// Send WhatsApp notification
	whatsappMsg := fmt.Sprintf(`⏰ VPN DIPERPANJANG

User: %d
Username: %s
Diperpanjang: %d hari
Harga: %s
Saldo Tersisa: %s

VPN berhasil diperpanjang.`,
		chatID, vpnUsername, days, formatPrice(price), formatPrice(balance.Balance))

	service.SendWhatsAppNotification(whatsappMsg)
}

// formatVPNConfig formats VPN configuration from API response
func formatVPNConfig(protocol string, data map[string]interface{}) string {
	var text string
	
	switch protocol {
	case "ssh":
		text += "🔧 *Konfigurasi SSH/SSL:*\n"
		if server, ok := data["server"]; ok {
			text += fmt.Sprintf("• 🌐 Server: `%v`\n", server)
		}
		if port, ok := data["port"]; ok {
			text += fmt.Sprintf("• 🔌 SSH Port: `%v`\n", port)
		}
		if config, ok := data["config"].(map[string]interface{}); ok {
			if sslPort, ok := config["ssl_port"]; ok {
				text += fmt.Sprintf("• 🔒 SSL Port: `%v`\n", sslPort)
			}
			if stunnelPort, ok := config["stunnel_port"]; ok {
				text += fmt.Sprintf("• 🔐 Stunnel Port: `%v`\n", stunnelPort)
			}
			if wsPort, ok := config["ws_port"]; ok {
				text += fmt.Sprintf("• 🌐 WebSocket Port: `%v`\n", wsPort)
			}
		}
		
	case "trojan":
		text += "🔧 *Konfigurasi Trojan:*\n"
		if server, ok := data["server"]; ok {
			text += fmt.Sprintf("• 🌐 Server: `%v`\n", server)
		}
		if port, ok := data["port"]; ok {
			text += fmt.Sprintf("• 🔌 Port: `%v`\n", port)
		}
		if password, ok := data["password"]; ok {
			text += fmt.Sprintf("• 🔑 Key: `%v`\n", password)
		}
		if config, ok := data["config"].(map[string]interface{}); ok {
			if configURL, ok := config["config_url"]; ok {
				text += fmt.Sprintf("• 📄 Config URL: %v\n", configURL)
			}
			if expiredOn, ok := config["expired_on"]; ok {
				text += fmt.Sprintf("• ⏰ Expired: %v\n", expiredOn)
			}
			if host, ok := config["host"]; ok {
				text += fmt.Sprintf("• 🏠 Host: `%v`\n", host)
			}
			if network, ok := config["network"]; ok {
				text += fmt.Sprintf("• 🌐 Network: %v\n", network)
			}
			if path, ok := config["path"]; ok {
				text += fmt.Sprintf("• 📁 Path: `%v`\n", path)
			}
			if serviceName, ok := config["serviceName"]; ok {
				text += fmt.Sprintf("• 🔧 Service Name: `%v`\n", serviceName)
			}
			
			text += "\n🔗 *Connection Links:*\n"
			if linkWs, ok := config["link_ws"]; ok {
				text += fmt.Sprintf("• WebSocket: `%v`\n", linkWs)
			}
			if linkGrpc, ok := config["link_grpc"]; ok {
				text += fmt.Sprintf("• gRPC: `%v`\n", linkGrpc)
			}
			if linkGo, ok := config["link_go"]; ok {
				text += fmt.Sprintf("• Trojan-Go: `%v`\n", linkGo)
			}
		}
		
	case "vless":
		text += "🔧 *Konfigurasi VLESS:*\n"
		if server, ok := data["server"]; ok {
			text += fmt.Sprintf("• 🌐 Server: `%v`\n", server)
		}
		if port, ok := data["port"]; ok {
			text += fmt.Sprintf("• 🔌 Port: `%v`\n", port)
		}
		if uuid, ok := data["uuid"]; ok {
			text += fmt.Sprintf("• 🆔 UUID: `%v`\n", uuid)
		}
		if config, ok := data["config"].(map[string]interface{}); ok {
			if configURL, ok := config["config_url"]; ok {
				text += fmt.Sprintf("• 📄 Config URL: %v\n", configURL)
			}
			if expiredOn, ok := config["expired_on"]; ok {
				text += fmt.Sprintf("• ⏰ Expired: %v\n", expiredOn)
			}
			if host, ok := config["host"]; ok {
				text += fmt.Sprintf("• 🏠 Host: `%v`\n", host)
			}
			if encryption, ok := config["encryption"]; ok {
				text += fmt.Sprintf("• 🔐 Encryption: %v\n", encryption)
			}
			if network, ok := config["network"]; ok {
				text += fmt.Sprintf("• 🌐 Network: %v\n", network)
			}
			if path, ok := config["path"]; ok {
				text += fmt.Sprintf("• 📁 Path: `%v`\n", path)
			}
			if portNtls, ok := config["port_ntls"]; ok {
				text += fmt.Sprintf("• 🔌 Port NTLS: `%v`\n", portNtls)
			}
			if portTls, ok := config["port_tls"]; ok {
				text += fmt.Sprintf("• 🔌 Port TLS: `%v`\n", portTls)
			}
			if serviceName, ok := config["serviceName"]; ok {
				text += fmt.Sprintf("• 🔧 Service Name: `%v`\n", serviceName)
			}
			
			text += "\n🔗 *Connection Links:*\n"
			if linkTls, ok := config["link_tls"]; ok {
				text += fmt.Sprintf("• TLS: `%v`\n", linkTls)
			}
			if linkNtls, ok := config["link_ntls"]; ok {
				text += fmt.Sprintf("• NTLS: `%v`\n", linkNtls)
			}
			if linkGrpc, ok := config["link_grpc"]; ok {
				text += fmt.Sprintf("• gRPC: `%v`\n", linkGrpc)
			}
		}
		
	case "vmess":
		text += "🔧 *Konfigurasi VMESS:*\n"
		if server, ok := data["server"]; ok {
			text += fmt.Sprintf("• 🌐 Server: `%v`\n", server)
		}
		if port, ok := data["port"]; ok {
			text += fmt.Sprintf("• 🔌 Port: `%v`\n", port)
		}
		if uuid, ok := data["uuid"]; ok {
			text += fmt.Sprintf("• 🆔 UUID: `%v`\n", uuid)
		}
		if config, ok := data["config"].(map[string]interface{}); ok {
			if configURL, ok := config["config_url"]; ok {
				text += fmt.Sprintf("• 📄 Config URL: %v\n", configURL)
			}
			if expiredOn, ok := config["expired_on"]; ok {
				text += fmt.Sprintf("• ⏰ Expired: %v\n", expiredOn)
			}
			if host, ok := config["host"]; ok {
				text += fmt.Sprintf("• 🏠 Host: `%v`\n", host)
			}
			if alterId, ok := config["alterId"]; ok {
				text += fmt.Sprintf("• 🔢 Alter ID: %v\n", alterId)
			}
			if security, ok := config["security"]; ok {
				text += fmt.Sprintf("• 🔐 Security: %v\n", security)
			}
			if network, ok := config["network"]; ok {
				text += fmt.Sprintf("• 🌐 Network: %v\n", network)
			}
			if path, ok := config["path"]; ok {
				text += fmt.Sprintf("• 📁 Path: `%v`\n", path)
			}
			if serviceName, ok := config["serviceName"]; ok {
				text += fmt.Sprintf("• 🔧 Service Name: `%v`\n", serviceName)
			}
			
			text += "\n🔗 *Connection Links:*\n"
			if linkWs, ok := config["link_ws"]; ok {
				text += fmt.Sprintf("• WebSocket: `%v`\n", linkWs)
			}
			if linkGrpc, ok := config["link_grpc"]; ok {
				text += fmt.Sprintf("• gRPC: `%v`\n", linkGrpc)
			}
		}
	}
	
	return text
}

// formatVPNConfigFromDB formats VPN configuration from database
func formatVPNConfigFromDB(protocol string, config map[string]interface{}, uuid string) string {
	var text string
	
	switch protocol {
	case "ssh":
		text += "🔧 *Konfigurasi SSH/SSL:*\n"
		if sslPort, ok := config["ssl_port"]; ok {
			text += fmt.Sprintf("• 🔒 SSL Port: `%v`\n", sslPort)
		}
		if stunnelPort, ok := config["stunnel_port"]; ok {
			text += fmt.Sprintf("• 🔐 Stunnel Port: `%v`\n", stunnelPort)
		}
		if wsPort, ok := config["ws_port"]; ok {
			text += fmt.Sprintf("• 🌐 WebSocket Port: `%v`\n", wsPort)
		}
		
	case "trojan":
		text += "🔧 *Konfigurasi Trojan:*\n"
		if uuid != "" {
			text += fmt.Sprintf("• 🔑 Key: `%s`\n", uuid)
		}
		if configURL, ok := config["config_url"]; ok {
			text += fmt.Sprintf("• 📄 Config URL: %v\n", configURL)
		}
		if expiredOn, ok := config["expired_on"]; ok {
			text += fmt.Sprintf("• ⏰ Expired: %v\n", expiredOn)
		}
		if host, ok := config["host"]; ok {
			text += fmt.Sprintf("• 🏠 Host: `%v`\n", host)
		}
		if network, ok := config["network"]; ok {
			text += fmt.Sprintf("• 🌐 Network: %v\n", network)
		}
		if path, ok := config["path"]; ok {
			text += fmt.Sprintf("• 📁 Path: `%v`\n", path)
		}
		if serviceName, ok := config["serviceName"]; ok {
			text += fmt.Sprintf("• 🔧 Service Name: `%v`\n", serviceName)
		}
		
		text += "\n🔗 *Connection Links:*\n"
		if linkWs, ok := config["link_ws"]; ok {
			text += fmt.Sprintf("• WebSocket: `%v`\n", linkWs)
		}
		if linkGrpc, ok := config["link_grpc"]; ok {
			text += fmt.Sprintf("• gRPC: `%v`\n", linkGrpc)
		}
		if linkGo, ok := config["link_go"]; ok {
			text += fmt.Sprintf("• Trojan-Go: `%v`\n", linkGo)
		}
		
	case "vless":
		text += "🔧 *Konfigurasi VLESS:*\n"
		if uuid != "" {
			text += fmt.Sprintf("• 🆔 UUID: `%s`\n", uuid)
		}
		if configURL, ok := config["config_url"]; ok {
			text += fmt.Sprintf("• 📄 Config URL: %v\n", configURL)
		}
		if expiredOn, ok := config["expired_on"]; ok {
			text += fmt.Sprintf("• ⏰ Expired: %v\n", expiredOn)
		}
		if host, ok := config["host"]; ok {
			text += fmt.Sprintf("• 🏠 Host: `%v`\n", host)
		}
		if encryption, ok := config["encryption"]; ok {
			text += fmt.Sprintf("• 🔐 Encryption: %v\n", encryption)
		}
		if network, ok := config["network"]; ok {
			text += fmt.Sprintf("• 🌐 Network: %v\n", network)
		}
		if path, ok := config["path"]; ok {
			text += fmt.Sprintf("• 📁 Path: `%v`\n", path)
		}
		if portNtls, ok := config["port_ntls"]; ok {
			text += fmt.Sprintf("• 🔌 Port NTLS: `%v`\n", portNtls)
		}
		if portTls, ok := config["port_tls"]; ok {
			text += fmt.Sprintf("• 🔌 Port TLS: `%v`\n", portTls)
		}
		if serviceName, ok := config["serviceName"]; ok {
			text += fmt.Sprintf("• 🔧 Service Name: `%v`\n", serviceName)
		}
		
		text += "\n🔗 *Connection Links:*\n"
		if linkTls, ok := config["link_tls"]; ok {
			text += fmt.Sprintf("• TLS: `%v`\n", linkTls)
		}
		if linkNtls, ok := config["link_ntls"]; ok {
			text += fmt.Sprintf("• NTLS: `%v`\n", linkNtls)
		}
		if linkGrpc, ok := config["link_grpc"]; ok {
			text += fmt.Sprintf("• gRPC: `%v`\n", linkGrpc)
		}
		
	case "vmess":
		text += "🔧 *Konfigurasi VMESS:*\n"
		if uuid != "" {
			text += fmt.Sprintf("• 🆔 UUID: `%s`\n", uuid)
		}
		if configURL, ok := config["config_url"]; ok {
			text += fmt.Sprintf("• 📄 Config URL: %v\n", configURL)
		}
		if expiredOn, ok := config["expired_on"]; ok {
			text += fmt.Sprintf("• ⏰ Expired: %v\n", expiredOn)
		}
		if host, ok := config["host"]; ok {
			text += fmt.Sprintf("• 🏠 Host: `%v`\n", host)
		}
		if alterId, ok := config["alterId"]; ok {
			text += fmt.Sprintf("• 🔢 Alter ID: %v\n", alterId)
		}
		if security, ok := config["security"]; ok {
			text += fmt.Sprintf("• 🔐 Security: %v\n", security)
		}
		if network, ok := config["network"]; ok {
			text += fmt.Sprintf("• 🌐 Network: %v\n", network)
		}
		if path, ok := config["path"]; ok {
			text += fmt.Sprintf("• 📁 Path: `%v`\n", path)
		}
		if serviceName, ok := config["serviceName"]; ok {
			text += fmt.Sprintf("• 🔧 Service Name: `%v`\n", serviceName)
		}
		
		text += "\n🔗 *Connection Links:*\n"
		if linkWs, ok := config["link_ws"]; ok {
			text += fmt.Sprintf("• WebSocket: `%v`\n", linkWs)
		}
		if linkGrpc, ok := config["link_grpc"]; ok {
			text += fmt.Sprintf("• gRPC: `%v`\n", linkGrpc)
		}
	}
	
	return text
}
