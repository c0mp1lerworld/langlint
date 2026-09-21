# LangLint

> **AI-Powered Writing & Grammar Analytics** — a language-learning platform based on contextual productive writing, with feedback from an AI acting as a native-speaker teacher.

[Español](README.md) · **English**

[![Contract](https://img.shields.io/badge/contract-3.0.0-informational)](apps/contracts/CHANGELOG.md)
[![Go](https://img.shields.io/badge/Go-1.26.8-00ADD8)](apps/backend/go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-15-black)](apps/frontend/package.json)

This repository is a **demonstration of modern software architecture**: a
polyglot monorepo (Go + Next.js), an OpenAPI contract as the *Single Source of
Truth*, and strict Domain-Driven Design, with automated quality gates.

---

## What is LangLint?

Traditional learning methods (Anki-style flashcards) work for **passive
memorization**, but fail at **active production**: the learner recognizes a word
yet still cannot write a correct text with it. LangLint closes that gap.

The user practices contextual productive writing, and an AI breaks the text down
**fragment by fragment**, explaining the grammatical *why* behind each mistake
and classifying it. An analytics engine tracks recurring errors over time to
eliminate blind spots.

### Usage flow

1. The user defines a **set of grammar targets** (e.g., irregular verbs to
   practice).
2. They write a **source text in Spanish** applying those targets.
3. They write their **experimental English translation** (the draft).
4. The **AI** corrects and explains fragment by fragment, classifying each error
   (preposition, possessive, false friend…).
5. The **analytics dashboard** shows recurring errors and progress.

### Value proposition

| For | Value delivered |
|---|---|
| The student | Move from recognizing a word to writing it correctly in context; understand the *why*, not just see the correction. |
| The product (portfolio) | A demonstration of adapted hexagonal, contract-first, event-driven architecture in a polyglot monorepo. |

---

## Project status

- **Functional MVP complete**: Phases 1–6 (foundations, pure domain, ports and
  adapters, AI engine, frontend, provisioner/privacy) are green.
- **Phase 7 — Hardening**: CI/CD (`7.1`) and Security (`7.2`) complete;
  documentation (`7.3`) being closed out. Code, tests and local gates are green.
- **Post-MVP enhancement — Phase 9**: structured deep feedback and active
  practice (AI quiz) complete (`9.1`–`9.6`).
- **MVP non-goals**: real multi-user, multi-language and streaming
  (see [`PRODUCT_DOMAIN.md §3.2`](docs/PRODUCT_DOMAIN.md)).

---

## Tech stack

| Layer | Technology |
|---|---|
| Languages | Go 1.26 · TypeScript 5.9 (strict) |
| Frontend | Next.js 15 (App Router) · React 19 · Tailwind CSS 4 |
| State | TanStack Query (server state) · Zustand (client state) |
| Forms | React Hook Form + Zod |
| Backend | Go: adapted hexagonal architecture · Uber Fx · chi |
| Contract | OpenAPI 3.1 (`apps/contracts/openapi/api.yaml`) |
| Generation | `oapi-codegen` (Go) · `openapi-typescript` (TS) |
| Persistence | PostgreSQL 16 · `pgx` · goose · Unit of Work · Outbox |
| AI | OpenAI with Structured Outputs, behind the `LLMExtractor` port |
| Tests | `go test -race` · testcontainers · Vitest · RTL · Playwright |
| Monorepo | pnpm workspaces + Turborepo (self-hosted remote cache) |
| Security | `gosec` · `govulncheck` · static audits |

---

## Architecture

### Contract-first monorepo (A12)

The `api.yaml` contract is the **single source of truth** for the wire. Every
change starts in the contract, goes through `pnpm generate` (which regenerates
Go **and** TS), and only then reaches the code. Generated artifacts (`gen_*.go`,
`gen.ts`) are never edited by hand. CI fails if the generated code drifts from
the contract.

### Dependency direction (A2)

```
cmd  →  adapters  →  services  →  ports  →  domain
                                            (pure core, stdlib only, A1)
```

### Isolated bounded contexts (A3)

| Context | Responsibility | Aggregate root |
|---|---|---|
| `identity` | Portable identity and data audit (A9). | `User` |
| `practice` | The exercise: source text, draft and target rules. | `Practice` |
| `analysis` | The fragment-level analysis produced by the AI. | `Analysis` |
| `analytics` | Materialization of error patterns and progress. | `ErrorMetric` |

They communicate through ports and **domain events** via the outbox, never
through cross-imports. Details live in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

### Transactions and events

- **Unit of Work**: services never touch SQL; the adapter opens
  `BEGIN/COMMIT/ROLLBACK`.
- **Outbox**: the state change and the event are persisted in the same
  transaction; a relay publishes them after commit.

### Privacy (A8/A9)

- **Anonymization** before the LLM and **pseudonymization** (HMAC-SHA256) in
  analytics; raw data separation and scheduled purging.
- `PIIHandler` as a runtime logging guard plus static audits in CI.

---

## Repository layout

```text
langlint/
├── apps/
│   ├── contracts/          # OpenAPI SSOT + generation pipeline
│   ├── backend/            # Go: cmd → adapters → services → ports → domain
│   └── frontend/           # Next.js App Router (3-column diff + analytics)
├── docs/                   # ARCHITECTURE, RUNBOOK, PRODUCTION_ENV, SECRET_ROTATION, …
├── ops/                    # docker-compose (Postgres + turbo-cache) and audit scripts
├── .github/workflows/      # ci · contracts · deploy-staging · security
├── MANIFEST_MONOREPO.md    # axioms A1–A12 and backend/monorepo patterns
├── MANIFEST_FRONTEND.md    # frontend axioms F1–F12
├── turbo.json · pnpm-workspace.yaml · .tool-versions
└── AGENTS.md               # onboarding and working memory for AI agents
```

---

## Engineering rigor

| Metric | Value | How it is verified |
|---|---|---|
| Domain coverage | **100 %** | `go test -cover ./internal/domain/...` |
| Services coverage | **91–98 %** | `go test -cover ./internal/api/services/...` |
| Backend tests (Tier 1+2) | green | `pnpm test` (`go test -race`) |
| Integration (Tier 3) | green | `pnpm test-integration --filter=backend` (testcontainers) |
| Frontend tests (Vitest) | **95** | `pnpm --filter frontend test` |
| e2e (Playwright) | 2 | `pnpm test:e2e` |
| Static security analysis | **0 findings** | `gosec` / `govulncheck` (CI `security.yml`) |
| Contract | **3.0.0** | `contracts.yml`: drift + `info.version` |

---

## Getting started

### Prerequisites

- **Go** 1.26.8 · **Node** 22 · **pnpm** 9.15.4 · Docker (for Postgres).
- Exact versions live in `.tool-versions` (works with `mise`/`asdf`) and `.nvmrc`.

### Install and run

```bash
pnpm install --frozen-lockfile

# Local infrastructure (Postgres :5433)
docker compose -f ops/docker/docker-compose.yml up -d

# Backend
cd apps/backend && cp .env.example .env   # fill in real secrets (A8)
go run ./cmd/migrate && go run ./cmd/api

# Frontend (another terminal)
cp apps/frontend/.env.example apps/frontend/.env.local
pnpm --filter frontend dev
```

See [`docs/RUNBOOK.md`](docs/RUNBOOK.md) for troubleshooting and
[`docs/PRODUCTION_ENV.md`](docs/PRODUCTION_ENV.md) for every variable.

### Canonical commands

| Command | Purpose |
|---|---|
| `pnpm build` | Build the whole monorepo (Turborepo) |
| `pnpm lint` | Lint the monorepo (redocly + go vet + eslint) |
| `pnpm test` | Tier 1+2 tests (Go + Vitest) |
| `pnpm test-integration --filter=backend` | Tier 3 tests (testcontainers) |
| `pnpm test:e2e` | Frontend e2e (Playwright) |
| `pnpm typecheck --filter=frontend` | `tsc --noEmit` |
| `pnpm generate` | Regenerate Go + TS types from the contract |
| `go run ./cmd/provisioner <job>` | Batch jobs (`refresh-aggregates`, …) |

---

## Roadmap

| # | Step | Status |
|---|---|---|
| 1 | Foundations | ✅ |
| 2 | Pure domain (100 % coverage) | ✅ |
| 3 | Ports and adapters (UoW, outbox, Postgres) | ✅ |
| 4 | AI engine (Structured Outputs) | ✅ |
| 5 | Frontend (3-column diff + analytics) | ✅ |
| 6 | Provisioner and privacy (A9) | ✅ |
| 7 | Hardening (CI/CD, security, documentation) | closing |

### Future (post-MVP): Adaptive Tutor

The natural evolution is to turn LangLint into an **adaptive, personalized
English tutor** with *spaced repetition* and learning intelligence driven by the
user's **real errors**. It is added as a fifth bounded context (`tutor/`) that
consumes `analytics/` aggregates via the event bus, **without breaking** any of
the four existing contexts. See [`PRODUCT_DOMAIN.md §12.1`](docs/PRODUCT_DOMAIN.md).

---

## Documentation

| Document | Content |
|---|---|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Global architectural contract |
| [`docs/PRODUCT_DOMAIN.md`](docs/PRODUCT_DOMAIN.md) | Vision, domain model, contract, roadmap |
| [`docs/RUNBOOK.md`](docs/RUNBOOK.md) | Operations and incidents |
| [`docs/PRODUCTION_ENV.md`](docs/PRODUCTION_ENV.md) | Environment variables |
| [`docs/SECRET_ROTATION.md`](docs/SECRET_ROTATION.md) | Secret rotation |
| [`docs/SECURITY_DEBT.md`](docs/SECURITY_DEBT.md) | Accepted security debt |
| [`docs/API_CONTRACT.md`](docs/API_CONTRACT.md) | Contract↔domain gaps |
| [`docs/bugs/`](docs/bugs/) | Known bugs |
| [`MANIFEST_MONOREPO.md`](MANIFEST_MONOREPO.md) | Axioms A1–A12 and backend patterns |
| [`MANIFEST_FRONTEND.md`](MANIFEST_FRONTEND.md) | Frontend axioms F1–F12 |
| [`AGENTS.md`](AGENTS.md) | Onboarding for AI agents |

---

## License

Distributed under the **Apache-2.0** license. See [`LICENSE`](LICENSE).
