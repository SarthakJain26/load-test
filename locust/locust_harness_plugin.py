"""
Locust Harness Plugin - Control Plane Integration

This plugin provides automatic integration with the Harness Control Plane.
Users should NOT modify this file - it's automatically injected into test runs.

Features:
- Automatic test start/stop notifications to control plane
- Real-time metrics pushing during test execution
- Duration-based auto-stop functionality
- Dynamic run context management via custom web endpoints

Environment Variables Required:
- CONTROL_PLANE_URL: URL of the control plane (e.g., http://localhost:8080)
- CONTROL_PLANE_TOKEN: Authentication token for control plane API
- METRICS_PUSH_INTERVAL: Interval in seconds for pushing metrics (default: 10)
"""

import os
import logging
import requests
import gevent
from typing import Optional
from locust import events
from locust.env import Environment

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Control plane configuration from environment variables
CONTROL_PLANE_URL = os.getenv("CONTROL_PLANE_URL", "")
CONTROL_PLANE_TOKEN = os.getenv("CONTROL_PLANE_TOKEN", "")
METRICS_PUSH_INTERVAL = int(os.getenv("METRICS_PUSH_INTERVAL", "10"))

# Global state for current test run (set dynamically per test)
_run_context = {
    "run_id": os.getenv("RUN_ID", ""),
    "tenant_id": os.getenv("TENANT_ID", ""),
    "env_id": os.getenv("ENV_ID", ""),
    "duration_seconds": os.getenv("DURATION_SECONDS", ""),
}

# Global greenlet references
_metrics_greenlet: Optional[gevent.Greenlet] = None
_duration_monitor_greenlet: Optional[gevent.Greenlet] = None
_test_start_time: Optional[float] = None
_auto_stopped: bool = False


def _control_plane_headers():
    """Returns headers for control plane API calls."""
    return {
        "X-Locust-Token": CONTROL_PLANE_TOKEN,
        "Content-Type": "application/json",
    }


def _is_control_plane_enabled():
    """Check if control plane integration is configured."""
    return bool(CONTROL_PLANE_URL and CONTROL_PLANE_TOKEN)


@events.test_start.add_listener
def on_test_start(environment: Environment, **kwargs):
    """Event handler called when a load test starts."""
    global _test_start_time
    import time
    _test_start_time = time.time()
    
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, skipping test_start callback")
        return
    
    run_id = _run_context.get("run_id", "")
    logger.info(f"Test started, notifying control plane (RUN_ID={run_id})")
    
    try:
        payload = {
            "runId": run_id,
            "tenantId": _run_context.get("tenant_id", ""),
            "envId": _run_context.get("env_id", ""),
        }
        
        url = f"{CONTROL_PLANE_URL}/v1/internal/locust/test-start"
        response = requests.post(
            url,
            json=payload,
            headers=_control_plane_headers(),
            timeout=10
        )
        response.raise_for_status()
        logger.info("Successfully notified control plane of test start")
    
    except Exception as e:
        logger.error(f"Failed to notify control plane of test start: {e}")


@events.test_stop.add_listener
def on_test_stop(environment: Environment, **kwargs):
    """Event handler called when a load test stops."""
    global _metrics_greenlet, _duration_monitor_greenlet, _auto_stopped
    
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, skipping test_stop callback")
        if _metrics_greenlet is not None:
            gevent.kill(_metrics_greenlet)
            _metrics_greenlet = None
        if _duration_monitor_greenlet is not None:
            gevent.kill(_duration_monitor_greenlet)
            _duration_monitor_greenlet = None
        return
    
    run_id = _run_context.get("run_id", "")
    stop_reason = "auto" if _auto_stopped else "manual"
    logger.info(f"Test stopped ({stop_reason}), notifying control plane with final metrics (RUN_ID={run_id})")
    
    try:
        final_metrics = _collect_metrics(environment)
        
        payload = {
            "runId": run_id,
            "tenantId": _run_context.get("tenant_id", ""),
            "envId": _run_context.get("env_id", ""),
            "finalMetrics": final_metrics,
            "autoStopped": _auto_stopped,
        }
        
        url = f"{CONTROL_PLANE_URL}/v1/internal/locust/test-stop"
        logger.info(f"Sending test-stop request to {url}")
        
        response = requests.post(
            url,
            json=payload,
            headers=_control_plane_headers(),
            timeout=5
        )
        
        logger.info(f"Test-stop response status: {response.status_code}")
        
        if response.status_code != 200:
            logger.error(f"Test-stop failed with status {response.status_code}: {response.text}")
        else:
            logger.info("Successfully notified control plane of test stop")
        
        response.raise_for_status()
    
    except requests.exceptions.RequestException as e:
        logger.error(f"Failed to notify control plane of test stop (network error): {e}")
    except Exception as e:
        logger.error(f"Failed to notify control plane of test stop (unexpected error): {e}", exc_info=True)
    
    finally:
        if _metrics_greenlet is not None:
            logger.info("Stopping metrics pusher greenlet")
            gevent.kill(_metrics_greenlet)
            _metrics_greenlet = None
        
        if _duration_monitor_greenlet is not None:
            logger.info("Stopping duration monitor greenlet")
            gevent.kill(_duration_monitor_greenlet)
            _duration_monitor_greenlet = None


