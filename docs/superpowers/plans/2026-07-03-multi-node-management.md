# Multi-Node Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build admin-managed multi-node support where every frps node can bind itself to the control plane and be remotely managed from the backend.

**Architecture:** Add a `Node` domain model persisted in both memory store and PostgreSQL. The control-plane API owns node inventory and per-node remote management; each node runs `node-agent`, which can self-bind to the control plane using a generated bind token and then receive remote management calls through its agent URL/token. Existing single-node environment mode remains as fallback.

**Tech Stack:** Go 1.19 API server, net/http, PostgreSQL migrations via inline schema, static Cloudflare-style admin WebUI, Docker Compose control/node deployments.

## Global Constraints

- Keep existing single-node and split deployment working.
- Do not require Kubernetes or external service discovery.
- Use token-based node binding and token-based node-agent remote management.
- Store node metadata in PostgreSQL and memory store.
- Keep APIs simple JSON over HTTP.
- Commit after implementation and push to GitHub master.

---

## File Structure

- Modify `apps/api-server/internal/platform/models.go`: add `Node` and node request/response structs.
- Modify `apps/api-server/internal/platform/backend.go`: add node inventory and bind methods.
- Modify `apps/api-server/internal/platform/store.go`: implement in-memory nodes.
- Modify `apps/api-server/internal/platform/sql_migrations.go`: add `nodes` table.
- Modify `apps/api-server/internal/platform/sql_store.go`: implement PostgreSQL node CRUD/bind.
- Modify `apps/api-server/internal/platform/server.go`: add admin node endpoints and public node bind endpoint.
- Modify `apps/api-server/internal/platform/node_agent_client.go`: add reusable constructor for per-node URL/token.
- Modify `apps/api-server/cmd/node-agent/main.go`: add optional self-bind loop to control plane.
- Modify `apps/admin-web/index.html`: add node list, creation form, bind token display, and per-node action buttons.
- Modify `deploy/.env.node.example`: add `CONTROL_PLANE_URL`, `NODE_BIND_TOKEN`, `NODE_NAME`, `NODE_PUBLIC_AGENT_URL`.
- Modify `deploy/SPLIT_DEPLOYMENT.md`: document node creation, node binding, and firewall requirements.

---

### Task 1: Node Domain Model and Store Interface

**Files:**
- Modify: `apps/api-server/internal/platform/models.go`
- Modify: `apps/api-server/internal/platform/backend.go`

**Interfaces:**
- Produces: `type Node struct`, `CreateNode(Node) (Node, error)`, `Nodes() []Node`, `Node(id int64) (Node, error)`, `BindNode(NodeBindRequest) (Node, error)`, `UpdateNodeStatus(id int64, status string, err string) (Node, error)`.

- [ ] **Step 1: Add model**

```go
type Node struct {
    ID int64 `json:"id"`
    Name string `json:"name"`
    AgentURL string `json:"agent_url"`
    AgentToken string `json:"-"`
    BindToken string `json:"bind_token,omitempty"`
    PublicURL string `json:"public_url"`
    FRPEntryDomain string `json:"frp_entry_domain"`
    ServerAddr string `json:"server_addr"`
    FRPServerPort int `json:"frp_server_port"`
    TCPPortStart int `json:"tcp_port_start"`
    TCPPortEnd int `json:"tcp_port_end"`
    UDPPortStart int `json:"udp_port_start"`
    UDPPortEnd int `json:"udp_port_end"`
    Status string `json:"status"`
    LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
    LastError string `json:"last_error,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

- [ ] **Step 2: Add Backend methods**

```go
Nodes() []Node
Node(id int64) (Node, error)
CreateNode(node Node) (Node, error)
BindNode(req NodeBindRequest) (Node, error)
UpdateNodeStatus(id int64, status string, lastError string) (Node, error)
```

- [ ] **Step 3: Run tests**

Run: `go test ./apps/api-server/...`

Expected: compile errors until store implementations are done.

---

### Task 2: In-Memory Node Store

**Files:**
- Modify: `apps/api-server/internal/platform/store.go`

**Interfaces:**
- Consumes: Node methods from Task 1.
- Produces: working memory implementation for tests/dev mode.

- [ ] **Step 1: Add fields**

```go
nodes map[int64]Node
nodesByBindToken map[string]int64
nextNodeID int64
```

- [ ] **Step 2: Implement CreateNode**

Generate missing tokens as `node-bind-<id>-<unixnano>` and `node-agent-<id>-<unixnano>`, default status `pending`.

- [ ] **Step 3: Implement BindNode**

Find by bind token, update `AgentURL`, `PublicURL`, `LastSeenAt`, `Status=online`, and any supplied domain/port metadata.

- [ ] **Step 4: Run tests**

Run: `go test ./apps/api-server/internal/platform -run Node -v`

Expected: PASS after tests are added in Task 5.

---

### Task 3: PostgreSQL Nodes Table and SQLStore

**Files:**
- Modify: `apps/api-server/internal/platform/sql_migrations.go`
- Modify: `apps/api-server/internal/platform/sql_store.go`

**Interfaces:**
- Consumes: Backend node methods.
- Produces: persistent multi-node inventory.

- [ ] **Step 1: Add `nodes` table**

```sql
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
```

