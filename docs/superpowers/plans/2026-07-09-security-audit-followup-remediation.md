# Security Audit Follow-up Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 2026-07-09 对 `D:/frpbusiness` 最新 `v0.1.5` 审查发现的安全漏洞、边界情况、性能/运维风险和未完成验收项。

**Architecture:** 按风险优先级推进：先封堵凭据泄露和跨域面，再补齐会话过期、生产启动门禁、注册输入校验，最后修复本地客户端 WebUI 的 `X-Local-Token` 调用链并重写验收清单。后端保持 `Store` 与 `SQLStore` 行为一致；前端只做最小必要改动。

**Tech Stack:** Go API Server, Go local client, React/Vite frontends, PostgreSQL, Docker Compose, Nginx, frps/frpc.

## Global Constraints

- 默认文档使用中文；代码、命令、API 字段保持英文。
- 不合并后台端、用户端、本地客户端端；只修复审查指出的问题。
- 后端安全行为必须同时覆盖 `Store` 与 `SQLStore`。
- 新增/修改后端逻辑必须有 Go 单元测试；前端改动必须通过 `npm run build`。
- 不在任何用户态 API 响应中返回全局 `FRP_TOKEN`。
- 生产启动不得静默降级到内存 Store，不得默认携带演示兑换码。
- 每个任务完成后单独提交，便于回滚和审查。

---

## Audit Findings Covered

| 优先级 | 问题 | 位置 |
|---|---|---|
| P0 | 任意登录用户可通过 `/api/client/tunnels` 获取全局 `FRP_TOKEN` | `D:/frpbusiness/apps/api-server/internal/platform/server.go:508-540` |
| P0 | API 全局 `Access-Control-Allow-Origin: *` | `D:/frpbusiness/apps/api-server/internal/platform/server.go:100-104` |
| P1 | 内存 Store 用户/管理员 session 永不过期 | `D:/frpbusiness/apps/api-server/internal/platform/store.go:67-83,145-175` |
| P1 | 缺少 `DATABASE_URL` 时生产 API 自动降级内存 Store，并内置演示兑换码 | `D:/frpbusiness/apps/api-server/cmd/server/main.go:21-33`; `D:/frpbusiness/apps/api-server/internal/platform/store.go:49-60` |
| P1 | SQL 注册空密码可能触发 `mustHashPassword` panic | `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go:96-109`; `D:/frpbusiness/apps/api-server/internal/platform/password.go:64-68` |
| P2 | 弱密钥/占位值校验不完整 | `D:/frpbusiness/apps/api-server/internal/platform/security.go:37-47`; `D:/frpbusiness/deploy/.env.control.example:7,11-16,22,31` |
| P2 | React 本地客户端 WebUI 未自动携带 `X-Local-Token` | `D:/frpbusiness/apps/client-webui/src/App.jsx:12-26` |
| P3 | 验收清单仍未完成，且编码/格式损坏 | `D:/frpbusiness/docs/plans/08-ACCEPTANCE-CHECKLIST.md:1-22` |

---

## File Structure

- `D:/frpbusiness/apps/api-server/internal/platform/server.go`：调整 CORS、登录响应、`/api/client/tunnels` 响应结构。
- `D:/frpbusiness/apps/api-server/internal/platform/store.go`：给内存 session 增加过期时间；演示兑换码改为显式开发模式。
- `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go`：注册输入校验，避免请求路径 panic。
- `D:/frpbusiness/apps/api-server/internal/platform/password.go`：新增注册输入标准化函数。
- `D:/frpbusiness/apps/api-server/internal/platform/security.go`：新增弱密钥检测、CORS allowlist、生产数据库门禁。
- `D:/frpbusiness/apps/api-server/cmd/server/main.go`：生产环境要求 `DATABASE_URL`。
- `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`：新增 API 安全行为测试。
- `D:/frpbusiness/apps/api-server/internal/platform/password_test.go`：新增密码/密钥边界测试。
- `D:/frpbusiness/client/frp-client/internal/clientcore/config.go`：本地 frpc 配置不再依赖远端返回全局 token。
- `D:/frpbusiness/client/frp-client/internal/clientcore/manager.go`：本地读取 `FRP_TOKEN` 或 `FRP_CLIENT_TOKEN`。
- `D:/frpbusiness/apps/shared/frontend/api/client.js`：支持自动附加 `X-Local-Token`。
- `D:/frpbusiness/apps/client-webui/src/App.jsx`：启动时获取并保存本地 token。
- `D:/frpbusiness/apps/user-web/src/**`：检查用户端测速/本地客户端调用是否需要本地 token。
- `D:/frpbusiness/deploy/.env.example`、`D:/frpbusiness/deploy/.env.control.example`：补充强密钥与 CORS 示例。
- `D:/frpbusiness/docs/plans/08-ACCEPTANCE-CHECKLIST.md`：重写为 UTF-8 验收清单。
- `D:/frpbusiness/docs/SECURITY.md`：补充发布安全门禁。

