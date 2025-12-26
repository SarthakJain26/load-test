#!/bin/bash

##############################################################################
# Load Test Script - Plugin Architecture
# 
# Uses clean user scripts with automatic Harness plugin injection
# Target: Products API on load-testing-vm-1 (35.239.233.230:8000)
# Setup: Control Plane + Locust on Mac, App on Linux VM
##############################################################################

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
API_TOKEN="api-token-products-test-2025"
CONTROL_PLANE_URL="http://localhost:8080"
TARGET_VM_EXTERNAL="http://35.239.233.230:8000"

# Default parameters
DURATION=60
USERS=10
SPAWN_RATE=2

# Parse arguments
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
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --duration N      Test duration in seconds (default: 60)"
            echo "  --users N         Number of concurrent users (default: 10)"
            echo "  --spawn-rate N    Users spawned per second (default: 2)"
            echo "  --help            Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                          # Run with defaults"
            echo "  $0 --duration 300 --users 50  # 5 min test with 50 users"
            echo "  $0 --duration 30 --users 5    # Quick 30s test"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     Harness Load Testing - Plugin Architecture Demo         ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}🎯 Target:${NC}        ${TARGET_VM_EXTERNAL}"
echo -e "${BLUE}⏱️  Duration:${NC}      ${DURATION} seconds"
echo -e "${BLUE}👥 Users:${NC}         ${USERS}"
echo -e "${BLUE}📈 Spawn Rate:${NC}    ${SPAWN_RATE}/sec"
echo -e "${BLUE}📝 Script:${NC}        Clean user script (auto-enhanced)"
echo ""

