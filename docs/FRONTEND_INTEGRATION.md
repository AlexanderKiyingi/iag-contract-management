# Contract-Management Frontend Integration Guide

Comprehensive guide for connecting a frontend (Next.js, SvelteKit, plain SPA)
to the contract-management backend. Covers auth, the full HTTP route catalog,

**Canonical UI:** `iagprojects/contracts` (Next.js ACP app, sibling to this meta-repo) — set `NEXT_PUBLIC_ACP_API_URL` to the gateway base in §2 and enable platform OAuth per §1.
the permission model, pagination, file uploads, and the short-key DTO
conventions shared with the event bus.

For deployment-side env config see [PLATFORM_INTEGRATION.md](./PLATFORM_INTEGRATION.md);
for production rollout notes see [RAILWAY.md](./RAILWAY.md). This guide is
the integration long form.

---

## 1. Authentication

Contract-management runs in **platform Bearer+aud mode** (no gateway
fallback). Every request — except the health probes (§4.1) — requires:

```
Authorization: Bearer <jwt>
```

The JWT must carry `aud=iag.contract-management`. A user-principal token
issued by auth already includes this audience; service-principal tokens
issued for `iag.contract-management` also work.

The service verifies signatures locally against the auth service's JWKS,
refreshed every 15 minutes — there is no callback to auth on the request
hot path.

### Token acquisition

```
┌─────────┐  1. POST /api/v1/authentication/oauth/token  ┌──────────┐
│ Browser │ ────────── grant_type=password ─────────────▶│   Auth   │
│  (FE)   │                                              │ Service  │
│         │◀────── access_token, refresh_token ──────────│          │
└─────────┘                                              └──────────┘
     │  2. Authorization: Bearer <access_token>
     ▼
┌──────────────────────┐
│ contract-management  │  (verifies JWT locally via cached JWKS)
└──────────────────────┘
```

**Frontend responsibilities:**
- Keep `access_token` in memory; refresh ~1 minute before its 15-minute TTL.
- On any 401 from contract-management, attempt refresh; on 401 from refresh,
  redirect to login.
- On 403, the call passed auth but the user lacks the specific permission
  for the route — hide the UI control rather than retry.

### Common 401 / 403 causes

1. Token expired (refresh).
2. `aud` claim missing `iag.contract-management` — re-login through auth so a
   fresh multi-audience token is issued.
3. JWKS rotation in flight (transient, resolves within 15 min).
4. 403 specifically: route requires a permission the JWT doesn't carry —
   inspect `permissions` from `GET /v1/permissions/me` (§5).

---

## 2. Base URLs

| Environment | API base |
|---|---|
| Local direct | `http://localhost:4103/v1` |
| Local via gateway | `http://localhost:8080/api/v1/contract-management/v1` |
| Production | `https://iag-api-gateway-production.up.railway.app/api/v1/contract-management/v1` |

**Always go through the gateway in non-local environments.** It owns rate
limiting, CORS, request IDs, and routes `/api/v1/contract-management/*` to
this service. In production the only public host is the gateway
(`https://iag-api-gateway-production.up.railway.app`); contract-management
itself runs on Railway's private network
(`iag-contract-management.railway.internal:4103`) and is not exposed via its
own `*.up.railway.app` URL.

### Required frontend env vars

```env
# Local (via gateway)
NEXT_PUBLIC_CONTRACTS_API_URL=http://localhost:8080/api/v1/contract-management/v1
NEXT_PUBLIC_AUTH_API_URL=http://localhost:8080/api/v1/authentication
NEXT_PUBLIC_GATEWAY_ORIGIN=http://localhost:8080
```

```env
# Production (Railway, via gateway)
NEXT_PUBLIC_CONTRACTS_API_URL=https://iag-api-gateway-production.up.railway.app/api/v1/contract-management/v1
NEXT_PUBLIC_AUTH_API_URL=https://iag-api-gateway-production.up.railway.app/api/v1/authentication
NEXT_PUBLIC_GATEWAY_ORIGIN=https://iag-api-gateway-production.up.railway.app
```

### CORS

Origins are configured via `CORS_ORIGIN` (or legacy `ALLOWED_ORIGINS`),
comma-separated. Default is `*` in dev, empty in production (gateway
terminates). Auth uses the `Authorization` header — **no cookies**. The
service does set the standard hardening headers (HSTS, no-sniff,
deny-frame, strict-referrer).

