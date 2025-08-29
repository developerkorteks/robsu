# 🚀 GRN Store API Documentation

Dokumentasi lengkap untuk semua endpoint API GRN Store Bot menggunakan curl.

## 📋 Base URL
```
http://localhost:8080/api
```

## 🔐 Authentication
Saat ini API tidak menggunakan authentication khusus, namun endpoint admin sebaiknya dilindungi dengan middleware auth di production.

---

## 📊 Health Check

### GET /health
Mengecek status API server.

```bash
curl -X GET "http://localhost:8080/api/health" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "status": "ok",
  "message": "GRN Store API is running"
}
```

---

## 👨‍💼 Admin Endpoints

### 1. Get Pending Top-Up Transactions

**GET /admin/topups/pending**

Mendapatkan semua transaksi top-up yang menunggu approval.

```bash
curl -X GET "http://localhost:8080/api/admin/topups/pending" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "TXN_1234567890_1234567890",
      "user_id": 123456789,
      "username": "john_doe",
      "amount": 50000,
      "status": "pending",
      "qris_code": "00020101021126580014ID.CO.QRIS.WWW0215ID20232...",
      "created_at": "2024-01-15 10:30:00",
      "expired_at": "2024-01-15 11:30:00"
    }
  ],
  "count": 1
}
```

### 2. Get All Transactions (with filters)

**GET /admin/transactions**

Mendapatkan semua transaksi dengan filter dan pagination.

**Query Parameters:**
- `status` (optional): pending, confirmed, rejected, expired
- `user_id` (optional): Filter by user ID
- `limit` (optional): Jumlah data per halaman (default: 50)
- `offset` (optional): Offset untuk pagination (default: 0)

```bash
# Get all transactions
curl -X GET "http://localhost:8080/api/admin/transactions" \
  -H "Content-Type: application/json"

# Get pending transactions only
curl -X GET "http://localhost:8080/api/admin/transactions?status=pending" \
  -H "Content-Type: application/json"

# Get transactions for specific user
curl -X GET "http://localhost:8080/api/admin/transactions?user_id=123456789" \
  -H "Content-Type: application/json"

# Get with pagination
curl -X GET "http://localhost:8080/api/admin/transactions?limit=10&offset=0" \
  -H "Content-Type: application/json"

# Combined filters
curl -X GET "http://localhost:8080/api/admin/transactions?status=confirmed&limit=20&offset=0" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "TXN_1234567890_1234567890",
      "user_id": 123456789,
      "username": "john_doe",
      "amount": 50000,
      "status": "confirmed",
      "qris_code": "00020101021126580014ID.CO.QRIS.WWW0215ID20232...",
      "created_at": "2024-01-15 10:30:00",
      "expired_at": "2024-01-15 11:30:00",
      "approved_by": 987654321,
      "approved_at": "2024-01-15 10:45:00"
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

### 3. Get Transaction Detail

**GET /admin/transactions/:id**

Mendapatkan detail transaksi berdasarkan ID.

```bash
curl -X GET "http://localhost:8080/api/admin/transactions/TXN_1234567890_1234567890" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "TXN_1234567890_1234567890",
    "user_id": 123456789,
    "username": "john_doe",
    "amount": 50000,
    "status": "confirmed",
    "qris_code": "00020101021126580014ID.CO.QRIS.WWW0215ID20232...",
    "created_at": "2024-01-15 10:30:00",
    "expired_at": "2024-01-15 11:30:00",
    "approved_by": 987654321,
    "approved_at": "2024-01-15 10:45:00"
  }
}
```

**Error Response (Transaction not found):**
```json
{
  "success": false,
  "error": "Transaction not found"
}
```

### 4. Approve/Reject Transaction

**POST /admin/topups/approve**

Approve atau reject transaksi top-up.

**Request Body:**
```json
{
  "transaction_id": "TXN_1234567890_1234567890",
  "status": "approved",  // "approved" or "rejected"
  "admin_note": "Pembayaran sudah dikonfirmasi"  // optional
}
```

**Approve Transaction:**
```bash
curl -X POST "http://localhost:8080/api/admin/topups/approve" \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "TXN_1234567890_1234567890",
    "status": "approved",
    "admin_note": "Pembayaran sudah dikonfirmasi"
  }'
