# iag-contract-management

Commercial contract management microservice for the IAG platform.

Follows the same platform conventions as `iag-finance`, `iag-notifications`,
and the other Go services: RS256 Bearer tokens with `aud=iag.contract-management`
verified against the authentication service's JWKS, OpenTelemetry traces to
the platform collector, and a permission catalogue published to the
authentication service at boot.

## Local dev

In the meta-repo:

```bash
docker compose -f deploy/docker-compose.yml up -d contract-management
```

This brings up Postgres + authentication + otel-collector + jaeger + the
contract-management service on port `4103`. The service auto-runs migrations
and registers its permission catalogue with the auth service.

Standalone (no platform infrastructure):

```bash
cp .env.example .env   # edit DATABASE_URL if using service-local Postgres
docker compose up -d   # local Postgres only (port 5434)
go run .
```

Or inline env for a one-off run:

```bash
docker compose up -d                                   # local Postgres only
DATABASE_URL=postgres://cm:cm@localhost:5434/cm?sslmode=disable \
  JWT_ISSUER=http://localhost:3001 \
  AUDIENCE=iag.contract-management \
  go run .
```

## Auth model

- All HTTP routes under `/v1` require a Bearer token whose `aud` array
  contains `iag.contract-management`. Public exceptions: `/v1/health`,
  `/v1/health/live`, `/v1/health/ready`.
- Tokens are issued by the authentication service
  (`POST /api/v1/authentication/oauth/token` via the gateway). This service
  does NOT issue or store credentials.
- Session is derived from JWT claims by
  `internal/middleware/jwt.go::SessionFromClaims`:
  - `Role` is mapped from platform groups (`superadmin` → `super_admin`,
    `admin` → `admin`, `staff`/`manager` → `manager`, `viewer`/`user` →
    `viewer`).
  - A user listed in the local `contractor_supervisors` table gets
    `ContractorSup` populated; if their JWT role was `viewer`, they are
    additionally promoted to `contractor` for portal scoping.
- Permissions come from the JWT `permissions` claim. The service's
  permission catalogue is registered with the auth service at startup
  (`POST /v1/permissions/register`), so an admin can assign them to
  platform groups through the auth admin UI.

## Required env

| Var                       | Default                                 |
| ------------------------- | --------------------------------------- |
| `DATABASE_URL`            | (required)                              |
| `JWT_ISSUER`              | `http://localhost:3001`                 |
| `JWKS_URL`                | derived from `JWT_ISSUER`               |
| `AUDIENCE`                | `iag.contract-management`               |
| `SERVICE_CLIENT_ID`       | `iag-contract-management`               |
| `SERVICE_CLIENT_SECRET`   | (required in production)                |
| `AUTH_TOKEN_URL`          | derived from `JWT_ISSUER`               |
| `ALLOWED_ORIGINS`         | (required in production)                |
| `TRUSTED_PROXIES`         | (comma-separated proxy CIDRs; default empty = trust none) |
| `REQUEST_TIMEOUT_SECONDS` | `30`                                    |
| `RATE_LIMIT_PER_MINUTE`   | `120` (per pod, per client IP)          |
| `PORT`                    | `4103`                                  |

In production:
- The initial JWKS fetch is **blocking**: a misconfigured `JWKS_URL` will
  cause the service to fail fast instead of serving 401s for 15 minutes.
- `TRUSTED_PROXIES` should list the platform gateway's IPs/CIDRs, otherwise
  `X-Forwarded-For` is ignored and the rate limiter sees one bucket for
  everyone.
- Permissions registration runs in the background with exponential backoff
  (capped at 5 min) so a transient auth-service outage at boot doesn't
  leave the service permanently un-registered.

## Smoke test

```bash
./scripts/smoke_test.sh \
    http://localhost:4103/v1 \
    http://localhost:3001/oauth/token \
    admin@iag.local changeme
```

## Production / Railway

- Production env template: `config/.env.production.example`
- Railway config-as-code: `railway.toml` (health probe `/ready`, `PORT=4103`)
- Deploy guide: [docs/RAILWAY.md](docs/RAILWAY.md)
- Platform wiring: [docs/PLATFORM_INTEGRATION.md](docs/PLATFORM_INTEGRATION.md)
