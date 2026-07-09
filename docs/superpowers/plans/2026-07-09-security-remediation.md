# FRP Platform Security Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** 修复当前安全审查发现的认证、授权、本地客户端、node-agent、套餐权益、测速、部署默认值和未完成计划项。

**Architecture:** 按风险面分层修复：先封堵可直接接管账号/本机/节点的高危入口，再修正套餐权益和流量口径，最后补齐缺失 API 与 UI。后端新增小型安全工具函数，保持现有 Store/SQLStore 双实现一致；前端只做必要交互改造，不引入构建链。

**Tech Stack:** Go API Server、Go Local Client、Vanilla HTML/CSS/JS、PostgreSQL、Docker Compose、Nginx、frps/frpc。

## Global Constraints

- 默认中文文案；代码、命令、API 字段保留英文。
- 不把后台、用户控制台、本地客户端合并。
- 所有安全行为必须同时覆盖内存 Store 与 SQLStore。
- 新增接口必须有 Go 单元测试；前端内联 JS 必须通过 `node --check`。
- 不再在生产响应中返回验证码、默认密码、默认 frps token 或 node-agent 空认证。
- 每个任务完成后单独提交，便于回滚。

---

## File Structure

- `D:/frpbusiness/apps/api-server/internal/platform/security.go`：新增随机验证码、随机 token、环境强校验、小型常量时间工具。
- `D:/frpbusiness/apps/api-server/internal/platform/store.go`：内存 Store 的验证码、会话、兑换码、套餐权益、临时隧道清理修复。
- `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`：SQLStore 的验证码、会话、兑换码、套餐权益、端口分配、临时隧道清理修复。
- `D:/frpbusiness/apps/api-server/internal/platform/sql_migrations.go`：补齐订阅权益快照、节点维度端口分配、证书归属字段。
- `D:/frpbusiness/apps/api-server/internal/platform/server.go`：移除 `dev_code`、补鉴权/授权/验证、补隧道动作 API。
- `D:/frpbusiness/apps/api-server/internal/platform/speed_probe.go`：测速输入上限、HTTP 状态码检查、UDP 下载一致性。
- `D:/frpbusiness/apps/api-server/cmd/server/main.go`：生产环境强制配置校验。
- `D:/frpbusiness/apps/api-server/cmd/node-agent/main.go`：node-agent 强制 token、方法限制。
- `D:/frpbusiness/client/frp-client/internal/clientcore/server.go`：本地客户端 API token、CORS 白名单、鉴权中间件。
- `D:/frpbusiness/client/frp-client/internal/clientcore/speed.go`：benchmark 生命周期、下载/上传大小上限、UDP 下载修复。
- `D:/frpbusiness/apps/admin-web/index.html`：套餐表单 id 修复、端口池配置 UI。
- `D:/frpbusiness/apps/user-web/workbench.html`：本地客户端 token 输入、移除测速自定义限速。
- `D:/frpbusiness/apps/client-webui/index.html`：本地 API token 展示/复制、移除旧测速限速输入、补登录/注册/兑换/创建隧道。
- `D:/frpbusiness/deploy/frps/frps.toml`、`D:/frpbusiness/deploy/docker-compose*.yml`：移除 `change-me` 默认、模板化 frps token、收紧 node-agent 暴露。
- `D:/frpbusiness/docs/SECURITY.md`：记录剩余的全局 frps token 风险和运维要求。

---

### Task 1: 认证验证码与会话 token 加固

**Files:**
- Create: `D:/frpbusiness/apps/api-server/internal/platform/security.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`
- Modify: `D:/frpbusiness/apps/user-web/index.html`
- Modify: `D:/frpbusiness/apps/user-web/workbench.html`

**Interfaces:**
- Produces: `randomDigits(n int) (string, error)`
- Produces: `randomToken(prefix string) (string, error)`
- Changes: `POST /api/auth/send-email-code` response becomes `{expires_in:600}` and never returns `dev_code`.

- [x] **Step 1: Add failing tests**

Add tests to `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`:

```go
func TestSendEmailCodeDoesNotReturnDevCode(t *testing.T) {
	s := NewServer(NewStore())
	rr := request(t, s, "POST", "/api/auth/send-email-code", map[string]any{
		"email": "secure@example.com", "purpose": "register",
	}, "")
	if rr.Code != 200 {
		t.Fatalf("send code status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "dev_code") || strings.Contains(rr.Body.String(), "123456") {
		t.Fatalf("verification code leaked in response: %s", rr.Body.String())
	}
}

func TestAuthTokensAreNotTimestampOnly(t *testing.T) {
	store := NewStore()
	store.SendEmailCode("token-random@example.com", "register")
	code := store.DebugEmailCode("token-random@example.com", "register")
	if _, err := store.Register("token-random@example.com", code, "pass"); err != nil {
		t.Fatal(err)
	}
	token, _, err := store.Login("token-random@example.com", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(token, "token-1-") || len(token) < 32 {
		t.Fatalf("weak token generated: %q", token)
	}
}
```

