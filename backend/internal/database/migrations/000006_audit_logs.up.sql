-- 000006 :: audit_logs — who did what, for compliance review
CREATE TABLE audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    actor_id    UUID REFERENCES users(id),
    action      VARCHAR(64) NOT NULL,
    entity_type VARCHAR(32) NOT NULL,
    entity_id   UUID,
    metadata    JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address  INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_entity ON audit_logs (entity_type, entity_id, created_at);
CREATE INDEX idx_audit_logs_actor ON audit_logs (actor_id, created_at);
