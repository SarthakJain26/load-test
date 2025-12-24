# Load Manager - Control Plane for Locust Load Testing

A Go-based control plane that orchestrates Locust load testing clusters with MongoDB persistence, script versioning, real-time metrics, and comprehensive API documentation via Swagger.

## Architecture

```
┌─────────────────┐
│   User/CI/CD    │
└────────┬────────┘
         │ REST API (Create/Stop Tests)
         ▼
┌──────────────────────────────────────────────┐
│      Go Control Plane (This Service)         │
│  ┌──────────┐  ┌──────────┐  ┌────────────┐ │
│  │   API    │  │Orchestr- │  │  MongoDB   │ │
│  │ Handlers │◄─┤  ator    │◄─┤   Store    │ │
│  │ +Swagger │  └────┬─────┘  └────────────┘ │
│  └────┬─────┘       │                        │
│       │      Start  │                        │
│       │      Test ──┤                        │
└───────┼─────────────┼────────────────────────┘
        │             │
        │             ▼
┌───────▼──────────────────────────────────────┐
│         Locust Master + Workers              │
│   ┌───────────┐     ┌──────────────┐        │
│   │  Master   │────►│   Workers    │        │
│   │  (Web UI) │     │  (Load Gen)  │        │
│   └─────┬─────┘     └──────────────┘        │
│         │                                    │
│         │ Push Metrics (every 10s)           │
│         │ Callbacks (test_start/stop)        │
│         │ Duration Monitor (auto-stop)       │
│         └───────────────────────────────────►│
└──────────────────────┬───────────────────────┘
                       │ HTTP Load
                       ▼
              ┌─────────────────┐
              │  Target System  │
              └─────────────────┘
```

## Features

- **Multi-tenant Support**: Manage load tests across multiple accounts, orgs, projects, and environments
- **Script Version Control**: Full revision tracking for test scripts with base64 storage and audit trail
- **MongoDB Persistence**: Scalable storage for load tests, runs, metrics, and script revisions
- **Locust Orchestration**: Start/stop tests via REST API without direct Locust access
- **Push-based Metrics**: Locust pushes real-time metrics to control plane (no polling overhead)
- **Visualization APIs**: Dashboard-optimized endpoints for charts, graphs, and real-time monitoring
- **Swagger/OpenAPI Documentation**: Interactive API documentation at `/swagger/index.html`
- **Duration Control**: Locust auto-stops tests after specified duration
- **Event Hooks**: Locust integration via Python event listeners
- **Clean Separation**: Control plane never runs on target systems

## Project Structure

```
Load-manager-cli/
├── cmd/
│   └── controlplane/
│       └── main.go                      # Application entry point
├── internal/
│   ├── api/
│   │   ├── dto.go                      # Request/response DTOs
│   │   ├── handlers.go                 # Core HTTP handlers
│   │   ├── loadtest_handlers.go        # LoadTest CRUD operations
│   │   ├── script_handlers.go          # Script revision management
│   │   ├── visualization_handlers.go   # Metrics & dashboard APIs
│   │   └── visualization_dto.go        # Visualization DTOs
│   ├── config/
│   │   └── config.go                   # Configuration management
│   ├── domain/
│   │   └── models.go                   # Domain models (LoadTest, Run, ScriptRevision)
│   ├── locustclient/
│   │   └── client.go                   # Locust HTTP client
│   ├── mongodb/
│   │   └── client.go                   # MongoDB connection client
│   ├── service/
│   │   └── orchestrator.go            # Orchestration logic
│   └── store/
│       ├── loadtest_store.go           # LoadTest MongoDB store
│       ├── loadtest_run_store.go       # LoadTestRun MongoDB store
│       ├── metrics_store.go            # Metrics time-series store
│       ├── script_revision_store.go    # Script revision store
│       └── memory_store.go             # In-memory store (dev/test)
├── docs/
│   ├── swagger.yaml                    # OpenAPI spec (YAML)
│   ├── swagger.json                    # OpenAPI spec (JSON)
│   └── docs.go                         # Generated Swagger docs
├── config/
│   └── config.yaml                     # Configuration file
├── locust/
│   ├── locustfile.py                   # Locust tests with hooks
│   ├── requirements.txt                # Python dependencies
│   ├── Dockerfile                      # Locust container
│   └── docker-compose.yml              # Local Locust cluster setup
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.22+
- MongoDB 4.4+
- Python 3.11+ (for Locust)
- Docker (optional, for containerized Locust and MongoDB)

### 1. Install Dependencies

Update Go dependencies:

```bash
go mod tidy
```

### 2. Start MongoDB

```bash
# Using Docker
docker run -d -p 27017:27017 --name mongodb mongo:latest

