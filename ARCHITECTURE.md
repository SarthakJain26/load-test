# Load Manager Control Plane - Architecture & Flow

## System Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         External Clients                                 │
│                    (API Consumers, UI, CLI)                              │
└────────────────────────────────┬────────────────────────────────────────┘
                                 │ HTTP/REST API
                                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Control Plane (Go)                               │
│                                                                           │
│  ┌────────────────────┐  ┌──────────────────┐  ┌───────────────────┐   │
│  │   API Handlers     │  │   Orchestrator   │  │  Locust Clients   │   │
│  │                    │  │                  │  │                   │   │
│  │ • LoadTest CRUD    │  │ • Test Lifecycle │  │ • HTTP Client     │   │
│  │ • LoadTestRun CRUD │  │ • Metrics Polling│  │ • Swarm Control   │   │
│  │ • Visualization    │  │ • Duration Check │  │ • Stats Fetching  │   │
│  │ • Locust Callbacks │  │ • Status Updates │  │ • Stop Commands   │   │
│  └─────────┬──────────┘  └────────┬─────────┘  └─────────┬─────────┘   │
│            │                      │                       │             │
│            │        ┌─────────────┴──────────┐           │             │
│            │        │                        │           │             │
│            ▼        ▼                        ▼           ▼             │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │                     Store Layer                                   │  │
│  │                                                                    │  │
│  │  ┌────────────────┐  ┌──────────────────┐  ┌──────────────────┐ │  │
│  │  │ LoadTestStore  │  │LoadTestRunStore  │  │  MetricsStore    │ │  │
│  │  │  (MongoDB)     │  │   (MongoDB)      │  │   (MongoDB)      │ │  │
│  │  └────────────────┘  └──────────────────┘  └──────────────────┘ │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│            │                      │                       │             │
└────────────┼──────────────────────┼───────────────────────┼─────────────┘
             │                      │                       │
             ▼                      ▼                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         MongoDB Database                                 │
│                                                                           │
│  ┌──────────────┐  ┌──────────────────┐  ┌─────────────────────────┐   │
│  │ load_tests   │  │ load_test_runs   │  │ metrics_timeseries      │   │
│  │              │  │                  │  │                         │   │
│  │ • name       │  │ • loadTestId ────┼──┤ • loadTestRunId         │   │
│  │ • targetUrl  │  │ • status         │  │ • timestamp             │   │
│  │ • locustfile │  │ • targetUsers    │  │ • totalRPS              │   │
│  │ • defaults   │  │ • spawnRate      │  │ • p50/p95/p99 latency   │   │
│  │ • audit      │  │ • metrics        │  │ • requestStats[]        │   │
│  └──────────────┘  │ • audit          │  └─────────────────────────┘   │
│                    └──────────────────┘                                  │
└─────────────────────────────────────────────────────────────────────────┘
             │                      │                       │
             │                      │                       │
             └──────────────────────┴───────────────────────┘
                                    │
                          HTTP API Calls (Swarm, Stop, Stats)
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Locust Cluster                                   │
│                                                                           │
│  ┌────────────────┐                      ┌──────────────────────┐       │
│  │ Locust Master  │◄─────────────────────┤  Locust Workers      │       │
│  │                │                      │                      │       │
│  │ • Coordinates  │                      │ • Execute Load       │       │
│  │ • Collects     │                      │ • Report Metrics     │       │
│  │ • Provides API │                      │                      │       │
│  └────────────────┘                      └──────────────────────┘       │
│         │                                                                │
│         │ HTTP Requests                                                  │
│         ▼                                                                │
│  ┌────────────────────────────────────────────────────────────────┐     │
│  │              Target Application Under Test                      │     │
│  └────────────────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. API Handlers Layer
- **LoadTest Handlers**: CRUD operations for test definitions
- **LoadTestRun Handlers**: Create and manage test executions
- **Visualization Handlers**: Serve metrics and charts data
- **Locust Callback Handlers**: Receive events from Locust