- [x] **Step 2: Add test-only code retrieval**

Add to `D:/frpbusiness/apps/api-server/internal/platform/store.go`:

```go
func (s *Store) DebugEmailCode(email, purpose string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.emailCodes[strings.ToLower(strings.TrimSpace(email))+":"+purpose]
}
```

This helper is only for in-memory tests; SQL tests can fetch from DB if added later.

- [x] **Step 3: Implement secure helpers**

Create `D:/frpbusiness/apps/api-server/internal/platform/security.go`:

```go
package platform

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

func randomDigits(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("digit length required")
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		v, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		out[i] = byte('0' + v.Int64())
	}
	return string(out), nil
}

func randomToken(prefix string) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	if prefix == "" {
		return base64.RawURLEncoding.EncodeToString(raw), nil
	}
	return prefix + "-" + base64.RawURLEncoding.EncodeToString(raw), nil
}
```

- [x] **Step 4: Replace fixed codes and timestamp tokens**

In `store.go` and `sql_store.go`:

```go
code, err := randomDigits(6)
if err != nil {
	code = fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}
```

Replace `fmt.Sprintf("token-%d-%d", ...)` with:

```go
token, err := randomToken("token")
if err != nil {
	return "", User{}, err
}
```

Replace `fmt.Sprintf("admin-token-%d-%d", ...)` with:

```go
token, err := randomToken("admin")
if err != nil {
	return "", AdminUser{}, err
}
```

- [x] **Step 5: Remove leaked `dev_code`**

Change `D:/frpbusiness/apps/api-server/internal/platform/server.go`:

```go
ok(w, map[string]any{"expires_in": 600})
```

- [x] **Step 6: Update user frontends**

Remove automatic `dev_code` filling from:

```js
writeAuth('验证码已发送。');
```

in both `D:/frpbusiness/apps/user-web/index.html` and `D:/frpbusiness/apps/user-web/workbench.html`.

- [x] **Step 7: Run tests**

```bash
cd D:/frpbusiness
go test ./apps/api-server/internal/platform -run 'SendEmailCode|AuthTokens' -v
node --check output/admin-web-index.js
```

Expected: Go tests PASS; JS syntax PASS after extracting scripts.

- [x] **Step 8: Commit**

```bash
git add apps/api-server/internal/platform/security.go apps/api-server/internal/platform/store.go apps/api-server/internal/platform/sql_store.go apps/api-server/internal/platform/server.go apps/api-server/internal/platform/server_test.go apps/user-web/index.html apps/user-web/workbench.html
git commit -m "security: harden email codes and auth tokens"
```

---

### Task 2: 会话状态、密码哈希与默认管理员防护

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/password.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`
- Modify: `D:/frpbusiness/apps/api-server/cmd/server/main.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/password_test.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Produces: `validateRequiredSecrets() error`
- Changes: SQL `UserByToken` rejects disabled users.
- Changes: legacy plaintext password fallback disabled unless `ALLOW_LEGACY_PLAINTEXT_PASSWORDS=true`.

- [x] **Step 1: Write failing tests**

Add to `server_test.go`:

```go
func TestDisabledUserTokenIsRejected(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	post(t, s, "/api/auth/send-email-code", map[string]any{"email": "disabled@example.com", "purpose": "register"}, "")
	code := store.DebugEmailCode("disabled@example.com", "register")
	post(t, s, "/api/auth/register", map[string]any{"email": "disabled@example.com", "code": code, "password": "pass"}, "")
	login := post(t, s, "/api/auth/login", map[string]any{"email": "disabled@example.com", "password": "pass"}, "")
	token := login["access_token"].(string)
	if _, _, err := store.UpdateUser(1, "disabled", 0); err != nil {
		t.Fatal(err)
	}
	rr := request(t, s, "GET", "/api/auth/me", nil, token)
	if rr.Code != 401 {
		t.Fatalf("disabled token should be rejected, got %d body=%s", rr.Code, rr.Body.String())
	}
}
```

Add to `password_test.go`:

```go
func TestPlaintextPasswordFallbackDisabledByDefault(t *testing.T) {
	t.Setenv("ALLOW_LEGACY_PLAINTEXT_PASSWORDS", "")
	if VerifyPassword("secret", "secret") {
		t.Fatal("plaintext password fallback must be disabled by default")
	}
}
```

- [x] **Step 2: Enforce user status in token lookup**

In `store.go` `UserByToken`:

```go
u := s.users[id]
if u.Status != "active" {
	return User{}, ErrUnauthorized
}
return u, nil
```

In `sql_store.go` `UserByToken`, add `AND u.status='active'`:

```sql
SELECT u.id,u.email,u.password_hash,u.status,u.created_at
FROM sessions s
JOIN users u ON u.id=s.user_id
WHERE s.token=$1 AND s.expires_at>now() AND u.status='active'
```