Request body is capped at **8 MB** (`MAX_BODY_BYTES`); per-request timeout
is 30 s (`REQUEST_TIMEOUT_SECONDS`); rate limit is 120/min/IP.

---

## 3. Permission Model

Contract-management uses **Django-style `module.action` codenames**, all
namespaced under `iag.contract-management`. The catalog is fixed and lives
in [internal/models/permissions.go](../internal/models/permissions.go).

### 3.1 Modules × Actions = 44 keys

| Module | Label |
|---|---|
| `contracts` | Contracts |
| `zones` | Zones |
| `payments` | Payments |
| `tasks` | Tasks |
| `milestones` | Milestones |
| `materials` | Materials |
| `users` | Users |
| `roles` | Roles |
| `audit` | Audit log |
| `reports` | Reports |
| `insights` | AI & insights |

Actions: `create`, `read`, `update`, `delete`. So `contracts.read`,
`payments.update`, `audit.create`, etc.

### 3.2 Built-in roles

| Role | Effective permissions |
|---|---|
| `super_admin` / `admin` | All 44 keys (bypass) |
| `manager` | `contracts.{create,read,update}` (no delete) · `zones.{read,update}` · `payments.read` · full CRUD on `tasks`/`milestones`/`materials` · read-only on `users`/`roles`/`audit` · `reports.{read,create}` · `insights.{read,update}` |
| `viewer` | All `*.read` (read-only every module) |
| `contractor` | `contracts.read`, **scoped** to contracts where `sup` matches their supervisor mapping |

### 3.3 Legacy aliases

Older clients may send any of these keys; the server resolves them to the
canonical permissions before enforcing. Useful when migrating UI gating
code.

| Alias | Resolves to |
|---|---|
| `portfolio.view` | `contracts.read, zones.read` |
| `portfolio.edit` | `contracts.create, contracts.read, contracts.update, zones.read, zones.update` |
| `portfolio.delete` | `contracts.delete` |
| `payments.view` | `payments.read` |
| `tasks.manage` | `tasks.create, tasks.read, tasks.update, tasks.delete` |
| `milestones.manage` | `milestones.create, milestones.read, milestones.update, milestones.delete` |
| `materials.manage` | `materials.{create,read,update,delete}` |
| `users.manage` | `users.{create,read,update,delete}` |
| `roles.manage` | `roles.{create,read,update,delete}` |
| `audit.view` | `audit.read` |
| `reports.export` | `reports.read, reports.create` |
| `insights.run` | `insights.read, insights.update` |

### 3.4 Permissions API (use these from the frontend)

| Method | Path | Description |
|---|---|---|
| GET | `/permissions/catalog` | Module/action lists, role labels, built-in roles, alias map |
| GET | `/permissions/builtin` | Built-in role → permission mapping (UI role picker) |
| GET | `/permissions/me` | Caller's effective permissions + context (role, canMutate, canManageRoles, isPortal) |
| POST | `/permissions/check` | Batch-check; body `{keys: ["contracts.update","..."]}` → `{allowed: {key: bool}}` |
| GET | `/permissions/users/:id` | Effective permissions for another user (requires `users.read`) |

**UI gating pattern:** at login call `/permissions/me`, cache the
permissions array, and hide controls the user doesn't have. The backend
re-checks on every mutating request — gating is purely UX.

---

## 4. Complete Endpoint Catalog

All routes are prefixed with the base URL (§2). Routes are gated by the
permission listed in the third column; `—` means any authenticated user.

### 4.1 Public probes (no auth)

| Method | Path | Description |
|---|---|---|
| GET | `/health` | Liveness — `{status:"ok"}` |
| GET | `/health/live` | Liveness alias |
| GET | `/health/ready` | Readiness (DB ping) |
| GET | `/ready` | Readiness alias |
| GET | `/v1/health` | v1-prefixed alias |
| GET | `/v1/health/live` | v1-prefixed alias |
| GET | `/v1/health/ready` | v1-prefixed alias |