---

### Task 1: 移除 `/api/client/tunnels` 中的全局 `FRP_TOKEN`

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go:508-540`
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/config.go`
- Modify: `D:/frpbusiness/client/frp-client/internal/clientcore/manager.go`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`
- Test: `D:/frpbusiness/client/frp-client/internal/clientcore/config_test.go`

**Interfaces:**
- `GET /api/client/tunnels` 不再返回 JSON 字段 `token`。
- 本地客户端从本机环境变量 `FRP_TOKEN` 或 `FRP_CLIENT_TOKEN` 注入 frpc 配置。

- [ ] **Step 1: 写失败测试**

在 `server_test.go` 增加：

```go
func TestClientTunnelsDoesNotExposeFRPToken(t *testing.T) {
	t.Setenv("FRP_TOKEN", "super-secret-frp-token-for-test")
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "true")
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "no-token-leak@example.com", "pass")
	post(t, s, "/api/user/redeem", map[string]any{"code": "DEMO-PLAN-2026"}, token)

	rr := request(t, s, http.MethodGet, "/api/client/tunnels", nil, token)
	if rr.Code != http.StatusOK {
		t.Fatalf("client tunnels status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "super-secret-frp-token-for-test") || strings.Contains(rr.Body.String(), `"token"`) {
		t.Fatalf("/api/client/tunnels leaked frp token: %s", rr.Body.String())
	}
}
```

- [ ] **Step 2: 确认测试失败**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/internal/platform -run ClientTunnelsDoesNotExposeFRPToken -v
```

Expected: FAIL，因为当前响应包含 `token`。

- [ ] **Step 3: 修改 API 响应**

把 `server.go:509-540` 改为不读取、不返回 `FRP_TOKEN`：

```go
ok(w, map[string]any{
	"server_addr":              st.ServerAddr,
	"server_port":              st.FRPServerPort,
	"bandwidth_limit_kbps":     bandwidth,
	"tunnels":                  tunnels,
	"requires_local_frp_token": true,
	"frp_token_delivery":       "local-client-secret",
})
```

- [ ] **Step 4: 修改本地客户端同步逻辑**

在 `manager.go` 解码远端响应后、调用 `WriteConfig` 前加入：

```go
localFRPToken := strings.TrimSpace(os.Getenv("FRP_TOKEN"))
if localFRPToken == "" {
	localFRPToken = strings.TrimSpace(os.Getenv("FRP_CLIENT_TOKEN"))
}
if localFRPToken == "" {
	return "", fmt.Errorf("FRP_TOKEN or FRP_CLIENT_TOKEN must be configured locally")
}
envelope.Data.Token = localFRPToken
return m.WriteConfig(envelope.Data)
```

- [ ] **Step 5: 加本地配置测试**

在 `config_test.go` 增加：

```go
func TestRenderFRPCConfigRequiresLocalToken(t *testing.T) {
	_, err := RenderFRPCConfig(ServerConfig{ServerAddr: "frp.example.com", ServerPort: 7000})
	if err == nil || !strings.Contains(err.Error(), "token required") {
		t.Fatalf("expected token error, got %v", err)
	}
}
```

