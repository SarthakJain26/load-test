# Harness Load Testing - User Guide

## Overview

Harness Load Testing allows you to write **clean, simple load test scenarios** without worrying about infrastructure, metrics collection, or control plane integration. Just define your test logic, and we handle the rest.

---

## Quick Start

### 1. Write Your Test (3 minutes)

Create a Python file with your load test scenarios:

```python
# my_api_test.py
from locust import HttpUser, task, between
import random

class ApiUser(HttpUser):
    wait_time = between(1, 3)
    
    @task
    def get_products(self):
        self.client.get("/api/products")
    
    @task
    def get_product_by_id(self):
        product_id = random.randint(1, 100)
        self.client.get(f"/api/products/{product_id}")
```

**That's it!** No configuration, no metrics code, no lifecycle management.

### 2. Upload to Harness (1 minute)

```bash
# Using Harness CLI or API
harness load-test create \
  --name "My API Test" \
  --script my_api_test.py \
  --target-url "https://api.example.com"
```

### 3. Run the Test

```bash
harness load-test run \
  --name "My API Test" \
  --users 50 \
  --duration 300  # 5 minutes
```

Harness automatically:
- ✅ Injects control plane integration
- ✅ Collects real-time metrics
- ✅ Stops test after duration
- ✅ Updates test status
- ✅ Stores historical data

---

## What Users Write vs What Harness Handles

### ✍️ What You Write (Clean Test Scenarios)

```python
from locust import HttpUser, task, between

class MyUser(HttpUser):
    wait_time = between(1, 3)
    
    @task
    def my_test(self):
        self.client.get("/api/endpoint")
```

### 🤖 What Harness Injects Automatically (Hidden from Users)

- **Test lifecycle management** (start/stop notifications)
- **Real-time metrics collection** (RPS, latency, errors)
- **Duration-based auto-stop**
- **Control plane communication**
- **Authentication & security**
- **Error handling & retries**

---

## Writing Load Tests

### Basic Structure

```python
from locust import HttpUser, task, between

class UserBehavior(HttpUser):
    # Wait 1-3 seconds between tasks
    wait_time = between(1, 3)
    
    @task
    def my_task(self):
        # Your test logic here
        response = self.client.get("/api/endpoint")
        assert response.status_code == 200
```

### Multiple User Types

Simulate different user behaviors with weights:

```python
class Browsers(HttpUser):
    """Users who just browse - 70% of traffic"""
    weight = 7
    wait_time = between(1, 2)
    
    @task(3)
    def browse_list(self):
        self.client.get("/api/products")
    
    @task(1)
    def view_details(self):
        self.client.get(f"/api/products/{random.randint(1, 100)}")


class Buyers(HttpUser):
    """Users who make purchases - 30% of traffic"""
    weight = 3
    wait_time = between(2, 5)
    
    @task
    def add_to_cart(self):
        self.client.post("/api/cart", json={"productId": 123, "qty": 1})
```

### Task Priorities

Use task weights to control frequency:

```python
class MyUser(HttpUser):
    @task(10)  # Runs 10x more often
    def common_action(self):
        self.client.get("/api/common")
    
    @task(1)   # Runs occasionally
    def rare_action(self):
        self.client.get("/api/rare")
```

### Sequential Tasks

Use `SequentialTaskSet` for workflows:

```python
from locust import HttpUser, SequentialTaskSet, task

class CheckoutFlow(SequentialTaskSet):
    @task
    def browse_products(self):
        self.client.get("/api/products")
    
    @task
    def add_to_cart(self):
        self.client.post("/api/cart", json={"productId": 1})
    
    @task
    def checkout(self):
        self.client.post("/api/checkout")

class ShopperUser(HttpUser):
    tasks = [CheckoutFlow]
    wait_time = between(1, 3)
```

---

## Advanced Features

### Custom Headers

```python
class AuthenticatedUser(HttpUser):
    def on_start(self):
        # Runs once per user at start
        self.client.headers = {"Authorization": "Bearer token123"}
    
    @task
    def get_profile(self):
        self.client.get("/api/profile")
```

### Session Management

