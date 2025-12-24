# Script Revision Management Guide

## Overview

The Load Manager now uses a **script revision system** instead of storing file paths. This enables:
1. **Base64 encoded script storage** - Scripts are stored directly in the database
2. **Version control** - Every script edit creates a new revision
3. **Revision tracking** - Each test run is associated with a specific script revision
4. **Script history** - View and retrieve any previous version of a script

---

## Key Changes

### ✅ What Changed

| Before | After |
|--------|-------|
| `LoadTest.locustfile` (string path) | `LoadTest.latestRevisionId` (reference to script revision) |
| Scripts stored as file paths | Scripts stored as base64 encoded content |
| No version control | Full revision history with `createdAt` and `createdBy` |
| No way to track which script version ran | `LoadTestRun.scriptRevisionId` tracks exact version used |

### 📦 New Database Collections

**`script_revisions`** collection:
```json
{
  "id": "rev-uuid",
  "loadTestId": "test-uuid",
  "revisionNumber": 1,
  "scriptContent": "aW1wb3J0IGxvY3VzdA...", // Base64 encoded
  "description": "Initial version",
  "createdAt": 1703232000000,
  "createdBy": "user@example.com"
}
```

**Indexes created:**
- `{id: 1}` (unique)
- `{loadTestId: 1, revisionNumber: -1}` (for getting latest)
- `{loadTestId: 1, createdAt: -1}` (for history listing)

---

## API Endpoints

### 1. Create Load Test with Script

**POST** `/v1/load-tests`

Now requires `scriptContent` (base64 encoded) instead of `locustfile`:

```json
{
  "name": "API Stress Test",
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

**Response:**
```json
{
  "id": "test-uuid",
  "name": "API Stress Test",
  "latestRevisionId": "rev-uuid-1",
  "createdAt": "2025-12-22T12:00:00Z",
  ...
}
```

### 2. Update Load Test Script

**PUT** `/v1/load-tests/{id}/script`

Creates a new revision when the script is edited:

```json
{
  "scriptContent": "ZnJvbSBsb2N1c3QgaW1wb3J0IEh0dHBVc2VyLCB0YXNrCgpjbGFzcyBRdWlja3N0YXJ0VXNlcihIdHRwVXNlcik6CiAgICBAdGFzawogICAgZGVmIGhlbGxvX3dvcmxkKHNlbGYpOgogICAgICAgIHNlbGYuY2xpZW50LmdldCgiLyIpCiAgICAgICAgc2VsZi5jbGllbnQuZ2V0KCIvYWJvdXQiKQ==",
  "description": "Added /about endpoint",
  "updatedBy": "user@example.com"
}
```

**Response:**
```json
{
  "id": "rev-uuid-2",
  "loadTestId": "test-uuid",
  "revisionNumber": 2,
  "scriptContent": "ZnJvbSBsb2N1c3QgaW1wb3J0...",
  "description": "Added /about endpoint",
  "createdAt": "2025-12-22T14:30:00Z",
  "createdBy": "user@example.com"
}
```

### 3. Get Latest Script

**GET** `/v1/load-tests/{id}/script`

Returns the latest script revision:

```json
{
  "id": "rev-uuid-2",
  "loadTestId": "test-uuid",
  "revisionNumber": 2,
  "scriptContent": "ZnJvbSBsb2N1c3QgaW1wb3J0...",
  "description": "Added /about endpoint",
  "createdAt": "2025-12-22T14:30:00Z",
  "createdBy": "user@example.com"
}
```

### 4. List Script Revisions

**GET** `/v1/load-tests/{id}/script/revisions?limit=10`

Returns revision history (most recent first):

```json
[
  {
    "id": "rev-uuid-2",
    "revisionNumber": 2,
    "description": "Added /about endpoint",
    "createdAt": "2025-12-22T14:30:00Z",
    "createdBy": "user@example.com"
  },
  {
    "id": "rev-uuid-1",
    "revisionNumber": 1,
    "description": "Initial version",
    "createdAt": "2025-12-22T12:00:00Z",
    "createdBy": "user@example.com"
  }
]
```

### 5. Get Specific Revision

**GET** `/v1/load-tests/{id}/script/revisions/{revisionId}`

Retrieve any past revision:

```json
{
  "id": "rev-uuid-1",
  "loadTestId": "test-uuid",
  "revisionNumber": 1,
  "scriptContent": "ZnJvbSBsb2N1c3QgaW1wb3J0...",
  "description": "Initial version",
  "createdAt": "2025-12-22T12:00:00Z",
  "createdBy": "user@example.com"
}
```

### 6. Create Test Run (Updated)

**POST** `/v1/load-tests/{id}/runs`

Now automatically uses the **latest script revision** when starting a test:

```json
{
  "targetUsers": 100,
  "spawnRate": 10,
  "durationSeconds": 600,
  "createdBy": "user@example.com"
}
```

**Response:**
```json
{
  "id": "run-uuid",
  "loadTestId": "test-uuid",
  "scriptRevisionId": "rev-uuid-2",  // ✨ Tracks which revision was used
  "status": "Running",
  ...
}
```

---

## Base64 Encoding

### Encoding a Python Script

**JavaScript/Node.js:**
```javascript
const fs = require('fs');
const script = fs.readFileSync('locustfile.py', 'utf-8');
const base64Script = Buffer.from(script).toString('base64');
console.log(base64Script);
```

**Python:**
```python
import base64

with open('locustfile.py', 'r') as f:
    script = f.read()
    base64_script = base64.b64encode(script.encode()).decode()
    print(base64_script)
