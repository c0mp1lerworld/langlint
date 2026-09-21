#!/usr/bin/env bash
#
# Valida que la versión declarada en el contrato (info.version de api.yaml)
# coincida con la versión del paquete @langlint/contracts (package.json).
#
# Un cambio de wire sube ambos de forma atómica (A12, AP-MR6); si divergen, el
# código generado de backend/frontend queda apuntando a dos "versiones" del
# contrato. Se ejecuta en `contracts.yml` y localmente con:
#   pnpm --filter @langlint/contracts check-version
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SPEC="$HERE/../openapi/api.yaml"
PKG="$HERE/../package.json"

# Primera clave `version:` indentada del spec = `info.version` (la cabecera
# `openapi:` no lleva key `version:`; `model_version` no matchea por el guion).
spec_version="$(grep -m1 -E '^[[:space:]]+version:' "$SPEC" | sed -E 's/^[[:space:]]*version:[[:space:]]*//' | tr -d '"'"'"'')"
pkg_version="$(grep -m1 -E '"version":' "$PKG" | sed -E 's/.*"version":[[:space:]]*"([^"]+)".*/\1/')"

if [ -z "$spec_version" ] || [ -z "$pkg_version" ]; then
  echo "check-version: no se pudo extraer la versión (spec='${spec_version}', package='${pkg_version}')" >&2
  exit 1
fi

if [ "$spec_version" != "$pkg_version" ]; then
  echo "check-version: drift de versión" >&2
  echo "  api.yaml   info.version = $spec_version" >&2
  echo "  package.json    version = $pkg_version" >&2
  echo "Sube ambos en el mismo cambio (AP-MR6)." >&2
  exit 1
fi

echo "check-version OK: contract version $spec_version"
