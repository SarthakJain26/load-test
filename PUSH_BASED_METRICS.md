# Push-Based Metrics Collection

## Overview

The Load Manager control plane now uses a **pure push-based metrics collection** model. Locust clusters push metrics to the control plane at regular intervals, replacing the previous polling-based approach.

## Architecture Change

### Before (Polling-Based)
```
Control Plane ──poll every 10s──► Locust Master (GetStats API)
```
**Problems:**
- Control plane needed to poll every running test
- Added load on Locust master
- Required maintaining cluster URLs for metrics fetching
- Less scalable with many concurrent tests

### After (Push-Based)
```
Locust Master ──push every 10s──► Control Plane (/v1/internal/locust/metrics)
```
**Benefits:**
- ✅ No polling overhead on Locust
- ✅ Simpler architecture - single direction data flow
- ✅ More scalable - Locust decides when to push
- ✅ Works better with dynamic/ephemeral Locust clusters
- ✅ Reduced complexity in control plane

---

## What Changed

### 1. Orchestrator Changes

**Removed:**
- `runMetricsPoller()` - Background goroutine that polled for metrics
- `pollMetrics()` - Method that fetched stats from Locust
- `pollInterval` field - No longer needed

**Enhanced:**
- `UpdateMetrics()` - Now stores metrics in time-series database (previously done by poller)

**Kept:**
- All callback handlers (`HandleTestStart`, `HandleTestStop`, `UpdateMetrics`)
- Metrics storage in MongoDB
- Test lifecycle management

### 2. Locust Integration Changes

**Added:**
- Duration monitoring greenlet - auto-stops test after configured duration
- `DURATION_SECONDS` environment variable support
- `_duration_monitor()` function - sleeps until duration elapses, then calls `runner.quit()`

**Enhanced:**
- Test start event now records start time
- Test stop event now cleans up duration monitor greenlet
- Better logging for duration-based auto-stop

**Kept:**
- Metrics pusher greenlet (unchanged)
- Test start/stop callbacks (unchanged)
- Metrics collection logic (unchanged)

### 3. Configuration Changes

**Deprecated:**
- `orchestrator.metricsPollIntervalSeconds` - No longer used, marked as deprecated

**Note:** Old config files will still work - the deprecated field is simply ignored.

---

## How Duration Management Works

### Previous Approach
- Control plane poller checked if test duration elapsed
- Poller called `StopTestRun()` when duration exceeded
- Required polling to be enabled

### New Approach
- Locust handles duration internally
- When test starts, duration monitor greenlet is spawned
- Greenlet sleeps for `DURATION_SECONDS`
- When duration elapses, calls `environment.runner.quit()`
- Test stops gracefully, sends stop callback to control plane

---

## Environment Variables for Locust

When starting a Locust cluster, configure these environment variables:

### Required
- `CONTROL_PLANE_URL` - Control plane base URL (e.g., `http://localhost:8080`)
- `CONTROL_PLANE_TOKEN` - Authentication token for callbacks
- `RUN_ID` - Test run ID from control plane

### Optional
- `DURATION_SECONDS` - Auto-stop test after N seconds (e.g., `300` for 5 minutes)
- `METRICS_PUSH_INTERVAL` - Metrics push interval in seconds (default: `10`)
- `TENANT_ID` - Tenant identifier (for multi-tenancy)
- `ENV_ID` - Environment identifier

### Example Docker Compose
```yaml
environment:
  - CONTROL_PLANE_URL=http://control-plane:8080
  - CONTROL_PLANE_TOKEN=my-secret-token
  - RUN_ID=${TEST_RUN_ID}
  - DURATION_SECONDS=600
  - METRICS_PUSH_INTERVAL=10
  - TENANT_ID=tenant-1
  - ENV_ID=production
```

### Example Manual Start
```bash
export CONTROL_PLANE_URL=http://localhost:8080
export CONTROL_PLANE_TOKEN=my-secret-token
export RUN_ID=test-run-abc123
export DURATION_SECONDS=300
export METRICS_PUSH_INTERVAL=10

locust -f locustfile.py --master --host https://api.example.com
```

---

## Metrics Flow

