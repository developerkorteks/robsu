package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/nabilulilalbab/bottele/config"
	"github.com/nabilulilalbab/bottele/internal/bot"
	"github.com/nabilulilalbab/bottele/service"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize database
	config.ConnectDatabase()

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

	for update := range updates {
		bot.HandleUpdate(botAPI, update)
	}
}
