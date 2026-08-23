package reimbursements

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	gormclause "gorm.io/gorm/clause"
	"gorm.io/gorm"

	"github.com/mumtaz/reimbursement-system/backend/internal/config"
	apperr "github.com/mumtaz/reimbursement-system/backend/internal/pkg/apperr"
)

// ApprovalStep mirrors the approvals snapshot rows.
type ApprovalStep struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	ReimbursementID uuid.UUID
	StepNumber      int
	ApproverRole    string
	ApproverID      *uuid.UUID
	Status          string
	Note            *string
	CreatedAt       time.Time
}

func (ApprovalStep) TableName() string { return "approvals" }

// WorkflowService owns submit + decision actions. Every transition opens a
// transaction and locks the claim row with SELECT ... FOR UPDATE so
// concurrent double-actions serialize instead of corrupting state (AGENTS.md
// non-negotiable #8).
type WorkflowService struct {
	cfg  *config.Config
	repo *Repository
	db   *gorm.DB
}

func NewWorkflowService(cfg *config.Config, repo *Repository, db *gorm.DB) *WorkflowService {
	return &WorkflowService{cfg: cfg, repo: repo, db: db}
}

// Submit runs the full policy engine then flips DRAFT/REJECTED → SUBMITTED,
// regenerating the approval snapshot. Violations accumulate — client sees
// every problem in one round-trip (docs/02 FR-3).
func (w *WorkflowService) Submit(ctx context.Context, id uuid.UUID, role string, userID, deptID uuid.UUID) (*DetailResponse, error) {
	current, err := w.repo.GetDetail(ctx, id, role, userID, deptID)
	if err != nil {
		return nil, err
	}
	if current.EmployeeID != userID {
		return nil, apperr.Forbidden("Only the owner can submit this claim")
	}
	if !canTransition(current.Status, "submit") {
		return nil, apperr.Conflict("Only DRAFT or REJECTED claims can be submitted")
	}

	var violations []apperr.Detail

	// Rule 3 — receipt required above threshold.
	if compareNumeric(current.Amount, w.cfg.ReceiptThreshold) > 0 {
		n, err := w.repo.CountAttachments(ctx, id)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		if n == 0 {
			violations = append(violations, apperr.Detail{
				Field:   "receipts",
				Message: "at least 1 receipt required above Rp " + current.Amount,
			})
		}
	}

	// Rule 2 — category monthly limit per employee.
	limit, hasLimit, err := w.repo.CategoryMonthlyLimit(ctx, current.CategoryID)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if hasLimit {
		spent, err := w.repo.MonthlyCategorySpend(ctx, current.EmployeeID, current.CategoryID, current.ExpenseDate, id)
		if err != nil {
			return nil, apperr.Internal(err)
		}
		if compareNumeric(addNumeric(spent, current.Amount), limit) > 0 {
			remaining := subtractNumeric(limit, spent)
			if remaining[0] == '-' || remaining == "0.00" {
				remaining = "0.00"
			}
			violations = append(violations, apperr.Detail{
				Field:   "amount",
				Message: "exceeds " + current.CategoryName + " monthly limit (remaining Rp " + remaining + ")",
			})
		}
	}

	// Rule 4 — duplicate suspect: same employee + same amount ±7 days.
	dupDate, err := w.repo.FindDuplicate(ctx, current.EmployeeID, current.Amount, current.ExpenseDate, id)
	if err != nil {
		return nil, apperr.Internal(err)
	}
	if dupDate != "" {
		return nil, apperr.New(409, "DUPLICATE_SUSPECTED", "Possible duplicate of a claim dated "+dupDate)
	}

	if len(violations) > 0 {
		return nil, apperr.WithDetails(apperr.BusinessRule("Policy check failed"), violations...)
	}

	chain := ExcludeSubmitter(ApprovalChain(current.Amount, w.cfg.ApprovalT1, w.cfg.ApprovalT2), current.SubmitterRole)
	if len(chain) == 0 {
		return nil, apperr.BusinessRule("No eligible approver: submitter is the only role in the approval path")
	}

	err = w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var locked Reimbursement
		if err := tx.Clauses(gormclause.Locking{Strength: "UPDATE"}).
			First(&locked, "id = ? AND deleted_at IS NULL", id).Error; err != nil {
			return err
		}
		if !canTransition(locked.Status, "submit") {
			return apperr.Conflict("Claim was already submitted")
		}
		if err := tx.Exec("DELETE FROM approvals WHERE reimbursement_id = ?", id).Error; err != nil {
			return err
		}
		for i, r := range chain {
			if err := tx.Exec(`
				INSERT INTO approvals (reimbursement_id, step_number, approver_role)
				VALUES (?, ?, ?::role_required)`, id, i+1, r).Error; err != nil {
				return err
			}
		}
		return tx.Exec(`
			UPDATE reimbursements SET status = 'SUBMITTED', current_step = 1, submitted_at = now(), decided_at = NULL, updated_at = now()
			WHERE id = ?`, id).Error
	})
	return finishTx(w.repo, ctx, id, err)
}

