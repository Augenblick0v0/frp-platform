# Tunnel Speed Test Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add client-side tunnel speed testing for Linux WebUI and Windows EXE users, covering TCP, UDP, HTTP, and HTTPS, while proving package bandwidth limits are actually enforced.

**Architecture:** Treat speed testing as a first-class client diagnostic flow. The client starts a local benchmark service, asks the server for a temporary speed-test tunnel or uses an eligible selected tunnel, runs download/upload/latency probes through frps/frpc, reports the consumed bytes to quota accounting, and displays average speed, peak speed, latency, and limit comparison. Bandwidth limits are modeled as a subscription-level cap with an optional per-tunnel override that may only lower the cap.

**Tech Stack:** Go standard library HTTP/TCP/UDP servers, existing API server stores, existing frp client manager, existing Linux WebUI/Windows EXE packaged client, official frps/frpc runtime.

## Global Constraints

- Linux users access the feature from the client WebUI.
- Windows users access the same feature from the packaged Windows EXE client surface.
- TCP, UDP, HTTP, and HTTPS speed tests are in scope.
- Speed-test bytes count as real package traffic.
- Package bandwidth is the upper limit for the user.
- A tunnel may set its own bandwidth limit only when that value is greater than zero and less than or equal to the active package limit.
- The result UI must show download speed, upload speed, average speed, peak speed, latency, and comparison against the effective limit.
- Prefer real frps/frpc E2E verification over only unit tests.
- Local Windows currently does not have `go` on PATH, so implementation verification may need bundled Go, remote fnOS, or a local Go install before running Go tests.

---

## Grilling Answers From User

1. **Client surfaces:** Linux has WebUI; Windows is EXE; both need the feature.
2. **Protocols:** test all tunnel types: TCP, UDP, HTTP, and HTTPS.
3. **Traffic accounting:** speed-test traffic counts toward the user's package traffic.
4. **Limit model:** package-level unified bandwidth limit, with optional per-tunnel override during tunnel creation; tunnel override cannot exceed the package limit.
5. **Result detail:** show download/upload, average/peak, latency, and limit comparison.

## Follow-Up Assumptions To Implement Unless Changed

- For all protocols, the safest implementation is a temporary speed-test tunnel owned by the current user, not hijacking a production tunnel that might point at an arbitrary local app.
- TCP and UDP speed tests use automatically allocated temporary ports from the same server-side port pools.
- HTTP speed tests use a temporary test host under a platform-controlled speed-test domain suffix.
- HTTPS speed tests use the same temporary host and the existing certificate automation path. If the platform has not configured a usable wildcard/test domain and certificate path, the UI should show HTTPS speed test unavailable with a precise reason instead of silently falling back to HTTP.
- Temporary speed-test tunnels expire automatically and release ports/domains even if the client exits.
- A tunnel-specific limit of `0` means "inherit package limit".

## File Structure

