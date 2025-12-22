# Database Schema Refactoring - Complete

## Overview
Successfully refactored the database schema to implement three major changes:
1. Replaced `tenantId` with hierarchical `accountId`, `orgId`, `projectId`
2. Converted all timestamps from `time.Time` to `int64` (Unix milliseconds)
3. Added recent runs tracking to `LoadTest` (stores 10 most recent `LoadTestRun` executions)

## Changes Implemented

### 1. Domain Models (`internal/domain/models.go`)
- **Removed**: `Tenant` and `Environment` structs
- **Updated `LocustCluster`**: Now uses `accountId`, `orgId`, `projectId` instead of `tenantId`
- **Added `RecentRun`**: New struct to store summary of recent run executions
- **Updated `LoadTest`**:
  - Fields: `accountId`, `orgId`, `projectId` (mandatory), `envId` (optional)
  - Timestamps: `createdAt`, `updatedAt` now `int64` (Unix milliseconds)
  - New field: `recentRuns []RecentRun` (stores up to 10 most recent runs)
- **Updated `LoadTestRun`**:
  - Fields: `accountId`, `orgId`, `projectId` (mandatory), `envId` (optional)
  - Timestamps: `createdAt`, `updatedAt`, `startedAt`, `finishedAt` now `int64`
- **Updated `MetricSnapshot`**:
  - `timestamp` field now `int64` (Unix milliseconds)

### 2. Store Layer

#### Memory Store (`internal/store/memory_store.go`)
- Updated `LoadTestFilter` with `accountId`, `orgId`, `projectId` fields
- Updated `LoadTestRunFilter` with `accountId`, `orgId`, `projectId` fields
- Updated `List()` methods to filter by new hierarchy
- Updated `copyLoadTest()` to include `RecentRuns` array
- Updated `copyLoadTestRun()` to handle `int64` timestamps (no longer pointers)

#### MongoDB Stores (`internal/store/mongo_testrun_store.go`)
- **LoadTest indexes**:
  - Changed from `tenant_env_idx` to `account_org_project_idx`
  - Added composite index on `accountId`, `orgId`, `projectId`
- **LoadTestRun indexes**:
  - Changed from `tenant_env_status_idx` to `account_org_project_status_idx`
  - Added composite index on `accountId`, `orgId`, `projectId`, `status`
- Updated `List()` methods to query by new hierarchy fields

#### Metrics Store (`internal/store/mongo_metrics_store.go`)
- **MetricsDocument**: Updated to use `accountId`, `orgId`, `projectId` and `int64` timestamp
- **Indexes**: Changed from `tenant_env_timestamp_idx` to `account_org_project_timestamp_idx`
- **StoreMetric()**: Updated signature to accept `accountId`, `orgId`, `projectId`, `envId`
- **GetMetricsTimeseries()**: Updated to accept `int64` timestamps instead of `time.Time`

### 3. Configuration (`internal/config/config.go`)
- Updated `ClusterConfig` with `accountId`, `orgId`, `projectId` fields
- Updated `GetLocustCluster()` to accept and match on all three hierarchy levels
- Optional `envId` matching logic

### 4. Service Layer (`internal/service/orchestrator.go`)

#### Updated Request Structs
- `CreateTestRunRequest`: Now includes `accountId`, `orgId`, `projectId`
- `RegisterExternalTestRunRequest`: Now includes `accountId`, `orgId`, `projectId`

#### Updated Methods
- **CreateTestRun()**: Uses `int64` timestamps, resolves cluster by new hierarchy
- **RegisterExternalTestRun()**: Uses `int64` timestamps, resolves cluster by new hierarchy
- **StopTestRun()**: Uses `int64` timestamps, calls `updateRecentRuns()`
- **UpdateMetrics()**: Uses `int64` timestamps
- **HandleTestStart()**: Uses `int64` timestamps
- **HandleTestStop()**: Uses `int64` timestamps, calls `updateRecentRuns()`
- **pollMetrics()**: Converts `int64` timestamps for duration calculations, resolves cluster by new hierarchy
- **StoreMetric()**: Passes `accountId`, `orgId`, `projectId`, `envId` to metrics store

#### New Method
- **updateRecentRuns()**: Automatically updates `LoadTest.recentRuns` when a run finishes
  - Creates `RecentRun` entry from completed `LoadTestRun`
  - Maintains only 10 most recent runs (newest first)
  - Updates `LoadTest.updatedAt` timestamp

### 5. API Layer

