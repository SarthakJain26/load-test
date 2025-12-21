# MongoDB Integration Guide

This guide covers the MongoDB integration for persistent storage and time-series metrics in the Load Testing Control Plane.

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Setup Instructions](#setup-instructions)
- [Database Schema](#database-schema)
- [Visualization APIs](#visualization-apis)
- [Performance Optimization](#performance-optimization)
- [Troubleshooting](#troubleshooting)

---

## Overview

The MongoDB integration provides:
- **Persistent storage** for test runs in a regular collection
- **Time-series metrics** storage for real-time performance data
- **Optimized indexes** for fast queries
- **Visualization APIs** for generating charts and graphs

### Key Features
- ✅ Automatic time-series collection creation
- ✅ Comprehensive indexing strategy
- ✅ Real-time metrics ingestion
- ✅ Chart-ready data endpoints
- ✅ Aggregation pipelines for analytics

---

## Architecture

### Collections

#### 1. `test_runs` (Regular Collection)
Stores test run metadata and final results.

**Schema:**
```javascript
{
  _id: ObjectId,
  id: String (unique),           // UUID
  tenantId: String,
  envId: String,
  locustClusterId: String,
  scenarioId: String,
  targetUsers: Int,
  spawnRate: Float,
  durationSeconds: Int,
  status: String,                // Pending, Running, Finished, Failed
  createdAt: Date,
  startedAt: Date,
  finishedAt: Date,
  lastMetrics: {
    totalRps: Float,
    totalRequests: Long,
    totalFailures: Long,
    errorRate: Float,
    currentUsers: Int,
    p50ResponseMs: Float,
    p95ResponseMs: Float,
    p99ResponseMs: Float,
    // ... more fields
  },
  metadata: Object
}
```

**Indexes:**
- `id` (unique)
- `tenantId + envId + status` (compound)
- `status`
- `createdAt` (descending)
- `tenantId + createdAt` (compound, descending)

#### 2. `metrics_timeseries` (Time-Series Collection)
Stores real-time metric snapshots for visualization.

**Time-Series Configuration:**
- `timeField`: timestamp
- `metaField`: testRunId
- `granularity`: seconds

**Schema:**
```javascript
{
  _id: ObjectId,
  timestamp: Date,               // Time-series time field
  testRunId: String,             // Time-series meta field
  tenantId: String,
  envId: String,
  totalRps: Float,
  totalRequests: Long,
  totalFailures: Long,
  errorRate: Float,
  currentUsers: Int,
  p50ResponseMs: Float,
  p95ResponseMs: Float,
  p99ResponseMs: Float,
  minResponseMs: Float,
  maxResponseMs: Float,
  avgResponseMs: Float,
  requestStats: [
    {
      method: String,
      name: String,
      numRequests: Long,
      numFailures: Long,
      avgResponseTimeMs: Float,
      minResponseTimeMs: Float,
      maxResponseTimeMs: Float,
      p50ResponseMs: Float,
      p95ResponseMs: Float,
      requestsPerSec: Float
    }
  ],
  metadata: Object
}
```

**Indexes:**
- `testRunId + timestamp` (compound)
- `tenantId + envId + timestamp` (compound, descending)
- `testRunId`

---

## Setup Instructions

### 1. Start MongoDB

**Using Docker Compose (Recommended):**
```bash
cd /Users/sarthakjain/harness/Load-manager-cli

# Start MongoDB and Mongo Express UI
docker-compose -f docker-compose.mongodb.yml up -d

# Verify MongoDB is running
docker-compose -f docker-compose.mongodb.yml ps

# View logs
docker-compose -f docker-compose.mongodb.yml logs -f mongodb
```

**Access Mongo Express UI:**
- URL: http://localhost:8081
- Username: `admin`
- Password: `admin123`

**Using Local MongoDB:**
```bash
# macOS (Homebrew)
brew tap mongodb/brew
brew install mongodb-community@7.0
brew services start mongodb-community@7.0

# Verify connection
mongosh --eval "db.adminCommand('ping')"
```

### 2. Configure Control Plane

Update `config/config.yaml`:
```yaml
mongodb:
  uri: "mongodb://localhost:27017"
  database: "load_testing"
  connectTimeoutSeconds: 10
  maxPoolSize: 100
```

**For production:**
```yaml
mongodb:
  uri: "mongodb://username:password@mongodb-host:27017/load_testing?authSource=admin"
  database: "load_testing"
  connectTimeoutSeconds: 10
  maxPoolSize: 100
```

### 3. Start Control Plane

```bash
# Build
go build -o bin/controlplane cmd/controlplane/main.go

# Run
./bin/controlplane -config config/config.yaml
```

**Expected logs:**
```
Connecting to MongoDB...
Connected to MongoDB successfully
Test run store initialized with indexes
Metrics time-series store initialized with indexes
Orchestrator started
```

### 4. Verify Setup

```bash
# Check MongoDB collections
mongosh load_testing --eval "show collections"

# Expected output:
# metrics_timeseries
# test_runs

# Check indexes
mongosh load_testing --eval "db.test_runs.getIndexes()"
mongosh load_testing --eval "db.metrics_timeseries.getIndexes()"
```

---

## Database Schema

### Test Run Lifecycle

```
1. Create Test (POST /v1/tests)
   └─> Insert into test_runs (status: Pending)

2. Test Starts
   └─> Update test_runs (status: Running, startedAt: timestamp)

3. Metrics Polling (every 10s)
   ├─> Insert into metrics_timeseries (real-time snapshot)
   └─> Update test_runs.lastMetrics (latest values)

4. Test Stops
   └─> Update test_runs (status: Finished, finishedAt: timestamp)
```

### Data Flow Diagram

```
┌─────────────┐
│   Locust    │
│   Master    │
└──────┬──────┘
       │ Stats API
       ▼
┌──────────────────┐
│  Orchestrator    │
│  (Metrics Poller)│
└────┬────────┬────┘
     │        │
     │        └──────────────────┐
     ▼                           ▼
┌─────────────┐         ┌──────────────────┐
│  test_runs  │         │ metrics_timeseries│
│ Collection  │         │   (Time-Series)   │
└─────────────┘         └──────────────────┘
     │                           │
     └───────────┬───────────────┘
                 ▼
         ┌──────────────┐
         │ Visualization│
         │     APIs     │
         └──────────────┘
```

---

## Visualization APIs

### 1. Time-Series Chart Data (Line Charts)

**Endpoint:** `GET /v1/tests/{id}/metrics/timeseries`

**Query Parameters:**
- `from` (optional): RFC3339 timestamp (e.g., `2024-12-21T00:00:00Z`)
- `to` (optional): RFC3339 timestamp

**Example:**
```bash
curl -X GET "http://localhost:8080/v1/tests/{test_id}/metrics/timeseries" \
  -H "Authorization: Bearer your-api-token" | jq '.'

# With time range
curl -X GET "http://localhost:8080/v1/tests/{test_id}/metrics/timeseries?from=2024-12-21T10:00:00Z&to=2024-12-21T11:00:00Z" \
  -H "Authorization: Bearer your-api-token" | jq '.'
```

**Response:**
```json
{
  "testRunId": "abc-123",
  "dataPoints": [
    {
      "timestamp": "2024-12-21T10:00:00Z",
      "totalRps": 150.5,
      "currentUsers": 50,
      "p50ResponseMs": 45.2,
      "p95ResponseMs": 120.5,
      "p99ResponseMs": 250.0,
      "errorRate": 0.5
    },
    {
      "timestamp": "2024-12-21T10:00:10Z",
      "totalRps": 155.3,
      "currentUsers": 50,
      "p50ResponseMs": 43.8,
      "p95ResponseMs": 118.2,
      "p99ResponseMs": 245.0,
      "errorRate": 0.4
    }
  ],
  "summary": {
    "avgRps": 145.3,
    "maxRps": 200.0,
    "minRps": 100.0,
    "avgP50Latency": 44.5,
    "avgP95Latency": 115.0,
    "avgP99Latency": 247.5,
    "maxP95Latency": 125.0,
    "totalRequests": 10000,
    "totalFailures": 50,
    "overallErrorRate": 0.5,
    "dataPoints": 18,
    "duration": "3m0s"
  }
}
```

**Use Case:** Plot line charts for RPS, latency, and error rate over time.

### 2. Scatter Plot Data

**Endpoint:** `GET /v1/tests/{id}/metrics/scatter`

**Example:**
```bash
curl -X GET "http://localhost:8080/v1/tests/{test_id}/metrics/scatter" \
  -H "Authorization: Bearer your-api-token" | jq '.'
```

**Response:**
```json
{
  "testRunId": "abc-123",
  "dataPoints": [
    {
      "timestamp": "2024-12-21T10:00:00Z",
      "endpoint": "/api/users",
      "method": "GET",
      "responseTimeMs": 45.2,
      "success": true
    },
    {
      "timestamp": "2024-12-21T10:00:00Z",
      "endpoint": "/api/products",
      "method": "GET",
      "responseTimeMs": 120.5,
      "success": true
    }
  ],
  "endpoints": [
    "GET /api/users",
    "GET /api/products",
    "POST /api/orders"
  ]
}
```

**Use Case:** Plot scatter plots showing response time distribution per endpoint.

### 3. Aggregated Statistics

**Endpoint:** `GET /v1/tests/{id}/metrics/aggregate`

**Example:**
```bash
curl -X GET "http://localhost:8080/v1/tests/{test_id}/metrics/aggregate" \
  -H "Authorization: Bearer your-api-token" | jq '.'
```

**Response:**
```json
{
  "testRunId": "abc-123",
  "status": "Finished",
  "timeseries": [...],
  "endpointStats": [
    {
      "endpoint": "/api/users",
      "method": "GET",
      "totalRequests": 5000,
      "totalFailures": 10,
      "errorRate": 0.2,
      "avgResponseTimeMs": 45.5,
      "minResponseTimeMs": 20.0,
      "maxResponseTimeMs": 200.0,
      "p50ResponseMs": 43.0,
      "p95ResponseMs": 120.0,
      "avgRps": 83.3
    }
  ],
  "summary": {
    "avgRps": 145.3,
    "maxRps": 200.0,
    "totalRequests": 10000,
    "duration": "3m0s"
  }
}
```

**Use Case:** Complete dashboard view with all metrics and per-endpoint breakdown.

---

## Performance Optimization

### Index Strategy

**test_runs Collection:**
```javascript
// Fast lookups by ID
{ id: 1 } [unique]

// Multi-tenant queries
{ tenantId: 1, envId: 1, status: 1 }

// Status filtering
{ status: 1 }

// Time-based queries
{ createdAt: -1 }

// Tenant history
{ tenantId: 1, createdAt: -1 }
```

**metrics_timeseries Collection:**
```javascript
// Time-range queries for specific test
{ testRunId: 1, timestamp: 1 }

// Multi-tenant time-series queries
{ tenantId: 1, envId: 1, timestamp: -1 }

// Fast test lookup
{ testRunId: 1 }
```

### Query Optimization Tips

**1. Use Time-Range Filters:**
```bash
# Good - limits data range
GET /v1/tests/{id}/metrics/timeseries?from=2024-12-21T10:00:00Z&to=2024-12-21T11:00:00Z

# Less efficient - returns all data
GET /v1/tests/{id}/metrics/timeseries
```

**2. Limit Data Points:**
MongoDB time-series collections automatically optimize storage and queries.

**3. Use Aggregation Pipelines:**
The `/metrics/aggregate` endpoint uses MongoDB aggregation for efficient data processing.

### Connection Pooling

Configure in `config.yaml`:
```yaml
mongodb:
  maxPoolSize: 100  # Adjust based on load
  connectTimeoutSeconds: 10
```

**Recommendations:**
- **Development:** 10-20 connections
- **Production (low traffic):** 50-100 connections
- **Production (high traffic):** 200-500 connections

---

## Troubleshooting

### Issue: MongoDB Connection Fails

**Error:**
```
Failed to connect to MongoDB: connection timeout
```

**Solutions:**
1. Verify MongoDB is running:
   ```bash
   docker-compose -f docker-compose.mongodb.yml ps
   ```

2. Check connectivity:
   ```bash
   mongosh --eval "db.adminCommand('ping')"
   ```

3. Verify URI in config:
   ```yaml
   mongodb:
     uri: "mongodb://localhost:27017"  # Check host and port
   ```

### Issue: Indexes Not Created

**Error:**
```
Failed to create indexes: ...
```

**Solutions:**
1. Check MongoDB version (7.0+ required for time-series):
   ```bash
   mongosh --eval "db.version()"
   ```

2. Manually create indexes:
   ```javascript
   use load_testing
   db.test_runs.createIndex({ id: 1 }, { unique: true })
   db.test_runs.createIndex({ tenantId: 1, envId: 1, status: 1 })
   ```

### Issue: No Metrics Data

**Symptoms:** `/metrics/timeseries` returns empty array

**Solutions:**
1. Verify test is running:
   ```bash
   curl http://localhost:8080/v1/tests/{id} | jq '.status'
   ```

2. Check orchestrator logs for metrics polling:
   ```
   Polled metrics for run abc-123: RPS=150.00, Requests=1000
   ```

3. Verify metrics store is initialized:
   ```
   Metrics time-series store initialized with indexes
   ```

### Issue: Slow Queries

**Solutions:**
1. Check index usage:
   ```javascript
   db.metrics_timeseries.find({ testRunId: "abc-123" }).explain("executionStats")
   ```

2. Use time-range filters to limit data:
   ```bash
   GET /metrics/timeseries?from=...&to=...
   ```

3. Monitor connection pool:
   ```javascript
   db.serverStatus().connections
   ```

---

## Data Retention

### Time-Series Collection TTL

**Set automatic expiration (optional):**
```javascript
use load_testing

// Expire metrics after 90 days
db.metrics_timeseries.createIndex(
  { "timestamp": 1 },
  { expireAfterSeconds: 7776000 }  // 90 days
)
```

### Manual Cleanup

**Delete old test runs:**
```javascript
// Delete tests older than 180 days
db.test_runs.deleteMany({
  createdAt: { $lt: new Date(Date.now() - 180*24*60*60*1000) }
})

// Delete associated metrics
db.metrics_timeseries.deleteMany({
  timestamp: { $lt: new Date(Date.now() - 180*24*60*60*1000) }
})
```

---

## Monitoring

### Key Metrics to Track

**MongoDB Metrics:**
```javascript
// Connection pool usage
db.serverStatus().connections

// Operation statistics
db.serverStatus().opcounters

// Collection statistics
db.test_runs.stats()
db.metrics_timeseries.stats()
```

**Application Metrics:**
- Metrics ingestion rate (documents/second)
- Query response times
- Index hit ratio
- Connection pool saturation

---

## Next Steps

1. **Set up monitoring:** Use MongoDB Atlas, Prometheus, or Datadog
2. **Configure backups:** Set up automated backups for production
3. **Optimize indexes:** Monitor query patterns and adjust indexes
4. **Scale horizontally:** Use MongoDB sharding for high-volume workloads
5. **Implement authentication:** Secure MongoDB in production environments

For more information, see:
- [MongoDB Time-Series Collections](https://www.mongodb.com/docs/manual/core/timeseries-collections/)
- [MongoDB Indexing Best Practices](https://www.mongodb.com/docs/manual/applications/indexes/)
- [Main README](../README.md)
