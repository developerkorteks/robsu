# 🏪 GRN Store Telegram Bot

Bot Telegram profesional untuk penjualan paket data dan kuota internet dengan sistem verifikasi OTP yang aman.

## ✨ Fitur Utama

- 📱 **Katalog Produk Lengkap**: Menampilkan daftar paket data dari semua operator
- 🔒 **Verifikasi OTP**: Sistem keamanan dengan verifikasi nomor HP melalui SMS
- 💰 **Format Harga Profesional**: Tampilan harga dengan pemisah ribuan
- 🎯 **Menu Interaktif**: Interface yang user-friendly dengan inline keyboard
- ⚡ **Error Handling**: Penanganan error yang tidak mengekspos detail teknis ke user
- 📊 **Pagination**: Navigasi halaman untuk daftar produk yang banyak
- 👨‍💼 **Sistem Admin**: Panel admin dengan statistik dan monitoring pesan
- 📩 **Contact Admin**: User bisa mengirim pesan langsung ke admin

## 🚀 Cara Menjalankan

1. **Setup Environment**
   ```bash
   # Edit .env dan isi konfigurasi
   TELEGRAM_TOKEN=your_bot_token
   ADMIN_CHAT_ID=your_admin_chat_id
   ADMIN_USERNAME=your_admin_username
   ```

2. **Install Dependencies**
   ```bash
   go mod tidy
   ```

3. **Build & Run**
   ```bash
   go build -o bot cmd/main.go
   ./bot
   ```

## 📋 Struktur Menu Bot

### Menu Utama
- 📱 **Lihat Produk** - Browse katalog paket data
- 📞 **Verifikasi Nomor** - Verifikasi HP dengan OTP
- ℹ️ **Bantuan** - Informasi cara penggunaan

### Flow Pembelian
1. User memilih "Verifikasi Nomor"
2. Input nomor HP (format: 08xxxxxxxxxx)
3. Menerima dan input kode OTP
4. Setelah terverifikasi, bisa memilih produk
5. Konfirmasi pembelian dan lanjut ke pembayaran

## 🔧 Konfigurasi API

Bot menggunakan API GRN Store dengan endpoint:

- **OTP Request**: `POST /api/otp/request`
- **OTP Verify**: `POST /api/otp/verify` 
- **Products**: `GET /api/packages`

API Key: `nadia-admin-2024-secure-key`

## 📁 Struktur Project

```
├── cmd/
│   └── main.go              # Entry point aplikasi
├── config/
│   └── config.go            # Konfigurasi bot
├── dto/
│   ├── request.go           # Request DTOs
│   └── response.go          # Response DTOs
├── internal/bot/
│   ├── handler.go           # Handler utama bot
│   └── state.go             # State management user
├── service/
│   ├── otp_service.go       # Service untuk OTP
│   └── package_service.go   # Service untuk produk
└── .env                     # Environment variables
```

## 🛡️ Keamanan

- ✅ Validasi format nomor HP Indonesia
- ✅ Validasi format kode OTP (4-6 digit)
- ✅ Error handling yang aman (tidak expose internal error)
- ✅ State management per user dengan mutex
- ✅ Normalisasi input nomor HP

## 🎨 User Experience

- **Professional Design**: Menu dengan emoji dan formatting yang menarik
- **Clear Navigation**: Tombol kembali di setiap halaman
- **Helpful Messages**: Pesan error yang informatif dan solusi yang jelas
- **Responsive**: Feedback langsung untuk setiap aksi user

## 📞 Format Nomor HP yang Didukung

- `08xxxxxxxxxx` (format lokal)
- `+628xxxxxxxxxx` (format internasional)
- `628xxxxxxxxxx` (tanpa +)

Semua format akan dinormalisasi ke format `08xxxxxxxxxx`.

## 🔄 State Management

Bot menggunakan state management untuk tracking:
- `start` - State awal
- `waiting_phone` - Menunggu input nomor HP
- `waiting_otp` - Menunggu input kode OTP
- `verified` - User sudah terverifikasi

## 🚀 Next Steps

Untuk implementasi selanjutnya, Anda bisa menambahkan:
1. Endpoint verifikasi OTP (jika belum ada)
2. Sistem pembayaran
3. Notifikasi status transaksi
4. History pembelian
5. Customer support integration

---

**GRN Store** - Toko terpercaya untuk kebutuhan paket data Anda! 🏪