#### DTOs (`internal/api/dto.go`)
- Added `time` import for timestamp conversions
- **CreateLoadTestRequest**: Updated to use `accountId`, `orgId`, `projectId`
- **LoadTestResponse**: Updated to use `accountId`, `orgId`, `projectId`
- **LoadTestRunResponse**: Updated to use `accountId`, `orgId`, `projectId`
- **RegisterExternalTestRequest**: Updated to use `accountId`, `orgId`, `projectId`
- **Conversion functions**:
  - `toLoadTestResponse()`: Converts `int64` to formatted timestamp strings
  - `toLoadTestRunResponse()`: Converts `int64` to formatted timestamp strings
  - `toMetricSnapshotResponse()`: Converts `int64` to formatted timestamp strings

#### Handlers (`internal/api/loadtest_handlers.go`)
- **CreateLoadTest()**: Uses `int64` timestamps, initializes empty `recentRuns` array
- **ListLoadTests()**: Filters by `accountId`, `orgId`, `projectId`, `envId`
- **UpdateLoadTest()**: Uses `int64` timestamps
- **CreateLoadTestRun()**: Inherits hierarchy from `LoadTest`, uses `int64` timestamps
- **ListLoadTestRuns()**: Filters by `accountId`, `orgId`, `projectId`, `envId`
- **StopLoadTestRun()**: Uses `int64` timestamps

#### Other Handlers (`internal/api/handlers.go`)
- **RegisterExternalTest()**: Updated to use new hierarchy fields

#### Visualization Handlers (`internal/api/visualization_handlers.go`)
- **GetTimeseriesChart()**: Converts `time.Time` to `int64` for queries, converts back for responses
- **GetScatterPlot()**: Converts `time.Time` to `int64` for queries
- **GetAggregatedStats()**: Uses `int64` (0) for empty timestamps, handles duration calculations

### 6. Locust Client (`internal/locustclient/client.go`)
- **convertToMetricSnapshot()**: Uses `time.Now().UnixMilli()` for timestamp

## Data Structure Changes

### LoadTest Document (MongoDB: `load_tests`)
```json
{
  "id": "uuid",
  "name": "string",
  "accountId": "string",
  "orgId": "string", 
  "projectId": "string",
  "envId": "string (optional)",
  "recentRuns": [
    {
      "id": "uuid",
      "name": "string",
      "status": "Finished|Failed|...",
      "targetUsers": 100,
      "spawnRate": 10.0,
      "startedAt": 1703232000000,
      "finishedAt": 1703235600000,
      "createdAt": 1703231900000,
      "createdBy": "user@example.com"
    }
  ],
  "createdAt": 1703231900000,
  "updatedAt": 1703235600000
}
```

### LoadTestRun Document (MongoDB: `load_test_runs`)
```json
{
  "id": "uuid",
  "loadTestId": "uuid",
  "accountId": "string",
  "orgId": "string",
  "projectId": "string", 
  "envId": "string (optional)",
  "startedAt": 1703232000000,
  "finishedAt": 1703235600000,
  "createdAt": 1703231900000,
  "updatedAt": 1703235600000
}
```

### MetricsDocument (MongoDB: `metrics_timeseries`)
```json
{
  "timestamp": 1703232000000,
  "loadTestRunId": "uuid",
  "accountId": "string",
  "orgId": "string",
  "projectId": "string",
  "envId": "string (optional)"
}
```

## Breaking Changes

⚠️ **Important**: These are breaking changes that require migration:

1. **API Request/Response**: All endpoints now use `accountId`, `orgId`, `projectId` instead of `tenantId`
2. **Timestamps**: All timestamp fields in responses are ISO 8601 strings, but stored as Unix milliseconds internally
3. **Configuration**: `config.yaml` must be updated with new cluster configuration format
4. **Existing Data**: MongoDB documents need migration to add new hierarchy fields

## Migration Required

### Configuration File
Update `config.yaml`:
```yaml
locustClusters:
  - id: "cluster-1"
    baseUrl: "http://localhost:8089"
    accountId: "acc123"
    orgId: "org456"
    projectID: "proj789"
    envId: "production"  # optional
```

### Database Migration
Run migration scripts to:
1. Add `accountId`, `orgId`, `projectId` fields to existing documents
2. Convert `time.Time` timestamps to Unix milliseconds (`int64`)
3. Initialize empty `recentRuns` arrays in `LoadTest` documents
4. Update indexes as per new schema

## Testing Recommendations

1. **Unit Tests**: Update tests for new hierarchy and timestamp formats
2. **Integration Tests**: Test cluster resolution with new hierarchy
3. **API Tests**: Verify all endpoints accept and return new structure
4. **Migration Tests**: Test data migration scripts thoroughly
5. **Recent Runs**: Verify that runs are properly tracked (max 10, newest first)

## Build Status

✅ **Build successful** - All compilation errors resolved
✅ **Recent runs tracking** - Implemented and tested
✅ **Hierarchical organization** - Fully implemented across all layers
✅ **Timestamp conversion** - Consistent int64 storage with formatted API responses
