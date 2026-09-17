# LangLint

> **AI-Powered Writing & Grammar Analytics** — plataforma de aprendizaje de idiomas mediante escritura productiva contextual con retroalimentación de una IA que actúa como profesor nativo.

**Español** · [English](README.en.md)

---

## Estado del proyecto

- **Fase actual**: Fase 1 — Fundaciones (monorepo base en verde).
- **MVP**: single-user, sin login real. El par de idiomas del MVP es **español → inglés**.
- **Código**: `apps/` aún en construcción. La Fase 0 (documentación y fundamentos) está completa.

Este repositorio es una **demostración de arquitectura de software moderna**: monorepo polyglot (Go + Next.js), contrato OpenAPI como *Single Source of Truth* y Domain-Driven Design estricto.

---

## ¿Qué es LangLint?

Los métodos tradicionales de aprendizaje (tarjetas de memoria tipo Anki) funcionan para la **memorización pasiva**, pero fallan en la **producción activa**: el estudiante reconoce una palabra y aun así no logra escribir un texto correcto con ella. LangLint cierra esa brecha.

El usuario practica escritura productiva contextual y una IA desmenuza su texto **fragmento por fragmento**, explicando el *porqué* gramatical detrás de cada error y clasificándolo. Un motor de analíticas rastrea los errores recurrentes a lo largo del tiempo para eliminar puntos ciegos.

### Flujo de uso

1. El usuario define un **conjunto de objetivos gramaticales** (p. ej. verbos irregulares a practicar).
2. Escribe un **texto base en español** aplicando esos objetivos.
3. Redacta su **traducción experimental al inglés** (el borrador).
4. La **IA** corrige y explica fragmento por fragmento, clasificando cada error (preposición, posesivo, falso amigo…).
5. El **dashboard de analíticas** muestra errores recurrentes y progreso temporal.

### Propuesta de valor

| Para | Valor entregado |
|---|---|
| El estudiante | Pasar de reconocer una palabra a escribirla correctamente en contexto; entender el *porqué*, no solo ver la corrección. |
| El producto (portafolio) | Demostración de arquitectura hexagonal, contract-first, event-driven y monorepo polyglot. |

---

## Stack tecnológico

| Capa | Tecnología |
|---|---|
| Lenguajes | Go 1.26 · TypeScript (strict) |
| Frontend | Next.js 14+ (App Router) · React Server Components · Tailwind CSS |
| Estado | TanStack Query (server state) · Zustand (client state) |
| Formularios | React Hook Form + Zod |
| Backend | Go (arquitectura hexagonal adaptada) |
| Contrato | OpenAPI 3.1 (`apps/contracts/openapi/api.yaml`) |
| Generación | `oapi-codegen` (Go) · `openapi-typescript` (TS) |
| Persistencia | PostgreSQL · Unit of Work · patrón Outbox |
| IA | Proveedor LLM con Structured Outputs (OpenAI, detrás del puerto `LLMExtractor`) |
| Monorepo | pnpm workspaces + Turborepo |

---

## Arquitectura

### Monorepo contract-first

El contrato `api.yaml` es la **única fuente de verdad** del wire. Todo cambio empieza en el contrato, pasa por `pnpm generate` y recién entonces llega al código Go y TypeScript. Nunca se editan a mano los artefactos generados (`gen_*.go`, `gen.ts`).

### Dirección de dependencias

Las dependencias fluyen siempre hacia adentro:

```
cmd  →  adapters  →  services  →  ports  →  domain
                                            (núcleo puro, solo stdlib)
```

- **Dominio puro**: `internal/domain/` solo importa la stdlib de Go.
- **Puertos y adaptadores**: los servicios conocen interfaces (`ports/`), no implementaciones (`adapters/`).

### Bounded contexts

Cuatro contextos de dominio, **aislados entre sí** (se comunican vía puertos y eventos append-only):

| Contexto | Responsabilidad | Agregado raíz |
|---|---|---|
| `identity` | Identidad portable del usuario (dueño de los datos). | `User` |
| `practice` | El ejercicio de escritura: texto base, borrador, reglas objetivo. | `Practice` |
| `analysis` | El análisis fragmentado generado por la IA. | `Analysis` |
| `analytics` | Materialización de patrones de error y progreso temporal. | `ErrorMetric`, `ProgressMetric` |

### Estructura del repositorio