### 4.2 Session / snapshots

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/v1/auth/session` | — | Current session + permission context |
| GET | `/v1/bootstrap` | — | One-shot workspace snapshot (contracts, zones, engineers, users, frontend store, permissions) tuned for the caller |
| GET | `/v1/workspace` | — | Workspace snapshot filtered by permissions + contractor scope |
| PUT | `/v1/workspace` | `super_admin` | Replace entire workspace (destructive) |
| GET | `/v1/frontend` | — | Frontend store snapshot (tasks, milestones, audit, materials, custom roles, profile photos) |
| PUT | `/v1/frontend` | `super_admin` | Replace entire frontend store |

> **Tip:** `/v1/bootstrap` is the recommended single fetch on app load —
> it returns everything the SPA needs in one round-trip.

### 4.3 Contracts

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/v1/contracts` | `contracts.read` | List (paginated; see §6) |
| POST | `/v1/contracts` | `contracts.create` | Create |
| GET | `/v1/contracts/:no` | `contracts.read` | Get one (by contract number) |
| PATCH | `/v1/contracts/:no` | `contracts.update`* | Partial update |
| PUT | `/v1/contracts/:no` | `contracts.update`* | Full replace |
| DELETE | `/v1/contracts/:no` | `contracts.delete` | Delete |

> `*` Contractors can only update contracts where `sup` matches their
> supervisor mapping ([session_access.go](../internal/handlers/session_access.go) `CanEditContract`).

### 4.4 Zones

| Method | Path | Permission |
|---|---|---|
| GET | `/v1/zones` | `zones.read` |
| GET | `/v1/zones/:code` | `zones.read` |

### 4.5 Engineers

| Method | Path | Permission |
|---|---|---|
| GET | `/v1/engineers` | `users.read` |
| GET | `/v1/engineers/:id` | `users.read` |
| POST | `/v1/engineers` | canMutate |
| PATCH | `/v1/engineers/:id` | canMutate |
| DELETE | `/v1/engineers/:id` | canMutate |

`canMutate` = super_admin OR admin OR manager OR holds `contracts.update`.

### 4.6 Users

| Method | Path | Permission |
|---|---|---|
| GET | `/v1/users` | `users.read` |
| GET | `/v1/users/:id` | `users.read` |
| POST | `/v1/users` | `users.create` |
| PATCH | `/v1/users/:id` | `users.update` |
| DELETE | `/v1/users/:id` | `users.delete` |

### 4.7 Milestones

| Method | Path | Permission |
|---|---|---|
| GET | `/v1/milestones` | `milestones.read` |
| POST | `/v1/milestones` | `milestones.create` |
| GET | `/v1/milestones/:id` | `milestones.read` |
| PATCH | `/v1/milestones/:id` | `milestones.update` |
| DELETE | `/v1/milestones/:id` | `milestones.delete` |

### 4.8 Materials

| Method | Path | Permission |
|---|---|---|
| GET | `/v1/materials` | `materials.read` |
| POST | `/v1/materials` | `materials.create` |
| PATCH | `/v1/materials/:id` | `materials.update` |
| DELETE | `/v1/materials/:id` | `materials.delete` |

### 4.9 Projects & tasks (nested)

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/v1/projects` | `tasks.read` | List task projects |
| POST | `/v1/projects` | `tasks.create` | Create project |
| PATCH | `/v1/projects/:index` | `tasks.update` | Update by ordinal index |
| DELETE | `/v1/projects/:index` | `tasks.delete` | Delete by index |
| POST | `/v1/projects/:index/tasks` | `tasks.create` | Add task to project |
| PATCH | `/v1/projects/:index/tasks/:taskId` | `tasks.update` | Update task |
| DELETE | `/v1/projects/:index/tasks/:taskId` | `tasks.delete` | Delete task |

> Projects are addressed by **ordinal index** (their position in the array),
> not by ID. Re-fetch the list after any add/delete to refresh indices.

### 4.10 Custom roles (workspace-defined)

| Method | Path | Permission |
|---|---|---|
| GET | `/v1/roles` | `roles.read` |
| POST | `/v1/roles` | canManageRoles |
| GET | `/v1/roles/:id` | `roles.read` |
| PATCH | `/v1/roles/:id` | canManageRoles |
| DELETE | `/v1/roles/:id` | canManageRoles |

`canManageRoles` = super_admin OR admin OR (`roles.create` OR `roles.update`).

### 4.11 Audit

| Method | Path | Permission |
|---|---|---|
| GET | `/v1/audit` | `audit.read` |
| GET | `/v1/audit/:id` | `audit.read` |
| POST | `/v1/audit` | `audit.create` |

### 4.12 Assistance (cross-team requests)

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/v1/assistance` | — | List assistance messages (self-scoped) |
| POST | `/v1/assistance` | canMutate | Request assistance — publishes `contracts.assistance.requested` on `iag.commercial`, dispatched as a notification |