# Or use existing MongoDB instance
```

### 3. Configure the Control Plane

Edit `config/config.yaml`:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

mongodb:
  uri: "mongodb://localhost:27017"
  database: "loadmanager"
  connectTimeoutSeconds: 10
  maxPoolSize: 100

security:
  locustCallbackToken: "my-secure-token"
  apiToken: "my-api-token"

accounts:
  - accountId: "acc123"
    orgs:
      - orgId: "org456"
        projects:
          - projectId: "proj789"
            environments:
              - envId: "dev"
                locustClusterId: "local-dev"

locustClusters:
  - id: "local-dev"
    baseUrl: "http://localhost:8089"
```

### 4. Run the Control Plane

```bash
go run cmd/controlplane/main.go -config config/config.yaml
```

The control plane will start on `http://localhost:8080`.

### 5. Start Locust Cluster

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
export DURATION_SECONDS=300
export METRICS_PUSH_INTERVAL=10
export TENANT_ID=tenant-1
export ENV_ID=dev

locust -f locustfile.py --master --web-host 0.0.0.0 --web-port 8089 --host http://your-target-app.com
```

Start Locust workers (in separate terminals):

```bash
locust -f locustfile.py --worker --master-host localhost --master-port 5557
```

## API Usage

### Interactive API Documentation

Access Swagger UI for interactive API testing:

```
http://localhost:8080/swagger/index.html
```

### Authentication

All API requests require a Bearer token:

```bash
Authorization: Bearer my-api-token
```

### Create a Load Test with Script

```bash
# First, base64 encode your Python script
SCRIPT_BASE64=$(base64 < locustfile.py)

curl -X POST http://localhost:8080/v1/load-tests \
  -H "Authorization: Bearer my-api-token" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"API Performance Test\",
    \"accountId\": \"acc123\",
    \"orgId\": \"org456\",
    \"projectId\": \"proj789\",
    \"envId\": \"dev\",
    \"locustClusterId\": \"local-dev\",
    \"targetUrl\": \"https://api.example.com\",
    \"scriptContent\": \"$SCRIPT_BASE64\",
    \"defaultUsers\": 100,
    \"defaultSpawnRate\": 10,
    \"createdBy\": \"user@example.com\"
  }"
```

Response:

```json
{
  "id": "test-uuid-123",
  "name": "API Performance Test",
  "latestRevisionId": "rev-uuid-1",
  "accountId": "acc123",
  "orgId": "org456",
  "projectId": "proj789",
  "status": "Active",
  "createdAt": "2025-12-23T12:00:00Z"
}
```

### Start a Test Run

```bash
curl -X POST http://localhost:8080/v1/load-tests/test-uuid-123/runs \
  -H "Authorization: Bearer my-api-token" \
  -H "Content-Type: application/json" \
  -d '{
    "targetUsers": 200,
    "spawnRate": 20,
    "durationSeconds": 600,
    "createdBy": "user@example.com"
  }'
```

Response:

```json
{
  "id": "run-uuid-456",
  "loadTestId": "test-uuid-123",
  "scriptRevisionId": "rev-uuid-1",
  "status": "Running",
  "targetUsers": 200,
  "spawnRate": 20,
  "startedAt": "2025-12-23T12:05:00Z"
}
```

### Get Run Details with Metrics

```bash
curl -X GET http://localhost:8080/v1/runs/run-uuid-456 \
  -H "Authorization: Bearer my-api-token"