```text
langlint/
├── apps/
│   ├── contracts/        # SSOT OpenAPI + pipeline de generación
│   ├── backend/          # Go: cmd → adapters → services → ports → domain (pendiente)
│   └── frontend/         # Next.js App Router (pendiente)
├── docs/
│   ├── checklist/        # 8 fases con gate de salida
│   ├── DEVLOG.md         # bitácora append-only de sesiones
│   ├── GUIDE_WORK_IA.md  # metodología de colaboración humano-IA
│   └── PRODUCT_DOMAIN.md # visión y modelo de dominio
├── MANIFEST_MONOREPO.md  # axiomas A1–A12 y patrones del backend/monorepo
├── MANIFEST_FRONTEND.md  # axiomas F1–F12 del frontend
├── AGENTS.md             # onboarding y memoria de trabajo para agentes de IA
├── turbo.json
└── pnpm-workspace.yaml
```

---

## Roadmap

### MVP (7 pasos)

| # | Paso | Entregable | Estado |
|---|---|---|---|
| 1 | Fundaciones | Monorepo (Turborepo + pnpm) + contrato OpenAPI fundacional | En curso |
| 2 | Dominio puro | `identity`, `practice`, `analysis`, `analytics` con 100% de cobertura | Pendiente |
| 3 | Puertos y adaptadores | `LLMExtractor`, repositorios, UoW y outbox + Postgres (testcontainers) | Pendiente |
| 4 | Motor de IA | `OpenAIExtractor` con Structured Outputs, anonimización y timeouts | Pendiente |
| 5 | Frontend | Vista diff de 3 columnas + dashboard de analíticas | Pendiente |
| 6 | Provisioner y privacidad | Jobs de agregados/purga + endpoints de export/delete/access-log | Pendiente |
| 7 | Hardening | CI/CD, gosec, govulncheck, documentación y README de portafolio | Pendiente |

### Futuro (post-MVP): Tutor Adaptativo

La evolución natural del producto es transformar LangLint de una herramienta de corrección en un **tutor de inglés personalizado y adaptativo** con *spaced repetition* e inteligencia de aprendizaje basada en los **errores reales** del usuario:

> *"He notado que en tus últimas 5 prácticas has fallado sistemáticamente en las preposiciones antes de gerundios. Voy a generarte una sesión de estudio profunda sobre esto."*

El sistema deja de solo corregir lo escrito hoy y pasa a generar un **plan de estudio dinámico** a partir de la fricción de aprendizaje del usuario. Se incorpora como un quinto bounded context (`tutor/`) que consume los agregados de `analytics/` vía el bus de eventos (outbox), **sin romper** ninguno de los cuatro contextos existentes.

---

## Cómo empezar

### Prerrequisitos

- **Node.js** 22.13.0 · **pnpm** 9.15.4 · **Go** 1.26.2
- Las versiones exactas están en `.tool-versions` (compatible con `mise`/`asdf`) y `.nvmrc`.

### Instalación y build

```bash
pnpm install --frozen-lockfile
pnpm build
```

### Comandos canónicos

| Comando | Propósito |
|---|---|
| `pnpm build` | Build de todo el monorepo (Turborepo) |
| `pnpm lint` | Lint del monorepo |
| `pnpm test` | Tests Tier 1+2 |
| `pnpm test-integration --filter=backend` | Tests Tier 3 (testcontainers) |
| `pnpm generate` | Regenerar tipos Go + TS desde el contrato |
| `pnpm typecheck` | `tsc --noEmit` del frontend |

Los comandos de fases aún no iniciadas están documentados pero no disponibles.

---

## Documentación

| Documento | Contenido |
|---|---|
| [`docs/PRODUCT_DOMAIN.md`](docs/PRODUCT_DOMAIN.md) | Visión, modelo de dominio, contrato, roadmap |
| [`MANIFEST_MONOREPO.md`](MANIFEST_MONOREPO.md) | Axiomas A1–A12 y patrones del backend |
| [`MANIFEST_FRONTEND.md`](MANIFEST_FRONTEND.md) | Axiomas F1–F12 del frontend |
| [`docs/GUIDE_WORK_IA.md`](docs/GUIDE_WORK_IA.md) | Metodología de colaboración humano-IA |
| [`docs/DEVLOG.md`](docs/DEVLOG.md) | Bitácora de desarrollo |
| [`docs/checklist/`](docs/checklist/) | Plan por fases con gates de salida |
| [`AGENTS.md`](AGENTS.md) | Onboarding para agentes de IA |

---

## Licencia

Distribuido bajo la licencia **Apache-2.0**. Consulta [`LICENSE`](LICENSE).
