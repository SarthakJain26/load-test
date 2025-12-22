# LoadTest and LoadTestRun Refactoring Summary

## Overview
Restructured the system to separate **LoadTest** (test definition/template) from **LoadTestRun** (actual execution).

## Changes Completed

### 1. Domain Models (`internal/domain/models.go`)
- **Replaced** `TestRun` with:
  - `LoadTest`: Test definition with name, description, tags, target URL, locustfile, default parameters
  - `LoadTestRun`: Actual execution referencing a LoadTest, with runtime parameter overrides
- **Added** audit fields to both entities:
  - `CreatedAt`, `CreatedBy`, `UpdatedAt`, `UpdatedBy`
- **Updated** `LoadTestRunStatus` (renamed from `TestRunStatus`)
- **Added** missing response time fields to `MetricSnapshot` and `ReqStat`

### 2. Memory Store (`internal/store/memory_store.go`)
- **Created** `LoadTestRepository` interface and `InMemoryLoadTestStore` implementation
- **Created** `LoadTestRunRepository` interface and `InMemoryLoadTestRunStore` implementation
- **Added** `LoadTestFilter` with support for filtering by tenant, env, and tags
- **Added** `LoadTestRunFilter` with support for filtering by loadTestId, tenant, env, and status
- **Updated** copy functions for deep cloning
- **Added** `hasAnyTag` helper for tag-based filtering

### 3. MongoDB Store (`internal/store/mongo_testrun_store.go`)
- **Created** `MongoLoadTestStore` with collection `load_tests`
  - Indexes: id (unique), tenant+env, tags, createdAt, tenant+createdAt
  - Full CRUD operations
- **Created** `MongoLoadTestRunStore` with collection `load_test_runs`
  - Indexes: id (unique), loadTestId, loadTestId+createdAt, tenant+env+status, status, createdAt
  - Full CRUD operations
  - Supports filtering by loadTestId for listing runs of a specific test

### 4. API DTOs (`internal/api/dto.go`)
- **Added** `CreateLoadTestRequest`, `UpdateLoadTestRequest`, `LoadTestResponse`
- **Added** `CreateLoadTestRunRequest`, `LoadTestRunResponse`
- **Added** conversion helpers: `toLoadTestResponse()`, `toLoadTestRunResponse()`
- Runtime parameters in `CreateLoadTestRunRequest` are optional and override LoadTest defaults

## Collections Structure

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
  "defaultUsers": "int",
  "defaultSpawnRate": "float",
  "defaultDurationSec": "int",
  "maxDurationSec": "int",
  "createdAt": "timestamp",
  "createdBy": "string",
  "updatedAt": "timestamp",
  "updatedBy": "string",
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
  "targetUsers": "int",
  "spawnRate": "float",
  "durationSeconds": "int",
  "status": "Pending|Running|Stopping|Finished|Failed",
  "startedAt": "timestamp",
  "finishedAt": "timestamp",
  "lastMetrics": {},
  "createdAt": "timestamp",
  "createdBy": "string",
  "updatedAt": "timestamp",
  "updatedBy": "string",
  "metadata": {}
}
```

## Next Steps Required

### 1. Service Layer Updates
File: `internal/service/orchestrator.go`
- Add LoadTest CRUD methods
- Update LoadTestRun creation to:
  - Fetch LoadTest by ID
  - Apply defaults from LoadTest
  - Allow runtime overrides
  - Copy tenantId, envId from LoadTest
- Update all TestRun references to LoadTestRun

### 2. API Handlers
File: `internal/api/handlers.go`
- Add LoadTest CRUD endpoints:
  - `POST /v1/load-tests` - Create LoadTest
  - `GET /v1/load-tests` - List LoadTests
  - `GET /v1/load-tests/{id}` - Get LoadTest
  - `PUT /v1/load-tests/{id}` - Update LoadTest
  - `DELETE /v1/load-tests/{id}` - Delete LoadTest
- Add LoadTestRun endpoints:
  - `POST /v1/load-tests/{id}/runs` - Start a run
  - `GET /v1/load-tests/{id}/runs` - List runs for a test
  - `GET /v1/runs/{id}` - Get run details
  - `POST /v1/runs/{id}/stop` - Stop a run
- Update existing handlers to use new entities

### 3. Main Application
File: `cmd/controlplane/main.go`
- Initialize both MongoDB stores
- Initialize both in-memory stores
- Wire up new handlers
- Update router registration

### 4. Visualization Handlers
File: `internal/api/visualization_handlers.go`
- Update to use LoadTestRun instead of TestRun
- Update filtering to support loadTestId

### 5. Metrics Store
File: `internal/store/mongo_metrics_store.go`
- Update StoreMetric to accept loadTestRunId instead of testRunId
- Update queries to use new field names

## API Usage Example

### 1. Create a LoadTest
```bash
POST /v1/load-tests
{
  "name": "API Performance Test",
  "description": "Tests API endpoints under load",
  "tags": ["api", "performance"],
  "tenantId": "tenant-1",
  "envId": "prod",
  "locustClusterId": "cluster-1",
  "targetUrl": "https://api.example.com",
  "locustfile": "api_test.py",
  "defaultUsers": 100,
  "defaultSpawnRate": 10,
  "defaultDurationSec": 300,
  "createdBy": "user@example.com"
}
```

### 2. Run the LoadTest
```bash
POST /v1/load-tests/{loadTestId}/runs
{
  "name": "Peak hour test",
  "targetUsers": 200,  # Override default
  "spawnRate": 20,     # Override default
  "createdBy": "user@example.com"
}
```

### 3. List runs for a LoadTest
```bash
GET /v1/load-tests/{loadTestId}/runs
```

### 4. Stop a run
```bash
POST /v1/runs/{runId}/stop
```

## Benefits

1. **Reusability**: Define a test once, run multiple times with different parameters
2. **History**: Track all runs of a specific test
3. **Governance**: Audit trail with created/updated by fields
4. **Organization**: Tag and categorize tests for easy discovery
5. **Defaults**: Set sensible defaults, override at runtime when needed
6. **Separation of Concerns**: Test definition separate from execution state

## Migration Notes

- Old `test_runs` collection is replaced by `load_tests` and `load_test_runs`
- No backward compatibility maintained (as per your requirement)
- All TestRun references need to be updated to LoadTestRun
- Metrics collection now references load test runs instead of test runs
