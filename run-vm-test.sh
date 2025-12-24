#!/bin/bash

##############################################################################
# Quick Start Script for Testing Products API on load-testing-vm-1
# Target: 10.128.0.81:8000
##############################################################################

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
API_TOKEN="api-token-products-test-2025"
CONTROL_PLANE_URL="http://localhost:8080"
TARGET_VM_INTERNAL="http://10.128.0.81:8000"
TARGET_VM_EXTERNAL="http://35.239.233.230:8000"

# Use internal IP by default (faster if on same network)
TARGET_HOST="${TARGET_VM_EXTERNAL}"

# Parse arguments
DURATION=60  # Default 60 seconds
USERS=10     # Default 10 concurrent users
SPAWN_RATE=2 # Default spawn 2 users per second

while [[ $# -gt 0 ]]; do
    case $1 in
        --duration)
            DURATION="$2"
            shift 2
            ;;
        --users)
            USERS="$2"
            shift 2
            ;;
        --spawn-rate)
            SPAWN_RATE="$2"
            shift 2
            ;;
        --external)
            TARGET_HOST="${TARGET_VM_EXTERNAL}"
            shift
            ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --duration N      Test duration in seconds (default: 60)"
            echo "  --users N         Number of concurrent users (default: 10)"
            echo "  --spawn-rate N    Users spawned per second (default: 2)"
            echo "  --external        Use external IP instead of internal"
            echo "  --help            Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                          # Run with defaults"
            echo "  $0 --duration 300 --users 50  # 5 min test with 50 users"
            echo "  $0 --external               # Use external IP"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║    Load Test - Products API on load-testing-vm-1         ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo -e "  Target:       ${TARGET_HOST}"
echo -e "  Duration:     ${DURATION} seconds"
echo -e "  Users:        ${USERS}"
echo -e "  Spawn Rate:   ${SPAWN_RATE}/sec"
echo ""

# Test connectivity to VM
echo -e "${YELLOW}🔍 Testing connectivity to VM...${NC}"
if curl -s --connect-timeout 5 "${TARGET_HOST}/api/products" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ VM is reachable${NC}"
else
    echo -e "${RED}✗ Cannot reach VM at ${TARGET_HOST}${NC}"
    echo -e "${YELLOW}  Trying to diagnose...${NC}"
    
    # Try external IP if we were using internal
    if [[ "$TARGET_HOST" == "$TARGET_VM_INTERNAL" ]]; then
        echo -e "${YELLOW}  Trying external IP...${NC}"
        if curl -s --connect-timeout 5 "${TARGET_VM_EXTERNAL}/api/products" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ External IP works! Re-run with --external flag${NC}"
            exit 1
        fi
    fi
    
    echo -e "${RED}  Please check:${NC}"
    echo -e "${RED}  1. VM is running and application is started${NC}"
    echo -e "${RED}  2. Port 8000 is accessible${NC}"
    echo -e "${RED}  3. Firewall rules allow connections${NC}"
    exit 1
fi

# Test control plane
echo -e "${YELLOW}🔍 Testing control plane...${NC}"
if curl -s "${CONTROL_PLANE_URL}/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Control plane is running${NC}"
else
    echo -e "${RED}✗ Control plane is not running at ${CONTROL_PLANE_URL}${NC}"
    echo -e "${YELLOW}  Start it with: ./bin/controlplane -config config/vm-test-config.yaml${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🚀 Creating load test...${NC}"

# Base64 encode the actual Locust script
SCRIPT_PATH="locust/vm-products-api.py"
if [ ! -f "$SCRIPT_PATH" ]; then
    echo -e "${RED}✗ Locust script not found at: $SCRIPT_PATH${NC}"
    exit 1
fi

# Base64 encode without line breaks (important for JSON)
SCRIPT_CONTENT_B64=$(base64 -i "$SCRIPT_PATH" | tr -d '\n')

echo -e "${YELLOW}Script encoded (${#SCRIPT_CONTENT_B64} bytes)${NC}"

# Create JSON payload using jq for proper escaping
JSON_PAYLOAD=$(jq -n \
  --arg name "Products API Test - $(date +%Y%m%d-%H%M%S)" \
  --arg desc "Load test for Products API on load-testing-vm-1" \
  --arg accountId "my-account" \
  --arg orgId "my-org" \
  --arg projectId "products-api" \
  --arg envId "vm-test" \
  --arg targetUrl "${TARGET_HOST}" \
  --argjson users "${USERS}" \
  --argjson spawnRate "${SPAWN_RATE}" \
  --argjson duration "${DURATION}" \
  --arg scriptContent "${SCRIPT_CONTENT_B64}" \
  '{
    name: $name,
    description: $desc,
    accountId: $accountId,
    orgId: $orgId,
    projectId: $projectId,
    envId: $envId,
    targetUrl: $targetUrl,
    targetUsers: $users,
    spawnRate: $spawnRate,
    durationSeconds: $duration,
    scriptContent: $scriptContent
  }')

