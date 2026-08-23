package reimbursements

import "testing"

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, action string
		want         bool
	}{
		{"DRAFT", "submit", true},
		{"REJECTED", "submit", true},
		{"SUBMITTED", "submit", false},
		{"PAID", "submit", false},
		{"CANCELLED", "submit", false},
		{"SUBMITTED", "approve", true},
		{"SUBMITTED", "reject", true},
		{"APPROVED", "approve", false},
		{"DRAFT", "approve", false},
		{"APPROVED", "pay", true},
		{"SUBMITTED", "pay", false},
		{"PAID", "pay", false},
		{"SUBMITTED", "cancel", true},
		{"APPROVED", "cancel", false},
	}
	for _, c := range cases {
		if got := canTransition(c.from, c.action); got != c.want {
			t.Errorf("canTransition(%q,%q)=%v want %v", c.from, c.action, got, c.want)
		}
	}
}

func TestApprovalChainMatrix(t *testing.T) {
	const t1, t2 = "500000", "5000000"
	cases := []struct {
		amount string
		want   int
		last   string
	}{
		{"0", 1, "manager"},
		{"499999.99", 1, "manager"},
		{"500000", 1, "manager"},      // boundary: ≤ tier1 manager only
		{"500000.01", 2, "finance"},
		{"5000000", 2, "finance"},     // boundary: ≤ tier2 +admin not needed
		{"5000000.01", 3, "admin"},
		{"99999999.99", 3, "admin"},
	}
	for _, c := range cases {
		got := ApprovalChain(c.amount, t1, t2)
		if len(got) != c.want || got[len(got)-1] != c.last {
			t.Errorf("ApprovalChain(%q)=%v want len %d ending %q", c.amount, got, c.want, c.last)
		}
	}
}

func TestExcludeSubmitter(t *testing.T) {
	chain := []string{"manager", "finance", "admin"}
	if got := ExcludeSubmitter(chain, "employee"); len(got) != 3 {
		t.Errorf("employee claim should keep all steps, got %v", got)
	}
	if got := ExcludeSubmitter(chain, "manager"); len(got) != 2 || got[0] != "finance" {
		t.Errorf("manager submitter drops own step, got %v", got)
	}
	if got := ExcludeSubmitter([]string{"manager"}, "manager"); len(got) != 0 {
		t.Errorf("empty chain when submitter is only approver, got %v", got)
	}
}

func TestNextPendingIndex(t *testing.T) {
	if i := nextPendingIndex([]string{"approved", "pending"}); i != 1 {
		t.Errorf("want 1, got %d", i)
	}
	if i := nextPendingIndex([]string{"approved", "rejected"}); i != -1 {
		t.Errorf("want -1, got %d", i)
	}
}

func TestDecimalMath(t *testing.T) {
	if got := compareNumeric("1000000", "999999.99"); got <= 0 {
		t.Errorf("compare 1000000 vs 999999.99 = %d", got)
	}
	if got := compareNumeric("500.00", "500"); got != 0 {
		t.Errorf("trailing zeros equal, got %d", got)
	}
	if got := addNumeric("999999999999.98", "0.03"); got != "1000000000000.01" {
		t.Errorf("add carry across cents+whole, got %s", got)
	}
	if got := addNumeric("10.50", "20.60"); got != "31.10" {
		t.Errorf("got %s", got)
	}
	if got := subtractNumeric("100.00", "0.01"); got != "99.99" {
		t.Errorf("subtract borrow, got %s", got)
	}
	if got := subtractNumeric("5000000.00", "1500000.25"); got != "3499999.75" {
		t.Errorf("got %s", got)
	}
	if compareNumeric("5", "5.00") != 0 || compareNumeric("4.99", "5") >= 0 {
		t.Errorf("boundary compares wrong")
	}
}

func TestSplitTruncatesLongFraction(t *testing.T) {
	w, c := splitDecimal("12.999")
	if w != 12 || c != 99 {
		t.Errorf("splitDecimal truncation wrong: %d.%02d", w, c)
	}
}
