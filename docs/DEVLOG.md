# DEVLOG — Bitácora de Desarrollo

> Bitácora **append-only** del proyecto. Registrar aquí decisiones, avances, bloqueos y próximos pasos. Cada sesión la actualiza al cerrar. Las entradas más recientes van arriba.

---

## 2026-09-18 — Fase 4 (inicio): `OpenAIExtractor` + prompt builder (4.1.1–4.1.3)

**Estado**: Fase 4 en curso. `go build/vet/test -race` en verde; nuevo paquete `adapters/llm` al **100%** de cobertura; dominio sigue 100%; A1 verificado.

**Hecho**:
- Dependencia `github.com/openai/openai-go v1.12.0` (pin del manifiesto §2.1). `go mod tidy` la deja como directa.
- `4.1.1` `internal/api/adapters/llm/openai_extractor.go`: `OpenAIExtractor{client openai.Client, model, modelVersion}` + `NewOpenAIExtractor` + `Extract` en modo JSON simple (`Chat.Completions.New` → `Choices[0].Message.Content` → `json.Unmarshal` a `[]analysis.Fragment`). Getters `Model()`/`ModelVersion()` para que el service futuro persista la trazabilidad en `Analysis`.
- `4.1.2` `internal/api/adapters/llm/prompt.go`: `buildPrompt` puro (system con el contrato de salida + taxonomía de `ErrorPattern` derivada de las constantes del dominio; user con source/draft/target rules). Consume el input **ya anonimizado** de `ExtractRequest` (A8).
- `4.1.3` `var _ ports.LLMExtractor = (*OpenAIExtractor)(nil)`; el proveedor se inyecta por el puerto y el dominio no lo conoce (A1).
- Mapeo de errores: fallo del proveedor, transporte, respuesta vacía o JSON no parseable → `*domain.LLMUnavailableError{Message:"llm unavailable"}` (sin filtrar el error crudo).
- Tests con `httptest.Server` + `openai.WithBaseURL` (sin `mockgen`): contrato y taxonomía del prompt, request enviado (model + prompt), respuesta válida, error 5xx, JSON inválido, sin `choices` y metadata del modelo.

**Decisiones**:
- **JSON-mode simple ahora; Structured Outputs estricto diferido a 4.2.1** (confirmado con el humano): 4.1.1 solo parsea `message.content`. El JSON Schema estricto (`response_format` + `Message.Parsed`) y la validación contra schema (4.2.2) llegan en su item, evitando encadenar trabajo no aprobado.
- **Paquete `adapters/llm`** (no `openai`): evita la colisión de nombre con el paquete del SDK y mantiene el adaptador provider-agnóstico a nivel de paquete.
- `openai.Client` se guarda **por valor** (lo que devuelve `NewClient`), no como puntero.
- **Timeout (4.3.3) y `PIIHandler` (4.3.4) no se adelantan**: `Extract` recibe y propaga `ctx` (AP6) y no lanza goroutines propias, por lo que la regla `goroutine_context` se cumple por construcción.
- `go mod tidy` promovió `stretchr/testify` de indirect a **direct**: lo importan los tests de integración ya existentes (Fase 3). Corrección legítima de `tidy`, sin cambio funcional.

**Verificación**:
- `go build ./...` · `go vet ./...` · `go vet -tags=integration ./...` · `go test -race -count=1 ./...` → OK.
- `go test -cover ./internal/api/adapters/llm/` → **100.0%**.
- A1: `go list -deps ./internal/domain/... | grep -c openai` → `0`.
- `gofmt -l .` limpio. Sin `t.Skip()`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 4, bloque `4.2` — Structured Outputs con el JSON Schema de `Fragment[]` (4.2.1), validación del output antes de persistir (4.2.2) y orquestación `AnalysisService` (LLM fuera de la transacción, 4.2.3/4.2.4).

---

## 2026-09-18 — Fase 3 (cierre de implementación): Outbox + Event Bus (3.3.3 · 3.4.1–3.4.4)