```

**Bash:**
```bash
base64 < locustfile.py
```

### Decoding a Script

**JavaScript:**
```javascript
const script = Buffer.from(base64Script, 'base64').toString('utf-8');
```

**Python:**
```python
import base64
script = base64.b64decode(base64_script).decode('utf-8')
```

---

## Revision Workflow

### Creating a Load Test
```
1. User submits script (base64 encoded)
2. System creates LoadTest with ID
3. System creates ScriptRevision #1
4. LoadTest.latestRevisionId → revision #1
```

### Editing the Script
```
1. User submits updated script
2. System gets latest revision number (e.g., 2)
3. System creates ScriptRevision #3 (next number)
4. LoadTest.latestRevisionId → revision #3
5. LoadTest.updatedAt and updatedBy are updated
```

### Running a Test
```
1. User starts a test run
2. System fetches LoadTest.latestRevisionId
3. System retrieves that script revision
4. LoadTestRun.scriptRevisionId → revision ID
5. Script content sent to Locust cluster (decoded)
```

### Viewing History
```
User can:
- List all revisions (GET /script/revisions)
- View specific revision (GET /script/revisions/{revisionId})
- See which revision a run used (LoadTestRun.scriptRevisionId)
```

---

## Data Model Changes

### Domain Models

**`LoadTest`:**
```go
type LoadTest struct {
    ID               string
    Name             string
    LatestRevisionID string  // Changed from Locustfile
    ...
}
```

**`LoadTestRun`:**
```go
type LoadTestRun struct {
    ID               string
    LoadTestID       string
    ScriptRevisionID string  // New field - tracks which revision was used
    ...
}
```

**`ScriptRevision`:** (New)
```go
type ScriptRevision struct {
    ID             string
    LoadTestID     string
    RevisionNumber int
    ScriptContent  string  // Base64 encoded
    Description    string
    CreatedAt      int64
    CreatedBy      string
}
```

---

## Benefits

### ✅ Version Control
- Every script change is tracked
- Know exactly who changed what and when
- Can revert to any previous version

### ✅ Audit Trail
- Each test run references the exact script version used
- Reproduce any test with the exact same script
- Debug issues by comparing script versions

### ✅ No File System Dependencies
- Scripts stored in database
- No file path management needed
- Works across different environments

### ✅ Collaboration
- Multiple users can edit scripts
- Clear history of changes
- Description field for change notes

---

## Example Complete Workflow

```bash
# 1. Create a load test with initial script
SCRIPT_BASE64=$(base64 < locustfile.py)
curl -X POST http://localhost:8080/v1/load-tests \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"API Load Test\",
    \"accountId\": \"acc123\",
    \"orgId\": \"org456\",
    \"projectId\": \"proj789\",
    \"locustClusterId\": \"cluster-1\",
    \"targetUrl\": \"https://api.example.com\",
    \"scriptContent\": \"$SCRIPT_BASE64\",
    \"defaultUsers\": 100,
    \"createdBy\": \"alice@example.com\"
  }"

# Response: { "id": "test-abc", "latestRevisionId": "rev-001", ... }

# 2. Update the script (creates revision #2)
UPDATED_SCRIPT=$(base64 < locustfile_v2.py)
curl -X PUT http://localhost:8080/v1/load-tests/test-abc/script \
  -H "Content-Type: application/json" \
  -d "{
    \"scriptContent\": \"$UPDATED_SCRIPT\",
    \"description\": \"Added new endpoints\",
    \"updatedBy\": \"bob@example.com\"
  }"

# Response: { "id": "rev-002", "revisionNumber": 2, ... }

# 3. View revision history
curl http://localhost:8080/v1/load-tests/test-abc/script/revisions?limit=10

# 4. Start a test run (uses latest revision automatically)
curl -X POST http://localhost:8080/v1/load-tests/test-abc/runs \
  -H "Content-Type: application/json" \
  -d "{
    \"targetUsers\": 200,
    \"spawnRate\": 20,
    \"createdBy\": \"alice@example.com\"
  }"

# Response includes: "scriptRevisionId": "rev-002"

# 5. Get a specific old revision
curl http://localhost:8080/v1/load-tests/test-abc/script/revisions/rev-001
```

---

## Migration from Old System

If you have existing load tests with `locustfile` paths:

1. **Read the file from disk**
2. **Encode to base64**
3. **Create initial script revision**
4. **Update LoadTest.latestRevisionId**

```python
import base64
import requests

# For each existing load test
load_test_id = "existing-test-id"
locustfile_path = "/path/to/locustfile.py"

# Read and encode
with open(locustfile_path, 'r') as f:
    script = f.read()
    base64_script = base64.b64encode(script.encode()).decode()

# Create revision via API
response = requests.put(
    f"http://localhost:8080/v1/load-tests/{load_test_id}/script",
    json={
        "scriptContent": base64_script,
        "description": "Migrated from file system",
        "updatedBy": "migration-script"
    }
)
```

---

## Best Practices

1. **Always add descriptions** when updating scripts for clear change tracking
2. **Review revision history** before making changes to understand what was modified
3. **Test new revisions** in a staging environment before using in production
4. **Keep revision limit** to 20-50 versions to manage database size
5. **Document major changes** in the description field

---

## Summary

The script revision system provides:
- ✅ Base64 encoded script storage
- ✅ Full version control with audit trail
- ✅ Revision tracking per test run
- ✅ Easy script history viewing
- ✅ No file system dependencies
- ✅ Collaborative editing with clear attribution

All scripts are now managed through the API with automatic revision creation on every edit.
