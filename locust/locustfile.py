"""
Locust load testing file with Control Plane integration hooks.

This locustfile includes:
- Event hooks to notify the control plane when tests start/stop
- Background greenlet to periodically push metrics to the control plane
- Example HTTP load testing scenarios

Environment variables required:
- CONTROL_PLANE_URL: Base URL of the control plane API (e.g., http://control-plane:8080)
- CONTROL_PLANE_TOKEN: Shared secret for authenticating callbacks (X-Locust-Token header)
- RUN_ID: The test run ID from the control plane
- TENANT_ID: (Optional) Tenant identifier
- ENV_ID: (Optional) Environment identifier
- TARGET_HOST: The target application to load test
"""

import os
import time
import json
import logging
from typing import Optional

import gevent
import requests
from locust import HttpUser, task, between, events
from locust.env import Environment

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Control plane configuration from environment variables
CONTROL_PLANE_URL = os.getenv("CONTROL_PLANE_URL", "")
CONTROL_PLANE_TOKEN = os.getenv("CONTROL_PLANE_TOKEN", "")
RUN_ID = os.getenv("RUN_ID", "")
TENANT_ID = os.getenv("TENANT_ID", "")
ENV_ID = os.getenv("ENV_ID", "")

# Metrics push interval in seconds
METRICS_PUSH_INTERVAL = int(os.getenv("METRICS_PUSH_INTERVAL", "10"))

# Global greenlet reference for metrics pusher
_metrics_greenlet: Optional[gevent.Greenlet] = None


def _control_plane_headers():
    """Returns headers for control plane API calls."""
    return {
        "X-Locust-Token": CONTROL_PLANE_TOKEN,
        "Content-Type": "application/json",
    }


def _is_control_plane_enabled():
    """Check if control plane integration is enabled."""
    return bool(CONTROL_PLANE_URL and CONTROL_PLANE_TOKEN and RUN_ID)


