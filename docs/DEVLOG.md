# DEVLOG — Bitácora de Desarrollo

> Bitácora **append-only** del proyecto. Registrar aquí decisiones, avances, bloqueos y próximos pasos. Cada sesión la actualiza al cerrar. Las entradas más recientes van arriba.

---

## 2026-09-17 — Fase 2: bounded context `analysis/` (items 2.3.1–2.3.5)

**Estado**: Fase 2 en curso. `go test -race -count=1 ./...` en verde; `internal/domain/...` con **100%** de cobertura; `go build ./...` y `go vet ./...` OK.

**Hecho**:
- Nuevo paquete `apps/backend/internal/domain/analysis/` (solo stdlib + raíz `domain`, A1):
  - `2.3.1` agregado `Analysis{ID, PracticeID, Fragments, Model, ModelVersion, Status, CreatedAt}`; `NewAnalysis(...)` genera UUID v7 (seam `generateID`) y arranca en `pending`; `Complete(fragments)` (pending → completed) y `Fail()` (pending → failed) con guard de estado (`InvalidStateError`).
  - `2.3.2` VO `Fragment` (7 campos de §5.2 con tags `snake_case`, incl. `error_patterns []domain.ErrorPattern`) — struct plano, sin validación (el adapter valida el JSON del LLM contra el schema, §6.2).
  - `2.3.4` invariante: `Complete` rechaza `fragments` vacío con `ValidationError`.
- `2.3.3` raíz `domain/error_pattern.go`: `ErrorPatternCode` (8 códigos + `IsValid`), `ErrorPatternSeverity` (`minor|moderate|critical` + `IsValid`), `ErrorPattern{Code, Severity, Note}` + `NewErrorPattern` (valida enums).
- `2.3.5` raíz `domain/events.go`: `AnalysisCompleted{AnalysisID, PracticeID, UserID, ErrorPatterns, Version}` y `AnalysisFailed{AnalysisID, PracticeID, Reason, Version}` + `EventName*`.

**Decisiones**:
- **`ErrorPattern` (y sus enums) en la raíz `domain/`**, no en `analysis/`: A2/A3 y el payload `[]ErrorPattern` de `AnalysisCompleted` (evento en la raíz) más el consumo de `ErrorPattern.Code` por `analytics/` obligan a que sea un tipo compartido (como `ID`). El checklist 2.3.3 lo agrupa bajo `analysis/`, pero su ubicación literal rompería A2/A3. Confirmado con el humano.
- **`Fragment` struct plano** (sin constructor validante): el contrato solo exige que los campos *estén presentes*, no que sean no vacíos; la validación del output del LLM es del adapter (§6.2).
- **`Analysis.CreatedAt`** existe en el dominio (§4.2.3) aunque el schema `Analysis` del contrato no lo exponga; no hay drift (la conversión a wire la hará el handler).
- **Sin errores nuevos**: 2.3 usa `ValidationError` (fragments vacío) e `InvalidStateError` (guards). `AnalysisPendingError`/`AnalysisFailedError`/`NotFoundError`/`LLMUnavailableError` (2.5.2) se difieren a cuando un servicio los necesite.
- **2.5.3 marcado completo**: los 4 eventos del MVP (§4.4) ya están definidos con `Version`.

**Verificación**:
- `go test -race -count=1 ./...` → OK (domain, analysis, identity, practice).
- `go test -cover ./internal/domain/...` → **100.0%** en los cuatro paquetes.
- `go build ./...` → OK · `go vet ./...` → OK · `gofmt -l` → sin diferencias.
- A1: imports de `analysis/` = `time` + raíz `domain`. A3: `analysis/` **no** importa `practice/` ni `identity/` (referencia por `PracticeID domain.ID`).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 2, bloque `2.4` — bounded context `analytics/` (agregados `ErrorMetric`/`ProgressMetric`, VO `Window` `day|week|month`).

---

## 2026-09-17 — Fase 2: bounded context `practice/` (items 2.2.1–2.2.5)

**Estado**: Fase 2 en curso. `go test -race -count=1 ./...` en verde; `internal/domain/...` con **100%** de cobertura; `go build ./...` y `go vet ./...` OK.