- [ ] **Step 6: 跑测试**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/internal/platform -run ClientTunnelsDoesNotExposeFRPToken -v
go test ./client/frp-client/internal/clientcore -run RenderFRPCConfigRequiresLocalToken -v
```

- [ ] **Step 7: 提交**

```bash
git add apps/api-server/internal/platform/server.go apps/api-server/internal/platform/server_test.go client/frp-client/internal/clientcore/config.go client/frp-client/internal/clientcore/manager.go client/frp-client/internal/clientcore/config_test.go
git commit -m "security: stop exposing frp token in client tunnel api"
```

---

### Task 2: 收紧 API CORS

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go:100-110`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/security.go`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`
- Modify: `D:/frpbusiness/deploy/.env.example`
- Modify: `D:/frpbusiness/deploy/.env.control.example`

**Interfaces:**
- 新增 `allowedCORSOrigin(origin string) (string, bool)`。
- 新增环境变量 `CORS_ALLOWED_ORIGINS=https://panel.example.com,https://admin.example.com`。

- [ ] **Step 1: 写失败测试**

在 `server_test.go` 增加：

```go
func TestCORSRejectsUnconfiguredOrigin(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "false")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://panel.example.com,https://admin.example.com")
	s := NewServer(NewStore())
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected CORS origin %q", got)
	}
}
```

- [ ] **Step 2: 实现 allowlist CORS**

替换 `cors` 函数为：

```go
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin, ok := allowedCORSOrigin(r.Header.Get("Origin")); ok && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

在 `security.go` 增加：

```go
func allowedCORSOrigin(origin string) (string, bool) {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return "", true
	}
	for _, item := range strings.Split(getenv("CORS_ALLOWED_ORIGINS", ""), ",") {
		if strings.TrimSpace(item) == origin {
			return origin, true
		}
	}
	if getenv("ALLOW_INSECURE_DEFAULTS", "false") == "true" {
		if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
			return origin, true
		}
	}
	return "", false
}
```

- [ ] **Step 3: 更新 env 示例**

加入：

```dotenv
CORS_ALLOWED_ORIGINS=https://panel.example.com,https://admin.example.com
```

- [ ] **Step 4: 跑测试并提交**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/internal/platform -run CORS -v
git add apps/api-server/internal/platform/server.go apps/api-server/internal/platform/security.go apps/api-server/internal/platform/server_test.go deploy/.env.example deploy/.env.control.example
git commit -m "security: restrict api cors origins"
```

---

### Task 3: 给内存 Store 会话添加过期时间

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/server.go`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- 新增 `const sessionTTL = 24 * time.Hour`。
- `Store.sessions` / `Store.adminSessions` 改为 `map[string]sessionRecord`。

- [ ] **Step 1: 写失败测试**

```go
func TestInMemoryUserSessionExpires(t *testing.T) {
	store := NewStore()
	s := NewServer(store)
	token := registerTestUser(t, s, store, "expire-user@example.com", "pass")
	store.sessions[token] = sessionRecord{UserID: 1, ExpiresAt: time.Now().Add(-time.Second)}
	rr := request(t, s, http.MethodGet, "/api/auth/me", nil, token)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected expired user session 401, got %d body=%s", rr.Code, rr.Body.String())
	}
}
```

- [ ] **Step 2: 实现 sessionRecord**

在 `store.go` 增加：

```go
const sessionTTL = 24 * time.Hour

type sessionRecord struct {
	UserID    int64
	ExpiresAt time.Time
}
```

登录成功时写入：

```go
s.sessions[token] = sessionRecord{UserID: u.ID, ExpiresAt: time.Now().Add(sessionTTL)}
s.adminSessions[token] = sessionRecord{UserID: admin.ID, ExpiresAt: time.Now().Add(sessionTTL)}
```

读取 token 时过期即删除并返回 `ErrUnauthorized`。

- [ ] **Step 3: 登录响应使用同一 TTL**

在 `server.go` 登录响应中使用：

```go
"expires_in": int(sessionTTL.Seconds())
```

- [ ] **Step 4: 跑测试并提交**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/internal/platform -run 'InMemory.*SessionExpires|DisabledUserToken' -v
git add apps/api-server/internal/platform/store.go apps/api-server/internal/platform/server.go apps/api-server/internal/platform/server_test.go
git commit -m "security: expire in-memory sessions"
```

