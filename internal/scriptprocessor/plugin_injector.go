package scriptprocessor

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const harnessPluginImport = `
# ============================================================================
# Harness Control Plane Plugin - AUTO-INJECTED
# This code is automatically added by the Harness platform.
# Users should NOT include this in their test files.
# ============================================================================
import sys
import os

# Import the Harness plugin for control plane integration
try:
    import locust_harness_plugin
except ImportError:
    # Plugin will be injected inline if not available as a separate file
    pass

`

// InjectHarnessPlugin takes a user's Locust script and injects the Harness plugin import
// This allows users to write clean test files without control plane integration code
func InjectHarnessPlugin(userScript string) string {
	// Check if the script already has the plugin imported
	if strings.Contains(userScript, "locust_harness_plugin") {
		// Already has plugin, return as-is
		return userScript
	}
	
	// Find the position after imports to inject the plugin
	lines := strings.Split(userScript, "\n")
	injectionPoint := 0
	
	// Find last import line
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			injectionPoint = i + 1
		}
		// Stop at first non-import, non-comment, non-empty line after imports
		if injectionPoint > 0 && trimmed != "" && !strings.HasPrefix(trimmed, "#") && 
		   !strings.HasPrefix(trimmed, "import ") && !strings.HasPrefix(trimmed, "from ") {
			break
		}
	}
	
	// If no imports found, inject at the beginning
	if injectionPoint == 0 {
		return harnessPluginImport + "\n" + userScript
	}
	
	// Inject after imports
	result := strings.Join(lines[:injectionPoint], "\n") + "\n" + 
	          harnessPluginImport + "\n" +
	          strings.Join(lines[injectionPoint:], "\n")
	
	return result
}

// InjectHarnessPluginBase64 takes a base64-encoded script, decodes it, injects the plugin, and re-encodes
func InjectHarnessPluginBase64(encodedScript string) (string, error) {
	// Decode the script
	decoded, err := base64.StdEncoding.DecodeString(encodedScript)
	if err != nil {
		return "", fmt.Errorf("failed to decode script: %w", err)
	}
	
	// Inject the plugin
	injected := InjectHarnessPlugin(string(decoded))
	
	// Re-encode
	reencoded := base64.StdEncoding.EncodeToString([]byte(injected))
	
	return reencoded, nil
}

// StripHarnessPlugin removes the auto-injected plugin import from a script
// This is used when returning scripts to users - they should see their clean original code
func StripHarnessPlugin(script string) string {
	lines := strings.Split(script, "\n")
	var cleanedLines []string
	inPluginSection := false
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Detect start of plugin injection section
		if strings.Contains(trimmed, "Harness Control Plane Plugin - AUTO-INJECTED") {
			inPluginSection = true
			continue
		}
		
		// Skip plugin section lines
		if inPluginSection {
			// End of plugin section when we hit user's imports or code
			if trimmed != "" && 
			   !strings.HasPrefix(trimmed, "#") && 
			   !strings.HasPrefix(trimmed, "import sys") &&
			   !strings.HasPrefix(trimmed, "import os") &&
			   !strings.HasPrefix(trimmed, "sys.path.insert") &&
			   !strings.HasPrefix(trimmed, "import locust_harness_plugin") &&
			   !strings.HasPrefix(trimmed, "try:") &&
			   !strings.HasPrefix(trimmed, "except ImportError:") &&
			   !strings.HasPrefix(trimmed, "pass") &&
			   !strings.Contains(trimmed, "============") {
				inPluginSection = false
				cleanedLines = append(cleanedLines, line)
			}
			continue
		}
		
		// Skip standalone plugin import line
		if strings.Contains(trimmed, "import locust_harness_plugin") {
			continue
		}
		
		cleanedLines = append(cleanedLines, line)
	}
	
	// Remove leading empty lines
	for len(cleanedLines) > 0 && strings.TrimSpace(cleanedLines[0]) == "" {
		cleanedLines = cleanedLines[1:]
	}
	
	return strings.Join(cleanedLines, "\n")
}