- Create: `CONTEXT.md` for resolved domain vocabulary.
- Modify: `apps/api-server/internal/platform/models.go`: add tunnel bandwidth override and speed-test tunnel/request/result domain types.
- Modify: `apps/api-server/internal/platform/sql_migrations.go`: add `tunnels.bandwidth_limit_kbps`, speed-test tunnel TTL metadata if stored separately, and indexes needed for cleanup.
- Modify: `apps/api-server/internal/platform/store.go`: enforce per-tunnel override <= package limit, create temporary speed-test tunnels, release speed-test resources, and account test traffic.
- Modify: `apps/api-server/internal/platform/sql_store.go`: mirror the in-memory store behavior in PostgreSQL.
- Modify: `apps/api-server/internal/platform/server.go`: expose speed-test tunnel create/finish APIs and include effective bandwidth in `/api/client/tunnels`.
- Modify: `apps/api-server/internal/platform/server_test.go`: cover package limit exposure, per-tunnel override validation, speed-test tunnel lifecycle, and quota accounting.
- Modify: `client/frp-client/internal/clientcore/config.go`: render effective per-proxy `transport.bandwidthLimit`.
- Modify: `client/frp-client/internal/clientcore/config_test.go`: prove inherited and per-tunnel limits render correctly.
- Modify: `client/frp-client/internal/clientcore/manager.go`: add benchmark service lifecycle, TCP/UDP/HTTP probe logic, latency measurement, peak/average aggregation, and server traffic report call.
- Modify: `client/frp-client/internal/clientcore/manager_test.go`: cover local benchmark probes without frps.
- Modify: `client/frp-client/internal/clientcore/server.go`: expose local `POST /api/speed-tests/run`.
- Modify: `apps/client-webui/index.html`: add speed-test controls/results for Linux WebUI and packaged Windows EXE surface.
- Modify: `apps/client-webui/style.css`: add compact speed-test result layout.
- Modify: `client/packaging/windows/build-windows.sh` and `client/packaging/windows/build-windows.ps1`: make sure the updated WebUI assets ship with the Windows EXE package.
- Modify: `client/packaging/linux/build-linux.sh`: make sure updated WebUI assets ship with the Linux package.
- Modify: `scripts/dev-smoke.sh`: include config-render and real frps/frpc speed-limit verification.
- Optionally create: `scripts/e2e-speed-limit.sh` if the E2E script becomes too large for `dev-smoke.sh`.
- Modify: `docs/IMPLEMENTATION_STATUS.md`: record whether limit verification passed and the measured cap.

---

### Task 1: Lock The Domain Model

**Files:**
- Create/Modify: `CONTEXT.md`

**Interfaces:**
- Produces canonical terms for later code and UI labels.

- [ ] **Step 1: Add glossary entries**

Create or update `CONTEXT.md` with these terms:

```md
# FRP Tunnel Platform

This context describes the commercial FRP tunnel platform: package-gated users create tunnels through managed frps nodes, and packaged clients operate frpc plus local diagnostics.

## Language

**Package Bandwidth Limit**:
The maximum bandwidth a user's active package allows across that user's tunnels.
_Avoid_: plan speed, account speed

**Tunnel Bandwidth Override**:
An optional per-tunnel bandwidth value that can only reduce the Package Bandwidth Limit for that tunnel.
_Avoid_: tunnel package, custom speed above package

**Effective Tunnel Bandwidth**:
The bandwidth value actually rendered into frpc for one tunnel after combining package limit and tunnel override.
_Avoid_: final speed

**Speed Test Tunnel**:
A temporary tunnel created only to measure TCP, UDP, HTTP, or HTTPS performance for the current user.
_Avoid_: production tunnel test

**Speed Test Traffic**:
Traffic generated by a Speed Test Tunnel that counts toward the user's package traffic quota.
_Avoid_: free test traffic
```

- [ ] **Step 2: Review terms against user answers**

Expected: the terms match the user's requirements and avoid treating speed tests as free or outside the package.

---

### Task 2: Add Package And Tunnel Limit Enforcement

**Files:**
- Modify: `apps/api-server/internal/platform/models.go`
- Modify: `apps/api-server/internal/platform/store.go`
- Modify: `apps/api-server/internal/platform/sql_migrations.go`
- Modify: `apps/api-server/internal/platform/sql_store.go`
- Test: `apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Consumes: active `Subscription.BandwidthKbps`
- Produces: `Tunnel.BandwidthLimitKbps`, effective bandwidth validation, and stored tunnel overrides

- [ ] **Step 1: Write failing tests**

Add tests proving:
- creating a tunnel with `bandwidth_limit_kbps` greater than the active package limit returns `403`
- creating a tunnel with `bandwidth_limit_kbps` less than or equal to the package limit succeeds
- creating a tunnel with `bandwidth_limit_kbps = 0` inherits the package limit

- [ ] **Step 2: Add model field**

Add to `Tunnel`:

```go
BandwidthLimitKbps int `json:"bandwidth_limit_kbps"`
```

- [ ] **Step 3: Add database migration**

Add to `tunnels`:

```sql
bandwidth_limit_kbps INTEGER NOT NULL DEFAULT 0
```

- [ ] **Step 4: Enforce override rules**

During tunnel creation:
- reject negative values
- if subscription `BandwidthKbps > 0` and tunnel override is greater than subscription limit, return `ErrForbidden`
- if subscription has no bandwidth limit, allow `0` only unless the product wants an explicit unlimited package

- [ ] **Step 5: Run targeted tests**

Run:

```bash
go test ./apps/api-server/internal/platform -run 'Bandwidth|Tunnel' -v
```

Expected: PASS.

---

### Task 3: Expose Effective Limits To The Client

**Files:**
- Modify: `apps/api-server/internal/platform/server.go`
- Test: `apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Consumes: subscription bandwidth and tunnel override
- Produces: `/api/client/tunnels` includes package and effective per-tunnel limit data

