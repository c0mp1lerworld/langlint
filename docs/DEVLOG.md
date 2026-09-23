# DEVLOG — Bitácora de Desarrollo

> Bitácora **append-only** del proyecto. Registrar aquí decisiones, avances, bloqueos y próximos pasos. Cada sesión la actualiza al cerrar. Las entradas más recientes van arriba.

---

## 2026-09-22 — Migración de prácticas manuales (tool `cmd/import`)

**Estado**: importadas **16 prácticas reales** con sus fechas (2026-09-11 ×5, 09-12 ×7, 09-15 ×4), análisis regenerado con el LLM actual y analytics reconstruidos vía `refresh-aggregates`. Se añade el tool one-off `cmd/import` (commiteado) con TDD en el parser; `import.json` queda **gitignored** (datos personales). `go build/vet`, `go test -race ./cmd/import/...` en verde.

**Hecho**:
- `cmd/import` (`main.go` + `import_test.go`): lee un JSON (`{date, source_text, draft_text, target_rules}`), valida vía value objects del dominio (máx 5 reglas, 2000 runes, verb no vacío), ordena por fecha (mediodía UTC) y por cada práctica: `practice.NewPractice(..., fecha)` → `Save`, anonimiza (A8, reusa `services.Anonymize*`) + `LLMExtractor.Extract` (reusa `llm.NewOpenAIExtractor`, `temperature=0`), `analysis.NewAnalysis(..., fecha)` → `Complete` → `Save`, y `StartAnalysis`+`MarkCompleted` con la fecha real. Flags: `-input`, `-reset` (TRUNCATE de las 6 tablas de datos, conserva auditoría), `-validate` (dry-run), `-attempts` (reintentos) y `-timeout`.
- **Reintentos** (`-attempts`, default 3): gpt-4o-mini a `temperature=0` emite *ocasionalmente* un fragmento que falla la validación local estricta (p. ej. `es_contrast`/`exception` vacíos); el retry con backoff de 3s lo resuelve. De los 16, 2 fallaron transitoriamente y pasaron al reintentar.
- Tras importar, `go run ./cmd/provisioner refresh-aggregates` reconstruyó `error_metrics` (**18 métricos**, 6 códigos distintos; `last_seen_at` = 2026-09-15). `progress` (on-read) refleja 2 semanas históricas.

**Verificación (endpoint real, Postgres dev + LLM real)**:
- `GET /analytics/error-patterns?window=week` → `tense_agreement` 47, `infinitive_conjugation` 22, `lexical_choice` 16, `word_order` 7, `pronoun_possession` 2, `preposition_infinitive` 1.
- `GET /analytics/progress?window=week` → 2 puntos (semana del 07-09 y del 14-09).
- `GET /practices?limit=100` → `total: 16`.
- Estado DB: 16 `completed`, 0 `failed`, 16 `analyses`.

**Decisiones / notas**:
- **No se reusa `AnalysisService`**: usa `now()` interno y emite eventos del outbox; el import back-datea `created_at`/`updated_at`, así que ensambla `Practice`/`Analysis` directo vía dominio + repos (A2, A4).
- **Feedback regenerado, no migrado**: solo había textos (source/draft/reglas); el análisis se genera con el prompt/esquema actual (los fragments anonimizan nombres/emails como en la app).
- **`import.json` gitignored**: contiene el historial de estudio real del dueño (A9/A8); no se commitea.
- **Matiz de `accuracy`**: puede verse en 0% cuando `error_count > total_fragments` (un fragmento puede acumular 2-3 patrones de error); es la fórmula previa (1 − errores/fragmentos con clamp a 0), no un bug del import.

**Bloqueos**: ninguno.

**Próximo paso**: el perfil real ya alimenta el tutor; sugerido arrancar el uso real (`POST /study-sessions`), o el opcional `GET /v1/study-sessions` (historial) / deuda `SD-3`.

---

## 2026-09-22 — Cierre del `9.7` (analytics del quiz) + Gate de Fase 7 en verde

**Estado**: el `9.7` (diferido por diseño desde Fase 9) queda **cerrado** con una **métrica separada**, y el *Gate de salida* de Fase 7 se marca en **verde** tras el primer run real de CI en GitHub sin errores (confirmado por el humano). Contrato `3.4.0` (`GET /v1/analytics/quiz`). `go build/vet`, `go test -race -count=1 ./...` (Tier 1+2) y `go test -race -count=1 -tags=integration ./...` (Tier 3 + e2e) en verde; `pnpm generate` idempotente, `pnpm lint` (contrato válido, 7 warnings), `pnpm test` (frontend **99**), `pnpm typecheck`, `pnpm build` y `pnpm test:e2e` en verde; `gofmt -l` limpio.

**Hecho (`9.7`)**:
- **Métrica separada** (decisión del diferido): ledger append-only `quiz_attempts` (migración `000006`) que **no** entra en `refresh-aggregates` (que reconstruye `error_metrics` solo desde `analyses`), evitando la inconsistencia señalada en el diferido.
- Dominio `analytics.QuizAttempt` (append-only, ID UUID v7) + `QuizStats`/`BuildQuizStats` (puro, `accuracy = correct/total`, vacío ⇒ 1) con 100 % de cobertura nueva.
- Puerto `storage.QuizAttemptRepository` (`Append`/`ListByUser`) + adaptador `PostgresQuizAttemptRepository` + mock (Makefile).
- `QuizService.Evaluate` registra el intento (acierto/fallo) tras la evaluación; `QuizService.Stats` deriva `QuizStats` on-read. `NewQuizService` gana el puerto `attempts` (A2); DI con `newQuizAttemptRepository`.
- Handler `GetQuizStats` (`GET /analytics/quiz`) + `quizStatsToWire`; `analytics_service.go` sin cambios.
- **Privacidad (A9)**: `DeletionRepository.Execute` borra `quiz_attempts` por `user_id`; el test `TestPrivacy_RawDataSeparatedFromAnalytics` lo cubre.
- Frontend: `useQuizStats` + sección "Quiz" en `analytics-dashboard.tsx` (aciertos/aciertos-totales/precisión, F11) + test.

**Verificación del endpoint** (manual, contra el Postgres de dev `langlint-postgres` + LLM real):
- `GET /analytics/quiz` → `200 {"accuracy":1,"correct_attempts":0,"total_attempts":0}`.
- `POST /study-sessions?window=week` → sesión real generada por gpt-4o-mini (teoría + 3 trampas + 5 ejercicios) y persistida en `study_sessions` (1 fila). Las migraciones `000005`/`000006` se aplicaron a la base de dev al arrancar.

**Decisiones**:
- **Registro directo (sin outbox)**: el intento es un ledger append-only single-user sin consumidor cross-context; el fallo al persistir se propaga (no se pierde silenciosamente el acierto/fallo).
- **Sin `code` por intento**: el modelo no informa a qué `ErrorPattern` ancla la pregunta (el fragmento puede tener varios), así que la métrica es agregada por usuario (aciertos/fallos totales), no por patrón.
- **`quiz_attempts` sin FK a `practices`** (documentado): los intentos sobreviven al soft-delete de la práctica (como `access_events`) y solo los purga el olvido.
- **`GET /analytics/quiz` sin `window`**: métrica acumulada (la serie temporal no aporta aquí); si se pide, se añade bucketing como en `progress`.

**Bloqueos**: ninguno. (El contrato pasa de 6 a 7 warnings de redocly por el nuevo endpoint sin 4xx — no bloqueante.)

**Próximo paso**: opcional — `GET /v1/study-sessions` (historial) o abordar la deuda de seguridad `SD-3`.

---

## 2026-09-22 — Fase 8.4: integración sin rupturas (8.4.1–8.4.3) — Gate de Fase 8

**Estado**: bloque 8.4 completado y **Gate de Fase 8 en verde**. El tutor adaptativo queda integrado end-to-end sin romper ninguno de los 4 bounded contexts del MVP: endpoint `POST /v1/study-sessions` (A12, contrato `3.3.0`), `StudySessionService` + repositorio/migración `study_sessions` + mocks, y feature frontend `study-session`. `go build/vet`, `go test -race -count=1 ./...` (Tier 1+2) y `go test -race -count=1 -tags=integration ./...` (Tier 3 + e2e HTTP) en verde; `pnpm generate` idempotente, `pnpm lint` (contrato válido, 6 warnings conocidos), `pnpm test` (frontend **98**), `pnpm typecheck`, `pnpm build` y `pnpm test:e2e` en verde; `gofmt -l` limpio.

**Hecho**:
- `8.4.1` **Cero rupturas**: `git diff --stat` sobre `internal/domain/{identity,practice,analysis,analytics}` vacío. El tutor se integra solo por las capas api/services/handlers/adapters/migrations/contracts/frontend; el dominio `tutor/` no se toca.
- `8.4.2` A12: `api.yaml` añade el tag `tutor` y `POST /study-sessions` (operationId `create_study_session`, param opcional `window` reutilizando `WindowQuery`, `201 StudySession` · `409 invalid_state` · `503`), con schemas `StudySession`, `WeaknessEntry`, `StudySessionTrap`, `StudySessionExercise`, `StudySessionStatus` y `ExerciseKind`. Bump `3.2.0 → 3.3.0` (aditivo) + `CHANGELOG.md` + `package.json`. Migración `000005_study_sessions.sql` (JSONB para `profile`/`traps`/`exercises`). Puerto `storage.StudySessionRepository` (solo `Save`) + adaptador `PostgresStudySessionRepository` + mocks (`StudySessionRepository`, `StudySessionGenerator`) y `Makefile`.
- `StudySessionService.Generate(userID, window)`: lee `ErrorMetricRepository.ListByUser`, construye el `tutor.WeaknessProfile`, llama al `generator.Generate`, ensambla con `tutor.NewStudySession` y persiste con `Save`. Sin UoW ni outbox (guardado único; el LLM va fuera de transacción). Handler `CreateStudySession` + `studySessionToWire` + DI (`newStudySessionRepository`, `newStudySessionGenerator`, `NewStudySessionService`) y `NewServer` ampliado.
- Tests: service (perfil+default severidad+guardado, caso vacío → `InvalidStateError`, error del generador), handler (`201`/`409`/`422`), repositorio Tier 3 (`Save`/upsert) y e2e HTTP (`[6]` paso `POST /study-sessions` con `fakeStudySessionGenerator`).
- `8.4.3` Frontend: `lib/query/study-sessions.ts` (`useCreateStudySession`), feature `features/study-session/` (selector de ventana + render de teoría/trampas/ejercicios, tipos solo de `gen.ts`, F11), ruta `app/study-session/page.tsx` + enlace en `app/page.tsx`, y 3 tests RTL con accesibilidad.

**Decisiones** (confirmadas por el humano):
- **Severidad default `moderate`**: `error_metrics` no materializa severidad (solo vive en el payload de `WeaknessDetected`); tocarla implicaría cambiar el BC `analytics` (prohibido por 8.4.1). El service asigna `moderate` neutro y lo documenta (`defaultWeaknessSeverity`).
- **Todas las métricas `count>0`** alimentan el perfil (no se reaplica `WeaknessThreshold`): el umbral gobierna la notificación proactiva; la generación es on-demand sobre el historial.
- **`POST /study-sessions?window=week`** sin body; sin debilidades → `409 invalid_state` (reusa `domain.InvalidStateError`).
- **Solo `Save`** en el repositorio (sin listado ni `GET`): el endpoint devuelve la sesión creada; el historial queda para una iteración futura si se pide.

**Verificación** (local):
- `go build ./...` · `go vet ./...` → OK; `go test -race -count=1 ./...` → OK; `go test -race -count=1 -tags=integration ./...` → OK (e2e incluido, 62s).
- Cobertura: `tutor` **100 %**, services **94.8 %** (≥90 %); adaptador `llm` ya en 94.6 %.
- `pnpm generate` idempotente (hashes md5 estables); `check-version.sh` → `3.3.0`; `redocly lint` válido (6 warnings conocidos, sin nuevos).
- `pnpm test` (frontend 98) · `pnpm typecheck --filter=frontend` · `pnpm lint` · `pnpm build` · `pnpm test:e2e` (2) → OK.

**Bloqueos**: ninguno. El *Gate de Fase 7* sigue pendiente del primer run real de CI en GitHub (acción humana, ajeno a la Fase 8).

**Próximo paso**: opcional — listar el historial de sesiones (`GET /v1/study-sessions`) o cerrar el `9.7` (analytics del quiz). La Fase 8 queda completa.

---

## 2026-09-22 — Fase 8.3: motor de generación de sesión (8.3.1–8.3.3)

**Estado**: bloque 8.3 completado. Motor de generación de sesiones de estudio con IA (`ports.StudySessionGenerator` + `OpenAIStudySessionGenerator` con Structured Outputs estricto, `temperature=0`, manejo de truncado y validación local). **Solo motor**: sin DI, sin mocks, sin persistencia ni endpoint (8.4). Sin cambio de wire (A12). `go build/vet`, `go test -race -count=1 ./...` (Tier 1+2) y `go test -race -count=1 -tags=integration ./...` (Tier 3) en verde; `gofmt -l` limpio.

**Hecho**:
- `8.3.1` `ports.StudySessionGenerator` (`Generate(ctx, StudySessionRequest) (StudySessionContent, error)`) reusando el patrón del `LLMExtractor`/`TutorQuestioner`; `OpenAIStudySessionGenerator` (adaptador) con Structured Outputs estricto + `temperature=0` (greedy) + `finish_reason=length → *domain.LLMOutputTruncatedError`.
- `8.3.2` Contenido: schema estricto `{theory, traps:[{code,description}], exercises:[{kind,prompt,answer}]}`; prompt que pide teoría resumida (español) + exactamente 3 trampas comunes + 5 ejercicios personalizados; los enums `code` (reutiliza `errorPatternCodeEnum()`) y `kind` (`open|fill`) derivan del dominio.
- `8.3.3` Validación local (`buildSessionContent`): theory no vacío; cada trap vía `tutor.NewTrap`, cada ejercicio vía `tutor.NewExercise`; violación → `invalidOutput()` (`*domain.LLMUnavailableError`, A5). **Anonimización (A8) por construcción**: la entrada es el `WeaknessProfile` (códigos/severidades/recuentos agregados, sin texto crudo), así que no hay PII que enviar; blindado con test de prompt que verifica solo códigos+severidades+recuentos.

**Decisiones**:
- **Solo motor (confirmado por el humano)**: 8.3 entrega el port + adaptador + schema + validación, testeado con proveedor fake (`httptest`); el consumo real (endpoint `POST /v1/study-sessions`, persistencia y frontend) llega en 8.4.
- **`StudySessionContent` como DTO en `ports/`** (no en `domain/tutor`): el adaptador devuelve solo el contenido generado (theory/traps/exercises); el `StudySession` agregado lo ensamblará el service de 8.4 con `tutor.NewStudySession` (id, user, status, timestamps).
- **Sin hard-enforce de "exactamente 3/5" en código**: el prompt lo instruye (como "at most three entries" del extractor); la validación local solo exige no-vacío (invariante del dominio).
- **A8 por construcción**: el tutor consume agregados (PRODUCT_DOMAIN §12.1), no datos crudos, así que anonimizar es trivial (no hay texto libre). Documentado y testado.

**Verificación** (local):
- `go build ./...` · `go vet ./...` → OK.
- `go test -race -count=1 ./...` → OK; `go test -race -count=1 -tags=integration ./...` → OK.
- Cobertura adaptador `llm` **94.6 %** (≥70 %); `gofmt -l` limpio.
- Sin `pnpm generate` (sin cambio de wire); sin mocks nuevos (ningún service consume el port todavía).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 8.4 — integración sin rupturas: endpoint `POST /v1/study-sessions` (A12, primero `api.yaml`), `StudySessionService` (lee el perfil agregado, genera con el motor y persiste), repositorio + migración `study_sessions`, mocks del port y frontend (`study-session`). Después cerrar el Gate de Fase 8.

