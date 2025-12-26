#!/bin/bash

##############################################################################
# Start Locust with Plugin Architecture
# 
# This script starts Locust on Mac with the Harness plugin loaded.
# The plugin provides automatic control plane integration.
##############################################################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
CONTROL_PLANE_URL="http://localhost:8080"
CONTROL_PLANE_TOKEN="secure-token-vm-test-2025"
METRICS_PUSH_INTERVAL="10"
TARGET_HOST="http://35.239.233.230:8000"  # Linux VM
LOCUST_PORT="8089"

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║         Starting Locust with Harness Plugin                 ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if Locust is already running
if lsof -Pi :${LOCUST_PORT} -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}⚠️  Locust is already running on port ${LOCUST_PORT}${NC}"
    echo -e "${YELLOW}   Kill it? (y/n)${NC}"
    read -r response
    if [[ "$response" == "y" || "$response" == "Y" ]]; then
        PID=$(lsof -ti :${LOCUST_PORT})
        kill -9 $PID
        echo -e "${GREEN}✓ Stopped existing Locust instance${NC}"
        sleep 2
    else
        echo -e "${YELLOW}Exiting...${NC}"
        exit 0
    fi
fi

# Check if plugin exists
PLUGIN_PATH="locust/locust_harness_plugin.py"
if [ ! -f "$PLUGIN_PATH" ]; then
    echo -e "${RED}✗ Plugin not found at: $PLUGIN_PATH${NC}"
    echo -e "${YELLOW}  Make sure you're running from the project root${NC}"
    exit 1
fi

# Check if clean script exists
SCRIPT_PATH="locust/vm-products-api-clean.py"
if [ ! -f "$SCRIPT_PATH" ]; then
    echo -e "${RED}✗ Test script not found at: $SCRIPT_PATH${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Plugin and test script found${NC}"
echo ""

# Display configuration
echo -e "${YELLOW}Configuration:${NC}"
echo -e "  Control Plane:    ${CYAN}${CONTROL_PLANE_URL}${NC}"
echo -e "  Target Host:      ${CYAN}${TARGET_HOST}${NC}"
echo -e "  Locust UI:        ${CYAN}http://localhost:${LOCUST_PORT}${NC}"
echo -e "  Metrics Interval: ${CYAN}${METRICS_PUSH_INTERVAL}s${NC}"
echo ""

# Set environment variables
export CONTROL_PLANE_URL
export CONTROL_PLANE_TOKEN
export METRICS_PUSH_INTERVAL

# Add plugin to Python path
export PYTHONPATH="$(pwd)/locust:$PYTHONPATH"

echo -e "${GREEN}🚀 Starting Locust...${NC}"
echo ""

# Start Locust
cd locust || exit 1

echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
echo -e "${CYAN}Locust Web UI will be available at: ${GREEN}http://localhost:${LOCUST_PORT}${NC}"
echo -e "${CYAN}════════════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${YELLOW}Note: This window will show Locust logs${NC}"
echo -e "${YELLOW}      Press Ctrl+C to stop Locust${NC}"
echo ""

# Run Locust with the clean script
locust -f vm-products-api-clean.py \
  --web-host 0.0.0.0 \
  --web-port ${LOCUST_PORT} \
  --host ${TARGET_HOST}
