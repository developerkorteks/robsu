# GRN Store Bot API Documentation

API ini menyediakan endpoint untuk mengelola transaksi top-up yang terintegrasi dengan bot Telegram/WhatsApp.

## Base URL
```
http://localhost:8253/api
```

## Endpoints

### 1. Admin Endpoints

#### Get Pending Top-Up Transactions
```http
GET /admin/topups/pending
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "TXN_123456_1234567890",
      "user_id": 123456,
      "username": "John Doe",
      "amount": 50000,
      "status": "pending",
      "qris_code": "00020101021126...",
      "created_at": "2024-01-15 10:30:00",
      "expired_at": "2024-01-15 11:00:00"
    }
  ],
  "count": 1
}
```

#### Get All Transactions (with filters)
```http
GET /admin/transactions?status=pending&user_id=123456&limit=10&offset=0
```

**Query Parameters:**
- `status` (optional): pending, confirmed, rejected, expired
- `user_id` (optional): Filter by specific user ID
- `limit` (optional): Number of results per page (default: 50)
- `offset` (optional): Pagination offset (default: 0)

**Response:**
```json
{
  "success": true,
  "data": [...],
  "total": 25,
  "limit": 10,
  "offset": 0
}
```

#### Get Transaction Detail
```http
GET /admin/transactions/{transaction_id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "TXN_123456_1234567890",
    "user_id": 123456,
    "username": "John Doe",
    "amount": 50000,
    "status": "pending",
    "qris_code": "00020101021126...",
    "created_at": "2024-01-15 10:30:00",
    "expired_at": "2024-01-15 11:00:00",
    "approved_by": null,
    "approved_at": null
  }
}
```

#### Approve/Reject Single Transaction
```http
POST /admin/topups/approve
```

**Request Body:**
```json
{
  "transaction_id": "TXN_123456_1234567890",
  "status": "approved",
  "admin_note": "Pembayaran sudah diterima"
}
```

**Fields:**
- `transaction_id` (required): ID transaksi yang akan diproses
- `status` (required): "approved" atau "rejected"
- `admin_note` (optional): Catatan admin untuk user

**Response:**
```json
{
  "success": true,
  "message": "Transaction approved successfully",
  "data": {
    "transaction_id": "TXN_123456_1234567890",
    "status": "approved",
    "admin_note": "Pembayaran sudah diterima",
    "processed_at": "2024-01-15T10:45:00Z"
  }
}
```

#### Bulk Approve Transactions
```http
POST /admin/topups/bulk-approve
```

**Request Body:**
```json
{
  "transaction_ids": [
    "TXN_123456_1234567890",
    "TXN_789012_1234567891"
  ],
  "admin_note": "Batch approval - semua pembayaran sudah diterima"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Bulk approval completed",
  "success_count": 2,
  "fail_count": 0,
  "results": [
    {
      "transaction_id": "TXN_123456_1234567890",
      "status": "success"
    },
    {
      "transaction_id": "TXN_789012_1234567891",
      "status": "success"
    }
  ]
}
```

### 2. Public Endpoints

#### Create Top-Up Transaction
```http
POST /public/topups/create
```

**Request Body:**
```json
{
  "user_id": 123456,
  "username": "John Doe",
  "amount": 50000
}
```

**Response:**
```json
{
  "success": true,
  "message": "Top up transaction created successfully",
  "data": {
    "transaction_id": "TXN_123456_1234567890",
    "qris_code": "00020101021126...",
    "amount": 50000,
    "expired_at": "2024-01-15 11:00:00"
  }
}
```

#### Get User Balance
```http
GET /public/users/{user_id}/balance
```

**Response:**
```json
{
  "success": true,
  "data": {
    "user_id": 123456,
    "balance": 150000
  }
}
```

### 3. Health Check
```http
GET /health
```

**Response:**
```json
{
  "status": "ok",
  "message": "GRN Store API is running"
}
```

## Error Responses

Semua endpoint akan mengembalikan error dalam format berikut:

```json
{
  "success": false,
  "error": "Error message description"
}
```

**Common HTTP Status Codes:**
- `200` - Success
- `400` - Bad Request (invalid input)
- `404` - Not Found (transaction/user not found)
- `500` - Internal Server Error

## Integration Notes

1. **Sinkronisasi Data**: API ini menggunakan sistem in-memory storage yang sama dengan bot Telegram/WhatsApp, dan secara otomatis menyinkronkan data ke database.

2. **Notifikasi**: Ketika transaksi di-approve/reject melalui API, user akan menerima notifikasi melalui bot Telegram/WhatsApp.

3. **Real-time Updates**: Perubahan status transaksi melalui API akan langsung terlihat di bot dan sebaliknya.

4. **Validasi**: API menggunakan validasi yang sama dengan bot (minimal Rp 10.000, maksimal Rp 1.000.000).

## Example Usage

### Approve a transaction using curl:
```bash
curl -X POST http://localhost:8253/api/admin/topups/approve \
  -H "Content-Type: application/json" \
  -d '{
    "transaction_id": "TXN_123456_1234567890",
    "status": "approved",
    "admin_note": "Pembayaran sudah diterima"
  }'
```

### Get pending transactions:
```bash
curl http://localhost:8253/api/admin/topups/pending
```

### Create new topup transaction:
```bash
curl -X POST http://localhost:8253/api/public/topups/create \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 123456,
    "username": "John Doe",
    "amount": 50000
  }'
```