---

## 2026-09-22 — Fase 8.2: detección de debilidades (8.2.1–8.2.3)

**Estado**: bloque 8.2 completado. `analytics/` emite `WeaknessDetected` vía outbox cuando la frecuencia de un patrón cruza el umbral; `tutor/` se suscribe como `EventHandler` y construye un `tutor.WeaknessProfile` consultando los agregados. Sin cambio de wire (A12; los eventos de dominio son Go-only) y sin tocar el dispatcher/relay existentes. `go build/vet`, `go test -race -count=1 ./...` (Tier 1+2) y `go test -race -count=1 -tags=integration ./...` (Tier 3, incluido el e2e HTTP) en verde; `gofmt -l` limpio.

**Hecho**:
- **`Window` promovido a la raíz (alias)**: `domain.Window` (canónico: `type Window string` + `WindowDay/Week/Month` + `IsValid`/`String`) y `analytics` lo re-exporta como `type Window = domain.Window` + constantes + `AllWindows`. Cero cambios en los ~17 ficheros/mocks que usaban `analytics.Window`; sin regenerar mocks. Mismo criterio que `ErrorPattern` (raíz, compartido por dos contexts).
- `8.2.1` `domain.WeaknessDetected` (`analytics.weakness_detected`, payload `UserID`+`[]ErrorPattern`+`Window`+`Version`) registrado en `NewEvent`; tests de `EventName`/snake_case. Emisión desde `AnalysisCompletedHandler` (ahora con `outbox portsevents.Outbox`): tras el upsert por ventana, `emitWeakness` usa `analytics.DetectWeakness` y `outbox.Append` (post-commit, sin transacción → el pool; no viola AP7). Idempotencia preservada por el guard `status != analyzing`.
- `8.2.3` `analytics.DetectWeakness(freq map[code]int, threshold int)` (puro, orden determinista) + `const WeaknessThreshold = 5`. Umbral sobre el `Count` materializado (frecuencia); el "5 prácticas" del checklist es ilustrativo ("p. ej.").
- `8.2.2` `WeaknessDetectedHandler` (en `api/services/event_handlers/`): al recibir el evento consulta `ErrorMetricRepository.ListByUser` y `buildWeaknessProfile` construye un `tutor.WeaknessProfile` validado (código + severidad del evento + `Count`/`LastSeenAt` del agregado). Sin persistencia: la generación de sesión es 8.3. Suscripción en `registerSubscriptions` y en el e2e HTTP (paso end-to-end outbox→relay→dispatcher→handler).
- Severidad del evento = máxima vista en el análisis disparador (`severityByCode` + `severityRank`), porque `error_metrics` no materializa severidad.

**Decisiones**:
- **Alias `type Window = domain.Window`** (confirmado por el humano) en lugar del refactor completo: evita regenerar mocks y tocar 17 ficheros, mantiene `domain.Window` canónico y `analytics.Window` compatible.
- **Handler que construye el perfil consultando métricas** (confirmado), sin persistir: el evento es la señal; el perfil se arma con los agregados (PRODUCT_DOMAIN §12.1). La sesión/persistencia llega en 8.3.
- **`refresh-aggregates` (provisioner) no emite `WeaknessDetected`**: la detección es incremental (solo el camino `AnalysisCompleted`); la reconciliación reconstruye `error_metrics` sin disparar el tutor. Límite documentado de 8.2.

**Verificación** (local):
- `go build ./...` · `go vet ./...` → OK.
- `go test -race -count=1 ./...` → OK; `go test -race -count=1 -tags=integration ./...` → OK (e2e HTTP incluido).
- Cobertura del **nuevo** dominio: `tutor` **100 %**, `analytics.DetectWeakness` 100 %, `domain.Window`/`WeaknessDetected` 100 %; services `event_handlers` ≥90 % (93.2 %). `gofmt -l` limpio.
- Sin `pnpm generate` (sin cambio de contrato wire).

**Bloqueos**: ninguno. (Gaps de cobertura **preexistentes** y ajenos a 8.2: `LLMOutputTruncatedError.Error()` y el branch de error de `BuildProgressSeries`.)

**Próximo paso**: Fase 8.3 — generación de la sesión con IA (Structured Outputs, reusando `LLMExtractor`), contenido teoría + 3 trampas + 5 ejercicios, anonimización previa al LLM (A8) y validación contra schema. Después 8.4 (integración sin rupturas + endpoint `POST /v1/study-sessions` en `api.yaml`).

---

## 2026-09-22 — Fase 8.1: bounded context `tutor/` (8.1.1–8.1.4)

**Estado**: primer bloque de la Fase 8 (tutor adaptativo, post-MVP) completado: el **quinto** bounded context `internal/domain/tutor/` con agregado y value objects, dominio puro y 100 % de cobertura. Sin cambios de wire (A12) ni de los 4 contexts existentes (cero rupturas). `go build/vet`, `go test -race -count=1 ./...` (Tier 1+2) y `go test -cover ./internal/domain/tutor/` → **100 %** en verde; `gofmt -l` limpio.

**Hecho**:
- `8.1.1` Paquete `internal/domain/tutor/` con `doc.go` documentando el aislamiento A3 (solo stdlib + `domain` raíz).
- `8.1.2` Agregado `StudySession` (`ID`, `UserID`, `WeaknessProfile`, `Theory`, `Traps []Trap`, `Exercises []Exercise`, `Status`, `CreatedAt`) con constructor `NewStudySession` (→ `generated`) y transiciones `Start()` (→ `active`) y `Complete()` (→ `completed`); value objects `Trap` (`Code`+`Description`) y `Exercise` (`Kind open|fill`+`Prompt`+`Answer`), y enum `StudySessionStatus` (`generated`/`active`/`completed`).
- `8.1.3` Value object `WeaknessProfile` (agrupación de `ErrorPattern` históricos) con `WeaknessEntry` (`Code`, `Severity`, `Count`, `LastSeenAt`); constructor valida y ordena por frecuencia descendente; método `Weakest()` (mayor frecuencia) que anticipa la regla de detección de 8.2.3.
- `8.1.4` Verificado por diseño de tipos: `tutor/` importa **solo** stdlib + `domain` raíz (nunca `practice`/`analysis`/`analytics`), consumiendo agregados (`ErrorPatternCode` + frecuencia), no datos crudos. Comprobación estática con `grep` sobre los imports del paquete.

**Decisiones**:
- **`WeaknessProfile` agnóstico de ventana**: `Window` vive en `analytics/` y moverlo a la raíz (`domain`) para poder referenciarlo desde `tutor/` habría sido un refactor que toca analytics + wire. El perfil agrupa frecuencia histórica plana; el `Window` aparecerá recién en el payload del evento `WeaknessDetected` (8.2), donde se resolverá su representación sin romper A3.
- **Status con ciclo de vida** (`generated → active → completed`): confirmado por el humano (opción "Completos + StudySessionStatus").
- **`ExerciseKind` espeja el quiz** (`open`/`fill`, sin `mcq`): coherente con el principio de producción activa de la Fase 9; son tipos propios de `tutor/` por A3 (sin reimportar `analysis.QuizQuestion`).
- **Validación de entradas en `WeaknessProfile`** reutilizada desde `WeaknessEntry.validate()` para que literales de struct no puedan saltarse los invariantes; consistente con el patrón del resto del dominio.

**Verificación** (local):
- `go build ./...` · `go vet ./...` → OK.
- `go test -race -count=1 ./internal/domain/tutor/` → OK; `go test -cover ./internal/domain/tutor/` → **100.0 %**.
- `go test -race -count=1 ./...` → OK (regresión total); `gofmt -l internal/domain/tutor/` limpio.
- Aislamiento A3: `grep` de imports del paquete solo muestra stdlib + `domain` raíz.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 8.2 — detección de debilidades (evento `WeaknessDetected` vía outbox desde `analytics/`, suscripción de `tutor/` como `EventHandler`, regla de umbral de frecuencia 8.2.3). Queda decidir la representación de `Window` en el payload del evento sin romper A3.

---

## 2026-09-21 — Cierre de gaps pre-post-MVP: GAP-1/BUG-003, BUG-002 y BUG-001

**Estado**: los tres bugs abiertos de `docs/bugs/` quedan **cerrados** y el único gap contrato↔dominio funcional (`GAP-1`) también. Contratos `3.1.0` (progress) y `3.2.0` (`llm_output_truncated`). `go build/vet/test -race` (Tier 1+2+3), `pnpm generate` idempotente, `pnpm lint` (contrato incluido, 6 warnings conocidos), `pnpm test` (frontend 95), `pnpm typecheck`, `pnpm build` y `pnpm test:e2e` en verde. Queda como único bloqueo del *Gate de Fase 7* el primer run real de CI en GitHub (acción humana).

**Hecho**:
- **GAP-1 / BUG-003 — `GET /analytics/progress`**: la serie se deriva **on-read** de las `analyses` completadas (verdad append-only), sin tabla materializada: puerto `storage.ProgressRepository` (`ListSamplesByUser`), adaptador `PostgresProgressRepository` y la función pura `analytics.BuildProgressSeries` + `BucketPeriod` (día / semana ISO / mes). El handler responde `200 ProgressSeries`; el frontend deja de mostrar el placeholder. Contrato `3.1.0`: se retira el `501` y el componente `NotImplemented`.
- **BUG-002 — `finish_reason=length`**: el adaptador devuelve `*domain.LLMOutputTruncatedError` (en vez de `llm_unavailable`); el `AnalysisService` registra `reason = "llm output truncated"`. Contrato `3.2.0`: `llm_output_truncated` en el enum `ErrorResponse.code`; mapeo HTTP `503`.
- **BUG-001 — run-ons**: `splitSegments`/`splitRunOns` en `prompt.go` subdividen las frases largas (≥25 palabras) por sub-cláusulas deterministas (coma + conjunción/relativo, o `;`) con **invariante de cobertura** (`reconstructs`); si no se cumple, se conserva la frase entera. El adaptador llama al modelo por sub-cláusula.
- Tests nuevos: dominio (`BucketPeriod`, `BuildProgressSeries`, `ProgressMetric.PeriodStart`), service (`ProgressSeries`), handler (`200`/`422`, mapeo del nuevo error), adapter (`finish_reason=length`, run-on multi-llamada) y Tier 3 (`progress_repository_integration_test.go` + paso `[4b]` del e2e HTTP).

**Decisiones**:
- **Progress on-read, no materializado**: el plan de cierre admitía "job o consulta"; con un único usuario, derivar de la fuente de verdad evita migración, tabla y reconciliación (el plan original de materializar en `refresh-aggregates` habría añadido drift sin beneficio). `ProgressMetric` gana `PeriodStart` y se agrupa por periodo.
- **`llm_output_truncated` → HTTP `503`**: es una condición transitoria del proveedor (tope de salida), reintentable tras reducir el trabajo por llamada; se distingue de `llm_unavailable` por el `code`.
- **Run-ons con umbral y validadas**: solo se subdividen frases ≥25 palabras y solo si las piezas reconstruyen la frase; en caso contrario se mantiene el comportamiento anterior (sin regresión de cobertura).

**Verificación** (local):
- `go build ./...` · `go vet ./...` (+ `-tags=integration`) · `go test -race -count=1 ./...` → OK; `gofmt -l` limpio.
- `go test -race -count=1 -tags=integration ./internal/api/adapters/postgres/repositories/ ./test/e2e/` → OK.
- `pnpm generate` idempotente; `check-version.sh` → `3.2.0`; `pnpm --filter @langlint/contracts lint` → válido.
- `pnpm --filter frontend test` (95) · `pnpm --filter frontend typecheck` · `pnpm --filter frontend lint` → OK.

**Bloqueos**: el *Gate de Fase 7* sigue requiriendo el primer run real de CI en GitHub (secrets `TURBO_*` y var `STAGING_API_URL`).

**Próximo paso**: cerrar el Gate de Fase 7 con el primer run de CI; después, post-MVP (tutor adaptativo) o el `9.7` diferido.

---

## 2026-09-21 — Fase 7.3: Documentación (7.3.1–7.3.4)

**Estado**: bloque Documentación completado; con él, **todos los items de Fase 7 (`7.1`–`7.3`) quedan hechos**. El *Gate de salida* de Fase 7 sigue pendiente únicamente del **primer run real de los workflows en GitHub** (no corren en local); `gosec`/`govulncheck` (0 hallazgos) y la documentación ya están cerrados. Sin cambios de código ni de contrato (A12).

**Hecho**:
- `7.3.1` `docs/ARCHITECTURE.md`: contrato arquitectónico global (monorepo, capas, bounded contexts, UoW/outbox, motor IA, frontend, CI/seguridad) con **§5.1 env vars** (exigido por MANIFEST_MONOREPO §8).
- `7.3.2` `docs/RUNBOOK.md` (10 escenarios; **Escenario 5 = Postgres caído**, referenciado por el manifiesto), `docs/PRODUCTION_ENV.md` (referencia de env vars por entry point), `docs/SECRET_ROTATION.md` (**§1.1 cuándo rotar** + procedimiento por secreto) y `docs/SECURITY_DEBT.md` (SD-1..SD-6). **Extra**: `docs/API_CONTRACT.md` (gaps contrato↔dominio, AP5), que el manifiesto referencia aunque el checklist no lo listaba.
- `7.3.3` `README.md` reescrito como **portafolio** (estado real, stack, arquitectura, métricas de rigor verificables, comandos) y `README.en.md` **alineado**.
- `7.3.4` `docs/bugs/` con índice y 3 bugs abiertos en formato anti-patrón de los manifiestos: BUG-001 (run-ons no subdivididos), BUG-002 (`finish_reason=length` → `llm_unavailable`), BUG-003 (`/analytics/progress` 501).

**Decisiones**:
- **`API_CONTRACT.md` sí se crea** (decisión del humano): el manifiesto lo referencia en AP5 para los "gaps conocidos"; se documentan GAP-1 (`/analytics/progress` 501) y GAP-2 (origen del email del export) con plan de cierre. Se corrige la referencia: es **AP5**, no AP-MR5.
- **Bugs en formato de manifiesto** (Qué pasó / Lección / Regla): reutiliza el lenguaje ya establecido para anti-patrones y evita inventar un formato nuevo.
- **`ARCHITECTURE.md` numera §5.1 para env vars** para cumplir la referencia exacta de MANIFEST_MONOREPO §8.
- **Métricas verificables, no marketing**: los números del README se midieron (`go test -cover` y Vitest), no se estimaron.

**Verificación** (local):
- Consistencia de **env vars**: todas las de `ARCHITECTURE.md`/`PRODUCTION_ENV.md` existen en `internal/shared/config/*.go` (o son prefijos/vars de CI).
- **Enlaces locales** de README(s), los 5 docs y `docs/bugs/*`: todos resuelven (script de comprobación; sin `MISS`).
- Estructuras exigidas por el manifiesto presentes: `ARCHITECTURE.md §5.1`, `RUNBOOK.md Escenario 5`, `SECRET_ROTATION.md §1.1`.
- Métricas reales: dominio **100 %** (5 paquetes), services **91.4–98.1 %**, frontend **95** tests Vitest (18 ficheros), e2e **2**.
- Sin cambios de código: `go build/vet/test` y `pnpm` no se ven afectados.

**Bloqueos**: ninguno. El cierre formal del *Gate de salida* de Fase 7 requiere el primer run de CI en GitHub (los workflows no se ejecutan en local).

**Próximo paso**: cerrar el Gate de Fase 7 con el primer run real de CI (o el `9.7` diferido / el tutor adaptativo post-MVP).

---

## 2026-09-21 — Fase 7.2: Seguridad (7.2.1–7.2.4)

