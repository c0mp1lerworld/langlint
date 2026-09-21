# PRODUCTION_ENV.md — Variables de Entorno

> Referencia operativa de **todas** las variables de entorno. El resumen está en
> [`ARCHITECTURE.md §5.1`](ARCHITECTURE.md#51-variables-de-entorno-resumen).
> Los valores secretos se rotan según [`SECRET_ROTATION.md`](SECRET_ROTATION.md).

---

## 1. Convenciones

- **Prefijos por app (AP-MR4)**: backend `APP_` (y `OPENAI_` para el SDK),
  frontend `NEXT_PUBLIC_`, contratos `APP_CONTRACTS_` (si aplica).
- **Plantilla**: `apps/backend/.env.example` y `apps/frontend/.env.example`.
- **`.env` está gitignoreado**: nunca se commitean secretos (A8).
- `cmd/api`, `cmd/migrate` y `cmd/provisioner` cargan `apps/backend/.env` de
  forma **relativa al directorio de trabajo** (ejecutar desde `apps/backend`).
- Las variables no declaradas en `.env` caen a su default o, si son requeridas,
  abortan el arranque con un mensaje explícito.

---

## 2. Backend

### 2.1 Base (varios entry points)

| Variable | Requerida | Default | Consumidores | Descripción |
|---|---|---|---|---|
| `APP_DATABASE_URL` | **sí** | — | api, provisioner, migrate | Cadena de conexión Postgres (pgx). |
| `APP_PSEUDONYM_SECRET` | **sí** | — | api, provisioner | Secreto HMAC-SHA256 para pseudonimizar analytics (A8). |

### 2.2 HTTP API (`cmd/api`)

| Variable | Requerida | Default | Descripción |
|---|---|---|---|
| `APP_HTTP_ADDR` | no | `:8080` | Dirección de escucha. |
| `APP_USER_ID` | **sí** | — | UUID v7 del usuario único del MVP. |
| `APP_USER_EMAIL` | **sí** | — | Email del usuario (export A9). |
| `APP_LLM_TIMEOUT` | no | `180s` | Timeout de la llamada al LLM (Go duration). |
| `APP_CORS_ALLOWED_ORIGINS` | no | `http://localhost:3000` | Orígenes permitidos, separados por comas. |

### 2.3 Motor de IA (`cmd/api`, `cmd/llmcheck`)

| Variable | Requerida | Default | Descripción |
|---|---|---|---|
| `OPENAI_API_KEY` | **sí** | — | Clave del proveedor. **Secreto.** |
| `OPENAI_MODEL` | **sí** | — | Alias del modelo; se persiste en `Analysis.model`. |
| `OPENAI_MODEL_VERSION` | no | = `OPENAI_MODEL` | Snapshot fechado (trazabilidad). |
| `OPENAI_BASE_URL` | no | `https://api.openai.com/v1` | Base URL alternativa (proxy/Azure). Vacío = default del SDK. |

### 2.4 Provisioner (`cmd/provisioner`)

| Variable | Requerida | Default | Descripción |
|---|---|---|---|
| `APP_RAW_RETENTION_DAYS` | no | `30` | Días antes de que `purge-raw-data` borre prácticas eliminadas. |
| `APP_DELETION_GRACE_DAYS` | no | `30` | Días de gracia antes de `execute-deletions` (A9). |

### 2.5 Ejemplo (`apps/backend/.env`)

```dotenv
OPENAI_API_KEY=
OPENAI_MODEL=gpt-4o-mini
OPENAI_MODEL_VERSION=gpt-4o-mini-2024-07-18
OPENAI_BASE_URL=
APP_DATABASE_URL=postgres://langlint:langlint@localhost:5433/langlint?sslmode=disable
APP_HTTP_ADDR=:8080
APP_USER_ID=0192f3a0-0000-7000-8000-000000000000
APP_USER_EMAIL=student@example.com
APP_LLM_TIMEOUT=180s
APP_CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:3001
APP_PSEUDONYM_SECRET=change-me-to-a-long-random-secret
APP_RAW_RETENTION_DAYS=30
APP_DELETION_GRACE_DAYS=30
```

---

## 3. Frontend

| Variable | Requerida | Default | Descripción |
|---|---|---|---|
| `NEXT_PUBLIC_API_URL` | sí (build time) | `http://localhost:8080` (fallback del cliente) | URL base del API. Se **inlinea** en el bundle durante el build. |

Ubicación: `apps/frontend/.env.local` (gitignoreado). En la imagen Docker se
inyecta con el build-arg `NEXT_PUBLIC_API_URL`.

---

## 4. CI/CD (GitHub Actions)

Se configuran en **Settings → Secrets and variables → Actions**.

| Nombre | Tipo | Uso |
|---|---|---|
| `TURBO_API` | secret | URL del remote cache self-hosted. |
| `TURBO_TEAM` | secret | Identificador de equipo para el cache. |
| `TURBO_TOKEN` | secret | Token del cache (debe coincidir con `TURBO_REMOTE_CACHE_TOKEN`). |
| `STAGING_API_URL` | **variable** | Valor de `NEXT_PUBLIC_API_URL` para la imagen de staging. |
| `GITHUB_TOKEN` | built-in | Login a GHCR (`packages: write`) en `deploy-staging.yml`. |

Si `TURBO_*` no está definido, Turborepo usa solo la caché local y **no falla**.

---

## 5. Remote cache self-hosted (`ops/docker`)

| Variable | Requerida | Default | Descripción |
|---|---|---|---|
| `TURBO_REMOTE_CACHE_TOKEN` | no | `local-dev-token` | Token del servicio `turbo-cache`. |
| `TURBO_REMOTE_CACHE_SIGNATURE_KEY` | no | — | Firma/verificación de artefactos (opcional). |

---

## 6. Toolchain

| Herramienta | Versión | Fuente |
|---|---|---|
| Go | 1.26.8 | `.tool-versions`, `toolchain` en `apps/backend/go.mod` |
| Node | 22.13.0 | `.tool-versions`, `.nvmrc` |
| pnpm | 9.15.4 | `.tool-versions`, `packageManager` en `package.json` |

---

**Última revisión**: 2026-09-21.
