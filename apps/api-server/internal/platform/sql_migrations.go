package platform

const postgresSchema = `
CREATE TABLE IF NOT EXISTS admin_users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS admin_sessions (
    token TEXT PRIMARY KEY,
    admin_user_id BIGINT NOT NULL REFERENCES admin_users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS admin_operation_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_id BIGINT,
    admin_email VARCHAR(255),
    action VARCHAR(128) NOT NULL,
    target TEXT,
    detail TEXT,
    ip VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_admin_operation_logs_created ON admin_operation_logs(created_at DESC);

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    email_verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS email_verification_codes (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    code VARCHAR(16) NOT NULL,
    purpose VARCHAR(32) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_ip VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_email_codes_email ON email_verification_codes(email);

CREATE TABLE IF NOT EXISTS plans (
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

CREATE TABLE IF NOT EXISTS subscriptions (
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
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);

CREATE TABLE IF NOT EXISTS redeem_codes (
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

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tunnels (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    client_id BIGINT,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(16) NOT NULL,
    local_host VARCHAR(255) NOT NULL,
    local_port INTEGER NOT NULL,
    remote_port INTEGER,
    domain VARCHAR(255) UNIQUE,
    use_https BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    public_url TEXT,
    error_message TEXT,
    last_online_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tunnels_user ON tunnels(user_id);

CREATE TABLE IF NOT EXISTS port_allocations (
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

CREATE TABLE IF NOT EXISTS certificates (
    id BIGSERIAL PRIMARY KEY,
    domain VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL DEFAULT 'none',
    issued_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    cert_path TEXT,
    key_path TEXT,
    last_command TEXT,
    last_output TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS traffic_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    tunnel_id BIGINT REFERENCES tunnels(id),
    bytes_in BIGINT NOT NULL DEFAULT 0,
    bytes_out BIGINT NOT NULL DEFAULT 0,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_traffic_user_time ON traffic_logs(user_id, recorded_at);


CREATE TABLE IF NOT EXISTS nodes (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    agent_url TEXT,
    agent_token TEXT NOT NULL,
    bind_token TEXT NOT NULL UNIQUE,
    public_url TEXT,
    frp_entry_domain VARCHAR(255),
    server_addr VARCHAR(255),
    frp_server_port INTEGER NOT NULL DEFAULT 7000,
    tcp_port_start INTEGER NOT NULL DEFAULT 20000,
    tcp_port_end INTEGER NOT NULL DEFAULT 29999,
    udp_port_start INTEGER NOT NULL DEFAULT 30000,
    udp_port_end INTEGER NOT NULL DEFAULT 39999,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    last_seen_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);

CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(128) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`
