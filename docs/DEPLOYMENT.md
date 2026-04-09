# Arkheion — Deployment Guide

This guide covers deploying Arkheion on a Linux server using Docker Compose, with nginx as a reverse proxy and Let's Encrypt for SSL.

---

## Prerequisites

- A Linux server (Ubuntu 22.04 LTS recommended)
- Docker Engine 24+ and Docker Compose v2
- A domain name pointing to your server's IP
- Ports 80 and 443 open in your firewall
- At least 4 GB RAM (8 GB recommended)

---

## Server Setup

### 1. Install Docker

```bash
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker
```

Verify:
```bash
docker --version
docker compose version
```

### 2. Clone Arkheion

```bash
git clone https://github.com/mathornton01/arkheion.git /opt/arkheion
cd /opt/arkheion
```

### 3. Configure Environment

```bash
cp .env.example .env
nano .env
```

**Critical variables to set:**

```bash
# Generate a strong API key
ARKHEION_API_KEYS=$(openssl rand -hex 32)

# Strong PostgreSQL password
POSTGRES_PASSWORD=$(openssl rand -base64 24)

# Update DATABASE_URL with the password above
DATABASE_URL=postgres://arkheion:YOURPASSWORD@postgres:5432/arkheion?sslmode=disable

# Strong Meilisearch key
MEILISEARCH_MASTER_KEY=$(openssl rand -hex 32)

# Strong MinIO secret
MINIO_SECRET_KEY=$(openssl rand -base64 24)

# Strong webhook secret
WEBHOOK_DEFAULT_SECRET=$(openssl rand -hex 32)

# Your domain
CORS_ALLOWED_ORIGINS=https://arkheion.example.com
MINIO_PUBLIC_URL=https://arkheion.example.com
```

### 4. Start the Stack

```bash
docker compose up -d
```

Check all services started:
```bash
docker compose ps
```

### 5. Run Database Migrations

```bash
./scripts/migrate.sh
```

### 6. Verify Backend Health

```bash
curl http://localhost:8080/api/v1/health
# {"status":"ok","service":"arkheion-backend"}
```

---

## Nginx Reverse Proxy

### Install Nginx and Certbot

```bash
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx
```

### Configure Nginx

Create `/etc/nginx/sites-available/arkheion`:

```nginx
# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name arkheion.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name arkheion.example.com;

    # SSL — managed by Certbot
    ssl_certificate     /etc/letsencrypt/live/arkheion.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/arkheion.example.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    # Security headers
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Frontend — SvelteKit
    location / {
        proxy_pass         http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade $http_upgrade;
        proxy_set_header   Connection "upgrade";
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_read_timeout 60s;
    }

    # Backend API — Go/Fiber
    location /api/ {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;

        # Allow large file uploads (500MB)
        client_max_body_size 500M;
        proxy_request_buffering off;    # Stream uploads directly
        proxy_read_timeout 300s;        # Allow long extractions
        proxy_send_timeout 300s;
    }
}
```

Enable and test:
```bash
sudo ln -s /etc/nginx/sites-available/arkheion /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### Obtain SSL Certificate

```bash
sudo certbot --nginx -d arkheion.example.com
```

Certbot auto-renews. Verify renewal works:
```bash
sudo certbot renew --dry-run
```

---

## Environment Variable Reference

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `ARKHEION_API_KEYS` | Yes | Comma-separated API keys | `abc123,xyz789` |
| `DATABASE_URL` | Yes | PostgreSQL connection string | `postgres://user:pass@host:5432/db` |
| `POSTGRES_PASSWORD` | Yes | PostgreSQL password | (generated) |
| `MEILISEARCH_MASTER_KEY` | Yes | Meilisearch admin key | (generated) |
| `MINIO_SECRET_KEY` | Yes | MinIO secret key | (generated) |
| `CORS_ALLOWED_ORIGINS` | Yes | Allowed browser origins | `https://example.com` |
| `WEBHOOK_DEFAULT_SECRET` | Recommended | HMAC signing secret | (generated) |
| `GOOGLE_BOOKS_API_KEY` | Optional | Google Books API key | `AIza...` |
| `LOG_LEVEL` | Optional | debug/info/warn/error | `info` |
| `TIKA_TIMEOUT_SECONDS` | Optional | Extraction timeout | `120` |
| `BACKEND_PORT` | Optional | Backend listen port | `8080` |
| `FRONTEND_PORT` | Optional | Frontend listen port | `3000` |