### 2. Orchestrator (Service Layer)
- **Lifecycle Management**: Controls test execution lifecycle
- **Metrics Polling**: Periodically fetches metrics from Locust
- **Duration Enforcement**: Stops tests when duration limit reached
- **Client Management**: Maintains connections to Locust clusters

### 3. Store Layer
- **LoadTestRepository**: Manages test definitions
- **LoadTestRunRepository**: Manages test executions
- **MetricsStore**: Stores time-series performance data

### 4. Locust Clients
- **HTTP Client**: Communicates with Locust Master API
- **Operations**: Swarm (start), Stop, GetStats

---

## Flow 1: Creating a LoadTest

```
User/API Client                Control Plane                    MongoDB
      │                              │                              │
      │  POST /v1/load-tests         │                              │
      ├─────────────────────────────►│                              │
      │  {                           │                              │
      │    name: "API Test",         │                              │
      │    targetUrl: "...",         │                              │
      │    locustfile: "...",        │  INSERT load_test            │
      │    defaultUsers: 100,        ├─────────────────────────────►│
      │    createdBy: "user@..."     │                              │
      │  }                           │                              │
      │                              │◄─────────────────────────────┤
      │                              │  Document ID + Indexes       │
      │                              │                              │
      │◄─────────────────────────────┤                              │
      │  201 Created                 │                              │
      │  {                           │                              │
      │    id: "lt-123",             │                              │
      │    name: "API Test",         │                              │
      │    status: "Created",        │                              │
      │    createdAt: "...",         │                              │
      │    ...                       │                              │
      │  }                           │                              │
      │                              │                              │
```

**Steps:**
1. Client sends POST request with LoadTest definition
2. Handler validates request (name, targetUrl, locustfile required)
3. Handler creates LoadTest entity with audit fields
4. Store persists to MongoDB `load_tests` collection
5. MongoDB creates indexes (tenantId, envId, tags, createdAt)
6. Handler returns LoadTest response with ID

---

## Flow 2: Starting a LoadTestRun

```
User/Client          Control Plane              MongoDB           Locust Master
    │                     │                         │                   │
    │  POST /v1/load-tests/{id}/runs                │                   │
    ├────────────────────►│                         │                   │
    │  {                  │                         │                   │
    │   targetUsers: 200, │  GET LoadTest           │                   │
    │   spawnRate: 20     ├────────────────────────►│                   │
    │  }                  │                         │                   │
    │                     │◄────────────────────────┤                   │
    │                     │  LoadTest details       │                   │
    │                     │  (defaults, cluster)    │                   │
    │                     │                         │                   │
    │                     │  Merge defaults         │                   │
    │                     │  + overrides            │                   │
    │                     │                         │                   │
    │                     │  CREATE LoadTestRun     │                   │
    │                     │  (Status: Pending)      │                   │
    │                     ├────────────────────────►│                   │
    │                     │                         │                   │
    │                     │◄────────────────────────┤                   │
    │                     │  Run ID: "run-456"      │                   │
    │                     │                         │                   │
    │                     │  POST /swarm            │                   │
    │                     ├────────────────────────────────────────────►│
    │                     │  {                      │                   │
    │                     │   user_count: 200,      │                   │
    │                     │   spawn_rate: 20        │                   │
    │                     │  }                      │                   │
    │                     │                         │                   │
    │                     │◄────────────────────────────────────────────┤
    │                     │  200 OK - Swarm started │                   │
    │                     │                         │                   │
    │                     │  UPDATE LoadTestRun     │                   │
    │                     │  (Status: Running,      │                   │
    │                     │   StartedAt: now)       │                   │
    │                     ├────────────────────────►│                   │
    │                     │                         │                   │
    │◄────────────────────┤                         │                   │
    │  201 Created        │                         │                   │
    │  {                  │                         │                   │
    │   id: "run-456",    │                         │                   │
    │   loadTestId: "lt-123",                       │                   │
    │   status: "Running",│                         │                   │
    │   targetUsers: 200, │                         │                   │
    │   ...               │                         │                   │
    │  }                  │                         │                   │
    │                     │                         │                   │
```

