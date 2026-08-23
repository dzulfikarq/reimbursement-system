# Reimbursement Management System

Fullstack take-home test — PT Mumtaz Teknologi Indonesia. Employees submit itemized expense claims with receipts; claims pass a tiered approval workflow (Manager → Finance → Admin by amount) until Finance pays.

- Design documents: [`docs/`](./docs/README.md)
- Agent guidance: [`AGENTS.md`](./AGENTS.md)

## Quick start

```bash
cp .env.example .env
docker compose up --build
```

| Service | URL |
|---|---|
| Frontend | http://localhost:5173 |
| API | http://localhost:8080/api/v1 |
| Health | http://localhost:8080/healthz |
| MinIO console | http://localhost:9001 |
| MailHog | http://localhost:8025 |

## Stack

Go + Gin + GORM + PostgreSQL 16 · React + TypeScript + Vite + Tailwind · Redis (queue/cache) · MinIO · Docker. Details in `docs/06`.

> Status: M0 skeleton — infra, health checks, migration runner. Feature milestones tracked in `docs/08`.
