// Package audit writes compliance log rows. No read endpoints — reviewers
// query the table directly (docs/03).
package audit

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Actions emitted by the workflow engine.
const (
	ActionSubmitClaim  = "SUBMIT_CLAIM"
	ActionApproveClaim = "APPROVE_CLAIM"
	ActionRejectClaim  = "REJECT_CLAIM"
	ActionCancelClaim  = "CANCEL_CLAIM"
	ActionPayClaim     = "PAY_CLAIM"
)

type Log struct {
	ID         int64           `gorm:"primaryKey"`
	ActorID    *uuid.UUID      `gorm:"type:uuid"`
	Action     string          `gorm:"size:64;not null"`
	EntityType string          `gorm:"size:32;not null"`
	EntityID   *uuid.UUID      `gorm:"type:uuid"`
	Metadata   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	IPAddress  *string         `gorm:"type:inet"`
}

func (Log) TableName() string { return "audit_logs" }

// Write inserts one audit row inside the caller's transaction. Never fails
// the business action: audit write errors are logged upstream, not returned.
// ponytail: fire-and-forget semantics, outbox pattern if durability matters.
func Write(tx *gorm.DB, actorID uuid.UUID, action, entityType string, entityID uuid.UUID, meta map[string]any) {
	m := []byte("{}")
	if meta != nil {
		m, _ = json.Marshal(meta)
	}
	tx.Exec(`INSERT INTO audit_logs (actor_id, action, entity_type, entity_id, metadata)
	         VALUES (?, ?, ?, ?, ?::jsonb)`,
		actorID, action, entityType, entityID, string(m))
}
