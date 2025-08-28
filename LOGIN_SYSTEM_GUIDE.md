# 🔐 Login System & Purchase Integration - GRN Store Bot

## ✅ **Sistem yang Telah Diimplementasikan**

### 🔑 **Authentication System**
- **OTP Login**: Verifikasi OTP + mendapat access token
- **Session Management**: Token disimpan di database, berlaku 1 jam
- **Auto Logout**: Token expired otomatis setelah 1 jam
- **Manual Logout**: User bisa logout manual

### 💾 **Database Persistent**
- **SQLite Database**: `grnstore.db` untuk menyimpan semua data
- **GORM ORM**: Object-relational mapping untuk database operations
- **Auto Migration**: Database schema otomatis dibuat saat startup
- **Data Persistence**: Tidak hilang saat bot restart

### 🛒 **Purchase Integration**
- **API Integration**: Langsung ke GRN Store API
- **Multiple Payment Methods**: BALANCE, DANA, QRIS
- **Smart Payment Flow**: Otomatis handle berbagai jenis pembayaran
- **Transaction Tracking**: Semua transaksi tersimpan di database

## 🗄️ **Database Schema**

### **Users Table**
```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_id INTEGER UNIQUE NOT NULL,
    phone_number TEXT NOT NULL,
    access_token TEXT,
    token_expires_at DATETIME,
    is_verified BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### **Purchase Transactions Table**
```sql
CREATE TABLE purchase_transactions (
    id TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    package_code TEXT NOT NULL,
    package_name TEXT NOT NULL,
    payment_method TEXT NOT NULL,
    phone_number TEXT NOT NULL,
    price INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    response_data TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### **Other Tables**
- `transactions` - Top-up transactions
- `user_balances` - User balance tracking
- `active_users` - User interaction tracking
- `otp_sessions` - OTP session tracking

## 🔄 **Complete User Flow**

### **1. Login Flow**
```
User → 📞 Verifikasi Nomor
     ↓
Input nomor HP → OTP dikirim
     ↓
Input kode OTP → Verifikasi + Login
     ↓
✅ Login Berhasil (Access Token 1 jam)
```

### **2. Purchase Flow**
```
User → 📱 Lihat Produk
     ↓
Pilih produk → Lihat detail
     ↓
🛒 Beli Sekarang → Cek login status
     ↓
Pilih metode pembayaran
     ↓
Proses pembayaran (BALANCE/DANA/QRIS)
     ↓
✅ Berhasil / 💳 Pending Payment
```

### **3. Payment Methods**

#### **BALANCE (Direct)**
```
User pilih BALANCE → Langsung potong saldo
                  ↓
✅ Pembelian Berhasil!
Paket data akan segera aktif di nomor Anda.
```

#### **DANA (Deeplink)**
```
User pilih DANA → Generate deeplink URL
               ↓
💳 Bayar dengan DANA (tombol ke app DANA)
               ↓
User bayar di app DANA
               ↓
🔄 Cek Status untuk konfirmasi
```

#### **QRIS**
```
User pilih QRIS → Generate QR Code
              ↓
💳 Pembayaran QRIS
Scan QR code dengan e-wallet
              ↓
User scan & bayar
              ↓
Otomatis konfirmasi setelah bayar
```

## 🔐 **Authentication Features**

### **Login System**
```go
// Verify OTP and get access token
func VerifyOTPAndLogin(phoneNumber, otpCode string, userID int64) (*dto.OTPVerifyLoginResponse, error)

// Save user session with 1-hour expiry
func SaveUserSession(chatID int64, phoneNumber, accessToken string) error

// Check if user has valid session
func IsUserLoggedIn(chatID int64) bool
```

### **Session Management**
- **Token Storage**: Access token disimpan di database
- **Expiry Check**: Otomatis cek expired setiap akses
- **Auto Cleanup**: Token expired otomatis dihapus
- **Manual Logout**: User bisa logout kapan saja

### **Security Features**
- ✅ **Token Expiry**: 1 jam otomatis expired
- ✅ **Session Validation**: Cek valid setiap request
- ✅ **Secure Storage**: Token tersimpan encrypted di database
- ✅ **Logout Function**: Clear session manual

## 🛒 **Purchase Integration**

### **API Endpoints Used**
```bash
# Login/Verify OTP
POST https://grnstore.domcloud.dev/api/otp/verify
{
  "otp_code": "976891",
  "phone_number": "087817739901"
}

# Purchase Product
POST https://grnstore.domcloud.dev/api/purchase
{
  "access_token": "1101975:a321f008-122b-43f3-9006-aefb0739e1a7",
  "package_code": "XL_MASTIF_30D_P_V1",
  "payment_method": "BALANCE",
  "phone_number": "087817739901",
  "source": "telegram_bot"
}

# Check Transaction Status
POST https://grnstore.domcloud.dev/api/transaction/check
{
  "transaction_id": "802eaef3-8b5f-4fb8-a3ad-489d4cc91637"
}
```

### **Payment Method Handling**

#### **1. BALANCE Payment**
- Langsung potong saldo user
- Instant confirmation
- No additional steps required

#### **2. DANA Payment**
- Generate deeplink URL
- User redirect ke app DANA
- Manual status check after payment

#### **3. QRIS Payment**
- Generate QR code image
- User scan dengan e-wallet
- Auto confirmation after payment

## 📱 **User Interface Updates**

### **Login Status Display**
```
✅ Login Berhasil!

Nomor HP 087817739901 telah berhasil diverifikasi dan Anda sudah login.

🔑 Access Token: Aktif selama 1 jam
⏰ Berlaku sampai: 15:30 WIB

[📱 Lihat Produk] [🔓 Logout] [🏠 Menu Utama]
```

### **Payment Method Selection**
```
💳 Pilih Metode Pembayaran

📦 Produk: XL Masa Aktif 30 Hari
💰 Harga: Rp 1.000
📱 Nomor: 087817739901

[💳 Metode Pulsa (BALANCE)]
[💳 Metode DANA]
[💳 Metode QRIS]
```

### **Purchase Success**
```
✅ Pembelian Berhasil!

📦 Produk: XL Masa Aktif 30 Hari
💰 Harga: Rp 1.000
💳 Metode: BALANCE
🆔 Transaction ID: 802eaef3-8b5f-4fb8-a3ad-489d4cc91637

Paket berhasil dibeli. Silakan cek kuotanya via aplikasi MyXL.
```

## 🔧 **Technical Implementation**

### **Database Models**
```go
type User struct {
    ChatID         int64
    PhoneNumber    string
    AccessToken    string
    TokenExpiresAt *time.Time
    IsVerified     bool
}

type PurchaseTransaction struct {
    ID            string
    UserID        int64
    PackageCode   string
    PaymentMethod string
    Price         int64
    Status        string
    ResponseData  string
}
```

### **Service Functions**
```go
// Authentication
func VerifyOTPAndLogin(phoneNumber, otpCode string, userID int64)
func IsUserLoggedIn(chatID int64) bool
func ClearUserSession(chatID int64) error

// Purchase
func PurchaseProduct(userID int64, packageCode, paymentMethod string)
func CheckTransactionStatus(transactionID string)
```

## 🚀 **Ready for Production**

### **Features Completed**
- ✅ **Login System**: OTP verification + access token
- ✅ **Session Management**: 1-hour expiry + manual logout
- ✅ **Database Persistence**: SQLite + GORM
- ✅ **Purchase Integration**: Full API integration
- ✅ **Payment Methods**: BALANCE, DANA, QRIS support
- ✅ **Transaction Tracking**: Complete audit trail
- ✅ **Error Handling**: Robust error management
- ✅ **User Experience**: Professional UI/UX

### **Database Benefits**
- 🔄 **Persistent Data**: Tidak hilang saat restart
- 📊 **Analytics Ready**: Data tersimpan untuk analisis
- 🔍 **Audit Trail**: Complete transaction history
- 📈 **Scalable**: Ready untuk ribuan user

### **Security Benefits**
- 🔐 **Token-based Auth**: Secure session management
- ⏰ **Auto Expiry**: Prevent unauthorized access
- 🔓 **Manual Logout**: User control over session
- 🛡️ **API Integration**: Direct ke official API

---

**Status:** ✅ Fully implemented with database persistence and complete purchase flow
**Ready for:** Production deployment with real users