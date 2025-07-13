# Deployment Guide

This guide covers deploying the Movie Database application in production with its single-binary architecture.

## Overview

The Movie Database app builds into a single, self-contained binary that includes:
- Go backend server
- React frontend (embedded)
- Database migrations
- Static assets

## Prerequisites

- Go 1.22+ (for building)
- Node.js 18+ (for building)
- Production server (Linux/macOS/Windows)
- Auth0 configuration
- TMDB API key

## Build Process

### 1. Production Environment Setup

Create production environment file `web/.env.production`:
```bash
VITE_AUTH0_DOMAIN=your-production-domain.auth0.com
VITE_AUTH0_CLIENT_ID=your-production-client-id
VITE_AUTH0_AUDIENCE=https://your-api-audience
```

### 2. Build Production Binary

```bash
# Clone repository
git clone <repository-url>
cd movie-db

# Install dependencies
make install

# Build optimized production binary
make build-prod
```

This creates `bin/moviedb` - a single executable containing everything.

### 3. Verify Build

```bash
# Check binary exists and size
ls -lh bin/moviedb

# Test binary runs
./bin/moviedb --help
```

## Deployment Options

### Option 1: Direct Server Deployment

#### 1. Upload Binary
```bash
# Copy to production server
scp bin/moviedb user@server:/opt/moviedb/

# SSH to server
ssh user@server
cd /opt/moviedb
```

#### 2. Set Environment Variables
Create `/opt/moviedb/.env`:
```bash
AUTH0_DOMAIN=your-domain.auth0.com
AUTH0_AUDIENCE=https://your-api-audience
TMDB_API_KEY=your-tmdb-api-key
DATABASE_PATH=./moviedb.db
PORT=8080
ENV=production
```

#### 3. Create Systemd Service
Create `/etc/systemd/system/moviedb.service`:
```ini
[Unit]
Description=Movie Database Application
After=network.target

[Service]
Type=simple
User=moviedb
WorkingDirectory=/opt/moviedb
ExecStart=/opt/moviedb/moviedb
Restart=always
RestartSec=10
Environment=PATH=/usr/local/bin:/usr/bin:/bin
EnvironmentFile=/opt/moviedb/.env

[Install]
WantedBy=multi-user.target
```

#### 4. Start Service
```bash
# Create user
sudo useradd -r -s /bin/false moviedb
sudo chown -R moviedb:moviedb /opt/moviedb

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable moviedb
sudo systemctl start moviedb

# Check status
sudo systemctl status moviedb
```

### Option 2: Docker Deployment

#### 1. Create Dockerfile
```dockerfile
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create app user
RUN addgroup -g 1001 -S moviedb && \
    adduser -u 1001 -S moviedb -G moviedb

# Set working directory
WORKDIR /app

# Copy binary
COPY bin/moviedb .
RUN chmod +x moviedb

# Change ownership
RUN chown moviedb:moviedb /app

# Switch to non-root user
USER moviedb

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run application
CMD ["./moviedb"]
```

#### 2. Build and Run
```bash
# Build image
docker build -t moviedb .

# Run container
docker run -d \
  --name moviedb \
  -p 8080:8080 \
  -e AUTH0_DOMAIN=your-domain.auth0.com \
  -e AUTH0_AUDIENCE=https://your-api-audience \
  -e TMDB_API_KEY=your-tmdb-key \
  -e DATABASE_PATH=/app/data/moviedb.db \
  -v moviedb-data:/app/data \
  moviedb
```

### Option 3: Docker Compose

Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  moviedb:
    build: .
    container_name: moviedb
    ports:
      - "8080:8080"
    environment:
      - AUTH0_DOMAIN=${AUTH0_DOMAIN}
      - AUTH0_AUDIENCE=${AUTH0_AUDIENCE}
      - TMDB_API_KEY=${TMDB_API_KEY}
      - DATABASE_PATH=/app/data/moviedb.db
      - PORT=8080
      - ENV=production
    volumes:
      - moviedb-data:/app/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3

volumes:
  moviedb-data:
```

Run with:
```bash
docker-compose up -d
```

## Reverse Proxy Configuration

### Nginx Configuration

```nginx
server {
    listen 80;
    server_name yourdomain.com;
    
    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name yourdomain.com;
    
    # SSL configuration
    ssl_certificate /path/to/certificate.crt;
    ssl_certificate_key /path/to/private.key;
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # Proxy to Movie DB app
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
```

### Caddy Configuration

```caddyfile
yourdomain.com {
    reverse_proxy localhost:8080
    
    # Optional: Enable compression
    encode gzip
    
    # Security headers
    header {
        X-Frame-Options DENY
        X-Content-Type-Options nosniff
        X-XSS-Protection "1; mode=block"
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
    }
}
```

## Auth0 Production Configuration

### 1. Update Auth0 Application Settings
Add production URLs to your Auth0 Single Page Application:

```
Allowed Callback URLs:
http://localhost:3000, https://yourdomain.com

Allowed Logout URLs:
http://localhost:3000, https://yourdomain.com

Allowed Web Origins:
http://localhost:3000, https://yourdomain.com

Allowed Origins (CORS):
http://localhost:3000, https://yourdomain.com
```

### 2. Environment Variables
Ensure production environment variables are set correctly:
- `AUTH0_DOMAIN`: Your Auth0 tenant domain
- `AUTH0_AUDIENCE`: Your API identifier
- Frontend build includes production Auth0 client ID

## Monitoring & Maintenance

### Health Check
The application provides a health endpoint:
```bash
curl https://yourdomain.com/health
# Should return: OK
```

### Logs
- **Systemd**: `sudo journalctl -u moviedb -f`
- **Docker**: `docker logs moviedb -f`

### Database Backup
```bash
# Create backup
cp /opt/moviedb/moviedb.db /opt/moviedb/backups/moviedb-$(date +%Y%m%d-%H%M%S).db

# Automated backup script
#!/bin/bash
DATE=$(date +%Y%m%d-%H%M%S)
cp /opt/moviedb/moviedb.db /opt/moviedb/backups/moviedb-$DATE.db
find /opt/moviedb/backups -name "moviedb-*.db" -mtime +7 -delete
```

### Updates
To update the application:
1. Build new binary: `make build-prod`
2. Stop service: `sudo systemctl stop moviedb`
3. Replace binary: `cp bin/moviedb /opt/moviedb/`
4. Start service: `sudo systemctl start moviedb`

## Security Considerations

### File Permissions
```bash
# Set proper permissions
chmod 755 /opt/moviedb/moviedb
chmod 600 /opt/moviedb/.env
chown moviedb:moviedb /opt/moviedb/*
```

### Firewall
```bash
# Allow only necessary ports
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw deny 8080/tcp   # Block direct access to app
sudo ufw enable
```

### SSL Certificate
Use Let's Encrypt for free SSL certificates:
```bash
# Install certbot
sudo apt install certbot python3-certbot-nginx

# Get certificate
sudo certbot --nginx -d yourdomain.com
```

## Troubleshooting

### Common Issues

| Issue | Solution |
|-------|----------|
| **Binary won't start** | Check file permissions and environment variables |
| **Database errors** | Verify DATABASE_PATH and write permissions |
| **Auth errors** | Confirm Auth0 configuration and environment variables |
| **CORS issues** | Update Auth0 allowed origins |
| **Static files 404** | Ensure frontend was built before Go binary |

### Debug Commands
```bash
# Check application logs
sudo journalctl -u moviedb -f

# Test endpoints
curl http://localhost:8080/health
curl -H "Authorization: Bearer $TOKEN" http://localhost:8080/api/me

# Check database
ls -la /opt/moviedb/moviedb.db
```

## Performance Optimization

### 1. Enable Gzip Compression
Configure reverse proxy to compress responses.

### 2. Set Proper Cache Headers
The embedded static files include cache-friendly names.

### 3. Database Optimization
SQLite is sufficient for moderate loads. For high traffic, consider:
- Regular `VACUUM` operations
- Database connection pooling
- Read replicas if needed

The single-binary architecture makes deployment simple while maintaining good performance for typical use cases.