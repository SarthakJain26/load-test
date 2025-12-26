# Plugin Architecture Setup Guide

Complete setup guide for running Locust with the Harness plugin architecture.

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────┐
│                        Mac (Local)                       │
│                                                          │
│  ┌─────────────────┐         ┌──────────────────┐      │
│  │  Control Plane  │◄────────┤   MongoDB        │      │
│  │  :8080         │         │   :27017         │      │
│  └────────┬────────┘         └──────────────────┘      │
│           │                                             │
│           │ HTTP Callbacks                              │
│           ▼                                             │
│  ┌─────────────────┐                                    │
│  │    Locust       │                                    │
│  │    :8089        │                                    │
│  │  + Plugin       │                                    │
│  └────────┬────────┘                                    │
└───────────┼─────────────────────────────────────────────┘
            │
            │ HTTP Requests
            ▼
┌──────────────────────────────────────────────────────────┐
│                   Linux VM (35.239.233.230)              │
│                                                          │
│  ┌─────────────────┐                                    │
│  │  Products API   │                                    │
│  │  :8000         │                                    │
│  └─────────────────┘                                    │
└──────────────────────────────────────────────────────────┘
```

---

## Prerequisites

### Mac (Local Machine)

- **Go 1.19+** (for control plane)
- **Python 3.9+** (for Locust)
- **MongoDB** (running on localhost:27017)
- **jq** (for JSON processing in scripts)

```bash
# Install dependencies
brew install go python3 mongodb-community jq

# Start MongoDB
brew services start mongodb-community

# Install Python packages
pip3 install locust requests gevent flask
```

### Linux VM (35.239.233.230)

- Products API running on port 8000
- Port 8000 accessible from Mac

---

## Setup Steps

### Step 1: Start MongoDB

```bash
# Start MongoDB service
brew services start mongodb-community

# Verify it's running
mongosh --eval "db.adminCommand('ping')"
```

### Step 2: Build and Start Control Plane

```bash
cd /Users/sarthakjain/harness/Load-manager-cli

# Build control plane
go build -o bin/controlplane cmd/controlplane/main.go

# Start control plane
./bin/controlplane -config config/vm-test-config.yaml
```

**Expected output:**
```
2025-12-24 16:00:00 [INFO] Starting control plane...
2025-12-24 16:00:00 [INFO] Connected to MongoDB
2025-12-24 16:00:00 [INFO] HTTP server listening on :8080
```

**Keep this terminal open.**

### Step 3: Deploy Harness Plugin

The plugin is already created at `locust/locust_harness_plugin.py`. 

**Verify it exists:**
```bash
ls -la locust/locust_harness_plugin.py
```

### Step 4: Start Locust with Plugin

**Option A: Using the startup script (Recommended)**

```bash
cd /Users/sarthakjain/harness/Load-manager-cli
./scripts/start-locust-plugin.sh
```

**Option B: Manual startup**

```bash
cd /Users/sarthakjain/harness/Load-manager-cli/locust

export CONTROL_PLANE_URL="http://localhost:8080"
export CONTROL_PLANE_TOKEN="secure-token-vm-test-2025"
export METRICS_PUSH_INTERVAL="10"
export PYTHONPATH="$(pwd):$PYTHONPATH"

locust -f vm-products-api-clean.py \
  --web-host 0.0.0.0 \
  --web-port 8089 \
  --host http://35.239.233.230:8000
```

**Expected output:**
```
[2025-12-24 16:00:00,000] INFO/locust.main: Starting Locust 2.x
[2025-12-24 16:00:00,001] INFO/root: Locust Harness Plugin loaded successfully
[2025-12-24 16:00:00,002] INFO/root: Harness Control Plane plugin initialized
[2025-12-24 16:00:00,003] INFO/locust.main: Starting web interface at http://0.0.0.0:8089
```

**Keep this terminal open.**

### Step 5: Verify Setup

**Terminal 3:**

```bash
# Check control plane
curl http://localhost:8080/health
# Expected: {"status":"ok"}