- [ ] **Step 1: Write failing API test**

Test that `/api/client/tunnels` returns:
- top-level `bandwidth_limit_kbps`
- each tunnel's `bandwidth_limit_kbps`
- each tunnel's `effective_bandwidth_limit_kbps`

- [ ] **Step 2: Implement response shape**

For each tunnel:
- if tunnel override is positive, effective limit is override
- otherwise effective limit is subscription `BandwidthKbps`

- [ ] **Step 3: Preserve old client compatibility**

Keep existing keys: `server_addr`, `server_port`, `token`, and `tunnels`.

- [ ] **Step 4: Run API tests**

Run:

```bash
go test ./apps/api-server/internal/platform -run ClientTunnels -v
```

Expected: PASS.

---

### Task 4: Render Effective Limits Into frpc

**Files:**
- Modify: `client/frp-client/internal/clientcore/config.go`
- Test: `client/frp-client/internal/clientcore/config_test.go`

**Interfaces:**
- Consumes: tunnel `effective_bandwidth_limit_kbps`
- Produces: frpc proxy config with `transport.bandwidthLimit`

- [ ] **Step 1: Write failing renderer tests**

Cover:
- package inherited limit renders on TCP, UDP, HTTP, and HTTPS proxies
- per-tunnel lower override renders instead of package limit
- zero/unlimited limit does not render `transport.bandwidthLimit`

- [ ] **Step 2: Add config fields**

Add fields to client `Tunnel`:

```go
BandwidthLimitKbps          int `json:"bandwidth_limit_kbps"`
EffectiveBandwidthLimitKbps int `json:"effective_bandwidth_limit_kbps"`
```

- [ ] **Step 3: Render per proxy**

After proxy local settings, emit:

```go
if t.EffectiveBandwidthLimitKbps > 0 {
	fmt.Fprintf(&b, "transport.bandwidthLimit = %q\n", frpcBandwidthLimit(t.EffectiveBandwidthLimitKbps))
}
```

- [ ] **Step 4: Run client config tests**

Run:

```bash
go test ./client/frp-client/internal/clientcore -run RenderFRPCConfig -v
```

Expected: PASS.

---

### Task 5: Add Server Speed-Test Tunnel Lifecycle

**Files:**
- Modify: `apps/api-server/internal/platform/models.go`
- Modify: `apps/api-server/internal/platform/store.go`
- Modify: `apps/api-server/internal/platform/sql_store.go`
- Modify: `apps/api-server/internal/platform/server.go`
- Test: `apps/api-server/internal/platform/server_test.go`

**Interfaces:**
- Consumes: authenticated user, requested protocol, optional per-tunnel bandwidth override
- Produces: temporary speed-test tunnel with TTL, public access info, and cleanup endpoint

- [ ] **Step 1: Write failing lifecycle tests**

Cover:
- TCP speed-test tunnel allocates a TCP port
- UDP speed-test tunnel allocates a UDP port
- HTTP speed-test tunnel allocates a platform test domain
- HTTPS speed-test tunnel validates certificate/domain readiness or returns a clear unavailable error
- cleanup releases port/domain

- [ ] **Step 2: Add API endpoints**

Add authenticated routes:

```text
POST /api/speed-tests/tunnels
POST /api/speed-tests/:id/finish
```

