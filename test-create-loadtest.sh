#!/bin/bash

# Debug script to test load test creation

API_TOKEN="api-token-products-test-2025"
CONTROL_PLANE_URL="http://localhost:8080"

# Create a simple test script
SIMPLE_SCRIPT=$(cat << 'EOF'
from locust import HttpUser, task, between

class TestUser(HttpUser):
    wait_time = between(1, 2)
    
    @task
    def test_products(self):
        self.client.get("/api/products")
EOF
)

# Base64 encode it
SCRIPT_B64=$(echo "$SIMPLE_SCRIPT" | base64)

echo "Creating load test with script..."
echo "Script (base64): ${SCRIPT_B64:0:50}..."

# Create the JSON payload properly
JSON_PAYLOAD=$(cat <<EOF
{
  "name": "Test Load Test - $(date +%s)",
  "description": "Debug test",
  "accountId": "my-account",
  "orgId": "my-org",
  "projectId": "products-api",
  "envId": "vm-test",
  "targetUrl": "http://10.128.0.81:8000",
  "targetUsers": 5,
  "spawnRate": 1,
  "durationSeconds": 30,
  "scriptContent": "$SCRIPT_B64"
}
EOF
)

echo "Sending request..."
RESPONSE=$(curl -s -X POST "${CONTROL_PLANE_URL}/v1/load-tests" \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "$JSON_PAYLOAD")

echo "Response:"
echo "$RESPONSE" | jq '.'

# Check if successful
if echo "$RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    LOAD_TEST_ID=$(echo "$RESPONSE" | jq -r '.id')
    echo ""
    echo "✓ Load test created: $LOAD_TEST_ID"
    
    # Check if script revision was created in MongoDB
    echo ""
    echo "Checking MongoDB for script revision..."
    echo "Run in MongoDB Compass:"
    echo "  Database: load_testing"
    echo "  Collection: script_revisions"
    echo "  Query: {\"loadTestId\": \"$LOAD_TEST_ID\"}"
    
    # Try to start a run
    echo ""
    echo "Attempting to start test run..."
    RUN_RESPONSE=$(curl -s -X POST "${CONTROL_PLANE_URL}/v1/load-tests/${LOAD_TEST_ID}/runs" \
      -H "Authorization: Bearer ${API_TOKEN}" \
      -H "Content-Type: application/json" \
      -d '{
        "targetUsers": 5,
        "spawnRate": 1,
        "durationSeconds": 30
      }')
    
    echo "Run Response:"
    echo "$RUN_RESPONSE" | jq '.'
    
    if echo "$RUN_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
        RUN_ID=$(echo "$RUN_RESPONSE" | jq -r '.id')
        echo ""
        echo "✓ Test run started: $RUN_ID"
    else
        echo ""
        echo "✗ Failed to start run"
    fi
else
    echo ""
    echo "✗ Failed to create load test"
fi
