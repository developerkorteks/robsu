#!/bin/bash

# Test script for GRN Store Bot API
BASE_URL="http://localhost:8253/api"

echo "=== Testing GRN Store Bot API ==="
echo

# Test health check
echo "1. Testing health check..."
curl -s "$BASE_URL/health" | jq '.'
echo
echo

# Test create topup transaction
echo "2. Creating test topup transaction..."
RESPONSE=$(curl -s -X POST "$BASE_URL/public/topups/create" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 123456,
    "username": "Test User",
    "amount": 50000
  }')

echo "$RESPONSE" | jq '.'
TRANSACTION_ID=$(echo "$RESPONSE" | jq -r '.data.transaction_id')
echo "Transaction ID: $TRANSACTION_ID"
echo
echo

# Test get pending transactions
echo "3. Getting pending transactions..."
curl -s "$BASE_URL/admin/topups/pending" | jq '.'
echo
echo

# Test get transaction detail
if [ "$TRANSACTION_ID" != "null" ] && [ "$TRANSACTION_ID" != "" ]; then
    echo "4. Getting transaction detail..."
    curl -s "$BASE_URL/admin/transactions/$TRANSACTION_ID" | jq '.'
    echo
    echo

    # Test approve transaction
    echo "5. Approving transaction..."
    curl -s -X POST "$BASE_URL/admin/topups/approve" \
      -H "Content-Type: application/json" \
      -d "{
        \"transaction_id\": \"$TRANSACTION_ID\",
        \"status\": \"approved\",
        \"admin_note\": \"Test approval via API\"
      }" | jq '.'
    echo
    echo

    # Test get user balance
    echo "6. Getting user balance..."
    curl -s "$BASE_URL/public/users/123456/balance" | jq '.'
    echo
    echo
fi

# Test get all transactions
echo "7. Getting all transactions..."
curl -s "$BASE_URL/admin/transactions?limit=5" | jq '.'
echo

echo "=== API Test Completed ==="