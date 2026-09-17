#!/usr/bin/env bash
#
# Orquestador de generación de código desde el contrato OpenAPI (SSOT, A12).
#
# Go  -> apps/backend/internal/api/handlers/gen_{types,server}.go  (oapi-codegen)
# TS  -> apps/frontend/src/lib/api/gen.ts                         (openapi-typescript)
#
# Uso: `pnpm generate` (raíz) o `pnpm --filter @langlint/contracts generate`.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"

# Go
make -C "$ROOT/apps/backend" generate

# TypeScript
(cd "$ROOT/apps/frontend" && pnpm run generate)
