# Dominio del Producto — LangLint (v1.0.0)

> **Documento de Visión de Producto y Especificación del Dominio.**
> Este documento define la razón de ser, los límites y el modelo de dominio de **LangLint (AI-Powered Writing & Grammar Analytics)**, una plataforma de aprendizaje de idiomas mediante escritura productiva contextual con retroalimentación de IA.
>
> Es el **documento autoritativo del dominio** (referenciado por `MANIFEST_MONOREPO.md §8` y `MANIFEST_FRONTEND.md §8`). Condensa *qué* se construye y *por qué* cada decisión técnica obedece a los axiomas de ambos manifiestos. No define *cómo* se construye el código (eso lo hacen los manifiestos).
>
> **Audiencia**: producto, arquitectos, leads técnicos, nuevos devs.
>
> **Prerrequisitos de lectura**: `MANIFEST_MONOREPO.md` (axiomas A1–A12, patrones transversales) y `MANIFEST_FRONTEND.md` (axiomas F1–F12).

---

## Tabla de contenidos

1. [Visión General](#1-visión-general)
2. [Objetivos del Producto](#2-objetivos-del-producto)
3. [Alcance y Límites](#3-alcance-y-límites)
4. [Modelo de Dominio (DDD)](#4-modelo-de-dominio-ddd)
5. [Contrato OpenAPI (Superficie)](#5-contrato-openapi-superficie)
6. [Motor de Análisis Cognitivo (IA)](#6-motor-de-análisis-cognitivo-ia)
7. [Analítica Histórica](#7-analítica-histórica)
8. [Interfaz de Usuario (Frontend)](#8-interfaz-de-usuario-frontend)
9. [Privacidad y Propiedad de los Datos](#9-privacidad-y-propiedad-de-los-datos)
10. [Matriz de Trazabilidad de Axiomas](#10-matriz-de-trazabilidad-de-axiomas)
11. [Definition of Done del MVP](#11-definition-of-done-del-mvp)
12. [Roadmap](#12-roadmap)
13. [Glosario del Dominio](#13-glosario-del-dominio)
14. [Documentos Autoritativos](#14-documentos-autoritativos)

---

## 1. Visión General

### 1.1 El Problema

Los métodos tradicionales de aprendizaje de idiomas (tarjetas de memoria, *flashcards* tipo Anki) son excelentes para la **memorización pasiva** y el recuerdo activo a corto plazo, pero fallan en la **aplicación contextual y activa**. Cuando un estudiante intenta redactar textos formales usando vocabulario nuevo —por ejemplo verbos irregulares avanzados como *sweep*, *kneel*, *sling*, *strike*— comete errores sintácticos complejos que las apps de flashcards **no pueden detectar ni corregir**:

- Confusión entre **infinitivos y verbos conjugados**.
- Uso erróneo de la **voz pasiva**.
- **Traducción literal** de modismos e idiomática.
- Carencia de un **ciclo de retroalimentación profunda** que explique el *"por qué"* gramatical detrás de un error.

El resultado es un estudiante que *reconoce* una palabra pero no es capaz de *producir* un texto correcto con ella. Existe un vacío entre el reconocimiento (passive recall) y la producción activa (active production).

### 1.2 La Solución

**LangLint** es una plataforma web inteligente diseñada para ingenieros y estudiantes avanzados que aprenden idiomas mediante la **escritura productiva contextual**. El flujo central es:

1. El usuario define un **conjunto de objetivos gramaticales** (p. ej. una lista de verbos irregulares a practicar).
2. Escribe un **texto base en español** aplicando esos objetivos.
3. Redacta su **traducción experimental al inglés** (el "borrador").
4. Una **Inteligencia Artificial** actúa como profesor nativo y editor profesional: desmenuza el texto **fragmento por fragmento**, corrige, explica las reglas subyacentes y clasifica los errores.
5. Un **motor de analíticas** rastrea los errores recurrentes del usuario a lo largo del tiempo, eliminando puntos ciegos.

### 1.3 Propuesta de Valor

| Para | Valor entregado |
|---|---|
| **El estudiante** | Pasar de reconocer una palabra a escribirla correctamente en contexto; entender el *porqué* gramatical, no solo ver la corrección. |
| **El producto (portafolio)** | Demostración de arquitectura de software moderna: monorepo polyglot (Go + Next.js), contratos OpenAPI como Single Source of Truth, y Domain-Driven Design estricto. |

---

## 2. Objetivos del Producto

1. **Acelerar la competencia activa** — Pasar de la memorización pasiva (reconocer) a la producción activa (escribir un texto formal sin errores gramaticales).
2. **Feedback en profundidad (Beyond Correction)** — No limitarse a mostrar el texto corregido, sino proveer explicaciones gramaticales profundas (reglas generales, excepciones y aplicabilidad a otros casos).
3. **Eliminación de puntos ciegos (Error Tracking)** — Medir y visualizar la frecuencia de los errores del usuario (p. ej. fallas recurrentes con preposiciones o posesivos) para prevenir que se repitan.
4. **Excelencia en ingeniería (Portfolio-Grade)** — Servir como demostración práctica de arquitectura moderna bajo los axiomas de los manifiestos.

### 2.1 Criterios de éxito (KPIs)

| Métrica | Objetivo | Cómo se mide |
|---|---|---|
| Cobertura del dominio (`internal/domain/`) | 100% | `go test -cover` (gate CI, A10) |
| Cobertura de services de aplicación | ≥90% | `go test -cover` (gate CI, A10) |
| Fragmentos analizados sin error de contrato | 100% | Structured Outputs validados contra el JSON Schema del contrato |
| Trazabilidad del flujo transaccional | 100% | Test Tier 3 del outbox relay (testcontainers) |
| Cache hit ratio de Turborepo | >80% (1ª semana) | Remote cache (AP-MR7) |

---

## 3. Alcance y Límites

Para mantener el proyecto acotado y evitar la parálisis por análisis (MVP acotado pero de alta calidad técnica), el sistema se divide en cuatro dimensiones.

### 3.1 Dimensiones del Sistema

| Dimensión | Responsabilidad | Corresponde a |
|---|---|---|
| **Autoría (Input)** | Ingreso de la lista de verbos/reglas objetivo, texto base en español y borrador de traducción manual del usuario. | BC `practice` |
| **Análisis Cognitivo (AI Engine)** | Procesamiento estructurado vía LLM con Structured Outputs (JSON mode), desglose fragmento por fragmento. | BC `analysis` + puerto `LLMExtractor` |
| **Analítica Histórica (Dashboard)** | Conteo y clasificación de patrones de error; métricas de progreso temporal. | BC `analytics` |
| **Interfaz (Frontend UX)** | Vista de tres columnas (Diff View estilo IDE), paneles interactivos de explicación con tooltips y alertas de errores recurrentes. | `apps/frontend/` |

### 3.2 Non-Goals (fuera de alcance del MVP)

- **Multi-tenancy B2B** — `organization/` y `billing/` del template genérico del manifiesto **no aplican**. LangLint es single-user. (Ver §4.2.)
- **Autenticación multi-usuario real** — no hay login, JWT ni API keys en el MVP. Existe una identidad portable mínima para cumplir A9.
- **Corrección en tiempo real (streaming)** — el análisis es asíncrono vía outbox/relay, no streaming SSE.
- **Soporte multi-idioma** — el MVP fija el par **español → inglés**. El diseño del dominio no lo impide a futuro, pero no se implementa.
- **Gamificación, planes de pago, compartir contenido** — fuera del alcance.

---

## 4. Modelo de Dominio (DDD)

> **Ubicación canónica**: el código del dominio vive en `apps/backend/internal/domain/` (raíz pura, solo stdlib, A1) y sus sub-paquetes por bounded context. Las interfaces (puertos) viven en `apps/backend/internal/api/ports/`; las implementaciones (adaptadores) en `apps/backend/internal/api/adapters/`.

### 4.1 Bounded Contexts

> **Nota de adaptación**: el manifiesto monorepo lista `identity/`, `organization/`, `billing/` y `audit/` como *ejemplos* pensados para un SaaS B2B multi-tenant. LangLint es una herramienta **personal de estudio**, por lo que su modelo de dominio define **bounded contexts propios**, preservando íntegramente los axiomas (aislamiento A3, pureza A1, errores A5, eventos A4). `organization/` y `billing/` se **descartan** por no tener correspondencia en el producto.

| Bounded Context | Responsabilidad | Agregado raíz |
|---|---|---|
| `identity/` | Identidad portable del usuario (A9). Propietario de los datos de estudio. | `User` |
| `practice/` | El ejercicio de escritura: texto base, borrador del usuario, reglas objetivo. | `Practice` |
| `analysis/` | El análisis fragmentado generado por la IA sobre una práctica. | `Analysis` |
| `analytics/` | Materialización de patrones de error y progreso temporal. | `ErrorMetric`, `ProgressMetric` |

**Regla de aislamiento (A3)**: los cuatro bounded contexts **no se importan entre sí**. Se comunican vía puertos en `internal/api/ports/` y eventos de dominio append-only. `analysis/` referencia a `Practice` por su `ID` (tipo primitivo `domain.ID`), nunca importando el paquete `practice/`.

> **Evolución futura (fuera del MVP)**: un quinto bounded context, **`tutor/`** (o `curriculum/`), se incorporará en el roadmap post-MVP como un **tutor adaptativo** que consume los agregados de `analytics/` para generar sesiones de estudio personalizadas (ver §12.1). Gracias al diseño por eventos (A3, A4), esta incorporación **no romperá** ninguno de los cuatro contexts existentes: `tutor/` será un módulo desacoplado que se suscribe al Domain Event Bus (outbox).

### 4.2 Entidades y Agregados

#### 4.2.1 `identity/` — `User`

| Campo | Tipo | Notas |
|---|---|---|
| `ID` | `domain.ID` (UUID v7) | Identidad portable, **sin** `tenant_id` (AP1). Generado vía `domain.NewID()`. |
| `Email` | `Email` (value object) | PII (A8). |
| `CreatedAt` | `time.Time` | |

- La identidad portable es el dueño global de los datos (A9). No hay tenants: el aislamiento por fila del manifiesto (A6) se simplifica a "un único usuario", pero se **mantiene** la estructura para no cerrar la puerta a multi-usuario futuro.

#### 4.2.2 `practice/` — `Practice`

| Campo | Tipo | Notas |
|---|---|---|
| `ID` | `domain.ID` (UUID v7) | |
| `UserID` | `domain.ID` | Propietario (identity portable). |
| `SourceText` | `SourceText` (value object) | Texto base en español. |
| `DraftText` | `DraftText` (value object) | Traducción experimental del usuario. |
| `TargetRules` | `[]TargetRule` | Lista de verbos/reglas objetivo. |
| `Status` | `PracticeStatus` | `draft` → `analyzing` → `completed` / `failed`. |
| `CreatedAt` / `UpdatedAt` | `time.Time` | |

- **Invariantes**: `SourceText` y `DraftText` no vacíos; `TargetRules` con al menos un elemento; transición de estado válida (`draft → analyzing → completed|failed`).

#### 4.2.3 `analysis/` — `Analysis`

| Campo | Tipo | Notas |
|---|---|---|
| `ID` | `domain.ID` (UUID v7) | |
| `PracticeID` | `domain.ID` | Referencia por ID (no importa `practice/`). |
| `Fragments` | `[]Fragment` | Análisis fragmento por fragmento. |
| `Model` / `ModelVersion` | `string` | Trazabilidad del proveedor LLM. |
| `Status` | `AnalysisStatus` | `pending` / `completed` / `failed`. |
| `CreatedAt` | `time.Time` | |

- **Invariantes**: un `Analysis` pertenece a una única `Practice`; `Fragments` no vacío cuando `Status == completed`.

#### 4.2.4 `analytics/` — `ErrorMetric` y `ProgressMetric`

| Agregado | Campos | Notas |
|---|---|---|
| `ErrorMetric` | `UserID`, `ErrorPattern.Code`, `Window`, `Count`, `LastSeenAt` | Materializado por el handler de `AnalysisCompleted`. |
| `ProgressMetric` | `UserID`, `Window`, `TotalFragments`, `ErrorCount`, `Accuracy` | Serie temporal de progreso. |

### 4.3 Value Objects

| Value Object | Contexto | Descripción |
|---|---|---|
| `Email` | `identity` | Email validado. |
| `SourceText` | `practice` | Texto en español (no vacío, longitud acotada). |
| `DraftText` | `practice` | Borrador en inglés (no vacío). |
| `TargetRule` | `practice` | Verbo/regla objetivo (`verb`, `tense`, `note`). |
| `PracticeStatus` | `practice` | Enumerado de ciclo de vida. |
| `Fragment` | `analysis` | Unidad atómica del análisis (ver §5.2). |
| `ErrorPattern` | `analysis`/`analytics` | Clasificación de un error (ver §4.6). |
| `Window` | `analytics` | Ventana temporal de agregación (`day`/`week`/`month`). |

### 4.4 Domain Events

> Los eventos se definen en `apps/backend/internal/domain/events.go` (tipos puros e inmutables, A4/A1). Se publican vía el **outbox genérico** dentro de la misma transacción que el cambio de estado (AP7/AP8), y un relay los publica post-commit (ver §4.7).

| Evento | Productor | Consumidor | Payload |
|---|---|---|---|
| `IdentityIssued` | `identity` | `practice` (asocia usuario) | `UserID` |
| `PracticeCreated` | `practice` | `analysis` (dispara análisis) | `PracticeID`, `UserID` |
| `AnalysisCompleted` | `analysis` | `analytics` (materializa métricas) | `AnalysisID`, `PracticeID`, `UserID`, `[]ErrorPattern` |
| `AnalysisFailed` | `analysis` | `practice` (marca `failed`) | `AnalysisID`, `PracticeID`, `Reason` |

### 4.5 Catálogo de Errores de Dominio (A5)

> Los errores son ciudadanos del dominio. Se definen en `apps/backend/internal/domain/errors.go`. Ningún service retorna `error` genérico.

| Error de dominio | Cuándo | HTTP status | `code` (wire) |
|---|---|---|---|
| `ValidationError` | Entrada inválida (texto vacío, `TargetRules` vacío, JSON malformado) | 422 | `validation_error` |
| `NotFoundError` | `Practice`/`Analysis` no existe | 404 | `not_found` |
| `AnalysisPendingError` | Se solicita el resultado antes de que termine el análisis | 409 | `analysis_pending` |
| `InvalidStateError` | Operación no válida para el estado actual (editar una práctica no `draft`; borrar una `analyzing`) | 409 | `invalid_state` |
| `AnalysisFailedError` | El análisis terminó con fallo (el cliente lo observa vía `analysis.status == "failed"`) | — (sin 4xx; ver §5.1) | `analysis_failed` (reservado) |
| `LLMUnavailableError` | Proveedor LLM caído / timeout | 503 | `llm_unavailable` |
| `InternalError` | Fallo de infraestructura inesperado (DB, dependencia); el adaptador envuelve el error crudo | 500 | `internal` (reservado) |

> **Regla (AP3)**: nunca reutilizar `ValidationError` para un concepto distinto. Cada error con un código HTTP distinto tiene su propio tipo.

### 4.6 Clasificación de `ErrorPattern`

> El `ErrorPattern` es el **núcleo del valor analítico**: la taxonomía de errores que el LLM asigna a cada fragmento y que alimenta el dashboard.

| `code` | Significado |
|---|---|
| `infinitive_conjugation` | Confusión entre infinitivo y verbo conjugado. |
| `passive_voice_misuse` | Uso erróneo de la voz pasiva. |
| `idiom_literal_translation` | Traducción literal de un modismo/idiomática. |
| `preposition_infinitive` | Preposición + infinitivo mal formado. |
| `pronoun_possession` | Confusión pronombre / posesivo. |
| `false_friend` | Falso amigo léxico. |
| `lexical_choice` | Elección de vocabulario o colocación incorrecta (no un falso amigo). |
| `word_order` | Orden sintáctico incorrecto. |
| `tense_agreement` | Concordancia de tiempos verbales. |

Cada `ErrorPattern` además lleva `severity` (`minor` | `moderate` | `critical`) y `note` (explicación breve). La enumeración es parte del **contrato OpenAPI** (ver §5.2), no un string libre.

### 4.7 Flujos Transaccionales (Unit of Work + Outbox)

> El service **nunca** inicia una transacción SQL (AP8). Usa el puerto `UnitOfWork`; el adaptador `PostgresUnitOfWork` es el único que toca `pgxpool.Pool`. El LLM (llamada lenta) se ejecuta **fuera** de la transacción.

#### Flujo 1 — Crear práctica y disparar análisis

```
[1] CreatePracticeService:
[2]   uow.InTransaction(ctx, func(txCtx) {
[3]     practiceRepo.Save(txCtx, practice)          // status = draft
[4]     outbox.Append(txCtx, PracticeCreated{...})  // misma transacción
[5]   })                                            // COMMIT atómico
[6] OutboxRelay publica PracticeCreated (post-commit)
[7] AnalysisService (handler): llama llmExtractor.Extract(ctx, ...)  // FUERA de tx
```

#### Flujo 2 — Completar análisis y materializar métricas

```
[1] AnalysisService (tras recibir el resultado del LLM):
[2]   uow.InTransaction(ctx, func(txCtx) {
[3]     analysisRepo.Save(txCtx, analysis)           // status = completed
[4]     outbox.Append(txCtx, AnalysisCompleted{...}) // misma transacción
[5]   })                                             // COMMIT atómico
[6] OutboxRelay publica AnalysisCompleted (post-commit)
[7] AnalyticsService (handler): errorMetricRepo.Upsert(ctx, patterns)
```

**Reglas (AP7, §5.1 del manifiesto)**:
1. Nunca `dispatcher.Dispatch(...)` directo desde un service dentro de un `UnitOfWork`. Siempre `outbox.Append(...)`.
2. El LLM se invoca **fuera** de la transacción (es una llamada de red lenta; mantenerla dentro violaría el timeout y el patrón).
3. Los handlers son **idempotentes** (pueden ejecutarse dos veces sin efecto adverso).

---

## 5. Contrato OpenAPI (Superficie)

> **SSOT (A12, F1)**: todo wire format nace en `apps/contracts/openapi/api.yaml` y se genera con `oapi-codegen` (Go) y `openapi-typescript` (TS). Nadie edita tipos de wire a mano (AP-MR2, AP-F1).

### 5.1 Endpoints

| Método | Ruta | Descripción | Respuesta |
|---|---|---|---|
| `POST` | `/v1/practices` | Crea una práctica (source + draft + target rules). | `201` `Practice` |
| `GET` | `/v1/practices` | Lista las prácticas del usuario (paginado `?limit&offset`). | `200` `PracticeList` (`{items, total}`) |
| `GET` | `/v1/practices/{practiceId}` | Obtiene práctica + análisis embebido. | `200` `PracticeDetail` |
| `PATCH` | `/v1/practices/{practiceId}` | Edita parcialmente una práctica en `draft`. | `200` `Practice` · `409 invalid_state` si no está en `draft` |
| `DELETE` | `/v1/practices/{practiceId}` | Elimina (soft-delete) una práctica. | `204` · `409 invalid_state` si está `analyzing` |
| `POST` | `/v1/practices/{practiceId}/analyze` | Dispara el análisis asíncrono. | `202` (polling) + `Retry-After` |
| `GET` | `/v1/analytics/error-patterns` | Agregados de patrones de error (`?window=day\|week\|month`). | `200` `ErrorPatternStats` |
| `GET` | `/v1/analytics/progress` | Serie temporal de progreso (`?window=day\|week\|month`). | `200` `ProgressSeries` |
| `GET` | `/v1/me/data/export` | Portabilidad de datos (A9, GDPR Art. 15+20). | `200` `DataExport` |
| `DELETE` | `/v1/me/data` | Derecho al olvido, gracia 30 días (A9, GDPR Art. 17). | `202` |
| `GET` | `/v1/me/access-log` | Auditoría del usuario (A9), paginada (`?limit&offset`). | `200` `AccessLog` (`{items, total}`) |

> **Idempotencia (AP4)**: `POST /v1/practices/{id}/analyze` documenta en el contrato que `409 analysis_pending` es un **éxito idempotente** (ya está en análisis); el cliente lo trata como éxito (F5).
>
> **Fallos de análisis (no usan 4xx)**: un análisis terminado con fallo se comunica con `200` y `analysis.status == "failed"` en `PracticeDetail`; **no** se devuelve `422 analysis_failed` (el `422` queda reservado a errores de validación del input). El código `analysis_failed` permanece en la taxonomía de errores del contrato como valor reservado.
>
> **Paginación y ventana**: los listados usan `limit` (1..100, por defecto 20) y `offset` (por defecto 0) y devuelven `{items, total}`. Los agregados de analítica aceptan `window` (`day|week|month`, por defecto `week`).
>
> **Ciclo de vida (CRUD de `Practice`)**: `Practice` es editable solo en `draft` (`PATCH`, parcial); el borrado es **soft-delete** (`DELETE` → `204`, el job `purge-raw-data` lo materializa después, A8). Los conflictos de estado devuelven `409 invalid_state`.

### 5.2 Schema `FragmentAnalysis` (el corazón del producto)

> El `FragmentAnalysis` estructura formalmente lo que antes era un prompt de texto plano. Es el JSON Schema que el LLM debe devolver en modo Structured Outputs.

```yaml
# apps/contracts/openapi/api.yaml (extracto conceptual)
components:
  schemas:
    Fragment:
      type: object
      required:
        - source_es
        - user_draft
        - correction
        - target_verb_review
        - lexical_clarification
        - grammar_explanation
        - error_patterns
      properties:
        source_es:            { type: string }   # Frase base en español
        user_draft:           { type: string }   # Borrador del usuario (inglés)
        correction:           { type: string }   # Corrección directa
        target_verb_review:   { type: string }   # Revisión del verbo objetivo
        lexical_clarification:{ type: string }   # Aclaración léxica
        grammar_explanation:  { type: string }   # Regla gramatical profunda
        error_patterns:
          type: array
          items: { $ref: '#/components/schemas/ErrorPattern' }

    ErrorPattern:
      type: object
      required: [code, severity]
      properties:
        code:
          type: string
          enum:
            - infinitive_conjugation
            - passive_voice_misuse
            - idiom_literal_translation
            - preposition_infinitive
            - pronoun_possession
            - false_friend
            - lexical_choice
            - word_order
            - tense_agreement
        severity:
          type: string
          enum: [minor, moderate, critical]
        note: { type: string }

    Analysis:
      type: object
      required: [id, practice_id, model, model_version, status, fragments]
      properties:
        id:           { type: string, format: uuid }
        practice_id:  { type: string, format: uuid }
        model:        { type: string }
        model_version:{ type: string }
        status:       { type: string, enum: [pending, completed, failed] }
        fragments:
          type: array
          items: { $ref: '#/components/schemas/Fragment' }
```

### 5.3 Reglas operativas del contrato

1. **Cambio de wire format** → primero `api.yaml`, luego `pnpm generate`, luego usar el tipo generado (A12).
2. **Cambio breaking** → bump major de `apps/contracts/` + entrada BREAKING en `CHANGELOG.md` + tag `contracts-v2.0.0` (AP-MR6).
3. `gen_*.go` y `gen.ts` están **commiteados** para builds offline; el lint rule `contract_drift` rechaza ediciones a mano.
4. Un campo documentado en el contrato **debe existir** en el modelo de dominio (AP5) o declararse gap explícito en `docs/API_CONTRACT.md`.

---

## 6. Motor de Análisis Cognitivo (IA)

### 6.1 Puerto `LLMExtractor`

> Definido como puerto en `apps/backend/internal/api/ports/` (interfaz). El dominio **no** conoce al proveedor (A1). Cambiar de OpenAI a Anthropic no toca `internal/domain/`.

```go
// apps/backend/internal/api/ports/llm_extractor.go (conceptual)
type LLMExtractor interface {
    // Extract devuelve el análisis estructurado de la práctica.
    Extract(ctx context.Context, req ExtractRequest) ([]analysis.Fragment, error)
}
```

| Adapter | Proveedor | Notas |
|---|---|---|
| `OpenAIExtractor` | `openai-go` (v1.12.0) | Structured Outputs (JSON Schema estricto) nativo. |

### 6.2 Structured Outputs (JSON Mode)

- El LLM devuelve `Fragment[]` conforme al schema de §5.2. El adapter **valida** el JSON contra el schema antes de persistir; un output inválido se trata como `LLMUnavailableError` (no se persiste basura).
- El prompt se construye a partir del `Practice` (source + draft + target rules), **anonimizado** previamente.

### 6.3 Anonimización, Contexto y Timeouts

| Preocupación | Mecanismo | Axioma/AP |
|---|---|---|
| **Anonimización previa al LLM** | `anonymizer.go` elimina emails, teléfonos y nombres propios antes de enviar a servicios externos. | A8 |
| **Contexto cancelable** | Toda goroutine recibe `context.Context` y chequea `ctx.Done()` (regla `goroutine_context`). | AP6 |
| **Timeouts y degradación** | Timeout de la llamada al LLM; ante fallo se emite `AnalysisFailed` y el cliente ve un mensaje de UX claro, nunca el error crudo. | Axioma UX (F5) |
| **No loguear contenido crudo** | El `PIIHandler` descarta texto libre >100 chars y patrones PII. | A8 |

---

## 7. Analítica Histórica

### 7.1 Modelo de agregación

- El handler de `AnalysisCompleted` (en `analytics`) hace `Upsert` de `ErrorMetric` por `(UserID, ErrorPattern.Code, Window)`, incrementando `Count` y actualizando `LastSeenAt`.
- `ProgressMetric` agrega `TotalFragments`, `ErrorCount` y `Accuracy = 1 - ErrorCount/TotalFragments` por ventana.

### 7.2 Jobs del Provisioner

> El provisioner (`apps/backend/cmd/provisioner`) ejecuta jobs batch que complementan la agregación incremental:

| Job | Propósito | Axioma |
|---|---|---|
| `refresh-aggregates` | Reconciliar métricas de analytics desde la fuente de verdad (reconstruir agregados ante drift). | A4 (historia reconstruible) |
| `purge-raw-data` | Purga programada de datos crudos de práctica (retención limitada). | A8 (retención limitada) |
| `execute-deletions` | Ejecuta el derecho al olvido tras la gracia de 30 días. | A9 (Art. 17) |

### 7.3 Dashboard

- `GET /v1/analytics/error-patterns` devuelve la frecuencia clasificada de errores (p. ej. *preposition + infinitive* recurrente).
- `GET /v1/analytics/progress` devuelve la serie temporal para visualizar la mejora.

---

## 8. Interfaz de Usuario (Frontend)

> El frontend vive en `apps/frontend/` (Next.js 14+). Cumple `MANIFEST_FRONTEND.md` en su totalidad.

### 8.1 Vista principal: Diff View de tres columnas (estilo IDE / Code Review)

| Columna | Contenido |
|---|---|
| **1 — Español** | Texto base fuente (fragmentos). |
| **2 — Tu borrador** | Traducción experimental del usuario, con errores resaltados. |
| **3 — Corrección IA** | Corrección del LLM con diff resaltado (rojo/verde). |

- **Paneles interactivos**: cada fragmento expandible muestra `target_verb_review`, `lexical_clarification` y `grammar_explanation` con tooltips.
- **Alertas de errores recurrentes**: banner que señala el `ErrorPattern` más frecuente del usuario.

### 8.2 Mapeo a los axiomas del frontend

| Axioma | Cómo se cumple |
|---|---|
| F1 | Tipos de wire solo desde `gen.ts` (openapi-typescript). |
| F2 | Dependencias `app → features → components → lib → types`. |
| F7 | Estado del servidor (prácticas, analíticas) en TanStack Query; estado efímero de UI en Zustand. |
| F8 | Formularios con React Hook Form + Zod, schemas derivados del contrato. |
| F9 | Polling de estado de análisis con optimistic update + rollback. |
| F10 | Vitest (unit) → RTL (componentes) → Playwright (e2e). |
| F12 | RSC para contenido estático, client components para el diff interactivo. |

---

## 9. Privacidad y Propiedad de los Datos

> Aunque el contenido de las prácticas son textos de estudio, el sistema se estructura asumiendo que los **datos personales (PII)** se tratan bajo el principio de mínimo privilegio (A8) y que el **usuario es dueño de sus datos** (A9).

| Principio | Mecanismo | Axioma |
|---|---|---|
| **Anonimización previa al LLM** | `anonymizer.go` elimina PII antes de enviar a servicios externos. | A8 |
| **Pseudonimización en analytics** | HMAC-SHA256 sobre identificadores antes de materializar. | A8 |
| **Retención limitada** | Datos crudos en tablas dedicadas con purga programada (`purge-raw-data`). | A8 |
| **Guard runtime** | `PIIHandler` descarta logs con patrones PII. | A8 |
| **Portabilidad** | `GET /v1/me/data/export` (GDPR Art. 15+20). | A9 |
| **Derecho al olvido** | `DELETE /v1/me/data` con gracia de 30 días. | A9 |
| **Auditoría del usuario** | `GET /v1/me/access-log` (eventos append-only). | A9, A4 |

---

## 10. Matriz de Trazabilidad de Axiomas

> Demuestra cómo cada axioma de los manifiestos se concreta en LangLint.

### 10.1 Monorepo (A1–A12)

| Axioma | Traducción en LangLint |
|---|---|
| **A1** Pureza del dominio | `practice/`, `analysis/`, `analytics/`, `identity/` solo importan stdlib. El proveedor LLM es un puerto (`LLMExtractor`), no una dependencia del dominio. |
| **A2** Dependencias hacia adentro | `cmd → adapters → services → ports → domain`. |
| **A3** Bounded contexts aislados | `practice` ↔ `analysis` ↔ `analytics` no se importan entre sí; se comunican vía eventos. |
| **A4** Append-only donde vive la verdad | `analysis` y métricas de analytics son historia reconstruible; `refresh-aggregates` reconstruye sin UPDATE destructivo. |
| **A5** Errores del dominio | `ValidationError`, `NotFoundError`, `AnalysisPendingError`, `LLMUnavailableError`, etc. |
| **A6** Multi-tenancy por fila | Simplificado a single-user, estructura conservada para futuro. |
| **A7** IDs del dominio | UUID v7 vía `domain.NewID()` (stdlib). |
| **A8** PII inviolable | Anonimización + pseudonimización + retención limitada + `PIIHandler`. |
| **A9** Usuario dueño de sus datos | Endpoints `export` / `delete` / `access-log`. |
| **A10** Tests como gate | Dominio 100%, services ≥90%, adapters ≥70%; outbox relay cubierto en Tier 3. |
| **A11** Monolito lógico | Un solo `go.mod`; fronteras por lint y convención. |
| **A12** Contratos SSOT | `api.yaml` único; `FragmentAnalysis` y `ErrorPattern` como schemas del contrato. |

### 10.2 Frontend (F1–F12)

| Axioma | Traducción en LangLint |
|---|---|
| **F1** Contrato SSOT del cliente | Diff view renderiza `Fragment` desde `gen.ts`. |
| **F2** Dependencias hacia adentro | Capas `app/features/components/lib/types`. |
| **F3** Sin imports cruzados | El frontend consume el backend solo vía HTTP (cliente generado). |
| **F4** PII mínimo privilegio | No loguear textos de práctica; pseudonimizar en analytics. |
| **F5** Errores → UX | `503 llm_unavailable` → "El motor de análisis no responde, intenta de nuevo"; `409 analysis_pending` → tratado como éxito. |
| **F6** Accesibilidad por defecto | Diff view navegable por teclado, contraste AA, `aria` en tooltips. |
| **F7** Server/client state separado | TanStack Query (prácticas, analíticas) vs Zustand (UI efímera). |
| **F8** Formularios contra contrato | Zod inferido de `gen.ts`. |
| **F9** Optimistic UI | Estado `analyzing` optimista con rollback. |
| **F10** Tests gate | Pirámide Vitest → RTL → Playwright. |
| **F11** Backend autoridad | El cliente no reimplementa reglas de análisis/clasificación. |
| **F12** Rendimiento | RSC + streaming + Suspense para cargas de análisis. |

---

## 11. Definition of Done del MVP

Para considerar el proyecto listo para el portafolio:

- [ ] **Monorepo funcional** orquestado con Turborepo, con separación limpia entre `apps/contracts`, `apps/backend` (Go) y `apps/frontend` (Next.js).
- [ ] **Pipeline de generación automática** operativo (`pnpm generate` actualiza tipos de Go y TypeScript de forma idéntica).
- [ ] **Backend con Arquitectura Hexagonal Adaptada**: dominio de `practice`/`analysis`/`analytics` 100% cubierto, Unit of Work transaccional, outbox relay funcional.
- [ ] **Integración con IA estructurada**: `LLMExtractor` devuelve `Fragment[]` validados contra el schema, fragmentados limpios y precisos.
- [ ] **Frontend interactivo de nivel profesional**: vista de diff de tres columnas y analíticas de errores históricos.
- [ ] **Cumplimiento A9**: endpoints de export/delete/access-log operativos.

---

## 12. Roadmap

| Paso | Entregable | Manifiesto |
|---|---|---|
| **1. Fundaciones** | Inicializar monorepo (Turborepo + pnpm workspaces) + contrato OpenAPI fundacional (`api.yaml` con `FragmentAnalysis`/`ErrorPattern`). | A11, A12, AP-MR1..9 |
| **2. Dominio puro** | `internal/domain/` con `identity`, `practice`, `analysis`, `analytics`, errores y eventos. 100% cobertura. | A1, A3, A5, A7, A10 |
| **3. Puertos y adaptadores** | `ports/` (`LLMExtractor`, repos, UoW, outbox) + adaptadores Postgres (testcontainers). | A2, A4, A6, AP8 |
| **4. Motor de IA** | `OpenAIExtractor` con Structured Outputs + anonimización + timeouts. | A8, AP6 |
| **5. Frontend** | Next.js: vista diff 3 columnas + dashboard de analíticas. | F1–F12 |
| **6. Provisioner y privacidad** | Jobs `refresh-aggregates`, `purge-raw-data`, `execute-deletions` + endpoints A9. | A8, A9 |
| **7. Hardening** | CI/CD (turbo remote cache, gosec, govulncheck), docs, README de portafolio. | AP-MR7, §6.5 |

### 12.1 Roadmap Post-MVP — Tutor Adaptativo (Spaced Repetition + Inteligencia de Aprendizaje)

> **Evolución natural del producto**, en el radar a futuro. No se implementa en el MVP; su diseño se anticipa para que su incorporación sea aditiva, nunca disruptiva.

**La idea**: transformar LangLint de una "herramienta de corrección de textos" a un **Tutor de Inglés Personalizado y Adaptativo**, un sistema de Spaced Repetition e inteligencia de aprendizaje basado en los **errores reales** del propio usuario.

#### El flujo a futuro

1. **Hoy (MVP)**: el usuario escribe un texto → la IA corrige fragmento por fragmento → detecta que falló en la regla *Preposition + Infinitive*.
2. **A futuro**: `analytics/` agrupa y detecta los puntos débiles históricos. La IA analiza ese perfil de errores y responde proactivamente:
   > *"He notado que en tus últimas 5 prácticas has fallado sistemáticamente en las preposiciones antes de gerundios. Voy a generarte una sesión de estudio profunda sobre esto, con teoría resumida, 3 trampas comunes que sueles cometer y 5 ejercicios interactivos personalizados."*

Es decir, el sistema deja de *solo corregir* lo que el usuario escribe hoy, y pasa a crear un **plan de estudio dinámico** a partir de su propia fricción de aprendizaje.

#### Cómo encaja con la arquitectura actual

| Aspecto | Decisión |
|---|---|
| **Nuevo Bounded Context** | `tutor/` (o `curriculum/`) — nuevo sub-paquete en `internal/domain/`, aislado por A3. |
| **Fuente de datos** | Consume los **agregados de `analytics/`** (perfil de `ErrorPattern` por ventana), no los datos crudos. |
| **Comunicación** | Vía el **Domain Event Bus (outbox)**: `analytics/` emite un evento (p. ej. `WeaknessDetected`) que `tutor/` consume para generar la sesión. |
| **Contenido generado** | Sesión de estudio = teoría resumida + trampas comunes + ejercicios interactivos, generada por IA (Structured Outputs, mismo patrón del `LLMExtractor`). |
| **Impacto en el MVP** | **Cero rupturas**. Se agrega un módulo desacoplado que se suscribe a eventos; los four contexts existentes no cambian. |

#### Anticipación en el contrato

- La taxonomía de `ErrorPattern` (§4.6) se diseñó desde el inicio para ser **la señal de entrada** del tutor (cada `code` es un punto débil detectable).
- `ErrorMetric`/`ProgressMetric` (§4.2.4) son el **input agregado** que el tutor necesita; ya se materializan en el MVP.
- El patrón outbox (§4.7) es **reusable** para el evento `WeaknessDetected` sin tocar el relay ni el dispatcher existentes.

**Regla de oro**: el MVP **no** implementa el tutor. Solo deja el terreno preparado (taxonomía, agregados, bus de eventos) para que el Roadmap Post-MVP lo incorpore como un paso aditivo.

---

## 13. Glosario del Dominio

| Término | Significado |
|---|---|
| **Práctica** | Ejercicio de escritura: texto base (es) + borrador del usuario (en) + reglas objetivo. |
| **Fragmento** | Unidad atómica del análisis (una frase) con corrección y explicación. |
| **Target Rule** | Verbo/regla gramatical que el usuario se propone practicar (p. ej. verbo irregular). |
| **Error Pattern** | Clasificación taxonómica de un error (preposición, posesivo, falso amigo…). |
| **Structured Outputs** | Modo del LLM que garantiza salida JSON conforme a un schema estricto. |
| **Active Production** | Capacidad de producir texto correcto (opuesto a reconocimiento pasivo). |
| **Diff View** | Vista de comparación estilo IDE entre el borrador y la corrección. |
| **Beyond Correction** | Feedback que explica el *porqué* gramatical, no solo la corrección. |

---

## 14. Documentos Autoritativos

Este documento **condensa** pero **nunca contradice** los documentos canónicos. Si hay conflicto, la jerarquía es:

| Pregunta | Documento autoritativo |
|---|---|
| ¿Cómo se orquesta el monorepo (workspaces, turbo)? | `MANIFEST_MONOREPO.md` |
| ¿Cómo se construye el backend Go (hexagonal, UoW, outbox)? | `MANIFEST_MONOREPO.md` |
| ¿Cómo se construye el frontend Next.js? | `MANIFEST_FRONTEND.md` |
| ¿Cuál es el contrato HTTP (SSOT del wire)? | `apps/contracts/openapi/api.yaml` (a crear en Paso 1) |
| ¿Cuál es el dominio del producto? | `docs/PRODUCT_DOMAIN.md` (este documento) |
| ¿Cuál es el contrato arquitectónico global? | `docs/ARCHITECTURE.md` (a crear) |

---

**Versión**: 1.0.0
**Mantenedor**: equipo de producto / arquitectura.
**Última revisión**: 2026-09-15.
**Próxima revisión**: tras cada release mayor o cuando cambie el modelo de dominio (nuevos bounded contexts, eventos o taxonomía de errores).
