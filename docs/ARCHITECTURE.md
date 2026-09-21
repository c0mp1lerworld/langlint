# ARCHITECTURE.md — Contrato Arquitectónico Global

> **Jerarquía**: este documento es el **contrato arquitectónico global** del
> proyecto. Condensa y **nunca contradice** los manifiestos; si hay conflicto,
> mandan `MANIFEST_MONOREPO.md` (backend/monorepo), `MANIFEST_FRONTEND.md`
> (frontend) y `apps/contracts/openapi/api.yaml` (wire).
>
> **Audiencia**: cualquier ingeniero (humano o IA) que se incorpore al
> repositorio. Es lectura previa de los manifiestos (§1–§3).

---

## Tabla de contenidos

1. [Propósito y jerarquía documental](#1-propósito-y-jerarquía-documental)
2. [Estructura del monorepo](#2-estructura-del-monorepo)
3. [Backend: arquitectura hexagonal adaptada](#3-backend-arquitectura-hexagonal-adaptada)
4. [Frontend: capas y flujo de datos](#4-frontend-capas-y-flujo-de-datos)
5. [Operación](#5-operación)
6. [Axiomas y documentos autoritativos](#6-axiomas-y-documentos-autoritativos)

---

## 1. Propósito y jerarquía documental

LangLint es un **monorepo polyglot** (Go + TypeScript/Next.js) que construye una
plataforma de práctica de escritura con corrección por IA. La arquitectura es un
**monolito lógico** (A11) con **contrato primero** (A12) y **dominio puro** (A1).

| Pregunta | Documento autoritativo |
|---|---|
| ¿Cuál es el dominio del producto? | [`PRODUCT_DOMAIN.md`](PRODUCT_DOMAIN.md) |
| ¿Cómo se construye el backend/monorepo? | [`MANIFEST_MONOREPO.md`](../MANIFEST_MONOREPO.md) |
| ¿Cómo se construye el frontend? | [`MANIFEST_FRONTEND.md`](../MANIFEST_FRONTEND.md) |
| ¿Cuál es el contrato HTTP (SSOT del wire)? | [`apps/contracts/openapi/api.yaml`](../apps/contracts/openapi/api.yaml) |
| ¿Qué env vars existen? | §5.1 de este documento + [`PRODUCTION_ENV.md`](PRODUCTION_ENV.md) |
| ¿Cuándo rotar secretos? | [`SECRET_ROTATION.md`](SECRET_ROTATION.md) §1.1 |
| ¿Qué hacer ante incidentes? | [`RUNBOOK.md`](RUNBOOK.md) |
| ¿Qué waivers de seguridad son legítimos? | [`SECURITY_DEBT.md`](SECURITY_DEBT.md) |
| ¿Qué gaps contrato↔dominio hay? | [`API_CONTRACT.md`](API_CONTRACT.md) |
| ¿Qué bugs están abiertos/cerrados? | [`bugs/`](bugs/) |

**Tres reglas que gobiernan todo cambio**:

1. **Contract-first (A12)**: el wire empieza en `api.yaml` → `pnpm generate` →
   código. Nunca se editan a mano `gen_*.go` ni `gen.ts`.
2. **Dirección de dependencias hacia adentro (A2)**:
   `cmd → adapters → services → ports → domain`.
3. **Los tests son un gate (A10)**: dominio 100 %, services ≥90 %, adapters
   ≥70 %; sin `t.Skip()`.

---

## 2. Estructura del monorepo

```
langlint/
├── apps/
│   ├── contracts/                  # SSOT OpenAPI + pipeline de generación
│   │   ├── openapi/api.yaml        # Contrato HTTP (versión 3.0.0)
│   │   ├── scripts/generate.sh     # oapi-codegen (Go) + openapi-typescript (TS)
│   │   └── scripts/check-version.sh
│   ├── backend/                    # Go: un único go.mod (A11)
│   │   ├── cmd/{api,provisioner,migrate,llmcheck}/
│   │   ├── internal/
│   │   │   ├── domain/{identity,practice,analysis,analytics}/  # puro (A1)
│   │   │   ├── api/{handlers,services,ports,adapters,accesslog,di}/
│   │   │   ├── provisioner/{handlers,services,ports,adapters,di}/
│   │   │   └── shared/{config,db,httpx,logger,pseudonymizer,testdb}/
│   │   ├── migrations/             # SQL goose (embebidas con //go:embed)
│   │   ├── test/e2e/               # tests Tier 3 cross-cutting
│   │   └── Makefile                # generate, mocks, test, test-integration
│   └── frontend/                   # Next.js 15 (App Router)
│       ├── src/app/                # rutas: /, /practices/[id], /practices/new, /analytics
│       ├── src/features/           # casos de uso (practice, analytics)
│       ├── src/components/{ui,feature}/
│       ├── src/lib/{api,query,store,schemas,utils}/
│       └── src/types/
├── ops/
│   ├── docker/                     # docker-compose (Postgres + turbo-cache)
│   └── scripts/{pii_audit.sh,security_audit.sh}
├── docs/                           # este documento y el resto de la documentación
├── .github/workflows/{ci,contracts,deploy-staging,security}.yml
├── turbo.json · pnpm-workspace.yaml · .tool-versions
└── MANIFEST_MONOREPO.md · MANIFEST_FRONTEND.md · AGENTS.md
```

**Polyglot**: `apps/contracts` es la dependencia común; `apps/backend` (Go) y
`apps/frontend` (TS) **solo se comunican por HTTP** a través del contrato (F3).
No hay imports cruzados entre apps.

---

## 3. Backend: arquitectura hexagonal adaptada

### 3.1 Capas y dirección de dependencias (A2)

```
cmd/  →  adapters/  →  services/  →  ports/  →  domain/
                                              (núcleo puro, solo stdlib)
```

| Capa | Ubicación | Responsabilidad | No hace |
|---|---|---|---|
| **Dominio** | `internal/domain/` | Reglas de negocio puras, entidades, value objects, eventos, errores. **Solo stdlib** (A1). | Saber de HTTP, DB, LLM, JSON. |
| **Puertos** | `internal/<entrypoint>/ports/` | Interfaces que el dominio/aplicación necesita (`LLMExtractor`, repositorios, `UnitOfWork`, `Outbox`). | Implementar. |
| **Services** | `internal/<entrypoint>/services/` | Casos de uso; orquestan puertos. Retornan errores de dominio (A5). | Iniciar transacciones SQL (AP8), importar `pgx`. |
| **Adaptadores** | `internal/<entrypoint>/adapters/` | Implementan puertos (Postgres, OpenAI, outbox relay). | Contener reglas de negocio. |
| **Entry points** | `cmd/` | Composición Fx, arranque/parada de procesos. | Lógica de negocio. |
| **Shared** | `internal/shared/` | Infraestructura transversal (config, db, httpx, logger, pseudonymizer). | Decisiones de negocio. |

La separación `api/` ↔ `provisioner/` es **lógica**, no física: comparten
`domain/` y `shared/`, pero **no se importan entre sí** (A2/A3); los puertos se
duplican por entry-point.

### 3.2 Entry points (`cmd/`)

| Binario | Función | Config |
|---|---|---|
| `cmd/api` | Servidor HTTP (puerto `APP_HTTP_ADDR`, default `:8080`). Monta el contrato y el middleware de auditoría. | `ServerConfig` + `OpenAIConfig` |
| `cmd/provisioner` | Jobs batch: `refresh-aggregates`, `purge-raw-data`, `execute-deletions`. | `ProvisionerConfig` |
| `cmd/migrate` | Aplica las migraciones goose embebidas. | `APP_DATABASE_URL` |
| `cmd/llmcheck` | Sonda de desarrollo para el motor de IA (no es producción). | `OpenAIConfig` |

Cada entry-point tiene su propio `fx.Module` en `internal/<entrypoint>/di/`; no
se comparten módulos Fx entre entry-points.

### 3.3 Bounded contexts (A3)

Cuatro contextos **aislados** que solo comparten primitivos del paquete raíz
`domain` (`ID`, errores sentinela). Se comunican por puertos y eventos de
dominio (nunca importándose entre sí).

| Contexto | Agregado raíz | Responsabilidad |
|---|---|---|
| `identity` | `User`, `AccessEvent`, `DeletionRequest` | Identidad portable (A9), portabilidad, olvido y auditoría de accesos. |
| `practice` | `Practice` | El ejercicio: texto base (es), borrador (en) y reglas objetivo. |
| `analysis` | `Analysis` | El análisis fragmentado generado por la IA. |
| `analytics` | `ErrorMetric`, `ProgressMetric` | Materialización de patrones de error y progreso. |

`analysis` referencia `Practice` por su `ID` (`domain.ID`), sin importar el
paquete `practice`.

### 3.4 Transacciones y comunicación asíncrona

**Unit of Work (AP8)**: el service nunca inicia transacciones SQL. Solo el
adaptador `PostgresUnitOfWork` abre `BEGIN/COMMIT/ROLLBACK`; el service recibe
el puerto y ejecuta `uow.InTransaction(ctx, fn)`.

**Outbox (AP7)**: para publicar hechos de negocio sin publicar antes del commit,
el service hace `outbox.Append(txCtx, event)` **dentro** de la transacción; un
`OutboxRelay` en background lee `outbox_events` post-commit y los entrega al
`InMemoryEventDispatcher`, que invoca los `EventHandler` suscritos.

```
service → uow.InTransaction( → repo.Save + outbox.Append )  → COMMIT
relay   → SELECT outbox_events WHERE published_at IS NULL → dispatcher → handler
```

Reglas: **nunca** `dispatcher.Dispatch()` dentro de una transacción; los
handlers son **idempotentes**.

### 3.5 Motor de IA

El análisis cognitivo vive detrás del puerto `LLMExtractor` (y `TutorQuestioner`
para el quiz). El adaptador `OpenAIExtractor` usa **Structured Outputs** (JSON
Schema estricto) y se invoca **una vez por frase** del borrador; el backend fija
el `user_draft` a partir de su propia segmentación, de modo que cobertura y
orden quedan garantizados por construcción.

- **Anonimización antes del LLM (A8)**: no se envían datos personales.
- **`temperature=0`** y prompt acotado para reducir varianza y truncado.
- **Timeout** configurable con `APP_LLM_TIMEOUT` (default `180s`).

---

## 4. Frontend: capas y flujo de datos

Dirección de dependencias (F2): `app → features → components → lib → types`.

| Capa | Ubicación | Responsabilidad |
|---|---|---|
| `app/` | `src/app/` | Rutas, layouts, Server Components. |
| `features/` | `src/features/` | Casos de uso (practice, analytics). |
| `components/` | `src/components/` | UI presentacional y accesible (WCAG 2.2 AA). |
| `lib/` | `src/lib/` | `api/` (cliente HTTP tipado), `query/` (TanStack Query), `store/` (Zustand), `schemas/` (Zod), `utils/`. |
| `types/` | `src/types/` | Tipos de wire desde `gen.ts` (F1). |

- **Server state** (API, con caché/revalidación) → **TanStack Query**.
  **Client state** (UI, preferencias) → **Zustand** (F7). Nunca se mezclan.
- **Formularios** con React Hook Form + Zod, derivados del contrato (F8).
- **Optimistic UI** con snapshot/rollback (F9).
- Los errores del API se mapean a UX en `lib/api/errors.ts`; los códigos de éxito
  idempotente (`409 analysis_pending`) se tratan como éxito (F5/AP-F6).
- El backend es la **fuente de verdad** de las reglas de negocio; el cliente no
  las reimplementa (F11).

---

## 5. Operación

### 5.1 Variables de entorno (resumen)

> Detalle operativo completo en [`PRODUCTION_ENV.md`](PRODUCTION_ENV.md).

Prefijos por app (AP-MR4): backend `APP_` (y `OPENAI_` para el SDK), frontend
`NEXT_PUBLIC_`, contratos `APP_CONTRACTS_` si aplica.

| Variable | Requerida | Default | Consumidor |
|---|---|---|---|
| `APP_DATABASE_URL` | sí | — | api, provisioner, migrate |
| `APP_HTTP_ADDR` | no | `:8080` | api |
| `APP_USER_ID` | sí | — | api |
| `APP_USER_EMAIL` | sí | — | api |
| `APP_LLM_TIMEOUT` | no | `180s` | api |
| `APP_CORS_ALLOWED_ORIGINS` | no | `http://localhost:3000` | api |
| `APP_PSEUDONYM_SECRET` | sí | — | api, provisioner |
| `APP_RAW_RETENTION_DAYS` | no | `30` | provisioner |
| `APP_DELETION_GRACE_DAYS` | no | `30` | provisioner |
| `OPENAI_API_KEY` | sí | — | api, llmcheck |
| `OPENAI_MODEL` | sí | — | api, llmcheck |
| `OPENAI_MODEL_VERSION` | no | = `OPENAI_MODEL` | api |
| `OPENAI_BASE_URL` | no | `https://api.openai.com/v1` | api, llmcheck |
| `NEXT_PUBLIC_API_URL` | sí (frontend) | — | frontend (build time) |

Los secretos nunca se commitean: `.env` está gitignoreado y `.env.example` es
la plantilla.

### 5.2 Persistencia y migraciones

- **PostgreSQL 16** con `pgx/v5` (SQL explícito, sin ORM).
- Migraciones **goose** embebidas (`//go:embed *.sql` en `migrations/`), aplicadas
  por `cmd/migrate`, `cmd/api` y `cmd/provisioner` al arrancar.
- Tablas: `practices`, `analyses`, `error_metrics`, `outbox_events`,
  `deletion_requests`, `access_events`.
- **Separación de datos crudos y analytics (A8)**: las tablas crudas
  (`practices`/`analyses`) se purgan; `error_metrics` solo guarda una clave
  **pseudonimizada** (HMAC-SHA256 con `APP_PSEUDONYM_SECRET`).
- Tablas **append-only** de verdad de negocio: `outbox_events` (no de negocio),
  `access_events` (auditoría A9, sin Update/Delete).

### 5.3 CI/CD y seguridad

| Workflow | Propósito |
|---|---|
| `.github/workflows/ci.yml` | `detect-changes` + jobs contracts/backend/frontend; Tier 3 en nightly, push `main` o label `ready-for-release`. |
| `.github/workflows/contracts.yml` | Regenera Go+TS y falla por drift; valida `info.version == package.json`. |
| `.github/workflows/deploy-staging.yml` | Build + push de imágenes a GHCR (backend distroless, frontend standalone). |
| `.github/workflows/security.yml` | `gosec` (SARIF) + `govulncheck` + `security_audit.sh`. |

Scripts de auditoría: `ops/scripts/pii_audit.sh` (PII en logs) y
`ops/scripts/security_audit.sh` (secretos, docs, `x-internal`). El backend
**no expone** `/docs` (AP-MR9).

### 5.4 Observabilidad, logging y PII

- Logging con `log/slog` (stdlib). El `PIIHandler` descarta atributos con
  patrones PII (emails, teléfonos, claves, JWT) como **guard runtime** (A8).
- Los únicos loggers de producción (`cmd/api`, `cmd/provisioner`) pasan por
  `logger.New` (envuelve `PIIHandler`).
- La pseudonimización se aplica en los **adaptadores** antes de materializar
  analytics.

---

## 6. Axiomas y documentos autoritativos

**Backend/monorepo (A1–A12)**: dominio puro · dependencias hacia adentro ·
bounded contexts aislados · append-only · errores de dominio · multi-tenancy
(simplificado a single-user) · IDs del dominio (UUID v7) · PII inviolable ·
el usuario es dueño de sus datos · tests como gate · monolito lógico · contrato
SSOT.

**Frontend (F1–F12)**: contrato SSOT del cliente · dependencias hacia adentro ·
sin imports cruzados · mínimo privilegio de PII · errores mapeados a UX ·
accesibilidad por defecto · server/client state separados · formularios contra
el contrato · optimistic UI con rollback · tests como gate · backend como fuente
de verdad · rendimiento por construcción.

- Anti-patrones y reglas transversales: `MANIFEST_MONOREPO.md` §4–§5 y
  `MANIFEST_FRONTEND.md` §4–§5.
- Metodología de trabajo humano-IA: [`GUIDE_WORK_IA.md`](GUIDE_WORK_IA.md).
- Bitácora de sesiones: [`DEVLOG.md`](DEVLOG.md).

---

**Versión**: 1.0.0
**Mantenedor**: equipo de arquitectura.
**Última revisión**: 2026-09-21.
**Próxima revisión**: tras cada release mayor o cuando cambie el contrato
arquitectónico (nuevos bounded contexts, cambios de capas o de despliegue).
