# 🔧 Troubleshooting Guide - GRN Store Bot

## ❌ Masalah yang Terjadi: "Transaksi Tidak Ditemukan"

### 🔍 **Root Cause Analysis**

**Masalah:** Admin tidak bisa confirm transaksi dengan error "transaksi tidak ditemukan"

**Penyebab:** Data transaksi disimpan in-memory dan hilang saat bot restart

### 📊 **Debug Process**

1. **User membuat transaksi top-up:**
   ```
   Transaction ID: TXN_6491485169_1756265392
   Status: pending
   ```

2. **Bot restart** (data in-memory hilang)

3. **Admin coba confirm:**
   ```
   /confirm TXN_6491485169_1756265392
   ❌ Gagal konfirmasi: transaksi tidak ditemukan
   ```

### 🛠️ **Solutions**

#### **Solution 1: Debug Command (Immediate)**
```bash
/debug
```
**Output:**
```
🔍 Debug Info

📊 Total Transactions: 0

Transaction IDs:
Tidak ada transaksi dalam memory.

Note: Data disimpan in-memory, akan hilang saat bot restart.
```

#### **Solution 2: Recreate Transaction (Workaround)**
1. User buat transaksi top-up baru
2. **Jangan restart bot**
3. Admin langsung confirm

#### **Solution 3: Persistent Storage (Recommended)**
Implementasi database untuk menyimpan transaksi:

```go
// TODO: Implementasi database
// - SQLite untuk development
// - PostgreSQL untuk production
// - Redis untuk caching
```

## 🔄 **Workflow Perbaikan**

### **Immediate Fix (Sekarang)**
1. **Restart bot** dengan build terbaru
2. **User buat transaksi baru**
3. **Admin gunakan `/debug`** untuk cek transaksi
4. **Admin confirm** sebelum restart bot

### **Long-term Fix (Development)**
1. Implementasi database persistent
2. Migration script untuk data existing
3. Backup & restore mechanism

## 📋 **Admin Commands untuk Debugging**

| Command | Deskripsi | Output |
|---------|-----------|--------|
| `/debug` | Lihat semua transaksi in-memory | Transaction IDs & status |
| `/pending` | Lihat transaksi pending | Filtered pending only |
| `/confirm <id>` | Confirm transaksi | Success/error message |

## 🔍 **Debug Logs**

Bot sekarang menampilkan debug logs:

```bash
# Saat transaksi dibuat
2025/08/27 10:29:52 Transaction created: ID=TXN_6491485169_1756265392, UserID=6491485169, Amount=10000

# Saat admin confirm
2025/08/27 10:35:15 Attempting to confirm transaction: TXN_6491485169_1756265392
2025/08/27 10:35:15 Available transactions: 1
2025/08/27 10:35:15   Available ID: TXN_6491485169_1756265392
```

## ⚠️ **Known Issues**

### **1. In-Memory Storage**
- **Issue:** Data hilang saat restart
- **Impact:** Transaksi pending hilang
- **Workaround:** Jangan restart bot saat ada pending transactions

### **2. Transaction ID Format**
- **Format:** `TXN_{userID}_{timestamp}`
- **Example:** `TXN_6491485169_1756265392`
- **Note:** Pastikan copy exact ID dari `/pending`

### **3. Expired Transactions**
- **Timeout:** 30 menit
- **Auto-expire:** Ya, otomatis berubah status
- **Recovery:** User harus buat transaksi baru

## 🚀 **Best Practices**

### **For Admin:**
1. **Selalu gunakan `/debug`** sebelum confirm
2. **Copy-paste exact Transaction ID** dari `/pending`
3. **Confirm segera** setelah user bayar
4. **Jangan restart bot** saat ada pending transactions

### **For Development:**
1. **Implement database** untuk production
2. **Add transaction recovery** mechanism
3. **Implement auto-backup** before restart
4. **Add health check** endpoints

## 📱 **Quick Fix Commands**

```bash
# 1. Check debug info
/debug

# 2. Check pending transactions
/pending

# 3. Confirm with exact ID
/confirm TXN_6491485169_1756265392

# 4. If still error, check logs
tail -f bot.log
```

## 🔄 **Recovery Process**

### **If Transaction Lost:**
1. **Admin:** Kirim `/debug` untuk confirm data hilang
2. **User:** Buat transaksi top-up baru
3. **Admin:** Monitor dengan `/pending`
4. **Admin:** Confirm segera setelah payment
5. **Manual adjustment:** Jika diperlukan, admin bisa manual add saldo

### **Prevention:**
1. **Backup transactions** before restart
2. **Use persistent storage**
3. **Implement transaction recovery**
4. **Add monitoring alerts**

---

**Status:** ✅ Debug tools implemented, workaround available
**Next:** 🔄 Implement persistent database storage