**Steps:**
1. Client sends POST request to start a run with optional overrides
2. Handler fetches LoadTest from database
3. Handler merges LoadTest defaults with runtime overrides
4. Handler creates LoadTestRun with status "Pending"
5. Handler resolves Locust cluster from config (tenant + env)
6. Orchestrator calls Locust Master `/swarm` API
7. On success, updates LoadTestRun status to "Running"
8. Returns LoadTestRun response to client

---

## Flow 3: Metrics Collection (Polling)

```
Orchestrator         LoadTestRunStore         MetricsStore      Locust Master
    │                       │                       │                 │
    │  [Every N seconds]    │                       │                 │
    │                       │                       │                 │
    │  LIST running runs    │                       │                 │
    ├──────────────────────►│                       │                 │
    │                       │                       │                 │
    │◄──────────────────────┤                       │                 │
    │  [run-456, run-789]   │                       │                 │
    │                       │                       │                 │
    │  For each run:        │                       │                 │
    │                       │                       │                 │
    │  Check duration limit │                       │                 │
    │  (if elapsed ≥ max)   │                       │                 │
    │  → Stop test          │                       │                 │
    │                       │                       │                 │
    │  GET /stats           │                       │                 │
    ├─────────────────────────────────────────────────────────────────►│
    │                       │                       │                 │
    │◄─────────────────────────────────────────────────────────────────┤
    │  {                    │                       │                 │
    │   total_rps: 1250.5,  │                       │                 │
    │   num_requests: 45000,│                       │                 │
    │   num_failures: 12,   │                       │                 │
    │   response_times: {   │                       │                 │
    │     p50: 120,         │                       │                 │
    │     p95: 450,         │                       │                 │
    │     p99: 890          │                       │                 │
    │   },                  │                       │                 │
    │   current_users: 200, │                       │                 │
    │   stats: [...]        │                       │                 │
    │  }                    │                       │                 │
    │                       │                       │                 │
    │  UPDATE LoadTestRun   │                       │                 │
    │  (lastMetrics)        │                       │                 │
    ├──────────────────────►│                       │                 │
    │                       │                       │                 │
    │  STORE time-series    │                       │                 │
    ├───────────────────────────────────────────────►│                 │
    │  {                    │                       │                 │
    │   loadTestRunId: "run-456",                   │                 │
    │   timestamp: now,     │                       │                 │
    │   totalRPS: 1250.5,   │                       │                 │
    │   p50ResponseMs: 120, │                       │                 │
    │   ...                 │                       │                 │
    │  }                    │                       │                 │
    │                       │                       │                 │
    │  [Wait N seconds]     │                       │                 │
    │  [Repeat]             │                       │                 │
    │                       │                       │                 │
```

**Steps:**
1. Orchestrator has a background goroutine polling every N seconds (configurable)
2. Queries LoadTestRunStore for all runs with status "Running"
3. For each running test:
   - Checks if duration limit reached (auto-stops if exceeded)
   - Resolves Locust cluster from tenant/env
   - Calls Locust Master `/stats` API
   - Parses metrics response
   - Updates LoadTestRun with latest metrics
   - Stores snapshot in time-series collection
4. Logs metrics summary for debugging
5. Repeats cycle

---

## Flow 4: Stopping a LoadTestRun

