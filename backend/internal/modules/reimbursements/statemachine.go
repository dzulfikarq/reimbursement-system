package reimbursements

import (
	"strconv"
	"strings"
)

// Pure state-machine + approval-matrix logic — no DB. Kept separate so unit
// tests cover every branch without fixtures.

// Claim statuses (mirrors reimb_status enum).
const (
	StatusDraft     = "DRAFT"
	StatusSubmitted = "SUBMITTED"
	StatusApproved  = "APPROVED"
	StatusRejected  = "REJECTED"
	StatusPaid      = "PAID"
	StatusCancelled = "CANCELLED"
)

// canTransition guards every mutation against the lifecycle in docs/02.
func canTransition(from, action string) bool {
	switch action {
	case "submit":
		return from == StatusDraft || from == StatusRejected
	case "approve", "reject":
		return from == StatusSubmitted
	case "cancel":
		return from == StatusSubmitted
	case "pay":
		return from == StatusApproved
	default:
		return false
	}
}

// ApprovalChain returns the step sequence for an amount under the tiered
// matrix (docs/02). Thresholds arrive as raw numeric strings (env config).
func ApprovalChain(amount, t1, t2 string) []string {
	switch {
	case compareNumeric(amount, t1) <= 0:
		return []string{"manager"}
	case compareNumeric(amount, t2) <= 0:
		return []string{"manager", "finance"}
	default:
		return []string{"manager", "finance", "admin"}
	}
}

// ExcludeSubmitter drops steps the submitter would fill themselves (docs/02
// rule 7 — nobody approves their own claim).
func ExcludeSubmitter(chain []string, submitterRole string) []string {
	out := make([]string, 0, len(chain))
	for _, r := range chain {
		if r != submitterRole {
			out = append(out, r)
		}
	}
	return out
}

func nextPendingIndex(statuses []string) int {
	for i, s := range statuses {
		if s == "pending" {
			return i
		}
	}
	return -1
}

// --- exact decimal string math (numeric(14,2); no floats ever) ---

func compareNumeric(a, b string) int {
	aw, ac := splitDecimal(a)
	bw, bc := splitDecimal(b)
	if aw != bw {
		if aw < bw {
			return -1
		}
		return 1
	}
	if ac != bc {
		if ac < bc {
			return -1
		}
		return 1
	}
	return 0
}

func splitDecimal(s string) (whole int64, cents int64) {
	intPart, fracPart := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	}
	if len(fracPart) > 2 {
		fracPart = fracPart[:2]
	}
	for len(fracPart) < 2 {
		fracPart += "0"
	}
	whole, _ = strconv.ParseInt(intPart, 10, 64)
	cents, _ = strconv.ParseInt(fracPart, 10, 64)
	return whole, cents
}

func formatDecimal(whole, cents int64) string {
	sign := ""
	if whole < 0 || cents < 0 {
		sign = "-"
		if whole < 0 {
			whole = -whole
		}
		if cents < 0 {
			cents = -cents
		}
	}
	return sign + strconv.FormatInt(whole, 10) + "." + pad2(cents)
}

func pad2(v int64) string {
	s := strconv.FormatInt(v, 10)
	for len(s) < 2 {
		s = "0" + s
	}
	return s
}

func addNumeric(a, b string) string {
	aw, ac := splitDecimal(a)
	bw, bc := splitDecimal(b)
	c := ac + bc
	w := aw + bw + c/100
	return formatDecimal(w, c%100)
}

// subtractNumeric assumes a >= b (callers enforce via compareNumeric first).
func subtractNumeric(a, b string) string {
	aw, ac := splitDecimal(a)
	bw, bc := splitDecimal(b)
	w, c := aw-bw, ac-bc
	if c < 0 {
		w--
		c += 100
	}
	return formatDecimal(w, c)
}
