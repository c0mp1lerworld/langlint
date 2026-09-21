# SECRET_ROTATION.md — Rotación de Secretos

> Procedimiento y cadencia de rotación de los secretos de LangLint. Los
> **valores** nunca viven en el repositorio: `.env` está gitignoreado y los de
> CI en GitHub Secrets (A8).

---

## 1. Política

### 1.1 ¿Cuándo rotar?

| Disparador | Plazo |
|---|---|
| **Filtración confirmada o sospechada** (commit, log, captura, tercero) | **Inmediata** (≤ 24 h) |
| Salida de una persona con acceso al valor | ≤ 7 días |
| Sospecha de acceso no autorizado (analytics anómalas, coste del LLM anómalo) | Inmediata |
| **Rotación programada** de secretos de alto valor (`OPENAI_API_KEY`) | trimestral |
| Rotación programada del resto | semestral |
| Cambio de proveedor/entorno (staging → producción) | al desplegar |

> Regla dura: **ante la duda, rotar**. El coste de rotar es bajo; el de filtrar
> un secreto es un incidente.

### 1.2 Principios

- Un secreto **no** se comparte entre entornos (dev ≠ staging ≠ prod).
- La rotación es **atómica**: se actualiza el valor en todos los consumidores
  (`.env`, Secrets de CI, docker-compose) y se revoca el anterior.
- El valor anterior se revoca **tras** confirmar que el nuevo funciona.
- Nunca se loguea el valor (el `PIIHandler` como guard runtime, A8).

---

## 2. Inventario

| Secreto | Consumidor | Ámbito |
|---|---|---|
| `OPENAI_API_KEY` | `cmd/api`, `cmd/llmcheck` | Proveedor LLM |
| `APP_PSEUDONYM_SECRET` | `cmd/api`, `cmd/provisioner` | HMAC de analytics (A8) |
| `TURBO_TOKEN` (+ `TURBO_API`, `TURBO_TEAM`) | CI, clientes de Turborepo | Remote cache |
| `TURBO_REMOTE_CACHE_TOKEN` | servicio `turbo-cache` (docker-compose) | Remote cache |
| `TURBO_REMOTE_CACHE_SIGNATURE_KEY` | cache server + clientes | Firma de artefactos (opcional) |
| Credenciales de `APP_DATABASE_URL` | api, provisioner, migrate | Postgres |
| `APP_API_DOCS_TOKEN` | — (reservado, no implementado) | Docs de API (AP-MR9) |

---

## 3. Procedimientos

### 3.1 `OPENAI_API_KEY`

1. Generar una clave nueva en el panel de OpenAI (con el mínimo de permisos).
2. Actualizar `apps/backend/.env` (dev) y el secreto del entorno de despliegue.
3. Reiniciar `cmd/api` (la config se lee al arrancar).
4. Verificar con `go run ./cmd/llmcheck` o un análisis real.
5. **Revocar** la clave antigua en el panel.

### 3.2 `APP_PSEUDONYM_SECRET`

> **Impacto**: los pseudónimos de `error_metrics` son
> `HMAC-SHA256(secret, user_id)`. Rotarlo hace que las claves almacenadas dejen
> de coincidir con las derivadas: las métricas existentes quedan **huérfanas**
> (no se leen) hasta reconstruirlas.

1. Generar un secreto nuevo: `openssl rand -hex 32`.
2. Actualizar `APP_PSEUDONYM_SECRET` en **api y provisioner** (deben coincidir).
3. Reiniciar `cmd/api` y `cmd/provisioner`.
4. Reconstruir analytics con la clave nueva:
   `go run ./cmd/provisioner refresh-aggregates`
   (reconstruye `error_metrics` desde `analyses`, la fuente de verdad; las filas
   con la clave antigua se reemplazan).

### 3.3 `TURBO_TOKEN` / `TURBO_REMOTE_CACHE_TOKEN`

1. Generar un token nuevo y colocarlo en la variable del servicio:
   `TURBO_REMOTE_CACHE_TOKEN` (docker-compose).
2. Reiniciar el contenedor `turbo-cache`.
3. Actualizar el secreto `TURBO_TOKEN` en GitHub (y en los entornos locales).
4. Invalidar el token antiguo (`TURBO_TOKEN` acepta una lista separada por comas,
   útil para rotar sin cortar el servicio).

### 3.4 `TURBO_REMOTE_CACHE_SIGNATURE_KEY` (opcional)

Debe ser **idéntico** en el servidor de caché y en todos los clientes. Al
rotarlo, actualizar ambos lados y reiniciar; los artefactos firmados con la
clave anterior dejarán de verificarse (se repoblarán en el siguiente build).

### 3.5 Credenciales de Postgres (`APP_DATABASE_URL`)

1. Crear la nueva contraseña/rol en Postgres.
2. Actualizar `APP_DATABASE_URL` en api, provisioner y migrate.
3. Reiniciar los procesos y verificar conectividad (`select 1`).
4. Revocar el rol/contraseña anterior.

### 3.6 Secrets de GitHub Actions

Se gestionan en **Settings → Secrets and variables → Actions**:
`TURBO_API`, `TURBO_TEAM`, `TURBO_TOKEN` (secrets) y `STAGING_API_URL`
(variable). `GITHUB_TOKEN` es efímero y lo provee el runner.

### 3.7 `APP_API_DOCS_TOKEN` (reservado)

Documentado por AP-MR9 para proteger el render de docs en staging, pero **no
está implementado** (el backend no sirve docs). Si en el futuro se añade el
render, este token pasa a ser un secreto real y entra en la rotación.

---

## 4. Secreto filtrado (incidente)

1. **Rotar de inmediato** el secreto afectado (§3).
2. Buscar el alcance: `ops/scripts/pii_audit.sh` y
   `ops/scripts/security_audit.sh` ayudan a detectar PII/secretos en logs y
   código.
3. Revisar `access_events` y el uso del proveedor (p. ej. coste del LLM) por si
   hubo abuso.
4. Registrar el incidente y, si aplica, abrir una entrada en
   [`SECURITY_DEBT.md`](SECURITY_DEBT.md) o en [`bugs/`](bugs/).

---

**Última revisión**: 2026-09-21.
