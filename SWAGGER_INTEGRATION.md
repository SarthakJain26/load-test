# Swagger/OpenAPI Integration Guide

## Overview

The Load Manager API now includes comprehensive **Swagger/OpenAPI 2.0** documentation for all public endpoints. Internal Locust callback endpoints are excluded from the documentation.

---

## 📁 Generated Files

The Swagger documentation is available in multiple formats:

- **`docs/swagger.yaml`** - OpenAPI spec in YAML format (29 KB)
- **`docs/swagger.json`** - OpenAPI spec in JSON format (57 KB)
- **`docs/docs.go`** - Generated Go code for runtime serving (58 KB)

---

## 🌐 Accessing Swagger UI

Once the control plane server is running, access the interactive Swagger UI at:

```
http://localhost:8080/swagger/index.html
```

### Features:
- **Interactive API Testing** - Try out endpoints directly from the browser
- **Request/Response Examples** - See example payloads for all operations
- **Schema Definitions** - View all data models and their properties
- **Parameter Documentation** - Detailed info on all query params, path params, and request bodies

---

## 📚 API Documentation Structure

### Tags (Organized Groups)

1. **LoadTests** - Load test configuration management
   - Create, list, get, update, delete load tests
   
2. **Scripts** - Script revision management  
   - Update script (creates new revision)
   - Get latest script
   - List revision history
   - Get specific revision

3. **Runs** - Load test execution
   - Start new run
   - Get run details
   - List runs (with filters)
   - Stop running test

4. **Visualization** - Metrics and dashboards
   - Graph data for charts
   - Summary metrics
   - Request statistics
   - Detailed timeseries
   - Scatter plots
   - Aggregated stats

---

## 🔧 Regenerating Swagger Docs

If you modify API handlers or add new endpoints, regenerate the docs:

```bash
# From project root
~/go/bin/swag init -g cmd/controlplane/main.go -o docs --parseDependency --parseInternal
```

The tool will:
1. Parse all Swagger annotations in your code
2. Generate `docs/swagger.json`
3. Generate `docs/swagger.yaml`
4. Update `docs/docs.go`

---

## 📖 Swagger Annotations Reference

### General API Info (in main.go)

```go
// @title Load Manager API
// @version 1.0
// @description Load testing management platform
// @host localhost:8080
// @BasePath /v1
// @schemes http https
```

### Endpoint Example

