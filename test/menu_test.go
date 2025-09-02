package test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const pageSize = 10 // jumlah produk per halaman

type ApiResponse struct {
	StatusCode int       `json:"statusCode"`
	Message    string    `json:"message"`
	Success    bool      `json:"success"`
	Data       []Package `json:"data"`
}

type Package struct {
	PackageCode string `json:"package_code"`
	PackageName string `json:"package_name"`
	Price       int64  `json:"package_harga_int"`
}

var packageMap = map[string]Package{}

func fetchPackages() ([]Package, error) {
	url := "https://grnstore.domcloud.dev/api/packages?limit=100"

	req, err := http.NewRequest("GET", url, nil)
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

	var apiResp ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func TestMenu(t *testing.T) {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		token = "7781367281:AAHyLZVfhAgb0M0b5HQuY_VlSz0tsE9FbDw"
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatal(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					handleStart(bot, update.Message.Chat.ID)
				case "products":
					sendProductList(bot, update.Message.Chat.ID, 0)
				default:
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Perintah tidak dikenal. Coba /products"))
				}
			}
		}

		if update.CallbackQuery != nil {
			data := update.CallbackQuery.Data
			if strings.HasPrefix(data, "buy:") {
				productID := strings.TrimPrefix(data, "buy:")
				handleBuy(bot, update.CallbackQuery, productID)
			} else if strings.HasPrefix(data, "page:") {
				pageStr := strings.TrimPrefix(data, "page:")
				page, _ := strconv.Atoi(pageStr)
				sendProductList(bot, update.CallbackQuery.Message.Chat.ID, page)
				// hapus pesan lama supaya chat tidak penuh
				bot.Request(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID))
			}
		}
	}
}

func handleStart(bot *tgbotapi.BotAPI, chatID int64) {
	text := "Halo! 👋\nSelamat datang di Bot Shop.\nKetik /products untuk lihat produk."
	bot.Send(tgbotapi.NewMessage(chatID, text))
}

func sendProductList(bot *tgbotapi.BotAPI, chatID int64, page int) {
	packages, err := fetchPackages()
	if err != nil {
		bot.Send(tgbotapi.NewMessage(chatID, "Gagal ambil data."))
		return
	}

	total := len(packages)
	start := page * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, p := range packages[start:end] {
		packageMap[p.PackageCode] = p
		btn := tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s - Rp%d", p.PackageName, p.Price), "buy:"+p.PackageCode)
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(btn))
	}

	// Tombol navigasi
	var navButtons []tgbotapi.InlineKeyboardButton
	if start > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅️ Prev", fmt.Sprintf("page:%d", page-1)))
	}
	if end < total {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Next ➡️", fmt.Sprintf("page:%d", page+1)))
	}
	if len(navButtons) > 0 {
		rows = append(rows, navButtons)
	}

	kb := tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Pilih produk (halaman %d/%d):", page+1, (total+pageSize-1)/pageSize))
	msg.ReplyMarkup = kb
	bot.Send(msg)
}

func handleBuy(bot *tgbotapi.BotAPI, cq *tgbotapi.CallbackQuery, code string) {
	p, ok := packageMap[code]
	if !ok {
		bot.Request(tgbotapi.NewCallback(cq.ID, "Produk tidak ditemukan."))
		return
	}
	bot.Request(tgbotapi.NewCallback(cq.ID, "Produk dipilih."))

	text := fmt.Sprintf("✅ Kamu memilih *%s*\nHarga: Rp%d\n\nSegera lakukan pembayaran!", p.PackageName, p.Price)
	msg := tgbotapi.NewMessage(cq.Message.Chat.ID, text)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