```

**Reject Transaction:**
```bash
curl -X POST "http://localhost:8080/api/admin/topups/approve" \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "TXN_1234567890_1234567890",
    "status": "rejected",
    "admin_note": "Bukti pembayaran tidak valid"
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Transaction approved successfully",
  "data": {
    "transaction_id": "TXN_1234567890_1234567890",
    "status": "approved",
    "admin_note": "Pembayaran sudah dikonfirmasi",
    "processed_at": "2024-01-15T10:45:00Z"
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "Status must be 'approved' or 'rejected'"
}
```

### 5. Bulk Approve Transactions

**POST /admin/topups/bulk-approve**

Approve multiple transaksi sekaligus.

**Request Body:**
```json
{
  "transaction_ids": [
    "TXN_1234567890_1234567890",
    "TXN_0987654321_0987654321",
    "TXN_1111111111_2222222222"
  ],
  "admin_note": "Bulk approval - semua pembayaran valid"  // optional
}
```

```bash
curl -X POST "http://localhost:8080/api/admin/topups/bulk-approve" \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_ids": [
      "TXN_1234567890_1234567890",
      "TXN_0987654321_0987654321",
      "TXN_1111111111_2222222222"
    ],
    "admin_note": "Bulk approval - semua pembayaran valid"
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Bulk approval completed",
  "success_count": 2,
  "fail_count": 1,
  "results": [
    {
      "transaction_id": "TXN_1234567890_1234567890",
      "status": "success"
    },
    {
      "transaction_id": "TXN_0987654321_0987654321",
      "status": "success"
    },
    {
      "transaction_id": "TXN_1111111111_2222222222",
      "status": "failed",
      "error": "Transaction not found"
    }
  ]
}
```

---

## 🌐 Public Endpoints

### 1. Create Top-Up Transaction

**POST /public/topups/create**

Membuat transaksi top-up baru (untuk integrasi eksternal).

**Request Body:**
```json
{
  "user_id": 123456789,
  "username": "john_doe",
  "amount": 50000
}
```

```bash
curl -X POST "http://localhost:8080/api/public/topups/create" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 123456789,
    "username": "john_doe",
    "amount": 50000
  }'
```

**Response:**
```json
{
  "success": true,
  "message": "Top up transaction created successfully",
  "data": {
    "transaction_id": "TXN_1234567890_1234567890",
    "qris_code": "00020101021126580014ID.CO.QRIS.WWW0215ID20232...",
    "amount": 50000,
    "expired_at": "2024-01-15 11:30:00"
  }
}
```

**Error Response (Invalid Amount):**
```json
{
  "success": false,
  "error": "Minimal top up adalah Rp 10.000"
}
```

### 2. Get User Balance

**GET /public/users/:user_id/balance**

Mendapatkan saldo user berdasarkan user ID.

```bash
curl -X GET "http://localhost:8080/api/public/users/123456789/balance" \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user_id": 123456789,
    "balance": 150000
  }
}
```

**Error Response (Invalid User ID):**
```json
{
  "success": false,
  "error": "Invalid user ID"
}
```

---

## 📝 Examples & Use Cases

### 1. Admin Workflow - Process Pending Transactions

```bash
# 1. Get all pending transactions
curl -X GET "http://localhost:8080/api/admin/topups/pending"

# 2. Get detail of specific transaction
curl -X GET "http://localhost:8080/api/admin/transactions/TXN_1234567890_1234567890"

# 3. Approve the transaction
curl -X POST "http://localhost:8080/api/admin/topups/approve" \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "TXN_1234567890_1234567890",
    "status": "approved",
    "admin_note": "Pembayaran valid"
  }'
```

### 2. Bulk Processing

```bash
# 1. Get all pending transactions
PENDING=$(curl -s -X GET "http://localhost:8080/api/admin/topups/pending")

# 2. Extract transaction IDs (using jq)
TRANSACTION_IDS=$(echo $PENDING | jq -r '.data[].id')

