# Deploying contract-management on Railway

## GitHub repo

| Setting | Value |
|---------|--------|
| Repository | `AlexanderKiyingi/iag-contract-management` |
| Branch | `main` |
| Root directory | **`/`** (repo root — `Dockerfile`, `railway.toml`, and `railway.json` must be here) |

The **standalone** Dockerfile target (default) uses the committed
`third_party/platform-go` vendored copy. No private git clone or GitHub token is
required at build time.

If wired to `IAG_multi_backend` instead, set **Root directory** to the
meta-repo root, **Dockerfile path** to
`services/commercial/contract-management/Dockerfile`, and build target
**`monorepo`**.

## Builder (Dockerfile, not Railpack)

Railway defaults new services to **Railpack**. This service must use the
**Dockerfile** builder.

Config-as-code (both files are intentional — Railway reads either):

- `railway.toml` — `[build] builder = "DOCKERFILE"`
- `railway.json` — `"builder": "DOCKERFILE"`, `"dockerfilePath": "Dockerfile"`

### If deployment metadata shows RAILPACK

1. Confirm **Root directory** is `/` for the standalone repo (not a monorepo subpath).
2. Confirm `Dockerfile`, `railway.toml`, and `railway.json` exist at that root.
3. In the service **Settings → Build**, set **Builder** to **Dockerfile** and
   **Dockerfile path** to `Dockerfile`.
4. Optional env fallback: `RAILWAY_DOCKERFILE_PATH=Dockerfile`
5. Trigger a **manual redeploy** from the dashboard after changing builder settings
   (git-push deploys have intermittently ignored config-as-code on Railway).

Successful build logs should show Docker stages (`build-standalone`, `standalone`),
not `Railpack 0.x.x`.

## Postgres

Use the shared `iag_platform` database with role `svc_iag_contracts` and schema
`iag_contracts`. Bootstrap scripts: `deploy/postgres/init/` in the meta-repo.

1. Add **PostgreSQL** in the Railway project (or reference an existing instance).
2. Remove any `DATABASE_URL` containing `localhost` or `127.0.0.1`.
3. Reference the Postgres plugin `DATABASE_URL`, then adjust if needed:

```text
postgresql://svc_iag_contracts:PASSWORD@HOST:5432/iag_platform?sslmode=require
```

Run `01-schemas.sql` and `02-service-roles.sh` once against the managed instance
before first deploy.

## Required variables

Copy from `config/.env.production.example`. Minimum for a healthy deploy:

| Variable | Notes |
|----------|--------|
| `PORT` | `4103` (Dockerfile default; Railway probes this) |
| `ENVIRONMENT` | `production` |
| `DATABASE_URL` | `svc_iag_contracts` role, `sslmode=require` |
| `JWT_ISSUER` / `JWKS_URL` | Public authentication service URL |
| `SERVICE_CLIENT_SECRET` | For permission catalogue registration |
| `ALLOWED_ORIGINS` | Required in production (CORS) |

## Health checks

Railway and Docker probe **`GET /ready`** (Postgres ping). Canonical API paths
also exist at `/v1/health/ready`. Do **not** set `ADDR` without matching `PORT`.

## Migrations

Schema migrations run **automatically on every startup** via embedded SQL in
`persistence.Connect`. No `AUTO_MIGRATE` flag — plan rollbacks with DB backups.

## Gateway

Set on **iag-api-gateway**:

```text
UPSTREAM_CONTRACT_MANAGEMENT=http://<railway-private-host>:4103
```

Public readiness via gateway: `GET /api/v1/contract-management/ready`.

## Common failures

| Symptom | Fix |
|---------|-----|
| Instant fail, metadata shows **RAILPACK**, no Docker logs | Set builder to Dockerfile (see above); verify root directory `/` |
| `Railpack could not determine how to build` | Same — Railpack must not run for this Go service |
| Connection refused on `127.0.0.1:5432` | Replace `DATABASE_URL` with Railway Postgres reference |
| Boot loop / JWKS error | Fix `JWKS_URL`; production requires successful initial JWKS fetch |
| 503 on `/ready` | Postgres unreachable or schema bootstrap not applied |
| Health check 404 | Ensure `PORT=4103` and probe path is `/ready` |
| Events not published | Set `EVENT_BUS_ENABLED=true` and `KAFKA_BROKERS` |
| `third_party/platform-go` missing in build | Run `sh scripts/sync-platform-go.sh` and commit before deploy |

## Scheduled jobs (Railway cron)

Add a second Railway service (same repo/image) or a cron schedule:

```text
Command: /app/jobs --milestone-reminders
```

Required env: `DATABASE_URL`, `EVENT_BUS_ENABLED=true`, `KAFKA_BROKERS`, optional `MILESTONE_REMINDER_DAYS=7`, `NOTIFY_DEFAULT_RECIPIENT`.

## Updating platform-go

When `shared/platform-go` changes in the meta-repo:

```bash
sh scripts/sync-platform-go.sh
git add third_party/platform-go
git commit -m "chore: sync vendored platform-go"
```