**Hecho**:
- Nuevo paquete `apps/backend/internal/domain/practice/` (solo stdlib + raíz `domain`, A1):
  - `2.2.2` value objects: `PracticeStatus` (`draft|analyzing|completed|failed`, `IsValid`/`String`), `SourceText` y `DraftText` (no vacíos, preservan el texto crudo), `TargetRule{Verb,Tense,Note}` (`verb` obligatorio).
  - `2.2.1` agregado `Practice{ID, UserID, SourceText, DraftText, TargetRules, Status, CreatedAt, UpdatedAt, DeletedAt}`; `NewPractice(...)` genera UUID v7 (seam `generateID`) y arranca en `draft`.
  - `2.2.3` invariantes: textos no vacíos (en los VOs), ≥1 `TargetRule` (`NewPractice`/`Edit`), transición `draft → analyzing → completed|failed` vía `StartAnalysis`/`MarkCompleted`/`MarkFailed` (cada una bumps `UpdatedAt`).
  - `2.2.5` `Edit` (solo en `draft`, PATCH parcial: `nil` = sin cambio; reglas vacías → `ValidationError`) y `Delete` (soft-delete: fija `DeletedAt *time.Time`; bloquea `analyzing`; re-borrado → `InvalidStateError`).
- `2.2.4` raíz `domain/events.go`: `EventNamePracticeCreated = "practice.created"` + `PracticeCreated{PracticeID, UserID, Version}` con tags `snake_case`.
- `2.5.2` (parcial) raíz `domain/errors.go`: nuevo `InvalidStateError{Field, Message}` (409 `invalid_state`), mismo patrón que `ValidationError`.

**Decisiones**:
- **`DeletedAt *time.Time`** (interno, `json:"-"`): da al job `purge-raw-data` la ventana de retención (A8); no se expone en el wire (el contrato `Practice` no lo tiene).
- **Solo no-vacío** en `SourceText`/`DraftText`: el checklist y §4.2.2 solo exigen no-vacío; la "longitud acotada" de §4.3 se difiere a cuando el contrato fije un `maxLength`.
- **Métodos de transición incluidos ya** (`StartAnalysis`/`MarkCompleted`/`MarkFailed`) para que el invariante de 2.2.3 sea verificable; el wiring a `AnalysisCompleted`/`AnalysisFailed` es de Fase 2.3/3.
- **`Edit` con `nil` = sin cambio** y no-op sin tocar `UpdatedAt`; el `minProperties: 1` del `PATCH` se valida en el wire, no en el dominio.
- **Sin guard de `DeletedAt` en `Edit`/transiciones**: el filtrado de borradas es responsabilidad de la capa de persistencia (query). El agregado solo enforce las reglas del contrato (estado).

**Verificación**:
- `go test -race -count=1 ./...` → OK (domain, identity y practice).
- `go test -cover ./internal/domain/...` → **100.0%** en los tres paquetes.
- `go build ./...` → OK · `go vet ./...` → OK · `gofmt -l` → sin diferencias.
- A1: imports de `internal/domain/practice/` = `strings`, `time` + raíz `domain`. A3: `practice/` **no** importa `identity/` ni ningún otro BC.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 2, bloque `2.3` — bounded context `analysis/` (agregado `Analysis`, VOs `Fragment`/`ErrorPattern`, invariante de `Fragments` no vacío cuando `Status == completed`, eventos `AnalysisCompleted`/`AnalysisFailed`).

---

## 2026-09-17 — Fase 2 (inicio): bootstrap raíz + bounded context `identity` (items 2.1.1–2.1.4)

**Estado**: Fase 2 en curso. `go test -race -count=1 ./...` en verde; dominio con **100%** de cobertura; `go build ./...` y `go vet ./...` OK.

**Hecho**:
- Módulo Go creado: `apps/backend/go.mod` → `module github.com/c0mp1lerworld/langlint/backend` + `go 1.26`; `go mod tidy` resuelve `go-chi/chi/v5 v5.3.2` y `oapi-codegen/runtime v1.7.0` (deps que ya importaban los `gen_*.go`), generando `go.sum`.
- `2.5.1` (bootstrap) `internal/domain/identifiers.go`: `ID [16]byte` UUID v7 con **solo** `crypto/rand` + `time` + `fmt` + `encoding/hex`. API: `NewID`, `MustNewID`, `ParseID`, `IsValid`, `Version`, `IsZero`, `String`, `MarshalJSON`/`UnmarshalJSON`.
- `2.5.2` (parcial) `internal/domain/errors.go`: `ValidationError{Field, Message}` + `Error()`.
- `2.5.3` (bootstrap) + `2.1.3` `internal/domain/events.go`: interfaz `DomainEvent` (`EventName()`) + struct inmutable `IdentityIssued{UserID, Version}` con tags `snake_case` y constante `EventNameIdentityIssued`.
- `2.1.1` + `2.1.4` `internal/domain/identity/user.go`: entidad `User{ID, Email, CreatedAt}`; `NewUser` genera el ID. **Sin `tenant_id`** (AP1).
- `2.1.2` `internal/domain/identity/email.go`: VO `Email` (`type Email string`) con `NewEmail` validando vía `net/mail` (stdlib) → `*domain.ValidationError`.
- Tests TDD por paquete (convención `TestXxx_Method_Condition_ExpectedResult`), incluyendo el test que prueba que el JSON de `User` **no** tiene `tenant_id`.