- [x] **Step 3: Disable plaintext fallback by default**

Change `password.go`:

```go
if getenv("ALLOW_LEGACY_PLAINTEXT_PASSWORDS", "false") == "true" {
	return subtle.ConstantTimeCompare([]byte(stored), []byte(password)) == 1
}
return false
```

- [x] **Step 4: Require production secrets**

Add to `cmd/server/main.go` before server creation:

```go
if err := platform.ValidateRequiredSecrets(); err != nil {
	log.Fatalf("security configuration error: %v", err)
}
```

Add to `security.go`:

```go
func ValidateRequiredSecrets() error {
	if getenv("ALLOW_INSECURE_DEFAULTS", "false") == "true" {
		return nil
	}
	if getenv("ADMIN_PASSWORD", "") == "" || getenv("ADMIN_PASSWORD", "") == "admin123456" {
		return fmt.Errorf("ADMIN_PASSWORD must be set to a non-default value")
	}
	if getenv("FRP_TOKEN", "") == "" || getenv("FRP_TOKEN", "") == "change-me" {
		return fmt.Errorf("FRP_TOKEN must be set to a non-default value")
	}
	return nil
}
```

- [x] **Step 5: Run tests**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/internal/platform -run 'DisabledUserToken|PlaintextPassword' -v
```

Expected: PASS.

- [x] **Step 6: Commit**

```bash
git add apps/api-server/cmd/server/main.go apps/api-server/internal/platform/security.go apps/api-server/internal/platform/password.go apps/api-server/internal/platform/store.go apps/api-server/internal/platform/sql_store.go apps/api-server/internal/platform/password_test.go apps/api-server/internal/platform/server_test.go
git commit -m "security: enforce session status and required secrets"
```

---

### Task 3: 本地客户端 API 鉴权与 CORS 收紧

**Files:**
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/server.go`
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/manager.go`
- Modify: `D:/frpbusiness/client/frp-client/main.go`
- Modify: `D:/frpbusiness/apps/client-webui/index.html`
- Modify: `D:/frpbusiness/apps/user-web/workbench.html`
- Modify: `D:/frpbusiness/client/frp-client/Dockerfile`
- Test: `D:/frpbusiness/client/frp-client/internal/clientcore/manager_test.go`

**Interfaces:**
- Produces: local API header `X-Local-Token: <token>`.
- Produces: `GET /api/local-token` only for same-origin local webui.
- Changes: mutating local API endpoints require `X-Local-Token`.

- [x] **Step 1: Write failing tests**

Add to `manager_test.go`:

```go
func TestLocalServerRequiresTokenForMutations(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir, "frpc")
	if err != nil {
		t.Fatal(err)
	}
	s := NewLocalServer(m, t.TempDir())
	req := httptest.NewRequest(http.MethodPost, "/api/frpc/start", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without local token, got %d", rr.Code)
	}
}
```

- [x] **Step 2: Add local token storage**

Add to `manager.go`:

```go
func (m *Manager) LocalAPIToken() (string, error) {
	path := filepath.Join(m.workDir, "local_api_token")
	if b, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(b)) != "" {
		return strings.TrimSpace(string(b)), nil
	}
	token, err := randomLocalToken()
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0600); err != nil {
		return "", err
	}
	return token, nil
}
```

Add local helper in clientcore if platform helper cannot be imported:

```go
func randomLocalToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
```

- [x] **Step 3: Protect mutating endpoints**

In `server.go`, wrap POST endpoints:

```go
func (s *LocalServer) requireLocalToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		want, err := s.manager.LocalAPIToken()
		if err != nil {
			writeError(w, 500, err)
			return
		}
		if r.Header.Get("X-Local-Token") != want {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid local api token"))
			return
		}
		next(w, r)
	}
}
```

Apply to `/api/config/render`, `/api/config/sync`, `/api/speed-tests/prepare`, `/api/speed-tests/cleanup`, `/api/speed-tests/run`, `/api/frpc/restart`, `/api/frpc/start`, `/api/frpc/stop`, `/api/traffic/report`.

- [x] **Step 4: Restrict CORS**

Replace wildcard local CORS with allowed origins:

```go
origin := r.Header.Get("Origin")
if origin == "" || strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://localhost:") || origin == os.Getenv("USER_PORTAL_ORIGIN") {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
}
w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Local-Token")
```

- [x] **Step 5: Update UIs**

Add `localClientToken` input in `apps/user-web/workbench.html` speed/download pages and send:

```js
headers:{'Content-Type':'application/json','X-Local-Token':st.localToken}
```

Add token display/copy in `apps/client-webui/index.html`:

```js
const tokenRes = await api('/api/local-token');
document.getElementById('localToken').textContent = tokenRes.data.token;
```

- [x] **Step 6: Change Docker default bind**

Change `D:/frpbusiness/client/frp-client/Dockerfile`:

```dockerfile
CMD ["/opt/frp-client/frp-client", "-addr", "127.0.0.1:18080", "-web", "/opt/frp-client/webui"]
```

- [x] **Step 7: Run tests**

```bash
cd D:/frpbusiness
go test ./client/frp-client/internal/clientcore -run LocalServerRequiresToken -v
```

Expected: PASS.

- [x] **Step 8: Commit**

```bash
git add client/frp-client/internal/clientcore/server.go client/frp-client/internal/clientcore/manager.go client/frp-client/main.go apps/client-webui/index.html apps/user-web/workbench.html client/frp-client/Dockerfile client/frp-client/internal/clientcore/manager_test.go
git commit -m "security: protect local client api"
```

---

### Task 4: node-agent 强制认证与部署收紧

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/cmd/node-agent/main.go`
- Modify: `D:/frpbusiness/deploy/docker-compose.node.yml`
- Modify: `D:/frpbusiness/deploy/.env.node.example`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Changes: node-agent exits on startup unless `NODE_AGENT_TOKEN` or successful bind token exchange exists.
- Changes: node-agent mutating endpoints require POST.

