# LangLint

> **AI-Powered Writing & Grammar Analytics** — plataforma de aprendizaje de idiomas mediante escritura productiva contextual, con retroalimentación de una IA que actúa como profesor nativo.

**Español** · [English](README.en.md)

[![Contract](https://img.shields.io/badge/contract-3.4.0-informational)](apps/contracts/CHANGELOG.md)
[![Go](https://img.shields.io/badge/Go-1.26.8-00ADD8)](apps/backend/go.mod)
[![Next.js](https://img.shields.io/badge/Next.js-15-black)](apps/frontend/package.json)

Este repositorio es una **demostración de arquitectura de software moderna**:
monorepo polyglot (Go + Next.js), contrato OpenAPI como *Single Source of Truth*
y Domain-Driven Design estricto, con gates de calidad automatizados.

---

## ¿Qué es LangLint?

Los métodos tradicionales de aprendizaje (tarjetas de memoria tipo Anki)
funcionan para la **memorización pasiva**, pero fallan en la **producción
activa**: el estudiante reconoce una palabra y aun así no logra escribir un texto
correcto con ella. LangLint cierra esa brecha.

El usuario practica escritura productiva contextual y una IA desmenuza su texto
**fragmento por fragmento**, explicando el *porqué* gramatical detrás de cada
error y clasificándolo. Un motor de analíticas rastrea los errores recurrentes a
lo largo del tiempo para eliminar puntos ciegos.

### Flujo de uso

1. El usuario define un **conjunto de objetivos gramaticales** (p. ej. verbos
   irregulares a practicar).
2. Escribe un **texto base en español** aplicando esos objetivos.
3. Redacta su **traducción experimental al inglés** (el borrador).
4. La **IA** corrige y explica fragmento por fragmento, clasificando cada error
   (preposición, posesivo, falso amigo…).
5. El **dashboard de analíticas** muestra errores recurrentes y progreso.

### Propuesta de valor

| Para | Valor entregado |
|---|---|
| El estudiante | Pasar de reconocer una palabra a escribirla correctamente en contexto; entender el *porqué*, no solo ver la corrección. |
| El producto (portafolio) | Demostración de arquitectura hexagonal adaptada, contract-first, event-driven y monorepo polyglot. |

---

## Estado del proyecto

- **MVP funcional completo**: Fases 1–6 (fundaciones, dominio puro, puertos y
  adaptadores, motor IA, frontend y provisioner/privacidad) en verde.
- **Fase 7 — Hardening**: CI/CD (`7.1`), Seguridad (`7.2`) y Documentación
  (`7.3`) completadas; **Gate de Fase 7 en verde** (primer run real de CI sin
  errores).
- **Fase 8 — Tutor Adaptativo (post-MVP)**: quinto bounded context `tutor/`,
  detección de debilidades (`WeaknessDetected` vía outbox) y generación de
  sesiones de estudio (`POST /v1/study-sessions`), aditivo y sin rupturas.
- **Fase 9 — Feedback profundo y práctica activa**: explicaciones estructuradas,
  quiz con IA y analytics del quiz (`9.1`–`9.7`, contrato `3.4.0`).
- **Fase 10 — Backlog del tutor**: historial de sesiones, elección de tema y
  repetición espaciada (ver [`docs/checklist/10-tutor-mejoras.md`](docs/checklist/10-tutor-mejoras.md)).
- **Non-goals del MVP**: multi-usuario real, multi-idioma y streaming
  (ver [`PRODUCT_DOMAIN.md §3.2`](docs/PRODUCT_DOMAIN.md)).

---

## Stack tecnológico

| Capa | Tecnología |
|---|---|
| Lenguajes | Go 1.26 · TypeScript 5.9 (strict) |
| Frontend | Next.js 15 (App Router) · React 19 · Tailwind CSS 4 |
| Estado | TanStack Query (server state) · Zustand (client state) |
| Formularios | React Hook Form + Zod |
| Backend | Go: arquitectura hexagonal adaptada · Uber Fx · chi |
| Contrato | OpenAPI 3.1 (`apps/contracts/openapi/api.yaml`) |
| Generación | `oapi-codegen` (Go) · `openapi-typescript` (TS) |
| Persistencia | PostgreSQL 16 · `pgx` · goose · Unit of Work · Outbox |
| IA | OpenAI con Structured Outputs, detrás del puerto `LLMExtractor` |
| Tests | `go test -race` · testcontainers · Vitest · RTL · Playwright |
| Monorepo | pnpm workspaces + Turborepo (remote cache self-hosted) |
| Seguridad | `gosec` · `govulncheck` · auditorías estáticas |

---

## Arquitectura

### Monorepo contract-first (A12)

El contrato `api.yaml` es la **única fuente de verdad** del wire. Todo cambio
empieza en el contrato, pasa por `pnpm generate` (que regenera Go **y** TS) y
recién entonces llega al código. Nunca se editan a mano `gen_*.go` ni `gen.ts`.
CI falla si el código generado diverge del contrato.

### Dirección de dependencias (A2)

```
cmd  →  adapters  →  services  →  ports  →  domain
                                            (núcleo puro, solo stdlib, A1)
```

### Bounded contexts aislados (A3)

| Contexto | Responsabilidad | Agregado raíz |
|---|---|---|
| `identity` | Identidad portable y auditoría de datos (A9). | `User` |
| `practice` | El ejercicio: texto base, borrador y reglas objetivo. | `Practice` |
| `analysis` | El análisis fragmentado generado por la IA. | `Analysis` |
| `analytics` | Materialización de patrones de error y progreso. | `ErrorMetric` |
| `tutor` | Tutor adaptativo (post-MVP): debilidades y sesiones de estudio. | `StudySession` |

Se comunican por puertos y **eventos de dominio** vía outbox, nunca por imports
cruzados. Los detalles están en [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

### Transacciones y eventos

- **Unit of Work**: los services no tocan SQL; el adaptador abre
  `BEGIN/COMMIT/ROLLBACK`.
- **Outbox**: el cambio de estado y el evento se persisten en la misma
  transacción; un relay los publica post-commit.

### Privacidad (A8/A9)

- **Anonimización** antes del LLM y **pseudonimización** (HMAC-SHA256) en
  analytics; separación de datos crudos y purga programada.
- `PIIHandler` como guard runtime del logging y auditorías estáticas en CI.

---

## Estructura del repositorio

```text
langlint/
├── apps/
│   ├── contracts/          # SSOT OpenAPI + pipeline de generación
│   ├── backend/            # Go: cmd → adapters → services → ports → domain
│   └── frontend/           # Next.js App Router (diff de 3 columnas + analíticas)
├── docs/                   # ARCHITECTURE, RUNBOOK, PRODUCTION_ENV, SECRET_ROTATION, …
├── ops/                    # docker-compose (Postgres + turbo-cache) y scripts de auditoría
├── .github/workflows/      # ci · contracts · deploy-staging · security
├── MANIFEST_MONOREPO.md    # axiomas A1–A12 y patrones del backend/monorepo
├── MANIFEST_FRONTEND.md    # axiomas F1–F12 del frontend
├── turbo.json · pnpm-workspace.yaml · .tool-versions
└── AGENTS.md               # onboarding y memoria de trabajo para agentes de IA
```

---

## Métricas de rigor

| Métrica | Valor | Cómo se verifica |
|---|---|---|
| Cobertura del dominio | **100 %** | `go test -cover ./internal/domain/...` |
| Cobertura de services | **91–98 %** | `go test -cover ./internal/api/services/...` |
| Tests backend (Tier 1+2) | en verde | `pnpm test` (`go test -race`) |
| Integración (Tier 3) | en verde | `pnpm test-integration --filter=backend` (testcontainers) |
| Tests frontend (Vitest) | **99** | `pnpm --filter frontend test` |
| e2e (Playwright) | 2 | `pnpm test:e2e` |
| Análisis estático de seguridad | **0 hallazgos** | `gosec` / `govulncheck` (CI `security.yml`) |
| Contrato | **3.4.0** | `contracts.yml`: drift + `info.version` |

---

## Cómo empezar

### Prerrequisitos

- **Go** 1.26.8 · **Node** 22 · **pnpm** 9.15.4 · Docker (para Postgres).
- Versiones exactas en `.tool-versions` (compatible con `mise`/`asdf`) y `.nvmrc`.

### Instalación y arranque

```bash
pnpm install --frozen-lockfile

# Infraestructura local (Postgres :5433)
docker compose -f ops/docker/docker-compose.yml up -d

# Backend
cd apps/backend && cp .env.example .env   # rellenar secretos reales (A8)
go run ./cmd/migrate && go run ./cmd/api

# Frontend (otra terminal)
cp apps/frontend/.env.example apps/frontend/.env.local
pnpm --filter frontend dev
```

Ver [`docs/RUNBOOK.md`](docs/RUNBOOK.md) para diagnóstico y
[`docs/PRODUCTION_ENV.md`](docs/PRODUCTION_ENV.md) para todas las variables.

### Comandos canónicos

| Comando | Propósito |
|---|---|
| `pnpm build` | Build de todo el monorepo (Turborepo) |
| `pnpm lint` | Lint del monorepo (redocly + go vet + eslint) |
| `pnpm test` | Tests Tier 1+2 (Go + Vitest) |
| `pnpm test-integration --filter=backend` | Tests Tier 3 (testcontainers) |
| `pnpm test:e2e` | e2e de frontend (Playwright) |
| `pnpm typecheck --filter=frontend` | `tsc --noEmit` |
| `pnpm generate` | Regenera tipos Go + TS desde el contrato |
| `go run ./cmd/provisioner <job>` | Jobs batch (`refresh-aggregates`, …) |

---

## Roadmap

| # | Paso | Estado |
|---|---|---|
| 1 | Fundaciones | ✅ |
| 2 | Dominio puro (100 % cobertura) | ✅ |
| 3 | Puertos y adaptadores (UoW, outbox, Postgres) | ✅ |
| 4 | Motor de IA (Structured Outputs) | ✅ |
| 5 | Frontend (diff de 3 columnas + analíticas) | ✅ |
| 6 | Provisioner y privacidad (A9) | ✅ |
| 7 | Hardening (CI/CD, seguridad, documentación) | ✅ |
| 8 | Tutor Adaptativo (post-MVP: `tutor/`, `POST /v1/study-sessions`) | ✅ |
| 9 | Feedback profundo y práctica activa (quiz + analytics) | ✅ |

### Tutor Adaptativo (Fase 8) y backlog (Fase 10)

LangLint incorpora un **tutor de inglés adaptativo**: un quinto bounded context
(`tutor/`) que consume los agregados de `analytics/` vía el bus de eventos y
genera sesiones de estudio personalizadas (teoría, trampas y ejercicios) con
Structured Outputs, **sin romper** ninguno de los cuatro contextos del MVP. Las
mejoras de historial, elección de tema y repetición espaciada están en el
backlog [`docs/checklist/10-tutor-mejoras.md`](docs/checklist/10-tutor-mejoras.md).
Ver [`PRODUCT_DOMAIN.md §12.1`](docs/PRODUCT_DOMAIN.md).

---

## Documentación

| Documento | Contenido |
|---|---|
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Contrato arquitectónico global |
| [`docs/PRODUCT_DOMAIN.md`](docs/PRODUCT_DOMAIN.md) | Visión, modelo de dominio, contrato, roadmap |
| [`docs/RUNBOOK.md`](docs/RUNBOOK.md) | Operación e incidentes |
| [`docs/PRODUCTION_ENV.md`](docs/PRODUCTION_ENV.md) | Variables de entorno |
| [`docs/SECRET_ROTATION.md`](docs/SECRET_ROTATION.md) | Rotación de secretos |
| [`docs/SECURITY_DEBT.md`](docs/SECURITY_DEBT.md) | Deuda de seguridad aceptada |
| [`docs/API_CONTRACT.md`](docs/API_CONTRACT.md) | Gaps contrato↔dominio |
| [`docs/bugs/`](docs/bugs/) | Bugs conocidos |
| [`MANIFEST_MONOREPO.md`](MANIFEST_MONOREPO.md) | Axiomas A1–A12 y patrones del backend |
| [`MANIFEST_FRONTEND.md`](MANIFEST_FRONTEND.md) | Axiomas F1–F12 del frontend |
| [`AGENTS.md`](AGENTS.md) | Onboarding para agentes de IA |

---

## Licencia

Distribuido bajo la licencia **Apache-2.0**. Consulta [`LICENSE`](LICENSE).
