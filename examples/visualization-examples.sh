#!/bin/bash

################################################################################
# Visualization API Examples
# Demonstrates how to use the MongoDB-backed visualization endpoints
################################################################################

set -e

# Configuration
CONTROL_PLANE_URL="http://localhost:8080"
API_TOKEN="your-api-token-here"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Load Testing Control Plane - Visualization API Examples${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Get test ID from user
echo -e "${YELLOW}Enter the test run ID (or press Enter to use example):${NC}"
read -r TEST_ID
if [ -z "$TEST_ID" ]; then
    TEST_ID="example-test-id-123"
    echo -e "${GREEN}Using example test ID: $TEST_ID${NC}"
fi
echo ""

################################################################################
# Example 1: Get Time-Series Data for Line Charts
################################################################################
echo -e "${GREEN}═══ Example 1: Time-Series Data (Line Charts) ═══${NC}"
echo ""
echo "This endpoint returns data suitable for plotting:"
echo "  - RPS over time"
echo "  - Latency (P50, P95, P99) over time"
echo "  - Error rate over time"
echo "  - User count over time"
echo ""
echo -e "${YELLOW}Request:${NC}"
echo "GET /v1/tests/${TEST_ID}/metrics/timeseries"
echo ""

curl -s -X GET "${CONTROL_PLANE_URL}/v1/tests/${TEST_ID}/metrics/timeseries" \
  -H "Authorization: Bearer ${API_TOKEN}" | jq '{
    testRunId,
    totalDataPoints: (.dataPoints | length),
    firstPoint: .dataPoints[0],
    lastPoint: .dataPoints[-1],
    summary
  }'

echo ""
echo -e "${GREEN}✓ Use this data to plot line charts in your dashboard${NC}"
echo ""
read -p "Press Enter to continue..."
echo ""

################################################################################
# Example 2: Get Time-Series Data with Time Range Filter
################################################################################
echo -e "${GREEN}═══ Example 2: Time-Series Data with Time Range ═══${NC}"
echo ""
echo "Filter metrics by time range for specific periods"
echo ""

# Get current time and 1 hour ago
TO_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
FROM_TIME=$(date -u -v-1H +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || date -u -d '1 hour ago' +"%Y-%m-%dT%H:%M:%SZ")

echo -e "${YELLOW}Request:${NC}"
echo "GET /v1/tests/${TEST_ID}/metrics/timeseries?from=${FROM_TIME}&to=${TO_TIME}"
echo ""

curl -s -X GET "${CONTROL_PLANE_URL}/v1/tests/${TEST_ID}/metrics/timeseries?from=${FROM_TIME}&to=${TO_TIME}" \
  -H "Authorization: Bearer ${API_TOKEN}" | jq '{
    testRunId,
    timeRange: {
      from: "'${FROM_TIME}'",
      to: "'${TO_TIME}'"
    },
    dataPointsInRange: (.dataPoints | length),
    summary
  }'

echo ""
echo -e "${GREEN}✓ Filter by time range for zoomed-in views${NC}"
echo ""
read -p "Press Enter to continue..."
echo ""

################################################################################
# Example 3: Get Scatter Plot Data
################################################################################
echo -e "${GREEN}═══ Example 3: Scatter Plot Data ═══${NC}"
echo ""
echo "This endpoint returns per-endpoint response time distribution"
echo "Perfect for scatter plots showing performance by endpoint"
echo ""
echo -e "${YELLOW}Request:${NC}"
echo "GET /v1/tests/${TEST_ID}/metrics/scatter"
echo ""

curl -s -X GET "${CONTROL_PLANE_URL}/v1/tests/${TEST_ID}/metrics/scatter" \
  -H "Authorization: Bearer ${API_TOKEN}" | jq '{
    testRunId,
    totalDataPoints: (.dataPoints | length),
    uniqueEndpoints: (.endpoints | length),
    endpoints,
    samplePoints: .dataPoints[0:3]
  }'

echo ""
echo -e "${GREEN}✓ Use this data to plot scatter plots by endpoint${NC}"
echo ""
read -p "Press Enter to continue..."
echo ""

################################################################################
# Example 4: Get Aggregated Statistics
################################################################################
echo -e "${GREEN}═══ Example 4: Aggregated Statistics (Complete Dashboard) ═══${NC}"
echo ""
echo "This endpoint returns comprehensive data for complete dashboards:"
echo "  - Time-series data"
echo "  - Per-endpoint statistics"
echo "  - Aggregated summary"
echo ""
echo -e "${YELLOW}Request:${NC}"
echo "GET /v1/tests/${TEST_ID}/metrics/aggregate"
echo ""

curl -s -X GET "${CONTROL_PLANE_URL}/v1/tests/${TEST_ID}/metrics/aggregate" \
  -H "Authorization: Bearer ${API_TOKEN}" | jq '{
    testRunId,
    status,
    timeseriesPoints: (.timeseries | length),
    endpointStats: .endpointStats[0:2],
    summary
  }'