- [ ] **Step 3: Request shape**

Use:

```json
{
  "type": "tcp",
  "local_host": "127.0.0.1",
  "local_port": 19090,
  "bandwidth_limit_kbps": 512
}
```

- [ ] **Step 4: Response shape**

Return:

```json
{
  "id": 123,
  "type": "tcp",
  "public_url": "frp.example.com:20000",
  "remote_port": 20000,
  "domain": "",
  "effective_bandwidth_limit_kbps": 512,
  "expires_at": "2026-07-04T16:00:00Z"
}
```

- [ ] **Step 5: Run server tests**

Run:

```bash
go test ./apps/api-server/internal/platform -run SpeedTest -v
```

Expected: PASS.

---

### Task 6: Add Client Benchmark Service And Probes

**Files:**
- Modify: `client/frp-client/internal/clientcore/manager.go`
- Test: `client/frp-client/internal/clientcore/manager_test.go`

**Interfaces:**
- Consumes: speed-test tunnel public access info
- Produces: download/upload average, download/upload peak, latency, bytes used, and limit comparison

- [ ] **Step 1: Write failing local probe tests**

Use `httptest.Server` for HTTP-style probes and local `net.Listen` / `net.ListenPacket` for TCP/UDP probes.

- [ ] **Step 2: Add benchmark service**

The client starts local listeners for:
- HTTP/HTTPS payload endpoints: `GET /__frp_speed/download`, `POST /__frp_speed/upload`, `GET /__frp_speed/ping`
- TCP binary probe: ping, download stream, upload stream
- UDP datagram probe: ping datagrams and upload/download packet bursts

- [ ] **Step 3: Add measurement aggregation**

Return:

```go
type SpeedTestResult struct {
	Type string `json:"type"`
	DownloadAverageKbps float64 `json:"download_average_kbps"`
	DownloadPeakKbps float64 `json:"download_peak_kbps"`
	UploadAverageKbps float64 `json:"upload_average_kbps"`
	UploadPeakKbps float64 `json:"upload_peak_kbps"`
	LatencyMs float64 `json:"latency_ms"`
	BytesIn int64 `json:"bytes_in"`
	BytesOut int64 `json:"bytes_out"`
	EffectiveBandwidthLimitKbps int `json:"effective_bandwidth_limit_kbps"`
	LimitRatio float64 `json:"limit_ratio"`
}
```

- [ ] **Step 4: Run clientcore tests**

Run:

```bash
go test ./client/frp-client/internal/clientcore -run Speed -v
```

Expected: PASS.

---

### Task 7: Add Local Client API For Linux WebUI And Windows EXE

**Files:**
- Modify: `client/frp-client/internal/clientcore/server.go`
- Test: `client/frp-client/internal/clientcore/manager_test.go`

**Interfaces:**
- Consumes: local request from packaged UI
- Produces: `POST /api/speed-tests/run`

- [ ] **Step 1: Add local endpoint**

Request:

```json
{
  "api_base": "https://api.example.com",
  "token": "user-token",
  "type": "tcp",
  "download_bytes": 10485760,
  "upload_bytes": 10485760,
  "duration_seconds": 10,
  "bandwidth_limit_kbps": 512
}
```

- [ ] **Step 2: Endpoint flow**

The endpoint:
1. starts benchmark local listener
2. asks server to create temporary speed-test tunnel
3. syncs/renders frpc config with the temporary tunnel
4. starts or reloads frpc
5. runs probes through public endpoint
6. reports bytes through `/api/client/traffic`
7. finishes temporary speed-test tunnel
8. returns result JSON to the UI

- [ ] **Step 3: Failure cleanup**

On any error after tunnel creation, call finish cleanup before returning.

- [ ] **Step 4: Run tests**

Run:

```bash
go test ./client/frp-client/internal/clientcore -v
```

Expected: PASS.

---

### Task 8: Add User-Facing UI

**Files:**
- Modify: `apps/client-webui/index.html`
- Modify: `apps/client-webui/style.css`