```

### Get Real-time Dashboard Data

```bash
# Summary metrics for dashboard cards
curl http://localhost:8080/v1/runs/run-uuid-456/summary

# Graph data for charts
curl http://localhost:8080/v1/runs/run-uuid-456/graph

# Request statistics
curl http://localhost:8080/v1/runs/run-uuid-456/requests
```

### Update Script (Creates New Revision)

```bash
NEW_SCRIPT=$(base64 < locustfile_v2.py)

curl -X PUT http://localhost:8080/v1/load-tests/test-uuid-123/script \
  -H "Authorization: Bearer my-api-token" \
  -H "Content-Type: application/json" \
  -d "{
    \"scriptContent\": \"$NEW_SCRIPT\",
    \"description\": \"Added new endpoints\",
    \"updatedBy\": \"user@example.com\"
  }"
```

### List Script Revision History

```bash
curl http://localhost:8080/v1/load-tests/test-uuid-123/script/revisions
```

### Stop a Running Test

```bash
curl -X POST http://localhost:8080/v1/runs/run-uuid-456/stop \
  -H "Authorization: Bearer my-api-token"
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Locust Integration

The `locustfile.py` includes event hooks that automatically integrate with the control plane:

### Event Hooks

1. **test_start**: Notifies control plane when test begins, starts background tasks
2. **test_stop**: Sends final metrics when test ends
3. **Metrics pusher**: Background greenlet that pushes metrics every 10 seconds to control plane
4. **Duration monitor**: Background greenlet that auto-stops test after configured duration

### Environment Variables for Locust

| Variable | Required | Description |
|----------|----------|-------------|
| `CONTROL_PLANE_URL` | Yes | Control plane base URL (e.g., `http://localhost:8080`) |
| `CONTROL_PLANE_TOKEN` | Yes | Shared secret for authentication |
| `RUN_ID` | Yes | Test run ID from control plane |
| `DURATION_SECONDS` | No | Test duration - auto-stops after N seconds |
| `METRICS_PUSH_INTERVAL` | No | Metrics push interval in seconds (default: 10) |
| `TENANT_ID` | No | Tenant identifier |
| `ENV_ID` | No | Environment identifier |

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

See `config/config.yaml` for full configuration options.

### Key Configuration Sections

**MongoDB:**
```yaml
mongodb:
  uri: "mongodb://localhost:27017"
  database: "loadmanager"
  connectTimeoutSeconds: 10
  maxPoolSize: 100
```

**Multi-tenant Hierarchy:**
```yaml
accounts:
  - accountId: "acc123"
    orgs:
      - orgId: "org456"
        projects:
          - projectId: "proj789"
            environments:
              - envId: "dev"
                locustClusterId: "cluster-1"
              - envId: "prod"
                locustClusterId: "cluster-2"
```

**Locust Clusters:**
```yaml
locustClusters:
  - id: "cluster-1"
    baseUrl: "http://locust-dev:8089"
  - id: "cluster-2"
    baseUrl: "http://locust-prod:8089"
```

**Security:**
```yaml
security:
  locustCallbackToken: "secret"  # Locust→Control Plane
  apiToken: "secret"             # User→Control Plane
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

## Documentation

- **README.md** (this file) - Getting started guide
- **SWAGGER_INTEGRATION.md** - API documentation guide
- **SCRIPT_REVISION_GUIDE.md** - Script version control guide
- **VISUALIZATION_API_GUIDE.md** - Dashboard API reference
- **docs/MONGODB_SETUP.md** - MongoDB setup and indexes
- **Swagger UI** - http://localhost:8080/swagger/index.html

## Roadmap

- [x] MongoDB persistence layer
- [x] Script revision control
- [x] Swagger/OpenAPI documentation
- [x] Real-time visualization APIs
- [ ] Real authentication (OAuth2/JWT)
- [ ] Grafana dashboard integration
- [ ] Scheduled test runs
- [ ] Test result comparison and trending
- [ ] Slack/email notifications
- [ ] Dynamic cluster scaling

## License

MIT License

## Contributing

Contributions welcome! Please open an issue or submit a pull request.