# Check Locust
curl http://localhost:8089/
# Expected: HTML page

# Check VM API
curl http://35.239.233.230:8000/api/products
# Expected: JSON array of products
```

---

## Running Tests

### Quick Test (30 seconds)

```bash
cd /Users/sarthakjain/harness/Load-manager-cli
./run-vm-test-plugin.sh --duration 30 --users 5
```

### Standard Test (1 minute)

```bash
./run-vm-test-plugin.sh --duration 60 --users 10
```

### Load Test (5 minutes, 50 users)

```bash
./run-vm-test-plugin.sh --duration 300 --users 50 --spawn-rate 5
```

---

## What Happens During a Test

### 1. Test Creation
```
[run-vm-test-plugin.sh]
  ↓ Uploads clean user script (base64 encoded)
  ↓
[Control Plane]
  ↓ Auto-injects plugin import
  ↓ Stores enhanced script in MongoDB
  ↓ Returns test ID
```

### 2. Test Run Start
```
[run-vm-test-plugin.sh]
  ↓ Calls /v1/load-tests/{id}/runs
  ↓
[Control Plane]
  ↓ Calls Locust /controlplane/set-context
  ↓ Calls Locust /swarm
  ↓
[Locust]
  ↓ Spawns users
  ↓ Plugin triggers test_start event
  ↓ Sends start callback to control plane
  ↓ Starts metrics push greenlet
  ↓ Starts duration monitor greenlet
```

### 3. During Test
```
[Locust Users]
  ↓ Make HTTP requests to VM API
  ↓
[Plugin Metrics Greenlet]
  ↓ Every 10 seconds
  ↓ Collects Locust stats
  ↓ Sends to control plane /v1/internal/locust/metrics
  ↓
[Control Plane]
  ↓ Stores in metrics_timeseries collection
  ↓ Updates load_test_runs.lastMetrics
```

### 4. Test Stop (Auto)
```
[Plugin Duration Monitor]
  ↓ After duration elapses
  ↓ Sets _auto_stopped = True
  ↓ Calls runner.stop()
  ↓
[Plugin test_stop handler]
  ↓ Collects final metrics
  ↓ Sends to /v1/internal/locust/test-stop
  ↓ Includes autoStopped: true
  ↓
[Control Plane]
  ↓ Updates status to "Finished"
  ↓ Updates recent runs
  ↓ Stores in MongoDB
```

---

## Monitoring

### Locust Web UI
```
http://localhost:8089
```
- Real-time RPS, users, failures
- Statistics per endpoint
- Charts and graphs

### Control Plane Swagger
```
http://localhost:8080/swagger/index.html
```
- API documentation
- Test endpoints interactively

### MongoDB Compass
```
mongodb://localhost:27017
Database: load_testing
```

**Collections:**
- `load_tests` - Test configurations
- `load_test_runs` - Run history and status
- `metrics_timeseries` - Historical metrics
- `script_revisions` - Script version history

---

## File Structure

```
Load-manager-cli/
├── bin/
│   └── controlplane                    # Compiled binary
├── locust/
│   ├── locust_harness_plugin.py       # Plugin (blackbox)
│   ├── vm-products-api-clean.py       # Clean user script
│   └── examples/                       # More examples
├── scripts/
│   └── start-locust-plugin.sh         # Locust startup
├── config/
│   └── vm-test-config.yaml            # Config file
├── run-vm-test-plugin.sh              # Main test script
└── docs/
    ├── PLUGIN_SETUP_GUIDE.md          # This file
    ├── USER_GUIDE.md                   # User documentation
    └── AUTOMATIC_INJECTION_GUIDE.md    # Plugin details
```

---

## Clean User Script Format

Users write ONLY test scenarios:

```python
"""
My Load Test
"""

import random
from locust import HttpUser, task, between

class MyUser(HttpUser):
    wait_time = between(1, 3)
    
    @task
    def test_api(self):
        self.client.get("/api/products")