// Approve acts on the caller's pending step — only when it's their turn and
// they hold the required role. Manager steps require same department.
func (w *WorkflowService) Approve(ctx context.Context, id uuid.UUID, actorRole string, userID, deptID uuid.UUID) (*DetailResponse, error) {
	return w.decide(ctx, id, actorRole, userID, deptID, "approved", "")
}

// Reject requires a note; first rejection ends the whole claim.
func (w *WorkflowService) Reject(ctx context.Context, id uuid.UUID, actorRole string, userID, deptID uuid.UUID, note string) (*DetailResponse, error) {
	if note == "" {
		return nil, apperr.Validation("note is required to reject a claim")
	}
	return w.decide(ctx, id, actorRole, userID, deptID, "rejected", note)
}

func (w *WorkflowService) decide(ctx context.Context, id uuid.UUID, actorRole string, userID, deptID uuid.UUID, action, note string) (*DetailResponse, error) {
	var bizErr error
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var claim Reimbursement
		if err := tx.Clauses(gormclause.Locking{Strength: "UPDATE"}).
			First(&claim, "id = ? AND deleted_at IS NULL", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				bizErr = apperr.NotFound("Claim not found")
			} else {
				bizErr = apperr.Internal(err)
			}
			return bizErr
		}
		if claim.Status != StatusSubmitted {
			bizErr = apperr.Conflict("Claim is not awaiting decisions")
			return bizErr
		}

		var steps []ApprovalStep
		if err := tx.Where("reimbursement_id = ?", id).Order("step_number ASC").Find(&steps).Error; err != nil {
			bizErr = apperr.Internal(err)
			return bizErr
		}
		idx := nextPendingIndex(stepStatuses(steps))
		if idx < 0 {
			bizErr = apperr.Conflict("No pending approval step")
			return bizErr
		}
		step := steps[idx]

		if step.ApproverRole != actorRole {
			bizErr = apperr.Forbidden("Not your turn — waiting for " + step.ApproverRole)
			return bizErr
		}
		if userID == claim.EmployeeID {
			bizErr = apperr.Forbidden("You cannot decide on your own claim")
			return bizErr
		}
		if actorRole == "manager" && deptID != uuid.Nil {
			var empDept uuid.NullUUID
			tx.Raw("SELECT department_id FROM users WHERE id = ?", claim.EmployeeID).Scan(&empDept)
			if empDept.Valid && empDept.UUID != deptID {
				bizErr = apperr.Forbidden("Claim belongs to another department")
				return bizErr
			}
		}

		if action == "approved" {
			if err := tx.Exec(`UPDATE approvals SET status = 'approved', approver_id = ?, acted_at = now() WHERE id = ?`,
				userID, step.ID).Error; err != nil {
				bizErr = apperr.Internal(err)
				return bizErr
			}
			if idx == len(steps)-1 {
				err := tx.Exec(`UPDATE reimbursements SET status = 'APPROVED', decided_at = now(), updated_at = now() WHERE id = ?`, id).Error
				bizErr = wrapInternal(err)
			} else {
				err := tx.Exec(`UPDATE reimbursements SET current_step = ?, updated_at = now() WHERE id = ?`, idx+2, id).Error
				bizErr = wrapInternal(err)
			}
		} else {
			if err := tx.Exec(`UPDATE approvals SET status = 'rejected', approver_id = ?, note = ?, acted_at = now() WHERE id = ?`,
				userID, note, step.ID).Error; err != nil {
				bizErr = apperr.Internal(err)
				return bizErr
			}
			bizErr = wrapInternal(tx.Exec(`UPDATE reimbursements SET status = 'REJECTED', decided_at = now(), updated_at = now() WHERE id = ?`, id).Error)
		}
		return bizErr
	})

	if err != nil && bizErr == nil {
		return nil, apperr.Internal(err)
	}
	if bizErr != nil {
		return nil, bizErr
	}
	return w.repo.GetDetailFresh(ctx, id)
}

