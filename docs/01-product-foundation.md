# 01 — Product Foundation

## Problem

Reimbursement claiming in most companies runs through paper receipts and spreadsheets:

- Employees lose receipts and wait weeks without knowing claim status.
- Managers approve via chat with no policy check (duplicate claims, over-limit categories slip through).
- Finance reconciles manually, with no audit trail of who approved what and when.

## Solution

A web-based **Reimbursement Management System** with a single workflow:

```
Employee submits → policy validated → routed to approver(s) by amount tier
→ approved → finance pays → everything logged
```

Policy enforcement (approval matrix, category limits, receipt requirement, duplicate detection) happens in the backend, not on the honor system.

## Target Users

| User | Pain today | What they get |
|------|-----------|---------------|
| Employee | No visibility, lost receipts | Self-service claim + status timeline |
| Manager | Approves blind via chat | Approval inbox with history & attachments |
| Finance | Manual reconciliation | Payment queue + monthly reports |
| Admin / HR | No control over spend rules | Configurable categories, limits, budgets |

## Value Proposition

1. **Enforced policy** — rules live in the backend; UI cannot bypass them.
2. **Traceability** — every state change recorded (approvals table + audit log).
3. **Faster cycle** — async email notifications remove "chasing" from the process.

## Non-Goals (v1)

- Multi-currency / multi-language (IDR only).
- Payroll integration (payment is a manual "mark as paid" by Finance).
- Mobile app (responsive web covers it).

## Success Criteria for This Test

- A reviewer can run `docker compose up` and exercise the full flow: login as each role → create → submit → approve → pay → see dashboard update.
- Every technical decision is documented and defensible in interview (see doc 06 rationale tables).
