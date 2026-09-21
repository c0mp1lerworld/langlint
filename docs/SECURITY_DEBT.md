# SECURITY_DEBT.md — Deuda de Seguridad Aceptada

> Registro de **waivers de seguridad legítimos**: riesgos conocidos, aceptados
> de forma consciente y con justificación. Cada entrada declara su alcance, la
> mitigación existente y el plan de cierre. Un waiver **no** es un olvido.
>
> Controles vigentes que **no** son deuda: `gosec` + `govulncheck` en CI (0
> hallazgos), `security_audit.sh` (secretos/docs), `pii_audit.sh` (logs),
> pseudonimización de analytics (A8) y ausencia de `/docs` en producción (AP-MR9).

---

## Resumen

| ID | Deuda | Riesgo | Estado |
|---|---|---|---|
| SD-1 | API sin autenticación (MVP single-user) | Alto (si se despliega público) | Aceptado (MVP) |
| SD-2 | `user_id` crudo en `access_events`/`deletion_requests` | Bajo | Aceptado (A9) |
| SD-3 | Sin escaneo de dependencias del frontend | Medio | Abierto |
| SD-4 | Remote cache no alcanzable desde runners *hosted* | Bajo (disponibilidad) | Aceptado |
| SD-5 | Sin render de docs / `x-internal` no implementado | Bajo | Aceptado |
| SD-6 | Sin rate limiting ni cuotas en el API | Medio (si público) | Aceptado (MVP) |

---

## SD-1 — API sin autenticación

- **Qué es**: el contrato no declara `securityScheme`; el usuario se resuelve
  desde la config (`APP_USER_ID`/`APP_USER_EMAIL`), no desde un token.
- **Justificación**: es un **non-goal** explícito del MVP
  (`PRODUCT_DOMAIN §3.2`). El sistema es de un único usuario y no debe
  desplegarse como servicio multi-tenant.
- **Mitigación**: la identidad es fija; no hay endpoint que exponga datos de
  otros usuarios (single-user por construcción).
- **Cierre**: autenticación real (sesión/JWT + tabla `users`) en el roadmap
  post-MVP; entonces `APP_USER_ID`/`APP_USER_EMAIL` dejan de venir de config.
- **Condición de exposición**: **no** exponer el API a Internet sin un proxy con
  auth mientras esta deuda siga abierta.

## SD-2 — `user_id` crudo en tablas de identidad/auditoría

- **Qué es**: `access_events` y `deletion_requests` almacenan el `user_id` sin
  pseudonimizar; solo `error_metrics` se pseudonimiza.
- **Justificación**: son datos de **identidad y auditoría** (A9/A4), no
  analytics. La pseudonimización (A8) aplica a la materialización de analytics,
  no a la auditoría del propio usuario, que debe poder resolver su `user_id`.
- **Mitigación**: `access_events` es **append-only** (sin Update/Delete); los
  datos crudos (`practices`/`analyses`) se purgan por retención.
- **Cierre**: no aplica (decisión de diseño); revisar si se introduce
  multi-tenant.

## SD-3 — Sin escaneo de dependencias del frontend

- **Qué es**: `security.yml` escanea **Go** (`gosec`/`govulncheck`). Las
  dependencias npm del frontend (`apps/frontend`) no se auditan en CI.
- **Justificación**: en el MVP el frontend no tiene dependencias de runtime
  sensibles más allá de Next/React; el escaneo de Go cubre el binario expuesto.
- **Mitigación**: Dependabot/renovación manual pendiente; revisar `pnpm audit`
  puntualmente.
- **Cierre**: añadir un job `pnpm audit --prod` (o Dependabot) al workflow de
  seguridad.

## SD-4 — Remote cache no alcanzable desde runners *hosted*

- **Qué es**: la caché self-hosted vive en la red local/cluster; los runners
  *hosted* de GitHub no la alcanzan salvo que el endpoint sea público.
- **Justificación**: es una limitación de **disponibilidad**, no de
  confidencialidad. Sin caché remota, Turborepo usa la local y **no falla**.
- **Mitigación**: documentado en `ops/docker/README.md`; el endpoint puede
  exponerse con TLS + token.
- **Cierre**: usar self-hosted runners o un endpoint público autenticado.

## SD-5 — Sin render de docs y `x-internal` no implementado

- **Qué es**: el backend no sirve `/docs` (Scalar/Swagger) y el contrato no
  marca ningún endpoint `x-internal`; no hay filtro porque no hay render.
- **Justificación**: AP-MR9 exige que el render **no** se exponga en producción;
  la forma más segura de cumplirlo es **no tener render**.
- **Mitigación**: `TestGeneratedRouter_DocsEndpoints_NotFound` (404 en rutas de
  docs) + guard en `security_audit.sh`.
- **Cierre**: si se añade un render, hacerlo **gated** por
  `APP_API_DOCS_ENABLED` (+ `APP_API_DOCS_TOKEN`) y filtrar `x-internal`.

## SD-6 — Sin rate limiting ni cuotas

- **Qué es**: el API no limita la tasa de peticiones ni el gasto en el LLM.
- **Justificación**: coherente con SD-1 (single-user, no público).
- **Mitigación**: el análisis es asíncrono (outbox); el coste del LLM se acota
  con el prompt y el timeout.
- **Cierre**: rate limiting en el proxy o middleware cuando el servicio se
  exponga.

---

**Última revisión**: 2026-09-21.
