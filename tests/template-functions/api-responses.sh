#!/bin/bash

# Mock API responses for testing
# This script provides fake API endpoints for the test containers

set -e

echo "=== Mock API Response Generator ==="

# This would be used by test containers to provide realistic API responses
# For now, the traefik/whoami container provides basic HTTP responses

# Future: Could use a lightweight API mock service like:
# - httpbin.org/anything
# - mockserver
# - custom node.js/python mock server

# Example API responses that containers should provide:

cat << 'EOF'
# Health Check Response
GET /api/health -> 200 OK
{
  "status": "healthy",
  "uptime": "2h30m",
  "version": "1.0.0"
}

# Data API Response  
GET /api/data -> 200 OK
{
  "message": "API Data from container",
  "timestamp": "2024-01-01T12:00:00Z",
  "container": "api-backend"
}

# Config API Response
GET /config -> 200 OK
APP_NAME=testapp APP_VERSION=1.0 ENV=testing DEBUG=true

# Status API Response
GET /status -> 200 OK
Container: running Healthy: true Uptime: 1h23m45s
EOF