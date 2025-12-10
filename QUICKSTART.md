# Quick Start Guide

Get the Load Manager Control Plane up and running in 5 minutes.

## Step 1: Install Dependencies

```bash
# Install Go dependencies
GOTOOLCHAIN=local go mod tidy

# Install Python dependencies for Locust (optional, if running locally)
cd locust && pip install -r requirements.txt && cd ..
```

## Step 2: Configure

Edit `config/config.yaml` and set your tokens:

```yaml
security:
  locustCallbackToken: "change-this-secret-token"
  apiToken: "change-this-api-token"

locustClusters:
  - id: "local-dev"
    baseUrl: "http://localhost:8089"
    tenantId: "tenant-1"
    envId: "dev"
```

## Step 3: Start Locust Cluster (Using Docker)

```bash
# Update tokens in locust/docker-compose.yml to match config.yaml
vim locust/docker-compose.yml

# Start Locust cluster
cd locust
docker-compose up -d
cd ..
```

**Locust Web UI:** http://localhost:8089

## Step 4: Start Control Plane

```bash
# Build and run
make run

# Or run directly with go run
GOTOOLCHAIN=local go run cmd/controlplane/main.go -config config/config.yaml
```

**Control Plane API:** http://localhost:8080

## Step 5: Test the API

```bash
# Health check (no auth required)
curl http://localhost:8080/health

# Create a load test (requires API token)
curl -X POST http://localhost:8080/v1/tests \
  -H "Authorization: Bearer change-this-api-token" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-1",
    "envId": "dev",
    "scenarioId": "test-scenario",
    "targetUsers": 10,
    "spawnRate": 2,
    "durationSeconds": 60
  }'

# Get test status (replace {test-id} with the ID from the response above)
curl http://localhost:8080/v1/tests/{test-id} \
  -H "Authorization: Bearer change-this-api-token"
```

## Step 6: View Metrics

Watch the test progress:

```bash
# Poll test status every 5 seconds
watch -n 5 'curl -s http://localhost:8080/v1/tests/{test-id} \
  -H "Authorization: Bearer change-this-api-token" | jq ".lastMetrics"'
```

Or use the Locust Web UI at http://localhost:8089

## Common Commands

```bash
# Build the control plane
make build

# Run tests
make test

# Start control plane
make run

# Start Locust cluster
make locust-up

# Stop Locust cluster
make locust-down

# View Locust logs
make locust-logs

# Clean build artifacts
make clean
```

## Architecture at a Glance

```
User → Control Plane API → Locust Master → Locust Workers → Target System
       ↑                    ↓
       └────── Metrics ─────┘
```

## Customizing Load Tests

Edit `locust/locustfile.py` to add your test scenarios:

```python
from locust import HttpUser, task, between

class MyUser(HttpUser):
    wait_time = between(1, 3)
    
    @task
    def my_endpoint(self):
        self.client.get("/my-endpoint")
```

Restart the Locust cluster after making changes:

```bash
make locust-down
make locust-up
```

## Troubleshooting

### Control Plane won't start

- Check `config/config.yaml` is valid YAML
- Ensure port 8080 is not already in use
- Check logs for specific errors

### Locust callbacks not working

- Verify `CONTROL_PLANE_TOKEN` in docker-compose.yml matches `security.locustCallbackToken` in config.yaml
- Check Locust can reach the control plane (use `host.docker.internal` on Docker Desktop)
- View Locust logs: `make locust-logs`

### Tests stuck in "Pending" status

- Verify Locust cluster is running: http://localhost:8089
- Check control plane can reach Locust at the configured `baseUrl`
- Ensure the tenant/env mapping exists in config.yaml

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Check [examples/](examples/) for API usage examples
- Customize `locustfile.py` for your target application
- Add more Locust clusters for different environments
- Integrate with CI/CD pipelines

## Need Help?

- Check the logs for both control plane and Locust
- Ensure all tokens match between config.yaml and docker-compose.yml
- Verify network connectivity between components
- Review the API examples in `examples/api-examples.sh`
