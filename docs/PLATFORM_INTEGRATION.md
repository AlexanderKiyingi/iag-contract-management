# IAG Contract Management — platform integration

Go/Gin service behind the **API gateway**, using **iag-authentication** for
identity (JWKS + `aud=iag.contract-management`) and **Postgres** (`iag_contracts`
schema) for contract workspace data.

## Services

| Service | Integration |
|---------|-------------|
| **iag-authentication** | RS256 JWT verification; permission catalogue registered at boot |
| **iag-api-gateway** | Public ingress at `/api/v1/contract-management/v1/...` |
| **iag-notifications** | Consumes `contracts.alert.raised` on `iag.commercial` |
| **Postgres** | Schema `iag_contracts`, role `svc_iag_contracts` |
| **Kafka (Redpanda)** | Publishes domain events to `iag.commercial` when `EVENT_BUS_ENABLED=true` |

## Environment

| Variable | Purpose |
|----------|---------|
| `PORT` | Listen port (default `4103`; Railway health checks use this) |
| `DATABASE_URL` | Postgres (`svc_iag_contracts` on `iag_platform`) |
| `JWT_ISSUER` / `JWKS_URL` | Platform token verification |
| `AUDIENCE` | `iag.contract-management` |
| `SERVICE_CLIENT_ID` / `SERVICE_CLIENT_SECRET` | Outbound calls (permissions register) |
| `ALLOWED_ORIGINS` | CORS (required in production) |
| `TRUSTED_PROXIES` | Gateway/load-balancer CIDRs for correct client IP |
| `EVENT_BUS_ENABLED` | `true` to publish to Kafka |
| `KAFKA_BROKERS` | Comma-separated broker list |
| `NOTIFY_DEFAULT_RECIPIENT` | Email for assistance/milestone alerts |
| `MILESTONE_REMINDER_DAYS` | Jobs CLI: due-soon window (default `7`) |

See `config/.env.production.example` for a full production template.

## Events (Kafka)

Topic: **`iag.commercial`** (source `iag.contract-management`)

| Type | When |
|------|------|
| `contracts.contract.created` | Contract created |
| `contracts.contract.updated` | Contract patched |
| `contracts.contract.status_changed` | Status field changed |
| `contracts.contract.deleted` | Contract deleted |
| `contracts.assistance.requested` | Assistance message posted |
| `contracts.milestone.due_soon` | Jobs CLI finds milestone within reminder window |
| `contracts.alert.raised` | Notification dispatch envelope (→ iag-notifications) |

## Jobs

Run milestone reminders (cron / Railway scheduled job):

```bash
/app/jobs --milestone-reminders
```

Compose one-shot worker: `contract-management-jobs` (same image, different command).
`restart: "no"` — it runs once per `docker compose up`; use Railway cron or
`scripts/run-contract-management-jobs.ps1` in the meta-repo for recurring local runs.

## Realtime (intentional split)

| Concern | Where it lives |
|---------|----------------|
| Workspace data freshness | Frontend polls `GET /v1/bootstrap` (or granular REST after mutations) |
| In-app notification inbox | **iag-notifications** WebSocket/SSE (`aud` must include `iag.notifications`) |
| Milestone due-soon email/in-app | Jobs CLI → Kafka `contracts.milestone.due_soon` → notifications pipeline |

This service does **not** ship a workspace WebSocket or Kafka consumer (see
project-management for that pattern). Adding either would require Redis fan-out
and/or consumer wiring — out of scope for the current contract-management v1 API.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ready` | Readiness (Postgres ping) — gateway/Railway probe |
| GET | `/v1/health/ready` | Same readiness under canonical API prefix |
| GET | `/v1/contracts` | List contracts (`contracts.read`) |
| POST | `/v1/contracts` | Create contract (`contracts.create`) |
| GET | `/v1/bootstrap` | Session bootstrap (authenticated) |
| GET | `/v1/exports/contracts.csv` | CSV export (`reports.read` + `contracts.read`) |

Login and token refresh live on authentication:
`POST /api/v1/authentication/oauth/token`.

## Local development

```bash
# Full platform stack
docker compose -f deploy/docker-compose.yml up -d contract-management

# Via gateway
curl http://localhost:8080/api/v1/contract-management/ready

# Smoke test
./scripts/smoke_test.sh
```

## Document storage (S3-compatible)

Governance contract documents are uploaded to an S3-compatible bucket (AWS S3 /
Cloudflare R2 / MinIO) using **presigned URLs** — the service signs URLs
(stdlib SigV4, path-style) and the browser PUTs/GETs the object directly; file
bytes never pass through the service.

Configure via env (`S3_ENDPOINT`, `S3_REGION`, `S3_BUCKET`, `S3_ACCESS_KEY_ID`,
`S3_SECRET_ACCESS_KEY`, `S3_USE_SSL`). When unset the upload endpoints return
`503` and the rest of the service is unaffected.

**Bucket CORS is required** — the browser uploads/downloads cross-origin, so the
bucket must allow `PUT` and `GET` (and `OPTIONS` preflight) from the app origin,
e.g.:

```json
[{ "AllowedOrigins": ["https://<app-origin>"],
   "AllowedMethods": ["GET", "PUT"],
   "AllowedHeaders": ["*"],
   "ExposeHeaders": ["ETag"] }]
```

Endpoints (under `/v1/governance`): `POST /contracts/:id/documents/presign`,
`POST /contracts/:id/documents`, `DELETE /contracts/:id/documents/:docId`,
`GET /documents/url?key=`, and the portal download `GET
/portal/contracts/:id/documents/:docId/url`.

## Railway

See [RAILWAY.md](./RAILWAY.md).
