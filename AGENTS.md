# AGENTS.md

Guidance for AI coding agents working on this repository. Design documents live in `docs/` — read them before non-trivial changes.

## Project

Reimbursement management system (take-home test, PT Mumtaz Teknologi Indonesia). Employees submit itemized expense claims with receipts; claims pass a tiered approval workflow until Finance marks them paid.

- Docs index: `docs/README.md`
- Key docs: business rules `docs/02`, schema `docs/03`, API contract `docs/04`, architecture decisions `docs/06`

## Stack (fixed — do not substitute)

| Layer | Choice |
|---|---|
| Backend | Go + Gin + GORM + PostgreSQL 16 |
| Auth | Cookie-based JWT access (15m) + opaque rotating refresh (7d, Redis) |
| Frontend | React + TypeScript + Vite + Tailwind CSS + Axios |
| FE state | TanStack Query (server), Zustand (session), react-hook-form + zod (forms) |
| Storage | MinIO (S3-compatible), presigned URLs for downloads |
| Queue | Redis via asynq (`email.send`, `report.generate`) |
| Migrations | golang-migrate, raw SQL files |
| Docs | Swagger via swaggo annotations |

## Commands

```bash
docker compose up -d          # full dev stack
make -C backend run           # api :8080
make -C backend migrate-up    # / migrate-down
make -C backend test
make -C backend seed
make -C backend swagger
cd frontend && npm run dev    # :5173 proxies /api → :8080
```

## Architecture Rules

- Modular layered: `handler → service → repository`. Business logic lives in **services only**; handlers bind/validate/respond; repositories do queries.
- New feature = new module under `backend/internal/modules/<name>/` with files `handler.go service.go repository.go dto.go routes.go`.
- Cross-cutting helpers go in `backend/internal/pkg/`; never create a shared "utils" dump.
- Frontend pages compose components from `src/components/ui/` — extend existing components instead of duplicating; feature logic stays inside `src/features/<name>/`.

## Non-Negotiables

1. **Auth via cookies only.** Never put tokens in localStorage/sessionStorage or Authorization headers. Cookies: access `HttpOnly; Secure; SameSite=Lax; Path=/`; refresh `HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth`.
2. **CSRF on every mutating request** (double-submit signed token, `X-CSRF-Token` header). New mutation endpoints are automatically covered by middleware — don't add bypasses.
3. **Backend is source of truth.** Frontend checks mirror backend rules for UX only. Never trust client-computed amounts — recompute server-side from items.
4. **Validate every input**: body, query, path, file uploads. Sort fields and limits come from whitelists. File uploads: MIME allowlist (png/jpeg/webp/pdf) checked by magic bytes, max 5 MB.
5. **Response envelope is fixed** (`docs/04`): success `{success, data, message}`, error `{success:false, error:{code,message,details}, request_id}`. Use correct status codes (200/201/204/400/401/403/404/409/422/429/500).
6. **Never expose internals**: DB errors, stack traces, secrets. Map errors through `pkg/apperr` → central handler. Unexpected errors log with request_id and return generic `INTERNAL_ERROR`.
7. **Migrations are hand-written SQL** with working `.down`. No GORM AutoMigrate. Money is `numeric(14,2)`, PKs are UUIDs.
8. **State transitions validate server-side** against the state machine in `docs/02`. Transitions run in transactions with `SELECT ... FOR UPDATE` on the claim row.

## Business Rules Quick Reference (details: docs/02)

- Approval matrix by amount: ≤500k manager; ≤5M manager+finance; >5M manager+finance+admin. Steps snapshotted into `approvals` at submit time.
- Header amount must equal Σ(qty × unit_price).
- Category monthly limit per employee enforced at submit; violations return `422 BUSINESS_RULE_VIOLATED` with **all** violations listed at once.
- Receipt required when amount > Rp 500.000.
- Duplicate block: same employee + same amount + expense date within ±7 days of active claim → `409 DUPLICATE_SUSPECTED`.
- Edit: owner, DRAFT/REJECTED only. Delete: owner, DRAFT only. Cancel: owner, SUBMITTED before any approval step acted. Reject requires note.
- Listing scope: employee=own, manager=own department, finance/admin=all.

## API Conventions

- Everything under `/api/v1`. List endpoints accept `page, limit, search, sort, order, status, category_id, date_from, date_to`.
- New endpoints need swaggo annotations + doc update in `docs/04-api-contract.md`.

## Frontend Requirements

Every listing/detail screen implements: loading skeleton, empty state, error state with retry, disabled-during-mutation states. Forms: inline validation, error messages, loading submit button, success toast. Destructive actions require ConfirmDialog. Axios interceptors behave exactly as `docs/05` (single-flight refresh, no infinite retry, 403/422/429 mapping).

## Testing

- Service-layer unit tests for state machine, policy engine, approval matrix — mock repositories.
- Integration tests (httptest + real PG/Redis) for auth and reimbursement flows.
- Any new business rule branch ships with a test covering both accept and reject paths.

## Git

Conventional Commits (`feat:`, `fix:`, `test:`, `chore:`...). One branch per milestone (`feat/m1-auth-core`). Never commit `.env`, credentials, or seed passwords beyond demo values.

## Milestone Order

Follow `docs/08-implementation-milestones.md` M0→M8. Don't start a milestone before the previous one meets its "done when" criteria. When unsure where code belongs or which rule applies, check the docs first, then ask.