**Estado**: bloque Seguridad completado. `7.2.1`–`7.2.4` implementados y verificados en local (gosec 0 hallazgos, govulncheck 0 vulnerabilidades, `security_audit.sh` limpio y probado contra una violación, tests Tier 1+2 en verde). Los workflows siguen sin ejecutarse en GitHub (no corre en local). El *Gate de salida* de Fase 7 solo espera la ejecución real y `7.3` (documentación).

**Hecho**:
- `7.2.1` `.github/workflows/security.yml`: `security_audit.sh` + `gosec` (pin `v2.29.0`, SARIF a Code Scanning, gate high/medium) + `govulncheck` (pin `v1.8.0`); triggers PR/push `main`/nightly/manual, `security-events: write`.
- `7.2.2` `ops/scripts/security_audit.sh`: auditoría estática sin red (secretos hardcodeados, guard de docs AP-MR9, consistencia `x-internal`), salida redactada y exit 0/1/2 como `pii_audit.sh` (que ya existía desde 6.3 y se deja intacto).
- `7.2.3` Verificado que el backend **no** expone `/docs`: no hay handler de docs, Scalar, Swagger ni `APP_API_DOCS_*` (el router solo monta el contrato). Se añade `TestGeneratedRouter_DocsEndpoints_NotFound` (404 en `/docs`, `/openapi.json`, `/swagger`, `/scalar`) y el guard estático.
- `7.2.4` El contrato no marca ningún endpoint `x-internal: true` (15 operaciones de producto/A9) y no hay render de docs: N/A por diseño. El guard de `security_audit.sh` exige un filtro en Go si en el futuro se marca alguno.

**Hallazgos y decisiones**:
- **`gosec` G115 (falso positivo)**: 6 conversiones `uint64 → byte` en `internal/domain/identifiers.go` (bytes del timestamp UUID v7). En vez de suprimir con `#nosec`, se enmascaró `& 0xff` (comportamiento idéntico, sin perder cobertura del scanner). gosec pasa de 6 a 0 hallazgos.
- **`govulncheck`: toolchain vulnerable**: con Go `1.26.2` (el que fija `.tool-versions`) aparecían **12 vulnerabilidades de la stdlib** (corregidas en `1.26.3`+ y siguientes). Se fija `toolchain go1.26.8` en `apps/backend/go.mod` y `.tool-versions` a `1.26.8` (último patch de la línea 1.26). Con `GOTOOLCHAIN=auto` el binario local descarga la versión segura; CI ya usaba `setup-go` (última 1.26.x). Resultado: 0 vulnerabilidades.
- **7.2.3/7.2.4 se cierran con verificación + guard, no implementando un renderer**: el proyecto nunca tuvo docs UI; montarla habría contradicho el objetivo de no exponerla. En su lugar se blinda la regla con test + auditoría estática.
- **`security_audit.sh` excluye `*_test.go` del guard de docs**: el test de 7.2.3 contiene las cadenas `/docs`/`/swagger`/`/scalar` y dispararía un falso positivo; los tests no montan rutas de producción.

**Verificación** (local, exit codes reales):
- `gosec -severity high -confidence medium ./...` → **0 issues** (102 ficheros).
- `govulncheck ./...` → `No vulnerabilities found.` (con `go1.26.8`).
- `bash ops/scripts/security_audit.sh` → limpio; con un `bad.go` con `/docs` + `sk-…` → exit 1 con valor redactado (probado y borrado).
- `printf '…' | bash ops/scripts/pii_audit.sh` → limpio.
- `go build ./...`, `go vet ./...`, `go test -race -count=1 ./...` → OK; `gofmt -l` limpio.
- YAML de `security.yml` validado; los 4 workflows parsean.

**Bloqueos / acciones humanas**: ninguna crítica. El upload de SARIF es `continue-on-error` para no bloquear si Code Scanning no está habilitado en el repo. Requiere el primer run real en GitHub para cerrar el gate.

**Próximo paso**: Fase 7.3 — Documentación (`ARCHITECTURE.md`, `RUNBOOK.md`, `PRODUCTION_ENV.md`, `SECRET_ROTATION.md`, `SECURITY_DEBT.md`, README de portafolio y `docs/bugs/`). Nota: `README.md` aún dice "Fase 1" (se reescribe en `7.3.3`).

---

## 2026-09-21 — Fase 7.1: CI/CD (7.1.1–7.1.5)

**Estado**: bloque CI/CD completado. `7.1.1`, `7.1.2`, `7.1.3`, `7.1.4` y `7.1.5` implementados. Verificación local en verde (YAML, generación idempotente, dry-runs de build); **los workflows no se han ejecutado** (GitHub Actions no corre en local) y las imágenes no se han construido (falta `buildx`), así que el *Gate de salida* de Fase 7 queda a expensas del primer run real en GitHub.

**Hecho**:
- `7.1.1` `.github/workflows/ci.yml`: `detect-changes` (`dorny/paths-filter@v3`) + jobs `contracts` (redocly), `backend` (`build lint test`) y `frontend` (`build lint typecheck test`); triggers PR, push `main`, nightly (`0 3 * * *`) y manual.
- `7.1.5` El job `backend` corre siempre en nightly y ejecuta `test-integration` (Tier 3) cuando el evento es `schedule`, `push` o el PR lleva el label `ready-for-release` (fiel a §6.5). Se implementa dentro de `ci.yml` (no un workflow aparte).
- `7.1.2` `.github/workflows/contracts.yml`: en cambios de `apps/contracts/**` regenera Go+TS (`pnpm generate`), falla por drift (`git diff --exit-code` sobre `gen_*.go`/`gen.ts`) y valida `info.version == package.json` con `apps/contracts/scripts/check-version.sh`.
- `7.1.4` Remote cache **self-hosted** (decisión del humano en lugar de Vercel): servicio `turbo-cache` (`ducktors/turborepo-remote-cache`, storage local en volumen, puerto `3300`) en `ops/docker/docker-compose.yml`; `TURBO_API`/`TURBO_TEAM`/`TURBO_TOKEN` cablados desde secrets en los tres jobs de `ci.yml`; guía en `ops/docker/README.md` (KPI cache hit >80%).
- `7.1.3` `apps/backend/Dockerfile` (multi-stage `golang:1.26` → `distroless/static-debian12:nonroot`; construye `api`/`provisioner`/`migrate`; migraciones embebidas con `go:embed`) + `.dockerignore`; `apps/frontend/Dockerfile` (`node:22-alpine` + `turbo prune` + Next `output: standalone`) + `.dockerignore` raíz; `.github/workflows/deploy-staging.yml` (build+push a GHCR con tags `staging`/`sha`).

**Decisiones**:
- **Fix del drift de versión del contrato**: `api.yaml` `info.version` estaba en `2.1.0` mientras `package.json`/`CHANGELOG` en `3.0.0` (el commit de arrays no lo subió). Se alinea a `3.0.0` y se convierte en invariante verificada por CI (A12/AP-MR6).
- **`contracts` en `ci.yml` hace solo lint del spec**; la regeneración + drift + versión viven en `contracts.yml` para no duplicar responsabilidades.
- **Remote cache self-hosted con storage local** (volumen) para dev; en staging migra a S3. Caveat documentado: los runners *hosted* solo alcanzan la caché si el endpoint es público; si no, se usan self-hosted runners (o la caché remota se omite sin fallar).
- **Contextos Docker por imagen**: backend con `context=apps/backend` (`apps/backend/.dockerignore`); frontend con `context=.` (raíz) porque `turbo prune` y el workspace pnpm lo exigen (`.dockerignore` raíz). `next.config.js` añade `output: standalone` y `public/` lleva `.gitkeep` (estaba vacío y sin trackear, y el `COPY` del `public` fallaría en un checkout limpio).
- **`pnpm/action-setup@v4` sin `version`**: toma `packageManager` de `package.json` (evita drift y el error de "múltiples versiones").

**Verificación** (local):
- `python3 -c "yaml.safe_load(...)"` → OK en `ci.yml`, `contracts.yml`, `deploy-staging.yml`.
- `pnpm generate` idempotente (`git status` sin cambios en `gen_*.go`/`gen.ts`); `check-version.sh` → `3.0.0`; `pnpm turbo run lint --filter=@langlint/contracts` → válido (6 warnings conocidos).
- Backend: `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"` de `cmd/{api,provisioner,migrate}` → binarios **estáticamente enlazados** (aptos para distroless static).
- Frontend: `pnpm --filter frontend build` con `output: standalone` → OK; `.next/standalone/apps/frontend/server.js` existe (coincide con el `CMD` de la imagen).
- `docker compose -f ops/docker/docker-compose.yml config` → OK.
- Pendiente (no reproducible en local): run real de los workflows en GitHub y `docker build` de las imágenes (`buildx` no instalado).

**Bloqueos / acciones humanas**:
- Crear los secrets `TURBO_API`, `TURBO_TEAM`, `TURBO_TOKEN` (y la variable `STAGING_API_URL`) en el repo; sin ellos la caché remota se omite (no falla).
- Endurecer el endpoint de caché (TLS) o usar self-hosted runners para que el CI lo alcance.

**Próximo paso**: Fase 7.2 — Seguridad (`security.yml` con `gosec`/`govulncheck`, `ops/scripts/security_audit.sh`, `/docs` no expuesto en prod y `x-internal: true` filtrado). Nota: `ops/scripts/pii_audit.sh` ya existe y `README.md` aún está desactualizado (Fase 1), ambos listados como pendientes en `7.2.2`/`7.3.3`: reconciliar antes de ejecutar.

---

## 2026-09-20 — Fix: alineación español↔borrador (extracción por frase)

**Estado**: bug de alineación diagnosticado y corregido. `go build/vet/test -race` (+ `-tags=integration`), `pnpm test-integration --filter=backend`, `pnpm lint` (3/3), `pnpm test`, `pnpm typecheck --filter=frontend` y `pnpm build` (3/3) en verde. **Sin cambio de contrato** (A12): el wire ya admitía arrays.

**Síntoma**: en la vista de 3 columnas, la frase en español, el borrador y la corrección aparecían **desfasados** (el borrador barajado/duplicado y faltaba la primera frase).

**Diagnóstico** (volcando la respuesta cruda del modelo):
- El español y el borrador tienen **distinto número de frases** (el alumno fusiona/divides). El prompt pedía partir por cláusulas del *español*, pero el modelo **alineaba por índice**, no por significado: emparejaba `source[i]` con `draft[i+1]` y acababa emitiendo basura (`"{"`).
- Se probó además un prompt con **invariante de cobertura** + validación en código: el modelo lo incumplía (reordenaba/omitía), con ~2/5 de éxito. Se probó "un fragmento por frase, sin dividir": el modelo seguía saltándose la primera frase.
- Conclusión: **gpt-4o-mini no segmenta + alinea ES↔EN + copia verbatim + explica de forma fiable en una sola llamada** para textos largos.

**Hecho** (contenido en el adapter, sin tocar puerto ni service):
- `prompt.go`: se pasa a un **prompt por frase** (`buildSentencePrompt`) y `SplitSentences` segmenta el borrador de forma determinista en `. ! ?`.
- `openai_extractor.go`: `Extract` **llama al modelo una vez por frase** del borrador y ensambla los fragmentos; el `user_draft` lo fija el backend a partir de su propia segmentación, de modo que la cobertura y el orden del borrador quedan **garantizados por construcción** (más fuerte que una validación posterior).
- `prompt_test.go` / `openai_extractor_test.go`: reescritos (una llamada por frase, N frases ⇒ N llamadas, borrador vacío ⇒ 0 llamadas, temperatura 0, esquema estricto).

**Verificación (texto real de ~2 KB, 6 frases en el borrador)**:
- 3/3 corridas OK, ~30s cada una, 6 fragmentos, 37–41 entradas, con alineación correcta (cada frase del borrador con su fuente).
- Antes: 0/5 fiables (desfase o truncado a 16 384 tokens).

**Decisiones**:
- **Chunking por frase** (acordado) en vez de una sola llamada: acota la salida de cada llamada, elimina el truncado y hace imposible el desfase.
- **El `user_draft` lo fija el backend**, no el modelo: garantiza el invariante sin depender de que el modelo copie verbatim.
- **Límite conocido**: una frase run-on muy larga (sin `. ! ?` internos) va en un único fragmento; partirla en sub-cláusulas de forma fiable queda como mejora futura.

**Próximo paso**: Fase 7 (Hardening) o `9.7`. Candidata de hardening: subdividir run-ons largos por sub-cláusulas deterministas validadas contra la frase original.

---

## 2026-09-20 — Fix: el análisis fallaba por desbordar el límite de salida del LLM

**Estado**: fallo diagnosticado y corregido. `go build/vet/test -race` (+ `-tags=integration`), `pnpm test-integration --filter=backend`, `pnpm lint` (3/3), `pnpm test`, `pnpm typecheck --filter=frontend` y `pnpm build` (3/3) en verde. Sin cambio de contrato (A12).

**Síntoma**: al analizar una práctica, tras ~60s la frontend mostraba "Fallida: el análisis no pudo completarse".

**Diagnóstico** (con un probe temporal de `finish_reason`/usage, ya retirado):
1. Se reprodujo por la API: fallaba a los ~59s. `llmcheck` con el mismo input a veces tardaba 21s y a veces fallaba a los ~110s.
2. El probe demostró la causa: `finish_reason=length`, `completion_tokens=16384` (**el tope de salida de gpt-4o-mini**). El prompt exhaustivo hacía que el modelo se disparara hasta el límite, truncando el JSON → el adapter lo marcaba como `llm_unavailable`. Ocurría ~50% de las veces en textos de ~2 KB (el máximo que admite el formulario).
3. De paso se halló un bug: **`APP_LLM_TIMEOUT` no se inyectaba** (config cargaba el valor, pero `NewAnalysisService` fijaba 60s a fuego).

**Hecho**:
- **Timeout cableado**: `NewAnalysisService` recibe `services.LLMTimeout` (nuevo tipo inyectado por Fx desde `cfg.LLMTimeout`); default subido a **180s** (`config/server.go`, `.env`, `.env.example`, tests).
- **Prompt acotado** (`prompt.go`): máximo **3 entradas por categoría y fragmento**, campos de **una frase (≤20 palabras)**, `error_patterns` sigue siendo la lista completa y las secciones estructuradas explican solo lo más importante. Se sustituye la regla de "cubrir todos los errores" (que causaba el rebose) por una de priorización.

**Medición (`llmcheck`, texto de ~2 KB)**:
- Antes: `finish=length`, 16 384 tokens de salida, 112s → fallo.
- Después: `finish=stop`, 2 763–3 373 tokens, 19–26s → éxito (3/3).

**Decisiones**:
- **Acotar el prompt en vez de chunking** (acordado): una sola llamada acotada es suficiente y mucho más simple; el chunking multi-llamada queda como posible mejora.
- **`temperature=0` + tope**: el muestreo greedy no bastaba (la varianza era de 3k a 16k tokens); el tope es lo que garantiza el techo.
- **Los ~110s no eran el timeout**: alargarlo no arreglaba el truncamiento; sí era necesario cablear `APP_LLM_TIMEOUT` y subir el default para las respuestas legítimamente lentas.

**Verificación**:
- `go build ./...` · `go vet ./...` · `go test -race -count=1 ./...` → OK.
- `pnpm test-integration --filter=backend` · `pnpm lint` (3/3) · `pnpm test` · `pnpm typecheck --filter=frontend` · `pnpm build` (3/3) → OK.
- Manual: 3 corridas de `llmcheck` con el texto de referencia devuelven `finish=stop` y ~3k tokens (antes: `finish=length` a 16 384).

**Bloqueos**: ninguno.

**Nota operativa**: hay que **reiniciar `cmd/api`** para que tome el nuevo timeout y prompt; `APP_LLM_TIMEOUT` en `.env` queda en `180s`.