**Decisiones**:
- **Eventos en la raíz** (`events.go`), no en `identity/`: A3/§4.4 obligan a que los `DomainEvent` los defina el paquete raíz; los BC no se importan entre sí.
- **`ID` como `[16]byte`** (comparable, usable como map key) con `MarshalJSON` a UUID canónico; así el payload del outbox y `DataExport.user_id` (`format: uuid`) serializan como string.
- **`Email` como `type Email string`**: valida por constructor y serializa como string sin `MarshalJSON` custom.
- **Seams `randRead`/`generateID`** (vars con default a `crypto/rand.Read`/`domain.NewID`): permiten forzar el fallo de generación en tests y cumplir el gate de **100%** de cobertura (A10) sin ramas muertas.
- **Alcance**: bootstrap mínimo de la raíz (solo `ValidationError`); `NotFoundError`, `InvalidStateError`, etc. se implementan en sus items (2.2.x/2.3.x/2.4.x/2.5.2).
- **Orden checklist vs dependencias**: los items 2.1.x dependían de las primitivas 2.5.x; se resolvió con un bootstrap mínimo de la raíz en el mismo incremento (acordado con el humano).

**Verificación**:
- `go test -race -count=1 ./internal/domain/...` → OK; con `-cover` → **100.0%** (domain y identity).
- `go mod tidy` → OK · `go build ./...` → OK · `go vet ./...` → OK · `go test -race -count=1 ./...` → OK.
- A1: imports de `internal/domain/` = solo stdlib + raíz `domain` (permitido por A3). `github.com/google/uuid` figura como **indirect** (dep de `oapi-codegen/runtime`) pero **no** se importa desde el dominio (A7).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 2, bloque `2.2` — bounded context `practice/` (agregado `Practice`, value objects `SourceText`/`DraftText`/`TargetRule`/`PracticeStatus`, invariantes y `Edit`/`Delete` con `InvalidStateError`). Antes, ampliar `errors.go` con los tipos que 2.2 necesita.

---

## 2026-09-17 — Fase 1: CRUD de `Practice` (PATCH + DELETE)

**Estado**: gate de Fase 1 sigue en verde. `pnpm generate` idempotente, `pnpm lint` `valid` (6 warnings `operation-4xx-response`), `pnpm build` OK.

**Contexto**: se evaluó si la superficie permitía un CRUD completo. Faltaban `Update` y `Delete` sobre `Practice`; se añaden (contract-first).

**Hecho (todo en `apps/contracts/openapi/api.yaml` → `pnpm generate`)**:
1. **`PATCH /practices/{practiceId}`**: edición **parcial** (`UpdatePracticeRequest` con `source_text`/`draft_text`/`target_rules` opcionales, `minProperties: 1`). Solo permitido en `status == "draft"` → `200 Practice`; si no, `409 invalid_state`.
2. **`DELETE /practices/{practiceId}`**: **soft-delete síncrono** → `204`; bloqueado con `409 invalid_state` si `status == "analyzing"`. La purga física la hará `purge-raw-data` (A8).
3. **Nuevo código `invalid_state`** (409) en `ErrorResponse.code` + componente de respuesta `InvalidState`.
4. Regenerados `gen_types.go` (nuevo `UpdatePracticeRequest`, alias `InvalidState = ErrorResponse`, enum `ErrorResponseCodeInvalidState`), `gen_server.go` (`DeletePractice`/`UpdatePractice`) y `gen.ts`.

