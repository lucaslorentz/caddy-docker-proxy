# Caddy-Native Django Deployment Pattern

This document describes the **pure Caddy** Django deployment pattern - no Nginx, no additional web servers, just Caddy handling everything through template functions and volume mounts.

## 🎯 Architecture Philosophy

**One Caddy to Rule Them All**: The main Caddy container serves static files, media files, and proxies to Django applications using volumes and template functions.

```
                    ┌─── Caddy Main Container ───┐
                    │                            │
┌─────────────┐     │  ┌─────────┐ ┌─────────────┐ │     ┌─────────────┐
│   Clients   │────▶│  │ Static  │ │   Media     │ │────▶│   Django    │
│             │     │  │ Files   │ │   Files     │ │     │     App     │
└─────────────┘     │  │file_srv │ │file_srv+auth│ │     └─────────────┘
                    │  └─────────┘ └─────────────┘ │            │
                    │           ▲         ▲       │            │
                    └───────────┼─────────┼───────┘            │
                                │         │                    │
                    ┌───────────┼─────────┼────────────────────┘
                    │           │         │
                    ▼           ▼         ▼
               ┌─────────┐ ┌─────────┐ ┌─────────┐
               │ Django  │ │ Django  │ │ Health  │
               │ Static  │ │ Media   │ │ Checks  │
               │ Volume  │ │ Volume  │ │ & APIs  │
               └─────────┘ └─────────┘ └─────────┘
```

## 🔧 Configuration Breakdown

### Main Caddy Container Labels

The Caddy container itself has labels for serving static and media files:

```yaml
caddy:
  image: caddy-docker-proxy:local
  volumes:
    # Mount Django volumes directly into Caddy
    - django-static:/srv/django-static:ro
    - django-media:/srv/django-media:ro
    - ./test-data/django/static:/srv/django-static:ro
    - ./test-data/django/media:/srv/django-media:ro
  labels:
    # Static Files Domain (caddy_0 for isolation)
    caddy_0: "static.{{env \"DOMAIN_NAME\"}}"
    caddy_0.root: "* /srv/django-static"
    caddy_0.file_server: ""
    caddy_0.header.Cache-Control: "public, max-age=31536000"
    caddy_0.tls: "internal"
    
    # Media Files Domain (caddy_1 for isolation)
    caddy_1: "media.{{env \"DOMAIN_NAME\"}}"
    # Protected files require authentication
    caddy_1.route.0_@protected.path: "/protected/*"
    caddy_1.route.0_forward_auth: "@protected {{env \"AUTH_BACKEND_URL\"}}/api/auth/"
    caddy_1.route.0_root: "@protected /srv/django-media"
    caddy_1.route.0_file_server: "@protected"
    # Public files - no auth
    caddy_1.route.1_@public.path: "/public/*"
    caddy_1.route.1_root: "@public /srv/django-media"
    caddy_1.route.1_file_server: "@public"
    caddy_1.tls: "internal"
```

### Django Application Container

The Django app only handles dynamic content:

```yaml
django-combined:
  image: python:3.11-slim
  # Minimal Django app with health checks and API endpoints
  environment:
    - DJANGO_ENV=production
    - DOMAIN_NAME=myapp.local
  volumes:
    - django-static:/app/static:ro  # Read access to see static files
    - django-media:/app/media:ro    # Read access to see media files
  labels:
    # Main Django app
    caddy: "{{env \"DOMAIN_NAME\"}}"
    caddy.reverse_proxy: "{{upstreams 8000}}"
    # Add helpful headers showing the setup
    caddy.header.X-Static-URL: "https://static.{{env \"DOMAIN_NAME\"}}"
    caddy.header.X-Media-URL: "https://media.{{env \"DOMAIN_NAME\"}}"
    caddy.header.X-Container: "{{containerName}}"
    caddy.header.X-Environment: "{{env \"DJANGO_ENV\"}}"
    caddy.tls: "internal"
```

### Volume Data Containers (Optional)

Simple containers to populate volumes with data:

```yaml
django-static-data:
  image: busybox:latest
  command: ["sleep", "3600"]  # Just keep volumes alive
  volumes:
    - django-static:/app/static
    - ./test-data/django/static:/app/static:ro
  # No labels - doesn't serve anything

django-media-data:
  image: busybox:latest  
  command: ["sleep", "3600"]  # Just keep volumes alive
  volumes:
    - django-media:/app/media
    - ./test-data/django/media:/app/media:ro
  # No labels - doesn't serve anything
```

## 🚀 Benefits of Caddy-Native Approach

### **1. Simplicity**
- One web server to configure and monitor
- No Nginx configuration files
- All routing in Docker labels
- Unified logging and metrics

### **2. Performance**  
- Caddy's built-in `file_server` is highly optimized
- HTTP/2 and HTTP/3 support out of the box
- Automatic compression (gzip, brotli)
- Built-in caching headers

### **3. Security**
- Caddy's `forward_auth` for protected media files
- Automatic HTTPS with Let's Encrypt
- Security headers built-in
- No additional attack surface from multiple web servers

### **4. Dynamic Configuration**
- Environment-based domains: `{{env "DOMAIN_NAME"}}`
- Template functions for paths and headers
- Container metadata in responses
- Health-aware routing

### **5. Operational Benefits**
- Single container to scale for static/media serving
- Unified monitoring and logging
- Caddy's admin API for runtime config
- Graceful configuration reloads