echo ""
echo -e "${GREEN}✓ All-in-one endpoint for dashboard views${NC}"
echo ""
read -p "Press Enter to continue..."
echo ""

################################################################################
# Example 5: Real-World Dashboard Queries
################################################################################
echo -e "${GREEN}═══ Example 5: Real-World Dashboard Queries ═══${NC}"
echo ""

# Query 1: Get average RPS over time
echo -e "${YELLOW}Query 1: Extract RPS values for charting${NC}"
curl -s -X GET "${CONTROL_PLANE_URL}/v1/tests/${TEST_ID}/metrics/timeseries" \
  -H "Authorization: Bearer ${API_TOKEN}" | jq '[.dataPoints[] | {
    time: .timestamp,
    rps: .totalRps
  }]' | head -20

echo ""
echo -e "${GREEN}✓ Use for RPS line chart${NC}"
echo ""

# Query 2: Get P95 latency over time
echo -e "${YELLOW}Query 2: Extract P95 latency for charting${NC}"
curl -s -X GET "${CONTROL_PLANE_URL}/v1/tests/${TEST_ID}/metrics/timeseries" \
  -H "Authorization: Bearer ${API_TOKEN}" | jq '[.dataPoints[] | {
    time: .timestamp,
    p95_latency: .p95ResponseMs
  }]' | head -20

echo ""
echo -e "${GREEN}✓ Use for latency line chart${NC}"
echo ""

# Query 3: Get per-endpoint performance
echo -e "${YELLOW}Query 3: Per-endpoint statistics${NC}"
curl -s -X GET "${CONTROL_PLANE_URL}/v1/tests/${TEST_ID}/metrics/aggregate" \
  -H "Authorization: Bearer ${API_TOKEN}" | jq '.endpointStats | map({
    endpoint: "\(.method) \(.endpoint)",
    avgLatency: .avgResponseTimeMs,
    p95Latency: .p95ResponseMs,
    errorRate: .errorRate,
    totalRequests: .totalRequests
  })'

echo ""
echo -e "${GREEN}✓ Use for endpoint comparison bar charts${NC}"
echo ""

################################################################################
# Example 6: Export Data for External Tools
################################################################################
echo -e "${GREEN}═══ Example 6: Export to CSV for Excel/Grafana ═══${NC}"
echo ""
echo "Exporting time-series data to CSV..."
echo ""

OUTPUT_FILE="test_${TEST_ID}_metrics.csv"

# Create CSV header
echo "timestamp,rps,users,p50_latency,p95_latency,p99_latency,error_rate" > "$OUTPUT_FILE"

# Extract and format data
curl -s -X GET "${CONTROL_PLANE_URL}/v1/tests/${TEST_ID}/metrics/timeseries" \
  -H "Authorization: Bearer ${API_TOKEN}" | jq -r '.dataPoints[] | 
    [.timestamp, .totalRps, .currentUsers, .p50ResponseMs, .p95ResponseMs, .p99ResponseMs, .errorRate] | 
    @csv' >> "$OUTPUT_FILE"

if [ -f "$OUTPUT_FILE" ]; then
    echo -e "${GREEN}✓ Data exported to: $OUTPUT_FILE${NC}"
    echo ""
    echo "First 5 rows:"
    head -6 "$OUTPUT_FILE"
    echo ""
    echo "You can now:"
    echo "  - Open in Excel/Google Sheets"
    echo "  - Import into Grafana"
    echo "  - Process with pandas/R"
else
    echo -e "${YELLOW}⚠ Export failed - test may not have data yet${NC}"
fi

echo ""

################################################################################
# Summary
################################################################################
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Summary - Visualization Endpoints${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo "Available endpoints:"
echo ""
echo -e "${YELLOW}1. Time-Series (Line Charts):${NC}"
echo "   GET /v1/tests/{id}/metrics/timeseries"
echo "   Use for: RPS, latency, error rate over time"
echo ""
echo -e "${YELLOW}2. Scatter Plot:${NC}"
echo "   GET /v1/tests/{id}/metrics/scatter"
echo "   Use for: Response time distribution by endpoint"
echo ""
echo -e "${YELLOW}3. Aggregated Stats:${NC}"
echo "   GET /v1/tests/{id}/metrics/aggregate"
echo "   Use for: Complete dashboard with all metrics"
echo ""
echo "All endpoints support:"
echo "  ✓ Real-time data during test execution"
echo "  ✓ Historical data after test completion"
echo "  ✓ Time-range filtering (?from=...&to=...)"
echo "  ✓ Multi-tenant isolation"
echo ""
echo -e "${GREEN}For more information, see: docs/MONGODB_SETUP.md${NC}"
echo ""
