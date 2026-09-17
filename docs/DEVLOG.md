# DEVLOG — Bitácora de Desarrollo

> Bitácora **append-only** del proyecto. Registrar aquí decisiones, avances, bloqueos y próximos pasos. Cada sesión la actualiza al cerrar. Las entradas más recientes van arriba.

---

## 2026-09-17 — Fase 1: contrato OpenAPI fundacional `api.yaml` (items 1.3.1–1.3.6)

**Estado**: Fase 1 en curso. `npx @redocly/cli lint apps/contracts/openapi/api.yaml` → **valid** (0 errores, 6 warnings). `pnpm build` en verde.

**Hecho**:
- `1.3.1` Creado `apps/contracts/openapi/api.yaml` (OpenAPI 3.1.0) con `info.version: 1.0.0` y `servers: [{ url: /v1 }]`.
- `1.3.2` Schemas núcleo: `Fragment`, `ErrorPattern`, `Analysis`, `Practice`, `PracticeStatus` (+ soporte y superficie, ver abajo).
- `1.3.3` `ErrorPatternCode` (los 8 códigos de §4.6) y `ErrorPatternSeverity` (`minor|moderate|critical`) como schemas nominales reutilizables.
- `1.3.4` Los 9 endpoints de §5.1: `/practices*`, `/analytics/*`, `/me/*` (paths relativos a `servers: /v1`).
- `1.3.5` `409 analysis_pending` documentado como **éxito idempotente** en `POST /practices/{practiceId}/analyze` (AP4).
- `1.3.6` Todos los campos de wire en `snake_case` (AP2).

**Decisiones**:
- **Convención de rutas**: `servers: [{ url: /v1 }]` + paths relativos (`/practices`, `/analytics/*`, `/me/*`). Evita la duplicación `/v1/v1/...` del ejemplo del manifiesto §6.3; las URLs finales coinciden con §5.1.
- **Sin `securityScheme`** y `security: []` explícito: el MVP no tiene login/JWT/API keys (non-goal §3.2); identidad portable mínima (A9).
- **Enums extraídos**: `ErrorPatternCode`/`ErrorPatternSeverity` como schemas nominales (en lugar de enums inline) para reutilizarlos en `ErrorPatternStats` y facilitar un tipo Go reutilizable en 1.4.
- **Superficie completa**: además de los 5 schemas del checklist se modelaron `CreatePracticeRequest`, `TargetRule`, `Window`, `PracticeDetail`, `ErrorPatternStats`/`ErrorPatternAggregate`, `ProgressSeries`/`ProgressPoint`, `DataExport`, `AccessLog`/`AccessLogEntry` y `ErrorResponse`. Los no detallados en los docs se definieron coherentes con §4.2/§4.3/§7 y §9 (cierre del gap de 1.3.4).
- **`PracticeDetail`** repite los campos de `Practice` (en lugar de `allOf`) para que `oapi-codegen` genere structs planos sin sorpresas.
- **`AnalysisFailed` (422)** se asocia a `GET /practices/{practiceId}` (única superficie donde el fallo del análisis es observable); así el response deja de estar huérfano.
- **Warnings aceptados**: Redocly emite `operation-4xx-response` en 6 GET (list, analytics×2, me/export, me/data, me/access-log). No se añade un 4xx artificial porque en el MVP single-user esos endpoints no tienen error de entrada real (AP5: no documentar lo que no existe). Se revisará si aparece un 4xx legítimo.

**Verificación**:
- `npx --yes @redocly/cli lint apps/contracts/openapi/api.yaml` → `valid` (0 errores, 6 warnings de `operation-4xx-response`).
- `pnpm build` → `1 successful, 1 total`.

**Bloqueos**: ninguno.

**Próximo paso**: bloque `1.4` — pipeline de generación (`apps/contracts/scripts/generate.sh`, `oapi-codegen` en `apps/backend/Makefile`, `openapi-typescript` en `apps/frontend`).

---

## 2026-09-17 — Fase 1: árbol canónico de directorios (items 1.2.1–1.2.4)

**Estado**: Fase 1 en curso. Gate `pnpm install --frozen-lockfile && pnpm build` en verde.

**Hecho**:
- `1.2.1` Creados `apps/contracts/openapi/`, `apps/contracts/scripts/`, `apps/backend/` y `apps/frontend/` (según §3.1 del manifiesto).
- `1.2.2` Creados `ops/scripts/` y `ops/docker/`. `docs/` ya existía (`PRODUCT_DOMAIN.md` incluido), no se tocó.
- `1.2.3` Creado `.github/workflows/` como placeholder para Fase 7.
- `1.2.4` Verificado que no hay wire code fuera de `apps/contracts/openapi/`: no existe todavía ningún `gen_*.go`, `gen.ts` ni spec OpenAPI en el repo.
- Cada directorio vacío lleva un `.gitkeep` (git no versiona directorios vacíos).
- Actualizado el estado de `AGENTS.md` (estaba en Fase 0 pese a que 1.1.x ya estaba hecho).

