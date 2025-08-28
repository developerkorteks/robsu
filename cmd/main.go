package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/nabilulilalbab/bottele/api"
	"github.com/nabilulilalbab/bottele/config"
	"github.com/nabilulilalbab/bottele/internal/bot"
	"github.com/nabilulilalbab/bottele/service"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8253"
	}
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize database
	config.ConnectDatabase()

	// Load existing transactions from database to in-memory storage
	if err := service.LoadTransactionsFromDatabase(); err != nil {
		log.Printf("Warning: Failed to load transactions from database: %v", err)
	}

	// Start cleanup routine for transaction locks
	service.StartCleanupRoutine()

	// Sekarang, panggil fungsi Anda seperti biasa
	// os.Getenv() akan berhasil menemukan variabelnya
	botToken := config.GetBotToken()

	log.Printf("Bot token berhasil didapatkan: %s", botToken)
	token := config.GetBotToken()
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("Gagal inisialisasi bot: %v", err)
	}

	// Store bot instance for admin notifications
	config.BotInstance = botAPI

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := botAPI.GetUpdatesChan(u)

	log.Printf("Bot berjalan sebagai %s", botAPI.Self.UserName)

	// Setup bot commands menu
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "🏠 Mulai menggunakan bot"},
		{Command: "menu", Description: "📋 Tampilkan menu utama"},
		{Command: "products", Description: "🛍️ Lihat semua produk"},
		{Command: "balance", Description: "💰 Cek saldo"},
		{Command: "topup", Description: "💳 Top up saldo"},
		{Command: "history", Description: "📜 Riwayat transaksi"},
		{Command: "help", Description: "❓ Bantuan dan panduan"},
		{Command: "rules", Description: "📋 Peraturan bot"},
	}

	setCommands := tgbotapi.NewSetMyCommands(commands...)
	if _, err := botAPI.Request(setCommands); err != nil {
		log.Printf("Error setting bot commands: %v", err)
	} else {
		log.Printf("Bot commands menu berhasil diatur")
	}

	// Setup API server
	go func() {
		router := gin.Default()

		// Setup CORS middleware
		router.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}

			c.Next()
		})

		// Setup API routes
		api.SetupRoutes(router)

		log.Printf("API Server starting on port %s...", port)
		if err := router.Run(":" + port); err != nil {
			log.Printf("Failed to start API server: %v", err)
		}
	}()

	for update := range updates {
		bot.HandleUpdate(botAPI, update)
	}
}
