-- 000003 :: categories master
CREATE TABLE categories (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                        VARCHAR(20) NOT NULL UNIQUE,
    name                        VARCHAR(100) NOT NULL UNIQUE,
    monthly_limit_per_employee  NUMERIC(14,2) CHECK (monthly_limit_per_employee IS NULL OR monthly_limit_per_employee > 0),
    is_active                   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_categories_name ON categories (name);
