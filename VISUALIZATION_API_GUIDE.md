# Visualization API Guide

This guide describes the optimized visualization APIs for plotting analytics graphs and displaying test run details, similar to Locust's UI.

## Overview

Three new APIs have been added to support the UI shown in your design:
1. **GET /v1/runs/{id}/graph** - Minimal graph data for plotting Users, RPS, and Errors over time
2. **GET /v1/runs/{id}/summary** - The 4 key metrics cards (Total Requests, RPS, Error Rate, Avg Response Time)
3. **GET /v1/runs/{id}/requests** - Recent endpoint statistics formatted as a request log

---

## 1. Run Graph API

**Endpoint**: `GET /v1/runs/{runId}/graph`

Returns minimal timeseries data optimized for plotting the main chart with three lines:
- Users (active users over time)
- Requests per Second (RPS)
- Errors per Second

### Query Parameters
- `from` (optional) - Start time in RFC3339 format (e.g., `2025-01-01T10:00:00Z`)
- `to` (optional) - End time in RFC3339 format

### Response Schema
```json
{
  "runId": "string",
  "runName": "string",
  "status": "Running|Finished|Failed",
  "startedAt": "2025-12-22T12:00:00Z",
  "dataPoints": [
    {
      "timestamp": 1703232000000,      // Unix milliseconds
      "users": 100,                     // Current active users
      "requestsPerSec": 98.4,          // Requests per second
      "errorsPerSec": 1.2,             // Errors per second
      "avgResponseTime": 1.11          // Average response time in seconds
    }
  ]
}
```

### Example Request
```bash
curl -X GET "http://localhost:8080/v1/runs/abc123/graph?from=2025-12-22T10:00:00Z&to=2025-12-22T12:00:00Z" \
  -H "Content-Type: application/json"
```

### Example Response
```json
{
  "runId": "abc123",
  "runName": "API Stress Test",
  "status": "Running",
  "startedAt": "2025-12-22T10:30:00Z",
  "dataPoints": [
    {
      "timestamp": 1703239800000,
      "users": 50,
      "requestsPerSec": 45.2,
      "errorsPerSec": 0.5,
      "avgResponseTime": 0.85
    },
    {
      "timestamp": 1703239810000,
      "users": 75,
      "requestsPerSec": 72.8,
      "errorsPerSec": 0.9,
      "avgResponseTime": 1.02
    },
    {
      "timestamp": 1703239820000,
      "users": 100,
      "requestsPerSec": 98.4,
      "errorsPerSec": 1.2,
      "avgResponseTime": 1.11
    }
  ]
}
```

### Frontend Usage
```javascript
// Fetch graph data
const response = await fetch(`/v1/runs/${runId}/graph`);
const data = await response.json();

// Plot using your charting library (e.g., Chart.js, Recharts, D3)
const chartData = data.dataPoints.map(point => ({
  time: new Date(point.timestamp),
  users: point.users,
  rps: point.requestsPerSec,
  errors: point.errorsPerSec
}));
```

---

## 2. Run Summary API

**Endpoint**: `GET /v1/runs/{runId}/summary`

Returns the 4 key metrics displayed in the summary cards at the top:
- **Total Requests** - Total number of requests made
- **Requests per Second** - Average RPS
- **Error Rate** - Percentage of failed requests
- **Avg Response Time** - Average response time in seconds

### Response Schema
```json
{
  "runId": "string",
  "runName": "string",
  "status": "Running|Finished|Failed",
  "startedAt": "2025-12-22T12:00:00Z",
  "finishedAt": "2025-12-22T12:10:00Z",  // Only if finished
  "duration": "10m15s",
  "totalRequests": 1086,
  "requestsPerSec": 98.4,
  "errorRate": 1.24,                      // Percentage
  "avgResponseTime": 1.11,                // Seconds
  "targetUsers": 100,
  "spawnRate": 10.0,
  "durationSeconds": 600                  // Optional
}
```

### Example Request
```bash
curl -X GET "http://localhost:8080/v1/runs/abc123/summary" \
  -H "Content-Type: application/json"
```

### Example Response
```json
{
  "runId": "abc123",
  "runName": "API Stress Test",
  "status": "Running",
  "startedAt": "2025-12-22T10:30:00Z",
  "finishedAt": "",
  "duration": "5m23s",
  "totalRequests": 1086,
  "requestsPerSec": 98.4,
  "errorRate": 1.24,
  "avgResponseTime": 1.11,
  "targetUsers": 100,
  "spawnRate": 10.0,
  "durationSeconds": 600
}
```