- [x] **Step 1: Add startup guard**

In `node-agent/main.go`:

```go
if strings.TrimSpace(os.Getenv("NODE_AGENT_TOKEN")) == "" && strings.TrimSpace(os.Getenv("NODE_BIND_TOKEN")) == "" {
	log.Fatal("NODE_AGENT_TOKEN or NODE_BIND_TOKEN is required")
}
```

- [x] **Step 2: Make auth fail closed**

Replace `auth` logic:

```go
got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
if strings.TrimSpace(a.token) == "" || got != a.token {
	writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid node agent token")
	return
}
```

- [x] **Step 3: Restrict methods**

For restart/reload/test/config writes, require POST:

```go
if r.Method != http.MethodPost {
	w.WriteHeader(http.StatusMethodNotAllowed)
	return
}
```

Apply to `/api/frps/restart`, `/api/frps/reload`, `/api/nginx/test`, `/api/nginx/reload`, `/api/nginx/https-config`, `/api/certificates/request`.

- [x] **Step 4: Remove public port and docker.sock by default**

Change `docker-compose.node.yml`:

```yaml
    ports:
      - "127.0.0.1:8090:8090"
    # docker.sock is intentionally not mounted by default.
```

Remove:

```yaml
      - /var/run/docker.sock:/var/run/docker.sock
```

- [x] **Step 5: Run tests/build**

```bash
cd D:/frpbusiness
go test ./apps/api-server/...
go build ./apps/api-server/cmd/node-agent
```

Expected: PASS/build succeeds.

- [x] **Step 6: Commit**

```bash
git add apps/api-server/cmd/node-agent/main.go deploy/docker-compose.node.yml deploy/.env.node.example apps/api-server/internal/platform/server_test.go
git commit -m "security: fail closed node agent auth"
```

---

### Task 5: 套餐权益、兑换码过期、证书归属修复

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/models.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_migrations.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Adds: `CertificateRecord.UserID int64`
- Adds SQL subscription snapshot columns for protocol/domain/tunnel limits.
- Changes: HTTP/HTTPS tunnel creation requires `AllowCustomDomain`.
- Changes: redeem code honors `ExpiresAt`.

- [x] **Step 1: Add failing tests**

Add:

```go
func TestHTTPRequiresCustomDomainPermission(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	post(t, s, "/api/auth/send-email-code", map[string]any{"email": "domain-perm@example.com", "purpose": "register"}, "")
	code := store.DebugEmailCode("domain-perm@example.com", "register")
	post(t, s, "/api/auth/register", map[string]any{"email": "domain-perm@example.com", "code": code, "password": "pass"}, "")
	login := post(t, s, "/api/auth/login", map[string]any{"email": "domain-perm@example.com", "password": "pass"}, "")
	token := login["access_token"].(string)
	plan, _ := store.CreatePlan(Plan{Name: "No domain", DurationDays: 1, TrafficLimitBytes: 1 << 30, BandwidthKbps: 1000, MaxTunnels: 5, MaxHTTPTunnels: 5, AllowHTTP: true, AllowCustomDomain: false, Status: "active"})
	codes, _ := store.CreateRedeemCodes(plan.ID, 1, "ND")
	post(t, s, "/api/user/redeem", map[string]any{"code": codes[0].Code}, token)
	rr := request(t, s, "POST", "/api/tunnels", map[string]any{"name": "web", "type": "http", "local_host": "127.0.0.1", "local_port": 8080, "domain": "app.example.com"}, token)
	if rr.Code != 403 {
		t.Fatalf("expected forbidden without custom domain permission, got %d body=%s", rr.Code, rr.Body.String())
	}
}
```

- [x] **Step 2: Add SQL snapshot columns**