```
User/Client          Control Plane         LoadTestRunStore    Locust Master
    │                     │                       │                 │
    │  POST /v1/runs/{id}/stop                    │                 │
    ├────────────────────►│                       │                 │
    │                     │                       │                 │
    │                     │  GET LoadTestRun      │                 │
    │                     ├──────────────────────►│                 │
    │                     │                       │                 │
    │                     │◄──────────────────────┤                 │
    │                     │  (status: Running)    │                 │
    │                     │                       │                 │
    │                     │  UPDATE status        │                 │
    │                     │  → "Stopping"         │                 │
    │                     ├──────────────────────►│                 │
    │                     │                       │                 │
    │                     │  POST /stop           │                 │
    │                     ├─────────────────────────────────────────►│
    │                     │                       │                 │
    │                     │◄─────────────────────────────────────────┤
    │                     │  200 OK - Stopped     │                 │
    │                     │                       │                 │
    │                     │  UPDATE status        │                 │
    │                     │  → "Finished"         │                 │
    │                     │  finishedAt: now      │                 │
    │                     ├──────────────────────►│                 │
    │                     │                       │                 │
    │◄────────────────────┤                       │                 │
    │  200 OK             │                       │                 │
    │  { success: true }  │                       │                 │
    │                     │                       │                 │
```

**Steps:**
1. Client sends POST to stop endpoint
2. Handler fetches LoadTestRun from database
3. Validates status is "Running"
4. Updates status to "Stopping"
5. Resolves Locust cluster and calls `/stop` API
6. On success, updates status to "Finished" with timestamp
7. Returns success response

---

## Flow 5: Locust Callbacks (Push Mode)

```
Locust Master        Control Plane         LoadTestRunStore      MetricsStore
    │                     │                       │                   │
    │  [Test Started]     │                       │                   │
    │                     │                       │                   │
    │  POST /v1/internal/locust/test-start        │                   │
    ├────────────────────►│                       │                   │
    │  {                  │                       │                   │
    │   test_run_id: "run-456"                    │                   │
    │  }                  │                       │                   │
    │                     │                       │                   │
    │                     │  UPDATE LoadTestRun   │                   │
    │                     │  (status: Running,    │                   │
    │                     │   startedAt: now)     │                   │
    │                     ├──────────────────────►│                   │
    │                     │                       │                   │
    │◄────────────────────┤                       │                   │
    │  200 OK             │                       │                   │
    │                     │                       │                   │
    │  [Test Running...]  │                       │                   │
    │                     │                       │                   │
    │  POST /v1/internal/locust/metrics           │                   │
    ├────────────────────►│                       │                   │
    │  {                  │                       │                   │
    │   test_run_id: "run-456",                   │                   │
    │   metrics: {...}    │                       │                   │
    │  }                  │  UPDATE lastMetrics   │                   │
    │                     ├──────────────────────►│                   │
    │                     │                       │                   │
    │                     │  STORE time-series    │                   │
    │                     ├───────────────────────────────────────────►│
    │                     │                       │                   │
    │◄────────────────────┤                       │                   │
    │  200 OK             │                       │                   │
    │                     │                       │                   │
    │  [Test Stopped]     │                       │                   │
    │                     │                       │                   │
    │  POST /v1/internal/locust/test-stop         │                   │
    ├────────────────────►│                       │                   │
    │  {                  │                       │                   │
    │   test_run_id: "run-456",                   │                   │
    │   final_metrics: {...}                      │                   │
    │  }                  │                       │                   │
    │                     │  UPDATE LoadTestRun   │                   │
    │                     │  (status: Finished,   │                   │
    │                     │   finishedAt: now,    │                   │
    │                     │   lastMetrics: {...}) │                   │
    │                     ├──────────────────────►│                   │
    │                     │                       │                   │
    │◄────────────────────┤                       │                   │
    │  200 OK             │                       │                   │
    │                     │                       │                   │
```

**Callback Types:**
1. **test-start**: Notifies when Locust begins a test
2. **metrics**: Pushes metrics updates during test (optional)
3. **test-stop**: Notifies when Locust finishes/stops a test

---

## Flow 6: Visualization / Metrics Retrieval