### Frontend Usage
```javascript
// Fetch summary data
const response = await fetch(`/v1/runs/${runId}/summary`);
const data = await response.json();

// Display in cards
console.log(`Total Requests: ${data.totalRequests.toLocaleString()}`);
console.log(`RPS: ${data.requestsPerSec.toFixed(1)} req/s`);
console.log(`Error Rate: ${data.errorRate.toFixed(2)}%`);
console.log(`Avg Response Time: ${data.avgResponseTime.toFixed(2)} s`);
```

---

## 3. Live Request Log API

**Endpoint**: `GET /v1/runs/{runId}/requests`

Returns recent endpoint statistics formatted like a request log. This provides aggregated endpoint data rather than individual requests to avoid storing massive amounts of data.

### Query Parameters
- `limit` (optional) - Number of entries to return (default: 100, max: 500)
- `from` (optional) - Start time in RFC3339 format
- `to` (optional) - End time in RFC3339 format

### Response Schema
```json
{
  "runId": "string",
  "requests": [
    {
      "timestamp": 1703232000000,      // Unix milliseconds
      "requestType": "GET",            // HTTP method
      "responseTime": 618.4,           // Response time in milliseconds
      "url": "https://api.example.com/endpoint",
      "responseLength": 0,             // Not tracked in aggregated stats
      "success": true
    }
  ],
  "total": 150,                        // Total entries returned
  "limit": 100                         // Limit applied
}
```

### Example Request
```bash
curl -X GET "http://localhost:8080/v1/runs/abc123/requests?limit=50" \
  -H "Content-Type: application/json"
```

### Example Response
```json
{
  "runId": "abc123",
  "requests": [
    {
      "timestamp": 1703239825000,
      "requestType": "GET",
      "responseTime": 618.4,
      "url": "https://api.example.com",
      "responseLength": 0,
      "success": true
    },
    {
      "timestamp": 1703239825000,
      "requestType": "GET",
      "responseTime": 618.4,
      "url": "https://api.example.com",
      "responseLength": 0,
      "success": true
    },
    {
      "timestamp": 1703239825000,
      "requestType": "GET",
      "responseTime": 618.4,
      "url": "https://api.example.com",
      "responseLength": 0,
      "success": false
    }
  ],
  "total": 3,
  "limit": 50
}
```

### Frontend Usage
```javascript
// Fetch request log
const response = await fetch(`/v1/runs/${runId}/requests?limit=100`);
const data = await response.json();

// Display in table
data.requests.forEach(req => {
  console.log(`${new Date(req.timestamp).toLocaleString()} | ${req.requestType} | ${req.responseTime}ms | ${req.url} | ${req.success ? 'SUCCESS' : 'FAILED'}`);
});
```

---

## Existing Comprehensive APIs

These APIs provide more detailed data and are useful for detailed analysis:

### 4. Timeseries Chart API
**Endpoint**: `GET /v1/runs/{runId}/chart`

Returns complete timeseries data with all metrics including P50, P95, P99 latencies.

### 5. Aggregated Stats API
**Endpoint**: `GET /v1/runs/{runId}/stats`

Returns comprehensive aggregated statistics including:
- Timeseries data
- Per-endpoint statistics
- Aggregated summary

### 6. Scatter Plot API
**Endpoint**: `GET /v1/runs/{runId}/scatter`

Returns scatter plot data for response time distribution across endpoints.

---

## API Architecture

### Data Flow
```
Locust → Orchestrator → MetricsStore (MongoDB) → Visualization APIs → Frontend
```

### Key Design Decisions

1. **Minimal Data Transfer**: New APIs return only the essential data needed for UI rendering
2. **Unix Milliseconds**: Timestamps are returned as Unix milliseconds for easy JavaScript Date conversion
3. **Aggregated Data**: Request log shows aggregated endpoint stats, not individual requests (to save storage)
4. **Percentage Error Rate**: Error rate is returned as a percentage (0-100) for easy display
5. **Time in Seconds**: Response times are in seconds for the summary, milliseconds for detailed logs

### Performance Considerations

- **Caching**: Consider caching summary data for completed runs
- **Polling**: For live runs, poll the graph API every 5-10 seconds
- **Time Ranges**: Use `from` and `to` parameters to limit data returned
- **Limits**: Request log API limits to 500 entries max to prevent large payloads