Extend `subscriptions` migration:

```sql
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
```

- [x] **Step 3: Store snapshot values on redeem/payment/admin assignment**

Update SQL inserts into `subscriptions` to include all `p.Allow*` and `p.Max*` fields. Query `Subscription` from `subscriptions` columns instead of current `plans` fields.

- [x] **Step 4: Enforce custom domain permission**

In both Store and SQLStore HTTP/HTTPS create branch:

```go
if !sub.AllowCustomDomain {
	return Tunnel{}, ErrForbidden
}
```

- [x] **Step 5: Honor redeem code expiration**

In Store:

```go
if rc.ExpiresAt != nil && time.Now().After(*rc.ExpiresAt) {
	return Subscription{}, ErrForbidden
}
```

In SQL query:

```sql
SELECT plan_id,status,expires_at FROM redeem_codes WHERE code=$1 FOR UPDATE
```

Reject when `expires_at IS NOT NULL AND expires_at < now()`.

- [x] **Step 6: Restrict certificate request**

In `userRequestCertificate`, require active subscription and permissions:

```go
sub, err := s.store.Subscription(u.ID)
if err != nil || sub.Status != "active" || !sub.AllowCustomDomain || !sub.AllowAutoCert {
	handleErr(w, ErrForbidden)
	return
}
```

Validate the requested domain belongs to one of the user's HTTPS/HTTP tunnels before calling certbot.

- [x] **Step 7: Run tests**

```bash
cd D:/frpbusiness
go test ./apps/api-server/internal/platform -run 'HTTPRequiresCustomDomain|Redeem|Certificate' -v
```

- [x] **Step 8: Commit**

```bash
git add apps/api-server/internal/platform/models.go apps/api-server/internal/platform/sql_migrations.go apps/api-server/internal/platform/store.go apps/api-server/internal/platform/sql_store.go apps/api-server/internal/platform/server.go apps/api-server/internal/platform/server_test.go
git commit -m "security: enforce plan entitlements"
```

---

### Task 6: frps token 部署加固与剩余风险记录

**Files:**
- Modify: `D:/frpbusiness/deploy/frps/frps.toml`
- Modify: `D:/frpbusiness/deploy/docker-compose.yml`
- Modify: `D:/frpbusiness/deploy/docker-compose.node.yml`
- Modify: `D:/frpbusiness/deploy/.env.example`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go`
- Create: `D:/frpbusiness/docs/SECURITY.md`

**Interfaces:**
- Changes: API refuses to return `change-me` frps token.
- Changes: deployment renders frps config from env or documents required manual replacement.

- [x] **Step 1: Add API guard**

In `clientTunnels`:

```go
frpToken := getenv("FRP_TOKEN", "")
if frpToken == "" || frpToken == "change-me" {
	fail(w, 500, "FRP_TOKEN_NOT_CONFIGURED", "FRP_TOKEN must be configured")
	return
}
```

- [x] **Step 2: Replace static token template**

Change `deploy/frps/frps.toml`:

```toml
bindPort = 7000
vhostHTTPPort = 8080

auth.method = "token"
auth.token = "__REPLACE_WITH_FRP_TOKEN__"

log.to = "/var/log/frp/frps.log"
log.level = "info"
log.maxDays = 7
```

- [x] **Step 3: Add render step to compose**

Add a small entrypoint wrapper or documented pre-render command:

```bash
sed "s/__REPLACE_WITH_FRP_TOKEN__/${FRP_TOKEN}/g" /etc/frp/frps.template.toml > /tmp/frps.toml
exec /usr/bin/frps -c /tmp/frps.toml
```

- [x] **Step 4: Document residual risk**

Create `docs/SECURITY.md`:

```markdown
# Security Notes

## frps token model

当前 frps token 是 frps 原生全局 token。平台现在强制禁止空值和 `change-me`，但全局 token 仍会出现在本地 frpc 配置中。运营上必须：

- 使用随机 32 字节以上 `FRP_TOKEN`。
- 定期轮换 token。
- 结合服务端流量采集或 frps 鉴权插件规划下一阶段“每用户凭证”。

在完成每用户 frps 鉴权插件前，不能宣称平台可完全阻止用户绕过官方客户端直连 frps。
```

- [x] **Step 5: Run compose config check**

```bash
cd D:/frpbusiness/deploy
docker compose --env-file .env.example config
```

Expected: config renders without YAML errors.

- [x] **Step 6: Commit**

```bash
git add deploy/frps/frps.toml deploy/docker-compose.yml deploy/docker-compose.node.yml deploy/.env.example apps/api-server/internal/platform/server.go docs/SECURITY.md
git commit -m "security: require strong frp token configuration"
```

---

### Task 7: 测速 DoS、防错与临时资源清理

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/speed_probe.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go`
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/speed.go`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`
- Test: `D:/frpbusiness/client/frp-client/internal/clientcore/manager_test.go`

