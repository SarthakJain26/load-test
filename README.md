# Load Manager - Control Plane for Locust Load Testing

A Go-based control plane that orchestrates Locust load testing clusters with full metrics integration and lifecycle management.

## Architecture

```
┌─────────────────┐
│   User/CI/CD    │
└────────┬────────┘
         │ REST API
         ▼
┌─────────────────────────────────────────┐
│     Go Control Plane (This Service)     │
│  ┌──────────┐  ┌──────────┐  ┌────────┐│
│  │   API    │  │Orchestr- │  │ Store  ││
│  │ Handlers │◄─┤  ator    │◄─┤(Memory)││
│  └────┬─────┘  └────┬─────┘  └────────┘│
│       │             │                    │
└───────┼─────────────┼────────────────────┘
        │             │
        │             ├───► Poll Metrics
        │             │
        │             ▼
┌───────▼──────────────────────────────────┐
│        Locust Master + Workers           │
│   ┌───────────┐     ┌──────────────┐    │
│   │  Master   │────►│   Workers    │    │
│   │  (Web UI) │     │  (Load Gen)  │    │
│   └─────┬─────┘     └──────────────┘    │
│         │ Callbacks (test_start/stop)    │
│         └───────────────────────────────►│
└──────────────────────┬───────────────────┘
                       │ HTTP Load
                       ▼
              ┌─────────────────┐
              │  Target System  │
              └─────────────────┘
```

## Features

- **Multi-tenant support**: Manage load tests across multiple tenants and environments
- **Locust orchestration**: Start/stop tests via REST API without direct Locust access
- **Real-time metrics**: Automatic polling and callback-based metrics collection
- **Duration control**: Auto-stop tests after specified duration
- **Persistent tracking**: Track test runs with status, metrics, and metadata
- **Event hooks**: Locust integration via Python event listeners
- **Clean separation**: Control plane never runs on target systems

## Project Structure

```
Load-manager-cli/
├── cmd/
│   └── controlplane/
│       └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   ├── dto.go              # Request/response DTOs
│   │   └── handlers.go         # HTTP handlers
│   ├── config/
│   │   └── config.go           # Configuration management
│   ├── domain/
│   │   └── models.go           # Domain models
│   ├── locustclient/
│   │   └── client.go           # Locust HTTP client
│   ├── service/
│   │   └── orchestrator.go    # Orchestration logic
│   └── store/
│       └── memory_store.go     # In-memory persistence
├── config/
│   └── config.yaml             # Configuration file
├── locust/
│   ├── locustfile.py           # Locust tests with hooks
│   ├── requirements.txt        # Python dependencies
│   ├── Dockerfile              # Locust container
│   └── docker-compose.yml      # Local Locust cluster setup
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.24+
- Python 3.11+ (for Locust)
- Docker (optional, for containerized Locust)

### 1. Install Dependencies

Update Go dependencies:

```bash
go mod tidy
```

### 2. Configure the Control Plane

Edit `config/config.yaml`:

```yaml
security:
  locustCallbackToken: "my-secure-token"
  apiToken: "my-api-token"

locustClusters:
  - id: "local-dev"
    baseUrl: "http://localhost:8089"
    tenantId: "tenant-1"
    envId: "dev"
```

### 3. Run the Control Plane

```bash
go run cmd/controlplane/main.go -config config/config.yaml
```

The control plane will start on `http://localhost:8080`.

### 4. Start Locust Cluster

#### Option A: Using Docker Compose

```bash
cd locust
docker-compose up --build
```

This starts a Locust master at `http://localhost:8089` with 2 workers.

#### Option B: Manual Setup

Install Python dependencies:

```bash
cd locust
pip install -r requirements.txt
```

Start Locust master:

```bash
export CONTROL_PLANE_URL=http://localhost:8080
export CONTROL_PLANE_TOKEN=my-secure-token
export RUN_ID=test-run-id
export TENANT_ID=tenant-1
export ENV_ID=dev

locust -f locustfile.py --master --web-host 0.0.0.0 --web-port 8089 --host http://your-target-app.com
```

Start Locust workers (in separate terminals):

```bash
locust -f locustfile.py --worker --master-host localhost --master-port 5557
```

## API Usage

### Authentication

All API requests require a Bearer token:

```bash
Authorization: Bearer my-api-token
```

### Create and Start a Test

```bash
curl -X POST http://localhost:8080/v1/tests \
  -H "Authorization: Bearer my-api-token" \
  -H "Content-Type: application/json" \
  -d '{
    "tenantId": "tenant-1",
    "envId": "dev",
    "scenarioId": "load-test-scenario-1",
    "targetUsers": 100,
    "spawnRate": 10,
    "durationSeconds": 300,
    "metadata": {
      "description": "Performance test for API v2",
      "jiraTicket": "PERF-123"
    }
  }'
```