@events.test_start.add_listener
def on_test_start(environment: Environment, **kwargs):
    """
    Event handler called when a load test starts.
    Notifies the control plane that the test has started.
    """
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, skipping test_start callback")
        return
    
    logger.info(f"Test started, notifying control plane (RUN_ID={RUN_ID})")
    
    try:
        payload = {
            "runId": RUN_ID,
            "tenantId": TENANT_ID,
            "envId": ENV_ID,
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
    """
    Event handler called when a load test stops.
    Notifies the control plane with final metrics.
    """
    global _metrics_greenlet
    
    # Stop the metrics pusher greenlet
    if _metrics_greenlet is not None:
        logger.info("Stopping metrics pusher greenlet")
        gevent.kill(_metrics_greenlet)
        _metrics_greenlet = None
    
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, skipping test_stop callback")
        return
    
    logger.info(f"Test stopped, notifying control plane with final metrics (RUN_ID={RUN_ID})")
    
    try:
        # Collect final metrics
        final_metrics = _collect_metrics(environment)
        
        payload = {
            "runId": RUN_ID,
            "tenantId": TENANT_ID,
            "envId": ENV_ID,
            "finalMetrics": final_metrics,
        }
        
        url = f"{CONTROL_PLANE_URL}/v1/internal/locust/test-stop"
        response = requests.post(
            url,
            json=payload,
            headers=_control_plane_headers(),
            timeout=10
        )
        response.raise_for_status()
        logger.info("Successfully notified control plane of test stop")
    
    except Exception as e:
        logger.error(f"Failed to notify control plane of test stop: {e}")


def _collect_metrics(environment: Environment) -> dict:
    """
    Collects current metrics from Locust environment.
    Returns a dictionary compatible with the control plane MetricSnapshot format.
    """
    stats = environment.stats
    
    # Calculate aggregate metrics
    total_rps = stats.total.current_rps if stats.total else 0
    total_requests = stats.total.num_requests if stats.total else 0
    total_failures = stats.total.num_failures if stats.total else 0
    error_rate = (total_failures / total_requests * 100) if total_requests > 0 else 0
    
    # Get response time percentiles
    p50 = stats.total.get_response_time_percentile(0.5) if stats.total else 0
    p95 = stats.total.get_response_time_percentile(0.95) if stats.total else 0
    p99 = stats.total.get_response_time_percentile(0.99) if stats.total else 0
    avg_response = stats.total.avg_response_time if stats.total else 0
    
    # Current user count
    current_users = environment.runner.user_count if environment.runner else 0
    
    # Per-request statistics
    request_stats = {}
    for name, stat in stats.entries.items():
        method, endpoint = name
        key = f"{method}_{endpoint}"
        request_stats[key] = {
            "method": method,
            "name": endpoint,
            "numRequests": stat.num_requests,
            "numFailures": stat.num_failures,
            "avgResponseTime": stat.avg_response_time,
            "minResponseTime": stat.min_response_time or 0,
            "maxResponseTime": stat.max_response_time or 0,
            "medianResponseTime": stat.median_response_time or 0,
            "requestsPerSec": stat.current_rps,
        }
    
    return {
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "totalRps": total_rps,
        "totalRequests": total_requests,
        "totalFailures": total_failures,
        "errorRate": error_rate,
        "avgResponseMs": avg_response,
        "p50ResponseMs": p50,
        "p95ResponseMs": p95,
        "p99ResponseMs": p99,
        "currentUsers": current_users,
        "requestStats": request_stats,
    }


def _metrics_pusher(environment: Environment):
    """
    Background task that periodically pushes metrics to the control plane.
    Runs in a greenlet for the duration of the test.
    """
    logger.info(f"Starting metrics pusher (interval: {METRICS_PUSH_INTERVAL}s)")
    
    while True:
        try:
            gevent.sleep(METRICS_PUSH_INTERVAL)
            
            if not _is_control_plane_enabled():
                continue
            
            # Collect and send metrics
            metrics = _collect_metrics(environment)
            
            payload = {
                "runId": RUN_ID,
                "metrics": metrics,
            }
            
            url = f"{CONTROL_PLANE_URL}/v1/internal/locust/metrics"
            response = requests.post(
                url,
                json=payload,
                headers=_control_plane_headers(),
                timeout=5
            )
            response.raise_for_status()
            
            logger.debug(f"Pushed metrics to control plane (RPS: {metrics['totalRps']:.2f})")
        
        except gevent.GreenletExit:
            logger.info("Metrics pusher greenlet killed")
            break
        except Exception as e:
            logger.error(f"Error pushing metrics to control plane: {e}")


@events.test_start.add_listener
def start_metrics_greenlet(environment: Environment, **kwargs):
    """
    Spawns a greenlet to periodically push metrics to the control plane.
    This greenlet runs for the entire duration of the test.
    """
    global _metrics_greenlet
    
    if not _is_control_plane_enabled():
        logger.warning("Control plane integration not configured, metrics pusher disabled")
        return
    
    _metrics_greenlet = gevent.spawn(_metrics_pusher, environment)
    logger.info("Metrics pusher greenlet started")


# ============================================================================
# Load Testing Scenarios
# ============================================================================

class ExampleUser(HttpUser):
    """
    Example load testing user with sample HTTP tasks.
    
    Replace these tasks with your actual application endpoints.
    The target host should be set via the --host CLI argument or TARGET_HOST env var.
    """
    
    # Wait time between tasks (random between 1-5 seconds)
    wait_time = between(1, 5)
    
    @task(3)
    def get_homepage(self):
        """Simulates a GET request to the homepage."""
        self.client.get("/", name="GET /")
    
    @task(2)
    def get_api_users(self):
        """Simulates a GET request to fetch users."""
        self.client.get("/api/users", name="GET /api/users")
    
    @task(1)
    def post_api_data(self):
        """Simulates a POST request to submit data."""
        payload = {
            "name": "Test User",
            "email": "test@example.com",
            "timestamp": time.time(),
        }
        self.client.post(
            "/api/data",
            json=payload,
            name="POST /api/data"
        )
    
    @task(1)
    def get_api_health(self):
        """Simulates a health check request."""
        with self.client.get("/health", catch_response=True, name="GET /health") as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Health check failed with status {response.status_code}")


# ============================================================================
# Advanced Example: Multiple User Types
# ============================================================================

class AdminUser(HttpUser):
    """
    Example of a different user type with admin-specific tasks.
    You can have multiple HttpUser classes to simulate different user behaviors.
    """
    wait_time = between(2, 6)
    weight = 1  # Lower weight means fewer of these users
    
    @task
    def get_admin_dashboard(self):
        """Admin-specific task."""
        self.client.get("/admin/dashboard", name="GET /admin/dashboard")
    
    @task
    def get_admin_reports(self):
        """Admin-specific task."""
        self.client.get("/admin/reports", name="GET /admin/reports")


class APIUser(HttpUser):
    """
    Example of an API-focused user.
    """
    wait_time = between(0.5, 2)
    weight = 3  # Higher weight means more of these users
    
    @task(5)
    def list_items(self):
        """Frequently accessed API endpoint."""
        self.client.get("/api/items", name="GET /api/items")
    
    @task(2)
    def get_item_details(self):
        """Get details for a specific item."""
        item_id = 12345  # In a real scenario, you'd vary this
        self.client.get(f"/api/items/{item_id}", name="GET /api/items/:id")
    
    @task(1)
    def create_item(self):
        """Create a new item."""
        payload = {"name": "New Item", "value": 100}
        self.client.post("/api/items", json=payload, name="POST /api/items")
