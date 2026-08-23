# 07 — Repository Structure

Monorepo. One clone, one compose file, reviewer runs everything from root.

```
reimbursement-system/
├── README.md
├── .env.example
├── .github/workflows/ci.yml        # lint + test + build (bonus)
├── docker-compose.yml              # dev: full stack, one command
├── docs/                           # design documents (this folder)
│
├── backend/
│   ├── cmd/
│   │   ├── api/main.go             # HTTP server entrypoint
│   │   └── worker/main.go          # queue worker entrypoint
│   ├── internal/
│   │   ├── config/config.go        # env parsing, single source of config
│   │   ├── modules/
│   │   │   ├── auth/               # handler.go service.go repository.go dto.go routes.go
│   │   │   ├── user/
│   │   │   ├── department/
│   │   │   ├── category/
│   │   │   ├── reimbursement/      # incl. state machine + policy engine
│   │   │   ├── approval/           # matrix + step generation
│   │   │   ├── dashboard/
│   │   │   ├── report/
│   │   │   └── notification/
│   │   ├── middleware/             # requestid, logger, recover, cors, headers,
│   │   │                           # ratelimit, csrf, authn, authz
│   │   └── pkg/                    # apperr, response, validator, jwt,
│   │                               # password, miniox, redisx, mailer
│   ├── migrations/                 # NNNN_name.up.sql / .down.sql (golang-migrate)
│   ├── seeds/seed.go               # demo users per role + master data
│   ├── Makefile                    # run, migrate-up/down, test, seed, swagger
│   ├── Dockerfile                  # multi-stage, distroless-ish final
│   ├── go.mod / go.sum
│   └── *_test.go                   # unit + integration tests alongside code
│
├── frontend/
│   ├── src/
│   │   ├── main.tsx, App.tsx, router.tsx
│   │   ├── lib/api.ts              # axios instance + interceptors (doc 05)
│   │   ├── stores/auth.ts          # zustand session store
│   │   ├── features/
│   │   │   ├── auth/               # login page, guards
│   │   │   ├── reimbursements/     # list/detail/form pages + queries
│   │   │   ├── approvals/
│   │   │   ├── payments/
│   │   │   ├── dashboard/
│   │   │   └── admin/              # users/departments/categories pages
│   │   ├── components/ui/          # reusable inventory (doc 05)
│   │   ├── components/layout/      # Sidebar, Topbar, shell
│   │   ├── hooks/                  # useSession, useDebounce, etc.
│   │   └── types/                  # shared API types
│   ├── index.html, vite.config.ts, tailwind.config.ts, tsconfig.json
│   ├── Dockerfile                  # build → nginx serve with /api proxy
│   └── package.json
│
└── deploy/                         # prod overrides if separated
```

## Conventions

- Backend module files named consistently (`handler/service/repository/dto/routes`); a reviewer knows where anything lives in one guess.
- No shared "utils" dumping ground — helpers live in the module that uses them or `pkg/` when genuinely cross-module.
- Frontend feature folders own their API calls and types; `components/ui` stays domain-agnostic.
- Commits follow Conventional Commits; branches per milestone (doc 08) so git history mirrors the build plan.
