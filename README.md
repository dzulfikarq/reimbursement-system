# Reimbursement Management System

Fullstack take-home test — PT Mumtaz Teknologi Indonesia. Employees submit itemized expense claims with receipts; claims pass a tiered approval workflow (Manager → Finance → Admin by amount) until Finance marks them paid.

- Design documents: [`docs/`](./docs/README.md) (business rules `docs/02`, schema `docs/03`, API contract `docs/04`, architecture decisions `docs/06`)
- Agent guidance: [`AGENTS.md`](./AGENTS.md)

## Quick start

```bash
cp .env.example .env   # defaults work out of the box
docker compose up --build
docker compose exec api /app/migrate up     # schema migrations
docker compose exec api /app/seed           # demo data (idempotent-ish; see seeds/main.go)
```

| Service | URL |
|---|---|
| Frontend | http://localhost:5173 |
| API | http://localhost:8080/api/v1 |
| Swagger UI | http://localhost:8080/swagger/index.html |
| Health | http://localhost:8080/healthz |
| MinIO console | http://localhost:9001 (minioadmin/minioadmin) |
| MailHog | http://localhost:8025 |

### Demo accounts

| Email | Password | Role | Scope |
|---|---|---|---|
| admin@mumtaz.test | Admin#12345 | admin | everything + user management |
| finance@mumtaz.test | Finance#12345 | finance | all claims, approvals step 2+, payments, exports |
| manager.eng@mumtaz.test | Manager#12345 | manager | own department's claims, approval step 1 |
| employee.eng@mumtaz.test | Employee#12345 | employee | own claims only |

## Architecture

Modular layered backend (`handler → service → repository`; business logic in services only), React SPA frontend composing a small Tailwind-based UI kit.

```
backend/
  cmd/api/          HTTP server entrypoint (router wiring lives in internal/server)
  cmd/worker/       asynq worker: email.send, report.generate
  internal/
    modules/<name>/ handler.go service.go repository.go dto.go (+ routes.go pattern)
    middleware/      authn, csrf, role guard, rate limit, request id, security headers…
    pkg/             apperr, csrftoken, jwt, listq, password, response, upload
    database/        golang-migrate SQL migrations (hand-written, with .down)
frontend/
  src/
    components/ui/   design-system primitives (Button, Table, Modal, Toast…)
    features/pages   pages compose primitives; feature logic stays in the page/lib layer
    lib/             axios instance (interceptors per docs/05), typed query hooks
    stores/          zustand: session mirror, ui state, toasts
docs/               design documents (source of truth for rules)
```

Key flows:

- **Auth** — cookie-based JWT access (15 min) + opaque rotating refresh (7 days, Redis-backed). Tokens never touch JS storage; CSRF double-submit token on every mutation (`X-CSRF-Token`).
- **Approval workflow** — steps snapshotted into `approvals` at submit time from an amount-based matrix (≤500k manager; ≤5M manager+finance; >5M manager+finance+admin). Transitions validate against the state machine in transactions with `SELECT … FOR UPDATE`.
- **Policy engine** — header amount must equal Σ(qty × price); receipt required > Rp 500k; category monthly limit per employee; duplicate block (same employee + amount within ±7 days). Violations return batched `422 BUSINESS_RULE_VIOLATED`.
- **Async jobs** — claim notifications land in MailHog; CSV export queued via asynq, uploaded to MinIO `exports`, downloaded through presigned URLs.
- **Rate limiting** — per-IP token buckets: login 5/burst·10s, global 40/s burst on `/api/v1`. Exceeding returns `429 RATE_LIMITED`.

## Environment

All configuration is environment-driven (see `.env.example`); notable variables:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | 8080 | API port |
| `DB_*` | postgres/reimbursement | PostgreSQL connection |
| `REDIS_ADDR` | redis:6379 | cache + queue + refresh sessions |
| `MINIO_*` | minio:9000 | attachments bucket |
| `MINIO_PUBLIC_ENDPOINT` | – | host used in presigned download URLs (set to `localhost:9000` locally) |
| `SMTP_ADDR` | mailhog:1025 | worker notification sink |
| `APP_SECRET` | – | HMAC secret for JWT + CSRF (change in prod!) |
| `APPROVAL_T1/T2` | 500000 / 5000000 | approval matrix thresholds |

## Testing

```bash
# Backend: unit (state machine, matrix, rate limiter) + integration (real PG/Redis).
# Integration tests auto-skip when SKIP_INTEGRATION_TESTS=1 or no DB is reachable.
make -C backend test

# Frontend: vitest (axios interceptor contract: CSRF attach, single-flight
# refresh, no infinite retry).
cd frontend && npm test -- --run
```

CI (`.github/workflows/ci.yml`) runs vet/build/tests with service containers for Postgres and Redis, plus the frontend typecheck/test/build.

## API

Interactive docs at `/swagger/index.html` (generated via swaggo annotations; regenerate with `swag init -g cmd/api/main.go --parseDependency --output docs`). The human-readable contract lives in [`docs/04-api-contract.md`](./docs/04-api-contract.md).

## Status

Milestones tracked in [`docs/08-implementation-milestones.md`](./docs/08-implementation-milestones.md): M0–M8 complete (skeleton, auth, master data, claims, workflow engine, dashboards & audit, worker & notifications, frontend, hardening & docs).