Response:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "tenantId": "tenant-1",
  "envId": "dev",
  "locustClusterId": "local-dev",
  "scenarioId": "load-test-scenario-1",
  "targetUsers": 100,
  "spawnRate": 10,
  "durationSeconds": 300,
  "status": "Running",
  "startedAt": "2024-12-10T12:00:00Z",
  "metadata": {
    "description": "Performance test for API v2",
    "jiraTicket": "PERF-123"
  }
}
```

### Get Test Status and Metrics

```bash
curl -X GET http://localhost:8080/v1/tests/550e8400-e29b-41d4-a716-446655440000 \
  -H "Authorization: Bearer my-api-token"
```

Response includes current metrics:

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "Running",
  "lastMetrics": {
    "timestamp": "2024-12-10T12:05:30Z",
    "totalRps": 250.5,
    "totalRequests": 75000,
    "totalFailures": 150,
    "errorRate": 0.2,
    "avgResponseMs": 45.3,
    "p50ResponseMs": 40.0,
    "p95ResponseMs": 95.5,
    "p99ResponseMs": 150.0,
    "currentUsers": 100
  }
}
```

### Stop a Test

```bash
curl -X POST http://localhost:8080/v1/tests/550e8400-e29b-41d4-a716-446655440000/stop \
  -H "Authorization: Bearer my-api-token"
```

### List Tests

```bash
# List all tests
curl -X GET http://localhost:8080/v1/tests \
  -H "Authorization: Bearer my-api-token"

# Filter by tenant and status
curl -X GET "http://localhost:8080/v1/tests?tenantId=tenant-1&status=Running" \
  -H "Authorization: Bearer my-api-token"
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Locust Integration

The `locustfile.py` includes event hooks that automatically integrate with the control plane:

### Event Hooks

1. **test_start**: Notifies control plane when test begins
2. **test_stop**: Sends final metrics when test ends
3. **Metrics pusher**: Background greenlet that sends metrics every 10 seconds

### Environment Variables for Locust

| Variable | Required | Description |
|----------|----------|-------------|
| `CONTROL_PLANE_URL` | Yes | Control plane base URL (e.g., `http://localhost:8080`) |
| `CONTROL_PLANE_TOKEN` | Yes | Shared secret for authentication |
| `RUN_ID` | Yes | Test run ID from control plane |
| `TENANT_ID` | No | Tenant identifier |
| `ENV_ID` | No | Environment identifier |
| `METRICS_PUSH_INTERVAL` | No | Metrics push interval in seconds (default: 10) |

### Custom Load Test Scenarios

Edit `locust/locustfile.py` and modify the `HttpUser` classes:

```python
class MyAppUser(HttpUser):
    wait_time = between(1, 3)
    
    @task(3)
    def browse_products(self):
        self.client.get("/api/products")
    
    @task(1)
    def add_to_cart(self):
        self.client.post("/api/cart", json={"productId": 123, "quantity": 1})
```

## Configuration Reference

### Server Configuration

```yaml
server:
  host: "0.0.0.0"    # Listen address
  port: 8080          # Listen port
```

### Locust Cluster Configuration

```yaml
locustClusters:
  - id: "unique-cluster-id"
    baseUrl: "http://locust-master:8089"  # Locust master URL
    tenantId: "tenant-1"                   # Tenant identifier
    envId: "dev"                           # Environment identifier
    authToken: ""                          # Optional Locust auth token
```

### Security Configuration

```yaml
security:
  locustCallbackToken: "secret"  # Token for Locust→Control Plane callbacks
  apiToken: "secret"             # Token for User→Control Plane API calls
```

### Orchestrator Configuration

```yaml
orchestrator:
  metricsPollIntervalSeconds: 10  # How often to poll Locust for metrics
```

## Development

### Build

```bash
go build -o bin/controlplane cmd/controlplane/main.go
```

### Run Tests

```bash
go test ./...
```

### Run with Custom Config

```bash
go run cmd/controlplane/main.go -config /path/to/config.yaml
```

## Deployment

### Docker Build (Control Plane)

```bash
docker build -t load-manager-control-plane:latest .
```

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: load-manager-control-plane
spec:
  replicas: 1
  selector:
    matchLabels:
      app: control-plane
  template:
    metadata:
      labels:
        app: control-plane
    spec:
      containers:
      - name: control-plane
        image: load-manager-control-plane:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: config
          mountPath: /config
        args:
        - -config
        - /config/config.yaml
      volumes:
      - name: config
        configMap:
          name: control-plane-config
```

## Roadmap

- [ ] PostgreSQL persistence layer
- [ ] Real authentication (OAuth2/JWT)
- [ ] Grafana dashboard integration
- [ ] Scheduled test runs
- [ ] Test result comparison
- [ ] Slack/email notifications
- [ ] Dynamic cluster scaling
- [ ] Historical metrics storage

## License

MIT License

## Contributing

Contributions welcome! Please open an issue or submit a pull request.