// Cancel: owner only, while SUBMITTED and before any step acted (rule 6).
func (w *WorkflowService) Cancel(ctx context.Context, id uuid.UUID, role string, userID, deptID uuid.UUID) (*DetailResponse, error) {
	var bizErr error
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var claim Reimbursement
		if err := tx.Clauses(gormclause.Locking{Strength: "UPDATE"}).
			First(&claim, "id = ? AND deleted_at IS NULL", id).Error; err != nil {
			bizErr = apperr.NotFound("Claim not found")
			return bizErr
		}
		if claim.EmployeeID != userID {
			bizErr = apperr.Forbidden("Only the owner can cancel this claim")
			return bizErr
		}
		if !canTransition(claim.Status, "cancel") {
			bizErr = apperr.Conflict("Only SUBMITTED claims can be cancelled")
			return bizErr
		}
		var acted int64
		tx.Table("approvals").Where("reimbursement_id = ? AND status <> 'pending'", id).Count(&acted)
		if acted > 0 {
			bizErr = apperr.Conflict("Approval already started — claim can no longer be cancelled")
			return bizErr
		}
		bizErr = wrapInternal(tx.Exec(`UPDATE reimbursements SET status = 'CANCELLED', cancelled_at = now(), updated_at = now() WHERE id = ?`, id).Error)
		return bizErr
	})
	if err != nil && bizErr == nil {
		return nil, apperr.Internal(err)
	}
	if bizErr != nil {
		return nil, bizErr
	}
	return w.repo.GetDetailFresh(ctx, id)
}

// Pay: finance only, APPROVED → PAID (terminal).
func (w *WorkflowService) Pay(ctx context.Context, id uuid.UUID, role string) (*DetailResponse, error) {
	if role != "finance" {
		return nil, apperr.Forbidden("Only Finance can mark claims as paid")
	}
	var bizErr error
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var claim Reimbursement
		if err := tx.Clauses(gormclause.Locking{Strength: "UPDATE"}).
			First(&claim, "id = ? AND deleted_at IS NULL", id).Error; err != nil {
			bizErr = apperr.NotFound("Claim not found")
			return bizErr
		}
		if !canTransition(claim.Status, "pay") {
			bizErr = apperr.Conflict("Only APPROVED claims can be paid")
			return bizErr
		}
		bizErr = wrapInternal(tx.Exec(`UPDATE reimbursements SET status = 'PAID', paid_at = now(), updated_at = now() WHERE id = ?`, id).Error)
		return bizErr
	})
	if err != nil && bizErr == nil {
		return nil, apperr.Internal(err)
	}
	if bizErr != nil {
		return nil, bizErr
	}
	return w.repo.GetDetailFresh(ctx, id)
}

// --- helpers ---

// finishTx maps transaction outcomes for Submit.
func finishTx(repo *Repository, ctx context.Context, id uuid.UUID, txErr error) (*DetailResponse, error) {
	if txErr != nil {
		var ae *apperr.Error
		if errors.As(txErr, &ae) {
			return nil, ae
		}
		if errors.Is(txErr, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("Claim not found")
		}
		return nil, apperr.Internal(txErr)
	}
	return repo.GetDetailFresh(ctx, id)
}

func wrapInternal(err error) error {
	if err == nil {
		return nil
	}
	return apperr.Internal(err)
}

func stepStatuses(steps []ApprovalStep) []string {
	out := make([]string, 0, len(steps))
	for _, s := range steps {
		out = append(out, s.Status)
	}
	return out
}
