#!/bin/bash
set -e

# Test script to verify automatic plugin injection works correctly

echo "==================================================================="
echo "Testing Harness Plugin Automatic Injection"
echo "==================================================================="

# Configuration
CONTROL_PLANE_URL="http://localhost:8080"
API_TOKEN="your-api-token-here"

# Create clean test script (no plugin code)
CLEAN_SCRIPT=$(cat << 'EOF'
from locust import HttpUser, task, between
import random

class TestUser(HttpUser):
    wait_time = between(1, 3)
    
    @task
    def test_endpoint(self):
        self.client.get("/api/products")
EOF
)

# Base64 encode the script
ENCODED_SCRIPT=$(echo "$CLEAN_SCRIPT" | base64)

echo ""
echo "1. Clean User Script (before injection):"
echo "----------------------------------------"
echo "$CLEAN_SCRIPT"
echo ""

# Create load test via API
echo "2. Creating load test (control plane will auto-inject plugin)..."
RESPONSE=$(curl -s -X POST "$CONTROL_PLANE_URL/v1/load-tests" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_TOKEN" \
  -d '{
    "name": "Plugin Injection Test",
    "description": "Testing automatic plugin injection",
    "scriptContent": "'"$ENCODED_SCRIPT"'",
    "accountId": "test-account",
    "orgId": "test-org",
    "projectId": "test-project",
    "envId": "test-env",
    "targetUrl": "http://localhost:8000",
    "createdBy": "test-user"
  }')

echo "Response: $RESPONSE"
echo ""

# Extract test ID
TEST_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$TEST_ID" ]; then
  echo "❌ Failed to create load test"
  exit 1
fi

echo "✅ Load test created with ID: $TEST_ID"
echo ""

# Retrieve the script to verify injection
echo "3. Retrieving script to verify plugin was injected..."
SCRIPT_RESPONSE=$(curl -s "$CONTROL_PLANE_URL/v1/load-tests/$TEST_ID/script" \
  -H "Authorization: Bearer $API_TOKEN")

# Extract and decode script content
STORED_SCRIPT=$(echo "$SCRIPT_RESPONSE" | grep -o '"scriptContent":"[^"]*"' | cut -d'"' -f4 | base64 -d)

echo ""
echo "4. Enhanced Script (after injection):"
echo "--------------------------------------"
echo "$STORED_SCRIPT"
echo ""

# Verify plugin import was added
if echo "$STORED_SCRIPT" | grep -q "locust_harness_plugin"; then
  echo "✅ Plugin import successfully injected!"
else
  echo "❌ Plugin import NOT found in stored script"
  exit 1
fi

echo ""
echo "==================================================================="
echo "Test completed successfully!"
echo "==================================================================="
echo ""
echo "Summary:"
echo "- User submitted clean script (no plugin code)"
echo "- Control plane automatically injected plugin import"
echo "- Script stored with plugin integration"
echo "- Ready to run on Locust with full control plane features"
echo ""
