-- 000005 :: approvals — snapshot of the approval chain frozen at submit time
CREATE TYPE role_required AS ENUM ('manager', 'finance', 'admin');
CREATE TYPE approval_status AS ENUM ('pending', 'approved', 'rejected');

CREATE TABLE approvals (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reimbursement_id  UUID NOT NULL REFERENCES reimbursements(id) ON DELETE CASCADE,
    step_number       INT NOT NULL CHECK (step_number >= 1),
    approver_role     role_required NOT NULL,
    approver_id       UUID REFERENCES users(id),
    status            approval_status NOT NULL DEFAULT 'pending',
    note              TEXT,
    acted_at          TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_approval_step UNIQUE (reimbursement_id, step_number)
);

CREATE INDEX idx_approvals_pending ON approvals (approver_role, status) WHERE status = 'pending';