```
User/Client          API Handlers         LoadTestRunStore      MetricsStore
    │                     │                       │                   │
    │  GET /v1/runs/{id}/metrics/timeseries       │                   │
    ├────────────────────►│                       │                   │
    │  ?from=2024-01-01   │                       │                   │
    │  &to=2024-01-02     │                       │                   │
    │                     │                       │                   │
    │                     │  GET LoadTestRun      │                   │
    │                     │  (verify exists)      │                   │
    │                     ├──────────────────────►│                   │
    │                     │                       │                   │
    │                     │◄──────────────────────┤                   │
    │                     │                       │                   │
    │                     │  QUERY time-series    │                   │
    │                     │  WHERE loadTestRunId  │                   │
    │                     │  AND timestamp BETWEEN│                   │
    │                     ├───────────────────────────────────────────►│
    │                     │                       │                   │
    │                     │◄───────────────────────────────────────────┤
    │                     │  [                    │                   │
    │                     │    {timestamp, rps, p95, ...},            │
    │                     │    {timestamp, rps, p95, ...},            │
    │                     │    ...                │                   │
    │                     │  ]                    │                   │
    │                     │                       │                   │
    │                     │  AGGREGATE metrics    │                   │
    │                     │  (avg, max, min)      │                   │
    │                     ├───────────────────────────────────────────►│
    │                     │                       │                   │
    │                     │◄───────────────────────────────────────────┤
    │                     │  {                    │                   │
    │                     │    avgRPS, maxRPS,    │                   │
    │                     │    avgP50, avgP95,    │                   │
    │                     │    totalRequests, ... │                   │
    │                     │  }                    │                   │
    │                     │                       │                   │
    │◄────────────────────┤                       │                   │
    │  200 OK             │                       │                   │
    │  {                  │                       │                   │
    │   dataPoints: [...],│                       │                   │
    │   summary: {...}    │                       │                   │
    │  }                  │                       │                   │
    │                     │                       │                   │
```

**Visualization Endpoints:**
1. **timeseries**: Line chart data (RPS, latency over time)
2. **scatter**: Scatter plot (response time distribution)
3. **aggregate**: Summary statistics (avg, max, min, totals)

---

## Database Schema Relationships

