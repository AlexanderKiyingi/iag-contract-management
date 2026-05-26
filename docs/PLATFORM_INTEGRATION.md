# IAG Contract Management — platform integration

Go/Gin service behind the **API gateway**, using **iag-authentication** for
identity (JWKS + `aud=iag.contract-management`) and **Postgres** (`iag_contracts`
schema) for contract workspace data.

## Services

| Service | Integration |
|---------|-------------|
| **iag-authentication** | RS256 JWT verification; permission catalogue registered at boot |
| **iag-api-gateway** | Public ingress at `/api/v1/contract-management/v1/...` |
| **Postgres** | Schema `iag_contracts`, role `svc_iag_contracts` |

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

See `config/.env.production.example` for a full production template.

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

## Railway

See [RAILWAY.md](./RAILWAY.md).