### 1. Test Start
```
1. User creates test run via API
2. Control plane calls Locust swarm API
3. Locust starts test
4. Locust sends test_start callback → Control plane
5. Control plane updates run status to "Running"
6. Locust spawns metrics pusher greenlet
7. Locust spawns duration monitor greenlet (if duration configured)
```

### 2. During Test (Every 10s)
```
1. Metrics pusher collects current stats from Locust
2. Metrics pusher sends POST /v1/internal/locust/metrics
3. Control plane receives metrics
4. Control plane stores in time-series DB (MongoDB)
5. Control plane updates run.LastMetrics
```

### 3. Test Stop
```
Option A: Manual stop
- User calls POST /v1/runs/{id}/stop
- Control plane calls Locust stop API
- Locust stops, sends test_stop callback
- Control plane marks run as Finished

Option B: Duration elapsed
- Duration monitor greenlet wakes up
- Calls environment.runner.quit()
- Locust stops gracefully
- Sends test_stop callback with final metrics
- Control plane marks run as Finished
```

---

## API Endpoints (Unchanged)

These internal endpoints are called by Locust:

### POST /v1/internal/locust/test-start
Notifies control plane that test has started.

**Request:**
```json
{
  "runId": "run-uuid-123",
  "tenantId": "tenant-1",
  "envId": "production"
}
```

### POST /v1/internal/locust/metrics
Pushes metrics to control plane (called every 10s by Locust).

**Request:**
```json
{
  "runId": "run-uuid-123",
  "metrics": {
    "timestamp": "2025-12-23T12:00:00Z",
    "totalRps": 250.5,
    "totalRequests": 75000,
    "totalFailures": 150,
    "errorRate": 0.2,
    "avgResponseMs": 45.3,
    "p50ResponseMs": 40.0,
    "p95ResponseMs": 95.5,
    "p99ResponseMs": 150.0,
    "currentUsers": 100,
    "requestStats": { ... }
  }
}
```

### POST /v1/internal/locust/test-stop
Notifies control plane that test has stopped.

**Request:**
```json
{
  "runId": "run-uuid-123",
  "tenantId": "tenant-1",
  "envId": "production",
  "finalMetrics": { ... }
}
```

---

## Migration Guide

### For Existing Deployments

**No action required!** The change is backward compatible:

1. ✅ Old config files still work (deprecated field is ignored)
2. ✅ Locust already had metrics pusher implemented
3. ✅ Control plane handlers already existed
4. ✅ Just needs redeployment - no data migration

### Recommended Steps

1. **Update control plane:**
   ```bash
   go build -o bin/controlplane cmd/controlplane/main.go
   # Deploy new binary
   ```

2. **Update Locust environment variables:**
   - Add `DURATION_SECONDS` if you want auto-stop behavior
   - Ensure `METRICS_PUSH_INTERVAL` is set (default: 10s)

3. **Remove deprecated config (optional):**
   ```yaml
   # Remove this from config.yaml (optional, not required)
   orchestrator:
     metricsPollIntervalSeconds: 10
   ```

4. **Restart services:**
   ```bash
   # Restart control plane
   systemctl restart load-manager-controlplane
   
   # Restart Locust clusters with updated env vars
   docker-compose up -d --force-recreate
   ```

---

## Troubleshooting

### Metrics not appearing in control plane

**Check Locust logs:**
```bash
docker logs locust-master | grep -i metrics
```

Look for:
- `Metrics pusher greenlet started`
- `Pushed metrics to control plane`

**Common issues:**
- `CONTROL_PLANE_URL` not set or incorrect
- `CONTROL_PLANE_TOKEN` mismatch
- Network connectivity between Locust and control plane
- Control plane not running

### Test not auto-stopping after duration

**Check if DURATION_SECONDS is set:**
```bash
docker exec locust-master env | grep DURATION
```

**Check Locust logs:**
```bash
docker logs locust-master | grep -i duration
```

Look for:
- `Duration monitor started: will stop test after N seconds`
- `Duration of N seconds has elapsed, stopping test`

**Common issues:**
- `DURATION_SECONDS` not set in environment
- Value not a valid integer
- Locust manually stopped before duration elapsed