// StripHarnessPluginBase64 takes a base64-encoded script, decodes it, strips the plugin, and re-encodes
func StripHarnessPluginBase64(encodedScript string) (string, error) {
	// Decode the script
	decoded, err := base64.StdEncoding.DecodeString(encodedScript)
	if err != nil {
		return "", fmt.Errorf("failed to decode script: %w", err)
	}
	
	// Strip the plugin
	cleaned := StripHarnessPlugin(string(decoded))
	
	// Re-encode
	reencoded := base64.StdEncoding.EncodeToString([]byte(cleaned))
	
	return reencoded, nil
}

// GetHarnessPluginCode returns the full Harness plugin code
// This can be embedded in the script or deployed as a separate file
func GetHarnessPluginCode() string {
	return harnessPluginCode
}

// harnessPluginCode contains the full control plane integration logic
// This is embedded in the binary so it can be injected at runtime
const harnessPluginCode = `"""
Locust Harness Plugin - Control Plane Integration
This plugin is automatically injected by the Harness platform.
"""

import os
import logging
import requests
import gevent
from typing import Optional
from locust import events
from locust.env import Environment

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger(__name__)

CONTROL_PLANE_URL = os.getenv("CONTROL_PLANE_URL", "")
CONTROL_PLANE_TOKEN = os.getenv("CONTROL_PLANE_TOKEN", "")
METRICS_PUSH_INTERVAL = int(os.getenv("METRICS_PUSH_INTERVAL", "10"))

_run_context = {
    "run_id": os.getenv("RUN_ID", ""),
    "tenant_id": os.getenv("TENANT_ID", ""),
    "env_id": os.getenv("ENV_ID", ""),
    "duration_seconds": os.getenv("DURATION_SECONDS", ""),
}

_metrics_greenlet: Optional[gevent.Greenlet] = None
_duration_monitor_greenlet: Optional[gevent.Greenlet] = None
_test_start_time: Optional[float] = None
_auto_stopped: bool = False

def _control_plane_headers():
    return {"X-Locust-Token": CONTROL_PLANE_TOKEN, "Content-Type": "application/json"}

def _is_control_plane_enabled():
    return bool(CONTROL_PLANE_URL and CONTROL_PLANE_TOKEN)

@events.test_start.add_listener
def on_test_start(environment: Environment, **kwargs):
    global _test_start_time
    _test_start_time = environment.runner.start_time
    if not _is_control_plane_enabled():
        return
    run_id = _run_context.get("run_id", "")
    logger.info(f"Test started, notifying control plane (RUN_ID={run_id})")
    try:
        payload = {"runId": run_id, "tenantId": _run_context.get("tenant_id", ""), "envId": _run_context.get("env_id", "")}
        url = f"{CONTROL_PLANE_URL}/v1/internal/locust/test-start"
        response = requests.post(url, json=payload, headers=_control_plane_headers(), timeout=10)
        response.raise_for_status()
        logger.info("Successfully notified control plane of test start")
    except Exception as e:
        logger.error(f"Failed to notify control plane of test start: {e}")

@events.test_stop.add_listener
def on_test_stop(environment: Environment, **kwargs):
    global _metrics_greenlet, _duration_monitor_greenlet, _auto_stopped
    if not _is_control_plane_enabled():
        if _metrics_greenlet: gevent.kill(_metrics_greenlet)
        if _duration_monitor_greenlet: gevent.kill(_duration_monitor_greenlet)
        return
    run_id = _run_context.get("run_id", "")
    stop_reason = "auto" if _auto_stopped else "manual"
    logger.info(f"Test stopped ({stop_reason}), notifying control plane (RUN_ID={run_id})")
    try:
        final_metrics = _collect_metrics(environment)
        payload = {"runId": run_id, "tenantId": _run_context.get("tenant_id", ""), "envId": _run_context.get("env_id", ""), "finalMetrics": final_metrics, "autoStopped": _auto_stopped}
        url = f"{CONTROL_PLANE_URL}/v1/internal/locust/test-stop"
        response = requests.post(url, json=payload, headers=_control_plane_headers(), timeout=5)
        if response.status_code == 200:
            logger.info("Successfully notified control plane of test stop")
        response.raise_for_status()
    except Exception as e:
        logger.error(f"Failed to notify control plane of test stop: {e}")
    finally:
        if _metrics_greenlet: gevent.kill(_metrics_greenlet)
        if _duration_monitor_greenlet: gevent.kill(_duration_monitor_greenlet)

def _collect_metrics(environment: Environment) -> dict:
    stats = environment.stats
    total_rps = stats.total.current_rps if stats.total else 0
    total_requests = stats.total.num_requests if stats.total else 0
    total_failures = stats.total.num_failures if stats.total else 0
    current_users = environment.runner.user_count if environment.runner else 0
    error_rate = (total_failures / total_requests * 100) if total_requests > 0 else 0
    percentiles = stats.total.get_response_time_percentile([0.50, 0.95, 0.99]) if stats.total else {}
    request_stats = []
    for stat in stats.entries.values():
        if stat.name != "Aggregated":
            request_stats.append({"name": stat.name, "method": stat.method, "numRequests": stat.num_requests, "numFailures": stat.num_failures, "avgResponseTimeMs": stat.avg_response_time, "minResponseTimeMs": stat.min_response_time, "maxResponseTimeMs": stat.max_response_time})
    return {"timestamp": int(environment.runner.start_time * 1000) if environment.runner else 0, "totalRps": total_rps, "totalRequests": total_requests, "totalFailures": total_failures, "currentUsers": current_users, "errorRate": error_rate, "p50ResponseMs": percentiles.get(0.50, 0), "p95ResponseMs": percentiles.get(0.95, 0), "p99ResponseMs": percentiles.get(0.99, 0), "requestStats": request_stats}

def _metrics_pusher(environment: Environment):
    if not _is_control_plane_enabled(): return
    run_id = _run_context.get("run_id", "")
    while True:
        try:
            gevent.sleep(METRICS_PUSH_INTERVAL)
            metrics = _collect_metrics(environment)
            payload = {"runId": run_id, "metrics": metrics}
            url = f"{CONTROL_PLANE_URL}/v1/internal/locust/metrics"
            response = requests.post(url, json=payload, headers=_control_plane_headers(), timeout=5)
            response.raise_for_status()
        except gevent.GreenletExit:
            break
        except Exception as e:
            logger.error(f"Error pushing metrics: {e}")

def _duration_monitor(environment: Environment):
    global _auto_stopped
    duration_str = _run_context.get("duration_seconds", "")
    if not duration_str: return
    try:
        duration = int(duration_str)
        logger.info(f"Duration monitor: will stop after {duration} seconds")
        gevent.sleep(duration)
        _auto_stopped = True
        logger.info(f"Duration elapsed, stopping test (auto-stop)")
        environment.runner.stop()
    except gevent.GreenletExit:
        pass
    except Exception as e:
        logger.error(f"Error in duration monitor: {e}")

@events.test_start.add_listener
def start_background_greenlets(environment: Environment, **kwargs):
    global _metrics_greenlet, _duration_monitor_greenlet
    if not _is_control_plane_enabled(): return
    _metrics_greenlet = gevent.spawn(_metrics_pusher, environment)
    _duration_monitor_greenlet = gevent.spawn(_duration_monitor, environment)

@events.init.add_listener
def on_locust_init(environment: Environment, **kwargs):
    if not isinstance(environment.web_ui, object): return
    from flask import request, jsonify
    @environment.web_ui.app.route("/controlplane/set-context", methods=["POST"])
    def set_run_context():
        global _run_context
        try:
            data = request.get_json()
            _run_context["run_id"] = data.get("runId", "")
            _run_context["tenant_id"] = data.get("tenantId", "")
            _run_context["env_id"] = data.get("envId", "")
            _run_context["duration_seconds"] = str(data.get("durationSeconds", ""))
            logger.info(f"Run context updated: {_run_context}")
            return jsonify({"success": True, "context": _run_context}), 200
        except Exception as e:
            return jsonify({"success": False, "error": str(e)}), 400
    @environment.web_ui.app.route("/controlplane/get-context", methods=["GET"])
    def get_run_context():
        return jsonify({"success": True, "context": _run_context}), 200
    logger.info("Harness Control Plane plugin initialized")

logger.info("Locust Harness Plugin loaded")
`
