# API_CONTRACT.md — Gaps Conocidos Contrato ↔ Dominio

> Registro de **gaps conocidos** entre el contrato OpenAPI
> ([`apps/contracts/openapi/api.yaml`](../apps/contracts/openapi/api.yaml)) y el
> modelo de dominio / la implementación, con su **plan de cierre** (AP5).
> Un gap **no** se deja divergir en silencio: o se cierra, o se documenta aquí.
>
> El contrato es la **SSOT** del wire (A12). Este documento **no** redefine el
> contrato; solo anota dónde la implementación aún no lo cumple del todo.

---

## 1. Reglas

- **A12/AP-MR6**: todo cambio de wire empieza en `api.yaml` → `pnpm generate` →
  código. Un cambio breaking sube la `major` de `apps/contracts` y se anota en
  [`apps/contracts/CHANGELOG.md`](../apps/contracts/CHANGELOG.md).
- **AP4 (idempotencia)**: los códigos que son éxito idempotente
  (`409 analysis_pending`, `409 invalid_state`) están descritos en el contrato y
  el cliente los trata como éxito; no son gaps.
- **AP5 (este documento)**: si el contrato documenta algo que el dominio no
  implementa, se registra aquí con plan de cierre.

---

## 2. Gaps

| ID | Gap | Tipo | Estado |
|---|---|---|---|
| GAP-1 | `GET /analytics/progress` responde `501` | Endpoint no implementado | Abierto |
| GAP-2 | `DataExport.email` se toma de la config, no de un usuario persistido | Decisión de diseño | Aceptado (MVP) |

---

## GAP-1 — `GET /analytics/progress` sin implementar

- **Contrato**: la operación `get_progress_series` declara `200` con
  `ProgressSeries` y `501 NotImplemented`.
- **Dominio**: existe `analytics.ProgressMetric` (con tests), pero **nada lo
  materializa**: `refresh-aggregates` solo reconstruye `error_metrics`; no hay
  repositorio ni adaptador de progreso.
- **Implementación**: `Server.GetProgressSeries` responde `writeNotImplemented`
  (`501`). El dashboard del frontend muestra un placeholder "disponible
  próximamente".
- **Causa**: la Fase 6 (provisioner) no incluyó un item para materializar
  progreso; se documentó como gap.
- **Impacto**: funcionalidad anunciada no disponible; el contrato ya lo declara
  como `501`, así que no hay divergencia silenciosa.
- **Plan de cierre**: implementar la agregación de `ProgressMetric` (job o
  consulta), un `ProgressRepository`, cablearlo en `refresh-aggregates` y
  sustituir el `501` por el `200`. Alternativa: retirar la operación del
  contrato hasta implementarla.
- **Seguimiento**: [BUG-003](bugs/BUG-003-analytics-progress-501.md).

---

## GAP-2 — Origen del `email` en `DataExport`

- **Contrato**: `DataExport` requiere `email` (formato email).
- **Dominio**: existe `identity.Email` (value object, validado), pero **no hay
  tabla `users`**: el MVP es single-user y la identidad viene de la config.
- **Implementación**: `IdentityService.Export` toma el email de
  `APP_USER_EMAIL` (validado con `identity.NewEmail`).
- **Justificación**: añadir persistencia de usuarios sería alcance de un modelo
  multi-usuario que el MVP no tiene (`PRODUCT_DOMAIN §3.2`).
- **Plan de cierre**: introducir `users` persistidos junto con la autenticación
  real (post-MVP); entonces el email saldrá del modelo, no de la config.

---

## 3. Campos del contrato ↔ dominio

A fecha de hoy **todos** los campos de wire mapean a tipos del dominio o del
contexto:

| Esquema | Tipo de dominio |
|---|---|
| `Practice` | `practice.Practice` |
| `Fragment` / `TargetVerbReview` / … | `analysis.Fragment` (+ value objects) |
| `QuizQuestion` / `QuizEvaluation` | `analysis.QuizQuestion` / `analysis.QuizEvaluation` |
| `ErrorPatternStats` | `analytics.ErrorMetric` |
| `AccessLogEntry` | `identity.AccessEvent` |
| `DataExport.email` | `identity.Email` (origen: ver GAP-2) |

Campos de **respuesta calculados** (no del agregado): `AccessLog.total`,
`DataExport.generated_at`.

---

**Última revisión**: 2026-09-21.
