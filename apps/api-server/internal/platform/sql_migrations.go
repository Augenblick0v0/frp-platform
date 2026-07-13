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
    price_cents BIGINT NOT NULL DEFAULT 0,
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
ALTER TABLE IF EXISTS plans ADD COLUMN IF NOT EXISTS price_cents BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS payment_orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    provider VARCHAR(32) NOT NULL,
    out_trade_no VARCHAR(64) NOT NULL UNIQUE,
    provider_trade_no VARCHAR(128),
    pay_type VARCHAR(32),
    name VARCHAR(255) NOT NULL,
    money VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_payment_orders_user ON payment_orders(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS subscriptions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    plan_id BIGINT NOT NULL REFERENCES plans(id),
    plan_name VARCHAR(100) NOT NULL DEFAULT '',
    starts_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    traffic_limit_bytes BIGINT NOT NULL,
    traffic_used_bytes BIGINT NOT NULL DEFAULT 0,
    bandwidth_limit_kbps INTEGER NOT NULL,
    allow_tcp BOOLEAN NOT NULL DEFAULT false,
    allow_udp BOOLEAN NOT NULL DEFAULT false,
    allow_http BOOLEAN NOT NULL DEFAULT false,
    allow_https BOOLEAN NOT NULL DEFAULT false,
    allow_custom_domain BOOLEAN NOT NULL DEFAULT false,
    allow_auto_cert BOOLEAN NOT NULL DEFAULT false,
    max_tunnels INTEGER NOT NULL DEFAULT 0,
    max_tcp_tunnels INTEGER NOT NULL DEFAULT 0,
    max_udp_tunnels INTEGER NOT NULL DEFAULT 0,
    max_http_tunnels INTEGER NOT NULL DEFAULT 0,
    max_https_tunnels INTEGER NOT NULL DEFAULT 0,
    max_domains INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user ON subscriptions(user_id);
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS plan_name VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS allow_tcp BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS allow_udp BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS allow_http BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS allow_https BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS allow_custom_domain BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS allow_auto_cert BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS max_tunnels INTEGER NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS max_tcp_tunnels INTEGER NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS max_udp_tunnels INTEGER NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS max_http_tunnels INTEGER NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS max_https_tunnels INTEGER NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS subscriptions ADD COLUMN IF NOT EXISTS max_domains INTEGER NOT NULL DEFAULT 0;
UPDATE subscriptions sub SET
    plan_name = COALESCE(NULLIF(sub.plan_name,''), p.name),
    allow_tcp = p.allow_tcp,
    allow_udp = p.allow_udp,
    allow_http = p.allow_http,
    allow_https = p.allow_https,
    allow_custom_domain = p.allow_custom_domain,
    allow_auto_cert = p.allow_auto_cert,
    max_tunnels = p.max_tunnels,
    max_tcp_tunnels = p.max_tcp_tunnels,
    max_udp_tunnels = p.max_udp_tunnels,
    max_http_tunnels = p.max_http_tunnels,
    max_https_tunnels = p.max_https_tunnels,
    max_domains = p.max_domains
FROM plans p
WHERE sub.plan_id=p.id AND sub.plan_name='';

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
    node_id BIGINT,
    client_id BIGINT,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(16) NOT NULL,
    local_host VARCHAR(255) NOT NULL,
    local_port INTEGER NOT NULL,
    remote_port INTEGER,
    domain VARCHAR(255) UNIQUE,
    use_https BOOLEAN NOT NULL DEFAULT false,
    bandwidth_limit_kbps INTEGER NOT NULL DEFAULT 0,
    speed_test BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'created',
    public_url TEXT,
    error_message TEXT,
    last_online_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_tunnels_user ON tunnels(user_id);
ALTER TABLE IF EXISTS tunnels ADD COLUMN IF NOT EXISTS node_id BIGINT;
ALTER TABLE IF EXISTS tunnels ADD COLUMN IF NOT EXISTS bandwidth_limit_kbps INTEGER NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS tunnels ADD COLUMN IF NOT EXISTS speed_test BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE IF EXISTS tunnels ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_tunnels_speed_test_expires ON tunnels(speed_test, expires_at);

CREATE TABLE IF NOT EXISTS port_allocations (
    id BIGSERIAL PRIMARY KEY,
    node_id BIGINT NOT NULL DEFAULT 0,
    protocol VARCHAR(8) NOT NULL,
    port INTEGER NOT NULL,
    tunnel_id BIGINT REFERENCES tunnels(id),
    user_id BIGINT REFERENCES users(id),
    status VARCHAR(32) NOT NULL DEFAULT 'allocated',
    allocated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at TIMESTAMPTZ,
    UNIQUE(node_id, protocol, port)
);
ALTER TABLE IF EXISTS port_allocations ADD COLUMN IF NOT EXISTS node_id BIGINT NOT NULL DEFAULT 0;
ALTER TABLE IF EXISTS port_allocations DROP CONSTRAINT IF EXISTS port_allocations_protocol_port_key;
CREATE UNIQUE INDEX IF NOT EXISTS idx_port_allocations_node_protocol_port ON port_allocations(node_id, protocol, port);

CREATE TABLE IF NOT EXISTS certificates (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id),
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
ALTER TABLE IF EXISTS certificates ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id);

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
    node_kind VARCHAR(32) NOT NULL DEFAULT 'frps',
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
    nat_provider VARCHAR(32),
    nat_instance_id TEXT,
    nat_instance_name TEXT,
    nat_entry_host TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    last_seen_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE IF EXISTS nodes ADD COLUMN IF NOT EXISTS node_kind VARCHAR(32) NOT NULL DEFAULT 'frps';
ALTER TABLE IF EXISTS nodes ADD COLUMN IF NOT EXISTS nat_provider VARCHAR(32);
ALTER TABLE IF EXISTS nodes ADD COLUMN IF NOT EXISTS nat_instance_id TEXT;
ALTER TABLE IF EXISTS nodes ADD COLUMN IF NOT EXISTS nat_instance_name TEXT;
ALTER TABLE IF EXISTS nodes ADD COLUMN IF NOT EXISTS nat_entry_host TEXT;
CREATE INDEX IF NOT EXISTS idx_nodes_status ON nodes(status);

CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(128) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`