---

### Task 4: 生产环境禁止无数据库降级和演示兑换码

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/cmd/server/main.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/security.go`
- Create: `D:/frpbusiness/apps/api-server/cmd/server/main_test.go`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- 新增 `InsecureDefaultsAllowed() bool`。
- 新增 `RequireDatabaseURL() error`。
- 仅当 `ALLOW_INSECURE_DEFAULTS=true` 时允许内存 Store 和 `DEMO-PLAN-2026`。

- [ ] **Step 1: 写数据库门禁测试**

创建 `cmd/server/main_test.go`：

```go
package main

import (
	"testing"
	"frp-platform/apps/api-server/internal/platform"
)

func TestRequireDatabaseURLInProduction(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "false")
	t.Setenv("DATABASE_URL", "")
	if err := platform.RequireDatabaseURL(); err == nil {
		t.Fatal("expected DATABASE_URL to be required in production")
	}
}
```

- [ ] **Step 2: 实现门禁**

在 `security.go` 增加：

```go
func InsecureDefaultsAllowed() bool {
	return strings.EqualFold(strings.TrimSpace(getenv("ALLOW_INSECURE_DEFAULTS", "false")), "true")
}

func RequireDatabaseURL() error {
	if InsecureDefaultsAllowed() {
		return nil
	}
	if strings.TrimSpace(getenv("DATABASE_URL", "")) == "" {
		return fmt.Errorf("DATABASE_URL must be set unless ALLOW_INSECURE_DEFAULTS=true")
	}
	return nil
}
```

在 `main.go` 选择 backend 前调用：

```go
if err := platform.RequireDatabaseURL(); err != nil {
	log.Fatalf("storage configuration error: %v", err)
}
```

- [ ] **Step 3: 演示兑换码改为开发模式**

在 `NewStore()` 中包裹：

```go
if InsecureDefaultsAllowed() {
	s.redeemCodes["DEMO-PLAN-2026"] = RedeemCode{Code: "DEMO-PLAN-2026", PlanID: plan.ID, Status: "unused"}
}
```

- [ ] **Step 4: 跑测试并提交**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/internal/platform -run 'NewStore.*Demo|UserRedeem|Tunnel' -v
go test ./apps/api-server/cmd/server -run RequireDatabaseURL -v
git add apps/api-server/cmd/server/main.go apps/api-server/cmd/server/main_test.go apps/api-server/internal/platform/security.go apps/api-server/internal/platform/store.go apps/api-server/internal/platform/server_test.go
git commit -m "security: require database outside explicit dev mode"
```

---

### Task 5: 修复 SQL 注册空密码 panic

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/sql_store.go:96-109`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/store.go:121-137`
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/password.go`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/password_test.go`

**Interfaces:**
- 新增 `NormalizeRegistrationInput(email, password string) (string, error)`。
- 请求路径必须调用 `HashPassword` 返回错误，不调用 `mustHashPassword`。

- [ ] **Step 1: 写输入校验测试**

```go
func TestNormalizeRegistrationInputRejectsEmptyPassword(t *testing.T) {
	_, err := NormalizeRegistrationInput("user@example.com", "")
	if err == nil || err.Error() != "email and password required" {
		t.Fatalf("expected email/password required, got %v", err)
	}
}
```

- [ ] **Step 2: 实现标准化函数**

在 `password.go` 增加：

```go
func NormalizeRegistrationInput(email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", fmt.Errorf("email and password required")
	}
	return email, nil
}
```

- [ ] **Step 3: Store/SQLStore 使用同一校验**

在两处 `Register` 开头调用：

```go
email, err := NormalizeRegistrationInput(email, password)
if err != nil {
	return User{}, err
}
```

SQLStore 中把：

```go
mustHashPassword(password)
```

改为：

```go
hash, err := HashPassword(password)
if err != nil {
	return User{}, err
}
```

并将 `hash` 传入 SQL。