- [ ] **Step 2: Implement SQL methods**

Implement `Nodes`, `Node`, `CreateNode`, `BindNode`, `UpdateNodeStatus` using `scanNode` helper.

- [ ] **Step 3: Run API tests**

Run: `go test ./apps/api-server/...`

Expected: PASS.

---

### Task 4: Admin Node API and Node Bind API

**Files:**
- Modify: `apps/api-server/internal/platform/server.go`

**Interfaces:**
- Consumes: Backend node methods, `NodeAgentClient`.
- Produces:
  - `GET /api/admin/nodes`
  - `POST /api/admin/nodes`
  - `GET /api/admin/nodes/{id}`
  - `POST /api/admin/nodes/{id}/status`
  - `POST /api/admin/nodes/{id}/frps-restart`
  - `POST /api/admin/nodes/{id}/frps-reload`
  - `GET /api/admin/nodes/{id}/frps-config`
  - `GET /api/admin/nodes/{id}/frps-logs`
  - `POST /api/admin/nodes/{id}/nginx-test`
  - `POST /api/admin/nodes/{id}/nginx-reload`
  - `POST /api/nodes/bind`

- [ ] **Step 1: Register routes**

Add handlers in `routes()`.

- [ ] **Step 2: Add action parser**

Parse `/api/admin/nodes/{id}/{action}` with `strings.TrimPrefix` and `strconv.ParseInt`.

- [ ] **Step 3: Implement node action forwarding**

Build `NodeAgentClient{BaseURL: node.AgentURL, Token: node.AgentToken}` and call existing methods.

- [ ] **Step 4: Record admin operation logs**

Record actions as `node.create`, `node.frps.restart`, `node.nginx.reload`, etc.

---

### Task 5: Tests for Node Inventory and Remote Forwarding

**Files:**
- Modify: `apps/api-server/internal/platform/server_test.go`
- Create: `apps/api-server/internal/platform/node_store_test.go`

**Interfaces:**
- Consumes: API endpoints from Task 4.
- Produces: coverage for node create, bind, and remote action forwarding.

- [ ] **Step 1: Test create/bind**

Create a node through admin API, call `/api/nodes/bind` with returned bind token, assert node becomes `online`.

- [ ] **Step 2: Test remote status**

Use `httptest.Server` as fake node-agent and assert `/api/admin/nodes/{id}/status` returns fake health.

- [ ] **Step 3: Run tests**

Run: `go test ./apps/api-server/internal/platform -run 'Node|Admin' -v`

Expected: PASS.

---

### Task 6: Node-Agent Self Binding

**Files:**
- Modify: `apps/api-server/cmd/node-agent/main.go`
- Modify: `deploy/.env.node.example`

**Interfaces:**
- Consumes: `POST /api/nodes/bind`.
- Produces: node-agent startup self-registration.

- [ ] **Step 1: Add env vars**

```env
CONTROL_PLANE_URL=https://api.example.com
NODE_BIND_TOKEN=paste-from-admin
NODE_NAME=edge-node-1
NODE_PUBLIC_AGENT_URL=http://node.example.com:8090
```

- [ ] **Step 2: Add bind goroutine**

On startup and every 60 seconds, POST bind payload to control plane. If env vars are empty, skip.

- [ ] **Step 3: Run build**

Run: `go build ./apps/api-server/cmd/node-agent`

Expected: PASS.

---

### Task 7: Admin WebUI Multi-Node Panel

**Files:**
- Modify: `apps/admin-web/index.html`

**Interfaces:**
- Consumes: admin node APIs.
- Produces: visible node inventory and management buttons.

- [ ] **Step 1: Add node section**

Add form fields: node name, agent URL, frp entry domain, server addr, frps port, TCP/UDP ranges.

- [ ] **Step 2: Add node table**

Columns: ID, name, agent URL, entry domain, status, last seen, actions.

- [ ] **Step 3: Add JS actions**

Call create/list/status/config/logs/restart/reload/nginx-test/nginx-reload endpoints and print results to log box.

---

### Task 8: Docs, Validation, Commit and Push

**Files:**
- Modify: `deploy/SPLIT_DEPLOYMENT.md`
- Modify: `README.md`
- Modify: `docs/FINAL_ACCEPTANCE_AUDIT.md` if relevant.

**Interfaces:**
- Consumes: completed implementation.
- Produces: committed and pushed code.

- [ ] **Step 1: Update docs**

Document admin creation -> copy bind token -> node `.env.node` -> node self-binding -> remote management.

- [ ] **Step 2: Validate**

Run:

```bash
go test ./apps/api-server/...
./scripts/dev-smoke.sh
cd deploy && cp .env.node.example .env.node && docker compose --env-file .env.node -f docker-compose.node.yml config >/tmp/frp-node-compose.yml
```

- [ ] **Step 3: Commit and push**

```bash
git add .
git commit -m "feat: add multi-node management"
git push origin master
```

---

## Self-Review

- Spec coverage: covers multi-node inventory, node binding to backend, remote node management, WebUI, docs, tests.
- Placeholder scan: no task says TBD/TODO; every task has concrete files, routes, fields, commands.
- Type consistency: `Node`, `NodeBindRequest`, and Backend method names are consistent across tasks.
