# LangLint

> **AI-Powered Writing & Grammar Analytics** — a language-learning platform based on contextual productive writing, with feedback from an AI acting as a native-speaker teacher.

[Español](README.md) · **English**

---

## Project status

- **Current phase**: Phase 1 — Foundations (base monorepo green).
- **MVP**: single-user, no real login. The MVP language pair is **Spanish → English**.
- **Code**: `apps/` is still under construction. Phase 0 (documentation and foundations) is complete.

This repository is a **demonstration of modern software architecture**: a polyglot monorepo (Go + Next.js), an OpenAPI contract as the *Single Source of Truth*, and strict Domain-Driven Design.

---

## What is LangLint?

Traditional learning methods (Anki-style flashcards) work for **passive memorization**, but fail at **active production**: the learner recognizes a word yet still cannot write a correct text with it. LangLint closes that gap.

The user practices contextual productive writing, and an AI breaks the text down **fragment by fragment**, explaining the grammatical *why* behind each mistake and classifying it. An analytics engine tracks recurring errors over time to eliminate blind spots.

### Usage flow

1. The user defines a **set of grammar targets** (e.g., irregular verbs to practice).
2. They write a **source text in Spanish** applying those targets.
3. They write their **experimental English translation** (the draft).
4. The **AI** corrects and explains fragment by fragment, classifying each error (preposition, possessive, false friend…).
5. The **analytics dashboard** shows recurring errors and progress over time.

### Value proposition

| For | Value delivered |
|---|---|
| The student | Move from recognizing a word to writing it correctly in context; understand the *why*, not just see the correction. |
| The product (portfolio) | A demonstration of hexagonal, contract-first, event-driven architecture in a polyglot monorepo. |

---

## Tech stack

| Layer | Technology |
|---|---|
| Languages | Go 1.26 · TypeScript (strict) |
| Frontend | Next.js 14+ (App Router) · React Server Components · Tailwind CSS |
| State | TanStack Query (server state) · Zustand (client state) |
| Forms | React Hook Form + Zod |
| Backend | Go (adapted hexagonal architecture) |
| Contract | OpenAPI 3.1 (`apps/contracts/openapi/api.yaml`) |
| Generation | `oapi-codegen` (Go) · `openapi-typescript` (TS) |
| Persistence | PostgreSQL · Unit of Work · Outbox pattern |
| AI | LLM provider with Structured Outputs (OpenAI, behind the `LLMExtractor` port) |
| Monorepo | pnpm workspaces + Turborepo |

---

## Architecture

### Contract-first monorepo

The `api.yaml` contract is the **single source of truth** for the wire. Every change starts in the contract, goes through `pnpm generate`, and only then reaches the Go and TypeScript code. Generated artifacts (`gen_*.go`, `gen.ts`) are never edited by hand.

### Dependency direction

Dependencies always flow inward:

```
cmd  →  adapters  →  services  →  ports  →  domain
                                            (pure core, stdlib only)
```

- **Pure domain**: `internal/domain/` only imports the Go standard library.
- **Ports and adapters**: services know interfaces (`ports/`), not implementations (`adapters/`).

### Bounded contexts

Four domain contexts, **isolated from one another** (they communicate via ports and append-only events):

| Context | Responsibility | Aggregate root |
|---|---|---|
| `identity` | Portable user identity (data owner). | `User` |
| `practice` | The writing exercise: source text, draft, target rules. | `Practice` |
| `analysis` | The fragment-level analysis produced by the AI. | `Analysis` |
| `analytics` | Materialization of error patterns and progress. | `ErrorMetric`, `ProgressMetric` |

### Repository layout

```text
langlint/
├── apps/
│   ├── contracts/        # OpenAPI SSOT + generation pipeline
│   ├── backend/          # Go: cmd → adapters → services → ports → domain (pending)
│   └── frontend/         # Next.js App Router (pending)
├── docs/
│   ├── checklist/        # 8 phases with exit gates
│   ├── DEVLOG.md         # append-only development log
│   ├── GUIDE_WORK_IA.md  # human-AI collaboration methodology
│   └── PRODUCT_DOMAIN.md # product vision and domain model
├── MANIFEST_MONOREPO.md  # axioms A1–A12 and backend/monorepo patterns
├── MANIFEST_FRONTEND.md  # frontend axioms F1–F12
├── AGENTS.md             # onboarding and working memory for AI agents
├── turbo.json
└── pnpm-workspace.yaml
```

---

## Roadmap

### MVP (7 steps)

| # | Step | Deliverable | Status |
|---|---|---|---|
| 1 | Foundations | Monorepo (Turborepo + pnpm) + foundational OpenAPI contract | In progress |
| 2 | Pure domain | `identity`, `practice`, `analysis`, `analytics` at 100% coverage | Pending |
| 3 | Ports and adapters | `LLMExtractor`, repositories, UoW and outbox + Postgres (testcontainers) | Pending |
| 4 | AI engine | `OpenAIExtractor` with Structured Outputs, anonymization and timeouts | Pending |
| 5 | Frontend | 3-column diff view + analytics dashboard | Pending |
| 6 | Provisioner and privacy | Aggregate/purge jobs + export/delete/access-log endpoints | Pending |
| 7 | Hardening | CI/CD, gosec, govulncheck, documentation and portfolio README | Pending |

### Future (post-MVP): Adaptive Tutor

The product's natural evolution is to turn LangLint from a correction tool into a **personalized, adaptive English tutor** with *spaced repetition* and learning intelligence driven by the user's **real errors**:

> *"I noticed that in your last 5 practices you systematically failed at prepositions before gerunds. Let me build you an in-depth study session on this."*

The system stops merely correcting what is written today and starts generating a **dynamic study plan** from the user's learning friction. It is added as a fifth bounded context (`tutor/`) that consumes `analytics/` aggregates via the event bus (outbox), **without breaking** any of the four existing contexts.

---

## Getting started

### Prerequisites

- **Node.js** 22.13.0 · **pnpm** 9.15.4 · **Go** 1.26.2
- Exact versions live in `.tool-versions` (works with `mise`/`asdf`) and `.nvmrc`.

### Install and build

```bash
pnpm install --frozen-lockfile
pnpm build
```

### Canonical commands

| Command | Purpose |
|---|---|
| `pnpm build` | Build the whole monorepo (Turborepo) |
| `pnpm lint` | Lint the monorepo |
| `pnpm test` | Tier 1+2 tests |
| `pnpm test-integration --filter=backend` | Tier 3 tests (testcontainers) |
| `pnpm generate` | Regenerate Go + TS types from the contract |
| `pnpm typecheck` | Frontend `tsc --noEmit` |

Commands for phases that have not started yet are documented but not available.

---

## Documentation

| Document | Content |
|---|---|
| [`docs/PRODUCT_DOMAIN.md`](docs/PRODUCT_DOMAIN.md) | Vision, domain model, contract, roadmap |
| [`MANIFEST_MONOREPO.md`](MANIFEST_MONOREPO.md) | Axioms A1–A12 and backend patterns |
| [`MANIFEST_FRONTEND.md`](MANIFEST_FRONTEND.md) | Frontend axioms F1–F12 |
| [`docs/GUIDE_WORK_IA.md`](docs/GUIDE_WORK_IA.md) | Human-AI collaboration methodology |
| [`docs/DEVLOG.md`](docs/DEVLOG.md) | Development log |
| [`docs/checklist/`](docs/checklist/) | Phase-by-phase plan with exit gates |
| [`AGENTS.md`](AGENTS.md) | Onboarding for AI agents |

---

## License

Distributed under the **Apache-2.0** license. See [`LICENSE`](LICENSE).
