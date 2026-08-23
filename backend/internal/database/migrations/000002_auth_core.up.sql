-- 000002 :: auth core — roles, departments, users
CREATE TYPE user_role AS ENUM ('employee', 'manager', 'finance', 'admin');

CREATE TABLE departments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(100) NOT NULL UNIQUE,
    monthly_budget NUMERIC(14,2),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    department_id UUID REFERENCES departments(id),
    name          VARCHAR(100) NOT NULL,
    email         CITEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          user_role NOT NULL DEFAULT 'employee',
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_department ON users (department_id);