def _collect_metrics(environment: Environment) -> dict:
    """Collects current metrics from Locust environment."""
    stats = environment.stats
    
    total_rps = stats.total.current_rps if stats.total else 0
    total_requests = stats.total.num_requests if stats.total else 0
    total_failures = stats.total.num_failures if stats.total else 0
    
    current_users = environment.runner.user_count if environment.runner else 0
    
    error_rate = (total_failures / total_requests * 100) if total_requests > 0 else 0
    
    # Get percentiles - call separately for each percentile value
    p50 = 0.0
    p95 = 0.0
    p99 = 0.0
    if stats.total:
        try:
            # Locust's get_response_time_percentile expects a single float, not a list
            p50 = float(stats.total.get_response_time_percentile(0.50) or 0)
            p95 = float(stats.total.get_response_time_percentile(0.95) or 0)
            p99 = float(stats.total.get_response_time_percentile(0.99) or 0)
        except (TypeError, ValueError, AttributeError) as e:
            logger.warning(f"Failed to get percentiles: {e}")
    
    # Build request stats as a map (dict) with key as "method:name"
    request_stats_map = {}
    avg_response_time = 0.0
    
    for stat in stats.entries.values():
        if stat.name != "Aggregated":
            key = f"{stat.method}:{stat.name}"
            request_stats_map[key] = {
                "method": stat.method,
                "name": stat.name,
                "numRequests": int(stat.num_requests),
                "numFailures": int(stat.num_failures),
                "avgResponseTime": float(stat.avg_response_time) if stat.avg_response_time else 0.0,
                "minResponseTime": float(stat.min_response_time) if stat.min_response_time else 0.0,
                "maxResponseTime": float(stat.max_response_time) if stat.max_response_time else 0.0,
                "medianResponseTime": float(stat.median_response_time) if hasattr(stat, 'median_response_time') and stat.median_response_time else 0.0,
                "requestsPerSec": float(stat.current_rps) if hasattr(stat, 'current_rps') and stat.current_rps else 0.0,
            }
    
    # Get average response time from total stats
    if stats.total:
        avg_response_time = float(stats.total.avg_response_time) if stats.total.avg_response_time else 0.0
    
    # Get timestamp as ISO string
    from datetime import datetime, timezone
    timestamp_str = datetime.now(timezone.utc).isoformat().replace('+00:00', 'Z')
    
    return {
        "timestamp": timestamp_str,  # String in ISO format
        "totalRps": float(total_rps) if total_rps else 0.0,
        "totalRequests": int(total_requests) if total_requests else 0,
        "totalFailures": int(total_failures) if total_failures else 0,
        "currentUsers": int(current_users) if current_users else 0,
        "errorRate": float(error_rate) if error_rate else 0.0,
        "avgResponseMs": avg_response_time,  # Added missing field
        "p50ResponseMs": float(p50) if p50 else 0.0,
        "p95ResponseMs": float(p95) if p95 else 0.0,
        "p99ResponseMs": float(p99) if p99 else 0.0,
        "requestStats": request_stats_map,  # Map/dict, not array
    }