**Estado**: Tier 1 y Tier 3 en verde. Adaptadores ≥70% (`events` 86.2%, `postgres` 74.1%, `repositories` 75.7%); dominio 100%. `3.3.3` y `3.4.x` marcados con nota: el **enforce en services y la cobertura de services** quedan pendientes de que existan services (Fase 4).

**Hecho**:
- Dominio (`events.go`, aditivo): `OutboxEvent{ID, EventType, Payload []byte, CreatedAt, PublishedAt *time.Time, Attempts}` + registro `NewEvent(eventType) (DomainEvent, bool)` (única fuente de los 4 tipos). Tests al 100%.
- `adapters/events/codec.go`: `MarshalEvent`/`UnmarshalEvent` (JSON + `domain.NewEvent`).
- `3.4.1` `adapters/events/in_memory_dispatcher.go`: `InMemoryEventDispatcher` (canal buffered, worker pool, **drop policy** para backpressure, `Run(ctx)`); implementa `ports/events.EventDispatcher`. Tests Tier 1: entrega, sin suscriptores, drop (buffer=1), defaults.
- `3.4.3` (mecanismo) `adapters/postgres/outbox.go`: `PostgresOutbox` implementa `ports/events.Outbox`; `INSERT` en `outbox_events` vía `txctx` (une la tx del UoW, o pool si no hay).
- `3.4.2` `adapters/events/outbox_relay.go`: `Tick(ctx)` (tx con `FOR UPDATE SKIP LOCKED`, `dispatch`, marca `published_at`; no decodificables/fallidos incrementan `attempts`) y `Run(ctx, intervalo)`.
- Tests Tier 3 (`adapters/events`): commit→entrega al handler + `published_at` marcado; rollback→nada; event-type desconocido→`attempts+1`; `Run` publica. `adapters/postgres`: `PostgresOutbox` fuera y dentro de tx.
- `3.3.3` check con `go list`: `pgx` solo en `adapters/*` y `migrations/`; dominio sin `pgx`.

**Decisiones**:
- **`NewEvent` (registro) en el dominio**: única lista de eventos concretos; el codec (adapter) solo hace JSON. Evita duplicar el catálogo en cada adapter.
- **Codec en `adapters/events`; `PostgresOutbox` (en `adapters/postgres`) lo importa** → dirección `adapters/postgres → adapters/events`, sin ciclo (eventos no importa postgres).
- **`Tick` en transacción con `FOR UPDATE SKIP LOCKED`**: evita doble publicación y bloquea filas mientras se marcan.
- **Idempotencia de handlers diferida a Fase 4**: aquí se entrega el mecanismo (entrega at-least-once vía outbox; `published_at` evita reenvío). Los handlers reales (p. ej. `AnalysisCompleted`→`analytics`, ya con `Upsert` absoluto) llegan con los services.
- **3.4.3/3.4.4 y gate "services ≥90%"**: no aplicables sin services (Fase 4). Se documenta en el checklist en lugar de adelantar Fase 4.
- **Test directo de `PostgresOutbox` en su paquete**: la cobertura de `postgres` bajó a 29.6% al añadir `outbox.go` (se ejercía solo desde el test de `events`); el test propio la devuelve a 74.1%.

**Verificación**:
- `go build ./...` · `go vet ./...` (`-tags=integration`) · `go test -race -count=1 ./...` → OK.
- `pnpm test-integration` → 1 successful. Sin `t.Skip`. `gofmt -l` limpio.
- Cobertura: adapters `events` 86.2% / `postgres` 74.1% / `repositories` 75.7%; dominio 100%.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 4 — Motor de IA (`docs/checklist/04-*.md`): `OpenAIExtractor` (Structured Outputs) + anonimización + timeouts, y los services/event_handlers que cierran `3.4.3`/`3.4.4` y la cobertura de services.

---

## 2026-09-18 — Fase 3: repositorios Postgres + UoW + migraciones (3.2.1–3.2.4 · 3.3.1–3.3.2)

