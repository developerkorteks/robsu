# 👨‍💼 Panduan Admin Bot GRN Store

## 🔧 Setup Admin

### 1. Konfigurasi Admin di .env
```env
# Admin Configuration
ADMIN_CHAT_ID=123456789
ADMIN_USERNAME=your_admin_username
```

### 2. Cara Mendapatkan Chat ID Admin
1. **Metode 1: Melalui Bot**
   - Kirim pesan `/start` ke bot
   - Admin kirim pesan apa saja ke bot
   - Check log bot untuk melihat Chat ID

2. **Metode 2: Melalui @userinfobot**
   - Forward pesan admin ke @userinfobot
   - Bot akan memberikan informasi Chat ID

3. **Metode 3: Melalui API Telegram**
   ```bash
   curl https://api.telegram.org/bot<BOT_TOKEN>/getUpdates
   ```

## 📋 Fitur Admin

### 🎛️ Panel Admin
- **Command**: `/admin`
- **Akses**: Hanya admin yang terdaftar
- **Fitur**:
  - 📊 Statistik Bot
  - 📢 Broadcast Message (coming soon)

### 📊 Statistik Bot
- **Command**: `/stats` atau tombol "📊 Statistik Bot"
- **Info yang ditampilkan**:
  - Total user bot
  - User yang sudah terverifikasi
  - Total transaksi
  - Total penjualan

### 📩 Sistem Pesan ke Admin

#### Dari User ke Admin:
1. User klik "👨‍💼 Hubungi Admin"
2. User ketik pesan
3. Pesan otomatis dikirim ke admin dengan format:
   ```
   📩 Pesan dari User
   
   👤 User: John Doe (@johndoe)
   🆔 User ID: 123456789
   🕐 Waktu: 02/01/2024 15:04:05
   
   💬 Pesan:
   [Pesan dari user]
   ```

#### Notifikasi Pesanan Baru:
Admin akan menerima notifikasi otomatis saat ada pesanan baru:
```
🛒 Pesanan Baru!

👤 Customer: 123456789
📱 Nomor: 087817739901
📦 Produk: Paket Data 10GB
💰 Harga: Rp 50.000

⏰ Waktu: Sekarang
```

## 🔐 Keamanan Admin

### Validasi Admin
- Bot memvalidasi Chat ID admin sebelum memberikan akses
- Hanya admin yang terdaftar di `.env` yang bisa mengakses fitur admin
- Error handling yang aman (tidak expose informasi sensitif)

### Log Security
- Semua akses admin dicatat di log
- Error admin dicatat untuk monitoring
- Panic recovery untuk stabilitas bot

## 📱 Command Admin

| Command | Deskripsi | Akses |
|---------|-----------|-------|
| `/admin` | Panel admin utama | Admin only |
| `/stats` | Statistik bot | Admin only |
| `/start` | Menu utama (sama seperti user) | Semua |
| `/help` | Bantuan (sama seperti user) | Semua |

## 🔄 Workflow Admin

### 1. Setup Awal
```bash
# 1. Edit .env
ADMIN_CHAT_ID=YOUR_CHAT_ID
ADMIN_USERNAME=your_username

# 2. Restart bot
./bot
```

### 2. Akses Panel Admin
1. Kirim `/admin` ke bot
2. Pilih menu yang diinginkan
3. Bot akan menampilkan informasi/opsi admin

### 3. Monitoring Pesan User
- Admin akan menerima semua pesan dari user secara real-time
- Format pesan sudah terstruktur dengan informasi lengkap user
- Admin bisa reply langsung ke user (manual)

### 4. Monitoring Pesanan
- Notifikasi otomatis saat ada pesanan baru
- Informasi lengkap customer dan produk
- Admin bisa follow up manual untuk pembayaran

## 🚀 Fitur Mendatang

### 📢 Broadcast Message
- Kirim pesan ke semua user
- Filter user berdasarkan status (verified/unverified)
- Laporan delivery status

### 📈 Advanced Analytics
- Grafik penjualan harian/bulanan
- Top selling products
- User engagement metrics
- Revenue tracking

### 🤖 Auto Response
- Template response untuk pertanyaan umum
- Auto-reply untuk jam non-operasional
- FAQ integration

## 🛠️ Troubleshooting

### Admin Tidak Bisa Akses
1. **Check Chat ID**: Pastikan `ADMIN_CHAT_ID` benar
2. **Check Format**: Chat ID harus berupa angka (tanpa quotes)
3. **Restart Bot**: Restart bot setelah mengubah `.env`

### Pesan User Tidak Masuk
1. **Check Admin Chat ID**: Pastikan admin chat ID valid
2. **Check Bot Permission**: Bot harus bisa kirim pesan ke admin
3. **Check Log**: Lihat log error di console

### Bot Tidak Respond
1. **Check Token**: Pastikan `TELEGRAM_TOKEN` valid
2. **Check Network**: Pastikan koneksi internet stabil
3. **Check Log**: Lihat error di console

## 📞 Support

Jika mengalami masalah dengan fitur admin:
1. Check log bot di console
2. Pastikan konfigurasi `.env` benar
3. Restart bot setelah perubahan konfigurasi
4. Test dengan user lain untuk isolasi masalah

---

**GRN Store Bot Admin Panel** - Kelola bot dengan mudah dan profesional! 👨‍💼