```
┌────────────────────────────────────────────────────────────────┐
│                        load_tests                               │
├────────────────────────────────────────────────────────────────┤
│ • _id (ObjectId)                                                │
│ • id (String) [UNIQUE INDEX]                                   │
│ • name (String)                                                 │
│ • description (String)                                          │
│ • tags (Array<String>) [INDEX]                                 │
│ • tenantId (String) [COMPOUND INDEX: tenant+env]               │
│ • envId (String)                                                │
│ • locustClusterId (String)                                      │
│ • targetUrl (String)                                            │
│ • locustfile (String)                                           │
│ • scenarioId (String)                                           │
│ • defaultUsers (Number)                                         │
│ • defaultSpawnRate (Number)                                     │
│ • defaultDurationSec (Number)                                   │
│ • maxDurationSec (Number)                                       │
│ • createdAt (Date) [INDEX]                                      │
│ • createdBy (String)                                            │
│ • updatedAt (Date)                                              │
│ • updatedBy (String)                                            │
│ • metadata (Object)                                             │
└────────────────────────────────────────────────────────────────┘
                                    │
                                    │ 1:N relationship
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────┐
│                      load_test_runs                             │
├────────────────────────────────────────────────────────────────┤
│ • _id (ObjectId)                                                │
│ • id (String) [UNIQUE INDEX]                                   │
│ • loadTestId (String) [INDEX] → references load_tests.id       │
│ • name (String)                                                 │
│ • tenantId (String) [COMPOUND INDEX: tenant+env+status]        │
│ • envId (String)                                                │
│ • targetUsers (Number)                                          │
│ • spawnRate (Number)                                            │
│ • durationSeconds (Number)                                      │
│ • status (String: Pending|Running|Stopping|Finished|Failed)    │
│ • startedAt (Date)                                              │
│ • finishedAt (Date)                                             │
│ • lastMetrics (Object)                                          │
│ • createdAt (Date) [INDEX]                                      │
│ • createdBy (String)                                            │
│ • updatedAt (Date)                                              │
│ • updatedBy (String)                                            │
│ • metadata (Object)                                             │
└────────────────────────────────────────────────────────────────┘
                                    │
                                    │ 1:N relationship
                                    │
                                    ▼
┌────────────────────────────────────────────────────────────────┐
│                    metrics_timeseries                           │
├────────────────────────────────────────────────────────────────┤
│ • _id (ObjectId)                                                │
│ • loadTestRunId (String) [COMPOUND INDEX: runId+timestamp]     │
│ • timestamp (Date) [INDEX]                                      │
│ • tenantId (String) [COMPOUND INDEX: tenant+env+timestamp]     │
│ • envId (String)                                                │
│ • totalRPS (Number)                                             │
│ • totalRequests (Number)                                        │
│ • totalFailures (Number)                                        │
│ • errorRate (Number)                                            │
│ • currentUsers (Number)                                         │
│ • p50ResponseMs (Number)                                        │
│ • p95ResponseMs (Number)                                        │
│ • p99ResponseMs (Number)                                        │
│ • minResponseMs (Number)                                        │
│ • maxResponseMs (Number)                                        │
│ • avgResponseMs (Number)                                        │
│ • requestStats (Array<Object>)                                  │
│   ├─ method (String)                                            │
│   ├─ name (String)                                              │
│   ├─ numRequests (Number)                                       │
│   ├─ numFailures (Number)                                       │
│   ├─ avgResponseTimeMs (Number)                                 │
│   ├─ minResponseTimeMs (Number)                                 │
│   ├─ maxResponseTimeMs (Number)                                 │
│   ├─ p50ResponseMs (Number)                                     │
│   ├─ p95ResponseMs (Number)                                     │
│   └─ requestsPerSec (Number)                                    │
│ • metadata (Object)                                             │
└────────────────────────────────────────────────────────────────┘
```

---

## Communication Protocols

### Control Plane → Locust Master

**1. Start Test (Swarm)**
```http
POST http://locust-master:8089/swarm
Content-Type: application/json

{
  "user_count": 200,
  "spawn_rate": 20
}
```

**2. Stop Test**
```http
POST http://locust-master:8089/stop
```

**3. Get Statistics**
```http
GET http://locust-master:8089/stats/requests
Accept: application/json
```

**Response:**
```json
{
  "stats": [
    {
      "method": "GET",
      "name": "/api/users",
      "num_requests": 45000,
      "num_failures": 12,
      "avg_response_time": 120.5,
      "min_response_time": 45,
      "max_response_time": 890,
      "median_response_time": 105,
      "ninetieth_response_time": 350,
      "ninety_fifth_response_time": 450,
      "ninety_ninth_response_time": 890,
      "current_rps": 1250.5
    }
  ],
  "total_rps": 1250.5,
  "user_count": 200
}
```

### Locust Master → Control Plane (Callbacks)

**1. Test Start Callback**
```http
POST http://control-plane:8080/v1/internal/locust/test-start
Content-Type: application/json

{
  "test_run_id": "run-456",
  "timestamp": "2024-01-01T10:00:00Z"
}
```

**2. Metrics Callback**
```http
POST http://control-plane:8080/v1/internal/locust/metrics
Content-Type: application/json

{
  "test_run_id": "run-456",
  "metrics": {
    "timestamp": "2024-01-01T10:05:00Z",
    "totalRps": 1250.5,
    "currentUsers": 200,
    ...
  }
}
```

**3. Test Stop Callback**
```http
POST http://control-plane:8080/v1/internal/locust/test-stop
Content-Type: application/json

{
  "test_run_id": "run-456",
  "final_metrics": {...},
  "timestamp": "2024-01-01T11:00:00Z"
}
```

---

## Concurrency & Threading Model

