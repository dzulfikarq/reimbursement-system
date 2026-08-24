-- 000007 down :: restore departments + users.department_id
CREATE TABLE departments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(100) UNIQUE NOT NULL,
    monthly_budget NUMERIC(14,2),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE users ADD COLUMN department_id UUID REFERENCES departments(id);
CREATE INDEX idx_users_department ON users (department_id);
