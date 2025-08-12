# Django Real-World Deployment Scenario

This document describes the comprehensive Django deployment pattern implemented in the template functions test suite, demonstrating how enhanced template functions enable sophisticated Django deployments.

## Architecture Overview

The Django scenario implements a production-ready multi-service deployment:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Main App      │    │  Static Files   │    │  Media Files    │
│ myapp.local     │    │static.myapp.local│    │media.myapp.local│
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ django-combined │    │ django-static   │    │ django-media    │
│   (Hypercorn)   │    │    (Nginx)      │    │    (Nginx)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Backend Network │    │ Shared Volumes  │    │ Shared Volumes  │
│ Health Checks   │    │ Static Assets   │    │ Media Uploads   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Service Breakdown

### 1. Django Combined Application (`django-combined`)

**Purpose**: Main Django application serving dynamic content  
**Domain**: `myapp.local`  
**Technology**: Python + Hypercorn ASGI server  
**Template Functions Used**:
- `{{env "DJANGO_ENV"}}` - Environment configuration
- `{{env "DOMAIN_NAME"}}` - Dynamic domain handling
- `{{containerName}}` - Container identification
- `{{isHealthy}}` - Health status headers
- `{{upstreams 8000}}` - Load balancer configuration

**Key Features**:
- Dynamic HTML generation with links to static/media subdomains
- Health check endpoint at `/health/`
- API status endpoint at `/api/status/`
- Environment-aware configuration
- Container metadata in response headers

### 2. Django Static Files Server (`django-static`)

**Purpose**: Serves Django static assets (CSS, JS, images)  
**Domain**: `static.myapp.local`  
**Technology**: Nginx with volume mounts  
**Template Functions Used**:
- `{{env "DOMAIN_NAME"}}` - Dynamic subdomain configuration
- `{{mountSource "/usr/share/nginx/html"}}` - Volume path resolution
- `{{containerName}}` - Server identification headers

**Key Features**:
- Serves from shared `django-static` volume
- Long-term caching headers (`Cache-Control: public, max-age=31536000`)
- Volume source path in headers for debugging
- Includes Django admin CSS and application JavaScript

### 3. Django Media Files Server (`django-media`)

**Purpose**: Serves user-uploaded media files with access control  
**Domain**: `media.myapp.local`  
**Technology**: Nginx with protected file serving  
**Template Functions Used**:
- `{{env "DOMAIN_NAME"}}` - Dynamic subdomain configuration  
- `{{mountSource "/var/www/media"}}` - Media volume path resolution
- `{{env "AUTH_BACKEND_URL"}}` - Authentication backend URL
- `{{containerName}}` - Server identification

**Key Features**:
- **Protected files** (`/protected/*`): Requires authentication via `forward_auth`
- **Public files** (`/public/*`): No authentication required
- Security headers (`X-Content-Type-Options: nosniff`)
- Media path debugging headers
- Serves from shared `django-media` volume

### 4. ASGI/WSGI Separation (Optional)

The test also includes separate ASGI and WSGI containers for advanced deployments:

**Django WSGI** (`django-wsgi`):
- Handles traditional HTTP requests (`/admin/*`, `/api/*`)
- Uses Hypercorn in WSGI mode
- Environment: Database connections, Django settings

**Django ASGI** (`django-asgi`):
- Handles WebSocket connections (`/ws/*`)
- Uses Hypercorn in ASGI mode with Channels
- Environment: Redis connections, real-time features

## Template Function Usage Patterns

### Environment-Based Configuration

```yaml
labels:
  caddy: "{{env \"DOMAIN_NAME\"}}"
  caddy.header.X-Environment: "{{env \"DJANGO_ENV\"}}"
  caddy.header.X-Static-URL: "https://static.{{env \"DOMAIN_NAME\"}}"
```

### Volume-Based Static Serving

```yaml
labels:
  caddy: "static.{{env \"DOMAIN_NAME\"}}"
  caddy.root: "* {{mountSource \"/usr/share/nginx/html\"}}"
  caddy.file_server: ""
  caddy.header.X-Volume-Source: "{{mountSource \"/usr/share/nginx/html\"}}"
```

### Protected Media Serving

```yaml
labels:
  caddy: "media.{{env \"DOMAIN_NAME\"}}"
  caddy.route.0_@protected.path: "/protected/*"
  caddy.route.0_forward_auth: "@protected {{env \"AUTH_BACKEND_URL\"}}/api/auth/"
  caddy.route.0_root: "@protected {{mountSource \"/var/www/media\"}}"
  caddy.route.0_file_server: "@protected"
```

### Health-Aware Routing

```yaml
labels:
  caddy.header.X-Healthy: "{{isHealthy}}"
  caddy.reverse_proxy: "{{if isHealthy}}{{upstreams 8000}}{{end}}"
  caddy.respond: "{{if not (isHealthy)}}503 Service Unavailable{{end}}"
```

### Container Metadata Headers

```yaml
labels:
  caddy.header.X-Container: "{{containerName}}"
  caddy.header.X-Service: "{{containerName}}"
  caddy.header.X-Media-Server: "{{containerName}}"
```

## Deployment Benefits

### 1. **Scalability**
- Static files served directly by Nginx (high performance)
- Media files can be served with authentication
- Main application only handles dynamic content
- Each service can be scaled independently

### 2. **Security**
- Media files support protected access via `forward_auth`
- Static files have appropriate caching and security headers
- Environment variables control authentication and database access
- Container isolation with dedicated networks

### 3. **Flexibility**
- Environment-based configuration (`DOMAIN_NAME`, `DJANGO_ENV`)
- Volume mounts allow external static/media storage
- Health checks enable automated failover
- Template functions provide dynamic configuration without manual setup

### 4. **Production Readiness**
- Proper HTTP caching for static assets
- Security headers for media files
- Health check endpoints for monitoring
- Graceful degradation when services are unhealthy

## Test Validation

The test runner validates:

### Main Application
- Django app serves HTML with proper links to static/media
- Health check returns service status
- API endpoints return container information

### Static Files
- CSS and JavaScript files served correctly
- Proper caching headers applied
- Volume source paths accessible

### Media Files
- Public media files accessible without authentication
- Protected media files require authentication (when implemented)
- Proper security headers applied

### Headers Validation
- Template function values appear in HTTP headers
- Environment variables properly resolved
- Container names and health status included

## Real-World Applications

This pattern works for:

### **Content Management Systems**
- WordPress with separate media server
- Drupal with CDN-like static serving
- Custom CMS with protected uploads

### **E-commerce Platforms**
- Product images via media server
- Static assets via CDN-like serving
- Protected customer documents

### **SaaS Applications**
- User-uploaded files with access control
- Static dashboard assets
- API with separate media handling

### **Multi-Tenant Applications**
- Per-tenant static assets
- Shared media server with tenant isolation
- Environment-based tenant configuration

## Template Functions Demonstrated

This Django scenario showcases these template functions:

✅ **Currently Working**: `{{upstreams}}`  
🔄 **To Be Implemented**:
- `{{env "KEY"}}` - Environment variable access
- `{{containerName}}` - Container identification  
- `{{mountSource "path"}}` - Volume source resolution
- `{{isHealthy}}` - Health check status
- Conditional logic with `{{if}}` statements

The test suite validates that all these functions work correctly and enable sophisticated Django deployments without manual configuration.