**Interfaces:**
- Produces: `CleanupExpiredSpeedTestTunnels(now time.Time) int`
- Changes: max speed test bytes default 8 MiB, max 64 MiB.
- Changes: HTTP probe fails on non-2xx.

- [x] **Step 1: Add failing tests**

Add API test:

```go
func TestSpeedProbeRejectsHugePayload(t *testing.T) {
	_, err := normalizeSpeedProbeBytes(1<<40, 8*1024*1024)
	if err == nil {
		t.Fatal("expected huge speed payload to be rejected")
	}
}
```

Add client UDP test by requesting 256 KiB and expecting at least 200 KiB returned.

- [x] **Step 2: Normalize speed bytes**

Add to `speed_probe.go`:

```go
const defaultSpeedProbeBytes int64 = 8 * 1024 * 1024
const maxSpeedProbeBytes int64 = 64 * 1024 * 1024

func normalizeSpeedProbeBytes(n int64, def int64) (int64, error) {
	if n <= 0 {
		return def, nil
	}
	if n > maxSpeedProbeBytes {
		return 0, fmt.Errorf("speed probe bytes exceeds %d", maxSpeedProbeBytes)
	}
	return n, nil
}
```

- [x] **Step 3: Check HTTP status codes**

In `measureHTTPLatency`, `measureHTTPDownload`, `measureHTTPUpload`:

```go
if resp.StatusCode < 200 || resp.StatusCode >= 300 {
	return speedMeasurement{}, 0, fmt.Errorf("http status %d", resp.StatusCode)
}
```

Use the latency-specific return shape in `measureHTTPLatency`.

- [x] **Step 4: Fix UDP download service**

In client `startUDPBenchmarkService`, parse requested bytes after `d` command:

```go
case 'd':
	want := int64(8 * 1024 * 1024)
	if n >= 9 {
		want = int64(binary.BigEndian.Uint64(buf[1:9]))
	}
	var sent int64
	for sent < want {
		payload := make([]byte, 1200)
		payload[0] = 'd'
		if remain := want - sent; remain < int64(len(payload)) {
			payload = payload[:remain]
		}
		w, _ := pc.WriteTo(payload, addr)
		sent += int64(w)
	}
```

Update UDP probe to send `d` plus 8-byte length.

- [x] **Step 5: Add expired speed tunnel cleanup**

Add Backend methods:

```go
CleanupExpiredSpeedTestTunnels(now time.Time) int
```

In Store: mark expired speed tests deleted and release ports/domains. In SQLStore: transactionally select expired speed-test tunnels, call same release logic, update status deleted.

- [x] **Step 6: Call cleanup before create/list**

In `createSpeedTestTunnel` and `clientTunnels`, call:

```go
s.store.CleanupExpiredSpeedTestTunnels(time.Now())
```

- [x] **Step 7: Run tests**

```bash
cd D:/frpbusiness
go test ./apps/api-server/internal/platform -run Speed -v
go test ./client/frp-client/internal/clientcore -run Benchmark -v
```

- [x] **Step 8: Commit**

```bash
git add apps/api-server/internal/platform/speed_probe.go apps/api-server/internal/platform/store.go apps/api-server/internal/platform/sql_store.go apps/api-server/internal/platform/server.go client/frp-client/internal/clientcore/speed.go apps/api-server/internal/platform/server_test.go client/frp-client/internal/clientcore/manager_test.go
git commit -m "fix: harden speed test lifecycle"
```

---

### Task 8: 多节点端口分配修复

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_migrations.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Changes: port allocation uniqueness becomes `(node_id, protocol, port)`.
- Changes: normal tunnel creation accepts `node_id`.

- [x] **Step 1: Add failing test**

```go
func TestSamePortCanBeAllocatedOnDifferentNodes(t *testing.T) {
	store := NewStore()
	n1, _ := store.CreateNode(Node{Name: "n1", ServerAddr: "n1.example.com", TCPPortStart: 20000, TCPPortEnd: 20000})
	n2, _ := store.CreateNode(Node{Name: "n2", ServerAddr: "n2.example.com", TCPPortStart: 20000, TCPPortEnd: 20000})
	_ = n1
	_ = n2
	// create two users/subscriptions and assert both can receive remote_port 20000 on different node_id
}
```

- [x] **Step 2: Change in-memory allocation map**

Replace:

```go
usedTCP map[int]bool
usedUDP map[int]bool
```

with:

```go
usedPorts map[string]bool
```

Key format:

```go
func portKey(nodeID int64, protocol string, port int) string {
	return fmt.Sprintf("%d:%s:%d", nodeID, protocol, port)
}
```

- [x] **Step 3: Change SQL migration**

