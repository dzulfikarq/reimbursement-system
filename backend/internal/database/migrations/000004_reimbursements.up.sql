-- 000004 :: reimbursements core — claims, items, attachments
CREATE TYPE reimb_status AS ENUM ('DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED', 'PAID', 'CANCELLED');

CREATE TABLE reimbursements (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id   UUID NOT NULL REFERENCES users(id),
    category_id   UUID NOT NULL REFERENCES categories(id),
    title         VARCHAR(150) NOT NULL,
    description   TEXT,
    expense_date  DATE NOT NULL,
    amount        NUMERIC(14,2) NOT NULL DEFAULT 0 CONSTRAINT chk_amount_positive CHECK (amount >= 0),
    status        reimb_status NOT NULL DEFAULT 'DRAFT',
    current_step  INT NOT NULL DEFAULT 0,
    submitted_at  TIMESTAMPTZ,
    decided_at    TIMESTAMPTZ,
    paid_at       TIMESTAMPTZ,
    cancelled_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE TABLE reimbursement_items (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reimbursement_id UUID NOT NULL REFERENCES reimbursements(id) ON DELETE CASCADE,
    description      VARCHAR(200) NOT NULL,
    quantity         INT NOT NULL CHECK (quantity >= 1),
    unit_price       NUMERIC(14,2) NOT NULL CHECK (unit_price > 0 AND unit_price <= 999999999),
    line_total       NUMERIC(16,2) GENERATED ALWAYS AS (quantity * unit_price) STORED,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE attachments (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reimbursement_id  UUID NOT NULL REFERENCES reimbursements(id) ON DELETE CASCADE,
    uploaded_by       UUID NOT NULL REFERENCES users(id),
    storage_key       VARCHAR(500) NOT NULL UNIQUE,
    original_filename VARCHAR(255) NOT NULL,
    mime_type         VARCHAR(100) NOT NULL,
    size_bytes        BIGINT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reimb_employee_status ON reimbursements (employee_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_reimb_status_submitted ON reimbursements (status, submitted_at DESC);
CREATE INDEX idx_reimb_expense_date ON reimbursements (expense_date);
CREATE INDEX idx_items_reimbursement ON reimbursement_items (reimbursement_id);
