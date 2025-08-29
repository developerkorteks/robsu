#!/bin/bash

# GRN Store API Testing Script
# Usage: ./test_api.sh [base_url]

BASE_URL=${1:-"http://localhost:8080/api"}
echo "🚀 Testing GRN Store API at: $BASE_URL"
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print test results
print_test() {
    local test_name="$1"
    local status_code="$2"
    local expected="$3"
    
    echo -e "\n${BLUE}Testing: $test_name${NC}"
    echo "Expected: $expected"
    
    if [ "$status_code" = "$expected" ]; then
        echo -e "${GREEN}✅ PASS${NC} (Status: $status_code)"
    else
        echo -e "${RED}❌ FAIL${NC} (Status: $status_code, Expected: $expected)"
    fi
}

# Function to make API call and return status code
api_call() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    if [ -n "$data" ]; then
        curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -s -o /dev/null -w "%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json"
    fi
}

# Function to make API call and return response
api_call_response() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    if [ -n "$data" ]; then
        curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data"
    else
        curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json"
    fi
}

echo -e "\n${YELLOW}1. HEALTH CHECK${NC}"
echo "=================================================="

# Test health check
status=$(api_call "GET" "/health")
print_test "Health Check" "$status" "200"

# Show health response
echo -e "\n${BLUE}Health Response:${NC}"
api_call_response "GET" "/health" | jq '.' 2>/dev/null || echo "Response received"

echo -e "\n${YELLOW}2. ADMIN ENDPOINTS${NC}"
echo "=================================================="

# Test get pending transactions
status=$(api_call "GET" "/admin/topups/pending")
print_test "Get Pending Transactions" "$status" "200"

# Test get all transactions
status=$(api_call "GET" "/admin/transactions")
print_test "Get All Transactions" "$status" "200"

# Test get all transactions with filters
status=$(api_call "GET" "/admin/transactions?status=pending&limit=10")
print_test "Get Transactions with Filters" "$status" "200"

# Test get transaction detail (will likely return 404 since no real transaction)
status=$(api_call "GET" "/admin/transactions/TXN_TEST_123456")
print_test "Get Transaction Detail (Non-existent)" "$status" "404"

# Test approve transaction (will likely return error since no real transaction)
approve_data='{
    "transaction_id": "TXN_TEST_123456",
    "status": "approved",
    "admin_note": "Test approval"
}'
status=$(api_call "POST" "/admin/topups/approve" "$approve_data")
print_test "Approve Transaction (Non-existent)" "$status" "500"

# Test reject transaction
reject_data='{
    "transaction_id": "TXN_TEST_123456",
    "status": "rejected",
    "admin_note": "Test rejection"
}'
status=$(api_call "POST" "/admin/topups/approve" "$reject_data")
print_test "Reject Transaction (Non-existent)" "$status" "500"

# Test bulk approve
bulk_data='{
    "transaction_ids": ["TXN_TEST_123456", "TXN_TEST_789012"],
    "admin_note": "Bulk test approval"
}'
status=$(api_call "POST" "/admin/topups/bulk-approve" "$bulk_data")
print_test "Bulk Approve Transactions" "$status" "200"

# Test invalid approve request
invalid_approve='{
    "transaction_id": "TXN_TEST_123456",
    "status": "invalid_status"
}'
status=$(api_call "POST" "/admin/topups/approve" "$invalid_approve")
print_test "Invalid Approve Status" "$status" "400"

echo -e "\n${YELLOW}3. PUBLIC ENDPOINTS${NC}"
echo "=================================================="

# Test create topup transaction
create_topup='{
    "user_id": 123456789,
    "username": "test_user",
    "amount": 50000
}'
status=$(api_call "POST" "/public/topups/create" "$create_topup")
print_test "Create Top-Up Transaction" "$status" "200"

# Test create topup with invalid amount (too low)
invalid_topup_low='{
    "user_id": 123456789,
    "username": "test_user",
    "amount": 5000
}'
status=$(api_call "POST" "/public/topups/create" "$invalid_topup_low")
print_test "Create Top-Up (Amount Too Low)" "$status" "400"

# Test create topup with invalid amount (too high)
invalid_topup_high='{
    "user_id": 123456789,
    "username": "test_user",
    "amount": 2000000
}'
status=$(api_call "POST" "/public/topups/create" "$invalid_topup_high")
print_test "Create Top-Up (Amount Too High)" "$status" "400"

# Test get user balance
status=$(api_call "GET" "/public/users/123456789/balance")
print_test "Get User Balance" "$status" "200"

# Test get user balance with invalid user ID
status=$(api_call "GET" "/public/users/invalid_id/balance")
print_test "Get User Balance (Invalid ID)" "$status" "400"

echo -e "\n${YELLOW}4. ERROR HANDLING TESTS${NC}"
echo "=================================================="

# Test invalid JSON
status=$(api_call "POST" "/admin/topups/approve" "invalid_json")
print_test "Invalid JSON Request" "$status" "400"

# Test missing required fields
missing_fields='{
    "status": "approved"
}'
status=$(api_call "POST" "/admin/topups/approve" "$missing_fields")
print_test "Missing Required Fields" "$status" "400"

# Test non-existent endpoint
status=$(api_call "GET" "/non-existent-endpoint")
print_test "Non-existent Endpoint" "$status" "404"

echo -e "\n${YELLOW}5. SAMPLE RESPONSES${NC}"
echo "=================================================="

echo -e "\n${BLUE}Sample: Get Pending Transactions${NC}"
api_call_response "GET" "/admin/topups/pending" | jq '.' 2>/dev/null || echo "Response received (jq not available for formatting)"

echo -e "\n${BLUE}Sample: Get All Transactions${NC}"
api_call_response "GET" "/admin/transactions?limit=5" | jq '.' 2>/dev/null || echo "Response received (jq not available for formatting)"

echo -e "\n${BLUE}Sample: Get User Balance${NC}"
api_call_response "GET" "/public/users/123456789/balance" | jq '.' 2>/dev/null || echo "Response received (jq not available for formatting)"

echo -e "\n${YELLOW}6. PERFORMANCE TEST${NC}"
echo "=================================================="

echo -e "\n${BLUE}Testing response times...${NC}"

# Test health endpoint response time
echo "Health endpoint:"
time curl -s -o /dev/null "$BASE_URL/health"

# Test admin endpoints response time
echo "Admin pending transactions:"
time curl -s -o /dev/null "$BASE_URL/admin/topups/pending"

echo "Admin all transactions:"
time curl -s -o /dev/null "$BASE_URL/admin/transactions"

echo -e "\n${GREEN}🎉 API Testing Complete!${NC}"
echo "=================================================="

# Summary
echo -e "\n${YELLOW}SUMMARY:${NC}"
echo "• Health Check: Basic server functionality"
echo "• Admin Endpoints: Transaction management for admins"
echo "• Public Endpoints: External integration capabilities"
echo "• Error Handling: Proper error responses"
echo "• Performance: Response time measurements"

echo -e "\n${BLUE}Next Steps:${NC}"
echo "1. Create real transactions via Telegram bot"
echo "2. Test admin approval workflow with real data"
echo "3. Monitor API performance under load"
echo "4. Implement authentication for production"

echo -e "\n${GREEN}For detailed documentation, see: API_DOCUMENTATION.md${NC}"