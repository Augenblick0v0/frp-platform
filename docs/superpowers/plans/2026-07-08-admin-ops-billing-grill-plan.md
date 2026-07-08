# Admin operations and billing fixes — grill plan

## User request
Fix the admin console so node operation buttons visibly work, collapsed sidebar labels are readable, plans can be added and edited, redeem codes select a plan instead of requiring manual IDs, redeem-code log errors are explained/fixed, payment configuration has an admin binding entry, and the final result is verified through a real end-to-end flow.

## Grill questions and decisions

1. **Where should node action output appear?**
   - Problem: the node page has action buttons, but the only visible log panel is on other pages, so successful/failed actions can look like no-op clicks.
   - Decision: add a dedicated Node Operation Result panel on the node page and route status/config/log/restart output there with loading/error states.

2. **What does “four characters show as two over two” mean for collapsed sidebar?**
   - Problem: collapsed labels squeeze into awkward one-line text.
   - Decision: when collapsed, split Chinese labels into two-character lines, e.g. `套餐管理` -> `套餐\n管理`, while keeping short labels centered.

3. **Is a Plan editable after users already bought/redeemed it?**
   - Risk: editing an existing Plan does not rewrite existing subscriptions, because subscriptions copy plan entitlements when activated.
   - Decision: implement admin create/update for the Plan catalog only; existing subscriptions remain unchanged. This matches the current data model.

4. **How should redeem codes bind to plans?**
   - Problem: manual `plan_id` is error-prone and not commercial-friendly.
   - Decision: use a select dropdown populated from active Plans and show generated codes plus recent code list with plan names.

5. **What could explain redeem-code errors when the user did not generate codes?**
   - Hypotheses: stale frontend log state, automatic data-load logging errors, admin operation log misread, API container logs, failed prior click, or DB query issue.
   - Decision: inspect DB operation logs and API logs; then make redeem page display persisted redeem-code list separately from transient operation logs.

6. **Where should payment binding live?**
   - Existing code reads EPAY settings from env only; frontend user page already calls `/api/payments/epay/orders`.
   - Decision: add an admin “支付配置” section under system settings that shows EPAY configured status and the effective non-secret endpoints/site name. Secret PID/key remain env/runtime configuration unless API support is added deliberately.

7. **What counts as a real full-flow test?**
   - Admin: login, create/edit plan, generate redeem code for selected plan, run node status/config/log/restart buttons, verify payment config visibility.
   - User: register/login a fresh user, redeem generated code, verify subscription activates the selected plan, validate paid-plan checkout entry.
   - Ops: run Go tests, JS syntax checks, rebuild/redeploy fnOS admin/user/API containers, browser/API smoke tests.

## Implementation slices

1. **API contract fixes**
   - Add `UpdatePlan(id, plan)` to Backend, memory Store, SQLStore.
   - Add `/api/admin/plans/{id}` PUT support.
   - Add read-only payment config/status endpoint under admin settings or include it in settings response.

2. **Admin UI fixes**
   - Plan form with create/update mode, prefill edit button, unit helpers for price/GB/Mbps/days/limits/status/protocols.
   - Redeem form dropdown with plan names, generated-code result panel, recent redeem-code table.
   - Node page operation result panel with per-button loading and visible output.
   - Collapsed sidebar label formatter.
   - Payment configuration section in settings.

3. **Diagnostics**
   - Query recent `admin_operation_logs`, `redeem_codes`, and API logs.
   - Fix any actual false generation/error; otherwise document finding and improve UI separation.

4. **Verification**
   - `go test ./apps/api-server/...`.
   - JS parse check for admin/user inline scripts.
   - Deploy fnOS stack.
   - API and browser-level end-to-end test using current admin account and fresh test user.