- [ ] **Step 4: 跑测试并提交**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/internal/platform -run 'NormalizeRegistrationInput|HashAndVerifyPassword' -v
git add apps/api-server/internal/platform/password.go apps/api-server/internal/platform/password_test.go apps/api-server/internal/platform/store.go apps/api-server/internal/platform/sql_store.go
git commit -m "fix: validate registration before hashing password"
```

---

### Task 6: 扩展弱密钥和占位值校验

**Files:**
- Modify: `D:/frpbusiness/apps/api-server/internal/platform/security.go`
- Modify: `D:/frpbusiness/deploy/.env.example`
- Modify: `D:/frpbusiness/deploy/.env.control.example`
- Test: `D:/frpbusiness/apps/api-server/internal/platform/password_test.go`

**Interfaces:**
- 新增 `isWeakSecret(name, value string) bool`。
- `ValidateRequiredSecrets()` 拒绝空值、`change-me`、`replace-with-*`、`example`、短密钥。

- [ ] **Step 1: 写弱密钥测试**

```go
func TestValidateRequiredSecretsRejectsPlaceholders(t *testing.T) {
	t.Setenv("ALLOW_INSECURE_DEFAULTS", "false")
	t.Setenv("ADMIN_PASSWORD", "replace-with-strong-admin-password")
	t.Setenv("FRP_TOKEN", "frp-token-with-at-least-32-randomish-chars")
	if err := ValidateRequiredSecrets(); err == nil {
		t.Fatal("expected weak admin password to be rejected")
	}
}
```

- [ ] **Step 2: 实现弱密钥检测**

```go
func isWeakSecret(name, value string) bool {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" || len(v) < 16 {
		return true
	}
	weakFragments := []string{"change-me", "replace-with", "example", "your-", "todo", "password", "secret", "admin123456"}
	for _, fragment := range weakFragments {
		if strings.Contains(v, fragment) {
			return true
		}
	}
	return false
}
```

在 `ValidateRequiredSecrets()` 使用该函数校验 `ADMIN_PASSWORD` 与 `FRP_TOKEN`。

- [ ] **Step 3: 更新 env 示例**

在 env 示例里标注：

```dotenv
# Generate with: openssl rand -base64 32
FRP_TOKEN=GENERATE_WITH_OPENSSL_RAND_BASE64_32
ADMIN_PASSWORD=GENERATE_UNIQUE_ADMIN_PASSWORD
```

- [ ] **Step 4: 跑测试并提交**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/internal/platform -run ValidateRequiredSecrets -v
git add apps/api-server/internal/platform/security.go apps/api-server/internal/platform/password_test.go deploy/.env.example deploy/.env.control.example
git commit -m "security: reject placeholder production secrets"
```

---

### Task 7: 修复 React 本地客户端 WebUI 的 `X-Local-Token`

**Files:**
- Modify: `D:/frpbusiness/apps/shared/frontend/api/client.js`
- Modify: `D:/frpbusiness/apps/client-webui/src/App.jsx`
- Build: `D:/frpbusiness/apps/client-webui/package.json`

**Interfaces:**
- `ApiClient` 新增 `localTokenKey`。
- 调用本地受保护 API 时自动添加 `X-Local-Token`。

- [ ] **Step 1: 扩展 ApiClient**

在 `ApiClient` constructor 增加：

```js
constructor({ baseURL = '', tokenKey = 'token', tokenPrefix = 'Bearer', local = false, localTokenKey = '' } = {}) {
  this.baseURL = String(baseURL || '').replace(/\/$/, '');
  this.tokenKey = tokenKey;
  this.tokenPrefix = tokenPrefix;
  this.local = local;
  this.localTokenKey = localTokenKey;
}
```

增加：

```js
localToken() {
  if (!this.localTokenKey) return '';
  try { return localStorage.getItem(this.localTokenKey) || ''; } catch { return ''; }
}
```

在 `request()` 中加入：

```js
const localToken = options.localToken ?? this.localToken();
if (localToken && !headers['X-Local-Token']) headers['X-Local-Token'] = localToken;
```

- [ ] **Step 2: 客户端 WebUI 启动时获取本地 token**

在 `App.jsx` 中：