**Decisiones**:
- **PATCH parcial** (no PUT): semánticamente correcto para editar un borrador; editable solo en `draft`.
- **Soft-delete + `204`**: coherente con la retención limitada (A8); reversible y purgable por el provisioner. Se descarta el hard-delete inmediato.
- **Código propio `invalid_state`** (no reutilizar `analysis_pending`): `analysis_pending` es un **éxito idempotente** del endpoint `analyze`; los conflictos de estado son errores reales y distintos (AP3: un concepto, un tipo).
- **Versionado**: cambio **aditivo** (nuevos endpoints + valor de enum) → `apps/contracts` sigue en `1.0.0`.

**Docs actualizados**: `docs/PRODUCT_DOMAIN.md` §5.1 (dos filas nuevas + nota de ciclo de vida) y §4.5 (`InvalidStateError` → 409 `invalid_state`). `docs/checklist/02-dominio-puro.md`: nuevo item `2.2.5` (métodos `Edit`/`Delete` del agregado) y `2.5.2` ampliado con `InvalidStateError`.

**Impacto en Fase 2**: el agregado `Practice` debe exponer `Edit(...)` (solo `draft`) y `Delete(...)` (soft-delete) y el error `InvalidStateError`.

**Verificación**: `pnpm generate` ×2 + `git diff --exit-code` (idempotente) → `pnpm lint` (`valid`, 6 warnings) → `pnpm build` (1 successful).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 2 — Dominio puro. Primero el módulo Go (`go.mod`, module path) y resolver la nota de bounded contexts del manifiesto (§3.1).

---

## 2026-09-17 — Fase 1: endurecimiento del contrato OpenAPI (revisión REST, puntos 1–4)

**Estado**: gate de Fase 1 sigue en verde. `pnpm generate` idempotente, `pnpm lint` `valid` (6 warnings `operation-4xx-response`, ya conocidos), `pnpm build` OK.

**Contexto**: revisión de la superficie REST antes de generar clientes. Se corrigieron 4 puntos; los puntos 5 (acciones como sub-recursos), 6 (idempotencia de `POST /practices`) y 7 (sin `PUT/PATCH` de práctica) se aceptan como decisiones/deuda conocida del MVP.

**Hecho (todo en `apps/contracts/openapi/api.yaml` → `pnpm generate`)**:
1. **`window` en `/analytics/*`**: nuevo `components.parameters.WindowQuery` (query, `$ref` a `Window`, opcional, `default: week`) en `GET /analytics/error-patterns` y `GET /analytics/progress`.
2. **Paginación**: `components.parameters.Limit` (int, 1..100, default 20) y `Offset` (int, ≥0, default 0) en `GET /practices` y `GET /me/access-log`. Nueva envoltura `PracticeList { items, total }`; `AccessLog` pasa de `{ entries }` a `{ items, total }` (renombrado `entries` → `items` por consistencia).
3. **Fallos de análisis**: `GET /practices/{practiceId}` ya no devuelve `422 analysis_failed`; se comunica con `200` y `analysis.status == "failed"`. Se elimina el componente `AnalysisFailed` (huérfano). `analysis_failed` permanece en el enum de `ErrorResponse.code` como valor reservado. El `422` queda exclusivo de `validation_error`.
4. **`Retry-After`**: header `Retry-After` (integer, segundos, `required: true`) en el `202` de `POST /practices/{practiceId}/analyze`.

**Decisiones**:
- `window` **opcional con default `week`** (no rompe llamadas sin parámetro). `limit`/`offset` con los defaults indicados.
- Versionado: `apps/contracts` se mantiene en **`1.0.0`**. El cambio de forma de respuesta (`array` → `{items,total}`) es breaking, pero el contrato **no está publicado ni tiene consumidores**; se documenta aquí en lugar de saltar a `2.0.0` (AP-MR6 se aplicará cuando exista release real).
- Generación: `$ref` + `default` (sibling 3.1) produjo alias limpios en Go (`WindowQuery = Window`, `Limit = Offset = int`).

**Docs actualizados**: `docs/PRODUCT_DOMAIN.md` §5.1 (tabla de endpoints + notas de idempotencia, fallos sin 4xx, paginación/ventana) y §4.5 (fila de `AnalysisFailedError`). `docs/explains/fase-1-fundaciones.md` revisado: sin contenido obsoleto.

**Verificación**: `pnpm generate` ×2 + `git diff --exit-code` (idempotente) → `pnpm lint` (`valid`, 6 warnings) → `pnpm build` (1 successful).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 2 — Dominio puro. Primero crear el módulo Go (`go.mod`, module path) y resolver la nota de bounded contexts del manifiesto (§3.1).

