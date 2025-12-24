# Load Manager - Documentation Index

## 📚 Complete Documentation Guide

This document provides an overview of all available documentation for the Load Manager Control Plane.

---

## Getting Started

### 🚀 [README.md](README.md)
**Main documentation and quick start guide**
- Architecture overview
- Features list
- Quick start instructions
- API usage examples
- Configuration reference
- Deployment guide

**Start here if you're new to Load Manager!**

---

## API Documentation

### 📖 [Swagger UI](http://localhost:8080/swagger/index.html)
**Interactive API documentation** (when server is running)
- Try out endpoints directly from your browser
- View request/response schemas
- See example payloads
- Test authentication

### 📘 [SWAGGER_INTEGRATION.md](SWAGGER_INTEGRATION.md)
**Complete Swagger integration guide**
- How to access Swagger UI
- API endpoint reference
- Regenerating documentation
- Exporting OpenAPI specs (YAML/JSON)
- Using specs with Postman/Insomnia

### ✅ [SWAGGER_COMPLETE.md](SWAGGER_COMPLETE.md)
**Implementation completion summary**
- What was delivered
- Available documentation formats
- Quick start guide
- Usage examples

---

## Feature Guides

### 📝 [SCRIPT_REVISION_GUIDE.md](SCRIPT_REVISION_GUIDE.md)
**Script version control and management**
- How script revisions work
- Base64 encoding scripts
- Creating and updating scripts
- Viewing revision history
- Revision workflow
- API examples for script management

**Read this to understand script versioning!**

### 📊 [VISUALIZATION_API_GUIDE.md](VISUALIZATION_API_GUIDE.md)
**Dashboard and metrics APIs**
- Optimized endpoints for dashboards
- Real-time graph data
- Summary metrics
- Request statistics
- Detailed timeseries data
- Scatter plots and aggregated stats

**Essential for building dashboards and monitoring UIs!**

---

## Technical Documentation

### 🗄️ [docs/MONGODB_SETUP.md](docs/MONGODB_SETUP.md)
**MongoDB setup and schema**
- Database collections
- Indexes and performance optimization
- Data models
- Setup instructions

### 🔧 [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)
**Technical implementation details**
- Script revision system implementation
- Base64 script storage
- Test execution logic
- Files modified/created
- Breaking changes

### 📡 [PUSH_BASED_METRICS.md](PUSH_BASED_METRICS.md)
**Push-based metrics collection guide**
- Architecture change from polling to push
- How duration management works
- Locust environment variables
- Migration guide
- Troubleshooting tips

---

## API Specification Files

### 📄 OpenAPI Specifications
Located in `docs/` directory:
- **swagger.yaml** - OpenAPI 2.0 spec in YAML format (29 KB)
- **swagger.json** - OpenAPI 2.0 spec in JSON format (57 KB)
- **docs.go** - Generated Go code for serving docs

**Use these files to:**
- Import into Postman or Insomnia
- Generate client SDKs
- Configure API gateways
- Share API contracts with teams

---

## Quick Reference

### 📋 API Endpoint Categories

**LoadTests** (5 endpoints)
- Create, list, get, update, delete load tests

**Scripts** (4 endpoints)
- Update script, get latest, list revisions, get specific revision

**Runs** (5 endpoints)
- Start run, list runs, get details, stop run

**Visualization** (6 endpoints)
- Graph data, summary, requests, timeseries, scatter, aggregated stats

### 🔑 Key Concepts

1. **Multi-tenant Hierarchy**: Account → Org → Project → Environment
2. **Script Revisions**: Every script edit creates a new immutable revision
3. **Base64 Storage**: Scripts stored as base64-encoded strings in MongoDB
4. **Revision Tracking**: Each test run references the exact script revision used
5. **Time-series Metrics**: All metrics stored with millisecond timestamps

---

## Common Tasks

