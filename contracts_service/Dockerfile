FROM python:3.11-slim

WORKDIR /app

# Install curl for healthcheck
RUN apt-get update && \
    apt-get install -y curl && \
    rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

# Default port (can be overridden by environment variable)
ENV CONTRACTS_SERVICE_PORT=8000

EXPOSE ${CONTRACTS_SERVICE_PORT}

# Use uvicorn with environment-based port
CMD ["sh", "-c", "uvicorn main:app --host 0.0.0.0 --port ${CONTRACTS_SERVICE_PORT}"]