**Estado**: Tier 1 y Tier 3 en verde. Cobertura adapters: `postgres` 72.7%, `repositories` 75.7% (gate ≥70%). Dominio sigue 100%.

**Hecho**:
- Deps (pins del manifiesto §2.1): `jackc/pgx/v5 v5.10.0`, `pressly/goose/v3 v3.27.2`, `stretchr/testify v1.11.1`, `testcontainers/testcontainers-go v0.44.0` (+ `modules/postgres`).
- `domain.InternalError` (A5, 500 `internal` reservado) + tests; `PRODUCT_DOMAIN §4.5` actualizado.
- `3.2.3` `apps/backend/migrations/000001_init.sql` (goose Up/Down): `practices`, `analyses` (FK 1:1 a practices), `error_metrics`, `outbox_events` + índices (incl. parcial `outbox_events WHERE published_at IS NULL`). `migrations.go` con `embed.FS` + `goose.Up` (vía `stdlib.OpenDBFromPool`).
- `3.3.2` `adapters/postgres/txctx/tx_context.go`: `txKey{}` privado + `With`/`From`. `3.3.1` `adapters/postgres/unit_of_work.go`: `PostgresUnitOfWork` con `Begin` → `With` → `Commit`/`Rollback`.
- `3.2.1`/`3.2.2` `adapters/postgres/repositories/`: interfaz privada `querier` (la satisfacen `pgx.Tx` y `*pgxpool.Pool`) + `conn(ctx)` tx-o-pool; `PostgresPracticeRepository`, `PostgresAnalysisRepository`, `PostgresErrorMetricRepository`.
- `3.2.4` `repositories/errors.go`: `pgx.ErrNoRows → *domain.NotFoundError`; `23505 → *domain.ValidationError`; resto → `*domain.InternalError`.
- Tests Tier 3 (`//go:build integration`) con `postgres:16-alpine`: roundtrip/actualización/soft-delete/paginación/aislamiento por usuario; análisis (roundtrip/completado/not-found); métricas (Upsert absoluto/filtro por ventana); UoW commit y **rollback**. Helper compartido `testsupport.Start()` (un contenedor por suite, `TestMain`).

**Decisiones**:
- **Goose no soporta `.up.sql`/`.down.sql` separados** (v3.27 los colecciona como dos migraciones de la misma versión → `duplicate version 1`). Se usa el formato nativo de goose: un archivo `000001_init.sql` con `-- +goose Up`/`-- +goose Down`. **Desviación del nombre literal del checklist** (documentada en el propio checklist).
- **`window` es palabra reservada en Postgres**: la columna se cita como `"window"` en DDL y queries.
- **Sin `tenant_id`/RLS** (single-user MVP; el dominio no tiene `TenantID`). Scoping por `user_id`. A6 se añadirá de forma aditiva con multi-usuario.
- **`tx_context` en subpaquete `txctx`**: `repositories/` es un paquete Go distinto; un helper privado en `postgres` no sería accesible. `txctx` mantiene la clave privada y compartida por UoW y repos (satisface 3.3.2: `tx_context.go`, `txKey` privado, helper).
- **Frontera SQL explícita**: `domain.ID` (`[16]byte`) se convierte a/desde `string` (`String()`/`ParseID`); JSONB con `json.Marshal` + cast `$n::jsonb` y `json.Unmarshal` al leer (pgx no usa `encoding/json`).
- **`Upsert` de `ErrorMetric` absoluto** (`count = EXCLUDED.count`), no incremental, para no romper la idempotencia del handler (AP7).
- **`cmd/migrate` y `shared/db` diferidos** (no están en 3.2); el runner embebido ya sirve a tests y futuro CLI.
- Errores inesperados de DB → `*domain.InternalError` con mensaje genérico (sin filtrar pgx/PgError al service).