**Próximo paso**: Fase 7 (Hardening) o `9.7`. Candidato de hardening: detectar `finish_reason=length` y devolver un error específico (hoy se ve como `llm_unavailable`).

---

## 2026-09-20 — Exhaustividad del feedback del LLM (prompt + `temperature=0`)

**Estado**: mejora de calidad aplicada. Sin cambio de contrato (A12). `go build/vet/test -race` (+ `-tags=integration`) OK; `pnpm test-integration --filter=backend`, `pnpm lint` (3/3), `pnpm test`, `pnpm typecheck --filter=frontend` y `pnpm build` (3/3) en verde; `gofmt -l` limpio.

**Contexto**: tras activar los arrays (`3.0.0`), un análisis real seguía mostrando **1** verbo/1 léxico/1 gramática. Al medir con `cmd/llmcheck` se descubrió que gpt-4o-mini **no era determinista**: la misma frase devolvía a veces 1 entrada por categoría y a veces varias. La causa era la adherencia del modelo + muestreo con `temperature` por defecto (1.0), no el wire.

**Hecho**:
- `prompt.go`: fragmentación **por cláusula** (partir oraciones largas en conjunciones/relativos/puntuación), **cardinalidad simétrica** por sección (*"one entry for every …; never collapse several"*) y **regla de cobertura** (`error_patterns` ↔ secciones estructuradas).
- `openai_extractor.go`: `Temperature=0` (`extractionTemperature`) en la llamada de extracción, para muestreo greedy y consistente.
- Tests: `prompt_test.go` verifica que el system prompt incluye las reglas nuevas; `openai_extractor_test.go` verifica que la petición envía `temperature=0`. Sin cambio de contrato → `pnpm generate` no aplica.

**Medición (`cmd/llmcheck`, misma frase del usuario)** — `(verbs, lexical, grammar, patterns)`:
- Antes (temp por defecto): `(3,2,2,4)` y `(1,0,1,1)` → alta varianza.
- Después (temp 0): `(5,1,1,1)`, `(6,6,1,1)`, `(3,2,5,5)`. Mejor caso: 3 verbos (`quit`,`let`,`wants`), 2 léxicos, 5 gramáticas, 5 patrones.

**Decisiones**:
- **Prompt + `temperature=0`**: se ataca la causa (adherencia) y la varianza (greedy) a la vez.
- **Sin few-shot ni tope** (acordado): el prompt + temp bastan para el MVP, sin coste extra de tokens.
- **Límite conocido**: gpt-4o-mini sigue siendo imperfecto; la cobertura exacta patrón↔explicación no es determinista. Se documenta como caveat del MVP (candidato a endurecer en Hardening: evaluación con golden examples).

**Verificación**:
- `go build ./...` · `go vet ./...` · `go test -race -count=1 ./...` → OK.
- `pnpm test-integration --filter=backend` → OK (Tier 3).
- `pnpm lint` (3/3) · `pnpm test` · `pnpm typecheck --filter=frontend` · `pnpm build` (3/3) → OK.
- Manual: 3 corridas de `llmcheck` con la frase de referencia devuelven múltiples entradas por categoría (antes del cambio: 1/0/1/1 en una corrida).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 7 (Hardening) o `9.7`.

---

## 2026-09-20 — Fragmentos con múltiples explicaciones (contrato `3.0.0`)

**Estado**: mejora de calidad completada. `go build/vet/test -race` (+ `-tags=integration`) OK; `pnpm test-integration --filter=backend` OK; `pnpm lint` (3/3), `pnpm test` (backend + frontend **95**), `pnpm typecheck --filter=frontend`, `pnpm test:e2e` (2) y `pnpm build` (3/3) en verde; `pnpm generate` idempotente. **Cambio de contrato A12 BREAKING** (`apps/contracts` → `3.0.0`).

**Contexto**: al revisar un análisis real se observó que un `Fragment` puede contener **varios** verbos objetivo, aclaraciones léxicas y reglas gramaticales, pero el contrato forzaba exactamente uno de cada (objeto único). El prompt ya decía "target verb(s)", pero el JSON Schema de Structured Outputs cerraba la cardinalidad a 1, así que el modelo descartaba el resto (p. ej. `bet on`, `my brother`, orden de palabras en la misma frase).

**Hecho**:
- A12: `Fragment.target_verb_review`/`lexical_clarification`/`grammar_explanation` pasan de objeto a **array** y se renombran a `target_verb_reviews`/`lexical_clarifications`/`grammar_explanations`; los esquemas internos (`TargetVerbReview`…) no cambian. `pnpm generate` (Go + TS). Bump a `3.0.0` + `CHANGELOG.md`.
- Dominio `analysis.Fragment`: los tres campos pasan a slices; test nuevo de round-trip con 2 elementos por categoría.
- Adapter LLM: `schema.go` envuelve cada esquema en `arrayOf(...)`; `validate.go` itera cada elemento y **permite listas vacías** (se añade test); `prompt.go` pide "una entrada por problema distinto" y un array vacío cuando la categoría no aplica (elimina el relleno forzado "sin error léxico en este fragmento").
- `handlers/server.go`: conversión dominio→wire slice→slice.
- Frontend `fragment-diff.tsx`: renderiza una tarjeta por entrada (`Verbo objetivo (i/n)` solo cuando hay varias; array vacío → sin tarjeta); fixtures, unit (95) y e2e actualizados.
- `docs/PRODUCT_DOMAIN.md §5.2/§8.1` alineado con el nuevo esquema.

**Decisiones**:
- **Arrays para las tres categorías** (no solo léxico/gramática): el usuario pidió paridad y el ejemplo real tiene varios verbos.
- **Renombrado a plural** en el wire (`..._reviews`): autodocumenta la cardinalidad; BREAKING asumido y versionado.
- **Listas vacías permitidas** (como `error_patterns`): evita que el modelo invente relleno para cumplir el `required`.
- **Sin migración de datos**: `analyses.fragments` es JSONB derivado; las filas antiguas con claves singulares se leerán con listas vacías. Se regenera re-analizando; no se escribe una migración para datos descartables.

**Verificación**:
- `go build ./...` · `go vet ./...` · `go test -race -count=1 ./...` · `go test -race -count=1 -tags=integration ./...` → OK.
- `pnpm test-integration --filter=backend` → OK (Tier 3, incluido el e2e HTTP que ahora usa arrays).
- `pnpm lint` (3/3) · `pnpm test` (frontend 95) · `pnpm typecheck --filter=frontend` · `pnpm test:e2e` (2) · `pnpm build` (3/3) → OK.
- `pnpm generate` idempotente (hashes md5 de `gen_types.go`/`gen.ts` estables); `gofmt -l` limpio.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 7 (Hardening) o cerrar `9.7`. Pendiente de decisión de producto: `PRODUCT_DOMAIN §11` (DoD) y `/analytics/progress` (sigue `501`).

---

## 2026-09-19 — Fase 9: feedback profundo y práctica activa (9.1–9.6; 9.7 diferido)

**Estado**: bloque `9.1`–`9.6` completado; **Gate de Fase 9 en verde** para esos items. `go build/vet/test -race` (+ `-tags=integration`) OK; `pnpm test-integration --filter=backend`, `pnpm lint`, `pnpm test` (frontend 94), `pnpm typecheck --filter=frontend` y `pnpm build` OK; `pnpm generate` idempotente. Cambio de contrato A12: `2.0.0` (BREAKING, `Fragment` estructurado) + `2.1.0` (endpoints de quiz).

**Contexto**: el feedback de la IA salía vago ("bet debe ir seguido de on" sin porqué). Se acordó (debate con el humano) mejorar la profundidad del feedback y añadir práctica activa con IA, sin romper axiomas.

**Hecho**:
- `9.1` Rúbrica de explicación en `systemPrompt` (`prompt.go`): regla nombrada, por qué, construcción, contra-ejemplo, excepción, contraste ES→EN y alternativas. Verificado con `cmd/llmcheck -show-prompt` y una llamada real con el ejemplo "bet on".
- `9.2` A12: `Fragment` deja de tener 3 `string` y pasa a `TargetVerbReview`/`LexicalClarification`/`GrammarExplanation` (objetos). `api.yaml` + `pnpm generate` (Go y TS). Bump `apps/contracts` a 2.0.0 + `CHANGELOG.md`.
- `9.3` Dominio `analysis` (structs anidados), `schema.go` (JSON Schema anidado strict), `validate.go` (sub-campos obligatorios; `alternatives` puede ir vacía) y conversión dominio→wire en `handlers/server.go`.
- `9.4` Frontend: `fragment-diff.tsx` renderiza tarjetas etiquetadas (`dl`/`dt`/`dd`, `h3` bajo las columnas `h2`); fixtures y tests Vitest/e2e actualizados.
- `9.5` Límites de práctica: máx. 5 `TargetRules` y 2000 runes en `source_text`/`draft_text` (dominio, `ValidationError`), espejo en Zod y botón "Añadir regla" acotado.
- `9.6` Práctica activa con IA: puerto `TutorQuestioner` (tipos `open`/`fill`; `mcq` descartado por ser reconocimiento, no producción), adapter `OpenAITutorQuestioner` con Structured Outputs, `QuizService`, endpoints `POST /practices/{id}/quiz` y `/quiz/answer` (evaluación sin estado + follow-up socrático) e UI `practice-quiz.tsx` (mutaciones on-demand). Bump contracts a 2.1.0.

**Decisiones**:
- **Pseudonimizar/estructurar sin tocar los bounded contexts**: el adapter LLM sigue siendo un puerto; el dominio añade value objects puros.
- **Explicaciones estructuradas en el wire** (opcional en wire, obligatorio en el schema LLM): el frontend pinta tarjetas y el modelo no puede dejar campos vacíos (validación local).
- **Quiz sin estado**: el cliente reenvía la pregunta evaluada; evita una tabla/estado de conversación en el MVP.
- **`9.7` diferido con criterio**: incrementar `error_metrics` desde el quiz sería inconsistente porque `refresh-aggregates` la reconstruye solo desde `analyses` (el incremento se perdería). Requiere una fuente de verdad propia (tabla/evento de intentos) que entre en la reconciliación, o una métrica separada.

**Verificación**:
- `go build/vet/test -race ./...` (+ `-tags=integration`) → OK; `gofmt -l` limpio.
- `pnpm test-integration --filter=backend` → OK (incluye e2e del quiz).
- `pnpm lint` (3/3) · `pnpm test` (frontend 94) · `pnpm typecheck --filter=frontend` · `pnpm build` (3/3) → OK.
- `pnpm generate` idempotente (hashes estables).
- Manual: `go run ./cmd/llmcheck` con el ejemplo "bet on" devuelve la explicación estructurada completa.

**Bloqueos**: ninguno.

**Próximo paso**: retomar Fase 7 (Hardening) o cerrar `9.7` con una tabla/evento de intentos de quiz.

---

## 2026-09-19 — Ops: reconciliación de goose y CORS en dev (`:3001`)

**Estado**: nota operativa (incidencias de entorno local). Sin cambios de código ni de contrato (A12); con esto el backend arranca y el frontend recibe CORS.

**Hecho**:
- **Drift de goose**: la base de dev tenía `deletion_requests` y `access_events` creadas, pero `goose_db_version` solo registraba `000001`, así que `go run ./cmd/api` fallaba al arrancar con `42P07 relation "deletion_requests" already exists`. Se verificó que el esquema de esas tablas **coincidía** con las migraciones `000002`/`000003` y que `error_metrics.user_id` seguía en `uuid` (la `000004` no estaba aplicada). Se marcaron 2 y 3 como aplicadas en `goose_db_version` y se dejó que goose corriera solo la 4 (`goose: successfully migrated database to version: 4`). `error_metrics` estaba vacía: sin filas huérfanas. **No requiere cambio de código**; en una base limpia (o reseteando el volumen `langlint_pgdata`) aplica en orden sin intervención.
- **CORS en `:3001`**: el dev server de Next arrancó en `:3001`, pero el backend solo permitía `:3000` por defecto (`APP_CORS_ALLOWED_ORIGINS`), de ahí el `No 'Access-Control-Allow-Origin' header`. Se añadió `http://localhost:3000,http://localhost:3001` al `.env` local (gitignored). Verificado con preflight `OPTIONS` y `GET` (`Access-Control-Allow-Origin: http://localhost:3001`). El `404` de `/favicon.ico` y `Can't add file system: <illegal path>` son ruidos del navegador/extensión, ajenos al proyecto.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 7 (Hardening).

---

## 2026-09-19 — Fase 6.3: retención y pseudonimización — Gate de Fase 6 (6.3.1–6.3.3)

**Estado**: bloque `6.3` completado y **Gate de Fase 6 en verde**. `go build/vet/test -race` (Tier 1+2+3) OK; `pnpm test-integration --filter=backend` OK; `pnpm lint` (3/3), `pnpm test`, `pnpm typecheck --filter=frontend` y `pnpm build` (3/3) en verde. Sin cambio de wire (A12; `pnpm generate` idempotente).

**Hecho**:
- `6.3.1` `internal/shared/pseudonymizer/pseudonymizer.go`: `Pseudonymize(id) = hex(HMAC-SHA256(secret, id))`, stdlib puro; config `APP_PSEUDONYM_SECRET` (requerida) en `ServerConfig` y `ProvisionerConfig` y en `.env.example`; migración `000004` (`error_metrics.user_id` `uuid → text`). La pseudonimización se aplica en los **adaptadores**: `Upsert`/`ListByUser` (API), `ReplaceAll` (`refresh-aggregates`) y el `DELETE FROM error_metrics` de `Execute` (`execute-deletions`). Dominio, puertos y services sin cambios; mocks no regenerados.
- `6.3.2` Separación de datos crudos verificada y documentada: `practices`/`analyses` son las tablas crudas y se purgan (`purge-raw-data`/`execute-deletions`); nuevo test Tier 3 `TestPrivacy_RawDataSeparatedFromAnalytics` certifica que analytics (`error_metrics`) solo lleva la clave pseudonimizada, sin texto crudo, y que el olvido elimina todo rastro.
- `6.3.3` Auditoría del `PIIHandler` como guard runtime: los únicos loggers de producción (`cmd/api`, `cmd/provisioner`) pasan por `logger.New` (envuelve `PIIHandler`); el resto de `slog.New` son tests con `io.Discard`. Nuevo `ops/scripts/pii_audit.sh` (grep defensivo de email/teléfono/API-key/JWT sobre logs, redacta las coincidencias y falla con exit 1).

**Decisiones**:
- **Pseudonimizador en `shared/`, no en `api/services/`** (desviación documentada del path del manifiesto A8): `refresh-aggregates` y `execute-deletions` (provisioner) también escriben/borran analytics, y `api`↔`provisioner` no se importan (A2/A3). Manifiesto §3.3 autoriza lo genuinamente transversal en `shared/`.
- **Pseudonimización en adaptadores, no en services** (acordado): HMAC es una función pura y un detalle de keying del almacén; mantiene `analytics.ErrorMetric.UserID domain.ID` y evita tocar dominio/puertos/mocks. La lectura (`ListByUser`) deriva el mismo pseudónimo y devuelve el id crudo del caller.
- **`error_metrics.user_id` `uuid → text`**: tabla derivada (la reconstruye `refresh-aggregates`), por lo que el cambio de tipo es seguro; el `down` trunca y revierte.
- **`access_events`/`deletion_requests` conservan `user_id` crudo**: son identidad/auditoría (A9/A4), no analytics; la pseudonimización aplica solo a la materialización de analytics.
- **`pii_audit.sh` espeja los patrones del `PIIHandler`** (email/teléfono/API-key/JWT) y redacta el hallazgo antes de imprimirlo, para no filtrar PII en el propio log de CI.

