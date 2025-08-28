# 💰 Panduan Sistem Top-Up QRIS - GRN Store Bot

## 🎯 Overview

Sistem top-up dengan QRIS dinamis yang terintegrasi dengan bot Telegram untuk approval admin dan notifikasi WhatsApp.

## 🔄 Flow Lengkap

```
User Request Top-Up → QRIS Dinamis Generated → DB Status=Pending
                                    ↓
                            Admin Telegram Panel
                        ┌─────────────────────────┐
                        │ /pending - Lihat Daftar │
                        │ /confirm <txn_id> - ACC │
                        │ /reject <txn_id> - Tolak│
                        └─────────┬───────────────┘
                                  ↓
                        Bot Update Database:
                        - status = confirmed
                        - saldo += nominal  
                        - approved_by = adminID
                        - approved_at = timestamp
                                  ↓
                    ┌─────────────────────────────┐
                    │ Notifikasi Telegram User    │
                    │ Notifikasi WhatsApp Admin   │
                    └─────────────────────────────┘
```

## 📱 Fitur User

### 1. **Top Up Saldo**
- **Akses**: Menu Utama → "💰 Top Up Saldo"
- **Minimal**: Rp 10.000
- **Maksimal**: Rp 1.000.000
- **Expired**: 30 menit

### 2. **Cek Saldo**
- **Akses**: Menu Utama → "💳 Cek Saldo"
- **Info**: Saldo real-time

### 3. **QRIS Dinamis**
- QR Code otomatis generated dengan nominal sesuai request
- Berlaku 30 menit
- Terintegrasi dengan e-wallet (GoPay, OVO, DANA, dll)

## 👨‍💼 Fitur Admin

### 1. **Lihat Pending Transactions**
```bash
/pending
```
**Response:**
```
📋 Pending Top-Up Transactions

Daftar transaksi yang menunggu konfirmasi:

1. John Doe (@johndoe)
   💳 Nominal: Rp 50.000
   🆔 ID: TXN_123456789_1234567890
   ⏰ Expired: 2024-01-02 15:30:00

Command untuk konfirmasi:
• /confirm <transaction_id> - ACC transaksi
• /reject <transaction_id> - Tolak transaksi
```

### 2. **Konfirmasi Top-Up**
```bash
/confirm TXN_123456789_1234567890
```
**Hasil:**
- ✅ Saldo user bertambah
- 📱 Notifikasi ke user
- 📞 Notifikasi WhatsApp ke admin
- 📊 Update database

### 3. **Tolak Top-Up**
```bash
/reject TXN_123456789_1234567890
```
**Hasil:**
- ❌ Transaksi ditolak
- 📱 Notifikasi ke user
- 📊 Update database

## 🔧 Technical Implementation

### **QRIS Dinamis Generator**
```go
// Generate QRIS dengan nominal dinamis
qrisCode, err := service.GenerateDynamicQRIS(amount)

// Generate QR Code image
qrBytes, err := service.GenerateQRCodeBytes(qrisCode)
```

### **Database Structure (In-Memory)**
```go
type Transaction struct {
    ID           string `json:"id"`
    UserID       int64  `json:"user_id"`
    Username     string `json:"username"`
    Amount       int64  `json:"amount"`
    Status       string `json:"status"` // pending, confirmed, rejected, expired
    QRISCode     string `json:"qris_code"`
    CreatedAt    string `json:"created_at"`
    ApprovedBy   int64  `json:"approved_by,omitempty"`
    ApprovedAt   string `json:"approved_at,omitempty"`
    ExpiredAt    string `json:"expired_at"`
}

type UserBalance struct {
    UserID  int64 `json:"user_id"`
    Balance int64 `json:"balance"`
}
```

## 📋 Command Reference

### **User Commands**
| Command | Deskripsi |
|---------|-----------|
| `/start` | Menu utama |
| `/balance` | Cek saldo |
| `💰 Top Up Saldo` | Request top-up |
| `💳 Cek Saldo` | Lihat saldo |

### **Admin Commands**
| Command | Deskripsi | Format |
|---------|-----------|--------|
| `/pending` | Lihat transaksi pending | `/pending` |
| `/confirm` | ACC top-up | `/confirm <transaction_id>` |
| `/reject` | Tolak top-up | `/reject <transaction_id>` |
| `/admin` | Panel admin | `/admin` |

## 🔔 Notifikasi

### **Telegram User (Top-Up Berhasil)**
```
✅ Top-Up Berhasil!

💳 Nominal: Rp 50.000
💰 Saldo Anda sekarang: Rp 75.000

Terima kasih! Saldo Anda telah berhasil ditambahkan.
Sekarang Anda dapat membeli paket data di GRN Store.
```

### **WhatsApp Admin**
```
Top-up berhasil:
User: John Doe (123456789)
Nominal: Rp 50.000
Saldo sekarang: Rp 75.000
```

## 🛡️ Security & Validation

### **Input Validation**
- ✅ Nominal minimal/maksimal
- ✅ Format angka only
- ✅ Admin authorization
- ✅ Transaction ID validation

### **Transaction Security**
- ✅ Expired time (30 menit)
- ✅ Status tracking
- ✅ Audit trail (approved_by, approved_at)
- ✅ Duplicate prevention

### **Error Handling**
- ✅ User-friendly error messages
- ✅ Admin error logging
- ✅ Graceful failure handling

## 🚀 Usage Examples

### **Scenario 1: User Top-Up Success**
1. User klik "💰 Top Up Saldo"
2. User input "50000"
3. Bot generate QRIS + QR Code
4. User scan & bayar via e-wallet
5. Admin terima notifikasi
6. Admin `/confirm TXN_xxx`
7. User saldo +50k, dapat notifikasi

### **Scenario 2: Admin Reject**
1. User request top-up
2. Admin lihat `/pending`
3. Admin `/reject TXN_xxx` (misal: nominal tidak sesuai)
4. User dapat notifikasi penolakan
5. User bisa hubungi admin atau coba lagi

### **Scenario 3: Expired Transaction**
1. User request top-up
2. User tidak bayar dalam 30 menit
3. Status otomatis berubah ke "expired"
4. Tidak muncul di `/pending`
5. User harus request ulang

## 📊 Monitoring & Analytics

### **Admin Dashboard**
- 📋 Pending transactions count
- 📈 Daily top-up volume
- 👥 Active users
- 💰 Total revenue

### **Logs**
- 📝 All transactions logged
- 🔍 Admin actions tracked
- ⚠️ Error monitoring
- 📞 WhatsApp delivery status

## 🔄 Integration Points

### **WhatsApp API**
```bash
curl -X POST http://128.199.109.211:25120/send-message \
  -H "Content-Type: application/json" \
  -d '{
    "number": "6285150588080",
    "message": "Top-up berhasil:\nUser: @username\nNominal: Rp50000"
  }'
```

### **QRIS Integration**
- ✅ Dynamic amount injection
- ✅ CRC16 calculation
- ✅ QR Code generation
- ✅ E-wallet compatibility

## 🎯 Best Practices

1. **Admin Management**
   - Hanya admin terdaftar yang bisa ACC
   - Semua aksi admin tercatat
   - Timeout untuk keamanan

2. **Transaction Handling**
   - Expired otomatis untuk mencegah konflik
   - Status tracking yang jelas
   - Audit trail lengkap

3. **User Experience**
   - Error message yang jelas
   - Progress indicator
   - Multiple payment options

4. **Monitoring**
   - Real-time notification
   - Log semua aktivitas
   - Performance tracking

---

**GRN Store Top-Up System** - Sistem top-up yang aman, cepat, dan user-friendly! 💰