**Verificación**:
- `go build ./...` OK · `go vet ./...` OK · `go test -race -count=1 ./...` OK.
- `go test -race -count=1 -tags=integration ./...` / `pnpm test-integration` → OK (1 successful).
- `go test -tags=integration -cover ./internal/api/adapters/...` → `postgres` 72.7%, `repositories` 75.7%.
- `go test -cover ./internal/domain/...` → 100% en los 5 paquetes. `gofmt -l` limpio. Sin `t.Skip()`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 3, `3.3.3` (verificar que los services no importan `pgx`; quedará significativo cuando existan services) y bloque `3.4` — `InMemoryEventDispatcher` + `OutboxRelay` (publicación vía `outbox.Append` dentro de la tx, handlers idempotentes).

---

## 2026-09-18 — Fase 3: integración del backend en turbo + cierre de drift (seguimiento de 3.1)

**Estado**: gate de 3.1 sigue en verde; además quedan operativos los comandos canónicos del monorepo sobre el backend Go.

**Hecho**:
- **Doc drift cerrado**: `PRODUCT_DOMAIN §6.1` pasa de `[]domain.Fragment` a `[]analysis.Fragment` (alineado con el puerto real, 3.1.3).
- **`apps/backend/package.json`** (`"name": "backend"`): scripts `build`/`lint`/`test`/`test-integration`/`generate`/`mocks`. `pnpm-lock.yaml` actualizado con el importer del workspace. Ahora `pnpm test-integration --filter=backend` (gate de Fase 3), `pnpm build`, `pnpm test` y `pnpm lint` orquestan el módulo Go.

**Decisiones**:
- **`ProgressMetric` no se materializa ni se porta (por ahora)**: el checklist `3.2.3` **no** incluye tabla `progress_metrics` (solo `practices`, `analyses`, `error_metrics`, `outbox_events`), así que `GET /analytics/progress` se resolverá **derivando** de `analyses`/`error_metrics` en la capa de consulta. Si el job `refresh-aggregates` (Fase 6) exige persistirlo, se añadirá entonces el puerto + tabla (aditivo).
- **Naming**: se usa `"backend"` (sin scope) para coincidir con los scripts raíz ya commiteados (`--filter=backend`) y con el gate de Fase 3 literal. Nota (no bloqueante): `apps/frontend` es `@langlint/frontend`, por lo que `pnpm dev:frontend` (`--filter=frontend`) no resuelve; se corregirá al cablear el frontend (Fase 5).

**Verificación**:
- `pnpm test-integration` → `backend:test-integration` OK (1 successful); `pnpm build` → 2 successful (contracts + backend); `pnpm test` → backend OK; `pnpm lint` → contracts valid + backend `go vet` OK.
- Warnings cosméticos de turbo (`no output files found for backend#build/test/test-integration`): los scripts Go no emiten artefactos a `coverage/**`/`bin/**`. Inofensivos.

**Bloqueos**: ninguno.

**Pendiente de decisión del humano**: `docs/promnt.txt` (template de sesión, tracked) quedó modificado (Fase 1 → Fase 3) sin commitear. Opciones: commitear el template, o `git rm --cached` + `.gitignore` (junto con `docs/promnt2.txt`).

**Próximo paso**: Fase 3, bloque `3.2` — repositorios Postgres + migraciones `goose`.

---

## 2026-09-18 — Fase 3 (inicio): puertos e interfaces + mocks (items 3.1.1–3.1.5)

**Estado**: Fase 3 en curso. `go build ./...`, `go vet ./...` y `go test -race -count=1 ./...` en verde; mocks idempotentes; `gofmt -l` sin diferencias.

**Hecho** (creado `apps/backend/internal/api/ports/`, solo `context` + raíz `domain` + BCs, A2):
- `3.1.1` puertos de repositorio en `ports/storage/`:
  - `PracticeRepository{Save, GetByID, ListByUser}`.
  - `AnalysisRepository{Save, GetByPracticeID}`.
  - `ErrorMetricRepository{Upsert, ListByUser(window)}`.