**Verificación**:
- `go build ./...` · `go vet ./...` (+ `-tags=integration`) · `go test -race -count=1 ./...` · `go test -race -count=1 -tags=integration ./...` → OK.
- `pnpm test-integration --filter=backend` → OK (Tier 3; incluido el nuevo test de separación).
- `pnpm lint` (3/3) · `pnpm test` · `pnpm typecheck --filter=frontend` · `pnpm build` (3/3) → OK.
- `pnpm generate` idempotente; `gofmt -l` limpio.
- `ops/scripts/pii_audit.sh` probado: log limpio → exit 0; log con email/API key → exit 1 con valores `[REDACTED]`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 7 (Hardening/CI) o cerrar el gap de `/analytics/progress` (sigue en `501`; no tiene item en el checklist 06). El gate de Fase 6 queda íntegro en verde.

---

## 2026-09-19 — Fase 6.2: endpoints A9 (export, olvido, access-log) (6.2.1–6.2.3)

**Estado**: bloque `6.2` completado. `go build/vet/test -race` (Tier 1+2+3) en verde; `pnpm test-integration --filter=backend` OK; `pnpm lint` (3/3), `pnpm test`, `pnpm typecheck --filter=frontend` y `pnpm build` (3/3) en verde. Cambio de contrato A12 (se quitó el `501` de los 3 endpoints) con `pnpm generate` idempotente.

**Hecho**:
- A12: `api.yaml` deja de documentar `501` en `/me/data/export`, `/me/data` y `/me/access-log` (`/analytics/progress` lo conserva); `pnpm generate` (solo cambia `gen.ts`; las firmas Go no dependen de la respuesta).
- `6.2.1`: `IdentityService.Export` + `PracticeRepository.ListAllByUser` (sin paginar). El email viene de `APP_USER_EMAIL` (nuevo, validado con `identity.NewEmail` en `LoadServerConfig`), coherente con `APP_USER_ID` (single-user, sin tabla `users`).
- `6.2.2`: `IdentityService.RequestDeletion` inserta una `deletion_request` pendiente y responde `202`; si ya hay una pendiente es idempotente (`HasPending`, AP4).
- `6.2.3`: dominio `identity.AccessEvent`, tabla `access_events` (migración `000003`, append-only) + `AccessLogRepository` (`Append`/`ListByUser`, sin Update/Delete); middleware `internal/api/accesslog` que registra cada petición autenticada (`action`=método, `resource_type`=path), best-effort.
- Handlers `ExportData`/`DeleteData`/`GetAccessLog`; `NewServer` recibe `*IdentityService`. Mocks regenerados (`DeletionRequestRepository`, `AccessLogRepository`, `ListAllByUser`).
- Tests: dominio `identity` 100%; `accesslog` 100%; services 97.1%; handlers con casos de éxito/error/idempotencia; Tier 3 de repos (`access_events`, `deletion_requests`, `ListAllByUser`) y e2e `/me/*` (export → access-log → delete idempotente).

**Decisiones**:
- **Email en config (`APP_USER_EMAIL`)**: el MVP no persiste usuarios (identidad desde config, §3.2); añadir una tabla `users` sería alcance de un modelo multi-usuario que no existe.
- **Middleware de auditoría en `internal/api/accesslog`** (no en `shared/httpx`): depende de un puerto storage del API y del bounded context `identity`; mantenerlo fuera de `shared/` respeta las capas (A2/A3).
- **Auditoría best-effort post-respuesta**: el evento se escribe tras `next.ServeHTTP`; un fallo no altera la respuesta ya enviada (solo se loguea sin PII).
- **`DELETE /me/data` idempotente**: dos llamadas con una solicitud pendiente dan `202` y una sola fila; el job `execute-deletions` (6.1) la materializa tras la gracia.
- **A12 sin cambio de firmas Go**: quitar un `response` no altera la interfaz generada (los handlers reciben `(w, r, params)`); solo regenera `gen.ts`.

**Verificación**:
- `go build ./...` · `go vet ./...` (+ `-tags=integration`) · `go test -race -count=1 ./...` · `go test -race -count=1 -tags=integration ./...` → OK.
- `pnpm lint` (3/3) · `pnpm test` · `pnpm typecheck --filter=frontend` · `pnpm build` (3/3) → OK.
- `pnpm test-integration --filter=backend` → OK (Tier 3).
- `pnpm generate` idempotente; `api.yaml`/`gen.ts` coherentes.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 6.3 — retención y pseudonimización (`pseudonymizer.go` con `APP_PSEUDONYM_SECRET` antes de analytics, separar datos crudos y verificar el `PIIHandler`) y el Gate de Fase 6 (`ops/scripts/pii_audit.sh`). Nota: `/analytics/progress` sigue en `501` (no tiene item en el checklist 06).

---

## 2026-09-19 — Fase 6.1: jobs batch del provisioner (6.1.1–6.1.5)

**Estado**: bloque `6.1` completado. `go build/vet` y `go test -race` (Tier 1+2) en verde; `pnpm test-integration --filter=backend` (Tier 3 con testcontainers) en verde; `pnpm lint` (3/3), `pnpm test` (backend + frontend 88), `pnpm build` (3/3), `pnpm typecheck --filter=frontend` y `pnpm generate` idempotente. Sin cambio de wire (A12).

**Hecho**:
- `6.1.1` `cmd/provisioner/main.go` con los subcomandos `refresh-aggregates`, `purge-raw-data` y `execute-deletions`; el despacho vive en `internal/provisioner/handlers/jobs.go` (ejecuta el service y loguea `affected`). Config `APP_DATABASE_URL` + `APP_RAW_RETENTION_DAYS`/`APP_DELETION_GRACE_DAYS` (default 30) en `shared/config/provisioner.go`.
- `6.1.2` `RefreshAggregatesService`: reconstruye `error_metrics` desde las `analyses` completadas (join con `practices` no borradas) con un `ReplaceAll` transaccional. `last_seen_at` = análisis más reciente; una métrica por `(user, code, window)`. Se añade `analytics.AllWindows` como fuente única de ventanas (el handler del API pasa a usarlo).
- `6.1.3` `PurgeRawDataService` + `RawDataRepository.PurgePracticesDeletedBefore`: borrado físico de prácticas `deleted_at < cutoff` y sus `analyses` en una transacción; idempotente.
- `6.1.4` Dominio `identity.DeletionRequest` (`Due`, `MarkExecuted`), migración `000002_deletion_requests.sql` y `ExecuteDeletionsService` (gracia configurable). Consumer-first: la tabla y el job existen antes del endpoint `DELETE /me/data` (6.2.2).
- `6.1.5` `internal/provisioner/di/module.go` con Fx propio (pool + repos + services + runner), sin compartir módulos con `api/`. El harness Tier 3 se movió de `internal/api/adapters/postgres/testsupport` a `internal/shared/testdb` (5 imports actualizados) para que provisioner lo reuse sin importar `api/`.
- Puertos `internal/provisioner/ports/storage/` (`AnalyticsSourceRepository`, `ErrorMetricRepository`, `RawDataRepository`, `DeletionRepository`) + mocks (`mockgen`, target nuevo en `Makefile`) + adaptadores Postgres propios.
- Tests: dominio `identity` 100%; services provisioner 98.1%; handlers provisioner 100%; adaptadores cubiertos con Tier 3; e2e de jobs (`test/e2e/provisioner_jobs_integration_test.go`) y smoke del grafo Fx (`provisioner_di_integration_test.go`).

**Decisiones**:
- **La transacción vive en el adaptador, no en el service** (AP8): los jobs son batch, así que los repos encapsulan la atomicidad (`ReplaceAll`, `PurgePracticesDeletedBefore`, `Execute`) y los services no necesitan `UnitOfWork` ni `txctx` propio. Se evita duplicar el UoW del API.
- **`refresh-aggregates` solo reconstruye `error_metrics`**: `progress_metrics`/`/analytics/progress` no tienen tabla ni item en el checklist 06 (el endpoint sigue en `501`); se documenta como gap para una fase posterior.
- **`execute-deletions` consumer-first**: se construye la tabla `deletion_requests`, el dominio y el job ahora; el productor (`DELETE /me/data`, 6.2.2) llegará después y solo insertará solicitudes.
- **`testsupport` → `shared/testdb`**: `api/` y `provisioner/` no pueden importarse (A2/A3) y el harness de testcontainers es genuinamente transversal. `txctx` se queda en `api/` porque los repos del provisioner no lo usan.
- **El provisioner aplica migraciones al arrancar** (`migrations.Up` en `newPool`), igual que `cmd/api`; goose es idempotente y usa advisory locks.
- **CLI sin dependencias**: subcomando posicional parseado de `os.Args` (stdlib); no se añade librería de CLI.

**Verificación**:
- `go build ./...` · `go vet ./...` · `go test -race -count=1 ./...` · `go test -race -count=1 -tags=integration ./...` → OK.
- `pnpm lint` (3/3) · `pnpm test` (backend + frontend 88 tests) · `pnpm typecheck --filter=frontend` · `pnpm build` (3/3) → OK.
- `pnpm test-integration --filter=backend` → OK (Tier 3; `purge-raw-data` y `execute-deletions` cubiertos como exige el gate).
- `pnpm generate` idempotente; `api.yaml`/`gen.*` sin drift (A12).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 6.2 — endpoints A9 (`GET /me/data/export`, `DELETE /me/data`, `GET /me/access-log`), hoy `501 not_implemented`. `DELETE /me/data` insertará una `deletion_request` (ya existen tabla y job). El `email` de `DataExport` obliga a decidir su origen: no hay tabla `users` en el MVP.

---

## 2026-09-19 — Docs: guía didáctica de Fase 5 (`docs/explains/fase-5-frontend.md`)

**Estado**: documentación. Sin cambios de código; el Gate de Fase 5 sigue en verde.

**Hecho**: creado `docs/explains/fase-5-frontend.md` con la misma estructura que las guías de las fases 2–4:
1. **Explicación para no técnicos** — analogía del "restaurante que abre al público": cocina/almacenes = backend; sala/mesas = frontend; camarero = `ApiClient`; carta = contrato/`gen.ts`; libreta de pedidos = TanStack Query; post-it = Zustand; corrector de la carta = Zod; espejo de 3 columnas = vista diff; panel del jefe = dashboard; inspector de sanidad = tests.
2. **Guía de estudio por bloques** (`5.0`–`5.5`): cableado HTTP del backend, base Next.js 15, tipos generados (F1), `ApiClient`/mapeo de errores e idempotencia, server vs client state (F7), Zod + React Hook Form (F8), word-diff LCS, polling + optimistic update/rollback (F9), accesibilidad (F6), dashboard de analíticas + fix de CORS/`fetch`, y la pirámide de tests (Vitest / RTL + axe / Playwright). Con snippets reales y callouts `> Concepto —`.
3. **Resumen de cambios en 3 viñetas** (base y datos · interfaz · tests).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 6 — Provisioner y privacidad.

---

## 2026-09-19 — Fase 5.5: tests frontend (Vitest, RTL + axe, Playwright e2e) — Gate de Fase 5

**Estado**: bloque `5.5` completado; **Gate de salida de Fase 5 en verde**. `pnpm typecheck` (1/1), `pnpm lint` (3/3), `pnpm test` (backend + frontend **88 tests**), `pnpm test:e2e` (Playwright **2 tests**) y `pnpm build` (3/3) OK. Sin cambio de wire (A12).

**Hecho**:
- `5.5.1` unit (Vitest 5 + jsdom): `diffWords`, `mostFrequentPattern`/`errorPatternLabel`, schemas Zod, `zodResolver`, `ApiClient`/`errors` (`toApiError`, idempotencia `409 analysis_pending`, `networkError`, `userMessage`, `buildPath`, query params, auth) y defaults de `query-client`. 47 tests.
- `5.5.2` componentes (RTL 16 + `axe-core`): presentacionales (`Tooltip`, `PracticeStatusBadge`, `ErrorPatternBadge`, `FragmentDiff`, `ErrorPatternFrequency`, `RecurringErrorAlert`) y con estado (`PracticeList`, `PracticeDiff`, `PracticeForm`, `AnalyticsDashboard`) con mocks de los hooks de query. 41 tests. `axe.run` en `src/test/a11y.ts` (reglas `region`/`color-contrast` desactivadas: no aplican a componentes aislados en jsdom).
- `5.5.3` e2e (Playwright): `playwright.config.ts` levanta `next dev` en `:3100` con `NEXT_PUBLIC_API_URL` same-origin; `e2e/practice-flow.spec.ts` mockea el backend con `page.route` (create `201` → analyze `202` → detail `completed` con fragmentos) y cubre el flujo crítica crear→analizar→diff, más la validación del formulario. Sin Postgres ni OpenAI.
- `5.5.4` sin `skip()`/`only` en ningún tier (verificado con `rg`).
- Infra: `vitest.config.mts` (`@vitejs/plugin-react`, alias `@ → ./src`, setup), `src/test/{setup.ts,a11y.ts,fixtures.ts}`; scripts `test`/`test:watch`/`test:e2e`; task `test:e2e` en `turbo.json` y script `pnpm test:e2e` en la raíz. `e2e/` excluido de `tsconfig` y ESLint (Playwright transpila aparte).

**Decisiones**:
- **`axe-core` directo en vez de `jest-axe`** (desviación de lo acordado): `jest-axe@11` no publica tipos TypeScript (solo `index.js`), lo que obligaría a `@types/jest-axe` + augmentación manual para Vitest. La checklist pide "axe-core"; se usa `axe.run()` (con tipos propios) y se asserta `violations == []`.
- **e2e con API mockeada y same-origin**: `NEXT_PUBLIC_API_URL=http://localhost:3100` evita CORS/preflight y hace la suite CI-friendly y determinista (MANIFEST_FRONTEND F10, "idealmente mockeado").
- **`page.route` ignora documentos y payloads RSC** (`resourceType === "document"`, header `RSC`, `?_rsc`): Next sirve páginas y RSC por los mismos paths que la API.
- **RTL mockea los hooks de query** (no HTTP): el `ApiClient` ya se testea aparte con `fetchImpl` inyectado; evita una dependencia extra (MSW).
- **`@vitejs/plugin-react`**: Vite 8/rolldown no transformaba JSX porque el `tsconfig` tiene `jsx: "preserve"` (lo exige Next); el plugin resuelve el transform.
- **`h3` → `h2` en las columnas del diff** (hallazgo de axe `heading-order`): `PracticeDiff` tiene `h1`, así que los encabezados de columna deben ser `h2` (antes `h1 → h3` saltaba un nivel).

**Verificación**:
- `pnpm typecheck --filter=frontend` → OK.
- `pnpm lint` → 3 successful (6 warnings conocidos del contrato).
- `pnpm test` → backend (`go test -race`) + frontend `Test Files 17 passed`, `Tests 88 passed`.
- `pnpm test:e2e` → `2 passed`.
- `pnpm build` → 3 successful (rutas `/`, `○ /analytics`, `ƒ /practices/[id]`, `○ /practices/new`).
- `go build/vet/test` sin cambios en esta sesión; `pnpm generate` no aplica (sin cambio de contrato).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 6 — Provisioner y privacidad (jobs `refresh-aggregates`, `purge-raw-data`, `execute-deletions` y endpoints A9: `/me/data/export`, `/me/data`, `/me/access-log`; hoy `501 not_implemented`). Los e2e de `/analytics/progress` y `/me/*` podrán ampliarse entonces.

---

## 2026-09-19 — Fase 5.4: verificación manual E2E del dashboard — fix CORS + fix `fetch` del cliente

**Estado**: flujo verificado end-to-end en navegador real (Chromium headless + CDP) contra Postgres y OpenAI reales. Dos bugs de cableado **preexistentes** encontrados y corregidos (uno bloqueaba todo el frontend). `go build/vet/test` y `pnpm typecheck/lint/build` en verde.

