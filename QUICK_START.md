# Quick Start Guide - Testing Products API on load-testing-vm-1

## Your Custom Setup

- **VM Internal IP**: 10.128.0.81
- **VM External IP**: 35.239.233.230
- **Application Port**: 8000
- **Endpoints**: `/api/products`, `/api/products/:id`, `/api/search`, `/api/cart/items`

---

## Prerequisites Check

```bash
# 1. Verify MongoDB is running
brew services list | grep mongodb
# If not running: brew services start mongodb-community@7.0

# 2. Test connection to your VM
curl http://10.128.0.81:8000/api/products
# Should return product data or 200 OK

# 3. Verify Go is installed
go version
# Should show go1.22 or higher

# 4. Verify Python and Locust
python3 --version
locust --version
```

---

## Step 1: Build Control Plane (One-time)

```bash
cd /Users/sarthakjain/harness/Load-manager-cli

# Build the binary
go build -o bin/controlplane cmd/controlplane/main.go

# Verify
ls -lh bin/controlplane
```

---

## Step 2: Start Control Plane

**Terminal 1:**

```bash
cd /Users/sarthakjain/harness/Load-manager-cli

# Start control plane with your custom config
./bin/controlplane -config config/vm-test-config.yaml
```

**Expected Output:**
```
Starting Load Manager Control Plane...
Connected to MongoDB: load_testing
Orchestrator started (push-based metrics mode)
Server listening on 0.0.0.0:8080
Swagger UI available at: http://localhost:8080/swagger/index.html
```

**Verify (in another terminal):**
```bash
curl http://localhost:8080/health
# Should return: {"status":"healthy"}
```

---

## Step 3: Start Locust

**Terminal 2:**

```bash
cd /Users/sarthakjain/harness/Load-manager-cli/locust

# Set environment variables
export CONTROL_PLANE_URL="http://localhost:8080"
export CONTROL_PLANE_TOKEN="secure-token-vm-test-2025"
export RUN_ID="manual-test-run-1"
export DURATION_SECONDS="120"
export METRICS_PUSH_INTERVAL="10"
export TENANT_ID="my-account"
export ENV_ID="vm-test"

# Target your VM (use internal IP for faster access)
export TARGET_HOST="http://10.128.0.81:8000"

# Start Locust with your custom script
locust -f vm-products-api.py \
  --master \
  --web-host 0.0.0.0 \
  --web-port 8089 \
  --host $TARGET_HOST
```

**Expected Output:**
```
[2025-12-23 16:15:00,000] INFO/locust.main: Starting web interface at http://0.0.0.0:8089
[2025-12-23 16:15:00,000] INFO/locust.main: Starting Locust 2.x.x
```

**Verify:**
Open http://localhost:8089 in your browser - you should see the Locust UI.

---

## Step 4: Connect MongoDB Compass

1. Open **MongoDB Compass**
2. Click **"New Connection"**
3. Enter: `mongodb://localhost:27017`
4. Click **"Connect"**
5. You should see the `load_testing` database

---

## Step 5: Run Your First Test (EASY MODE)

**Terminal 3:**

```bash
cd /Users/sarthakjain/harness/Load-manager-cli

# Run automated test (60 seconds, 10 users)
./run-vm-test.sh
```

**This script will automatically:**
1. ✅ Test connectivity to your VM
2. ✅ Create a load test
3. ✅ Start the test run
4. ✅ Wait for completion
5. ✅ Display results

**Custom options:**
```bash
# 5-minute test with 50 users
./run-vm-test.sh --duration 300 --users 50 --spawn-rate 5

# Use external IP instead of internal
./run-vm-test.sh --external

# See all options
./run-vm-test.sh --help
```

---

## Step 6: View Results

### Option 1: MongoDB Compass
1. Navigate to `load_testing` database
2. Open `load_test_runs` collection
3. Find your test run
4. Open `metrics` collection to see time-series data

### Option 2: Swagger UI
1. Open: http://localhost:8080/swagger/index.html
2. Click **"Authorize"** and enter: `api-token-products-test-2025`
3. Try the visualization endpoints

### Option 3: Command Line
```bash
# Set your API token
export API_TOKEN="api-token-products-test-2025"

# Get run summary (replace RUN_ID)
curl http://localhost:8080/v1/visualization/runs/RUN_ID/summary \
  -H "Authorization: Bearer $API_TOKEN" | jq '.summary'

# Get time-series data for charts
curl http://localhost:8080/v1/visualization/runs/RUN_ID/timeseries \
  -H "Authorization: Bearer $API_TOKEN" | jq '.timeseries | length'
```

---

## Manual Test (Advanced)

If you prefer manual control:

**Terminal 3:**

```bash
export API_TOKEN="api-token-products-test-2025"
export TARGET_HOST="http://10.128.0.81:8000"

# 1. Create load test
curl -X POST http://localhost:8080/v1/load-tests \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Products API Load Test\",
    \"description\": \"Testing Products API on VM\",
    \"accountId\": \"my-account\",
    \"orgId\": \"my-org\",
    \"projectId\": \"products-api\",
    \"envId\": \"vm-test\",
    \"targetUrl\": \"$TARGET_HOST\",
    \"targetUsers\": 20,
    \"spawnRate\": 2,
    \"durationSeconds\": 120,
    \"scriptContent\": \"IyBQcm9kdWN0cyBBUEkgdGVzdAo=\"
  }" | jq '.'

# Save the "id" from response
export LOAD_TEST_ID="lt_xxx"

# 2. Start test run
curl -X POST http://localhost:8080/v1/load-tests/$LOAD_TEST_ID/runs \
  -H "Authorization: Bearer $API_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetUsers": 20,
    "spawnRate": 2,
    "durationSeconds": 120
  }' | jq '.'

# Save the "id" from response
export RUN_ID="run_xxx"

# 3. Watch progress
watch -n 5 "curl -s http://localhost:8080/v1/runs/$RUN_ID \
  -H 'Authorization: Bearer $API_TOKEN' | jq '.status, .lastMetrics.totalRps'"

# 4. Get final results (after test completes)
curl http://localhost:8080/v1/visualization/runs/$RUN_ID/summary \
  -H "Authorization: Bearer $API_TOKEN" | jq '.summary'
```

