# 04 — API Contract

Base: `/api/v1`. JSON only. Cookie auth (`access_token`, HttpOnly) + `X-CSRF-Token` header on mutations.

## Conventions

Success:

```json
{ "success": true, "data": {}, "message": "OK" }
```

Lists wrap in `data.items` + `data.pagination`:

```json
{
  "success": true,
  "data": {
    "items": [],
    "pagination": { "page": 1, "limit": 20, "total_items": 137, "total_pages": 7 }
  }
}
```

Error (centralized handler):

```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": [{ "field": "amount", "message": "must be greater than 0" }]
  },
  "request_id": "b7c9e2..."
}
```

| HTTP | code | When |
|---|---|---|
| 400 | `BAD_REQUEST` | Malformed body/query |
| 401 | `UNAUTHORIZED` | Missing/expired/invalid session |
| 403 | `FORBIDDEN` | Role lacks permission / CSRF failed |
| 404 | `NOT_FOUND` | Missing resource or out of scope |
| 409 | `CONFLICT` | Illegal state transition, duplicate suspect |
| 422 | `VALIDATION_ERROR` / `BUSINESS_RULE_VIOLATED` | Field errors / policy violations |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Unhandled; internals never exposed |

List query contract: `?page=1&limit=20&search=&sort=created_at&order=desc&status=&category_id=&date_from=&date_to=`. Sort fields come from a per-resource whitelist; every param validated.

## Endpoints

### Auth

| Method | Path | Auth | Notes |
|---|---|---|---|
| POST | `/auth/login` | none | `{email, password}` → sets HttpOnly cookies (`access_token`, `refresh_token`) + readable `csrf_token`; rate-limited per IP+email |
| POST | `/auth/logout` | yes | Revokes refresh token (Redis), clears cookies → `204` |
| POST | `/auth/refresh` | refresh cookie | Rotates both tokens; single-use refresh jti, race-safe |
| GET | `/auth/me` | yes | Current user + role + department |
| GET | `/auth/csrf` | cookie | Issues fresh CSRF token |

### Master data

| Method | Path | Who | Notes |
|---|---|---|---|
| GET | `/departments`, `/categories` | any authed | Needed for forms; search/pagination |
| POST/PATCH/DELETE | `/departments/:id`, `/categories/:id` | Admin | Delete blocked when referenced → `409` |
| GET | `/users` | Admin | Search, filter by role/department, pagination |
| POST | `/users` | Admin | Creates user with initial password |
| PATCH | `/users/:id` | Admin | Role/department/is_active changes |
| POST | `/users/:id/reset-password` | Admin | |

### Reimbursements

| Method | Path | Who | Notes |
|---|---|---|---|
| GET | `/reimbursements` | scoped | Employee=own, Manager=department, Finance/Admin=all; full SFP |
| POST | `/reimbursements` | employee+ | Creates DRAFT with items; amount computed server-side |
| GET | `/reimbursements/:id` | owner/scope | Items + approval timeline + attachment metadata |
| PATCH | `/reimbursements/:id` | owner | DRAFT/REJECTED only |
| DELETE | `/reimbursements/:id` | owner | DRAFT only → `204` |
| POST | `/reimbursements/:id/submit` | owner | Policy checks → SUBMITTED; generates approval steps |
| POST | `/reimbursements/:id/cancel` | owner | Only before first step acted |
| POST | `/reimbursements/:id/approve` | current approver | Acts on pending step matching role |
| POST | `/reimbursements/:id/reject` | current approver | `note` required |
| POST | `/reimbursements/:id/pay` | Finance | APPROVED → PAID |
| POST | `/reimbursements/:id/attachments` | owner | multipart `file`; ≤5 MB, MIME allowlist |
| GET | `/attachments/:id/download` | claim-scoped | 302 → MinIO presigned URL (60 s) |

### Dashboard & misc

| Method | Path | Who |
|---|---|---|
| GET | `/dashboard/summary` | Role-scoped: pending count, monthly total, approval rate, budget usage |
| GET | `/dashboard/monthly-trend?months=6` | Role-scoped series |
| GET | `/dashboard/category-breakdown?month=` | Role-scoped |
| GET | `/reports/export?month=&status=` | Finance/Admin → queues job, returns job id |
| GET | `/reports/export/:jobId` | Poll status → download URL when done |
| GET | `/healthz` | Liveness + db/redis/minio readiness flags |

## Key Payloads

POST `/auth/login` → 200:

```json
{
  "success": true,
  "data": {
    "user": {
      "id": "uuid", "name": "Budi", "role": "employee",
      "department": { "id": "uuid", "name": "Engineering" }
    }
  }
}
```

POST `/reimbursements` → 201:

```json
{
  "title": "Client visit Jakarta",
  "category_id": "uuid",
  "expense_date": "2026-08-10",
  "description": "Transport and meals",
  "items": [
    { "description": "Train tickets", "quantity": 2, "unit_price": 175000 },
    { "description": "Team lunch", "quantity": 4, "unit_price": 60000 }
  ]
}
```

Response includes server-computed `amount: 590000`.

POST `.../submit` → 422 with all violations at once:

```json
{
  "success": false,
  "error": {
    "code": "BUSINESS_RULE_VIOLATED",
    "message": "Policy check failed",
    "details": [
      { "field": "receipts", "message": "at least 1 receipt required above Rp 500.000" },
      { "field": "amount", "message": "exceeds Medical monthly limit (remaining Rp 300.000)" }
    ]
  }
}
```

POST `.../reject` body: `{ "note": "Receipt number unreadable" }`.

## Documentation Artifact

OpenAPI 3.1 spec served by Swagger UI at `/swagger/index.html`, generated from annotated handlers (swaggo). Every endpoint documented with auth, params, request/response examples, and error codes.