**Contexto**: se ejecutó la verificación manual de `5.4`: crear práctica → analizar → `GET /analytics/error-patterns` → render del dashboard en `/analytics`. Al probar en navegador aparecieron fallos que el walkthrough anterior había diferido ("revisión en navegador").

**Hallazgos y fixes**:
1. **CORS ausente (bloqueante)**: el backend no emitía cabeceras CORS; el navegador (origen `:3000` → API `:8080`) no podía leer las respuestas. Fix: middleware `httpx.CORS` (`internal/shared/httpx/cors.go`, responde preflight `OPTIONS` con `204`) + config `APP_CORS_ALLOWED_ORIGINS` (default `http://localhost:3000`, `config/server.go`) cableada en `di/module.go`. Tests `cors_test.go` + `server_test.go`. Documentado en `.env.example`.
2. **`TypeError: Illegal invocation` en `client.ts` (bloqueante, afectaba a todo el frontend)**: `this.fetchImpl = options.fetchImpl ?? fetch` guardaba el `fetch` nativo desvinculado; al invocarlo como `this.fetchImpl(...)` el `this` era la instancia de `ApiClient` y el navegador lo rechazaba. Fix: `options.fetchImpl ?? ((input, init) => fetch(input, init))`. También rompía `/` y `/practices/[id]` (nunca antes verificados en navegador).
3. **`NEXT_PUBLIC_API_URL` sin definir**: no existía ningún `.env.local`; Next no inlinea la variable no definida y no hay `process` en el navegador → `ReferenceError`. Se crea `apps/frontend/.env.local` (gitignored) con `NEXT_PUBLIC_API_URL=http://localhost:8080`. `.env.example` ya lo documentaba (5.1.5).
4. **Dev server corrupto por `pnpm build` concurrente**: `next build` sobrescribió `.next/` mientras `next dev` seguía vivo → chunks (`main-app.js`, `app-pages-internals.js`) daban `404` y no hidrataba. Se reinició el dev server (operativo). Cuidado: no correr `build` con `dev` en marcha.

**Verificación E2E (real)**:
- API: `POST /practices` `201` → `POST /analyze` `202` → polling `analyzing`→`completed` (fragmento `tense_agreement`) → `GET /analytics/error-patterns?window=week` pasó de `count 1` a `count 2`; `day`/`month` coherentes.
- Navegador (CDP): `/analytics` renderiza "Tu error más frecuente es tiempo/concordancia (2 veces)", la frecuencia con `2 veces`/`Última vez: 19 sept` y "La serie de progreso estará disponible próximamente (Fase 6)"; el selector `Día` mueve `aria-pressed` y refetch con `window=day`; `/` lista las prácticas (`GET /practices` 200).
- Estático: `go build/vet` + `go test -count=1 ./...` OK; `pnpm typecheck --filter=frontend` (1/1), `pnpm lint` (3/3) y `pnpm build` (3/3, rutas `/` y `○ /analytics`) OK.

**Decisiones**:
- **CORS en el backend (no rewrites de Next)**: las apps se despliegan por separado (F3); las rutas del API colisionan con páginas (`/analytics`, `/practices`) y un proxy no es viable. Allow-list configurable por env.
- **`httpx.CORS` como middleware de `NewRouter` vía `di`**: no se cambia la firma de `NewRouter`; se compone junto a `UserResolver`.
- **Verificación con Chromium de la caché de Playwright + CDP sobre Node 24** (`WebSocket` global): sin instalar dependencias; permitió ver el DOM hidratado y la red. La infra de tests formal (Vitest/RTL/Playwright) llega en `5.5`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 5, bloque `5.5` — Vitest (unit, empezando por `diffWords`, `mostFrequentPattern`, schemas y el cliente HTTP), RTL + axe-core y Playwright e2e. Añadir un test unitario del `ApiClient` (fetch inyectado) que habría atrapado el bug #2.

---

## 2026-09-19 — Fase 5.4: dashboard de analíticas (5.4.1–5.4.3)

**Estado**: bloque completado. `pnpm typecheck --filter=frontend` (1/1), `pnpm lint` (3/3) y `pnpm build` (3/3) en verde; nueva ruta `○ /analytics`. `pnpm generate` idempotente; **sin cambio de wire** (A12 no aplica).

**Hecho**:
- `5.4.1` `lib/query/analytics.ts`: `analyticsKeys`, `useErrorPatternStats(window)` (`GET /analytics/error-patterns`) y `useProgressSeries(window)` (`GET /analytics/progress`), tipados desde `gen.ts` (F1). La ventana forma parte de la key de query.
- `5.4.2` feature `features/analytics/`: `analytics-dashboard.tsx` (selector de ventana day/week/month, estados loading/error/empty), `error-pattern-frequency.tsx` (frecuencia por patrón con barras horizontales, count y `last_seen_at`; WCAG AA) y `recurring-error-alert.tsx` (banner `role="status"` del patrón más frecuente — PRODUCT_DOMAIN §8.1). Ruta `app/analytics/page.tsx` y enlace "Analíticas" en la home.
- `5.4.3` F11: el cliente no clasifica ni calcula severidad; solo consume `count`/`last_seen_at` ya agregados por el backend.
- `lib/store/analytics.ts`: `useAnalyticsStore` (Zustand) con la ventana seleccionada (client state, F7).
- `lib/utils/error-patterns.ts`: se extrae `ERROR_PATTERN_LABELS` (antes duplicado en `error-pattern-badge.tsx`) + `errorPatternLabel` + `mostFrequentPattern` (puro, testable en 5.5.1).

**Decisiones**:
- **`/progress` con placeholder graceful** (acordado): el backend responde `501 not_implemented` (diferido a Fase 6, `server.go:222`). Se consume igualmente con tipos de `gen.ts` y el dashboard muestra "disponible próximamente"; así el item 5.4.1 se cumple sin tocar el backend ni encadenar Fase 6.
- **Ventana en Zustand, no `useState`** (acordado): "preferencias locales" son client state (F7); store dedicado (`lib/store/analytics.ts`) en vez de mezclarlo con `useUiStore`.
- **Labels compartidos extraídos a `lib/utils/`**: la feature analytics y el badge de práctica comparten el mapeo `code → label`; una sola fuente evita drift (F11). Los labels son presentación; el enum/severidad los define el backend.
- **`retry: false` en `useProgressSeries`**: reintentar un `501` es inútil y genera ruido.

**Verificación**:
- `pnpm typecheck --filter=frontend` → OK (`next typegen && tsc --noEmit`).
- `pnpm lint` → 3 successful (6 warnings conocidos del contrato `operation-4xx-response`).
- `pnpm build` → 3 successful; rutas `/` y `○ /analytics` (3.54 kB).
- `pnpm generate` idempotente: `gen.ts`/`gen_*.go` sin cambios (`git status` solo muestra fuentes propias).

**Bloqueos**: ninguno.

**Nota (F10)**: la infra de tests de frontend (Vitest/RTL/Playwright) es el bloque `5.5` y aún no existe; `mostFrequentPattern` queda como lógica pura lista para unit-test.

**Próximo paso**: Fase 5, bloque `5.5` — Vitest (unit), React Testing Library + axe-core (componentes) y Playwright (e2e crear → analizar → diff), o `5.5.1` en particular. La serie `/analytics/progress` (con su gráfica) sigue diferida a Fase 6.

---

## 2026-09-19 — Fase 5.3.5: formulario de creación + walkthrough E2E y fix del base URL de OpenAI

**Estado**: `5.3.5` completado; gap de creación cerrado. Frontend verde (typecheck/lint/build); backend verde (build/vet/test sin y con `-race`). Recorrido manual E2E verificado contra backend + OpenAI reales.

**Hecho**:
- `5.3.5` frontend: dep `react-hook-form@7.88`; `lib/utils/zod-rhf.ts` (resolver Zod→RHF propio); mensajes en español en `lib/schemas/practice.ts`; `useCreatePractice` (`onSuccess` invalida `["practices"]` y navega al detalle); `practice-form.tsx` (RHF + `useFieldArray`, `aria-invalid`/`role="alert"`); ruta `/practices/new` y enlace "Nueva práctica" en la home.
- `apps/frontend/package.json`: el script `typecheck` pasa a `next typegen && tsc --noEmit` (genera tipos de ruta frescos; elimina el falso fallo por `.next` obsoleto).
- **Bug backend (bloqueaba el walkthrough)**: `openai-go` lee `OPENAI_BASE_URL` del entorno (`client.go:45`, `os.LookupEnv`); el `.env` trae `OPENAI_BASE_URL=` vacío y el loader lo deja presente-pero-vacío, así que el SDK pisaba su default → `Post "/chat/completions": unsupported protocol scheme ""`. Fix: `llm.BaseURLOrDefault` (`internal/api/adapters/llm/base_url.go`) y pasar siempre `option.WithBaseURL(...)` (default `https://api.openai.com/v1`) en `cmd/llmcheck` y `di/module.go`; test `base_url_test.go`.
- `.env.example`: `OPENAI_API_KEY=` (placeholder) tras rotar la clave real (higiene A8).

**Decisiones**:
- **RHF + resolver propio** (acordado): evita `@hookform/resolvers` y su posible desajuste con Zod 4; F8 se mantiene (RHF + Zod).
- **`5.3.5` como item nuevo** (acordado): el checklist no cubría la creación; sin ella no había datos para el diff ni para el e2e de `5.5.3`.
- **`next typegen` en `typecheck`**: preferido a castear `Link`/`router.push`; el Gate deja de depender de un `.next` rancio.
- **Fix del base URL en el adapter/wiring**, no en `config`: mantiene la semántica de `OpenAIConfig` y hace explícito el default del proveedor.

**Verificación**:
- Frontend: `pnpm typecheck` (1/1) · `pnpm lint` (3/3) · `pnpm build` (3/3; rutas `/`, `ƒ /practices/[id]`, `○ /practices/new`).
- Backend: `go build ./...` · `go vet ./...` · `go test -count=1 ./...` · `go test -race -count=1 ./...` → OK.
- **Walkthrough E2E real** (Postgres `:5433` + `cmd/api` + OpenAI `gpt-4o-mini`): `POST /practices` 201 → `POST /analyze` 202 → polling `analyzing`→`completed` → 1 fragmento con `tense_agreement/critical`. El frontend dev sirve `/`, `/practices/new` y `/practices/{id}` (200). El render client-side del diff/polling y la a11y por teclado quedan para revisión en navegador.

**Bloqueos**: ninguno.

**Nota de seguridad**: la `OPENAI_API_KEY` real que estaba en `.env.example` fue rotada y sustituida por placeholder.

**Próximo paso**: Fase 5, bloque `5.4` (dashboard de analíticas) o `5.5` (Vitest/RTL/Playwright, ya con el flujo de creación desbloqueado para el e2e).

---

## 2026-09-19 — Fase 5.3: vista diff de 3 columnas (5.3.1–5.3.4)

**Estado**: bloque completado. `pnpm typecheck` (1/1), `pnpm lint` (3/3) y `pnpm build` (3/3) en verde; ruta `ƒ /practices/[id]` generada. Sin cambio de wire (A12).

**Hecho**:
- `5.3.2` `lib/utils/diff.ts`: `diffWords(before, after)` (LCS a nivel palabra, sin dependencias) → `DiffToken[]` (`equal|del|ins`) con tokens adyacentes fusionados. Lógica pura (presentación, F11).
- `5.3.3` `lib/query/practices.ts`: `practiceKeys`, `useListPractices`, `usePractice(id)` (`refetchInterval` 2s mientras `status === "analyzing"`, luego `false`) y `useAnalyzePractice(id)` (optimistic: `onMutate` snapshot + `status:"analyzing"`; `onError` rollback salvo `isAnalysisPending` → éxito idempotente sin rollback; `onSettled` invalida). Consume `isAnalysisPending` de 5.2.3 (AP-F6).
- `5.3.1`/`5.3.2` componentes: `features/practice/components/{fragment-diff,practice-diff,practice-list,status-badge,error-pattern-badge}.tsx` + `components/ui/tooltip.tsx`. Fila de 3 columnas (Español | borrador con `del` rojo tachado | corrección con `ins` verde), badges de `error_patterns` y paneles expandibles con los 3 campos. Estados de la práctica: `draft` (botón Analizar) / `analyzing` (auto-refresh) / `completed` (fragmentos) / `failed`.
- `5.3.4` accesibilidad: semántica (`section`, `h1`–`h3`, `ul`), botones nativos con `aria-expanded`/`aria-controls`, `role="status"`/`role="alert"`, tooltip `role="tooltip"` + `aria-describedby` (focus/hover/Escape) y diff con doble cue (color + tachado/negrita — WCAG 1.4.1) con contraste AA.
- Rutas: `app/practices/[id]/page.tsx` (server que lee `params`) y `app/page.tsx` (lista con `Link` a cada práctica). `lib/api/errors.ts`: `userMessage(error)` (mapper código → UX, F5).

**Decisiones**:
- **Word-diff propio (no `jsdiff`)** (acordado con el humano): LCS puro y testeable, sin dependencia nueva.
- **Ruta `[id]` + lista en home** (acordado): permite navegar al diff sin depender de un formulario de creación (que no existe aún en el checklist).
- **Sin botón de reintento tras `failed`**: `StartAnalysis` solo permite `draft → analyzing`; desde `failed` sería `invalid_state`. El dominio manda (F11); se muestra el mensaje de fallo.
- **`userMessage` en `lib/api/errors.ts`**: el manifiesto §5.3 pide el mapper de errores centralizado ahí.
- **`Link` con template literal** (`/practices/${id}`): el objeto con `params` no encaja con los tipos de `Link` de Next 15.5. `typedRoutes` genera los tipos en `.next/types`; un `.next` **obsoleto** (de una build anterior al alta de la ruta) hace fallar `tsc` de forma espuria — se resuelve reconstruyendo (`next build`). En checkout limpio (sin `.next`) `typecheck` pasa.

**Verificación**:
- `pnpm typecheck` · `pnpm lint` · `pnpm build` → OK (build genera `ƒ /practices/[id]`).
- Runtime del word-diff (harness en `/tmp`, fuente compilada): **9/9** — identidad, reemplazo (`go`→`went`), inserción, eliminación, múltiples cambios (`run`→`ran` + `whole`), vacíos y puntuación; con invariante de reconstrucción (equal+del = borrador; equal+ins = corrección) y sin tokens adyacentes del mismo tipo.
- **Hallazgo**: el harness atrapó un bug real en el backtracking (faltaba avanzar `j` en la rama `equal`, que duplicaba tokens); corregido antes del commit.

**Bloqueos**: ninguno.

**Gap pendiente (decisión del humano)**: el flujo de producto necesita **crear** una práctica (y editarla) para alimentar el diff, pero el checklist de Fase 5 no incluye un item de formulario/creación; solo `5.2.6` dejó los schemas Zod listos y `react-hook-form` quedó diferido. Se implementará cuando se apruebe (probablemente como item propio antes de `5.5.3`, que exige el e2e "crear práctica → análisis → diff").

**Próximo paso**: Fase 5, bloque `5.4` — dashboard de analíticas (`GET /v1/analytics/error-patterns`, tipos desde `gen.ts`, sin reimplementar reglas — F11), o cerrar el gap de creación si se aprueba.

---

## 2026-09-19 — Fase 5.2: cliente HTTP, TanStack Query, Zustand y Zod (5.2.1–5.2.6)

**Estado**: bloque completado. `pnpm typecheck --filter=frontend`, `pnpm lint` (3 apps) y `pnpm build --filter=frontend` en verde; `pnpm generate` idempotente.

