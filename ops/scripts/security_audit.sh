#!/usr/bin/env bash
#
# security_audit.sh — Auditoría estática de seguridad sobre el código fuente.
#
# Complementa a pii_audit.sh (que audita logs) y a gosec/govulncheck (que corren
# en .github/workflows/security.yml). Aquí no hace falta red ni herramientas
# externas: solo grep determinista sobre el repo.
#
# Comprueba:
#   1. Secretos hardcodeados (OpenAI, AWS, claves privadas, Slack).
#   2. AP-MR9: ningún render de docs (/docs, Scalar, Swagger, Redoc) sin el
#      guard `APP_API_DOCS_ENABLED` (no debe exponerse en producción).
#   3. AP-MR9: si el contrato marca algún endpoint `x-internal: true`, debe
#      existir código que lo filtre del render.
#
# Uso: ops/scripts/security_audit.sh
# Códigos de salida: 0 = limpio, 1 = hallazgo, 2 = error de uso/IO.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REPO_ROOT"

readonly SCAN_DIRS=(apps ops .github)
readonly EXCLUDE_DIRS=(node_modules .next .turbo .git dist coverage test-results playwright-report)

# Nota: `sk-` exige >=32 chars (longitud de una clave OpenAI real) para no
# confundir fixtures de test (p. ej. "sk-abcdefghijklmnop1234" en pii_handler_test.go).
readonly SECRETS='sk-[A-Za-z0-9]{32,}|AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY-----|xox[baprs]-[A-Za-z0-9-]{10,}'
readonly DOCS_REF='(/docs|[Ss]calar|[Ss]wagger|[Rr]edoc)'
readonly DOCS_GUARD='APP_API_DOCS_ENABLED'
readonly SPEC='apps/contracts/openapi/api.yaml'

status=0

grep_src() {
  # grep recursivo sobre los directorios de código, excluyendo artefactos.
  local pattern="$1"
  shift
  local args=()
  local d
  for d in "${EXCLUDE_DIRS[@]}"; do
    args+=("--exclude-dir=$d")
  done
  grep -rInE "$pattern" "${args[@]}" "$@" 2>/dev/null || true
}

redact() {
  sed -E \
    -e 's/(sk-[A-Za-z0-9]{4})[A-Za-z0-9]+/\1[REDACTED]/g' \
    -e 's/(AKIA)[0-9A-Z]{16}/\1[REDACTED]/g' \
    -e 's/(xox[baprs]-)[A-Za-z0-9-]+/\1[REDACTED]/g' \
    -e 's/(-----BEGIN [A-Z ]*PRIVATE KEY-----).*/\1[REDACTED]/g'
}

# --- 1. Secretos hardcodeados -------------------------------------------------
secrets_hits="$(grep_src "$SECRETS" \
  --include='*.go' --include='*.ts' --include='*.tsx' --include='*.js' \
  --include='*.sh' --include='*.yml' --include='*.yaml' --include='*.json' \
  "${SCAN_DIRS[@]}" \
  | grep -v 'security_audit.sh' || true)"

if [ -n "$secrets_hits" ]; then
  echo "security_audit: posibles secretos hardcodeados (rota/filtra el valor):" >&2
  printf '%s\n' "$secrets_hits" | redact >&2
  status=1
else
  echo "security_audit [1/3] sin secretos hardcodeados: OK"
fi

# --- 2. AP-MR9: docs no expuesto sin guard -----------------------------------
# Se excluyen los tests: asertan la AUSENCIA de /docs y no montan rutas de prod.
docs_files="$(grep_src "$DOCS_REF" --include='*.go' --exclude='*_test.go' apps/backend \
  | cut -d: -f1 | sort -u || true)"

docs_violations=""
for f in $docs_files; do
  [ -z "$f" ] && continue
  if ! grep -q "$DOCS_GUARD" "$f"; then
    docs_violations="${docs_violations}${f}"$'\n'
  fi
done

if [ -n "$docs_violations" ]; then
  echo "security_audit: render de docs sin guard '${DOCS_GUARD}' (AP-MR9):" >&2
  printf '%s' "$docs_violations" >&2
  status=1
else
  echo "security_audit [2/3] /docs no expuesto sin guard: OK"
fi

# --- 3. AP-MR9: x-internal filtrado ------------------------------------------
if grep -qE 'x-internal' "$SPEC"; then
  if [ -z "$(grep_src 'x-internal|XInternal' --include='*.go' --exclude='*_test.go' apps/backend)" ]; then
    echo "security_audit: '${SPEC}' marca x-internal pero no hay filtro en Go (AP-MR9)." >&2
    status=1
  else
    echo "security_audit [3/3] x-internal con filtro presente: OK"
  fi
else
  echo "security_audit [3/3] sin endpoints x-internal (N/A): OK"
fi

if [ "$status" -ne 0 ]; then
  echo "security_audit: FALLO (ver hallazgos arriba)." >&2
  exit 1
fi

echo "security_audit: limpio."
exit 0