### Create a Load Test
```bash
# 1. Encode your script
SCRIPT=$(base64 < locustfile.py)

# 2. Create load test
curl -X POST http://localhost:8080/v1/load-tests \
  -H "Authorization: Bearer token" \
  -d '{"name":"Test", "scriptContent":"'$SCRIPT'", ...}'
```

See: [README.md - API Usage](README.md#api-usage)

### Update a Script
```bash
# Creates new revision
curl -X PUT http://localhost:8080/v1/load-tests/{id}/script \
  -d '{"scriptContent":"base64...", "description":"What changed"}'
```

See: [SCRIPT_REVISION_GUIDE.md](SCRIPT_REVISION_GUIDE.md)

### Start a Test Run
```bash
curl -X POST http://localhost:8080/v1/load-tests/{id}/runs \
  -d '{"targetUsers":100, "spawnRate":10}'
```

See: [README.md - Start a Test Run](README.md#start-a-test-run)

### View Real-time Metrics
```bash
# Dashboard summary
curl http://localhost:8080/v1/runs/{id}/summary

# Graph data
curl http://localhost:8080/v1/runs/{id}/graph
```

See: [VISUALIZATION_API_GUIDE.md](VISUALIZATION_API_GUIDE.md)

---

## Documentation by Use Case

### 🎯 I want to...

**Get started quickly**
→ [README.md](README.md)

**Understand the API**
→ [Swagger UI](http://localhost:8080/swagger/index.html) + [SWAGGER_INTEGRATION.md](SWAGGER_INTEGRATION.md)

**Manage test scripts**
→ [SCRIPT_REVISION_GUIDE.md](SCRIPT_REVISION_GUIDE.md)

**Build a dashboard**
→ [VISUALIZATION_API_GUIDE.md](VISUALIZATION_API_GUIDE.md)

**Set up MongoDB**
→ [docs/MONGODB_SETUP.md](docs/MONGODB_SETUP.md)

**Understand implementation details**
→ [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)

**Export API specs**
→ [SWAGGER_COMPLETE.md](SWAGGER_COMPLETE.md)

**Integrate with Postman**
→ Import `docs/swagger.json`

**Generate client SDK**
→ Use `docs/swagger.yaml` with OpenAPI Generator

---

## Configuration Files

- **config/config.yaml** - Main configuration file
- **locust/locustfile.py** - Locust test script with event hooks
- **locust/docker-compose.yml** - Local Locust cluster setup

---

## Support & Resources

### Documentation Files Summary

| File | Purpose | When to Read |
|------|---------|--------------|
| README.md | Getting started | First time setup |
| SWAGGER_INTEGRATION.md | API documentation | Using the API |
| SCRIPT_REVISION_GUIDE.md | Script versioning | Managing scripts |
| VISUALIZATION_API_GUIDE.md | Dashboard APIs | Building UIs |
| IMPLEMENTATION_SUMMARY.md | Technical details | Understanding internals |
| docs/MONGODB_SETUP.md | Database setup | Setting up MongoDB |
| SWAGGER_COMPLETE.md | Implementation status | Reference |

### External Resources

- **Locust Documentation**: https://docs.locust.io/
- **MongoDB Documentation**: https://docs.mongodb.com/
- **OpenAPI Specification**: https://swagger.io/specification/
- **Go Documentation**: https://golang.org/doc/

---

## Version Information

- **API Version**: 1.0
- **OpenAPI Version**: 2.0
- **Go Version**: 1.22+
- **MongoDB Version**: 4.4+
- **Locust Version**: 2.x

---

## Quick Links

- 🌐 **Swagger UI**: http://localhost:8080/swagger/index.html
- 🏥 **Health Check**: http://localhost:8080/health
- 📊 **API Base URL**: http://localhost:8080/v1

---

**Need Help?**

1. Check the relevant guide above
2. Try the Swagger UI for interactive API testing
3. Review the examples in README.md
4. Check MongoDB setup if having database issues

**Happy Testing! 🚀**