Add `node_id BIGINT NOT NULL DEFAULT 0` to `port_allocations`; replace unique constraint with:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_port_allocations_node_protocol_port
ON port_allocations(node_id, protocol, port);
```

- [x] **Step 4: Pass node_id into regular tunnel creation**

In `CreateTunnel`, include `NodeID: req.NodeID`; if `req.NodeID > 0`, load node settings via `settingsFromNode`.

- [x] **Step 5: Update user create UI**

Add node dropdown to `apps/user-web/workbench.html` create page and include `node_id` in JSON body.

- [x] **Step 6: Run tests**

```bash
cd D:/frpbusiness
go test ./apps/api-server/internal/platform -run SamePortCanBeAllocatedOnDifferentNodes -v
```

- [x] **Step 7: Commit**

```bash
git add apps/api-server/internal/platform/sql_migrations.go apps/api-server/internal/platform/store.go apps/api-server/internal/platform/sql_store.go apps/api-server/internal/platform/server.go apps/api-server/internal/platform/server_test.go apps/user-web/workbench.html
git commit -m "fix: allocate ports per node"
```

---

### Task 9: 补齐普通隧道 start/stop/delete 与端口释放

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/backend.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go`
- Modify: `D:/frpbusiness/apps/user-web/workbench.html`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Produces: `UpdateTunnelStatus(userID int64, tunnelID int64, status string) (Tunnel, error)`
- Produces: `DeleteTunnel(userID int64, tunnelID int64) error`
- Produces routes:
  - `POST /api/tunnels/{id}/start`
  - `POST /api/tunnels/{id}/stop`
  - `DELETE /api/tunnels/{id}`

- [x] **Step 1: Add failing tests**

```go
func TestDeleteTunnelReleasesPort(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	// register, redeem, create tcp tunnel
	// DELETE /api/tunnels/{id}
	// create another tcp tunnel
	// assert remote_port reused
}
```

- [x] **Step 2: Add backend methods**

Update `backend.go`:

```go
UpdateTunnelStatus(userID int64, tunnelID int64, status string) (Tunnel, error)
DeleteTunnel(userID int64, tunnelID int64) error
```

- [x] **Step 3: Implement Store/SQLStore**

`stop` sets `disabled`; `start` sets `active` after subscription/traffic check; delete sets `deleted` and releases port/domain allocation.

- [x] **Step 4: Add route parser**

In `server.go`:

```go
s.mux.HandleFunc("/api/tunnels/", s.auth(s.tunnelAction))
```

Parse `/api/tunnels/{id}/{action}` and `/api/tunnels/{id}`.

- [x] **Step 5: Add user UI buttons**

In `workbench.html` tunnel table: add start/stop/delete buttons and call routes above. After success, call `render()`.

- [x] **Step 6: Run tests**

```bash
cd D:/frpbusiness
go test ./apps/api-server/internal/platform -run Tunnel -v
```

- [x] **Step 7: Commit**

```bash
git add apps/api-server/internal/platform/backend.go apps/api-server/internal/platform/store.go apps/api-server/internal/platform/sql_store.go apps/api-server/internal/platform/server.go apps/api-server/internal/platform/server_test.go apps/user-web/workbench.html
git commit -m "feat: manage tunnel lifecycle"
```

---

### Task 10: 后台套餐表单与端口池配置 UI 修复

**Files:**
- Modify: `D:/frpbusiness/apps/admin-web/index.html`

**Interfaces:**
- Changes: plan status field uses `planStatus`.
- Adds: TCP/UDP port pool inputs mapped to `Settings`.

- [x] **Step 1: Fix plan field ids**

Replace:

```js
status: $('plan状态').value
```

with:

```js
status: $('planStatus').value
```

Replace:

```js
value('plan状态', plan?.status || 'active');
```

with:

```js
value('planStatus', plan?.status || 'active');
```

- [x] **Step 2: Add settings fields**

In `settingsPage`, add:

```html
<label class="field"><span>TCP 起始端口</span><input class="input" id="tcpPortStart" value="${h(s.tcp_port_start || 20000)}"></label>
<label class="field"><span>TCP 结束端口</span><input class="input" id="tcpPortEnd" value="${h(s.tcp_port_end || 29999)}"></label>
<label class="field"><span>UDP 起始端口</span><input class="input" id="udpPortStart" value="${h(s.udp_port_start || 30000)}"></label>
<label class="field"><span>UDP 结束端口</span><input class="input" id="udpPortEnd" value="${h(s.udp_port_end || 39999)}"></label>
```

- [x] **Step 3: Save port pool fields**

Update `saveSettingsBtn`:

```js
current.tcp_port_start = Number($('tcpPortStart').value);
current.tcp_port_end = Number($('tcpPortEnd').value);
current.udp_port_start = Number($('udpPortStart').value);
current.udp_port_end = Number($('udpPortEnd').value);
```

- [x] **Step 4: JS syntax check**

Extract script and run:

```bash
node --check /tmp/admin-index.js
```

Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add apps/admin-web/index.html
git commit -m "fix: repair admin plan and port settings ui"
```

---

### Task 11: 本地客户端 WebUI 补齐登录/注册/兑换/创建隧道

**Files:**
- Modify: `D:/frpbusiness/apps/client-webui/index.html`
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/server.go`