## 📁 Volume Strategy

### **Shared Volumes Pattern**
```yaml
volumes:
  django-static:    # Shared between Django app and Caddy
  django-media:     # Shared between Django app and Caddy

# Django app writes to volumes
django-app:
  volumes:
    - django-static:/app/static    # Write access
    - django-media:/app/media      # Write access

# Caddy reads from volumes  
caddy:
  volumes:
    - django-static:/srv/django-static:ro  # Read-only access
    - django-media:/srv/django-media:ro    # Read-only access
```

### **Host Mount Pattern**
```yaml
# For development/simple deployments
caddy:
  volumes:
    - ./static:/srv/django-static:ro
    - ./media:/srv/django-media:ro
```

## 🔒 Protected Media Files

Caddy handles authentication for protected files:

```yaml
# Protected route
caddy_1.route.0_@protected.path: "/protected/*"
caddy_1.route.0_forward_auth: "@protected http://django-app:8000/api/auth/"
caddy_1.route.0_root: "@protected /srv/django-media"
caddy_1.route.0_file_server: "@protected"

# Public route  
caddy_1.route.1_@public.path: "/public/*"
caddy_1.route.1_root: "@public /srv/django-media"
caddy_1.route.1_file_server: "@public"
```

**Flow:**
1. User requests `/protected/document.pdf`
2. Caddy calls Django auth endpoint: `POST http://django-app:8000/api/auth/`
3. If auth succeeds (200), Caddy serves file from volume
4. If auth fails (401/403), Caddy returns auth error
5. Public files skip auth entirely

## 🎨 Template Functions Usage

### **Environment Configuration**
```yaml
caddy_0: "static.{{env \"DOMAIN_NAME\"}}"           # → static.myapp.local
caddy.header.X-Environment: "{{env \"DJANGO_ENV\"}}" # → production
```

### **Container Metadata**
```yaml
caddy.header.X-Container: "{{containerName}}"       # → django-combined
caddy.header.X-Served-By: "caddy-main"             # → caddy-main  
```

### **Conditional Logic (Future)**
```yaml
# Only serve if container is healthy
caddy.file_server: "{{if isHealthy}}{{end}}"
caddy.respond: "{{if not (isHealthy)}}503{{end}}"

# Different caching based on environment
caddy.header.Cache-Control: "{{if eq (env \"DJANGO_ENV\") \"production\"}}public, max-age=31536000{{else}}no-cache{{end}}"
```

### **Volume Path Access (Future)**
```yaml
caddy.root: "* {{mountSource \"/srv/django-static\"}}"  # Get actual mount path
caddy.header.X-Volume-Path: "{{mountSource \"/srv/django-static\"}}"
```

## 📊 Monitoring & Debugging

### **Headers for Debugging**
Every response includes helpful headers:
```
X-Served-By: caddy-main
X-Static-Path: /srv/django-static
X-Media-Path: /srv/django-media
X-Container: django-combined
X-Environment: production
X-Static-URL: https://static.myapp.local
X-Media-URL: https://media.myapp.local
```

### **Health Checks**
```yaml
caddy.header.X-Healthy: "{{isHealthy}}"  # Container health status
```

### **Caddy Metrics**
- Built-in metrics endpoint: `http://caddy:2019/metrics`
- File serving stats, request counts, response times
- No need for separate monitoring tools

## 🌍 Real-World Deployment

### **Production Example**
```yaml
version: '3.7'
services:
  caddy:
    image: caddy-docker-proxy:latest
    environment:
      - DOMAIN_NAME=myapp.com
      - DJANGO_ENV=production
      - AUTH_BACKEND_URL=http://django-app:8000
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - django-static:/srv/static:ro
      - django-media:/srv/media:ro
      - caddy-data:/data  # For Let's Encrypt certificates
    labels:
      caddy_0: "static.{{env \"DOMAIN_NAME\"}}"
      caddy_0.file_server: "/srv/static"
      # Automatic HTTPS from Let's Encrypt
      
      caddy_1: "media.{{env \"DOMAIN_NAME\"}}"
      caddy_1.forward_auth: "{{env \"AUTH_BACKEND_URL\"}}/auth/"
      caddy_1.file_server: "/srv/media"

  django-app:
    image: myapp:latest
    environment:
      - DOMAIN_NAME=myapp.com
      - DATABASE_URL=postgres://...
    volumes:
      - django-static:/app/static
      - django-media:/app/media
    labels:
      caddy: "{{env \"DOMAIN_NAME\"}}"
      caddy.reverse_proxy: "{{upstreams}}"

volumes:
  django-static:
  django-media:
  caddy-data:
```

## 🎯 Key Advantages Over Nginx

| Feature | Caddy-Native | Nginx + Caddy |
|---------|-------------|---------------|
| **Configuration** | Labels only | Labels + nginx.conf |
| **TLS** | Automatic Let's Encrypt | Manual setup |
| **HTTP/3** | Built-in | Requires nginx 1.25+ |
| **Monitoring** | Unified logs | Multiple log sources |
| **Scaling** | Scale one service | Scale two services |
| **Memory** | Single process | Multiple processes |
| **Updates** | Update one image | Update two images |

This pure Caddy approach eliminates complexity while providing all the functionality of a traditional multi-tier setup, with the added benefit of dynamic configuration through template functions!