# Pre-flight checks
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}🔍 Pre-flight Checks${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"

# Check if Locust is running
echo -e "${CYAN}[1/4]${NC} Checking Locust..."
if curl -s --connect-timeout 3 "http://localhost:8089" > /dev/null 2>&1; then
    echo -e "${GREEN}      ✓ Locust is running at http://localhost:8089${NC}"
else
    echo -e "${RED}      ✗ Locust is not running${NC}"
    echo ""
    echo -e "${YELLOW}      Start Locust with:${NC}"
    echo -e "${YELLOW}      cd locust${NC}"
    echo -e "${YELLOW}      export CONTROL_PLANE_URL=\"http://localhost:8080\"${NC}"
    echo -e "${YELLOW}      export CONTROL_PLANE_TOKEN=\"secure-token-vm-test-2025\"${NC}"
    echo -e "${YELLOW}      export METRICS_PUSH_INTERVAL=\"10\"${NC}"
    echo -e "${YELLOW}      locust -f vm-products-api-clean.py --host http://35.239.233.230:8000${NC}"
    exit 1
fi

# Check control plane
echo -e "${CYAN}[2/4]${NC} Checking Control Plane..."
if curl -s "${CONTROL_PLANE_URL}/health" > /dev/null 2>&1; then
    echo -e "${GREEN}      ✓ Control Plane is running at ${CONTROL_PLANE_URL}${NC}"
else
    echo -e "${RED}      ✗ Control Plane is not running${NC}"
    echo ""
    echo -e "${YELLOW}      Start it with:${NC}"
    echo -e "${YELLOW}      ./bin/controlplane -config config/vm-test-config.yaml${NC}"
    exit 1
fi

# Check VM connectivity
echo -e "${CYAN}[3/4]${NC} Checking VM connectivity..."
if curl -s --connect-timeout 5 "${TARGET_VM_EXTERNAL}/api/products" > /dev/null 2>&1; then
    echo -e "${GREEN}      ✓ VM is reachable and responding${NC}"
else
    echo -e "${RED}      ✗ Cannot reach VM at ${TARGET_VM_EXTERNAL}${NC}"
    echo -e "${YELLOW}      Check:${NC}"
    echo -e "${YELLOW}      1. VM is running${NC}"
    echo -e "${YELLOW}      2. Application is started on port 8000${NC}"
    echo -e "${YELLOW}      3. Firewall allows connections${NC}"
    exit 1
fi

# Check if clean script exists
echo -e "${CYAN}[4/4]${NC} Checking test script..."
SCRIPT_PATH="locust/vm-products-api-clean.py"
if [ ! -f "$SCRIPT_PATH" ]; then
    echo -e "${RED}      ✗ Clean script not found at: $SCRIPT_PATH${NC}"
    exit 1
fi
echo -e "${GREEN}      ✓ Test script found: $SCRIPT_PATH${NC}"

echo ""
echo -e "${GREEN}✅ All pre-flight checks passed!${NC}"
echo ""

# Create load test
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}🚀 Creating Load Test${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"

# Base64 encode the CLEAN script (no control plane code)
SCRIPT_CONTENT_B64=$(base64 -i "$SCRIPT_PATH" | tr -d '\n')
echo -e "${CYAN}📦 Clean script encoded (${#SCRIPT_CONTENT_B64} bytes)${NC}"
echo -e "${CYAN}   Control plane will auto-inject Harness plugin${NC}"

# Create load test with jq for proper JSON handling
JSON_PAYLOAD=$(jq -n \
  --arg name "VM Products API - Plugin Test $(date +%Y%m%d-%H%M%S)" \
  --arg desc "Clean user script with auto plugin injection" \
  --arg accountId "my-account" \
  --arg orgId "my-org" \
  --arg projectId "products-api" \
  --arg envId "vm-test" \
  --arg targetUrl "${TARGET_VM_EXTERNAL}" \
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
    defaultUsers: $users,
    defaultSpawnRate: $spawnRate,
    defaultDurationSec: $duration,
    scriptContent: $scriptContent,
    createdBy: "run-vm-test-plugin.sh"
  }')

echo -e "${CYAN}📤 Sending to control plane...${NC}"
CREATE_RESPONSE=$(curl -s -X POST "${CONTROL_PLANE_URL}/v1/load-tests" \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "${JSON_PAYLOAD}")

# Verify creation
if echo "$CREATE_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    LOAD_TEST_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id')
    echo -e "${GREEN}✓ Load test created successfully${NC}"
    echo -e "${CYAN}   Test ID: ${LOAD_TEST_ID}${NC}"
else
    echo -e "${RED}✗ Failed to create load test${NC}"
    echo -e "${RED}Response:${NC}"
    echo "$CREATE_RESPONSE" | jq '.' 2>/dev/null || echo "$CREATE_RESPONSE"
    exit 1
fi

echo ""

# Start test run
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}🏃 Starting Test Run${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"

RUN_RESPONSE=$(curl -s -X POST "${CONTROL_PLANE_URL}/v1/load-tests/${LOAD_TEST_ID}/runs" \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"targetUsers\": ${USERS},
    \"spawnRate\": ${SPAWN_RATE},
    \"durationSeconds\": ${DURATION},
    \"triggeredBy\": \"run-vm-test-plugin.sh\"
  }")

# Verify run started
if echo "$RUN_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
    RUN_ID=$(echo "$RUN_RESPONSE" | jq -r '.id')
    echo -e "${GREEN}✓ Test run started successfully${NC}"
    echo -e "${CYAN}   Run ID: ${RUN_ID}${NC}"
else
    echo -e "${RED}✗ Failed to start test run${NC}"
    echo -e "${RED}Response:${NC}"
    echo "$RUN_RESPONSE" | jq '.' 2>/dev/null || echo "$RUN_RESPONSE"
    exit 1
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}           🎉 TEST IS NOW RUNNING 🎉${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Display monitoring info
echo -e "${YELLOW}📊 Monitor Your Test:${NC}"
echo ""
echo -e "  ${CYAN}Locust Web UI:${NC}"
echo -e "    → http://localhost:8089"
echo -e "    ${GREEN}(Watch real-time metrics, users, RPS)${NC}"
echo ""
echo -e "  ${CYAN}Control Plane Swagger:${NC}"
echo -e "    → http://localhost:8080/swagger/index.html"
echo -e "    ${GREEN}(View API endpoints and test data)${NC}"
echo ""
echo -e "  ${CYAN}Run Details:${NC}"
echo -e "    → curl http://localhost:8080/v1/runs/${RUN_ID} \\"
echo -e "         -H 'Authorization: Bearer ${API_TOKEN}' | jq ."
echo ""

# Progress bar during test
echo -e "${YELLOW}⏱️  Test Progress:${NC}"
echo ""

PROGRESS_WIDTH=50
for i in $(seq 1 $DURATION); do
    # Calculate progress
    PERCENT=$((i * 100 / DURATION))
    FILLED=$((PERCENT * PROGRESS_WIDTH / 100))
    
    # Build progress bar
    BAR="["
    for j in $(seq 1 $PROGRESS_WIDTH); do
        if [ $j -le $FILLED ]; then
            BAR="${BAR}█"
        else
            BAR="${BAR}░"
        fi
    done
    BAR="${BAR}]"
    
    # Print progress (overwrite same line)
    printf "\r  ${CYAN}${BAR} ${PERCENT}%% ${NC}(${i}/${DURATION}s)"
    
    sleep 1
done

echo ""
echo ""
echo -e "${GREEN}✓ Test duration completed${NC}"
echo ""

# Wait for final metrics
echo -e "${YELLOW}⏳ Collecting final metrics...${NC}"
sleep 5

# Fetch results
echo ""
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${YELLOW}📊 Fetching Results${NC}"
echo -e "${YELLOW}═══════════════════════════════════════════════════════════════${NC}"

SUMMARY=$(curl -s "${CONTROL_PLANE_URL}/v1/visualization/runs/${RUN_ID}/summary" \
  -H "Authorization: Bearer ${API_TOKEN}")

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}                   TEST RESULTS SUMMARY${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

if echo "$SUMMARY" | jq -e '.summary' > /dev/null 2>&1; then
    TOTAL_REQUESTS=$(echo "$SUMMARY" | jq -r '.summary.totalRequests // 0')
    TOTAL_RPS=$(echo "$SUMMARY" | jq -r '.summary.requestsPerSecond // 0')
    ERROR_RATE=$(echo "$SUMMARY" | jq -r '.summary.errorRate // 0')
    AVG_RESPONSE=$(echo "$SUMMARY" | jq -r '.summary.avgResponseTime // 0')
    P95_RESPONSE=$(echo "$SUMMARY" | jq -r '.summary.p95ResponseTime // 0')
    P99_RESPONSE=$(echo "$SUMMARY" | jq -r '.summary.p99ResponseTime // 0')
    
    echo -e "${YELLOW}📈 Performance Metrics:${NC}"
    echo -e "   Total Requests:       ${CYAN}${TOTAL_REQUESTS}${NC}"
    echo -e "   Requests/Second:      ${CYAN}${TOTAL_RPS}${NC}"
    echo -e "   Error Rate:           ${CYAN}${ERROR_RATE}%${NC}"
    echo ""
    echo -e "${YELLOW}⚡ Response Times:${NC}"
    echo -e "   Average:              ${CYAN}${AVG_RESPONSE}ms${NC}"
    echo -e "   95th Percentile:      ${CYAN}${P95_RESPONSE}ms${NC}"
    echo -e "   99th Percentile:      ${CYAN}${P99_RESPONSE}ms${NC}"
    echo ""
    
    # Status assessment
    ERROR_RATE_NUM=$(echo "$ERROR_RATE" | bc -l 2>/dev/null || echo "0")
    if (( $(echo "$ERROR_RATE_NUM < 1.0" | bc -l 2>/dev/null || echo "0") )); then
        echo -e "${GREEN}✅ PASS - Test completed with low error rate${NC}"
    elif (( $(echo "$ERROR_RATE_NUM < 5.0" | bc -l 2>/dev/null || echo "0") )); then
        echo -e "${YELLOW}⚠️  WARN - Test completed with moderate error rate${NC}"
    else
        echo -e "${RED}❌ FAIL - Test completed with high error rate${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Summary not available yet${NC}"
    echo -e "${YELLOW}   Metrics may still be processing${NC}"
fi

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════${NC}"
echo ""

# Additional info
echo -e "${YELLOW}🔍 View Detailed Results:${NC}"
echo ""
echo -e "  ${CYAN}MongoDB Compass:${NC}"
echo -e "    mongodb://localhost:27017"
echo -e "    Database: ${GREEN}load_testing${NC}"
echo -e "    Collections:"
echo -e "      → ${GREEN}load_test_runs${NC} (run details and status)"
echo -e "      → ${GREEN}metrics_timeseries${NC} (historical metrics)"
echo ""
echo -e "  ${CYAN}API Endpoints:${NC}"
echo -e "    → Summary:  /v1/visualization/runs/${RUN_ID}/summary"
echo -e "    → Timeline: /v1/visualization/runs/${RUN_ID}/timeline"
echo -e "    → Graph:    /v1/visualization/runs/${RUN_ID}/graph"
echo ""

echo -e "${YELLOW}💾 Test Artifacts:${NC}"
echo -e "   Load Test ID: ${CYAN}${LOAD_TEST_ID}${NC}"
echo -e "   Run ID:       ${CYAN}${RUN_ID}${NC}"
echo ""

echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                  ✅ TEST COMPLETE ✅                          ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