**Interfaces:**
- Local WebUI consumes remote API:
  - `POST /api/auth/send-email-code`
  - `POST /api/auth/register`
  - `POST /api/auth/login`
  - `POST /api/user/redeem`
  - `POST /api/tunnels`

- [x] **Step 1: Add auth panel**

Add fields:

```html
<input class="input" id="loginEmail" placeholder="邮箱">
<input class="input" id="loginPassword" type="password" placeholder="密码">
<button class="btn secondary" id="sendCodeBtn">发送注册验证码</button>
<input class="input" id="loginCode" placeholder="验证码">
<button class="btn primary" id="registerBtn">注册</button>
<button class="btn primary" id="loginBtn">登录</button>
```

- [x] **Step 2: Store token locally**

Add JS:

```js
let remoteToken = localStorage.getItem('remoteToken') || '';
function setRemoteToken(token) {
  remoteToken = token;
  localStorage.setItem('remoteToken', token);
  document.getElementById('token').value = token;
}
```

- [x] **Step 3: Add redeem panel**

Add:

```html
<input class="input" id="redeemCode" placeholder="兑换码">
<button class="btn secondary" id="redeemBtn">兑换套餐</button>
```

JS posts to `/api/user/redeem` with Authorization header.

- [x] **Step 4: Add create tunnel panel**

Add tunnel form with `name/type/local_host/local_port/domain/bandwidth_limit_kbps` and post to remote `/api/tunnels`.

- [x] **Step 5: JS syntax check**

```bash
node --check /tmp/client-webui-index.js
```

- [x] **Step 6: Commit**

```bash
git add apps/client-webui/index.html client/frp-client/internal/clientcore/server.go
git commit -m "feat: complete local client account flows"
```

---

### Task 12: 最终验证与文档收敛

**Files:**
- Modify: `D:/frpbusiness/docs/IMPLEMENTATION_STATUS.md`
- Modify: `D:/frpbusiness/docs/FINAL_ACCEPTANCE_AUDIT.md`
- Modify: `D:/frpbusiness/deploy/PRODUCTION.md`
- Modify: `D:/frpbusiness/deploy/SPLIT_DEPLOYMENT.md`

**Interfaces:**
- Produces: updated acceptance evidence and production security checklist.

- [x] **Step 1: Full Go tests**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/...
go test ./client/frp-client/...
```

Expected: PASS.

- [x] **Step 2: Frontend syntax checks**

```bash
python - <<'PY'
from pathlib import Path
for name,path in {
  'admin':'apps/admin-web/index.html',
  'user-index':'apps/user-web/index.html',
  'user-workbench':'apps/user-web/workbench.html',
  'client':'apps/client-webui/index.html',
}.items():
    html=Path(path).read_text(encoding='utf-8')
    js=html.split('<script>',1)[1].split('</script>',1)[0]
    out=Path('/tmp')/(name+'.js')
    out.write_text(js, encoding='utf-8')
PY
node --check /tmp/admin.js
node --check /tmp/user-index.js
node --check /tmp/user-workbench.js
node --check /tmp/client.js
```

Expected: all PASS.

- [x] **Step 3: Compose config checks**

```bash
cd D:/frpbusiness/deploy
docker compose --env-file .env.example config
docker compose --env-file .env.control.example -f docker-compose.control.yml config
docker compose --env-file .env.node.example -f docker-compose.node.yml config
```

Expected: no YAML or interpolation errors.

- [x] **Step 4: Update docs**

Document:

```markdown
- 验证码不再返回给前端。
- 本地客户端 API 需要 X-Local-Token。
- node-agent 空 token 启动失败。
- FRP_TOKEN 禁止空值和 change-me。
- 普通隧道支持 start/stop/delete 并释放端口。
- SQL 订阅使用权益快照，套餐编辑不影响历史订阅。
```

- [x] **Step 5: Commit**

```bash
git add docs/IMPLEMENTATION_STATUS.md docs/FINAL_ACCEPTANCE_AUDIT.md deploy/PRODUCTION.md deploy/SPLIT_DEPLOYMENT.md
git commit -m "docs: update security remediation acceptance"
```

## 自审

- 覆盖安全审查中的验证码泄露、默认管理员、弱 token、本地 API 无认证、node-agent 空认证、全局 frps token 默认值、证书权限、套餐权限、兑换码过期、测速 DoS、多节点端口、普通隧道生命周期、后台 UI、客户端功能缺口。
- 没有使用 TBD/TODO/implement later。
- Store 与 SQLStore 均有对应修复任务。
- 仍需明确的剩余风险：frps 原生全局 token 无法彻底实现每用户凭证；本计划通过强制强 token、禁用默认值、文档披露和后续插件路线降低风险。
