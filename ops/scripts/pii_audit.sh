#!/usr/bin/env bash
#
# pii_audit.sh — Auditoría estática de PII en logs (A8).
#
# Defensa en profundidad: complementa al PIIHandler runtime. Escanea los logs
# de CI/servicio buscando patrones que nunca deberían haberse registrado
# (emails, teléfonos, API keys y JWTs). Falla (exit 1) si encuentra alguno.
#
# Uso:
#   ops/scripts/pii_audit.sh                      # lee de stdin
#   ops/scripts/pii_audit.sh service.log api.log  # escanea ficheros/dirs
#   cat *.log | ops/scripts/pii_audit.sh
#
# Códigos de salida:
#   0 = limpio
#   1 = PII detectada (violación A8)
#   2 = error de uso/IO

set -euo pipefail

readonly EMAIL='[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}'
readonly PHONE='(\+[0-9]{1,3}([ .-]?\(?[0-9]{1,4}\)?){2,})|(\([0-9]{1,4}\)[ .-]?[0-9]{2,4}([ .-]?[0-9]{2,4})?\b)|(\b[0-9]{2,3}[ .-][0-9]{2,4}([ .-]?[0-9]{2,4})?\b)'
readonly API_KEY='(sk|pk)-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16}'
readonly JWT='eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+'
readonly PII="${EMAIL}|${PHONE}|${API_KEY}|${JWT}"

if [ "$#" -gt 0 ]; then
  for path in "$@"; do
    if [ ! -e "$path" ]; then
      echo "pii_audit: no such file or directory: $path" >&2
      exit 2
    fi
  done
  matches=$(grep -rnE "$PII" "$@" || true)
else
  matches=$(grep -nE "$PII" || true)
fi

if [ -n "$matches" ]; then
  echo "pii_audit: PII detected in logs (A8 violation):" >&2
  printf '%s\n' "$matches" | sed -E "s/(${PII})/[REDACTED]/g" >&2
  exit 1
fi

echo "pii_audit: no PII detected"
