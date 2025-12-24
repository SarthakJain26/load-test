# Testing Setup Guide - Mac Control Plane with Linux VM Target

## Overview

This guide will help you set up and run load tests with:
- **Control Plane**: Running on your Mac
- **MongoDB**: Running on your Mac (accessible via MongoDB Compass)
- **Locust**: Running on your Mac (can also run on Linux VM if needed)
- **Target Application**: Running on your Linux VM

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [MongoDB Setup](#mongodb-setup)
3. [Control Plane Setup](#control-plane-setup)
4. [Locust Setup](#locust-setup)
5. [Configuration Files](#configuration-files)
6. [Running Your First Load Test](#running-your-first-load-test)
7. [Viewing Results](#viewing-results)
8. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### On Your Mac

**Install MongoDB:**
```bash
# Using Homebrew
brew tap mongodb/brew
brew install mongodb-community@7.0

# Start MongoDB service
brew services start mongodb-community@7.0

# Verify it's running
mongosh --eval "db.adminCommand('ping')"
```

**Install Go (1.22+):**
```bash
# Using Homebrew
brew install go

# Verify installation
go version
```

**Install Python 3.9+ and Locust:**
```bash
# Check Python version
python3 --version

# Install Locust
pip3 install locust gevent requests

# Verify installation
locust --version
```

**Install jq (for JSON parsing in terminal):**
```bash
brew install jq
```

### On Your Linux VM

**Note:** We only need the application running. No additional software required on the VM for this setup.

**Get VM Details:**
- IP address or hostname (e.g., `192.168.1.100` or `myapp.local`)
- Application port (e.g., `8080`)
- Test endpoints you want to load test (e.g., `/api/users`, `/api/products`)

---

## MongoDB Setup

### 1. Connect MongoDB Compass

**MongoDB Connection Details:**
- **Host**: `localhost`
- **Port**: `27017`
- **Connection String**: `mongodb://localhost:27017`

**Steps:**
1. Open **MongoDB Compass**
2. Click **"New Connection"**
3. Enter connection string: `mongodb://localhost:27017`
4. Click **"Connect"**

**Expected Databases:**
After running the control plane, you'll see a database named `load_testing` with these collections:
- `load_tests` - Test configurations
- `load_test_runs` - Test execution records
- `script_revisions` - Script version history
- `metrics` - Time-series metrics data

### 2. Verify MongoDB Connection

```bash
# Test connection from terminal
mongosh mongodb://localhost:27017

# Inside mongosh, list databases
show dbs

# Exit
exit
```

---

## Control Plane Setup

### 1. Build the Control Plane

```bash
cd /Users/sarthakjain/harness/Load-manager-cli

# Build the binary
go build -o bin/controlplane cmd/controlplane/main.go

# Verify build
ls -lh bin/controlplane
```

### 2. Create Configuration File

**File:** `config/my-test-config.yaml`

```yaml
# Control Plane Configuration for Mac Testing Setup

server:
  host: "0.0.0.0"
  port: 8080

# Locust clusters configuration
# Update the baseUrl to match where your Locust master will run
locustClusters:
  - id: "local-test-cluster"
    baseUrl: "http://localhost:8089"
    accountId: "my-account"
    orgId: "my-org"
    projectId: "my-project"
    envId: "test"
    authToken: ""

# Security tokens
security:
  # Token that Locust uses to authenticate callbacks to control plane
  # Must match CONTROL_PLANE_TOKEN in Locust environment
  locustCallbackToken: "test-secret-token-12345"
  
  # API token for making API calls to control plane
  apiToken: "my-api-token-67890"

# MongoDB configuration
mongodb:
  uri: "mongodb://localhost:27017"
  database: "load_testing"
  connectTimeoutSeconds: 10
  maxPoolSize: 100
```

**Save this file as:** `/Users/sarthakjain/harness/Load-manager-cli/config/my-test-config.yaml`

### 3. Start the Control Plane

```bash
cd /Users/sarthakjain/harness/Load-manager-cli

# Start the control plane
./bin/controlplane -config config/my-test-config.yaml
```

**Expected Output:**
```
2025/12/23 13:15:00 Starting Load Manager Control Plane...
2025/12/23 13:15:00 Connected to MongoDB: load_testing
2025/12/23 13:15:00 Orchestrator started (push-based metrics mode)
2025/12/23 13:15:00 Server listening on 0.0.0.0:8080
2025/12/23 13:15:00 Swagger UI available at: http://localhost:8080/swagger/index.html
```

### 4. Verify Control Plane

**Open a new terminal:**

```bash
# Health check
curl http://localhost:8080/health

# Expected: {"status":"healthy"}

# View Swagger documentation
open http://localhost:8080/swagger/index.html
```

**Leave the control plane running in the first terminal.**

---

## Locust Setup

### 1. Create Custom Locustfile for Your Linux VM

**File:** `locust/vm-test-locustfile.py`

```python
"""
Locustfile for testing application on Linux VM
Update the TARGET_HOST environment variable to match your VM's address
"""

import os
import time
import json
import logging
from typing import Optional

import gevent
import requests
from locust import HttpUser, task, between, events
from locust.env import Environment

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Control plane configuration from environment variables
CONTROL_PLANE_URL = os.getenv("CONTROL_PLANE_URL", "")
CONTROL_PLANE_TOKEN = os.getenv("CONTROL_PLANE_TOKEN", "")
RUN_ID = os.getenv("RUN_ID", "")
TENANT_ID = os.getenv("TENANT_ID", "")
ENV_ID = os.getenv("ENV_ID", "")

# Metrics push interval in seconds
METRICS_PUSH_INTERVAL = int(os.getenv("METRICS_PUSH_INTERVAL", "10"))

# Duration in seconds (if set, test will auto-stop after this duration)
DURATION_SECONDS = os.getenv("DURATION_SECONDS", "")

# Global greenlet references
_metrics_greenlet: Optional[gevent.Greenlet] = None
_duration_monitor_greenlet: Optional[gevent.Greenlet] = None
_test_start_time: Optional[float] = None


def _control_plane_headers():
    """Returns headers for control plane API calls."""
    return {
        "X-Locust-Token": CONTROL_PLANE_TOKEN,
        "Content-Type": "application/json",
    }


def _is_control_plane_enabled():
    """Check if control plane integration is enabled."""
    return bool(CONTROL_PLANE_URL and CONTROL_PLANE_TOKEN and RUN_ID)


@events.test_start.add_listener
def on_test_start(environment: Environment, **kwargs):
    """Event handler called when a load test starts."""
    global _test_start_time
    _test_start_time = time.time()
    
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, skipping test_start callback")
        return
    
    logger.info(f"Test started, notifying control plane (RUN_ID={RUN_ID})")
    
    try:
        payload = {
            "runId": RUN_ID,
            "tenantId": TENANT_ID,
            "envId": ENV_ID,
        }
        
        url = f"{CONTROL_PLANE_URL}/v1/internal/locust/test-start"
        response = requests.post(
            url,
            json=payload,
            headers=_control_plane_headers(),
            timeout=10
        )
        response.raise_for_status()
        logger.info("Successfully notified control plane of test start")
    
    except Exception as e:
        logger.error(f"Failed to notify control plane of test start: {e}")


@events.test_stop.add_listener
def on_test_stop(environment: Environment, **kwargs):
    """Event handler called when a load test stops."""
    global _metrics_greenlet, _duration_monitor_greenlet
    
    # Stop the metrics pusher greenlet
    if _metrics_greenlet is not None:
        logger.info("Stopping metrics pusher greenlet")
        gevent.kill(_metrics_greenlet)
        _metrics_greenlet = None
    
    # Stop the duration monitor greenlet
    if _duration_monitor_greenlet is not None:
        logger.info("Stopping duration monitor greenlet")
        gevent.kill(_duration_monitor_greenlet)
        _duration_monitor_greenlet = None
    
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, skipping test_stop callback")
        return
    
    logger.info(f"Test stopped, notifying control plane with final metrics (RUN_ID={RUN_ID})")
    
    try:
        # Collect final metrics
        final_metrics = _collect_metrics(environment)
        
        payload = {
            "runId": RUN_ID,
            "tenantId": TENANT_ID,
            "envId": ENV_ID,
            "finalMetrics": final_metrics,
        }
        
        url = f"{CONTROL_PLANE_URL}/v1/internal/locust/test-stop"
        response = requests.post(
            url,
            json=payload,
            headers=_control_plane_headers(),
            timeout=10
        )
        response.raise_for_status()
        logger.info("Successfully notified control plane of test stop")
    
    except Exception as e:
        logger.error(f"Failed to notify control plane of test stop: {e}")


def _collect_metrics(environment: Environment) -> dict:
    """Collects current metrics from Locust environment."""
    stats = environment.stats
    
    # Calculate aggregate metrics
    total_rps = stats.total.current_rps if stats.total else 0
    total_requests = stats.total.num_requests if stats.total else 0
    total_failures = stats.total.num_failures if stats.total else 0
    error_rate = (total_failures / total_requests * 100) if total_requests > 0 else 0
    
    # Get response time percentiles
    p50 = stats.total.get_response_time_percentile(0.5) if stats.total else 0
    p95 = stats.total.get_response_time_percentile(0.95) if stats.total else 0
    p99 = stats.total.get_response_time_percentile(0.99) if stats.total else 0
    avg_response = stats.total.avg_response_time if stats.total else 0
    
    # Current user count
    current_users = environment.runner.user_count if environment.runner else 0
    
    # Per-request statistics
    request_stats = {}
    for name, stat in stats.entries.items():
        method, endpoint = name
        key = f"{method}_{endpoint}"
        request_stats[key] = {
            "method": method,
            "name": endpoint,
            "numRequests": stat.num_requests,
            "numFailures": stat.num_failures,
            "avgResponseTime": stat.avg_response_time,
            "minResponseTime": stat.min_response_time or 0,
            "maxResponseTime": stat.max_response_time or 0,
            "medianResponseTime": stat.median_response_time or 0,
            "requestsPerSec": stat.current_rps,
        }
    
    return {
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "totalRps": total_rps,
        "totalRequests": total_requests,
        "totalFailures": total_failures,
        "errorRate": error_rate,
        "avgResponseMs": avg_response,
        "p50ResponseMs": p50,
        "p95ResponseMs": p95,
        "p99ResponseMs": p99,
        "currentUsers": current_users,
        "requestStats": request_stats,
    }


def _metrics_pusher(environment: Environment):
    """Background task that periodically pushes metrics to the control plane."""
    logger.info(f"Starting metrics pusher (interval: {METRICS_PUSH_INTERVAL}s)")
    
    while True:
        try:
            gevent.sleep(METRICS_PUSH_INTERVAL)
            
            if not _is_control_plane_enabled():
                continue
            
            # Collect and send metrics
            metrics = _collect_metrics(environment)
            
            payload = {
                "runId": RUN_ID,
                "metrics": metrics,
            }
            
            url = f"{CONTROL_PLANE_URL}/v1/internal/locust/metrics"
            response = requests.post(
                url,
                json=payload,
                headers=_control_plane_headers(),
                timeout=5
            )
            response.raise_for_status()
            
            logger.debug(f"Pushed metrics to control plane (RPS: {metrics['totalRps']:.2f})")
        
        except gevent.GreenletExit:
            logger.info("Metrics pusher greenlet killed")
            break
        except Exception as e:
            logger.error(f"Error pushing metrics to control plane: {e}")


def _duration_monitor(environment: Environment):
    """Background task that monitors test duration and stops the test when duration elapses."""
    if not DURATION_SECONDS:
        logger.info("No duration limit set, duration monitor disabled")
        return
    
    try:
        duration = int(DURATION_SECONDS)
        logger.info(f"Duration monitor started: will stop test after {duration} seconds")
        
        # Sleep until duration elapses
        gevent.sleep(duration)
        
        # Stop the test
        logger.info(f"Duration of {duration} seconds has elapsed, stopping test")
        environment.runner.quit()
        
    except gevent.GreenletExit:
        logger.info("Duration monitor greenlet killed")
    except Exception as e:
        logger.error(f"Error in duration monitor: {e}")


@events.test_start.add_listener
def start_background_greenlets(environment: Environment, **kwargs):
    """Spawns background greenlets for metrics pushing and duration monitoring."""
    global _metrics_greenlet, _duration_monitor_greenlet
    
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, background tasks disabled")
        return
    
    # Start metrics pusher
    _metrics_greenlet = gevent.spawn(_metrics_pusher, environment)
    logger.info("Metrics pusher greenlet started")
    
    # Start duration monitor if duration is configured
    if DURATION_SECONDS:
        _duration_monitor_greenlet = gevent.spawn(_duration_monitor, environment)
        logger.info("Duration monitor greenlet started")
    else:
        logger.info("No duration configured, test will run until manually stopped")


# ============================================================================
# Load Testing Scenarios for Your Linux VM Application
# ============================================================================

class VMApplicationUser(HttpUser):
    """
    Load testing user for the Linux VM application.
    
    IMPORTANT: Update the tasks below to match your actual API endpoints!
    """
    
    # Wait time between tasks (random between 1-3 seconds)
    wait_time = between(1, 3)
    
    @task(5)
    def test_health_endpoint(self):
        """Test health/status endpoint - REPLACE with your actual endpoint"""
        with self.client.get("/health", catch_response=True, name="GET /health") as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Health check failed: {response.status_code}")
    
    @task(3)
    def test_api_endpoint_1(self):
        """Test your first API endpoint - REPLACE with actual endpoint"""
        # Example: GET /api/users
        self.client.get("/api/users", name="GET /api/users")
    
    @task(2)
    def test_api_endpoint_2(self):
        """Test your second API endpoint - REPLACE with actual endpoint"""
        # Example: GET /api/products
        self.client.get("/api/products", name="GET /api/products")
    
    @task(1)
    def test_post_endpoint(self):
        """Test a POST endpoint - REPLACE with actual endpoint"""
        # Example: POST /api/data
        payload = {
            "test_data": "load_test",
            "timestamp": time.time()
        }
        self.client.post(
            "/api/data",
            json=payload,
            name="POST /api/data"
        )


# ============================================================================
# Add more user classes if needed for different user behaviors
# ============================================================================

class AdminUser(HttpUser):
    """Example of a different user type - OPTIONAL"""
    wait_time = between(2, 5)
    weight = 1  # Lower weight = fewer users
    
    @task
    def admin_dashboard(self):
        """Admin-specific endpoint"""
        self.client.get("/admin/dashboard", name="GET /admin/dashboard")
```

**Save this file as:** `/Users/sarthakjain/harness/Load-manager-cli/locust/vm-test-locustfile.py`

**IMPORTANT:** Edit the file and update the tasks to match your actual Linux VM application endpoints!

### 2. Start Locust

**Open a new terminal (Terminal 2):**

```bash
cd /Users/sarthakjain/harness/Load-manager-cli/locust

# Set environment variables
export CONTROL_PLANE_URL="http://localhost:8080"
export CONTROL_PLANE_TOKEN="test-secret-token-12345"
export RUN_ID="manual-test-run-1"
export DURATION_SECONDS="120"  # Test will auto-stop after 2 minutes
export METRICS_PUSH_INTERVAL="10"
export TENANT_ID="my-account"
export ENV_ID="test"

# IMPORTANT: Replace with your Linux VM's IP/hostname and port
export TARGET_HOST="http://192.168.1.100:8080"

# Start Locust master
locust -f vm-test-locustfile.py \
  --master \
  --web-host 0.0.0.0 \
  --web-port 8089 \
  --host $TARGET_HOST
```

**Expected Output:**
```
[2025-12-23 13:20:00,000] INFO/locust.main: Starting web interface at http://0.0.0.0:8089
[2025-12-23 13:20:00,000] INFO/locust.main: Starting Locust 2.x.x
```

### 3. Verify Locust Web UI

**Open browser:**
```
http://localhost:8089
```

You should see the Locust web interface.

**Leave Locust running in Terminal 2.**

---

## Configuration Files

### Summary of Files You Need

1. **Control Plane Config:** `config/my-test-config.yaml` (created above)
2. **Locust Script:** `locust/vm-test-locustfile.py` (created above)

### Environment Variables Reference

**For Locust (Terminal 2):**

| Variable | Value | Description |
|----------|-------|-------------|
| `CONTROL_PLANE_URL` | `http://localhost:8080` | Control plane address |
| `CONTROL_PLANE_TOKEN` | `test-secret-token-12345` | Must match config.yaml |
| `RUN_ID` | Set by control plane | Unique test run ID |
| `DURATION_SECONDS` | `120` | Auto-stop after 2 minutes |
| `METRICS_PUSH_INTERVAL` | `10` | Push metrics every 10 seconds |
| `TENANT_ID` | `my-account` | Account identifier |
| `ENV_ID` | `test` | Environment identifier |
| `TARGET_HOST` | `http://192.168.1.100:8080` | **YOUR LINUX VM ADDRESS** |

---

## Running Your First Load Test

Now that everything is set up, let's run a complete load test using the API.

### Step 1: Create a Load Test

**Open a new terminal (Terminal 3):**

```bash
# Set your API token
export API_TOKEN="my-api-token-67890"

# Create a load test with a simple script
curl -X POST http://localhost:8080/v1/load-tests \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Linux VM API Load Test",
    "description": "Testing my application on Linux VM",
    "accountId": "my-account",
    "orgId": "my-org",
    "projectId": "my-project",
    "envId": "test",
    "targetUrl": "http://192.168.1.100:8080",
    "targetUsers": 10,
    "spawnRate": 2,
    "durationSeconds": 120,
    "scriptContent": "IyBTaW1wbGUgdGVzdCBzY3JpcHQK"
  }' | jq '.'
```

**Save the `id` from the response!** You'll need it for the next step.

**Example Response:**
```json
{
  "id": "lt_abc123xyz",
  "name": "Linux VM API Load Test",
  "accountId": "my-account",
  ...
}
```

### Step 2: Start the Load Test

```bash
# Replace LOAD_TEST_ID with the ID from Step 1
export LOAD_TEST_ID="lt_abc123xyz"

# Start a test run
curl -X POST http://localhost:8080/v1/load-tests/$LOAD_TEST_ID/runs \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetUsers": 10,
    "spawnRate": 2,
    "durationSeconds": 120
  }' | jq '.'
```

**Example Response:**
```json
{
  "id": "run_xyz789abc",
  "loadTestId": "lt_abc123xyz",
  "status": "Running",
  "startedAt": 1703332800000,
  ...
}
```

**Save the `id` (run ID) for viewing metrics!**

### Step 3: Monitor the Test

**Watch Control Plane logs (Terminal 1):**
You should see:
```
Updated metrics for run run_xyz789abc: RPS=8.50, Requests=1020, Failures=0, Users=10
```

**Watch Locust logs (Terminal 2):**
You should see:
```
Pushed metrics to control plane (RPS: 8.50)
```

**View Locust Web UI:**
Open `http://localhost:8089` to see real-time statistics.

### Step 4: View Results via API

```bash
# Get run details
export RUN_ID="run_xyz789abc"

curl http://localhost:8080/v1/runs/$RUN_ID \
  -H "Authorization: Bearer $API_TOKEN" | jq '.'

# Get time-series metrics for charts
curl http://localhost:8080/v1/visualization/runs/$RUN_ID/timeseries \
  -H "Authorization: Bearer $API_TOKEN" | jq '.'

# Get summary statistics
curl http://localhost:8080/v1/visualization/runs/$RUN_ID/summary \
  -H "Authorization: Bearer $API_TOKEN" | jq '.'

# Get aggregated stats
curl http://localhost:8080/v1/visualization/runs/$RUN_ID/aggregated-stats \
  -H "Authorization: Bearer $API_TOKEN" | jq '.'
```

---

## Viewing Results

### Option 1: MongoDB Compass

1. Open MongoDB Compass
2. Connect to `mongodb://localhost:27017`
3. Navigate to `load_testing` database
4. View collections:
   - **load_test_runs**: See all test runs and their status
   - **metrics**: Time-series metrics data
   - **load_tests**: Test configurations

**Example Query in Compass:**
```javascript
// Find all running tests
{ "status": "Running" }

// Find tests by account
{ "accountId": "my-account" }
```

### Option 2: Swagger UI

1. Open `http://localhost:8080/swagger/index.html`
2. Click **"Authorize"** and enter your API token
3. Try out the visualization endpoints:
   - `GET /v1/runs/{id}` - Get run details
   - `GET /v1/visualization/runs/{id}/timeseries` - Time-series data
   - `GET /v1/visualization/runs/{id}/summary` - Summary stats

### Option 3: Command Line

```bash
# List all test runs
curl http://localhost:8080/v1/runs \
  -H "Authorization: Bearer $API_TOKEN" | jq '.[] | {id, name, status, startedAt}'

# Get specific run with metrics
curl http://localhost:8080/v1/runs/$RUN_ID \
  -H "Authorization: Bearer $API_TOKEN" | jq '.lastMetrics'
```

---

## Complete Example Workflow

Here's a complete bash script to run everything:

**File:** `run-load-test.sh`

```bash
#!/bin/bash

# Configuration
API_TOKEN="my-api-token-67890"
CONTROL_PLANE_URL="http://localhost:8080"
TARGET_VM="http://192.168.1.100:8080"  # UPDATE THIS!

echo "🚀 Creating load test..."
CREATE_RESPONSE=$(curl -s -X POST $CONTROL_PLANE_URL/v1/load-tests \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"VM Test - $(date +%Y%m%d-%H%M%S)\",
    \"description\": \"Automated load test\",
    \"accountId\": \"my-account\",
    \"orgId\": \"my-org\",
    \"projectId\": \"my-project\",
    \"envId\": \"test\",
    \"targetUrl\": \"$TARGET_VM\",
    \"targetUsers\": 10,
    \"spawnRate\": 2,
    \"durationSeconds\": 60,
    \"scriptContent\": \"IyBTaW1wbGUgdGVzdAo=\"
  }")

LOAD_TEST_ID=$(echo $CREATE_RESPONSE | jq -r '.id')
echo "✅ Load test created: $LOAD_TEST_ID"

echo ""
echo "🏃 Starting test run..."
RUN_RESPONSE=$(curl -s -X POST $CONTROL_PLANE_URL/v1/load-tests/$LOAD_TEST_ID/runs \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetUsers": 10,
    "spawnRate": 2,
    "durationSeconds": 60
  }')

RUN_ID=$(echo $RUN_RESPONSE | jq -r '.id')
echo "✅ Test run started: $RUN_ID"

echo ""
echo "⏳ Test will run for 60 seconds..."
echo "📊 View live stats at: http://localhost:8089"
echo "📈 View metrics: $CONTROL_PLANE_URL/v1/visualization/runs/$RUN_ID/timeseries"

# Wait for test to complete
sleep 65

echo ""
echo "📊 Fetching results..."
curl -s $CONTROL_PLANE_URL/v1/visualization/runs/$RUN_ID/summary \
  -H "Authorization: Bearer $API_TOKEN" | jq '.summary'

echo ""
echo "✅ Test complete! View full results in MongoDB Compass or Swagger UI"
```

**Make it executable and run:**
```bash
chmod +x run-load-test.sh
./run-load-test.sh
```

---

## Troubleshooting

### Issue 1: Control Plane Can't Connect to MongoDB

**Symptoms:**
```
Error connecting to MongoDB: connection refused
```

**Solution:**
```bash
# Check if MongoDB is running
brew services list | grep mongodb

# Start MongoDB if not running
brew services start mongodb-community@7.0

# Test connection
mongosh mongodb://localhost:27017
```

### Issue 2: Locust Can't Reach Linux VM

**Symptoms:**
```
Connection refused or timeout errors in Locust
```

**Solution:**
```bash
# Test connectivity from Mac to Linux VM
ping 192.168.1.100

# Test specific port
curl -v http://192.168.1.100:8080/health

# Check Linux VM firewall
# On Linux VM:
sudo ufw status
sudo ufw allow 8080/tcp
```

### Issue 3: Metrics Not Showing in Control Plane

**Symptoms:**
- Locust running but no metrics in MongoDB

**Solution:**
```bash
# Check Locust logs for errors
# Look for "Pushed metrics to control plane" messages

# Verify environment variables match
echo $CONTROL_PLANE_TOKEN  # Should match config.yaml
echo $CONTROL_PLANE_URL    # Should be http://localhost:8080

# Check control plane logs for incoming metrics
# Look for "Updated metrics for run" messages
```

### Issue 4: MongoDB Compass Shows Empty Collections

**Symptoms:**
- Database exists but collections are empty

**Solution:**
```bash
# Create a test run first (see "Running Your First Load Test" section)
# Collections are created when first data is written

# Verify control plane can write to MongoDB
# Check control plane logs for MongoDB errors
```

### Issue 5: Authentication Errors

**Symptoms:**
```
401 Unauthorized
```

**Solution:**
```bash
# Verify API token matches config.yaml
grep apiToken config/my-test-config.yaml

# Use correct token in API calls
export API_TOKEN="my-api-token-67890"

# Check Authorization header format
curl -v http://localhost:8080/v1/load-tests \
  -H "Authorization: Bearer $API_TOKEN"
```

---

## Next Steps

### 1. Customize Your Locustfile

Edit `locust/vm-test-locustfile.py` to match your actual application:

```python
@task(10)
def test_my_specific_endpoint(self):
    """Test your specific endpoint"""
    self.client.get("/api/your-endpoint", name="GET /api/your-endpoint")
```

### 2. Create Multiple Test Scenarios

Create different locustfiles for different test scenarios:
- `locust/stress-test.py` - High load testing
- `locust/spike-test.py` - Sudden traffic spikes
- `locust/endurance-test.py` - Long-running tests

### 3. Automate with Scripts

Create shell scripts to:
- Run daily smoke tests
- Compare performance over time
- Alert on performance degradation

### 4. Integrate with CI/CD

Add load tests to your CI/CD pipeline:
```bash
# In your CI pipeline
./run-load-test.sh
# Check exit code and metrics
```

---

## Quick Reference

### Terminal Layout

- **Terminal 1**: Control Plane (`./bin/controlplane -config config/my-test-config.yaml`)
- **Terminal 2**: Locust (`locust -f vm-test-locustfile.py --master --host $TARGET_HOST`)
- **Terminal 3**: API calls and commands

### Important URLs

- Control Plane API: `http://localhost:8080`
- Swagger UI: `http://localhost:8080/swagger/index.html`
- Locust Web UI: `http://localhost:8089`
- MongoDB Compass: `mongodb://localhost:27017`

### Key Commands

```bash
# Build control plane
go build -o bin/controlplane cmd/controlplane/main.go

# Start control plane
./bin/controlplane -config config/my-test-config.yaml

# Start Locust
locust -f vm-test-locustfile.py --master --host http://192.168.1.100:8080

# Create test
curl -X POST http://localhost:8080/v1/load-tests -H "Authorization: Bearer $API_TOKEN" -d '{...}'

# Start run
curl -X POST http://localhost:8080/v1/load-tests/{id}/runs -H "Authorization: Bearer $API_TOKEN" -d '{...}'

# Get results
curl http://localhost:8080/v1/runs/{id} -H "Authorization: Bearer $API_TOKEN"
```

---

## Support

For issues or questions:
1. Check the [Troubleshooting](#troubleshooting) section
2. Review control plane and Locust logs
3. Verify all services are running
4. Check MongoDB Compass for data

Happy load testing! 🚀