- `3.1.2` `ports/storage/unit_of_work.go`: `UnitOfWork{InTransaction(ctx, func(ctx) error) error}` (AP8).
- `3.1.3` `ports/llm_extractor.go` (package `ports`): `ExtractRequest{PracticeID, SourceText, DraftText, TargetRules []practice.TargetRule}` + `LLMExtractor.Extract(ctx, ExtractRequest) ([]analysis.Fragment, error)`.
- `3.1.4` `ports/events/`: `EventDispatcher{Dispatch, Subscribe}`, `EventHandler{Handle}`, `Outbox{Append}`, todos sobre `domain.DomainEvent`.
- `3.1.5` `go.uber.org/mock v0.6.0` añadido a `go.mod`; target `mocks` en `apps/backend/Makefile` (mockgen reflect-mode, pineado, agrupado por paquete) → `ports/mocks/{storage,events,llm_extractor}.go` generados y commiteados.

**Decisiones**:
- **3.1.3 devuelve `[]analysis.Fragment`**: el checklist y `PRODUCT_DOMAIN §6.1` dicen `[]domain.Fragment`, pero `Fragment` vive en `internal/domain/analysis/` (`fragment.go:7`). El puerto importa el BC `analysis` (permitido por A2); se corrige la redacción del checklist (documentado como drift, sin cambiar el modelo).
- **Set de métodos mínimo y orientado a los flujos de §4.7 y a los endpoints de §5.1**: `ListByUser` con `limit/offset` + total para la paginación; `GetByPracticeID` como único acceso a `Analysis` (invariante 1:1 práctica↔análisis). **`ProgressMetric` no tiene puerto** en 3.1.1; su persistencia/derivación se difiere (no se amplía el alcance del checklist).
- **mockgen en reflect-mode** (no `-source`): un archivo por paquete de puertos, con `-package mocks`. La dependencia `go.uber.org/mock` queda **directa** en `go.mod` porque el código generado importa `gomock` (se ejecutó `go mod tidy`).
- **`make mocks` depende de `tools`**: se añade la instalación pineada de `mockgen@v0.6.0` junto a `oapi-codegen`, reproduciable en CI.

**Verificación**:
- `go build ./...` → OK · `go vet ./...` → OK · `go test -race -count=1 ./...` → OK (5 paquetes de dominio; puertos/mocks sin tests, compilan).
- `make mocks` ×2 + `sha256sum` antes/después → idénticos (idempotencia).
- `gofmt -l internal/api/ports/` → sin diferencias.
- A2: imports de `ports/` = `context` + `internal/domain` + BCs (`analysis`, `analytics`, `practice`); sin `pgx` ni infra (la mención a pgx en `unit_of_work.go` es solo comentario).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 3, bloque `3.2` — repositorios Postgres en `adapters/postgres/repositories/` + migraciones `goose` (items 3.2.1–3.2.4). Antes/después: `3.3` (`PostgresUnitOfWork` + `ctxTx`) y `3.4` (dispatcher + relay). Requiere `testcontainers-go` (AP-MR8) y el `apps/backend/package.json` para `pnpm test-integration --filter=backend` (hoy inexistente).

---

## 2026-09-17 — Docs: guía didáctica de Fase 2 (`docs/explains/fase-2-dominio-puro.md`)

**Estado**: documentación. Sin cambios de código; gate de Fase 2 sigue en verde.

**Hecho**: creado `docs/explains/fase-2-dominio-puro.md` con la misma estructura que `fase-1-fundaciones.md`:
1. Explicación para no técnicos (analogía del "reglamento y las piezas de un juego de mesa").
2. Guía de estudio por bloques (dominio puro/DDD, raíz `domain/`, `identity/`, `practice/`, `analysis/`, `analytics/`, verificación/gate) con snippets reales y notas de concepto.
3. Resumen de cambios en 3 viñetas.

**Decisión**: `ErrorPattern` se documenta en la raíz `domain/` (no en `analysis/`) explicando el porqué A2/A3.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 3 — Puertos y adaptadores (`docs/checklist/03-puertos-adaptadores.md`).

---

## 2026-09-17 — Fase 2 (cierre): `analytics/`, errores A5 y Gate de salida (items 2.4.1–2.5.6)