```
┌─────────────────────────────────────────────────────────┐
│                    Control Plane                         │
│                                                           │
│  Main Goroutine                                          │
│  ├─ HTTP Server (Gorilla Mux)                           │
│  │  ├─ Handler Goroutines (per request)                 │
│  │  └─ Middleware (Auth, Logging)                       │
│  │                                                        │
│  Background Goroutines                                   │
│  ├─ Orchestrator.runMetricsPoller()                     │
│  │  └─ Ticker (every N seconds)                         │
│  │     ├─ Query active runs                             │
│  │     ├─ Poll Locust for each                          │
│  │     ├─ Update stores                                 │
│  │     └─ Check duration limits                         │
│  │                                                        │
│  Store Layer (Thread-Safe)                              │
│  ├─ sync.RWMutex (In-Memory)                            │
│  └─ MongoDB Connection Pool                             │
│                                                           │
└─────────────────────────────────────────────────────────┘
```

**Thread Safety:**
- In-memory stores use `sync.RWMutex` for concurrent access
- MongoDB driver handles connection pooling internally
- Each HTTP request handled in separate goroutine
- Background metrics poller runs independently

---

## Configuration Flow

```
config/config.yaml
       │
       ▼
┌──────────────────────────────┐
│   Config Loading             │
│   (On Startup)               │
└───────────┬──────────────────┘
            │
            ▼
┌──────────────────────────────┐
│   Parsed Config              │
│   • Server (host, port)      │
│   • MongoDB (URI, database)  │
│   • LocustClusters[]         │
│   │  ├─ id                   │
│   │  ├─ baseURL              │
│   │  ├─ authToken            │
│   │  ├─ tenantId             │
│   │  └─ envId                │
│   • Orchestrator             │
│      └─ metricsPollInterval  │
└───────────┬──────────────────┘
            │
            ▼
┌──────────────────────────────┐
│   Component Initialization   │
│   • MongoDB Clients          │
│   • Store Instances          │
│   • Locust Clients (by ID)   │
│   • Orchestrator             │
│   • HTTP Handlers            │
└──────────────────────────────┘
```

**Cluster Resolution:**
```
LoadTestRun (tenantId + envId)
         │
         ▼
Config.GetLocustCluster(tenantId, envId)
         │
         ▼
Returns: LocustCluster{id, baseURL, authToken}
         │
         ▼
Orchestrator.getClient(clusterID)
         │
         ▼
Returns: LocustClient (HTTP client)
```

---

## Error Handling & Resilience

**1. Store Failures:**
- MongoDB operations wrapped with context timeouts
- Errors logged and returned to client
- Connection pooling handles reconnects

**2. Locust Communication Failures:**
- HTTP client timeouts (30s for swarm/stop, 10s for stats)
- On failure, test status updated to "Failed"
- Metrics polling continues for other active tests

**3. Duration Enforcement:**
- Background poller checks elapsed time
- Auto-stops tests exceeding max duration
- Prevents runaway load tests

**4. Graceful Shutdown:**
- SIGINT/SIGTERM triggers shutdown
- Orchestrator stopped (metrics poller exits)
- HTTP server graceful shutdown (30s timeout)
- MongoDB connections closed

---

## Summary

The Control Plane acts as the **orchestration layer** between:
1. **Users/API Clients** - Manage test definitions and executions
2. **Locust Clusters** - Execute actual load tests
3. **MongoDB** - Persist test data and metrics

**Key Design Principles:**
- **Separation of Concerns**: LoadTest (definition) vs LoadTestRun (execution)
- **Audit Trail**: Full tracking of who created/updated what and when
- **Scalability**: Time-series metrics storage for long-running tests
- **Flexibility**: Runtime parameter overrides
- **Resilience**: Timeout handling, graceful shutdowns, duration limits

**Data Flows:**
- **Pull Mode**: Orchestrator polls Locust for metrics
- **Push Mode**: Locust sends callbacks to Control Plane
- **Hybrid**: Both modes supported simultaneously