```js
const api = new ApiClient({ tokenKey: 'unused_local_token', localTokenKey: 'localApiToken' });

async function ensureLocalToken() {
  const current = localSetting('localApiToken', '');
  if (current) return current;
  const res = await fetch('/api/local-token');
  const json = await res.json();
  const token = json?.data?.token || '';
  if (token) localStorage.setItem('localApiToken', token);
  return token;
}

async function request(path, options = {}) {
  await ensureLocalToken();
  return api.request(path, options);
}
```

- [ ] **Step 3: 构建验证并提交**

```bash
cd D:/frpbusiness/apps/client-webui
npm run build
cd D:/frpbusiness
git add apps/shared/frontend/api/client.js apps/client-webui/src/App.jsx apps/client-webui/dist
git commit -m "fix: send local token from client webui"
```

---

### Task 8: 检查用户端测速到本地客户端的 token 传递

**Files:**
- Inspect/Modify: `D:/frpbusiness/apps/user-web/src/App.jsx`
- Inspect/Modify: `D:/frpbusiness/apps/user-web/src/**/*.jsx`
- Inspect/Modify: `D:/frpbusiness/apps/user-web/src/**/*.js`

**Interfaces:**
- 用户端调用本地客户端 API 时必须携带 `X-Local-Token`。
- 调用远端 Master API 时不得附带 `X-Local-Token`。

- [ ] **Step 1: 定位本地客户端调用**

```bash
cd D:/frpbusiness
powershell -NoProfile -Command "Select-String -Path apps/user-web/src/**/*.jsx,apps/user-web/src/**/*.js -Pattern '127.0.0.1|localhost|18080|speed-tests|frpc|config/sync|X-Local-Token|localApiToken' -Context 2,2"
```

- [ ] **Step 2: 对本地调用加 token 字段与请求头**

如果发现本地 API 调用，增加表单字段：

```jsx
<Form.Item name="local_token" label="Local Client Token">
  <Input.Password placeholder="从本地客户端 WebUI 获取 X-Local-Token" />
</Form.Item>
```

本地 fetch 加：

```js
headers: {
  'Content-Type': 'application/json',
  'X-Local-Token': values.local_token,
}
```

- [ ] **Step 3: 构建验证并提交**

```bash
cd D:/frpbusiness/apps/user-web
npm run build
cd D:/frpbusiness
git add apps/user-web/src apps/user-web/dist
git commit -m "fix: pass local token for user speed tests"
```

---

### Task 9: 重写验收清单和安全门禁文档

**Files:**
- Modify: `D:/frpbusiness/docs/plans/08-ACCEPTANCE-CHECKLIST.md`
- Modify: `D:/frpbusiness/docs/FINAL_MEFRP_REDESIGN_ACCEPTANCE.md`
- Modify: `D:/frpbusiness/docs/SECURITY.md`

**Interfaces:**
- 验收清单 UTF-8，一项一行，带证据。
- 安全门禁必须覆盖 P0/P1 项。

- [ ] **Step 1: 重写验收清单**

用以下结构替换 `08-ACCEPTANCE-CHECKLIST.md`：

```markdown
# 验收清单

> 状态说明：`[x]` 已验证，`[ ]` 未验证，`[-]` 不适用或被后续计划替代。

## 1. 部署验收

- [x] 用户端入口 `http://192.168.110.56:18188` 返回 HTTP 200。证据：2026-07-09 `Invoke-WebRequest`。
- [x] 后台端入口 `http://192.168.110.56:18189` 返回 HTTP 200。证据：2026-07-09 `Invoke-WebRequest`。
- [x] 用户端/后台端构建 asset SHA256 与本地 `v0.1.5` 一致。
- [x] `/health` 通过前端 Nginx 代理返回 `status=ok`。
- [ ] 后端容器二进制 SHA256 与本地 `dist/fnos/api-server` 一致。

## 2. 安全验收

- [ ] `/api/client/tunnels` 不返回 `FRP_TOKEN` 或 JSON 字段 `token`。
- [ ] 生产环境 CORS 只允许 `CORS_ALLOWED_ORIGINS`。
- [ ] 内存 Store 会话 24 小时后过期。
- [ ] 生产环境缺少 `DATABASE_URL` 启动失败。
- [ ] 空密码注册返回错误，不触发 panic。
- [ ] 弱占位密钥启动失败。
- [ ] 本地客户端 WebUI 调用本地受保护 API 自动携带 `X-Local-Token`。
```

- [ ] **Step 2: 更新安全门禁**

在 `SECURITY.md` 增加：

```markdown
## Release Gate