### 4.13 Profile photos

| Method | Path | Permission | Description |
|---|---|---|---|
| GET | `/v1/profile/photo` | — | `?email=…` (defaults to caller); returns `{email, dataUrl}` |
| PUT | `/v1/profile/photo` | — | Body `{email, dataUrl}` — set photo |
| DELETE | `/v1/profile/photo` | — | `?email=…` — clear photo |
| POST | `/v1/uploads/profile` | — | Multipart `file` field **or** JSON `{email, dataUrl}` — images only, **≤ 2 MB** |

### 4.14 Insights & exports

| Method | Path | Permission | Description |
|---|---|---|---|
| PUT | `/v1/insights/scan` | `insights.update` | Update AI scan result |
| GET | `/v1/exports/contracts.csv` | `contracts.read` OR `reports.create` | RFC-4180 CSV: `no, name, zone, status, priority, prog, cs, paid, bal, workers, sup, created` |

---

## 5. The Contract DTO (and short-key conventions)

Both REST responses and Kafka events use the same compact keys. There is
**no field expansion** between the two surfaces — what you see in events
is what REST returns.

```ts
type Contract = {
  no:      string;           // contract number (primary key)
  name:    string;
  zone:    string;           // zone code
  cs:      number;           // contract sum (integer, smallest unit)
  paid:    number;           // amount paid
  bal:     number;           // balance (cs − paid)
  prog:    number;           // progress %, 0–100
  status:  "Planning" | "Active" | "On Hold" | "Complete";
  pri:     "High" | "Medium" | "Low";
  workers: number;
  sup:     string;           // supervisor display name
  remarks: string;
  created: string;           // YYYY-MM-DD
};
```

Status transitions are not state-machine-enforced server-side — the
frontend is responsible for offering valid next states. Listen for
`contracts.contract.status_changed` events (§9) to keep multiple open
clients in sync.

---

## 6. Pagination & filtering

The contracts list supports optional pagination. Pass **either** `page` or
`pageSize` to opt in:

```
GET /v1/contracts?page=1&pageSize=50
```

| Param | Default | Max |
|---|---|---|
| `page` | 1 | — |
| `pageSize` | 50 | 500 |

**Paginated response shape:**
```json
{
  "data": [ /* Contract[] */ ],
  "meta": { "page": 1, "pageSize": 50, "total": 123, "totalPages": 3 }
}
```

**Unpaginated response shape** (no `page`/`pageSize` supplied):
```json
[ /* Contract[] */ ]
```

No declarative filter syntax — fetch the page and filter client-side, or
fetch via `/v1/bootstrap` for the indexed view the SPA usually wants.

---

## 7. Error Conventions

| Status | Meaning | Frontend action |
|---|---|---|
| 400 | Bad request body / validation | Show inline field error |
| 401 | Missing / invalid / expired token | Refresh; on second 401, re-login |
| 403 | Permission denied (route gate failed) | Hide the UI control; show toast |
| 404 | Resource not found | Treat as soft state (deleted elsewhere?) |
| 409 | Conflict (e.g. duplicate contract number) | Re-fetch and retry |
| 413 | Request body > 8 MB | Trim payload or use multipart for files |
| 422 | Domain validation (e.g. invalid status string) | Show domain error |
| 429 | Rate limit (120/min/IP) | Backoff |
| 500 | Server error | Generic toast + retry button |
| 503 | Dependency unavailable (DB) | Show maintenance banner |

Response bodies follow `{"error":"message"}` — there's no
machine-readable code field, so prefer status codes for branching logic.

---

## 8. File Uploads (Profile Photos Only)

There are **no document attachments** at the API surface — contract
attachments today are URLs to external storage and live as string fields
on the Contract DTO.

Profile photos accept two shapes against `POST /v1/uploads/profile`:

