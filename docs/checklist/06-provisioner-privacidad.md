# Checklist — Fase 6: Provisioner y Privacidad

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 6
> **Predecesora**: Fase 5 (Frontend) · **Sucesora**: Fase 7 (Hardening)

---

## 6.1 Jobs batch (provisioner)

- [x] `6.1.1` Crear `cmd/provisioner/main.go` con subcomandos CLI (PRODUCT_DOMAIN §7.2). _(Composición Fx vía `fx.Populate`; el despacho de `refresh-aggregates`/`purge-raw-data`/`execute-deletions` vive en `internal/provisioner/handlers/jobs.go`. Config propia `LoadProvisionerConfig`; sin deps nuevas (stdlib).)_
- [x] `6.1.2` Implementar `refresh-aggregates` (reconstruir métricas de analytics desde la fuente de verdad, A4). _(`RefreshAggregatesService` cuenta los `error_patterns` de las `analyses` completadas (join con `practices` no borradas) y hace `ReplaceAll` transaccional de `error_metrics`; una tabla derivada se reconstruye, no se parchea. La serie de progreso (`/analytics/progress`) queda fuera: no tiene tabla ni item en este checklist.)_
- [x] `6.1.3` Implementar `purge-raw-data` (purga programada de datos crudos, A8 retención limitada). _(`PurgeRawDataService` borra físicamente prácticas `deleted_at < now - APP_RAW_RETENTION_DAYS` (default 30) y sus `analyses` en una transacción; idempotente.)_
- [x] `6.1.4` Implementar `execute-deletions` (ejecuta derecho al olvido tras gracia de 30 días, A9). _(`identity.DeletionRequest` + tabla `deletion_requests` (migración `000002`); `ExecuteDeletionsService` ejecuta las solicitudes vencidas (`APP_DELETION_GRACE_DAYS`, default 30) purgando prácticas/analyses/error_metrics del usuario y marcando la solicitud. Consumer-first: el endpoint `DELETE /me/data` (6.2.2) llega después.)_
- [x] `6.1.5` Crear `internal/provisioner/di/` (fx.Module propio, sin compartir módulos con `api/`). _(`Module()` cablea repos + services + runner; `api/` y `provisioner/` no se importan (A2/A3). El harness Tier 3 `testsupport` se movió a `internal/shared/testdb` para que ambas entry points lo reusen sin import cruzado.)_

## 6.2 Endpoints A9 (portabilidad, olvido, auditoría)

- [x] `6.2.1` Implementar `GET /v1/me/data/export` (portabilidad, GDPR Art. 15+20). _(`IdentityService.Export` + `PracticeRepository.ListAllByUser`; el email sale de `APP_USER_EMAIL` (identidad single-user de config: no hay tabla `users` en el MVP). `DataExport` 200 con todas las prácticas no borradas. A12: se quitó el `501` documentado.)_
- [x] `6.2.2` Implementar `DELETE /v1/me/data` (derecho al olvido con gracia de 30 días, Art. 17). _(`IdentityService.RequestDeletion` inserta una `deletion_request` (tabla + job de 6.1) y responde `202`; idempotente si ya hay una pendiente (`HasPending`, AP4).)_
- [x] `6.2.3` Implementar `GET /v1/me/access-log` (auditoría del usuario, append-only). _(Tabla `access_events` (migración `000003`) + `AccessLogRepository` sin Update/Delete (A4); middleware `internal/api/accesslog` registra cada petición autenticada (`action`=método, `resource_type`=path), best-effort. `GET` paginado (`limit`/`offset`).)_

## 6.3 Retención y pseudonimización

- [x] `6.3.1` Implementar `pseudonymizer.go` (HMAC-SHA256 con `APP_PSEUDONYM_SECRET` antes de analytics, A8). _(Vive en `internal/shared/pseudonymizer/` y no en `api/services/` porque el provisioner también materializa/borra analytics y `api`↔`provisioner` no se importan (A2/A3). `error_metrics.user_id` pasa a `text` (migración `000004`) y se pseudonimiza en los **adaptadores**: `Upsert`/`ListByUser` (API), `ReplaceAll` (`refresh-aggregates`) y el `DELETE` de `error_metrics` en `Execute` (`execute-deletions`). Dominio, puertos y services intactos.)_
- [x] `6.3.2` Separar datos crudos en tablas dedicadas con retención limitada (A8). _(Los datos crudos viven en `practices`/`analyses` (tablas dedicadas) y se purgan con `purge-raw-data`/`execute-deletions` (6.1). Se certifica con el test Tier 3 `TestPrivacy_RawDataSeparatedFromAnalytics`: analytics solo lleva la clave pseudonimizada, sin texto crudo, y la purga/olvido eliminan todo rastro (incl. `error_metrics` por pseudónimo).)_
- [x] `6.3.3` Verificar que el `PIIHandler` es guard runtime en todos los logs (A8). _(Auditoría: los únicos loggers de producción (`cmd/api`, `cmd/provisioner`) se construyen con `logger.New`, que envuelve `PIIHandler`; el resto de `slog.New` son tests con `io.Discard`. Se añade `ops/scripts/pii_audit.sh` — grep defensivo de email/teléfono/API-key/JWT sobre logs, exit ≠0 si hay PII.)_

---

## ✅ Gate de salida

- [x] Endpoints A9 operativos y testeados (export/delete/access-log).
- [x] `purge-raw-data` y `execute-deletions` cubiertos por tests Tier 3 (testcontainers).
- [x] Ningún log contiene PII en crudo (auditoría `ops/scripts/pii_audit.sh`).
- [x] `pnpm test-integration --filter=backend` en verde.

## Fuente normativa

- **A8, A9, A4**.
- **PRODUCT_DOMAIN.md** §7 (analítica/provisioner) y §9 (privacidad).
