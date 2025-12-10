# API Examples

This directory contains example scripts and configurations for using the Load Manager Control Plane.

## Scripts

### api-examples.sh

A complete walkthrough of the Control Plane API including:
- Health check
- Creating a test
- Getting test status
- Listing tests with filters
- Viewing metrics
- Stopping tests

**Usage:**

1. Make sure the control plane is running:
   ```bash
   make run
   ```

2. Update the configuration in the script:
   ```bash
   vim api-examples.sh
   # Update CONTROL_PLANE_URL and API_TOKEN
   ```

3. Run the examples:
   ```bash
   chmod +x api-examples.sh
   ./api-examples.sh
   ```

## Manual cURL Examples

### Create a Test

```bash
curl -X POST http://localhost:8080/v1/tests \
  -H "Authorization: Bearer your-api-token-here" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-1",
    "envId": "dev",
    "scenarioId": "api-load-test",
    "targetUsers": 100,
    "spawnRate": 10,
    "durationSeconds": 600,
    "metadata": {
      "jiraTicket": "PERF-456",
      "environment": "staging"
    }
  }'
```

### Get Test Status

```bash
curl -X GET http://localhost:8080/v1/tests/{test-id} \
  -H "Authorization: Bearer your-api-token-here"
```

### Stop a Test

```bash
curl -X POST http://localhost:8080/v1/tests/{test-id}/stop \
  -H "Authorization: Bearer your-api-token-here"
```

### List All Tests

```bash
curl -X GET http://localhost:8080/v1/tests \
  -H "Authorization: Bearer your-api-token-here"
```

### Filter Tests by Tenant

```bash
curl -X GET "http://localhost:8080/v1/tests?tenantId=tenant-1" \
  -H "Authorization: Bearer your-api-token-here"
```

### Filter Tests by Status

```bash
curl -X GET "http://localhost:8080/v1/tests?status=Running" \
  -H "Authorization: Bearer your-api-token-here"
```

### Combined Filters

```bash
curl -X GET "http://localhost:8080/v1/tests?tenantId=tenant-1&envId=dev&status=Finished" \
  -H "Authorization: Bearer your-api-token-here"
```

## Response Examples

### Test Created Response

```json
{
  "id": "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d",
  "tenantId": "tenant-1",
  "envId": "dev",
  "locustClusterId": "cluster-dev-tenant1",
  "scenarioId": "api-load-test",
  "targetUsers": 100,
  "spawnRate": 10,
  "durationSeconds": 600,
  "status": "Running",
  "startedAt": "2024-12-10T12:30:00Z",
  "metadata": {
    "jiraTicket": "PERF-456",
    "environment": "staging"
  }
}
```

### Test Status with Metrics

```json
{
  "id": "a1b2c3d4-e5f6-4a5b-8c9d-0e1f2a3b4c5d",
  "tenantId": "tenant-1",
  "envId": "dev",
  "status": "Running",
  "targetUsers": 100,
  "lastMetrics": {
    "timestamp": "2024-12-10T12:35:30Z",
    "totalRps": 450.5,
    "totalRequests": 135000,
    "totalFailures": 27,
    "errorRate": 0.02,
    "avgResponseMs": 42.3,
    "p50ResponseMs": 38.0,
    "p95ResponseMs": 85.5,
    "p99ResponseMs": 125.0,
    "currentUsers": 100,
    "requestStats": {
      "GET_/api/users": {
        "method": "GET",
        "name": "/api/users",
        "numRequests": 45000,
        "numFailures": 10,
        "avgResponseTime": 35.2,
        "minResponseTime": 12.5,
        "maxResponseTime": 450.0,
        "medianResponseTime": 33.0,
        "requestsPerSec": 150.2
      }
    }
  }
}
```

## Integration Examples

### CI/CD Pipeline (GitHub Actions)

```yaml
- name: Run Load Test
  run: |
    TEST_ID=$(curl -s -X POST $CONTROL_PLANE_URL/v1/tests \
      -H "Authorization: Bearer $API_TOKEN" \
      -H "Content-Type: application/json" \
      -d '{"tenantId":"tenant-1","envId":"staging","scenarioId":"ci-test","targetUsers":50,"spawnRate":5,"durationSeconds":120}' \
      | jq -r '.id')
    
    echo "Test ID: $TEST_ID"
    
    # Wait for test to complete
    sleep 120
    
    # Get final metrics
    curl -s -X GET $CONTROL_PLANE_URL/v1/tests/$TEST_ID \
      -H "Authorization: Bearer $API_TOKEN" \
      | jq '.lastMetrics'
```

### Python Client Example

```python
import requests
import time

CONTROL_PLANE_URL = "http://localhost:8080"
API_TOKEN = "your-api-token-here"

headers = {
    "Authorization": f"Bearer {API_TOKEN}",
    "Content-Type": "application/json"
}

# Create test
response = requests.post(
    f"{CONTROL_PLANE_URL}/v1/tests",
    headers=headers,
    json={
        "tenantId": "tenant-1",
        "envId": "dev",
        "scenarioId": "python-test",
        "targetUsers": 50,
        "spawnRate": 5,
        "durationSeconds": 300
    }
)
test_id = response.json()["id"]
print(f"Started test: {test_id}")

# Poll for status
while True:
    time.sleep(10)
    status_response = requests.get(
        f"{CONTROL_PLANE_URL}/v1/tests/{test_id}",
        headers=headers
    )
    test = status_response.json()
    
    print(f"Status: {test['status']}")
    if test.get("lastMetrics"):
        print(f"RPS: {test['lastMetrics']['totalRps']:.2f}")
        print(f"Error Rate: {test['lastMetrics']['errorRate']:.2f}%")
    
    if test["status"] in ["Finished", "Failed"]:
        break
```
