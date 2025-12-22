# LoadTest/LoadTestRun Implementation Guide

## Completed Work ✅

### 1. Domain Models
- **File**: `internal/domain/models.go`
- Replaced `TestRun` with `LoadTest` and `LoadTestRun`
- Added audit fields (createdAt, createdBy, updatedAt, updatedBy)
- Added missing response time fields to metrics

### 2. Store Layer
- **Files**: 
  - `internal/store/memory_store.go` - In-memory implementations
  - `internal/store/mongo_testrun_store.go` - MongoDB implementations
  - `internal/store/mongo_metrics_store.go` - Updated for loadTestRunId
- Two separate repositories: `LoadTestRepository` and `LoadTestRunRepository`
- Collections: `load_tests` and `load_test_runs`
- Proper indexes for query optimization

### 3. API Layer
- **Files**:
  - `internal/api/dto.go` - Request/response DTOs
  - `internal/api/loadtest_handlers.go` - NEW handlers for CRUD operations
  - `internal/api/handlers.go` - Updated Handler struct
- Handler methods for both LoadTest and LoadTestRun entities

## Remaining Work 🔧

### Step 1: Update Service Layer (Orchestrator)

**File**: `internal/service/orchestrator.go`

The orchestrator currently references `TestRun`. You need to:

1. **Add LoadTest service methods**:
```go
// Add these to the Orchestrator struct
loadTestStore    store.LoadTestRepository
loadTestRunStore store.LoadTestRunRepository
```

2. **Update constructor** to accept both stores

3. **Add LoadTest methods**:
```go
func (o *Orchestrator) CreateLoadTest(req *CreateLoadTestRequest) (*domain.LoadTest, error)
func (o *Orchestrator) GetLoadTest(id string) (*domain.LoadTest, error)
func (o *Orchestrator) UpdateLoadTest(id string, req *UpdateLoadTestRequest) (*domain.LoadTest, error)
func (o *Orchestrator) DeleteLoadTest(id string) error
func (o *Orchestrator) ListLoadTests(filter *store.LoadTestFilter) ([]*domain.LoadTest, error)
```

4. **Update LoadTestRun methods** to:
   - Accept `LoadTest` reference
   - Interact with Locust cluster
   - Start/stop tests
   - Update metrics

5. **Update all `TestRun` references** to `LoadTestRun` throughout the file

### Step 2: Update Main Application

**File**: `cmd/controlplane/main.go`

1. **Initialize stores**:
```go
// MongoDB stores
loadTestStore, err := store.NewMongoLoadTestStore(mongoDB)
loadTestRunStore, err := store.NewMongoLoadTestRunStore(mongoDB)
metricsStore, err := store.NewMongoMetricsStore(mongoDB)

// Or in-memory stores for testing
loadTestStore := store.NewInMemoryLoadTestStore()
loadTestRunStore := store.NewInMemoryLoadTestRunStore()
```

2. **Pass stores to orchestrator**:
```go
orchestrator := service.NewOrchestrator(
    loadTestStore,
    loadTestRunStore,
    metricsStore,
    locustClient,
    config,
)
```

3. **Update API handler initialization**:
```go
handler := api.NewHandler(
    orchestrator,
    loadTestStore,
    loadTestRunStore,
    config,
)
```

4. **Register new routes**:
```go
// LoadTest routes
router.HandleFunc("/v1/load-tests", handler.CreateLoadTest).Methods("POST")
router.HandleFunc("/v1/load-tests", handler.ListLoadTests).Methods("GET")
router.HandleFunc("/v1/load-tests/{id}", handler.GetLoadTest).Methods("GET")
router.HandleFunc("/v1/load-tests/{id}", handler.UpdateLoadTest).Methods("PUT")
router.HandleFunc("/v1/load-tests/{id}", handler.DeleteLoadTest).Methods("DELETE")

// LoadTestRun routes
router.HandleFunc("/v1/load-tests/{id}/runs", handler.CreateLoadTestRun).Methods("POST")
router.HandleFunc("/v1/load-tests/{id}/runs", handler.ListLoadTestRuns).Methods("GET")
router.HandleFunc("/v1/runs", handler.ListLoadTestRuns).Methods("GET")
router.HandleFunc("/v1/runs/{id}", handler.GetLoadTestRun).Methods("GET")
router.HandleFunc("/v1/runs/{id}/stop", handler.StopLoadTestRun).Methods("POST")
```

### Step 3: Update Visualization Handlers

