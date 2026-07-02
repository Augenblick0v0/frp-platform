# 数据库设计

## 1. 设计原则

- PostgreSQL 作为主数据库。
- 所有用户资源必须带 user_id。
- 隧道、域名、证书、端口分配分表管理。
- 兑换码使用后不可重复使用。
- 端口分配需要唯一约束。
- 域名需要唯一约束。

## 2. 表清单

```text
admin_users
admin_operation_logs
users
email_verification_codes
plans
subscriptions
redeem_codes
clients
tunnels
domains
certificates
port_allocations
traffic_logs
mail_logs
system_settings
nginx_reload_logs
```

## 3. 核心表结构

### 3.1 users

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 3.2 email_verification_codes

```sql
CREATE TABLE email_verification_codes (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    code VARCHAR(16) NOT NULL,
    purpose VARCHAR(32) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_ip VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_email_codes_email ON email_verification_codes(email);
```

### 3.3 plans

```sql
CREATE TABLE plans (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    duration_days INTEGER NOT NULL,
    traffic_limit_bytes BIGINT NOT NULL,
    bandwidth_limit_kbps INTEGER NOT NULL,
    max_tunnels INTEGER NOT NULL,
    max_tcp_tunnels INTEGER NOT NULL,
    max_udp_tunnels INTEGER NOT NULL,
    max_http_tunnels INTEGER NOT NULL,
    max_https_tunnels INTEGER NOT NULL,
    allow_tcp BOOLEAN NOT NULL DEFAULT false,
    allow_udp BOOLEAN NOT NULL DEFAULT false,
    allow_http BOOLEAN NOT NULL DEFAULT false,
    allow_https BOOLEAN NOT NULL DEFAULT false,
    allow_custom_domain BOOLEAN NOT NULL DEFAULT false,
    max_domains INTEGER NOT NULL DEFAULT 0,
    allow_auto_cert BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 3.4 subscriptions

```sql
CREATE TABLE subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    traffic_limit_bytes BIGINT NOT NULL,
    traffic_used_bytes BIGINT NOT NULL DEFAULT 0,
    bandwidth_limit_kbps INTEGER NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_subscriptions_user ON subscriptions(user_id);
```

### 3.5 redeem_codes

```sql
CREATE TABLE redeem_codes (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    status VARCHAR(32) NOT NULL DEFAULT 'unused',
    expires_at TIMESTAMPTZ,
    redeemed_by_user_id BIGINT REFERENCES users(id),
    redeemed_at TIMESTAMPTZ,
    remark TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 3.6 clients

```sql
CREATE TABLE clients (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    device_id VARCHAR(128) NOT NULL,
    device_name VARCHAR(255),
    os VARCHAR(64) NOT NULL,
    version VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'offline',
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, device_id)
);
```

### 3.7 tunnels

```sql
CREATE TABLE tunnels (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    client_id BIGINT REFERENCES clients(id),
    name VARCHAR(100) NOT NULL,
    type VARCHAR(16) NOT NULL,
    local_host VARCHAR(255) NOT NULL,
    local_port INTEGER NOT NULL,
    remote_port INTEGER,
    domain_id BIGINT,
    use_https BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    error_message TEXT,
    last_online_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_tunnels_user ON tunnels(user_id);
CREATE INDEX idx_tunnels_type ON tunnels(type);
```

### 3.8 domains

```sql
CREATE TABLE domains (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    domain VARCHAR(255) NOT NULL UNIQUE,
    target_cname VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    cname_checked_at TIMESTAMPTZ,
    tunnel_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 3.9 certificates

```sql
CREATE TABLE certificates (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    domain_id BIGINT NOT NULL REFERENCES domains(id),
    domain VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'none',
    issued_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    cert_path TEXT,
    key_path TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 3.10 port_allocations

```sql
CREATE TABLE port_allocations (
    id BIGSERIAL PRIMARY KEY,
    protocol VARCHAR(8) NOT NULL,
    port INTEGER NOT NULL,
    tunnel_id BIGINT REFERENCES tunnels(id),
    user_id BIGINT REFERENCES users(id),
    status VARCHAR(32) NOT NULL DEFAULT 'allocated',
    allocated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at TIMESTAMPTZ,
    UNIQUE(protocol, port)
);
```

### 3.11 traffic_logs

```sql
CREATE TABLE traffic_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    tunnel_id BIGINT REFERENCES tunnels(id),
    bytes_in BIGINT NOT NULL DEFAULT 0,
    bytes_out BIGINT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_traffic_user_time ON traffic_logs(user_id, recorded_at);
```

### 3.12 system_settings

```sql
CREATE TABLE system_settings (
    key VARCHAR(128) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```