### Control plane shows "Orchestrator started (push-based metrics mode)"

**This is normal!** The log message confirms the new push-based mode is active.

---

## Performance Considerations

### Metrics Push Interval

**Default: 10 seconds**

- **Lower (5s):** More granular metrics, higher load on control plane
- **Higher (30s):** Less granular metrics, lower load on control plane

Adjust based on your needs:
```bash
export METRICS_PUSH_INTERVAL=5   # More frequent
export METRICS_PUSH_INTERVAL=30  # Less frequent
```

### Concurrent Tests

The push-based model scales better with many concurrent tests:
- No polling overhead per test
- Control plane only processes metrics when pushed
- Locust controls push rate

**Recommendation:** With 100+ concurrent tests, consider:
- Increasing `METRICS_PUSH_INTERVAL` to 20-30s
- Scaling control plane horizontally (if needed)
- Monitoring MongoDB performance

---

## Testing the Implementation

### 1. Start Control Plane
```bash
go run cmd/controlplane/main.go -config config/config.yaml
```

Expected log:
```
Orchestrator started (push-based metrics mode)
```

### 2. Start Locust with Duration
```bash
export CONTROL_PLANE_URL=http://localhost:8080
export CONTROL_PLANE_TOKEN=my-secret-token
export RUN_ID=test-123
export DURATION_SECONDS=60
export METRICS_PUSH_INTERVAL=5

locust -f locust/locustfile.py --master --host http://example.com
```

### 3. Start a Test
```bash
curl -X POST http://localhost:8089/swarm \
  -d '{"user_count":10,"spawn_rate":1}'
```

### 4. Verify Metrics Pushing
```bash
# Check control plane logs
tail -f controlplane.log | grep "Updated metrics"

# Check Locust logs
docker logs locust-master | grep "Pushed metrics"
```

### 5. Verify Auto-Stop
After 60 seconds, the test should automatically stop.

Check control plane logs:
```
Test run test-123 finished (via callback)
```

---

## Code Changes Summary

### Files Modified
- `internal/service/orchestrator.go` - Removed polling logic, enhanced UpdateMetrics
- `internal/config/config.go` - Deprecated poll interval config
- `locust/locustfile.py` - Added duration monitor
- `locust/docker-compose.yml` - Added DURATION_SECONDS example

### Lines Removed
- ~90 lines of polling code from orchestrator
- Background poller goroutine

### Lines Added
- ~25 lines for duration monitoring in Locust
- ~10 lines for time-series storage in UpdateMetrics

**Net change:** Simpler codebase with ~55 fewer lines!

---

## FAQ

**Q: Can I still manually stop tests?**  
A: Yes! Call `POST /v1/runs/{id}/stop` anytime.

**Q: What if I don't set DURATION_SECONDS?**  
A: Test runs until manually stopped. No auto-stop behavior.

**Q: Does this work with Locust UI-started tests?**  
A: Yes! As long as environment variables are set correctly.

**Q: Can I change metrics push interval per test?**  
A: Currently no - it's set at Locust startup. Feature can be added if needed.

**Q: What happens if control plane is down during test?**  
A: Locust continues running, metrics pushes fail (logged as errors). Test completes normally.

**Q: Are old metrics lost if control plane is down?**  
A: Yes - metrics are not queued/buffered. When control plane returns, new metrics resume.

**Q: Can I use different push intervals for different clusters?**  
A: Yes! Each Locust cluster has its own `METRICS_PUSH_INTERVAL` env var.

---

## Additional Resources

- [README.md](README.md) - General setup and usage
- [SWAGGER_INTEGRATION.md](SWAGGER_INTEGRATION.md) - API documentation
- [VISUALIZATION_API_GUIDE.md](VISUALIZATION_API_GUIDE.md) - Metrics visualization
- [locust/locustfile.py](locust/locustfile.py) - Full Locust integration code

---

## Summary

✅ **Pure push-based metrics collection implemented**  
✅ **Duration handling moved to Locust**  
✅ **Simpler, more scalable architecture**  
✅ **Backward compatible - no breaking changes**  
✅ **Ready for production use**

**Questions or issues?** Check the troubleshooting section above or open an issue.