def _metrics_pusher(environment: Environment):
    """Background task that periodically pushes metrics to control plane."""
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, metrics pusher disabled")
        return
    
    run_id = _run_context.get("run_id", "")
    logger.info(f"Metrics pusher started for run {run_id}, pushing every {METRICS_PUSH_INTERVAL} seconds")
    
    while True:
        try:
            gevent.sleep(METRICS_PUSH_INTERVAL)
            
            logger.info(f"Collecting metrics for run {run_id}...")
            metrics = _collect_metrics(environment)
            
            payload = {
                "runId": run_id,
                "metrics": metrics,
            }
            
            url = f"{CONTROL_PLANE_URL}/v1/internal/locust/metrics"
            logger.info(f"Pushing metrics to {url} (RPS: {metrics['totalRps']:.2f}, Requests: {metrics['totalRequests']}, Users: {metrics['currentUsers']})")
            
            response = requests.post(
                url,
                json=payload,
                headers=_control_plane_headers(),
                timeout=5
            )
            response.raise_for_status()
            
            logger.info(f"✓ Metrics pushed successfully (Status: {response.status_code})")
        
        except gevent.GreenletExit:
            logger.info("Metrics pusher greenlet killed")
            break
        except Exception as e:
            logger.error(f"Error pushing metrics to control plane: {e}", exc_info=True)


def _duration_monitor(environment: Environment):
    """Background task that monitors test duration and stops the test when duration elapses."""
    global _auto_stopped
    
    duration_str = _run_context.get("duration_seconds", "")
    if not duration_str:
        logger.info("No duration limit set, duration monitor disabled")
        return
    
    try:
        duration = int(duration_str)
        logger.info(f"Duration monitor started: will stop test after {duration} seconds")
        
        gevent.sleep(duration)
        
        _auto_stopped = True
        
        logger.info(f"Duration of {duration} seconds has elapsed, stopping test (auto-stop)")
        environment.runner.stop()
        
    except gevent.GreenletExit:
        logger.info("Duration monitor greenlet killed")
    except Exception as e:
        logger.error(f"Error in duration monitor: {e}")


@events.test_start.add_listener
def start_background_greenlets(environment: Environment, **kwargs):
    """Spawns background greenlets for metrics pushing and duration monitoring."""
    global _metrics_greenlet, _duration_monitor_greenlet
    
    logger.info(f"Attempting to start background greenlets. Control Plane URL: {CONTROL_PLANE_URL}, Token: {'set' if CONTROL_PLANE_TOKEN else 'not set'}, Run ID: {_run_context.get('run_id', 'NOT SET')}")
    
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, background greenlets disabled")
        return
    
    run_id = _run_context.get("run_id", "")
    if not run_id:
        logger.error("Cannot start background greenlets: run_id is not set in context")
        return
    
    _metrics_greenlet = gevent.spawn(_metrics_pusher, environment)
    logger.info(f"Metrics pusher greenlet spawned for run {run_id}")
    
    _duration_monitor_greenlet = gevent.spawn(_duration_monitor, environment)
    logger.info(f"Duration monitor greenlet spawned for run {run_id}")


@events.init.add_listener
def on_locust_init(environment: Environment, **kwargs):
    """
    Register custom web routes for control plane integration.
    This allows the control plane to set run context before starting tests.
    """
    if not isinstance(environment.web_ui, object):
        return
    
    from flask import request, jsonify
    
    @environment.web_ui.app.route("/controlplane/set-context", methods=["POST"])
    def set_run_context():
        """
        Custom endpoint to set run context before starting a test.
        Called by the control plane orchestrator before calling /swarm.
        """
        global _run_context
        
        try:
            data = request.get_json()
            
            _run_context["run_id"] = data.get("runId", "")
            _run_context["tenant_id"] = data.get("tenantId", "")
            _run_context["env_id"] = data.get("envId", "")
            _run_context["duration_seconds"] = str(data.get("durationSeconds", ""))
            
            logger.info(f"Run context updated: runId={_run_context['run_id']}, "
                       f"tenantId={_run_context['tenant_id']}, "
                       f"envId={_run_context['env_id']}, "
                       f"duration={_run_context['duration_seconds']}")
            
            return jsonify({
                "success": True,
                "message": "Run context set successfully",
                "context": _run_context
            }), 200
            
        except Exception as e:
            logger.error(f"Failed to set run context: {e}")
            return jsonify({
                "success": False,
                "error": str(e)
            }), 400
    
    @environment.web_ui.app.route("/controlplane/get-context", methods=["GET"])
    def get_run_context():
        """Get current run context (for debugging)."""
        return jsonify({
            "success": True,
            "context": _run_context
        }), 200
    
    logger.info("Harness Control Plane plugin initialized: custom endpoints registered")


logger.info("Locust Harness Plugin loaded successfully")