**Interfaces:**
- Consumes: local `POST /api/speed-tests/run`
- Produces: Linux WebUI and Windows EXE packaged UI for speed tests

- [ ] **Step 1: Add controls**

Controls:
- protocol selector: TCP, UDP, HTTP, HTTPS
- download test size
- upload test size
- duration
- optional per-tunnel speed limit
- start test button

- [ ] **Step 2: Add result display**

Show:
- download average
- download peak
- upload average
- upload peak
- latency
- package limit
- tunnel override
- effective limit
- pass/warn/fail comparison
- bytes counted toward traffic quota

- [ ] **Step 3: Add validation copy**

If tunnel override exceeds package limit, show: `Tunnel limit cannot exceed package limit`.

- [ ] **Step 4: Manual UI smoke**

Check Linux WebUI and Windows packaged asset layout:
- desktop width
- narrow mobile-like width
- long API URL/token values
- running state while test is active
- failure message for unavailable HTTPS test domain/cert

---

### Task 9: Package The Updated Client

**Files:**
- Modify: `client/packaging/windows/build-windows.sh`
- Modify: `client/packaging/windows/build-windows.ps1`
- Modify: `client/packaging/linux/build-linux.sh`

**Interfaces:**
- Consumes: updated `apps/client-webui`
- Produces: packages where Linux WebUI and Windows EXE include the speed-test UI

- [ ] **Step 1: Verify asset copy paths**

Confirm both Linux and Windows package builders copy the updated WebUI directory.

- [ ] **Step 2: Build packages**

Run:

```bash
./scripts/release.sh 0.1.3
./scripts/verify-release.sh 0.1.3
```

Expected: generated Linux and Windows packages include the speed-test UI.

---

### Task 10: Real Limit Verification

**Files:**
- Modify: `scripts/dev-smoke.sh`
- Optionally create: `scripts/e2e-speed-limit.sh`
- Modify: `docs/IMPLEMENTATION_STATUS.md`

**Interfaces:**
- Consumes: real api-server, frps, frpc, temporary speed-test tunnel
- Produces: evidence that bandwidth limiting works or a precise failure note

- [ ] **Step 1: Test current state before fix**

Before implementation, run or document that generated frpc config lacks bandwidth limit. Expected current result: limit not actually enforced because it is not rendered into frpc.

- [ ] **Step 2: Add E2E test after fix**

Start:
- api-server with test settings
- frps with known token
- frp-client with official frpc

Create:
- package with `bandwidth_limit_kbps = 512`
- temporary speed-test tunnel for each protocol

Assert:
- generated `frpc.toml` contains `transport.bandwidthLimit = "64KB"` for a `512 Kbps` package limit
- measured average speed is near the cap with a jitter tolerance
- speed-test bytes are visible in `/api/user/traffic`

- [ ] **Step 3: Record verification**

Update `docs/IMPLEMENTATION_STATUS.md` with:
- protocol tested
- configured package limit
- effective tunnel limit
- measured download average/peak
- measured upload average/peak
- latency
- traffic bytes reported

---

## Acceptance Criteria

- Linux client WebUI includes speed-test controls and results.
- Windows EXE package includes the same user-facing speed-test capability.
- TCP, UDP, HTTP, and HTTPS speed tests are supported or return a precise setup error when HTTPS domain/cert prerequisites are missing.
- Speed-test traffic is reported to package traffic usage.
- Package bandwidth limit is returned to the client.
- Tunnel creation supports optional per-tunnel bandwidth override.
- Server rejects tunnel override values greater than the active package limit.
- frpc config renders the effective limit per proxy.
- UI shows download/upload average, download/upload peak, latency, and limit comparison.
- Real frps/frpc verification proves whether limiting is effective.
- Existing tunnel creation and config sync behavior remains compatible.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-07-04-tunnel-speed-test.md`.

Recommended execution: inline with checkpoints. This work crosses domain model, API, local client runtime, UI, packaging, and E2E verification, and several steps should be reviewed against real frps/frpc behavior before release.
