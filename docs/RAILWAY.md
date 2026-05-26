# Deploying contract-management on Railway

## GitHub repo

| Setting | Value |
|---------|--------|
| Repository | `AlexanderKiyingi/iag-contract-management` |
| Branch | `main` |
| Root directory | `/` (repo root — `Dockerfile` and `railway.toml` are here) |

If wired to `IAG_multi_backend` instead, set **Root directory** to the
meta-repo root and **Dockerfile path** to
`services/commercial/contract-management/Dockerfile` (build context must include
`shared/platform-go`).

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
| Connection refused on `127.0.0.1:5432` | Replace `DATABASE_URL` with Railway Postgres reference |
| Boot loop / JWKS error | Fix `JWKS_URL`; production requires successful initial JWKS fetch |
| 503 on `/ready` | Postgres unreachable or schema bootstrap not applied |
| Health check 404 | Ensure `PORT=4103` and probe path is `/ready` |
