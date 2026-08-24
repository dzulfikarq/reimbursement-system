# 02 — Business Requirements

## Roles & Permissions (RBAC)

Authorization is enforced **in the backend** (middleware + service checks). Frontend only hides/shows UI.

| Capability | Employee | Manager | Finance | Admin |
|---|:-:|:-:|:-:|:-:|
| Create / edit / delete own draft | ✅ | ✅ | ✅ | — |
| Submit / cancel own claim | ✅ | ✅ | ✅ | — |
| View own claims | ✅ | ✅ | ✅ | ✅ |
| View team claims | — | ✅ | ✅ | ✅ |
| View all claims | — | — | ✅ | ✅ |
| Approve / reject (per approval matrix) | — | ✅ | ✅ | ✅ |
| Mark as paid | — | — | ✅ | — |
| Manage users | — | — | — | ✅ |
| Manage categories & limits | — | — | — | ✅ |
| View dashboards (scoped) | own | all | all | all |

Managers see and approve claims from the whole organization (no department structure). Admin does **not** approve; admin manages the system.

## Claim Lifecycle (State Machine)

```
            submit                approve chain complete        pay
 DRAFT ────────────► SUBMITTED ────────────────► APPROVED ───────► PAID  (terminal)
   ▲                     │    │                      ▲
   │  edit & resubmit    │    │ reject at any step   │
   └────────────── REJECTED ◄─┘                      │
   ▲                                                 │
   └────────────── CANCELLED ◄── owner, while SUBMITTED & no step acted
                    (terminal)
```

Transitions are validated server-side. Illegal transition → `409 CONFLICT` / `422`.

## Approval Matrix (amount-tiered)

Steps are generated and snapshotted into the `approvals` table when a claim is submitted.

| Total amount | Approval path |
|---|---|
| ≤ Rp 500.000 | Manager → done |
| Rp 500.001 – 5.000.000 | Manager → Finance |
| > Rp 5.000.000 | Manager → Finance → Admin |

Rationale: low-value claims shouldn't burn senior time; high-value claims get financial + executive oversight. Thresholds configurable per environment.

## Business Rules (backend-enforced)

1. **Amount integrity** — header amount must equal Σ(item qty × unit_price); recomputed server-side, never trusted from client.
2. **Category monthly limit** — each category may define `monthly_limit_per_employee`. Sum of that employee's non-rejected claims in the category for that calendar month + this claim must not exceed it (`422 BUSINESS_RULE_VIOLATED`).
3. **Receipt required** — claims > Rp 500.000 require ≥ 1 attachment before submit.
4. **Duplicate detection** — same employee, same total amount, expense date within ±7 days of an existing non-cancelled/non-rejected claim → block with `409 DUPLICATE_SUSPECTED` (message shows the conflicting claim).
5. **Edit window** — editable only in `DRAFT`/`REJECTED` by owner. Deletable only in `DRAFT` by owner.
6. **Cancel window** — owner may cancel while `SUBMITTED` only if no approval step has been acted on.
7. **Sequential approvals** — approver can act only on their own pending step and only when all prior steps are approved. Approving own submission impossible: submitter is excluded from generated steps (their manager/finance/admin handle it).

## Functional Requirements

- FR-1 Auth: login, logout, session state, current-user endpoint, refresh rotation, CSRF token issuance.
- FR-2 Claims CRUD (draft), itemized lines, receipt attachments (MinIO).
- FR-3 Submit with full policy validation; rejection lists every violated rule at once.
- FR-4 Approval inbox per role with filters; approve/reject with mandatory note on reject.
- FR-5 Payment queue for Finance; mark-as-paid records payer + timestamp.
- FR-6 Listing pages: search (title/description), filter (status, category, date range), sort, pagination — all query params validated.
- FR-7 Dashboards: totals by status, monthly trend, category breakdown (role-scoped, Redis-cached).
- FR-8 Notifications: async email on submit/approve/reject/paid via queue worker.
- FR-9 Audit trail of privileged actions (login, approve, pay, admin changes).
- FR-10 Export: finance can request a CSV report; generated asynchronously by worker, download link via toast.

## Validation Matrix (requirement #7 coverage)

| Input class | Rules |
|---|---|
| Auth input | email format, password length 8–72, required |
| Body | required fields, string lengths, enum status/category codes, numeric ranges (qty ≥ 1, unit_price > 0 ≤ 999.999.999), date format YYYY-MM-DD not in future beyond 30 days |
| Query | page ≥ 1, limit 1–100, sort whitelist per resource, order enum, UUID format for ids, date range from ≤ to |
| Path | UUID validation |
| File upload | MIME allowlist (image/png, jpeg, webp, application/pdf), max 5 MB, extension sniffed from magic bytes not filename |

Backend is source of truth; frontend mirrors rules for fast feedback only.