# Create load test
CREATE_RESPONSE=$(curl -s -X POST "${CONTROL_PLANE_URL}/v1/load-tests" \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "${JSON_PAYLOAD}")

# Check if creation was successful
if echo "$CREATE_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    LOAD_TEST_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id')
    echo -e "${GREEN}✓ Load test created: ${LOAD_TEST_ID}${NC}"
else
    echo -e "${RED}✗ Failed to create load test${NC}"
    echo -e "${RED}Response: ${CREATE_RESPONSE}${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}🏃 Starting test run...${NC}"

# Start test run
RUN_RESPONSE=$(curl -s -X POST "${CONTROL_PLANE_URL}/v1/load-tests/${LOAD_TEST_ID}/runs" \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"targetUsers\": ${USERS},
    \"spawnRate\": ${SPAWN_RATE},
    \"durationSeconds\": ${DURATION}
  }")

# Check if run was started successfully
if echo "$RUN_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    RUN_ID=$(echo "$RUN_RESPONSE" | jq -r '.id')
    echo -e "${GREEN}✓ Test run started: ${RUN_ID}${NC}"
else
    echo -e "${RED}✗ Failed to start test run${NC}"
    echo -e "${RED}Response: ${RUN_RESPONSE}${NC}"
    exit 1
fi

echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}Test is running!${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}📊 Monitor your test:${NC}"
echo -e "  • Locust Web UI:    http://localhost:8089"
echo -e "  • Swagger UI:       http://localhost:8080/swagger/index.html"
echo -e "  • Run ID:           ${RUN_ID}"
echo ""
echo -e "${YELLOW}📈 View metrics (after test completes):${NC}"
echo -e "  curl http://localhost:8080/v1/visualization/runs/${RUN_ID}/summary \\"
echo -e "    -H 'Authorization: Bearer ${API_TOKEN}' | jq '.summary'"
echo ""
echo -e "${YELLOW}⏱️  Test duration: ${DURATION} seconds${NC}"
echo ""

# Wait for test to complete
echo -e "${YELLOW}⏳ Waiting for test to complete...${NC}"
for i in $(seq 1 $DURATION); do
    sleep 1
    if [ $((i % 10)) -eq 0 ]; then
        echo -e "${YELLOW}   ${i}/${DURATION} seconds elapsed...${NC}"
    fi
done

# Give it a few extra seconds for final metrics
echo -e "${YELLOW}⏳ Waiting for final metrics...${NC}"
sleep 5

echo ""
echo -e "${GREEN}📊 Fetching test results...${NC}"
echo ""

# Get summary
SUMMARY=$(curl -s "${CONTROL_PLANE_URL}/v1/visualization/runs/${RUN_ID}/summary" \
  -H "Authorization: Bearer ${API_TOKEN}")

# Display results
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}               TEST RESULTS SUMMARY${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo ""

if echo "$SUMMARY" | jq -e '.summary' > /dev/null 2>&1; then
    TOTAL_REQUESTS=$(echo "$SUMMARY" | jq -r '.summary.totalRequests // 0')
    TOTAL_RPS=$(echo "$SUMMARY" | jq -r '.summary.requestsPerSecond // 0')
    ERROR_RATE=$(echo "$SUMMARY" | jq -r '.summary.errorRate // 0')
    AVG_RESPONSE=$(echo "$SUMMARY" | jq -r '.summary.avgResponseTime // 0')
    
    echo -e "${YELLOW}Total Requests:${NC}      $TOTAL_REQUESTS"
    echo -e "${YELLOW}Requests/Second:${NC}     $TOTAL_RPS"
    echo -e "${YELLOW}Error Rate:${NC}          ${ERROR_RATE}%"
    echo -e "${YELLOW}Avg Response Time:${NC}   ${AVG_RESPONSE}ms"
    echo ""
    
    # Check if error rate is acceptable
    if (( $(echo "$ERROR_RATE < 1.0" | bc -l) )); then
        echo -e "${GREEN}✓ Test completed successfully with low error rate${NC}"
    elif (( $(echo "$ERROR_RATE < 5.0" | bc -l) )); then
        echo -e "${YELLOW}⚠ Test completed with moderate error rate${NC}"
    else
        echo -e "${RED}✗ Test completed with high error rate - investigate issues${NC}"
    fi
else
    echo -e "${YELLOW}Summary not available yet. View full results:${NC}"
    echo -e "  curl http://localhost:8080/v1/runs/${RUN_ID} \\"
    echo -e "    -H 'Authorization: Bearer ${API_TOKEN}' | jq ."
fi

echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}📁 View detailed results:${NC}"
echo -e "  • MongoDB Compass: mongodb://localhost:27017"
echo -e "  • Database: load_testing"
echo -e "  • Collection: load_test_runs"
echo -e "  • Run ID: ${RUN_ID}"
echo ""
echo -e "${GREEN}✅ Test complete!${NC}"
