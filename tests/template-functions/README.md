# Enhanced Template Functions Test Suite

This test suite validates all the documented template functions for caddy-docker-proxy enhanced functionality.

## Overview

The test suite covers all major template function categories:

- **Volume/Storage Functions**: `mountSource`, `bindMounts`, `volumeMounts`, `hasMount`
- **Environment Variable Functions**: `env`, `hasEnv`, `envPrefix`
- **Network/Port Functions**: `networkIP`, `primaryIP`, `portMapping`, `exposedPorts`, `networks`
- **Container State Functions**: `isRunning`, `isHealthy`, `uptime`, `restartCount`
- **Label/Metadata Functions**: `label`, `hasLabel`, `labelPrefix`, `containerName`, `imageName`, `imageTag`

## Test Scenarios

### Volume/Storage Tests

1. **Static File Serving** (`static-server`)
   - Serves files from volume using `{{mountSource "/path"}}`
   - Tests basic file_server directive with dynamic root

2. **Mixed Static + API** (`mixed-app`)
   - Combines static file serving with reverse proxy
   - Uses route directives for path-based routing

### Environment Variable Tests

3. **Environment Config Display** (`env-app`)
   - Displays environment variables using `{{env "KEY"}}`
   - Tests multiple environment variable access

4. **Environment-Based Authentication** (`auth-app`)
   - Conditional basicauth using environment variables
   - Tests `{{if env "ENABLE_AUTH"}}` patterns

### Network/Port Tests

5. **Network Information** (`network-app`)
   - Displays network info using `{{primaryIP}}` and `{{networks}}`
   - Tests network metadata access

6. **Multi-Port Routing** (`multi-port-app`)
   - Routes based on `{{portMapping}}` function
   - Tests port-aware configuration

### Container State Tests

7. **Health-Aware Routing** (`health-app`)
   - Uses `{{isRunning}}` and `{{isHealthy}}` for conditional routing
   - Tests uptime display with `{{uptime}}`

### Label/Metadata Tests

8. **Metadata Display** (`metadata-app`)
   - Shows container info using `{{containerName}}`, `{{imageName}}`
   - Tests label access with `{{label "key"}}`

9. **Dynamic Configuration** (`dynamic-config-app`)
   - Uses labels to configure rate limiting and CORS
   - Tests `{{label "prefix.*"}}` patterns

### Real-World Scenarios

10. **React App with API Backend** (`react-frontend` + `api-backend`)
    - Complete SPA deployment scenario
    - Static files + API proxy + environment headers

11. **Conditional Authentication** (`conditional-app`)
    - Complex conditional logic using multiple functions
    - Tests authentication and maintenance mode patterns

## Running Tests

### Quick Test Run

```bash
# Run all tests
./run.sh
```

### Comprehensive Validation

```bash
# Run validation with detailed checking
./validate.sh
```

### Manual Testing

```bash
# Deploy stack manually
docker network create --driver overlay --attachable caddy_test
docker stack deploy -c compose.yaml --prune caddy_test

# Test individual endpoints
curl -k --resolve static.example.com:443:127.0.0.1 https://static.example.com/
curl -k --resolve env-config.example.com:443:127.0.0.1 https://env-config.example.com/config
curl -k --resolve auth.example.com:443:127.0.0.1 https://auth.example.com/admin -u admin:secret123

# Cleanup
docker stack rm caddy_test
```

## Expected Behavior

### Current Implementation (Baseline)

With the current caddy-docker-proxy implementation, only basic reverse proxy functionality works:

- ✅ `{{upstreams}}` function works
- ✅ Basic label-to-Caddyfile conversion
- ❌ Enhanced template functions return empty strings (graceful failure)

### After Implementation

Once enhanced template functions are implemented:

- ✅ All template functions return proper values
- ✅ Volume serving works correctly
- ✅ Environment-based configuration works
- ✅ Network and state-aware routing works
- ✅ Dynamic configuration from labels works

## Test Data

The test suite includes realistic test data:

- **Static Files**: HTML, CSS, text files in `test-data/static/`
- **Web Apps**: Simulated React build in `test-data/react-build/`  
- **Mixed Content**: Combined static/dynamic content in `test-data/webapp/`

## Debugging

### View Generated Caddyfile

```bash
docker service logs caddy_test_caddy | grep -A 50 "New Caddyfile"
```

### Check Container Status

```bash
docker service ls
docker service ps caddy_test_caddy
```

### Test Individual Services

```bash
# Check if services are responding
docker service logs caddy_test_static-server
docker service logs caddy_test_env-app
```

## Integration with CI/CD

This test suite can be integrated into the main test runner:

```bash
# Add to tests/run.sh
cd template-functions && ./run.sh && cd ..
```

Or run as part of the build process:

```bash
# After building new image
./tests/template-functions/validate.sh
```

## Development Workflow

1. **Implement Function**: Add new template function to `generator/labels.go`
2. **Update Tests**: Add test scenario to `compose.yaml`
3. **Add Validation**: Update `run.sh` with new test cases
4. **Run Suite**: Execute `./validate.sh` to verify functionality
5. **Update Docs**: Add examples to `TEMPLATE_FUNCTIONS.md`

This comprehensive test suite ensures that all documented template functions work correctly and provides a solid foundation for development and validation of enhanced caddy-docker-proxy functionality.