```python
class LoginUser(HttpUser):
    def on_start(self):
        # Login and store session
        response = self.client.post("/api/login", json={
            "username": "test",
            "password": "test123"
        })
        self.token = response.json()["token"]
    
    @task
    def get_data(self):
        self.client.get(
            "/api/data",
            headers={"Authorization": f"Bearer {self.token}"}
        )
```

### Data-Driven Tests

```python
import csv
from locust import HttpUser, task

class DataDrivenUser(HttpUser):
    def on_start(self):
        # Load test data once per user
        with open("test_data.csv") as f:
            self.test_data = list(csv.DictReader(f))
    
    @task
    def test_with_data(self):
        row = random.choice(self.test_data)
        self.client.get(f"/api/users/{row['user_id']}")
```

---

## Best Practices

### ✅ DO

- **Keep tests simple** - Focus on user scenarios, not infrastructure
- **Use descriptive names** - Name tasks and users clearly
- **Test realistic flows** - Mimic actual user behavior
- **Set appropriate wait times** - Match real user think time
- **Use weights wisely** - Reflect actual traffic patterns

### ❌ DON'T

- **Don't add control plane code** - Harness handles this automatically
- **Don't hardcode credentials** - Use environment variables or test data
- **Don't test unrealistic scenarios** - Keep it close to production behavior
- **Don't ignore errors** - Assert on response codes when needed

---

## Examples

### REST API Test

```python
from locust import HttpUser, task, between
import random

class ApiLoadTest(HttpUser):
    wait_time = between(1, 3)
    
    @task(3)
    def list_items(self):
        self.client.get("/api/items")
    
    @task(2)
    def get_item(self):
        item_id = random.randint(1, 1000)
        self.client.get(f"/api/items/{item_id}")
    
    @task(1)
    def create_item(self):
        self.client.post("/api/items", json={
            "name": f"Item-{random.randint(1, 9999)}",
            "price": random.uniform(10, 1000)
        })
```

### E-commerce Workflow

```python
from locust import HttpUser, SequentialTaskSet, task, between
import random

class ShoppingFlow(SequentialTaskSet):
    @task
    def browse_products(self):
        self.client.get("/api/products")
    
    @task
    def view_product(self):
        self.product_id = random.randint(1, 100)
        self.client.get(f"/api/products/{self.product_id}")
    
    @task
    def add_to_cart(self):
        self.client.post("/api/cart", json={
            "productId": self.product_id,
            "quantity": random.randint(1, 3)
        })
    
    @task
    def checkout(self):
        self.client.post("/api/checkout")

class Shopper(HttpUser):
    tasks = [ShoppingFlow]
    wait_time = between(2, 5)
```

---

## Migration from Standalone Locust

If you have existing Locust tests, migration is simple:

### Before (Standalone Locust)
```python
# mytest.py - Old standalone version
from locust import HttpUser, task

# Custom metrics code, lifecycle management, etc.
def setup_metrics():
    # 50+ lines of boilerplate...
    pass

class MyUser(HttpUser):
    @task
    def my_test(self):
        # Your test code
        pass
```

### After (Harness)
```python
# mytest.py - Clean Harness version
from locust import HttpUser, task

class MyUser(HttpUser):
    @task
    def my_test(self):
        # Same test code - that's it!
        pass
```

**Just remove all the infrastructure code!** Harness handles everything automatically.

---

## Troubleshooting

### Test Not Starting?
- Check that your file has valid Python syntax
- Ensure at least one `HttpUser` class with `@task` methods exists
- Verify target URL is accessible

### No Metrics Showing?
- Harness automatically collects metrics - no action needed
- Check control plane logs if issues persist

### Need Help?
- Check [API Documentation](./API_DOCS.md)
- Review [Examples](../locust/examples/)
- Contact support

---

## Next Steps

- 📚 Read [API Documentation](./API_DOCS.md)
- 🔧 Explore [Advanced Configuration](./ADVANCED.md)
- 🚀 Deploy to [Production](./PRODUCTION.md)

---

**Remember:** With Harness, you write tests, we handle everything else. Keep it simple! 🎯