**Multipart form-data:**
```
Content-Type: multipart/form-data; boundary=…

--boundary
Content-Disposition: form-data; name="email"

alex@example.com
--boundary
Content-Disposition: form-data; name="file"; filename="me.jpg"
Content-Type: image/jpeg

<binary>
--boundary--
```

**JSON data URL:**
```json
{ "email": "alex@example.com", "dataUrl": "data:image/jpeg;base64,/9j/..." }
```

Both return `{email, dataUrl}`. Max **2 MB**, images only (`image/*`
MIME). The dataUrl form is convenient when you've already read+cropped
the file client-side; the multipart form is preferred for large originals
to avoid the base64 ~33% size penalty.

---

## 9. Event Bus (Server-side only)

Contract-management is publish-only on Kafka topic `iag.commercial`. **The
frontend never connects to Kafka.** Events are surfaced via the notifications
service or by PM, which consumes some of them. They're listed here only so
you know which UI states have an audit trail:

| Event type | When emitted | Payload keys |
|---|---|---|
| `contracts.contract.created` | POST /v1/contracts | `no, name, zone, status, cs, paid, bal, prog, sup, created` |
| `contracts.contract.updated` | PATCH /v1/contracts/:no | same |
| `contracts.contract.deleted` | DELETE /v1/contracts/:no | same |
| `contracts.contract.status_changed` | PATCH changing `status` | same + `previousStatus` |
| `contracts.assistance.requested` | POST /v1/assistance | `from, text, at` |
| `contracts.milestone.due_soon` | Background job (see [PLATFORM_INTEGRATION.md](./PLATFORM_INTEGRATION.md) `MILESTONE_REMINDER_DAYS`) | `id, title, due, zone, status, owner` |
| `contracts.alert.raised` | Internal — drives notifications dispatch | `channel, recipient, templateId, variables` |

If you need live updates in the UI, **poll**: `/v1/bootstrap` is cheap
(server caches the snapshot) and gives you a fresh view. There is no SSE
or WebSocket on this service.

---

## 10. What's Missing (Not Shipped Today)

If you hit any of these and need them, file an issue against the
contract-management repo:

- **No OpenAPI spec.** Routes are hand-registered in
  [`internal/router/router.go`](../internal/router/router.go). A future
  pass will add `swag` or `oapi-codegen` annotations.
- **No SSE / WebSocket.** Use polling.
- **No declarative filter syntax** on list endpoints — fetch + filter
  client-side, or use `/v1/bootstrap`.
- **No workflow verbs** like `/approve` or `/complete` — status is a
  field, not a sub-resource. Frontend enforces valid transitions.
- **No batch endpoints.** Each create / patch / delete is one request.
- **No shared TS client package** (unlike `@iag/fleet-client`). Use
  `fetch` directly or extract one from the route table here.

---

## 11. Quickstart Checklist

For a new contract-management frontend project:

- [ ] Set `NEXT_PUBLIC_CONTRACTS_API_URL` and `NEXT_PUBLIC_AUTH_API_URL` (§2).
- [ ] Implement OAuth password-grant login against the auth service.
- [ ] Store access token in memory; set up silent refresh (§1).
- [ ] On app load, call `GET /v1/bootstrap` for the workspace snapshot
      (§4.2) — one round-trip, then maintain via individual mutations.
- [ ] Call `GET /v1/permissions/me` once at login; cache and gate UI on
      the returned `permissions` array (§3.4).
- [ ] For long contract lists, paginate with `page`/`pageSize` (§6); for
      smaller workspaces, just hold the snapshot.
- [ ] Polish status transitions on the client — the server doesn't
      enforce valid next-states (§5).
- [ ] Handle 401 → refresh, 403 → hide control, 409 → re-fetch (§7).
- [ ] If you want live updates, poll `/v1/bootstrap` every 30–60 s — no
      SSE on this service (§9).

---

## See Also

- [PLATFORM_INTEGRATION.md](./PLATFORM_INTEGRATION.md) — backend deployment
  + env config.
- [RAILWAY.md](./RAILWAY.md) — production rollout notes.
- Auth service `/oauth/token` —
  [shared/services/authentication](../../../../shared/services/authentication).
- Sibling guides:
  [Fleet](../../../operations/fleet/docs/FRONTEND_INTEGRATION.md) for
  service-to-service comparison.
