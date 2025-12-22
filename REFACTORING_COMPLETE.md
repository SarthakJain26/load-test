# LoadTest/LoadTestRun Refactoring - COMPLETED ✅

## Summary

Successfully restructured the Load Manager CLI to separate **LoadTest** (test definition) from **LoadTestRun** (execution), with full audit trail support (createdAt, createdBy, updatedAt, updatedBy).

## What Was Changed

### 1. Domain Models ✅
**File**: `internal/domain/models.go`
- Replaced `TestRun` with `LoadTest` and `LoadTestRun`
- Added audit fields to both entities
- Fixed missing response time fields in `MetricSnapshot` and `ReqStat`

### 2. Store Layer ✅
**Files**: 
- `internal/store/memory_store.go` - In-memory stores
- `internal/store/mongo_testrun_store.go` - MongoDB stores
- `internal/store/mongo_metrics_store.go` - Updated for loadTestRunId

**Changes**:
- Two new repositories: `LoadTestRepository` and `LoadTestRunRepository`
- MongoDB collections: `load_tests` and `load_test_runs`
- Proper indexes for efficient queries
- Updated metrics collection to use `loadTestRunId`

### 3. Service Layer ✅
**File**: `internal/service/orchestrator.go`
- Updated to use both LoadTest and LoadTestRun stores
- All methods now use `LoadTestRun` instead of `TestRun`
- Updated status references to `LoadTestRunStatus`
- Added audit field population

### 4. API Layer ✅
**Files**:
- `internal/api/dto.go` - New request/response DTOs
- `internal/api/loadtest_handlers.go` - NEW file with CRUD handlers
- `internal/api/handlers.go` - Updated, removed old handlers
- `internal/api/visualization_handlers.go` - Updated to use LoadTestRun

**New Endpoints**:
- `POST /v1/load-tests` - Create LoadTest
- `GET /v1/load-tests` - List LoadTests
- `GET /v1/load-tests/{id}` - Get LoadTest
- `PUT /v1/load-tests/{id}` - Update LoadTest
- `DELETE /v1/load-tests/{id}` - Delete LoadTest
- `POST /v1/load-tests/{id}/runs` - Start a run
- `GET /v1/load-tests/{id}/runs` - List runs for a test
- `GET /v1/runs` - List all runs
- `GET /v1/runs/{id}` - Get run details
- `POST /v1/runs/{id}/stop` - Stop a run
- `GET /v1/runs/{id}/metrics/timeseries` - Get metrics
- `GET /v1/runs/{id}/metrics/scatter` - Get scatter plot
- `GET /v1/runs/{id}/metrics/aggregate` - Get aggregated stats

### 5. Application Wiring ✅
**File**: `cmd/controlplane/main.go`
- Initialize both MongoDB stores
- Wire up orchestrator with both stores
- Updated all route registrations

### 6. Build Verification ✅
- Compiled successfully with `go build`
- All imports resolved
- No compilation errors

## Database Collections

### `load_tests` Collection
```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "tags": ["string"],
  "tenantId": "string",
  "envId": "string",
  "locustClusterId": "string",
  "targetUrl": "string",
  "locustfile": "string",
  "scenarioId": "string",
  "defaultUsers": 100,
  "defaultSpawnRate": 10,
  "defaultDurationSec": 300,
  "maxDurationSec": 600,
  "createdAt": "timestamp",
  "createdBy": "user@example.com",
  "updatedAt": "timestamp",
  "updatedBy": "user@example.com",
  "metadata": {}
}
```

### `load_test_runs` Collection
```json
{
  "id": "string",
  "loadTestId": "string",
  "name": "string",
  "tenantId": "string",
  "envId": "string",
  "targetUsers": 200,
  "spawnRate": 20,
  "durationSeconds": 300,
  "status": "Running",
  "startedAt": "timestamp",
  "finishedAt": "timestamp",
  "lastMetrics": {},
  "createdAt": "timestamp",
  "createdBy": "user@example.com",
  "updatedAt": "timestamp",
  "updatedBy": "user@example.com",
  "metadata": {}
}
```