生产发布必须满足：

1. `/api/client/tunnels` 不返回全局 `FRP_TOKEN`。
2. `CORS_ALLOWED_ORIGINS` 已配置且不使用 `*`。
3. `DATABASE_URL` 已配置；除显式 `ALLOW_INSECURE_DEFAULTS=true` 外不得使用内存 Store。
4. 所有本地客户端写操作均要求 `X-Local-Token`。
5. 验收清单中安全验收项全部为 `[x]`。
```

- [ ] **Step 3: 提交**

```bash
git add docs/plans/08-ACCEPTANCE-CHECKLIST.md docs/FINAL_MEFRP_REDESIGN_ACCEPTANCE.md docs/SECURITY.md
git commit -m "docs: add security follow-up acceptance gates"
```

---

### Task 10: 全量验证和飞牛部署确认

**Files:**
- Evidence target: `D:/frpbusiness/docs/FINAL_MEFRP_REDESIGN_ACCEPTANCE.md`

**Interfaces:**
- 输出本地 commit、测试结果、构建结果、飞牛部署 hash 证据。

- [ ] **Step 1: 后端测试**

```bash
cd D:/frpbusiness
ALLOW_INSECURE_DEFAULTS=true go test ./apps/api-server/...
go test ./client/frp-client/...
```

Expected: PASS。若本机没有 Go，则在飞牛或构建容器执行并记录输出。

- [ ] **Step 2: 前端构建**

```bash
cd D:/frpbusiness/apps/user-web && npm run build
cd D:/frpbusiness/apps/admin-web && npm run build
cd D:/frpbusiness/apps/client-webui && npm run build
```

- [ ] **Step 3: Compose 校验**

```bash
cd D:/frpbusiness/deploy
docker compose --env-file .env.example config
docker compose --env-file .env.control.example -f docker-compose.control.yml config
docker compose --env-file .env.example -f docker-compose.fnos.yml config
```

- [ ] **Step 4: 飞牛后端 hash 确认**

在飞牛执行：

```bash
cd /root/frp-platform
git rev-parse HEAD
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'
docker exec frp-fnos-api sha256sum /app/api-server
```

Expected: commit 为当前修复 commit；容器内 `/app/api-server` SHA256 与本地对应发布包一致。

- [ ] **Step 5: HTTP 健康检查**

```powershell
Invoke-WebRequest -UseBasicParsing http://192.168.110.56:18188/health
Invoke-WebRequest -UseBasicParsing http://192.168.110.56:18189/health
```

- [ ] **Step 6: 记录证据并提交**

在 `FINAL_MEFRP_REDESIGN_ACCEPTANCE.md` 追加：

```markdown
## Final Security Follow-up Evidence

- Local commit: `<commit>`
- API tests: PASS
- Client tests: PASS
- User/Admin/Client builds: PASS
- fnOS backend hash: `<sha256>`
- fnOS health checks: PASS
```

提交：

```bash
git add docs/FINAL_MEFRP_REDESIGN_ACCEPTANCE.md apps/user-web/dist apps/admin-web/dist apps/client-webui/dist
git commit -m "chore: record security follow-up verification"
```

---

## Self-Review

- 覆盖范围：已覆盖 `FRP_TOKEN` 泄露、CORS、内存 session 过期、生产 DB 门禁、SQL 空密码 panic、弱密钥占位值、本地客户端 `X-Local-Token`、验收清单未完成。
- 无占位步骤：每个任务都有明确文件、测试、修改片段和验证命令。
- 类型一致性：`sessionRecord`、`sessionTTL`、`NormalizeRegistrationInput`、`RequireDatabaseURL`、`InsecureDefaultsAllowed`、`isWeakSecret` 均在定义后使用。
- 剩余架构风险：frps 原生全局 token 模型仍需后续通过 frps 鉴权插件或每用户凭据隔离彻底解决；本计划先停止通过用户 API 分发全局 token。