```

**That's it!** No control plane code, no configuration.

The control plane automatically:
1. Injects plugin import
2. Stores enhanced script
3. Deploys to Locust
4. Collects metrics
5. Updates status

---

## Troubleshooting

### Locust Can't Import Plugin

**Error:**
```
ModuleNotFoundError: No module named 'locust_harness_plugin'
```

**Solution:**
```bash
# Make sure PYTHONPATH includes the locust directory
export PYTHONPATH="$(pwd)/locust:$PYTHONPATH"

# Or use the startup script which sets this automatically
./scripts/start-locust-plugin.sh
```

### Control Plane Not Receiving Callbacks

**Check:**
1. Environment variables are set in Locust
2. Control plane is running on port 8080
3. Check control plane logs for callback receipt

**Verify:**
```bash
# From Locust terminal, check if plugin loaded
# Should see: "Locust Harness Plugin loaded successfully"
```

### VM API Not Reachable

**Error:**
```
Cannot reach VM at http://35.239.233.230:8000
```

**Check:**
```bash
# Test connectivity
curl -v http://35.239.233.230:8000/api/products

# Check if app is running on VM
ssh user@35.239.233.230 "netstat -tlnp | grep 8000"
```

### MongoDB Connection Failed

**Error:**
```
Failed to connect to MongoDB
```

**Solution:**
```bash
# Start MongoDB
brew services start mongodb-community

# Check status
brew services list | grep mongodb

# Test connection
mongosh --eval "db.adminCommand('ping')"
```

---

## Testing the Setup

### End-to-End Test

```bash
# Terminal 1: Start control plane
./bin/controlplane -config config/vm-test-config.yaml

# Terminal 2: Start Locust
./scripts/start-locust-plugin.sh

# Terminal 3: Run test
./run-vm-test-plugin.sh --duration 30 --users 5
```

**Expected Results:**
- ✅ Test creates successfully
- ✅ Locust spawns users
- ✅ Requests hit VM API
- ✅ Metrics flow to control plane
- ✅ Test stops after 30 seconds
- ✅ Status updates to "Finished"
- ✅ Summary displays in terminal

---

## Advanced Usage

### Custom Test Scripts

Create your own clean script:

```python
# my_test.py
from locust import HttpUser, task

class MyUser(HttpUser):
    @task
    def my_endpoint(self):
        self.client.get("/my/endpoint")
```

Update the script path in `run-vm-test-plugin.sh`:
```bash
SCRIPT_PATH="locust/my_test.py"
```

### Multiple User Types

```python
class Browser(HttpUser):
    weight = 3  # 75% of users
    @task
    def browse(self):
        self.client.get("/api/products")

class Buyer(HttpUser):
    weight = 1  # 25% of users
    @task
    def buy(self):
        self.client.post("/api/cart/items", json={"productId": 1})
```

---

## Next Steps

1. ✅ **Setup Complete** - Run your first test
2. 📖 **Read User Guide** - Learn best practices
3. 🚀 **Scale Up** - Run longer, larger tests
4. 📊 **Analyze Results** - Use MongoDB Compass
5. 🔧 **Customize** - Create your own test scenarios

---

## Quick Reference

### Start Everything
```bash
# Terminal 1
./bin/controlplane -config config/vm-test-config.yaml

# Terminal 2
./scripts/start-locust-plugin.sh

# Terminal 3
./run-vm-test-plugin.sh --duration 60 --users 10
```

### Stop Everything
```bash
# Ctrl+C in each terminal

# Or kill processes
pkill -f controlplane
pkill -f locust
```

### View Logs
```bash
# Control plane logs (Terminal 1)
# Locust logs (Terminal 2)
# Test output (Terminal 3)
```

---

## Support

- 📚 Documentation: `/docs/`
- 🎯 Examples: `/locust/examples/`
- 🐛 Issues: Check logs in each terminal

**Happy Load Testing! 🚀**
