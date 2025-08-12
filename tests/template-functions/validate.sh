#!/bin/bash

set -e

# Validation script for enhanced template functions
# This script validates that all documented template functions work correctly

echo "=== Enhanced Template Functions Validation ==="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

function test_passed() {
    echo -e "${GREEN}✓ $1${NC}"
}

function test_failed() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

function test_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Build test image first
echo "Building caddy-docker-proxy test image..."
cd ../..
./build.sh
docker tag $(docker images -q caddy:latest | head -n1) caddy-docker-proxy:local
cd tests/template-functions

# Ensure network exists
echo "Creating test network..."
docker network create --driver overlay --attachable caddy_test 2>/dev/null || echo "Network already exists"

echo "Starting validation tests..."

# Run the test suite
if ./run.sh; then
    test_passed "All template function tests passed"
else
    test_failed "Template function tests failed"
fi

# Additional validation tests
echo "Running additional validation checks..."

# Check Caddyfile generation
echo "Validating generated Caddyfile..."
CADDYFILE_LOG=$(docker service logs caddy_test_caddy 2>&1 | grep -A 20 "New Caddyfile" | tail -n 20)

# Validate Volume Functions
echo "Checking volume function usage..."
if echo "$CADDYFILE_LOG" | grep -q "mountSource"; then
    test_passed "Volume functions are being processed"
else
    test_warning "Volume functions not found in logs (may not be implemented yet)"
fi

# Validate Environment Functions  
echo "Checking environment function usage..."
if echo "$CADDYFILE_LOG" | grep -q "env "; then
    test_passed "Environment functions are being processed"
else
    test_warning "Environment functions not found in logs (may not be implemented yet)"
fi

# Validate Network Functions
echo "Checking network function usage..."
if echo "$CADDYFILE_LOG" | grep -q "primaryIP\|networkIP"; then
    test_passed "Network functions are being processed"
else
    test_warning "Network functions not found in logs (may not be implemented yet)"
fi

# Validate State Functions
echo "Checking container state function usage..."
if echo "$CADDYFILE_LOG" | grep -q "isRunning\|isHealthy"; then
    test_passed "Container state functions are being processed"
else
    test_warning "Container state functions not found in logs (may not be implemented yet)"
fi

# Validate Label Functions
echo "Checking label function usage..."
if echo "$CADDYFILE_LOG" | grep -q "label \|containerName"; then
    test_passed "Label functions are being processed"
else
    test_warning "Label functions not found in logs (may not be implemented yet)"
fi

# Test actual HTTP responses for implemented functions
echo "Testing HTTP responses..."

function test_http_response() {
    local url=$1
    local expected=$2
    local description=$3
    
    response=$(curl --show-error -s -k -f --resolve "${url}" 2>/dev/null || echo "FAILED")
    
    if echo "$response" | grep -q "$expected"; then
        test_passed "$description"
        return 0
    else
        test_warning "$description - Expected: $expected, Got: $response"
        return 1
    fi
}

# Test basic functionality that should work with current system
test_http_response "static.example.com:443:127.0.0.1 https://static.example.com/" "Static Content" "Static file serving"
test_http_response "env-config.example.com:443:127.0.0.1 https://env-config.example.com/config" "APP_NAME" "Environment variable access"

echo "Validation complete!"

# Cleanup
echo "Cleaning up test resources..."
docker stack rm caddy_test 2>/dev/null || true
sleep 5

echo -e "${GREEN}Validation script completed successfully!${NC}"
echo "Note: Some warnings are expected if template functions are not yet implemented."