### `metrics_timeseries` Collection (Updated)
- Field renamed: `testRunId` → `loadTestRunId`
- Indexes updated for new field name

## Next Steps

### 1. Test the Application

```bash
# Start MongoDB
docker-compose -f docker-compose.mongodb.yml up -d

# Build and run
make build
./load-manager
```

### 2. Create Your First LoadTest

```bash
curl -X POST http://localhost:8080/v1/load-tests \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API Performance Test",
    "description": "Tests API endpoints under load",
    "tags": ["api", "performance"],
    "tenantId": "tenant-1",
    "envId": "prod",
    "locustClusterId": "cluster-1",
    "targetUrl": "https://api.example.com",
    "locustfile": "loadtest.py",
    "defaultUsers": 100,
    "defaultSpawnRate": 10,
    "defaultDurationSec": 300,
    "createdBy": "user@example.com"
  }'
```

### 3. Run the LoadTest

```bash
# Get the loadTestId from the previous response
curl -X POST http://localhost:8080/v1/load-tests/{loadTestId}/runs \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetUsers": 200,
    "spawnRate": 20,
    "createdBy": "user@example.com"
  }'
```

### 4. Monitor the Run

```bash
# Get run details
curl -X GET http://localhost:8080/v1/runs/{runId} \
  -H "Authorization: Bearer YOUR_TOKEN"

# Get metrics
curl -X GET http://localhost:8080/v1/runs/{runId}/metrics/timeseries \
  -H "Authorization: Bearer YOUR_TOKEN"

# Stop the run
curl -X POST http://localhost:8080/v1/runs/{runId}/stop \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 5. List Tests and Runs

```bash
# List all load tests
curl -X GET http://localhost:8080/v1/load-tests \
  -H "Authorization: Bearer YOUR_TOKEN"

# List runs for a specific test
curl -X GET http://localhost:8080/v1/load-tests/{loadTestId}/runs \
  -H "Authorization: Bearer YOUR_TOKEN"

# List all runs
curl -X GET http://localhost:8080/v1/runs \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Migration Notes

### Database Migration
Since backward compatibility was not maintained:

```javascript
// MongoDB shell commands
use your_database;

// Drop old collection
db.test_runs.drop();

// Verify new collections
db.load_tests.find().pretty();
db.load_test_runs.find().pretty();

// Check indexes
db.load_tests.getIndexes();
db.load_test_runs.getIndexes();
db.metrics_timeseries.getIndexes();
```

### Configuration
No configuration changes needed - existing config structure is compatible.

## Files Modified

### Core Changes
- ✅ `internal/domain/models.go`
- ✅ `internal/store/memory_store.go`
- ✅ `internal/store/mongo_testrun_store.go`
- ✅ `internal/store/mongo_metrics_store.go`
- ✅ `internal/service/orchestrator.go`
- ✅ `internal/api/dto.go`
- ✅ `internal/api/handlers.go`
- ✅ `internal/api/loadtest_handlers.go` (NEW)
- ✅ `internal/api/visualization_handlers.go`
- ✅ `cmd/controlplane/main.go`

### Documentation
- ✅ `LOADTEST_REFACTOR_SUMMARY.md`
- ✅ `IMPLEMENTATION_GUIDE.md`
- ✅ `REFACTORING_COMPLETE.md` (this file)

## Key Benefits

1. **Reusability**: Define a test once, run multiple times
2. **History**: Track all runs of a specific test
3. **Governance**: Full audit trail with user attribution
4. **Flexibility**: Override defaults at runtime
5. **Organization**: Tag-based test categorization
6. **Separation of Concerns**: Clear separation between definition and execution

## Verification

Build status: ✅ **SUCCESS**

```bash
$ go build -o /tmp/load-manager-test ./cmd/controlplane
# Exit code: 0 (Success)
```

All components successfully integrated and compiled.

## Support

For questions or issues:
1. Review `LOADTEST_REFACTOR_SUMMARY.md` for architecture overview
2. Check `IMPLEMENTATION_GUIDE.md` for detailed explanations
3. Test with the example curl commands above

---

**Status**: ✅ COMPLETE - Ready for testing and deployment
**Date**: December 21, 2025
**Build**: Verified and passing