**Estado**: **Fase 2 completada y Gate de salida en verde.** `go test -race -count=1 ./...` OK; `internal/domain/...` con **100%** de cobertura en los 5 paquetes; `go build ./...` y `go vet ./...` OK.

**Hecho**:
- `2.4.2` nuevo paquete `apps/backend/internal/domain/analytics/` (solo stdlib + raíz `domain`): VO `Window` (`day|week|month`) con `IsValid`/`String`.
- `2.4.1` agregado `ErrorMetric{UserID, Code, Window, Count, LastSeenAt}` (clave natural `(UserID, Code, Window)`, sin UUID): `NewErrorMetric` valida `code`/`window` y arranca `Count=1`; `Record(now)` incrementa `Count` y actualiza `LastSeenAt` (§7.1). Agregado `ProgressMetric{UserID, Window, TotalFragments, ErrorCount, Accuracy}`: `NewProgressMetric` valida `window`/conteos `>= 0` y deriva `Accuracy`.
- `2.5.2` completado `errors.go`: añadidos `NotFoundError` (404), `AnalysisPendingError` (409), `AnalysisFailedError` (reservado), `LLMUnavailableError` (503) — todos con shape `{Field, Message}` + `Error()`, consistente con `ValidationError`/`InvalidStateError`.
- `2.5.4`/`2.5.5` smoke checks: imports de `internal/domain/` = **solo stdlib + raíz `domain`**; ningún BC importa a otro (`go list`).
- `2.5.6` todos los fields exportados de structs del dominio llevan tag `json` (verificado con un checker AST temporal, ver abajo).

**Decisiones**:
- **`Window` en `analytics/`** (no en la raíz): §4.3 lo asigna solo a analytics; no hay consumo cruzado.
- **`Accuracy` con clamp**: `NewProgressMetric` valida `TotalFragments >= 0` y `ErrorCount >= 0`; `Accuracy = 1 - ErrorCount/TotalFragments` con clamp inferior a `0`; `TotalFragments == 0 → Accuracy = 1`. Confirmado con el humano. El clamp superior (`> 1`) es innecesario: con `ErrorCount >= 0`, la fórmula nunca supera `1`; no se añade rama muerta.
- **`ErrorMetric`/`ProgressMetric` sin UUID**: son agregados materializados por clave natural (`Upsert` por tupla), no entidades con identidad propia.
- **Tags `json` en los errores**: AP2 los exime (no cruzan el wire), pero 2.5.6 pide "todos los fields exportados". Se añaden tags a los 6 errores para que el checklist sea literalmente cierto y consistente con logging estructurado futuro; inocuo (nadie marshala errores hoy).
- **Verificación 2.5.6 con checker AST temporal** (`/tmp/opencode/checktags`, no commiteado): parsea `internal/domain/**/*.go` (sin tests) y comprueba que todo field exportado de struct tiene tag `json`. Resultado: `OK`. No se añade herramienta al repo (el lint `domain_purity`/`forbidden_imports` de golangci-lint es de Fase 7).
- **`make test` no existe**: el `Makefile` solo tiene `tools`/`generate`; la verificación canónica usada es `go test -race -count=1 ./...` + `go test -cover`, no `make test` (no se inventan comandos).

**Verificación**:
- `go test -race -count=1 ./...` → OK (domain, analysis, analytics, identity, practice).
- `go test -cover ./internal/domain/...` → **100.0%** en los 5 paquetes.
- `go build ./...` → OK · `go vet ./...` → OK · `gofmt -l` → sin diferencias.
- Purity: `grep` de imports → `crypto/rand`, `encoding/hex`, `fmt`, `net/mail`, `strings`, `time` + raíz `domain`.
- Gate de salida de Fase 2: las 4 casillas en verde (cobertura 100%, `-race`, purity, convención de nombres).

**Bloqueos**: ninguno.

**Próximo paso**: **Fase 3 — Puertos y adaptadores** (`docs/checklist/03-*.md`): puertos `LLMExtractor`, repositorios, `UnitOfWork` y outbox en `internal/api/ports/` + adaptadores Postgres con testcontainers.

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