---

## 2026-09-17 — Fase 1 (cierre): pipeline de generación `pnpm generate` (items 1.4.1–1.4.5)

**Estado**: **Gate de salida de Fase 1 en verde**. `pnpm install --frozen-lockfile`, `pnpm generate` (idempotente), `pnpm build`, `pnpm lint` (OpenAPI validado) — todo OK.

**Hecho**:
- `1.4.1` `apps/contracts/scripts/generate.sh`: orquestador que invoca `make -C apps/backend generate` (Go) y `pnpm run generate` en `apps/frontend` (TS). `apps/contracts/package.json` → `"generate": "bash scripts/generate.sh"` + `"lint": "redocly lint openapi/api.yaml"`.
- `1.4.2` `apps/backend/Makefile` con `tools` (instala `oapi-codegen@v2.8.0`) y `generate` (dos invocaciones). Configs `oapi-codegen-types.yaml` (models → `internal/api/handlers/gen_types.go`) y `oapi-codegen-server.yaml` (chi-server → `internal/api/handlers/gen_server.go`), package `httpapi`.
- `1.4.3` `apps/frontend/package.json`: `"generate": "openapi-typescript ../contracts/openapi/api.yaml -o src/lib/api/gen.ts"` + devDeps `openapi-typescript@^7.13.0` y `typescript@5.9.3`. Genera `apps/frontend/src/lib/api/gen.ts`.
- `1.4.4` `gen_types.go` (343 líneas), `gen_server.go` (404) y `gen.ts` (569) commiteados (builds offline, sin `go.mod` — ver decisiones).
- `1.4.5` Idempotencia verificada: `pnpm generate` ×2 + `git diff --exit-code` → sin drift.
- Bump `apps/contracts` `0.1.0 → 1.0.0` (gate `contracts-v1.0.0`; el tag git se difiere: no hay push).
- Lint OpenAPI cableado de forma permanente (`@redocly/cli@^2.53.3` como devDep de contracts, script `lint`), cerrando lo acordado en 1.3.

**Decisiones**:
- **`go.mod` diferido a Fase 2** (acordado): el pipeline solo genera/commitea. `gen_server.go` importa `go-chi/chi/v5` y `oapi-codegen/runtime`, por lo que **no compilará** hasta que Fase 2 cree el módulo (A11). Es deliberado; el gate de Fase 1 no exige compilar Go.
- **oapi-codegen v2 con archivo de config**: el manifiesto §6.3 muestra flags v1 (`-generate types`, `-package`) **obsoletos** en la versión actual. Se usa v2.8.0 con config YAML (package `httpapi`, dos salidas), preservando la intención (dos archivos `gen_*.go`).
- **Orquestación sin doble ejecución**: `turbo run generate` ejecutaría el `generate` de contracts (orquestador) **y** el de frontend, generando `gen.ts` dos veces en paralelo. Se acota el script raíz a `turbo run generate --filter=@langlint/contracts` para que el orquestador sea la única vía. `pnpm generate` es el comando canónico (no invocar `turbo run generate` a pelo).
- **TS 5.9.3 local en frontend**: `openapi-typescript@7.13.0` es incompatible con el `typescript@7.0.2` de la raíz (peer `^5.x`; `ts.factory` undefined). Se añade TS 5.9.3 como devDep de frontend (solo afecta a la generación; la raíz mantiene TS7).
- **`ProgressPoint.accuracy` sale `float32`** (OpenAPI `number` sin `format`). Si se quiere `float64`, cambiar a `format: double` en `api.yaml` y regenerar. Se difiere (no bloquea).

**Verificación**:
- `pnpm generate` ×2 → `git diff --exit-code` limpio (idempotente/drift-free).
- `pnpm install --frozen-lockfile` → OK. `pnpm build` → `1 successful, 1 total` (persiste el warning conocido de outputs del stub de contracts).
- `pnpm lint` → redocly: `valid` (0 errores, 6 warnings `operation-4xx-response` ya documentados en 1.3).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 2 — Dominio puro (`docs/checklist/02-dominio-puro.md`). Primer item sugerido `2.5.1`/`2.1.x`: crear el módulo Go (`go.mod`, module path) y arrancar `identity/`. Antes, resolver la nota de bounded contexts del manifiesto (§3.1 lista `organization/billing/audit`; el producto usa `identity/practice/analysis/analytics`).

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
