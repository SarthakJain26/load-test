"""
Clean Locust Test for VM Products API

This is a clean user script with ONLY test scenarios.
The Harness Control Plane automatically injects integration code.

Endpoints tested:
- GET /api/products (list all products)
- GET /api/products/:id (get specific product)
- GET /api/search (search products)
- POST /api/cart/items (add item to cart)

Target: http://35.239.233.230:8000 (load-testing-vm-1)
"""

# Import Harness plugin for control plane integration (local development)
# When uploaded via API, this is handled automatically by the control plane
import locust_harness_plugin

import random
import gevent
from locust import HttpUser, task, between


class CasualBrowser(HttpUser):
    """
    Simulates casual browsers who mostly view products without purchasing.
    This represents the majority of traffic - 50% of users.
    """
    
    wait_time = between(2, 5)
    weight = 2  # 50% of users
    
    def on_start(self):
        """Initialize user state when spawned."""
        self.product_ids = list(range(1, 101))
    
    @task(5)
    def browse_products_list(self):
        """Browse the products listing page - most common action."""
        self.client.get("/api/products", name="GET /api/products")
    
    @task(3)
    def view_product_details(self):
        """View details of a specific product."""
        product_id = random.choice(self.product_ids)
        self.client.get(f"/api/products/{product_id}", name="GET /api/products/:id")
    
    @task(2)
    def search_products(self):
        """Search for products using various terms."""
        search_terms = ["laptop", "phone", "camera", "watch", "keyboard", 
                       "mouse", "tablet", "headphones", "speaker", "monitor"]
        query = random.choice(search_terms)
        self.client.get(f"/api/search?q={query}", name="GET /api/search")


class Shopper(HttpUser):
    """
    Simulates active shoppers who browse and add items to cart.
    These users make purchases - 25% of traffic.
    """
    
    wait_time = between(1, 3)
    weight = 1  # 25% of users
    
    def on_start(self):
        """Initialize shopper state."""
        self.product_ids = list(range(1, 101))
        self.cart_items = []
    
    @task(3)
    def browse_before_buying(self):
        """Browse products before making purchase decision."""
        self.client.get("/api/products", name="GET /api/products")
    
    @task(4)
    def view_product_to_buy(self):
        """View product details before adding to cart."""
        product_id = random.choice(self.product_ids)
        with self.client.get(
            f"/api/products/{product_id}",
            catch_response=True,
            name="GET /api/products/:id"
        ) as response:
            if response.status_code == 200:
                response.success()
                # Store this as a potential purchase
                if product_id not in self.cart_items:
                    self.cart_items.append(product_id)
            else:
                response.failure(f"Failed to load product {product_id}")
    
    @task(2)
    def search_for_products(self):
        """Search to find specific products."""
        search_terms = ["best", "new", "sale", "cheap", "premium", 
                       "popular", "trending", "featured"]
        query = random.choice(search_terms)
        self.client.get(f"/api/search?q={query}", name="GET /api/search")
    
    @task(3)
    def add_to_cart(self):
        """Add a product to the shopping cart."""
        # Either add a previously viewed product or a random one
        if self.cart_items and random.random() > 0.3:
            product_id = random.choice(self.cart_items)
        else:
            product_id = random.choice(self.product_ids)
        
        payload = {
            "productId": product_id,
            "quantity": random.randint(1, 3)
        }
        
        with self.client.post(
            "/api/cart/items",
            json=payload,
            catch_response=True,
            name="POST /api/cart/items"
        ) as response:
            if response.status_code in [200, 201]:
                response.success()
            else:
                response.failure(f"Failed to add product {product_id} to cart")


class PowerUser(HttpUser):
    """
    Simulates power users who rapidly browse and purchase.
    These are heavy users making many requests - 25% of traffic.
    """
    
    wait_time = between(0.5, 1.5)  # Faster interaction
    weight = 1  # 25% of users
    
    def on_start(self):
        """Initialize power user state."""
        self.product_ids = list(range(1, 101))
    
    @task(5)
    def rapid_search(self):
        """Perform multiple rapid searches."""
        search_terms = ["new", "sale", "best", "top", "popular", 
                       "featured", "recommended", "trending"]
        query = random.choice(search_terms)
        self.client.get(f"/api/search?q={query}", name="GET /api/search")
    
    @task(8)
    def browse_multiple_products(self):
        """View multiple product details in quick succession."""
        num_products = random.randint(2, 4)
        for _ in range(num_products):
            product_id = random.choice(self.product_ids)
            self.client.get(
                f"/api/products/{product_id}",
                name="GET /api/products/:id"
            )
            gevent.sleep(0.2)  # Brief pause between requests
    
    @task(3)
    def add_multiple_to_cart(self):
        """Add multiple items to cart rapidly."""
        num_items = random.randint(2, 3)
        for _ in range(num_items):
            product_id = random.choice(self.product_ids)
            payload = {
                "productId": product_id,
                "quantity": random.randint(1, 2)
            }
            self.client.post(
                "/api/cart/items",
                json=payload,
                name="POST /api/cart/items"
            )
            gevent.sleep(0.3)
    
    @task(2)
    def view_all_products_frequently(self):
        """Frequently check the full product catalog."""
        self.client.get("/api/products", name="GET /api/products")
