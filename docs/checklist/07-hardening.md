# Checklist — Fase 7: Hardening

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 7
> **Predecesora**: Fase 6 (Provisioner y privacidad) · **Sucesora**: Fase 8 (Tutor adaptativo, post-MVP)

---

## 7.1 CI/CD

- [x] `7.1.1` Crear `.github/workflows/ci.yml` con `detect-changes` (dorny/paths-filter) + jobs de contracts/backend/frontend (§6.5).
- [x] `7.1.2` Crear `.github/workflows/contracts.yml` (regenera Go+TS y falla por drift; valida `info.version == package.json`).
- [x] `7.1.3` Crear `.github/workflows/deploy-staging.yml` (build + push imágenes; Dockerfiles backend distroless y frontend standalone).
- [x] `7.1.4` Remote cache **self-hosted** (`ducktors/turborepo-remote-cache` en `ops/docker/`); `TURBO_API`/`TURBO_TEAM`/`TURBO_TOKEN` cablados desde secrets (AP-MR7); objetivo cache hit >80% documentado.
- [x] `7.1.5` Tier 3 (`test-integration`) en schedule nightly + push `main` + label `ready-for-release` (§6.5).

## 7.2 Seguridad

- [x] `7.2.1` Crear `.github/workflows/security.yml` con `gosec` y `govulncheck` (pin `v2.29.0`/`v1.8.0`; SARIF de gosec a Code Scanning).
- [x] `7.2.2` Crear `ops/scripts/pii_audit.sh` (ya existía, 6.3) y `ops/scripts/security_audit.sh` (secretos + guard de docs/x-internal).
- [x] `7.2.3` Verificar que `/docs` (Scalar/Swagger) **no** se expone en producción desde el backend (AP-MR9): sin handler de docs + test `TestGeneratedRouter_DocsEndpoints_NotFound` + guard en `security_audit.sh`.
- [x] `7.2.4` Endpoints marcados `x-internal: true` filtrados del render de docs (AP-MR9): el contrato no tiene endpoints `x-internal` (N/A); el guard de `security_audit.sh` lo exige si se añaden.

## 7.3 Documentación

- [x] `7.3.1` Redactar `docs/ARCHITECTURE.md` (contrato arquitectónico global; incluye §5.1 de env vars).
- [x] `7.3.2` Redactar `docs/RUNBOOK.md`, `docs/PRODUCTION_ENV.md`, `docs/SECRET_ROTATION.md`, `docs/SECURITY_DEBT.md`. Extra: `docs/API_CONTRACT.md` (gaps contrato↔dominio, AP5).
- [x] `7.3.3` Redactar `README.md` raíz de portafolio (visión, stack, arquitectura, métricas de rigor). `README.en.md` alineado.
- [x] `7.3.4` Registrar bugs conocidos en `docs/bugs/` (BUG-001 run-ons, BUG-002 `finish_reason=length`, BUG-003 `/analytics/progress` 501).

---

## ✅ Gate de salida

- [x] CI verde en PR (lint + test + build de todo el monorepo). *(primer run real en GitHub confirmado sin errores — 2026-09-22)*
- [x] `pnpm test-integration` verde en nightly (Tier 3 con testcontainers). *(idem, sin errores en el primer run)*
- [x] `gosec` + `govulncheck` sin hallazgos críticos. *(verificado en local: 0 hallazgos)*
- [x] Documentación canónica completa y consistente con los manifiestos.
- [x] **Definition of Done del MVP** (PRODUCT_DOMAIN §11) íntegramente cumplida. *(alcance funcional completo; el gate de CI quedó verde)*

## Fuente normativa

- **A10, A12**, **AP5**, **AP-MR7, AP-MR9**, **§6.5** (CI/CD del monorepo).
- **PRODUCT_DOMAIN.md** §11 (DoD) y §14 (documentos autoritativos).