---

## What the Test Does

Your custom `vm-products-api.py` script simulates **two types of users**:

### ProductsBrowserUser (75% of traffic)
- **Weight**: 3 (most users)
- **Behavior**: Typical browsing patterns
- **Tasks**:
  - `GET /api/products` (50% of actions) - List all products
  - `GET /api/products/:id` (25%) - View product details
  - `GET /api/search` (15%) - Search products
  - `POST /api/cart/items` (10%) - Add to cart

### PowerUser (25% of traffic)
- **Weight**: 1 (fewer users)
- **Behavior**: Rapid browsing, multiple actions
- **Tasks**:
  - Rapid searches with different terms
  - Browse multiple products quickly
  - Add multiple items to cart

---

## Test Scenarios

### Scenario 1: Light Load (Baseline)
```bash
./run-vm-test.sh --duration 60 --users 10 --spawn-rate 2
```

### Scenario 2: Moderate Load
```bash
./run-vm-test.sh --duration 180 --users 50 --spawn-rate 5
```

### Scenario 3: Stress Test
```bash
./run-vm-test.sh --duration 300 --users 100 --spawn-rate 10
```

### Scenario 4: Spike Test
```bash
# Start with 10 users, manually increase to 100 in Locust UI
./run-vm-test.sh --duration 300 --users 10 --spawn-rate 2
# Then use Locust UI to spike to 100 users
```

---

## Monitoring During Test

### Locust Web UI (http://localhost:8089)
- Real-time RPS (requests per second)
- Response times (min, max, average, percentiles)
- Failure rate
- Number of users
- Charts and graphs

### Control Plane Logs (Terminal 1)
```
Updated metrics for run run_xxx: RPS=45.20, Requests=5424, Failures=2, Users=50
```

### Locust Logs (Terminal 2)
```
Pushed metrics to control plane (RPS: 45.20)
Duration monitor started: will stop test after 120 seconds
```

---

## Customizing the Test

### Modify User Behavior

Edit `locust/vm-products-api.py`:

```python
@task(10)  # Change weight to adjust frequency
def your_new_test(self):
    """Test a new endpoint"""
    self.client.get("/api/your-endpoint", name="GET /api/your-endpoint")
```

### Add Authentication

If your API requires auth:

```python
def on_start(self):
    """Called when user starts"""
    # Add authentication
    self.client.headers.update({
        "Authorization": "Bearer your-api-token"
    })
```

### Test Different Scenarios

```python
# Add a new user class
class CheckoutUser(HttpUser):
    wait_time = between(3, 5)
    weight = 1
    
    @task
    def complete_checkout(self):
        # Add items to cart
        # Proceed to checkout
        # Complete purchase
        pass
```

---

## Troubleshooting

### Issue: "Cannot reach VM"

```bash
# Test internal IP
curl -v http://10.128.0.81:8000/api/products

# If that fails, try external IP
curl -v http://35.239.233.230:8000/api/products

# If external works, use it
./run-vm-test.sh --external
```

### Issue: "Control plane not running"

```bash
# Check if running
ps aux | grep controlplane

# Check port
lsof -i :8080

# Restart if needed
./bin/controlplane -config config/vm-test-config.yaml
```

### Issue: "MongoDB connection error"

```bash
# Check MongoDB status
brew services list | grep mongodb

# Start if not running
brew services start mongodb-community@7.0

# Test connection
mongosh mongodb://localhost:27017
```

### Issue: "Locust not pushing metrics"

Check environment variables match:
- `CONTROL_PLANE_TOKEN` in Locust = `locustCallbackToken` in config.yaml
- Both should be: `secure-token-vm-test-2025`

---

## Next Steps

1. **Run baseline test** (10 users, 60s) to establish normal performance
2. **Gradually increase load** to find breaking points
3. **Monitor VM resources** (CPU, memory, network)
4. **Analyze results** in MongoDB Compass
5. **Optimize application** based on bottlenecks
6. **Automate tests** in CI/CD pipeline

---

## Key Files Created

✅ `config/vm-test-config.yaml` - Control plane configuration
✅ `locust/vm-products-api.py` - Custom test script for your API
✅ `run-vm-test.sh` - Automated test runner

---

## Quick Reference

| Component | URL/Location |
|-----------|-------------|
| Control Plane API | http://localhost:8080 |
| Swagger UI | http://localhost:8080/swagger/index.html |
| Locust Web UI | http://localhost:8089 |
| MongoDB Compass | mongodb://localhost:27017 |
| Your VM (Internal) | http://10.128.0.81:8000 |
| Your VM (External) | http://35.239.233.230:8000 |
| API Token | `api-token-products-test-2025` |

---

**Ready to start? Run the automated test:**

```bash
./run-vm-test.sh
```

**Need help?** Check the [TESTING_SETUP_GUIDE.md](TESTING_SETUP_GUIDE.md) for detailed explanations.