**Decisiones**:
- Alcance ajustado al checklist: solo directorios de primer nivel. `apps/backend/internal/**` (Fase 2), `apps/frontend/src/**` (Fase 5) y `ops/terraform/` + `ops/k8s/` (posteriores) se difieren; el árbol §3.1 es plantilla, no se materializa completo aquí.
- `.gitkeep` como placeholder mínimo en lugar de `README.md`/`package.json` prematuros.

**Observación (no bloqueante)**: el árbol §3.1 del manifiesto lista los bounded contexts `identity/organization/billing/audit`, pero `PRODUCT_DOMAIN`/`DEVLOG` fijaron `identity/practice/analysis/analytics`. No afecta a 1.2.x (no se crean subdirs de `internal/domain/`). A resolver antes de Fase 2.

**Bloqueos**: ninguno.

**Próximo paso**: item `1.3.1` (redactar `apps/contracts/openapi/api.yaml` como OpenAPI 3.1.0) y siguientes del bloque 1.3.

---

## 2026-09-17 — Fase 1 (inicio): base del monorepo (items 1.1.1–1.1.6)

**Estado**: Fase 1 en curso. `pnpm install --frozen-lockfile && pnpm build` en verde.

**Hecho**:
- `package.json` raíz: `"private": true`, `"packageManager": "pnpm@9.15.4"`, scripts globales (`build`, `lint`, `test`, `test-integration`, `generate`, `typecheck`, `dev:backend`, `dev:frontend`).
- `pnpm-workspace.yaml` con `packages: ['apps/*']`.
- `turbo.json` con tasks `build` (dependsOn `^build`), `lint`, `test`, `test-integration`, `typecheck`, `dev`, `generate` (según manifiesto §2.2).
- `.nvmrc` (Node 22), `.tool-versions` (golang 1.26.2, nodejs 22.13.0, pnpm 9.15.4), `.editorconfig`, `.gitignore`.
- `devDependencies`: `turbo@^2.10.13`, `typescript@^7.0.2` (versiones concretas para reproducibilidad con `--frozen-lockfile`, en lugar de `latest`).
- Placeholder `apps/contracts/package.json` (`@langlint/contracts`, `build` stub) para que `turbo run build` tenga objetivo.
- `git init` + commit inicial (primer commit del repositorio).

**Decisiones**:
- Se usa un placeholder mínimo de `apps/contracts` con `build` stub para que el gate `pnpm build` quede verde sin adelantar `1.2.1`/`1.3`/`1.4` (los dirs `openapi/` y `scripts/generate.sh` siguen pendientes).
- `typescript@7.0.2` (native) y `turbo@2.10.13` resueltos por pnpm como `latest` (manifiesto decía `latest`); se pinnean con caret.

**Bloqueos**: ninguno.

**Nota**: `turbo` emite `WARNING no output files found for task @langlint/contracts#build` (el `build` stub no produce artefactos y `turbo.json` declara `outputs`). Inofensivo; desaparecerá cuando contracts genere outputs reales (1.4).

**Próximo paso**: items `1.2.x` (árbol canónico de directorios) y `1.3.x` (contrato OpenAPI `api.yaml`).

---

## 2026-09-15 — Fase 0: Preparación y fundamentos documentales

**Estado**: antes de Fase 1. Sin código.

**Hecho**:
- `MANIFEST_MONOREPO.md` (v2.0.0) y `MANIFEST_FRONTEND.md` (v1.0.0) como base normativa.
- `docs/PRODUCT_DOMAIN.md` (v1.0.0): visión, modelo de dominio (4 bounded contexts + futuro `tutor`), contrato, motor IA, analítica, privacidad, matriz de trazabilidad, DoD, roadmap.
- `docs/checklist/` con 8 fases (7 MVP + 1 post-MVP), cada una con subfases `x.x`, procedimientos `x.x.x` y `Gate de salida`.
- `AGENTS.md` (memoria de trabajo para agentes de IA).
- `docs/GUIDE_WORK_IA.md` (metodología de colaboración humano-IA).

**Decisiones de diseño bloqueadas**:
- Bounded contexts propios: `identity`, `practice`, `analysis`, `analytics` (se descartan `organization`/`billing` del template B2B).
- **Single-user MVP**: identidad portable mínima (A9), sin login real.
- Tutor adaptativo (`tutor/`) como **fase 8 post-MVP** (§12.1 del PRODUCT_DOMAIN).
- Proveedor IA: OpenAI (`openai-go`) detrás del puerto `LLMExtractor`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 1 — Fundaciones (`docs/checklist/01-fundaciones.md`): inicializar monorepo + contrato OpenAPI fundacional.