```go
// CreateLoadTest godoc
// @Summary Create a new load test
// @Description Creates a new load test configuration with an initial script revision
// @Tags LoadTests
// @Accept json
// @Produce json
// @Param request body CreateLoadTestRequest true "Load test configuration"
// @Success 201 {object} LoadTestResponse "Load test created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 500 {object} ErrorResponse "Failed to create load test"
// @Router /load-tests [post]
func (h *Handler) CreateLoadTest(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### Common Annotations

| Annotation | Description | Example |
|------------|-------------|---------|
| `@Summary` | Short description | `@Summary Get load test by ID` |
| `@Description` | Detailed description | `@Description Retrieves a specific load test...` |
| `@Tags` | Group name | `@Tags LoadTests` |
| `@Accept` | Request content type | `@Accept json` |
| `@Produce` | Response content type | `@Produce json` |
| `@Param` | Parameter definition | `@Param id path string true "Load Test ID"` |
| `@Success` | Success response | `@Success 200 {object} LoadTestResponse` |
| `@Failure` | Error response | `@Failure 404 {object} ErrorResponse` |
| `@Router` | Endpoint path & method | `@Router /load-tests/{id} [get]` |

---

## 🚀 API Endpoints Summary

### LoadTests (5 endpoints)

```
POST   /v1/load-tests           - Create new load test
GET    /v1/load-tests           - List all load tests
GET    /v1/load-tests/{id}      - Get load test details
PUT    /v1/load-tests/{id}      - Update load test
DELETE /v1/load-tests/{id}      - Delete load test
```

### Scripts (4 endpoints)

```
PUT    /v1/load-tests/{id}/script                       - Update script (new revision)
GET    /v1/load-tests/{id}/script                       - Get latest script
GET    /v1/load-tests/{id}/script/revisions             - List revision history
GET    /v1/load-tests/{id}/script/revisions/{revisionId} - Get specific revision
```

### Runs (4 endpoints)

```
POST   /v1/load-tests/{id}/runs  - Start new test run
GET    /v1/load-tests/{id}/runs  - List runs for load test
GET    /v1/runs                  - List all runs
GET    /v1/runs/{id}             - Get run details
POST   /v1/runs/{id}/stop        - Stop running test
```

### Visualization (6 endpoints)

```
GET    /v1/runs/{id}/graph              - Graph data (RPS, response time)
GET    /v1/runs/{id}/summary            - Summary metrics (4 cards)
GET    /v1/runs/{id}/requests           - Request statistics log
GET    /v1/runs/{id}/metrics/timeseries - Detailed timeseries data
GET    /v1/runs/{id}/metrics/scatter    - Scatter plot data
GET    /v1/runs/{id}/metrics/aggregate  - Aggregated statistics
```

---

## 📦 Data Models

All request and response models are documented with their properties, types, and descriptions.

### Key Models:

**Requests:**
- `CreateLoadTestRequest`
- `UpdateLoadTestRequest`
- `CreateLoadTestRunRequest`
- `UpdateScriptRequest`

**Responses:**
- `LoadTestResponse`
- `LoadTestRunResponse`
- `ScriptRevisionResponse`
- `RunGraphResponse`
- `RunSummaryResponse`
- `ErrorResponse`
- `SuccessResponse`

**Visualization:**
- `TimeseriesChartResponse`
- `ScatterPlotResponse`
- `VisualizationSummaryResponse`
- `RequestLogEntry`

---

## 🧪 Testing with Swagger UI

### Example: Create a Load Test

1. Navigate to **http://localhost:8080/swagger/index.html**
2. Expand the **LoadTests** section
3. Click **POST /v1/load-tests**
4. Click **"Try it out"**
5. Edit the request body:

```json
{
  "name": "API Performance Test",
  "accountId": "acc123",
  "orgId": "org456",
  "projectId": "proj789",
  "locustClusterId": "cluster-1",
  "targetUrl": "https://api.example.com",
  "scriptContent": "ZnJvbSBsb2N1c3QgaW1wb3J0IEh0dHBVc2VyLCB0YXNrCgpjbGFzcyBRdWlja3N0YXJ0VXNlcihIdHRwVXNlcik6CiAgICBAdGFzawogICAgZGVmIGhlbGxvX3dvcmxkKHNlbGYpOgogICAgICAgIHNlbGYuY2xpZW50LmdldCgiLyIp",
  "defaultUsers": 100,
  "defaultSpawnRate": 10,
  "createdBy": "user@example.com"
}
```

6. Click **"Execute"**
7. View the response below (status code, headers, body)

---

## 🔒 Excluded Endpoints

The following **internal endpoints** are NOT included in Swagger documentation:

```
POST /v1/internal/locust/test-start        - Locust callback (test started)
POST /v1/internal/locust/test-stop         - Locust callback (test stopped)
POST /v1/internal/locust/metrics           - Locust callback (metrics update)
POST /v1/internal/locust/register-external - Register external test
```

These are used internally by Locust workers and should not be exposed in public API docs.

---

## 🎨 Customization

### Update API Information

Edit the annotations in `cmd/controlplane/main.go`:

```go
// @title Load Manager API
// @version 1.0
// @description Your custom description
// @contact.name API Support
// @contact.email support@example.com
// @license.name Apache 2.0
// @host your-domain.com
// @BasePath /v1
```

### Change Host/BasePath

For production deployment, update the `@host` annotation:

```go
// @host api.loadmanager.io
// @BasePath /v1
// @schemes https
```

Then regenerate:

```bash
~/go/bin/swag init -g cmd/controlplane/main.go -o docs --parseDependency --parseInternal
```

---

## 🐛 Troubleshooting

### Swagger UI not loading

**Issue:** Cannot access http://localhost:8080/swagger/index.html

**Solution:**
1. Ensure server is running: `./load-manager-cli -config config/config.yaml`
2. Check logs for "Route: [GET] /swagger/"
3. Verify import: `_ "Load-manager-cli/docs"` in main.go

### Documentation outdated

**Issue:** API changes not reflected in Swagger UI

**Solution:**
```bash
# Regenerate docs
~/go/bin/swag init -g cmd/controlplane/main.go -o docs --parseDependency --parseInternal

# Rebuild binary
go build -o load-manager-cli ./cmd/controlplane

# Restart server
```

### Missing model in docs

**Issue:** Response type shows as empty object

**Solution:**
1. Ensure the struct is exported (starts with capital letter)
2. Add JSON tags: `json:"fieldName"`
3. Use correct reference in annotation: `@Success 200 {object} YourType`
4. Regenerate docs

---

## 📊 Export OpenAPI Spec

### YAML Format

```bash
# Already available at:
cat docs/swagger.yaml
```

### JSON Format

```bash
# Already available at:
cat docs/swagger.json
```

### Import to Tools

Use these files with:
- **Postman** - Import → OpenAPI 2.0 → Select `swagger.json`
- **Insomnia** - Import → OpenAPI 2.0 → Select `swagger.json`
- **API Gateway** - Import spec for proxy configuration
- **Code Generators** - Generate client SDKs in various languages

---

## 🔗 Useful Links

- **Swaggo Documentation:** https://github.com/swaggo/swag
- **OpenAPI Specification:** https://swagger.io/specification/v2/
- **Swagger UI:** https://swagger.io/tools/swagger-ui/

---

## 📝 Summary

✅ **Swagger UI** integrated at `/swagger/index.html`  
✅ **19 documented endpoints** across 4 tag groups  
✅ **OpenAPI 2.0 spec** in YAML and JSON formats  
✅ **Internal endpoints excluded** from public docs  
✅ **Interactive testing** via Swagger UI  
✅ **Complete data models** with examples  
✅ **Easy regeneration** with swag CLI tool  

Access your API documentation at: **http://localhost:8080/swagger/index.html**
