# Checklist — Fase 1: Fundaciones (Monorepo + Contrato OpenAPI)

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 1
> **Predecesora**: — · **Sucesora**: Fase 2 (Dominio puro)

---

## 1.1 Monorepo base (Turborepo + pnpm workspaces)

- [x] `1.1.1` Crear `package.json` raíz con `"private": true` obligatorio (AP-MR5) y `"packageManager": "pnpm@9.15.4"`.
- [x] `1.1.2` Definir scripts globales: `build`, `lint`, `test`, `test-integration`, `generate`, `typecheck`, `dev:backend`, `dev:frontend` (§2.2 del manifiesto monorepo).
- [x] `1.1.3` Crear `pnpm-workspace.yaml` con `packages: ['apps/*']`.
- [x] `1.1.4` Crear `turbo.json` con tasks `build` (dependsOn `^build`), `lint`, `test`, `test-integration`, `typecheck`, `dev`, `generate` (§2.2).
- [x] `1.1.5` Instalar `turbo` y `typescript` como `devDependencies` de la raíz.
- [x] `1.1.6` Crear `.nvmrc`, `.tool-versions` (Go, Node, pnpm), `.editorconfig`, `.gitignore`.

## 1.2 Árbol canónico de directorios

- [ ] `1.2.1` Crear `apps/contracts/` (con `openapi/` y `scripts/`), `apps/backend/`, `apps/frontend/` (§3.1 del manifiesto monorepo).
- [ ] `1.2.2` Crear `docs/` (incluye `PRODUCT_DOMAIN.md` ya existente) y `ops/` (`scripts/`, `docker/`).
- [ ] `1.2.3` Crear `.github/workflows/` (placeholder para Fase 7).
- [ ] `1.2.4` Verificar que ningún código de wire vive fuera de `apps/contracts/openapi/` (AP-MR2).

## 1.3 Contrato OpenAPI fundacional (`api.yaml`)

- [ ] `1.3.1` Redactar `apps/contracts/openapi/api.yaml` como OpenAPI 3.1.0 con `info.version` y `servers` `/v1`.
- [ ] `1.3.2` Declarar schemas `Fragment`, `ErrorPattern`, `Analysis`, `Practice`, `PracticeStatus` (PRODUCT_DOMAIN §5.2).
- [ ] `1.3.3` Declarar la enumeración de `ErrorPattern.code` (8 códigos) y `severity` (`minor|moderate|critical`).
- [ ] `1.3.4` Declarar endpoints `/v1/practices*`, `/v1/analytics/*`, `/v1/me/*` (PRODUCT_DOMAIN §5.1).
- [ ] `1.3.5` Documentar `409 analysis_pending` como éxito idempotente en el response (AP4).
- [ ] `1.3.6` Todos los campos de wire en `snake_case` (AP2).

## 1.4 Pipeline de generación (`pnpm generate`)

- [ ] `1.4.1` Crear `apps/contracts/scripts/generate.sh` que invoque `oapi-codegen` y `openapi-typescript`.
- [ ] `1.4.2` Configurar `oapi-codegen` en `apps/backend/Makefile` → `internal/api/handlers/gen_types.go` + `gen_server.go` (§6.3).
- [ ] `1.4.3` Configurar `openapi-typescript` en `apps/frontend/package.json` → `src/lib/api/gen.ts`.
- [ ] `1.4.4` Verificar que `gen_*.go` y `gen.ts` quedan commiteados (builds offline, §6.3).
- [ ] `1.4.5` Verificar que `pnpm generate` produce outputs idénticos en Go y TS sin drift (AP-MR6).

---

## ✅ Gate de salida

- [ ] `pnpm install --frozen-lockfile` termina sin error.
- [ ] `pnpm generate` regenera tipos Go y TS de forma idéntica.
- [ ] `pnpm build` en verde (monorepo orquestado por Turborepo).
- [ ] `api.yaml` validado (lint de OpenAPI) y versionado como `contracts-v1.0.0`.

## Fuente normativa

- **A11** (monolito lógico, un solo `go.mod`), **A12** (SSOT), **AP-MR1..AP-MR9** (anti-patrones de monorepo).
- **PRODUCT_DOMAIN.md** §5 (contrato) y §12 (Paso 1).