# 3. Bulk approve all pending transactions
curl -X POST "http://localhost:8080/api/admin/topups/bulk-approve" \
  -H "Content-Type: application/json" \
  -d "{
    \"transaction_ids\": $(echo $PENDING | jq '[.data[].id]'),
    \"admin_note\": \"Bulk approval - verified payments\"
  }"
```

### 3. External Integration

```bash
# 1. Create top-up for external user
curl -X POST "http://localhost:8080/api/public/topups/create" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 999888777,
    "username": "external_user",
    "amount": 100000
  }'

# 2. Check user balance
curl -X GET "http://localhost:8080/api/public/users/999888777/balance"
```

### 4. Monitoring & Analytics

```bash
# Get all confirmed transactions for revenue calculation
curl -X GET "http://localhost:8080/api/admin/transactions?status=confirmed&limit=1000"

# Get transactions for specific user
curl -X GET "http://localhost:8080/api/admin/transactions?user_id=123456789"

# Get recent transactions (last 50)
curl -X GET "http://localhost:8080/api/admin/transactions?limit=50&offset=0"
```

---

## 🚨 Error Handling

### Common Error Responses

**400 Bad Request:**
```json
{
  "success": false,
  "error": "Invalid request format or missing required fields"
}
```

**404 Not Found:**
```json
{
  "success": false,
  "error": "Resource not found"
}
```

**500 Internal Server Error:**
```json
{
  "success": false,
  "error": "Internal server error message"
}
```

---

## 📊 Response Format

Semua endpoint menggunakan format response yang konsisten:

**Success Response:**
```json
{
  "success": true,
  "data": { ... },
  "message": "Optional success message"
}
```

**Error Response:**
```json
{
  "success": false,
  "error": "Error message"
}
```

**Paginated Response:**
```json
{
  "success": true,
  "data": [ ... ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

---

## 🔧 Development Tips

### 1. Testing with curl
```bash
# Set base URL as variable
BASE_URL="http://localhost:8080/api"

# Test health check
curl -X GET "$BASE_URL/health"

# Pretty print JSON responses
curl -X GET "$BASE_URL/admin/topups/pending" | jq '.'
```

### 2. Environment Variables
```bash
# For different environments
export API_BASE_URL="http://localhost:8080/api"  # Development
export API_BASE_URL="https://api.grnstore.com/api"  # Production
```

### 3. Batch Scripts
```bash
#!/bin/bash
# approve_all_pending.sh

BASE_URL="http://localhost:8080/api"

# Get all pending transactions
PENDING=$(curl -s -X GET "$BASE_URL/admin/topups/pending")

# Extract transaction IDs
TRANSACTION_IDS=$(echo $PENDING | jq -r '.data[].id')

# Approve each transaction
for tx_id in $TRANSACTION_IDS; do
  echo "Approving transaction: $tx_id"
  curl -X POST "$BASE_URL/admin/topups/approve" \
    -H "Content-Type: application/json" \
    -d "{
      \"transaction_id\": \"$tx_id\",
      \"status\": \"approved\",
      \"admin_note\": \"Auto-approved by script\"
    }"
done
```

---

## 🔒 Security Considerations

1. **Authentication**: Implementasikan API key atau JWT untuk production
2. **Rate Limiting**: Tambahkan rate limiting untuk mencegah abuse
3. **Input Validation**: Semua input sudah divalidasi di level handler
4. **CORS**: Sudah dikonfigurasi untuk cross-origin requests
5. **HTTPS**: Gunakan HTTPS di production environment

---

## 📈 Performance Notes

- **In-Memory Storage**: Data transaksi disimpan di memory untuk performa tinggi
- **Pagination**: Gunakan limit dan offset untuk dataset besar
- **Caching**: Response dapat di-cache untuk endpoint yang jarang berubah
- **Concurrent Access**: Thread-safe dengan mutex untuk data consistency

---

## 🚀 Production Deployment

```bash
# Build aplikasi
go build -o bottele ./cmd/main.go

# Run dengan environment variables
export PORT=8080
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=grnstore
export DB_USER=postgres
export DB_PASSWORD=password

# Start server
./bottele
```

API akan tersedia di `http://localhost:8080/api` 🎉