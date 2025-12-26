"""
Example Load Test: Products API

This file contains ONLY the load test scenarios.
All control plane integration is handled automatically by the Harness plugin.

Users should focus on:
1. Defining HttpUser classes
2. Writing @task methods with test logic
3. Setting wait_time and weights

The Harness platform handles:
- Metrics collection and reporting
- Test lifecycle management
- Duration-based auto-stop
- Integration with control plane
"""

# REQUIRED: Import the Harness plugin (this is the only "magic" line needed)
import sys
import os
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))
import locust_harness_plugin  # noqa: F401

# Now write your test scenarios as normal
from locust import HttpUser, task, between
import random


class ProductsBrowser(HttpUser):
    """
    Simulates users browsing the products catalog.
    Weight=3 means this user type is 3x more common than others.
    """
    wait_time = between(1, 3)
    weight = 3
    
    @task(3)
    def browse_products(self):
        """List all products - most common action."""
        self.client.get("/api/products", name="GET /api/products")
    
    @task(2)
    def view_product_details(self):
        """View a specific product by ID."""
        product_id = random.randint(1, 100)
        self.client.get(f"/api/products/{product_id}", name="GET /api/products/:id")
    
    @task(1)
    def search_products(self):
        """Search for products."""
        search_terms = ["laptop", "phone", "camera", "watch", "keyboard"]
        query = random.choice(search_terms)
        self.client.get(f"/api/search?q={query}", name="GET /api/search")


class ProductsBuyer(HttpUser):
    """
    Simulates users who actively add items to cart.
    Weight=1 means this is less common than browsing.
    """
    wait_time = between(2, 5)
    weight = 1
    
    @task(2)
    def view_product(self):
        """View product before buying."""
        product_id = random.randint(1, 50)
        self.client.get(f"/api/products/{product_id}", name="GET /api/products/:id")
    
    @task(1)
    def add_to_cart(self):
        """Add a random product to cart."""
        product_id = random.randint(1, 50)
        quantity = random.randint(1, 3)
        
        self.client.post(
            "/api/cart/items",
            json={
                "productId": product_id,
                "quantity": quantity
            },
            name="POST /api/cart/items"
        )
    
    @task(1)
    def view_cart(self):
        """Check current cart contents."""
        self.client.get("/api/cart", name="GET /api/cart")
