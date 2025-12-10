#!/bin/bash
# Example API calls for the Load Manager Control Plane

# Configuration
CONTROL_PLANE_URL="http://localhost:8080"
API_TOKEN="your-api-token-here"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Load Manager Control Plane API Examples ===${NC}\n"

# 1. Health Check
echo -e "${GREEN}1. Health Check${NC}"
curl -s "$CONTROL_PLANE_URL/health" | jq .
echo -e "\n"

# 2. Create a new load test
echo -e "${GREEN}2. Create and Start Load Test${NC}"
TEST_RESPONSE=$(curl -s -X POST "$CONTROL_PLANE_URL/v1/tests" \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-1",
    "envId": "dev",
    "scenarioId": "load-test-scenario-1",
    "targetUsers": 50,
    "spawnRate": 5,
    "durationSeconds": 300,
    "metadata": {
      "description": "Example load test",
      "version": "1.0"
    }
  }')

echo "$TEST_RESPONSE" | jq .

# Extract test ID for subsequent calls
TEST_ID=$(echo "$TEST_RESPONSE" | jq -r '.id')
echo -e "\nTest ID: $TEST_ID\n"

# 3. Get test status
echo -e "${GREEN}3. Get Test Status and Metrics${NC}"
curl -s -X GET "$CONTROL_PLANE_URL/v1/tests/$TEST_ID" \
  -H "Authorization: Bearer $API_TOKEN" | jq .
echo -e "\n"

# 4. List all tests
echo -e "${GREEN}4. List All Tests${NC}"
curl -s -X GET "$CONTROL_PLANE_URL/v1/tests" \
  -H "Authorization: Bearer $API_TOKEN" | jq .
echo -e "\n"

# 5. List tests by tenant
echo -e "${GREEN}5. List Tests by Tenant${NC}"
curl -s -X GET "$CONTROL_PLANE_URL/v1/tests?tenantId=tenant-1" \
  -H "Authorization: Bearer $API_TOKEN" | jq .
echo -e "\n"

# 6. List running tests
echo -e "${GREEN}6. List Running Tests${NC}"
curl -s -X GET "$CONTROL_PLANE_URL/v1/tests?status=Running" \
  -H "Authorization: Bearer $API_TOKEN" | jq .
echo -e "\n"

# Wait a bit to let some metrics accumulate
echo -e "${BLUE}Waiting 15 seconds for metrics to accumulate...${NC}\n"
sleep 15

# 7. Get updated metrics
echo -e "${GREEN}7. Get Updated Test Metrics${NC}"
curl -s -X GET "$CONTROL_PLANE_URL/v1/tests/$TEST_ID" \
  -H "Authorization: Bearer $API_TOKEN" | jq '.lastMetrics'
echo -e "\n"

# 8. Stop the test
echo -e "${GREEN}8. Stop Load Test${NC}"
curl -s -X POST "$CONTROL_PLANE_URL/v1/tests/$TEST_ID/stop" \
  -H "Authorization: Bearer $API_TOKEN" | jq .
echo -e "\n"

# 9. Get final test status
echo -e "${GREEN}9. Get Final Test Status${NC}"
sleep 2
curl -s -X GET "$CONTROL_PLANE_URL/v1/tests/$TEST_ID" \
  -H "Authorization: Bearer $API_TOKEN" | jq .
echo -e "\n"

echo -e "${BLUE}=== Examples Complete ===${NC}"