**Hecho**:
- `5.2.1` verificado: `gen.ts` ya está commiteado y `pnpm generate` es idempotente (sha256 estable); `api.yaml` intacto (A12).
- `5.2.2` `lib/api/errors.ts`: `ErrorCode` (union del contrato), `ApiError{status, code}`, `toApiError(Response)` (parsea `ErrorResponse` sin exponer detalle de infraestructura, F5) y `networkError()`.
- `5.2.2` `lib/api/client.ts`: `ApiClient` tipado contra `paths` (F1) con `get/post/put/patch/delete`, base URL desde `NEXT_PUBLIC_API_URL`, query params vía `URLSearchParams`, `buildPath()` para `{param}`, `AbortSignal`, auth inyectable (`getAuthToken`, no-op en el MVP single-user) y `request<T>` con normalización de errores a `ApiError`. Singleton `apiClient`.
- `5.2.3` `isIdempotentSuccess(409, "analysis_pending")` + flag `ApiError.isIdempotentSuccess` + guard `isAnalysisPending` (AP-F6); lo consumirá la mutación `analyze` en 5.3.3.
- `5.2.4` `lib/query/query-client.ts`: `makeQueryClient`/`getQueryClient` (singleton SSR-safe) con defaults `staleTime` 30s, `gcTime` 5min, `retry` 1, `refetchOnWindowFocus:false`. `app/providers.tsx` (`QueryClientProvider`, client component) cableado en `layout.tsx`.
- `5.2.5` `lib/store/ui.ts`: `useUiStore` (Zustand v5) con estado de UI efímero; sin datos del API (F7/AP-F4).
- `5.2.6` `lib/schemas/practice.ts`: schemas Zod (`targetRule`, `createPractice`, `updatePractice.partial()`) con aserciones compile-time `AssertAssignable<z.infer<...>, components["schemas"][...]>` (F8).
- Deps añadidas: `@tanstack/react-query@5.103.1`, `zustand@5.0.15`, `zod@4.6.5`.

**Decisiones**:
- **Cliente `fetch` manual tipado, no `openapi-fetch`** (acordado con el humano): el checklist pide "fetch wrapper"; se evita una dependencia runtime extra y se mantiene el control del mapeo de errores (F5). El tipado contra `paths`/`components` viene de `gen.ts` (F1).
- **`react-hook-form` diferido a 5.3**: el item 5.2.6 pide solo *schemas* Zod; el hook de formulario se añade cuando exista el formulario de práctica.
- **Idempotencia en el mapper, no en el cliente**: `ApiError.isIdempotentSuccess` clasifica `409 analysis_pending`; el tratamiento como éxito se materializa en la mutación (5.3.3, F9). No se lanza excepción de éxito en el `fetch` genérico.
- **Sin caché de API en Zustand**: el store solo contiene UI efímera (F7/AP-F4).
- **Assertions de contrato en Zod (`AssertAssignable`)**: verifican en compilación que `z.infer` es asignable al tipo de `gen.ts`; se exportan para que ESLint no las marque como no usadas.

**Verificación**:
- `pnpm typecheck --filter=frontend` → OK (incluye las aserciones de contrato Zod).
- `pnpm lint` → 3 successful (6 warnings conocidos del contrato `operation-4xx-response`).
- `pnpm build --filter=frontend` → `next build` OK (4 páginas estáticas).
- `pnpm generate` idempotente; sin cambio de wire (A12).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 5, bloque `5.3` — vista diff de 3 columnas: componente de diff, paneles expandibles y polling del análisis con TanStack Query + optimistic update/rollback (consumiendo `apiClient`, `ApiError.isIdempotentSuccess` y los schemas Zod).

---

## 2026-09-18 — Dev: Postgres de desarrollo (docker-compose) + fix de `APP_LLM_TIMEOUT`

**Estado**: sin cambios de lógica; desbloqueo del arranque local del backend. Verificado en vivo: `docker compose up` → `go run ./cmd/migrate` → `go run ./cmd/api` → `POST /practices` 201 y `GET /practices` 200.

**Hecho**:
- `ops/docker/docker-compose.yml`: Postgres 16 dev con credenciales `langlint`/`langlint`/`langlint` (las mismas de testcontainers), expuesto en el puerto host **5433** (el 5432 ya estaba ocupado por un Postgres local ajeno). Volumen persistente.
- `.env.example`: `APP_DATABASE_URL` apunta a `localhost:5433`; **fix** `APP_LLM_TIMEOUT=60s` (estaba `60` sin unidad → `time.ParseDuration` fallaba).

**Decisiones**: puerto **5433** en el host para no colisionar con el Postgres existente del equipo; el puerto interno del contenedor sigue siendo 5432.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 5, bloque `5.2` (cliente HTTP del frontend). Para desarrollo: `docker compose -f ops/docker/docker-compose.yml up -d` y asegurarse de que `.env` tenga `APP_DATABASE_URL` en `:5433` y `APP_LLM_TIMEOUT=60s`.

---

## 2026-09-18 — Fase 5.0: cableado HTTP del backend (5.0.1–5.0.11)

**Estado**: bloque completado; Gate en verde. `go build/vet/test -race` (+ `-tags=integration`) OK; dominio 100%, services **97.5%**, event_handlers **91.4%**; e2e Tier 3 del flujo HTTP completo. `pnpm build`/`pnpm lint`/`pnpm typecheck` (3 apps) OK; `pnpm generate` idempotente.

**Contexto**: las Fases 1–4 nunca cubrieron `cmd/api`, los handlers chi, los services de práctica ni el wiring de eventos (gap ya anotado en AGENTS/DEVLOG). Se cerró como bloque propio, con las decisiones acordadas con el humano: solo la superficie que consume el frontend; `AnalysisRequested` como trigger; transición de estado en event handlers; DI con Uber Fx; `/analytics/progress` y `/me/*` diferidos a Fase 6.

**Hecho (A12 primero)**:
- `5.0.1` contrato: `ErrorResponse.code` + `not_implemented` e `internal` (este último cerraba el drift de `PRODUCT_DOMAIN §4.5`), respuestas `501` en `/me/*` y `/analytics/progress`; `pnpm generate` (Go + TS).
- `5.0.2` dominio: evento `AnalysisRequested{PracticeID,UserID,Version}` + registro `NewEvent`.
- `5.0.3`/`5.0.4` `shared/db` (pool + ping), `cmd/migrate` (goose embebido), `shared/config` server (`APP_DATABASE_URL`/`APP_HTTP_ADDR`/`APP_USER_ID`/`APP_LLM_TIMEOUT`) y `shared/logger.New` (cablea el `PIIHandler`, cierra 4.3.4).
- `5.0.5` `handlers/errors.go` (map A5 → status/`ErrorResponse`) + `shared/httpx` (router chi + middlewares + resolvedor de usuario single-user).
- `5.0.6` `PracticeService` (create/get/list/update/delete/analyze; `outbox.Append` en tx) y `AnalyticsService`.
- `5.0.7` `handlers/server.go`: implementa `ServerInterface`, conversión dominio↔wire, `Retry-After` en el 202.
- `5.0.8` `services/event_handlers/`: `AnalysisRequested`→`RunAnalysis`; `AnalysisCompleted`→`MarkCompleted` + `Upsert ErrorMetric` (day/week/month); `AnalysisFailed`→`MarkFailed`. Idempotentes (guard `status == analyzing`).
- `5.0.9` analytics `error-patterns`; `501 not_implemented` en progress/access-log/export/delete.
- `5.0.10` `di/` Fx v1.24.0 + `cmd/api` (pool migra al arrancar; dispatcher + relay con lifecycle; HTTP server).
- `5.0.11` tests: unit de services/handlers/event_handlers + `test/e2e` Tier 3 (`create→analyze→poll→completed→analytics`) + smoke del grafo Fx.

**Decisiones**:
- **`AnalysisRequested` (dominio) en vez de reusar `PracticeCreated`**: crear deja la práctica en `draft`; el análisis lo dispara `POST /analyze`. El evento es interno (no toca el wire) y mantiene `PracticeCreated` para futuros consumidores.
- **Transición `analyzing → completed|failed` en los event handlers**, no en `RunAnalysis`: el service del LLM queda enfocado en el agregado `Analysis`; el estado de la práctica es event-driven post-commit.
- **`ErrorMetric` incremental leído-y-escrito**: `Upsert` es absoluto (Fase 3) y el evento trae las ocurrencias de un análisis; el handler lee el conteo actual (`ListByUser`), suma y hace `Upsert` en las 3 ventanas. Idempotencia best-effort vía guard de estado (at-least-once).
- **Usuario único vía `APP_USER_ID`** en un middleware: el contrato no tiene auth (MVP).
- **`/me/*` y `/analytics/progress` devuelven `501 not_implemented`**: son Fase 6 (A9 y `refresh-aggregates`); el enum del contrato y las respuestas 501 se añadieron vía A12 para no mentir en el wire.
- **Mapper de errores en `handlers/`** (no en `internal/api/errors.go`, como sugiere el árbol del manifiesto): AP-MR2 exige que los tipos `gen_*.go` solo se importen en `handlers/`; el mapper necesita `ErrorResponse`. Desviación documentada.
- **`AnalysisRunner` como interfaz estrecha** en `event_handlers` (la satisface `*services.AnalysisService`): desacopla y hace testeable el handler.
- **Fx `NopLogger`**: el único logger del proyecto es `log/slog` (A8); el logger interno de Fx no se usa.

**Verificación**:
- `go build ./...` · `go vet ./...` · `go test -race -count=1 ./...` · `go test -race -count=1 -tags=integration ./...` → OK.
- Cobertura: dominio 100%; `services` **97.5%**; `services/event_handlers` **91.4%**; `handlers` 43.2% (incluye el código generado, sin gate).
- e2e Tier 3 (`test/e2e`): create→analyze→poll(completed)→error-patterns + arranque/parada del módulo Fx con Postgres 16 (testcontainers).
- `pnpm build` 3/3 · `pnpm lint` 3/3 · `pnpm test` backend OK · `pnpm typecheck --filter=frontend` OK · `pnpm generate` idempotente. `gofmt -l` limpio; sin `t.Skip()`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 5, bloque `5.2` — cliente HTTP tipado (`lib/api/client.ts`/`errors.ts`), TanStack Query, Zustand, schemas Zod. El backend ya sirve prácticas, analyze y `error-patterns` en `http://localhost:8080` (con Postgres + `cmd/migrate`).

---

## 2026-09-18 — Fase 5 (inicio): base Next.js 15 + Tailwind v4 (5.1.1–5.1.5)

**Estado**: Fase 5 en curso. Items `5.1.1`–`5.1.5` completados; `pnpm build` (3 successful), `pnpm lint` (3 successful) y `pnpm typecheck --filter=frontend` en verde.

**Hecho**:
- `5.1.1` `apps/frontend/`: Next.js **15.5.25** (App Router) + React **19.3.0** + TypeScript **5.9.3** `strict` + Tailwind **v4.3** (CSS-first). Scripts `dev`/`build`/`start`/`lint`/`typecheck`/`generate`; se conserva el `generate` de `openapi-typescript` (lo invoca `contracts/scripts/generate.sh`). `src/app/{layout.tsx,page.tsx,globals.css}` con `@import "tailwindcss"`.
- `5.1.2` Estructura canónica `src/{app,components,features,lib,types}` (MANIFEST_FRONTEND §3.1): `components/{ui,feature}`, `lib/{api,query,store,utils}`, `types/` con re-export de `components`/`paths`/`operations` desde `lib/api/gen.ts` (F1). Subcarpetas vacías con `.gitkeep`.
- `5.1.3` `tsconfig.json`: `strict`, `target ES2022`, `moduleResolution bundler`, `jsx preserve`, `paths @/* → ./src/*`, `noUncheckedIndexedAccess`, `noFallthroughCasesInSwitch`, plugin `next`.
- `5.1.4` `next.config.js`: `reactStrictMode: true` + `typedRoutes: true`.
- `5.1.5` `apps/frontend/.env.example`: `NEXT_PUBLIC_API_URL` (única expuesta al navegador) + prefijo server-only `FRONTEND_*` documentado (AP-F8).
- ESLint 9 flat config (`eslint.config.mjs`, `eslint-config-next`) con `ignores` de `.next/**` y `next-env.d.ts`; `*.tsbuildinfo` añadido a `.gitignore`.

**Decisiones**:
- **Next 15.5 (no 14) y Tailwind v4** (confirmado con el humano): "última" del manifiesto §2.1; React 19. La checklist dice "14+".
- **`typedRoutes` top-level, no `experimental`**: Next 15.5 lo promovió a estable; en `experimental` emite warning de deprecación. Se documenta la desviación del snippet §2.2 del manifiesto (que aún lo muestra experimental).
- **Renombrar el paquete `@langlint/frontend` → `frontend`**: cierra el drift anotado en el DEVLOG de Fase 3 ("se corregirá al cablear el frontend"); el gate exige literalmente `--filter=frontend` y el backend ya es `backend` sin scope. `contracts/scripts/generate.sh` invoca por directorio (`cd apps/frontend`), así que no se ve afectado.
- **Scaffold manual (no `create-next-app`)**: para respetar el árbol exacto del manifiesto §3.1 y controlar versiones/config.
- **Tailwind v4 CSS-first**: sin `tailwind.config.ts` (postcss plugin `@tailwindcss/postcss`), coherente con la elección de "última".
- **`tsc --noEmit` con `incremental` genera `tsconfig.tsbuildinfo`**: se ignora en git (artefacto) y ESLint no lintea `.next/` ni `next-env.d.ts` (generados; `next build` añade ahí la referencia a `.next/types/routes.d.ts`).

**Verificación**:
- `pnpm build --filter=frontend` → `next build` OK (compila, lintea y valida tipos; 4 páginas estáticas).
- `pnpm build` (monorepo) → 3 successful; `pnpm lint` → 3 successful; `pnpm typecheck --filter=frontend` → OK.
- Sin cambio de wire: `api.yaml` y `gen.ts` intactos (A12). ESLint sin errores/warnings.

**Bloqueos**: ninguno.

**Pendiente de decisión del humano** (heredado de Fase 4): el cableado HTTP del backend (`cmd/api`, handlers chi, `CreatePracticeService`, handler de `PracticeCreated`, `PIIHandler` y timeout por config) sigue sin cubrirse en ningún checklist. No bloquea 5.1 (scaffolding), pero es prerequisito real para `5.3` (polling contra el API).

**Próximo paso**: Fase 5, bloque `5.2` — regenerar/committear `lib/api/gen.ts` (`5.2.1`, ya existe y se regenera con `pnpm generate`), `lib/api/client.ts` + `lib/api/errors.ts` (`5.2.2`), idempotencia `409 analysis_pending` (`5.2.3`), TanStack Query (`5.2.4`), Zustand (`5.2.5`) y schemas Zod (`5.2.6`).

---

## 2026-09-18 — Docs: guía didáctica de Fase 4 (`docs/explains/fase-4-motor-ia.md`)

**Estado**: documentación. Sin cambios de código; el Gate de Fase 4 sigue en verde.

**Hecho**: creado `docs/explains/fase-4-motor-ia.md` con la misma estructura que `fase-2-dominio-puro.md` y `fase-3-puertos-adaptadores.md`:
1. Explicación para no técnicos (analogía de la "academia con tutor externo": el LLM en otro edificio, el secretario con rotulador = anonymizer, el formulario con casillas = JSON Schema, el revisor = validación, el cajón abierto = transacción, el aviso de fallo = `AnalysisFailed`, la trituradora de papel = `PIIHandler`).
2. Guía de estudio por 9 bloques: puerto `LLMExtractor` (A1), prompt, Structured Outputs + JSON Schema (incluida la lección del *root object*), validación, `AnalysisService` (LLM fuera de tx + outbox + fallo como estado terminal), anonimización, timeout/`AnalysisFailed`, `PIIHandler` y verificación/gate. Con snippets reales y 33 callouts `> Concepto —`.
3. Resumen de cambios en 3 viñetas (motor IA, `AnalysisService`, privacidad + gate).

