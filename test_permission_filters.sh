#!/bin/bash

# Test script to verify permission filtering functionality
# This script tests api_id, role_id, client_id, and status filters

BASE_URL="http://localhost:8080/api/v1"

echo "🧪 Testing Permission Filtering Functionality"
echo "=============================================="

# Test 1: Filter by status
echo ""
echo "1️⃣ Testing status filter..."
echo "GET $BASE_URL/permissions?status=active"
curl -s -X GET "$BASE_URL/permissions?status=active" | jq '.success, .data.total, .data.permissions[0].status' 2>/dev/null || echo "❌ Failed to parse JSON response"

echo ""
echo "GET $BASE_URL/permissions?status=inactive"
curl -s -X GET "$BASE_URL/permissions?status=inactive" | jq '.success, .data.total, .data.permissions[0].status' 2>/dev/null || echo "❌ Failed to parse JSON response"

# Test 2: Filter by api_id (you'll need to replace with actual API UUID)
echo ""
echo "2️⃣ Testing api_id filter..."
echo "GET $BASE_URL/permissions?api_id=<API_UUID>"
echo "Note: Replace <API_UUID> with actual API UUID from your database"

# Test 3: Filter by role_id (you'll need to replace with actual Role UUID)
echo ""
echo "3️⃣ Testing role_id filter..."
echo "GET $BASE_URL/permissions?role_id=<ROLE_UUID>"
echo "Note: Replace <ROLE_UUID> with actual Role UUID from your database"

# Test 4: Filter by client_id (you'll need to replace with actual Auth Client UUID)
echo ""
echo "4️⃣ Testing client_id filter..."
echo "GET $BASE_URL/permissions?client_id=<CLIENT_UUID>"
echo "Note: Replace <CLIENT_UUID> with actual Auth Client UUID from your database"

# Test 5: Combined filters
echo ""
echo "5️⃣ Testing combined filters..."
echo "GET $BASE_URL/permissions?status=active&is_system=false"
curl -s -X GET "$BASE_URL/permissions?status=active&is_system=false" | jq '.success, .data.total' 2>/dev/null || echo "❌ Failed to parse JSON response"

# Test 6: Test status update
echo ""
echo "6️⃣ Testing status update..."
echo "Note: Replace <PERMISSION_UUID> with actual Permission UUID from your database"
echo "PUT $BASE_URL/permissions/<PERMISSION_UUID>/status"
echo 'Body: {"status": "inactive"}'

echo ""
echo "✅ Test script completed!"
echo "💡 To run actual tests, replace placeholder UUIDs with real values from your database"
