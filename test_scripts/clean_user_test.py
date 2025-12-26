"""
Clean User Test - No Configuration Required

This is what users write - ONLY test scenarios.
The control plane automatically injects all Harness integration.
"""

from locust import HttpUser, task, between
import random


class ApiUser(HttpUser):
    """Simple API load test user."""
    wait_time = between(1, 3)
    
    @task(3)
    def get_products(self):
        """Most common action - browse products."""
        self.client.get("/api/products", name="GET /api/products")
    
    @task(2)
    def get_product_by_id(self):
        """View specific product details."""
        product_id = random.randint(1, 100)
        self.client.get(f"/api/products/{product_id}", name="GET /api/products/:id")
    
    @task(1)
    def search_products(self):
        """Search functionality."""
        search_terms = ["laptop", "phone", "camera", "watch"]
        query = random.choice(search_terms)
        self.client.get(f"/api/search?q={query}", name="GET /api/search")