**Bloqueos**: ninguno.

**Próximo paso**: Fase 5 según roadmap (Frontend). Pendiente de decisión: el cableado HTTP (`cmd/api`, handlers, `CreatePracticeService`, handler de `PracticeCreated`) no está cubierto por el checklist de Fase 4.

---

## 2026-09-18 — Fase 4 (cierre): anonimización, timeout/`AnalysisFailed` y `PIIHandler` (4.3.1–4.3.4)

**Estado**: **Fase 4 completada; Gate de salida en verde.** `go build/vet/test -race` verdes (Tier 1 + Tier 3); `adapters/llm` **100%**, `services` **96.9%**, `shared/logger` **100%**; A1 en 0.

**Hecho**:
- `4.3.1` `services/anonymizer.go`: `Anonymize` redacta emails (`[email]`), teléfonos (`[phone]`) y nombres de una **lista curada ES/EN** (`[name]`; solo ocurrencias capitalizadas, para no confundir el verbo inglés *mark* con el nombre *Mark*); `AnonymizeSpanish` añade la **heurística de mayúsculas a mitad de frase** (solo para el source en español). `AnalysisService` anonimiza `SourceText`/`DraftText` antes de `Extract`.
- `4.3.2` sin cambios: no hay goroutines propias en `internal/` (`grep "go func"` → 0); `Extract` es síncrono y propaga el `ctx` al SDK. AP6 se cumple **por construcción** (la regla de lint `goroutine_context` es de Fase 7).
- `4.3.3` `analysis_service.go`: `context.WithTimeout` (**60s** por defecto) alrededor de la llamada al LLM; ante fallo (`Extract` o `Complete`) se hace `Analysis.Fail()`, se persiste el `Analysis` failed + `outbox.Append(AnalysisFailed{Reason genérico})` en una transacción y se **retorna `nil`** (fallo manejado). El error crudo del proveedor nunca se filtra.
- `4.3.4` `shared/logger/pii_handler.go`: decorador de `slog.Handler` que descarta registros con texto libre >100 runas, email, teléfono, API key (`sk-`/`pk-`/`AKIA`) o JWT; `WithAttrs` elimina atributos sensibles para que `Logger.With` no filtre PII.

**Decisiones**:
- **Anonimizar en el service (caller), no en el adapter**: el puerto `ExtractRequest` ya documenta "anonymized input" y el adapter es provider-specific (A1/A2).
- **Heurística de mayúsculas solo en español**: el borrador inglés usa `Anonymize` (solo lista) para no redactar palabras capitalizadas legítimas ("English", "Monday").
- **Lista de nombres con matching capitalizado** (hallazgo de la revisión manual): distingue nombres propios de palabras homógrafas comunes (`Mark` vs el verbo `mark`, `Frank` vs `frank`), evitando corromper el análisis del borrador.
- **Redacción con tokens, no borrado**: conserva la estructura de la frase para que el LLM analice gramática y no "corrija" huecos.
- **Fallo manejado → retorno `nil`**: evita reintentos infinitos del `OutboxRelay` sobre un fallo terminal; el estado queda en `analysis.status == failed` + evento `AnalysisFailed`.
- **`PIIHandler` no detecta nombres propios**: se cubren por el guard de texto libre >100 runas (la checklist 4.3.4 pide "contenido libre y patrones PII").
- **`PIIHandler` no se cablea aún**: no existe `cmd/api`; el wiring del logger llega en Fase 5.
- **Timeout como campo del service** (default 60s, configurable por DI): el `cmd/llmcheck` ya usaba `context.WithTimeout`; la config por env se difiere al wiring.

**Verificación**:
- `go build ./...` · `go vet ./...` (+ `-tags=integration`) · `go test -race -count=1 ./...` (+ `-tags=integration`) → OK.
- Cobertura: `adapters/llm` 100%, `services` 96.9%, `shared/logger` 100%.
- A1: `go list -deps ./internal/domain/... | grep -c openai` → `0`. Sin `go func` en `internal/`. `gofmt -l` limpio. Sin `t.Skip()`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 5 según roadmap (Frontend). **Pendiente de decidir con el humano**: el checklist de Fase 4 no cubre el cableado HTTP (`cmd/api`, handlers chi, `CreatePracticeService` y el handler que consume `PracticeCreated` para invocar `AnalysisService.RunAnalysis`), necesario antes/además del frontend; además falta cablear `PIIHandler` y el timeout por config.

---

## 2026-09-18 — Fase 4: corrección del schema de Structured Outputs (raíz `object`) tras verificación real

**Estado**: Fase 4.2 corregida y verificada contra la API real. `go build/vet/test -race` verdes; `adapters/llm` **100%**. `go run ./cmd/llmcheck` con `gpt-4o-mini` OK.

**Contexto**: la verificación manual del bloque 4.2 falló con `llm unavailable`. Como el adapter oculta el error crudo del proveedor (A5/A8), se expuso temporalmente para diagnosticar: la API devolvió `400 Bad Request` — `Invalid schema for response_format 'fragment_analysis': schema must be a JSON Schema of 'type: "object"', got 'type: "array"'`.

**Causa**: OpenAI Structured Outputs **no acepta un array en la raíz** del schema de `response_format`; exige `type: "object"`. El schema de `Fragment[]` era un array raíz.

**Fix**:
- `adapters/llm/schema.go`: el schema raíz pasa a `{"type":"object","properties":{"fragments":{"type":"array","items":<Fragment>}},"required":["fragments"],"additionalProperties":false}`; se extrae `fragmentItemSchema()` para el fragmento individual.
- `adapters/llm/openai_extractor.go`: el adapter desenvuelve `fragmentEnvelope{Fragments []analysis.Fragment}` y sigue validando con `validateFragments` (4.2.2). El contrato wire (`Fragment[]`) **no** cambia: la envoltura es interna al schema del prompt.
- Tests actualizados (`schema_test.go`, `openai_extractor_test.go`) al formato `{"fragments":[...]}`.

**Verificación real**: `go run ./cmd/llmcheck` → `Fragment[]` válido (`tense_agreement` para "we run" → "we ran"; `lexical_clarification` para "all the park" → "the whole park").

**Verificación estática**: `go build/vet/test -race ./...` + `-tags=integration` OK; `adapters/llm` 100%; `gofmt -l` limpio; sin `t.Skip()`.

**Decisión**: el adapter mantiene el error genérico `LLMUnavailableError` (no filtra el error crudo del proveedor); el diagnóstico se hizo con un cambio temporal, no commiteado.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 4, bloque `4.3` — anonimización previa al LLM (`anonymizer.go`, 4.3.1), timeout + `AnalysisFailed` (4.3.3) y `PIIHandler` (4.3.4).

---

## 2026-09-18 — Fase 4: Structured Outputs + `AnalysisService` (4.2.1–4.2.4)

**Estado**: Fase 4, items 4.2.1–4.2.4 completados. `go build/vet/test -race` verdes (Tier 1 + Tier 3); `adapters/llm` **100%**, nuevo `services` **95.2%**; A1 en 0.

**Hecho**:
- `4.2.1` `adapters/llm/schema.go`: `fragmentSchema()` (JSON Schema estricto de `Fragment[]`, espejo de §5.2) + `fragmentResponseFormat()`; `Extract` ahora envía `response_format` con `strict: true`. Los enums de `code`/`severity` se derivan de las constantes del dominio, así el schema nunca diverge de la taxonomía.
- `4.2.2` `adapters/llm/validate.go`: `validateFragments` exige los campos de texto requeridos no vacíos y enums válidos; cualquier violación → `*domain.LLMUnavailableError` (no se persiste basura).
- `4.2.3`/`4.2.4` `services/analysis_service.go`: `AnalysisService.RunAnalysis` carga la práctica, llama al LLM **fuera** de la transacción, completa el `Analysis` y persiste `Analysis.Save` + `outbox.Append(AnalysisCompleted)` en una única `InTransaction` (COMMIT atómico).
- `ports.LLMExtractor` ampliado con `Model()`/`ModelVersion()` para que el service persista la trazabilidad del proveedor; mocks regenerados con `make mocks`.

**Decisiones**:
- **`Message.Parsed` no existe en openai-go v1.12.0**: se corrige la nota aspiracional del DEVLOG de 4.1.x. El adapter parsea `message.content` como JSON (ya constreñido por el schema estricto) y lo valida en Go.
- **Validación sin librería externa de JSON Schema** (manifiesto §2.1: "sin paquetes externos de validación"): validación semántica en Go (campos requeridos no vacíos + enums). El schema estricto enviado al proveedor es la primera línea; la validación local es el guardrail.
- **`note` requerido en el schema LLM** (`additionalProperties: false` y todos los campos en `required`) porque OpenAI strict mode lo exige; el contrato wire conserva `note` opcional.
- **Puerto `LLMExtractor` con `Model()`/`ModelVersion()`**: única vía limpia para persistir `Analysis.model`/`model_version` sin acoplar el service a OpenAI (A1/A2).
- **Camino de fallo diferido a 4.3.3**: un error del extractor se propaga; no se persiste `Analysis` fallido ni se emite `AnalysisFailed` en este bloque (scope de 4.3.x).
- **Test "fuera de tx"**: el mock del `UnitOfWork` inyecta un context con marker; se asserta que `Extract` lo recibe **sin** marker y `Save`/`Append` **con** marker (prueba directa de 4.2.3/4.2.4).

**Verificación**:
- `go build ./...` · `go vet ./...` · `go vet -tags=integration ./...` · `go test -race -count=1 ./...` · `go test -race -count=1 -tags=integration ./...` → OK.
- `go test -cover ./internal/api/adapters/llm/` → **100.0%**; `./internal/api/services/` → **95.2%**.
- A1: `go list -deps ./internal/domain/... | grep -c openai` → `0`. `gofmt -l` limpio. Sin `t.Skip()`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 4, bloque `4.3` — anonimización previa al LLM (`anonymizer.go`, 4.3.1), timeout + `AnalysisFailed` (4.3.3) y `PIIHandler` (4.3.4).

---

## 2026-09-18 — Fase 4: nuevo `ErrorPatternCode` `lexical_choice` (cambio A12, previo a 4.2.1)

**Estado**: contrato, dominio, prompt y docs alineados. `adapters/llm` 100%; `go build/vet/test -race` verdes; `pnpm generate` idempotente; `pnpm lint` / `pnpm build` OK.

**Contexto**: la verificación manual con `gpt-4o-mini` mostró que el modelo mapeaba colocaciones/léxico ("all the park" → "the whole park") a `word_order`, porque la taxonomía de §4.6 no tenía un código léxico. Decisión del humano: **Opción (b)**, añadir `lexical_choice`.

**Hecho (A12: contrato → `pnpm generate` → código)**:
- `apps/contracts/openapi/api.yaml`: `ErrorPatternCode` enum + `lexical_choice` (tras `false_friend`).
- `pnpm generate` → `gen_types.go` (`LexicalChoice ErrorPatternCode = "lexical_choice"`) y `gen.ts` actualizados; `gen_server.go` sin cambios. Idempotente (sha256 estable en pasadas sucesivas).
- Dominio: constante `ErrorPatternCodeLexicalChoice` + `IsValid()`; test de valores conocidos ampliado.
- Prompt: glosario `lexical_choice: wrong vocabulary choice or collocation that is not a false friend`; `word_order` recortado a "wrong syntactic order"; instrucción explícita de usar `lexical_choice` para vocabulario/colocación.
- Docs: `PRODUCT_DOMAIN §4.6` (nueva fila) y §5.2 (bloque YAML conceptual).

**Verificación real**: `go run ./cmd/llmcheck` → "all the park" ahora se clasifica como **`lexical_choice`** (minor), no `word_order`. ✓

**Verificación estática**: `pnpm generate` idempotente · `pnpm lint` OK (6 warnings conocidos de `operation-4xx-response`) · `pnpm build` 2 successful · `go build/vet/test -race` OK · `gofmt -l` limpio.

**Decisiones**:
- Nombre `lexical_choice` (cubre elección léxica y colocación, sin ambigüedad con `false_friend`).
- **Versionado**: se mantiene `apps/contracts@1.0.0`. El cambio es aditivo y no hay consumidores publicados (misma decisión documentada para `invalid_state` en Fase 1); AP-MR6 (bump minor + tag) se aplicará cuando exista un release real.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 4, bloque `4.2` — Structured Outputs con el JSON Schema de `Fragment[]` (4.2.1), que ya incluye el enum completo con `lexical_choice`, y validación del output (4.2.2).

---

## 2026-09-18 — Fase 4: verificación manual del prompt + runner `cmd/llmcheck` (sobre 4.1.x)

**Estado**: Fase 4, items 4.1.x. Verificación real contra OpenAI (`gpt-4o-mini`) exitosa. `go build/vet/test -race` en verde; `adapters/llm` 100%, `shared/config` 88.9%; A1 sigue en 0.

**Hecho**:
- `internal/shared/config/` (`dotenv.go`, `config.go`): loader `.env` solo-stdlib (`ParseDotEnv`/`LoadDotEnv`; **no** sobreescribe variables ya presentes) + `LoadOpenAIConfig` (`OPENAI_API_KEY`/`OPENAI_MODEL` requeridos; `OPENAI_MODEL_VERSION` cae al alias si falta; `OPENAI_BASE_URL` opcional).
- `cmd/llmcheck/main.go`: runner manual. Lee `.env`, construye `OpenAIExtractor` a través del puerto, envía una práctica (sample embebido o `-input` JSON), imprime `Fragment[]` como JSON. `-show-prompt` imprime el prompt exacto; `-timeout` (60s por defecto).
- `adapters/llm/prompt.go`: `PromptText` exportado (auditoría del prompt sin llamar a la API); prompt reforzado — fragmentos **verbatim**, cobertura total del texto, explicaciones en **español** (la `correction` sigue en inglés), **glosario** de la taxonomía derivado de las constantes del dominio e instrucción de no forzar códigos.
- `apps/backend/.env.example` (tracked) documenta las 4 variables; `.env` sigue gitignoreado.
- **modelVersion**: `OPENAI_MODEL` = alias (`Analysis.model`) + `OPENAI_MODEL_VERSION` = snapshot fechado (`Analysis.model_version`). Añadido al `.env` local (`gpt-4o-mini-2024-07-18`).

**Verificación real**: `go run ./cmd/llmcheck` con `gpt-4o-mini` → `Fragment[]` válido: `source_es`/`user_draft` verbatim, `correction` en inglés, explicaciones en español, `tense_agreement` para "we run" → "we ran".

**Hallazgo (decisión de producto pendiente)**: los problemas de colocación/léxico ("all the park" → "the whole park") se clasifican como `word_order`, porque la taxonomía de §4.6 **no tiene** un código léxico/colocación. El glosario lo excluye explícitamente y aun así el modelo fuerza el código más cercano. Opciones: (a) aceptarlo como limitación conocida; (b) añadir un código `lexical_choice` al contrato (cambio A12: `api.yaml` + enum del dominio + docs). **No se toca el contrato sin aprobación.**

**Decisiones**:
- Env vars con el prefijo estándar `OPENAI_*` (convención del SDK) en lugar del `APP_` de AP-MR4, para que el mismo `.env` alimente al cliente oficial; anotado como desviación.
- `cmd/llmcheck` es una herramienta de desarrollo (no parte de la HTTP API); se commitea según lo acordado.
- Sin `t.Skip`: el runner no es un test; los tests del `cmd` cubren solo el parseo del input.

**Verificación estática**: `go build ./...` · `go vet ./...` · `go test -race -count=1 ./...` → OK. `gofmt -l` limpio. A1: `go list -deps ./internal/domain/... | grep -c openai` → `0`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 4, bloque `4.2` — Structured Outputs con el JSON Schema de `Fragment[]` (4.2.1) y validación del output antes de persistir (4.2.2).

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
