# Checklist — Fase 6: Provisioner y Privacidad

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 6
> **Predecesora**: Fase 5 (Frontend) · **Sucesora**: Fase 7 (Hardening)

---

## 6.1 Jobs batch (provisioner)

- [ ] `6.1.1` Crear `cmd/provisioner/main.go` con subcomandos CLI (PRODUCT_DOMAIN §7.2).
- [ ] `6.1.2` Implementar `refresh-aggregates` (reconstruir métricas de analytics desde la fuente de verdad, A4).
- [ ] `6.1.3` Implementar `purge-raw-data` (purga programada de datos crudos, A8 retención limitada).
- [ ] `6.1.4` Implementar `execute-deletions` (ejecuta derecho al olvido tras gracia de 30 días, A9).
- [ ] `6.1.5` Crear `internal/provisioner/di/` (fx.Module propio, sin compartir módulos con `api/`).

## 6.2 Endpoints A9 (portabilidad, olvido, auditoría)

- [ ] `6.2.1` Implementar `GET /v1/me/data/export` (portabilidad, GDPR Art. 15+20).
- [ ] `6.2.2` Implementar `DELETE /v1/me/data` (derecho al olvido con gracia de 30 días, Art. 17).
- [ ] `6.2.3` Implementar `GET /v1/me/access-log` (auditoría del usuario, append-only).

## 6.3 Retención y pseudonimización

- [ ] `6.3.1` Implementar `pseudonymizer.go` (HMAC-SHA256 con `APP_PSEUDONYM_SECRET` antes de analytics, A8).
- [ ] `6.3.2` Separar datos crudos en tablas dedicadas con retención limitada (A8).
- [ ] `6.3.3` Verificar que el `PIIHandler` es guard runtime en todos los logs (A8).

---

## ✅ Gate de salida

- [ ] Endpoints A9 operativos y testeados (export/delete/access-log).
- [ ] `purge-raw-data` y `execute-deletions` cubiertos por tests Tier 3 (testcontainers).
- [ ] Ningún log contiene PII en crudo (auditoría `ops/scripts/pii_audit.sh`).
- [ ] `pnpm test-integration --filter=backend` en verde.

## Fuente normativa

- **A8, A9, A4**.
- **PRODUCT_DOMAIN.md** §7 (analítica/provisioner) y §9 (privacidad).
