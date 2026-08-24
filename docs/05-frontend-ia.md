# 05 — Frontend IA

React 19 + TypeScript + Vite. Tailwind CSS, Axios, React Router (data mode off — plain routes), TanStack Query for server state, Zustand for session state, react-hook-form + zod for forms.

## Route Map

| Route | Page | Access |
|---|---|---|
| `/login` | Login | public (redirect to `/` if authed) |
| `/` | Dashboard | all (content role-scoped) |
| `/reimbursements` | My claims / All claims | scoped by role |
| `/reimbursements/new` | Create claim wizard | employee+ |
| `/reimbursements/:id` | Claim detail + approval timeline | owner/scope |
| `/reimbursements/:id/edit` | Edit draft/rejected | owner |
| `/approvals` | Approval inbox | manager/finance/admin |
| `/payments` | Payment queue (APPROVED claims) | finance |
| `/admin/users` | User management | admin |
| `/admin/categories` | Categories & limits | admin |
| `*` | 404 · `403` page for denied routes | |

Route guarding: `ProtectedRoute` checks session store; `RoleRoute` checks allowed roles → 403 page. Backend remains source of truth; frontend guards are UX only.

## Navigation

- Desktop: left sidebar with sections; items filtered by role.
- Mobile/tablet (< lg): collapsible top bar drawer navigation.
- Badge counts on "Approvals" and "Payments" from dashboard summary (polled lightly).

Sidebar per role:

```
employee :  Dashboard · My Claims
manager  :  Dashboard · My Claims · Approvals
finance  :  Dashboard · All Claims · Approvals · Payments
admin    :  Dashboard -> All Claims -> Users -> Categories
```

## Key Screens & States

Every listing/detail screen implements the required state set:

1. **Loading** — skeleton rows/cards (no spinners where skeleton fits).
2. **Empty** — illustration-lite EmptyState with primary action ("Create your first claim").
3. **Error** — inline error panel with retry button; toast for mutation failures.
4. **Disabled** — buttons disabled during mutations (`isSubmitting`), inputs disabled while loading initial data.

Forms (login, claim form, admin forms): inline validation on blur, error text under fields, submit button shows spinner + disabled, success via toast, unsaved-changes guard on edit pages.

Destructive actions (delete draft, reject, cancel) → ConfirmDialog typing the consequence.

Claim detail = header info + item table + attachment viewer + vertical **approval timeline** (each step: role, approver name, status, note, timestamp).

## Component Inventory (reusable, requirement #11)

| Component | Notes |
|---|---|
| Button | variants: primary/secondary/danger/ghost; sizes; loading state built in |
| Input, Textarea, Select, DatePicker | native `<input type="date">`; wrapped as FormField |
| FormField | label + control + inline error + hint |
| Form | RHF context + zod resolver |
| Table | generic columns config, sticky header, sortable headers |
| Pagination | page controls + limit selector + total count |
| Modal, ConfirmDialog, Drawer | focus trap, Esc close |
| DropdownMenu | row actions |
| Badge | status colors mapped from claim status |
| Card, StatCard | dashboard tiles |
| Toast | global toaster, promise-aware |
| Spinner, Skeleton | primitives |
| EmptyState | icon/title/action props |
| FileUpload | drag-drop, client MIME+size pre-check, progress |
| Timeline | approval history |
| RoleGate | conditional render by permission |

No component duplicated with same function; pages compose these only.

## Axios Layer (requirement #9)

Single instance, `withCredentials: true`, baseURL `/api/v1`.

Request interceptor: attach `X-CSRF-Token` from csrf cookie on mutating verbs.

Response interceptor mapping:

| Status | Behavior |
|---|---|
| 401 | If not already refreshing and refresh cookie exists → single-flight refresh, queue concurrent requests, replay once. Refresh fails/expired → logout + redirect `/login?expired=1`. No infinite retry (single replay flag). |
| 403 | Toast "You don't have permission" + redirect 403 page on route-level denials |
| 422 | Map `error.details[]` onto form fields when inside a form; else toast |
| 429 | Toast with retry-after countdown |
| 500 | Generic "Something went wrong" toast, log request_id |

Refresh race-safety: module-level `refreshPromise` — concurrent 401s await the same promise instead of firing duplicate refresh calls.
