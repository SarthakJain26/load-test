# Dockerfile for Locust load generator with control plane integration
FROM python:3.11-slim

WORKDIR /locust

# Install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy locustfile
COPY locustfile.py .

# Expose Locust web UI port (master) and worker communication ports
EXPOSE 8089 5557

# Default command (override with --master or --worker flags)
# Example for master: docker run ... locust -f locustfile.py --master --web-host 0.0.0.0
# Example for worker: docker run ... locust -f locustfile.py --worker --master-host <master-host>
ENTRYPOINT ["locust"]
CMD ["-f", "locustfile.py"]