---

## Resource Recommendations

| Service | Min RAM | Recommended |
|---------|---------|-------------|
| Backend (Go) | 64 MB | 256 MB |
| Frontend (SvelteKit) | 64 MB | 128 MB |
| PostgreSQL | 256 MB | 1 GB |
| Meilisearch | 512 MB | 2 GB |
| Apache Tika | 512 MB | 2 GB |
| MinIO | 128 MB | 512 MB |
| **Total** | **~1.5 GB** | **~6 GB** |

Meilisearch and Tika are the largest consumers. For collections over 5,000 books with full text, allocate 4–8 GB to Meilisearch.

Set Docker resource limits in `docker-compose.yml`:

```yaml
services:
  meilisearch:
    deploy:
      resources:
        limits:
          memory: 4G
  tika:
    deploy:
      resources:
        limits:
          memory: 2G
```

---

## Backup and Restore

### Automated Backup (Cron)

```bash
# Add to crontab
crontab -e

# Daily backup at 2 AM
0 2 * * * /opt/arkheion/scripts/backup.sh >> /var/log/arkheion-backup.log 2>&1
```

### Manual Backup

```bash
./scripts/backup.sh
```

Backups are stored in `./backups/<timestamp>/`.

### Restore PostgreSQL

```bash
zcat backups/20240101_020000/arkheion_20240101_020000.sql.gz | \
  psql "$DATABASE_URL"
```

### Restore MinIO

Use the MinIO console at `http://localhost:9001` to upload objects, or use `mc`:

```bash
mc mirror backups/20240101_020000/minio/ local/arkheion
```

---

## Updating Arkheion

```bash
cd /opt/arkheion
git pull origin main
docker compose pull
docker compose up -d --remove-orphans
./scripts/migrate.sh   # Run any new migrations
```

---

## Monitoring

### View Logs

```bash
# All services
docker compose logs -f

# Backend only
docker compose logs -f arkheion-backend

# Error level only
docker compose logs arkheion-backend | grep '"level":"error"'
```

### Health Checks

```bash
# Backend
curl http://localhost:8080/api/v1/health

# Meilisearch
curl http://localhost:7700/health

# PostgreSQL
docker compose exec postgres pg_isready
```

---

## Firewall Configuration

Only expose ports 80 and 443 publicly. All other ports (PostgreSQL 5432, Meilisearch 7700, MinIO 9000/9001) should be accessible only on localhost or via Docker network.

```bash
# Ubuntu UFW example
sudo ufw allow ssh
sudo ufw allow http
sudo ufw allow https
sudo ufw enable
```

The `docker-compose.yml` only exposes port 9001 (MinIO console) bound to `127.0.0.1`, so it is not publicly accessible by default.

---

## Troubleshooting

**Backend fails to start:**
```bash
docker compose logs arkheion-backend
# Check: DATABASE_URL, MEILISEARCH_MASTER_KEY, MINIO_SECRET_KEY are set
```

**Migrations fail:**
```bash
# Run psql directly to check connectivity
docker compose exec postgres psql -U arkheion -d arkheion -c "\dt"
```

**Text extraction is slow:**
- Tika is CPU-bound. Extraction of a 400-page PDF takes 30–120 seconds on a single core.
- Allocate more CPU to the Tika container.

**Meilisearch returns stale results:**
```bash
# Force re-index of all books
curl -X POST http://localhost:8080/api/v1/admin/reindex \
  -H "X-API-Key: your-key"
```