---

## Complete Example: Building a Dashboard

```javascript
// React component example
import React, { useState, useEffect } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend } from 'recharts';

function RunDashboard({ runId }) {
  const [summary, setSummary] = useState(null);
  const [graphData, setGraphData] = useState([]);
  const [requests, setRequests] = useState([]);

  useEffect(() => {
    // Initial fetch
    fetchData();
    
    // Poll every 10 seconds for live updates
    const interval = setInterval(fetchData, 10000);
    return () => clearInterval(interval);
  }, [runId]);

  const fetchData = async () => {
    // Fetch summary
    const summaryRes = await fetch(`/v1/runs/${runId}/summary`);
    setSummary(await summaryRes.json());

    // Fetch graph data
    const graphRes = await fetch(`/v1/runs/${runId}/graph`);
    const graphJson = await graphRes.json();
    setGraphData(graphJson.dataPoints.map(p => ({
      time: new Date(p.timestamp).toLocaleTimeString(),
      users: p.users,
      rps: p.requestsPerSec,
      errors: p.errorsPerSec
    })));

    // Fetch recent requests
    const reqRes = await fetch(`/v1/runs/${runId}/requests?limit=50`);
    setRequests((await reqRes.json()).requests);
  };

  return (
    <div>
      {/* Summary Cards */}
      <div className="metrics-cards">
        <div className="card">
          <h3>Total Requests</h3>
          <p>{summary?.totalRequests.toLocaleString()}</p>
        </div>
        <div className="card">
          <h3>Requests per Second</h3>
          <p>{summary?.requestsPerSec.toFixed(1)} req/s</p>
        </div>
        <div className="card">
          <h3>Error Rate</h3>
          <p>{summary?.errorRate.toFixed(2)}%</p>
        </div>
        <div className="card">
          <h3>Avg Response Time</h3>
          <p>{summary?.avgResponseTime.toFixed(2)} s</p>
        </div>
      </div>

      {/* Graph */}
      <LineChart width={800} height={400} data={graphData}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="time" />
        <YAxis />
        <Tooltip />
        <Legend />
        <Line type="monotone" dataKey="users" stroke="#8884d8" name="Users" />
        <Line type="monotone" dataKey="rps" stroke="#82ca9d" name="RPS" />
        <Line type="monotone" dataKey="errors" stroke="#ff7300" name="Errors/sec" />
      </LineChart>

      {/* Request Log */}
      <table>
        <thead>
          <tr>
            <th>Time</th>
            <th>Method</th>
            <th>Response Time</th>
            <th>URL</th>
            <th>Success</th>
          </tr>
        </thead>
        <tbody>
          {requests.map((req, i) => (
            <tr key={i}>
              <td>{new Date(req.timestamp).toLocaleString()}</td>
              <td>{req.requestType}</td>
              <td>{req.responseTime.toFixed(1)}ms</td>
              <td>{req.url}</td>
              <td>{req.success ? '✅' : '❌'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

---

## Testing the APIs

```bash
# 1. Create a load test
curl -X POST http://localhost:8080/v1/load-tests \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API Stress Test",
    "accountId": "acc123",
    "orgId": "org456",
    "projectId": "proj789",
    "targetURL": "https://api.example.com",
    "defaultUsers": 100,
    "defaultSpawnRate": 10,
    "createdBy": "test@example.com"
  }'

# 2. Start a test run (returns runId)
curl -X POST http://localhost:8080/v1/load-tests/{testId}/runs \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Run #1",
    "targetUsers": 100,
    "spawnRate": 10,
    "durationSeconds": 600,
    "createdBy": "test@example.com"
  }'

# 3. Get run summary
curl http://localhost:8080/v1/runs/{runId}/summary

# 4. Get graph data
curl http://localhost:8080/v1/runs/{runId}/graph

# 5. Get request log
curl "http://localhost:8080/v1/runs/{runId}/requests?limit=50"
```

---

## Notes

- **Real-time Updates**: Poll the graph and summary APIs every 5-10 seconds for live updates during test execution
- **Error Handling**: All APIs return appropriate HTTP status codes (404 for not found, 500 for server errors)
- **Timestamps**: All timestamps are Unix milliseconds; convert to Date objects in JavaScript: `new Date(timestamp)`
- **Completed Runs**: Data is still available after a run completes for historical analysis
- **Time Ranges**: Use `from` and `to` query parameters to fetch specific time windows for large datasets
