# 03 — Database / ERD

PostgreSQL 16 + GORM. Migrations as raw SQL via `golang-migrate` (full control over constraints/indexes; GORM handles querying, not schema).

## ERD

```mermaid
erDiagram
    departments ||--o{ users : "has"
    departments ||--o{ department_budgets : "budget per month"
    users ||--o{ reimbursements : "submits"
    categories ||--o{ reimbursements : "categorizes"
    reimbursements ||--o{ reimbursement_items : "contains"
    reimbursements ||--o{ approvals : "approval chain"
    reimbursements ||--o{ attachments : "receipts"
    users ||--o{ approvals : "acts as approver"
    users ||--o{ audit_logs : "actor"

    departments {
        uuid id PK
        varchar name UK
        numeric monthly_budget
        timestamptz created_at
        timestamptz updated_at
    }
    users {
        uuid id PK
        uuid department_id FK
        varchar name
        citext email UK
        text password_hash
        user_role role "enum: employee|manager|finance|admin"
        boolean is_active
        timestamptz created_at
        timestamptz updated_at
    }
    categories {
        uuid id PK
        varchar code UK
        varchar name UK
        numeric monthly_limit_per_employee NULL
        boolean is_active
        timestamptz created_at
        timestamptz updated_at
    }
    department_budgets {
        uuid id PK
        uuid department_id FK
        date period_month "first day of month, UNIQUE with dept"
        numeric budget_amount
    }
    reimbursements {
        uuid id PK
        uuid employee_id FK
        uuid category_id FK
        varchar title
        text description NULL
        date expense_date
        numeric amount "14,2 — server-computed"
        reimb_status status "enum"
        int current_step
        timestamptz submitted_at NULL
        timestamptz decided_at NULL
        timestamptz paid_at NULL
        timestamptz cancelled_at NULL
        timestamptz created_at
        timestamptz updated_at
        timestamptz deleted_at NULL
    }
    reimbursement_items {
        uuid id PK
        uuid reimbursement_id FK,CASCADE
        varchar description
        int quantity CHECK >=1
        numeric unit_price "14,2 CHECK >0"
        numeric line_total "generated column"
    }
    approvals {
        uuid id PK
        uuid reimbursement_id FK,CASCADE
        int step_number
        approver_role role_required "enum: manager|finance|admin"
        uuid approver_id FK,NULL
        approval_status status "pending|approved|rejected"
        text note NULL
        timestamptz acted_at NULL
        UNIQUE(reimbursement_id, step_number)
    }
    attachments {
        uuid id PK
        uuid reimbursement_id FK,CASCADE
        uuid uploaded_by FK
        varchar storage_key "MinIO object key"
        varchar original_filename
        varchar mime_type
        bigint size_bytes
        timestamptz created_at
    }
    audit_logs {
        bigserial id PK
        uuid actor_id FK,NULL
        varchar action "e.g. APPROVE_CLAIM"
        varchar entity_type
        uuid entity_id NULL
        jsonb metadata
        inet ip_address NULL
        timestamptz created_at
    }
```

## Design Decisions

| Decision | Rationale |
|---|---|
| UUID PKs | Non-guessable ids in URLs/API; safe for distributed future. |
| Enum types (`user_role`, `reimb_status`, ...) | DB-level integrity for closed sets; app mirrors constants. |
| `approvals` snapshot rows | Approval path frozen at submit time; matrix changes don't rewrite history. |
| `line_total` generated column | DB guarantees arithmetic; app still recomputes header sum in a transaction. |
| Soft delete only on `reimbursements` | Claims must survive accidental deletion for audit; masters use hard delete + `is_active`. |
| `citext` email | Case-insensitive uniqueness without lower() index ceremony. |
| Money as `numeric(14,2)` | Never float for currency. |
| `audit_logs` separate table, append-only | No FK to business rows beyond entity_id → survives row deletion, cheap inserts. |

## Indexes & Constraints

```sql
-- hot paths from listing queries (doc 04)
CREATE INDEX idx_reimb_employee_status ON reimbursements (employee_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_reimb_status_submitted ON reimbursements (status, submitted_at DESC);
CREATE INDEX idx_reimb_expense_date ON reimbursements (expense_date);
CREATE INDEX idx_approvals_pending ON approvals (role_required, status) WHERE status = 'pending';
CREATE INDEX idx_items_reimbursement ON reimbursement_items (reimbursement_id);

-- integrity
ALTER TABLE users ADD CONSTRAINT uq_users_email UNIQUE (email);
ALTER TABLE department_budgets ADD CONSTRAINT uq_budget_period UNIQUE (department_id, period_month);
ALTER TABLE reimbursements ADD CONSTRAINT chk_amount_positive CHECK (amount > 0);
ALTER TABLE approvals ADD CONSTRAINT uq_approval_step UNIQUE (reimbursement_id, step_number);
```

## Migration Strategy

- Tool: `golang-migrate`, SQL files `NNNN_name.up.sql` / `.down.sql`.
- `make migrate-up` / `make migrate-down` (steps=1 default, all supported).
- First migration creates enums + extensions (`citext`); every later alter has a working down.
- Runs against empty database reproducibly; docker compose runs it on boot before API starts.

## Scaling Notes (interview topic)

- Listings are index-covered; pagination is keyset-ready (`submitted_at + id` cursor) if offset degrades.
- Dashboard aggregates move to Redis cache first, then materialized view if needed.
- Audit/attachment tables partition by month once > tens of millions of rows.