**File**: `internal/api/visualization_handlers.go`

Search and replace all occurrences:
- `TestRun` → `LoadTestRun`
- `testRunId` → `loadTestRunId`
- `TestRunStatus` → `LoadTestRunStatus`

Update filter logic to support `loadTestId` for querying runs of a specific test.

### Step 4: Update Old Handlers (Optional Backward Compatibility)

**File**: `internal/api/handlers.go`

**Option A: Remove old handlers** (recommended since you said no backward compatibility):
- Remove `CreateTest`, `StopTest`, `GetTest`, `ListTests`
- Remove `CreateTestRequest`, `TestRunResponse`

**Option B: Keep as aliases** (redirect to new endpoints):
- Update them to use the new LoadTest/LoadTestRun structure internally

### Step 5: Update Locust Client (if exists)

**File**: `internal/locustclient/client.go`

Update any references from `testRunId` to `loadTestRunId` in:
- Start test calls
- Stop test calls
- Metrics callbacks

### Step 6: Update Visualization DTOs

**File**: `internal/api/visualization_dto.go`

Update any DTOs that reference test runs to use `loadTestRunId`.

## Testing Checklist

After implementation, test these scenarios:

### LoadTest Operations
- [ ] Create a LoadTest with all fields
- [ ] Get LoadTest by ID
- [ ] List LoadTests with filters (tenantId, envId, tags)
- [ ] Update LoadTest
- [ ] Delete LoadTest

### LoadTestRun Operations
- [ ] Start a run from LoadTest (using defaults)
- [ ] Start a run with parameter overrides
- [ ] List runs for a specific LoadTest
- [ ] List all runs with filters
- [ ] Get run details
- [ ] Stop a running test
- [ ] Verify status transitions (Pending → Running → Finished/Failed)

### Metrics
- [ ] Verify metrics are stored with loadTestRunId
- [ ] Query metrics for a specific run
- [ ] Verify time-series data retrieval
- [ ] Check aggregated metrics

### Audit Trail
- [ ] Verify createdBy and updatedBy are populated
- [ ] Verify timestamps are correct
- [ ] Check audit trail persists across updates

## Database Migration

Since you're not maintaining backward compatibility:

1. **Drop old collection**:
```javascript
db.test_runs.drop()
```

2. **Verify new collections exist**:
```javascript
db.load_tests.find()
db.load_test_runs.find()
```

3. **Verify indexes**:
```javascript
db.load_tests.getIndexes()
db.load_test_runs.getIndexes()
db.metrics_timeseries.getIndexes()
```

## API Endpoint Changes

### Before (Old)
```
POST   /v1/tests           - Create and start test
GET    /v1/tests           - List tests
GET    /v1/tests/{id}      - Get test
POST   /v1/tests/{id}/stop - Stop test
```

### After (New)
```
# LoadTest management
POST   /v1/load-tests              - Create test definition
GET    /v1/load-tests              - List test definitions
GET    /v1/load-tests/{id}         - Get test definition
PUT    /v1/load-tests/{id}         - Update test definition
DELETE /v1/load-tests/{id}         - Delete test definition

# LoadTestRun execution
POST   /v1/load-tests/{id}/runs    - Start a new run
GET    /v1/load-tests/{id}/runs    - List runs for a test
GET    /v1/runs                     - List all runs
GET    /v1/runs/{id}                - Get run details
POST   /v1/runs/{id}/stop           - Stop a run
```

## Quick Start Commands

### 1. Build the application
```bash
go build -o load-manager ./cmd/controlplane
```

### 2. Run with MongoDB
```bash
# Start MongoDB
docker-compose -f docker-compose.mongodb.yml up -d

# Run application
./load-manager
```

### 3. Create a LoadTest
```bash
curl -X POST http://localhost:8080/v1/load-tests \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API Load Test",
    "description": "Tests API performance",
    "tags": ["api", "prod"],
    "tenantId": "tenant-1",
    "envId": "prod",
    "locustClusterId": "cluster-1",
    "targetUrl": "https://api.example.com",
    "locustfile": "loadtest.py",
    "defaultUsers": 100,
    "defaultSpawnRate": 10,
    "createdBy": "user@example.com"
  }'
```

### 4. Start a Run
```bash
curl -X POST http://localhost:8080/v1/load-tests/{loadTestId}/runs \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetUsers": 200,
    "spawnRate": 20,
    "createdBy": "user@example.com"
  }'
```