# Docker 部署方案

## 1. 目录结构

```text
frp-platform/
├── docker-compose.yml
├── .env
├── nginx/
│   ├── nginx.conf
│   ├── conf.d/
│   └── certs/
├── frps/
│   └── frps.toml
├── postgres/
│   └── data/
├── redis/
│   └── data/
├── mail/
│   └── data/
├── certbot/
│   ├── www/
│   └── letsencrypt/
└── logs/
```

## 2. 环境变量

```env
PLATFORM_DOMAIN=example.com
ADMIN_DOMAIN=admin.example.com
PANEL_DOMAIN=panel.example.com
API_DOMAIN=api.example.com
FRP_ENTRY_DOMAIN=frp.example.com

POSTGRES_DB=frp_platform
POSTGRES_USER=frp_platform
POSTGRES_PASSWORD=replace-with-strong-postgres-password

REDIS_PASSWORD=replace-with-strong-redis-password

FRP_BIND_PORT=7000
FRP_TOKEN=replace-with-random-32-byte-frp-token
FRP_VHOST_HTTP_PORT=8080

TCP_PORT_START=20000
TCP_PORT_END=29999
UDP_PORT_START=30000
UDP_PORT_END=39999

LETSENCRYPT_EMAIL=admin@example.com
MAIL_DOMAIN=example.com
MAIL_HOST=mail.example.com
```

## 3. docker-compose 初版

```yaml
version: "3.9"

services:
  nginx:
    image: nginx:1.27-alpine
    container_name: frp-platform-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/conf.d:/etc/nginx/conf.d
      - ./certbot/www:/var/www/certbot
      - ./certbot/letsencrypt:/etc/letsencrypt
      - ./logs/nginx:/var/log/nginx
    depends_on:
      - api-server
      - admin-web
      - user-web
      - frps
    restart: unless-stopped

  frps:
    image: snowdreamtech/frps:latest
    container_name: frp-platform-frps
    command: ["-c", "/etc/frp/frps.toml"]
    ports:
      - "7000:7000"
      - "20000-29999:20000-29999/tcp"
      - "30000-39999:30000-39999/udp"
    volumes:
      - ./frps/frps.toml:/etc/frp/frps.toml:ro
      - ./logs/frps:/var/log/frp
    restart: unless-stopped

  api-server:
    image: frp-platform/api-server:latest
    container_name: frp-platform-api
    env_file:
      - .env
    depends_on:
      - postgres
      - redis
    volumes:
      - ./nginx/conf.d:/app/runtime/nginx-conf.d
      - ./certbot/www:/var/www/certbot
      - ./certbot/letsencrypt:/etc/letsencrypt
      - ./logs/api:/app/logs
    restart: unless-stopped

  admin-web:
    image: frp-platform/admin-web:latest
    container_name: frp-platform-admin-web
    restart: unless-stopped

  user-web:
    image: frp-platform/user-web:latest
    container_name: frp-platform-user-web
    restart: unless-stopped

  postgres:
    image: postgres:16-alpine
    container_name: frp-platform-postgres
    environment:
      POSTGRES_DB: ${POSTGRES_DB}
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - ./postgres/data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    container_name: frp-platform-redis
    command: ["redis-server", "--requirepass", "${REDIS_PASSWORD}"]
    volumes:
      - ./redis/data:/data
    restart: unless-stopped

  mail-server:
    image: docker.io/mailserver/docker-mailserver:latest
    container_name: frp-platform-mail
    hostname: mail
    domainname: ${PLATFORM_DOMAIN}
    env_file:
      - .env
    volumes:
      - ./mail/data:/var/mail
      - ./mail/state:/var/mail-state
      - ./mail/config:/tmp/docker-mailserver
    ports:
      - "25:25"
      - "587:587"
      - "993:993"
    restart: unless-stopped
```

## 4. frps.toml 示例

```toml
bindPort = 7000
auth.method = "token"
auth.token = "__REPLACE_WITH_FRP_TOKEN__"

vhostHTTPPort = 8080

log.to = "/var/log/frp/frps.log"
log.level = "info"
log.maxDays = 7
```

## 5. Nginx 初始配置

```nginx
events {}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    sendfile on;
    keepalive_timeout 65;

    include /etc/nginx/conf.d/*.conf;
}
```

## 6. 基础 server 配置

```nginx
server {
    listen 80;
    server_name api.example.com;

    location / {
        proxy_pass http://api-server:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}

server {
    listen 80 default_server;
    server_name _;

    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }

    location / {
        proxy_pass http://frps:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 7. HTTPS 用户域名配置模板

```nginx
server {
    listen 443 ssl http2;
    server_name {{DOMAIN}};

    ssl_certificate /etc/letsencrypt/live/{{DOMAIN}}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/{{DOMAIN}}/privkey.pem;

    location / {
        proxy_pass http://frps:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }
}
```

## 8. 部署命令

```bash
cp .env.example .env
vim .env
docker compose pull
docker compose up -d
```

## 9. 健康检查

```bash
docker compose ps
curl http://api.example.com/health
curl http://panel.example.com
curl http://admin.example.com
```
