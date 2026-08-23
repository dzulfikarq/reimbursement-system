package integration

import (
	"fmt"
	"net/http"
	"testing"
)

// claimPayload builds a create-claim body.
func claimPayload(catID, title string, qty int, price string) map[string]any {
	return map[string]any{
		"category_id":  catID,
		"title":        title,
		"expense_date": "2026-08-23",
		"items": []map[string]any{{
			"description": "Test item",
			"quantity":    qty,
			"unit_price":  price,
		}},
	}
}

// Full happy path: employee creates → submits → manager approves (tier 1) →
// finance pays. Verifies state machine + snapshot chain end-to-end.
func TestClaimHappyPathToPaid(t *testing.T) {
	setup(t)
	catID := seedCategory(t, "") // no limit
	empID, empEmail := seedUser(t, "Happy Emp", "employee")
	mgrID, mgrEmail := seedUser(t, "Happy Mgr", "manager")
	_, finEmail := seedUser(t, "Happy Fin", "finance")
	_, _ = empID, mgrID

	// Manager & finance need to be in the same department for scope? Listing
	// is scoped, but approve/pay only check role+turn — different depts OK.

	emp := login(t, empEmail, testPassword)
	code, out := emp.do(http.MethodPost, "/api/v1/reimbursements", claimPayload(catID, "Happy path", 2, "150000"), true)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("create: got %d: %v", code, out)
	}
	data := out["data"].(map[string]any)
	id := data["id"].(string)
	if data["status"] != "DRAFT" && data["status"] != "draft" {
		t.Fatalf("new claim should be draft, got %v", data["status"])
	}

	// Submit → tier-1 approval steps snapshotted.
	code, _ = emp.do(http.MethodPost, fmt.Sprintf("/api/v1/reimbursements/%s/submit", id), nil, true)
	if code != http.StatusOK {
		t.Fatalf("submit: got %d", code)
	}

	// Employee cannot approve own claim.
	if code, _ := emp.do(http.MethodPost, fmt.Sprintf("/api/v1/reimbursements/%s/approve", id), nil, true); code != http.StatusForbidden {
		t.Fatalf("self approve should be 403, got %d", code)
	}

	// Manager approves step 1 → APPROVED (300k ≤ 5M ⇒ manager only).
	mgr := login(t, mgrEmail, testPassword)
	code, _ = mgr.do(http.MethodPost, fmt.Sprintf("/api/v1/reimbursements/%s/approve", id), nil, true)
	if code != http.StatusOK {
		t.Fatalf("manager approve: got %d", code)
	}

	// Finance pays → PAID.
	fin := login(t, finEmail, testPassword)
	code, _ = fin.do(http.MethodPost, fmt.Sprintf("/api/v1/reimbursements/%s/pay", id), nil, true)
	if code != http.StatusOK {
		t.Fatalf("pay: got %d", code)
	}

	// Pay twice → 409.
	if code, _ := fin.do(http.MethodPost, fmt.Sprintf("/api/v1/reimbursements/%s/pay", id), nil, true); code != http.StatusConflict {
		t.Fatalf("double pay should be 409, got %d", code)
	}
}

// Policy rejections at submit: missing receipt over threshold and monthly
// category limit — violations must arrive batched in one 422.
func TestSubmitPolicyViolations(t *testing.T) {
	setup(t)
	catID := seedCategory(t, "400000") // 400k monthly limit
	_, empEmail := seedUser(t, "Policy Emp", "employee")
	emp := login(t, empEmail, testPassword)

	// 500001 > receipt threshold AND > limit → both violations listed once.
	code, out := emp.do(http.MethodPost, "/api/v1/reimbursements", claimPayload(catID, "Too big", 1, "500001"), true)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("create: %d", code)
	}
	id := out["data"].(map[string]any)["id"].(string)

	code, out = emp.do(http.MethodPost, fmt.Sprintf("/api/v1/reimbursements/%s/submit", id), nil, true)
	if code != http.StatusUnprocessableEntity {
		t.Fatalf("submit with violations: want 422, got %d: %v", code, out)
	}
	errBody := out["error"].(map[string]any)
	if errBody["code"] != "BUSINESS_RULE_VIOLATED" {
		t.Fatalf("want BUSINESS_RULE_VIOLATED, got %v", errBody["code"])
	}
	details := errBody["details"].([]any)
	if len(details) < 2 {
		t.Fatalf("violations should be batched (>=2), got %v", details)
	}
}

// Duplicate guard: same employee + amount within ±7 days of an active claim.
func TestDuplicateDetection(t *testing.T) {
	setup(t)
	catID := seedCategory(t, "")
	_, empEmail := seedUser(t, "Dup Emp", "employee")
	emp := login(t, empEmail, testPassword)

	code, out := emp.do(http.MethodPost, "/api/v1/reimbursements", claimPayload(catID, "Original trip", 3, "100000"), true)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("first create failed: %d", code)
	}
	first := out["data"].(map[string]any)["id"].(string)
	if code, _ := emp.do(http.MethodPost, fmt.Sprintf("/api/v1/reimbursements/%s/submit", first), nil, true); code != http.StatusOK {
		t.Fatalf("first submit failed")
	}

	// Same total (300000), same date → duplicate suspected.
	code, out = emp.do(http.MethodPost, "/api/v1/reimbursements", claimPayload(catID, "Copy of trip", 30, "10000"), true)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("second create failed: %d", code)
	}
	second := out["data"].(map[string]any)["id"].(string)
	code, _ = emp.do(http.MethodPost, fmt.Sprintf("/api/v1/reimbursements/%s/submit", second), nil, true)
	if code != http.StatusConflict {
		t.Fatalf("duplicate submit: want 409, got %d", code)
	}
}

// Scope: another employee's claim is invisible in detail.
func TestDetailScopeIsolation(t *testing.T) {
	setup(t)
	catID := seedCategory(t, "")
	ownerID, ownerEmail := seedUser(t, "Owner", "employee")
	strangerID, strangerEmail := seedUser(t, "Stranger", "employee")

	owner := login(t, ownerEmail, testPassword)
	code, out := owner.do(http.MethodPost, "/api/v1/reimbursements", claimPayload(catID, "Private", 1, "1000"), true)
	if code != http.StatusCreated && code != http.StatusOK {
		t.Fatalf("create: %d", code)
	}
	id := out["data"].(map[string]any)["id"].(string)
	stranger := login(t, strangerEmail, testPassword)
	if code, _ := stranger.do(http.MethodGet, "/api/v1/reimbursements/"+id, nil, false); code != http.StatusNotFound {
		t.Fatalf("stranger detail: want 404, got %d", code)
	}
	if code, _ := stranger.do(http.MethodPatch, "/api/v1/reimbursements/"+id, claimPayload(catID, "Hacked", 1, "1"), true); code >= 300 {
		t.Logf("stranger patch status=%d (403 or 404 acceptable)", code)
	}
	_ = ownerID
	_ = strangerID
}
