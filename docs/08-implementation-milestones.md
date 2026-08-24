# 08 — Implementation Milestones

Each milestone ends runnable and committed. Order = risk first (auth/workflow are the graded core), polish later.

| # | Milestone | Contents | Done when |
|---|---|---|---|
| M0 | Skeleton & Infra | compose (pg/redis/minio/mailhog/api/web), Makefiles, config, healthz, migration runner, CI stub | `docker compose up` -> healthz green, empty SPA served |
| M1 | Auth Core | users migration, login/logout/me/refresh + rotation, CSRF middleware, RBAC middleware, seed 4 role users | curl flow: login -> protected endpoint with cookie -> refresh rotates -> logout kills session; POST without CSRF gets 403 |
| M2 | Master Data | categories/users CRUD (admin), validations, search/filter/sort/pagination on all three lists | Admin manages masters via API with proper status codes |
| M3 | Claim CRUD | reimbursements + items tables, draft create/edit/delete, attachments to MinIO (MIME/size validation), listing SFP, scoped access | Employee creates itemized draft, uploads receipt, sees only own list |
| M4 | Workflow Engine | submit + policy checks (limits, receipt rule, duplicates), approval matrix generation, approve/reject/cancel/pay state machine with row locking, timeline endpoint | Happy path plus each rejection path covered by API tests; concurrent double-approve test passes |
| M5 | Dashboards & Audit | summary/trend/breakdown endpoints, Redis caching, audit log writes | Dashboard reflects workflow actions; second load hits cache |
| M6 | Worker & Notifications | asynq worker, email jobs on submit/approve/reject/pay, async CSV export job | Emails land in Mailhog after actions; export produces downloadable CSV |
| M7 | Frontend Complete | all pages per doc 05, component inventory, form UX, loading/empty/error states, responsive, toasts, confirm dialogs, full interceptor mapping | Whole flow clickable in browser for every role at 3 breakpoints |
| M8 | Hardening & Docs | unit+integration suites green, FE component/interceptor tests, Swagger complete, README full section set, security headers, rate limits verified | CI green; reviewer self-serves from README alone |

Stretch (post-M8, pick if time): Playwright E2E critical flow, Prometheus metrics, transactional outbox for enqueue, keyset pagination benchmark note.

## Effort Shape

M0-M2 fast foundation. M4 is the heart: allocate most time and testing there. M7 largest by file count but mechanical.
