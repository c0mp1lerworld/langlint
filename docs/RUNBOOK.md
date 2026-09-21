# RUNBOOK.md — Operación e Incidentes

> Guía operativa para levantar, diagnosticar y recuperar LangLint. Cada
> escenario es una receta accionable. Para variables de entorno, ver
> [`PRODUCTION_ENV.md`](PRODUCTION_ENV.md); para secretos,
> [`SECRET_ROTATION.md`](SECRET_ROTATION.md).

**Toolchain**: Go 1.26.8 (`toolchain` en `apps/backend/go.mod`), Node 22,
pnpm 9.15.4 (ver `.tool-versions`).

---

## Escenario 1 — Levantar el entorno local

```bash
# 1. Infraestructura (Postgres :5433 y remote cache :3300)
docker compose -f ops/docker/docker-compose.yml up -d

# 2. Backend (desde apps/backend; carga apps/backend/.env)
cp .env.example .env          # solo la primera vez; rellenar valores reales
go run ./cmd/migrate          # aplica migraciones goose
go run ./cmd/api              # sirve en :8080

# 3. Frontend (otra terminal)
cp apps/frontend/.env.example apps/frontend/.env.local   # NEXT_PUBLIC_API_URL
pnpm install --frozen-lockfile
pnpm --filter frontend dev
```

**Verificar**: `curl -s localhost:8080/v1/practices` responde `200` (o `[]`).

---

## Escenario 2 — La API no arranca por configuración

**Síntoma**: `api: APP_XXX is required` o `is not a valid …`.

`cmd/api` valida la config al arrancar y aborta con un mensaje explícito:

- `APP_DATABASE_URL is required` → definir la cadena de conexión (Escenario 5
  si además no responde).
- `APP_USER_ID is required` / `is not a valid UUID` → debe ser un UUID v7.
- `APP_USER_EMAIL is required` / `is not a valid email` → email válido.
- `APP_PSEUDONYM_SECRET is required` → generar uno (`openssl rand -hex 32`).
- `OPENAI_API_KEY is required` / `OPENAI_MODEL is required`.

`.env` se carga **relativo al directorio de trabajo**: ejecuta `go run ./cmd/api`
desde `apps/backend`.

---

## Escenario 3 — Migraciones y drift de goose

**Síntoma**: al arrancar, `42P07 relation "…" already exists`.

Ocurre si el esquema existe en la base pero `goose_db_version` no lo registra
(por ejemplo, tras aplicar SQL a mano). Comprobar:

```bash
psql "$APP_DATABASE_URL" -c 'SELECT * FROM goose_db_version ORDER BY id;'
```

Si las tablas ya coinciden con las migraciones, marcar las versiones aplicadas
con `goose up-to`/inserción en `goose_db_version` y dejar que goose aplique solo
las restantes. En un entorno limpio el remedio es rotar el volumen:

```bash
docker compose -f ops/docker/docker-compose.yml down -v   # borra langlint_pgdata
docker compose -f ops/docker/docker-compose.yml up -d
go run ./cmd/migrate
```

Las migraciones van **embebidas** en el binario (`//go:embed` en
`apps/backend/migrations/`), así que `cmd/migrate` no necesita ficheros en el
filesystem.

---

## Escenario 4 — El frontend no conecta con la API

**Síntoma**: error de red o `No 'Access-Control-Allow-Origin' header`.

1. **`NEXT_PUBLIC_API_URL`**: debe estar definida en `apps/frontend/.env.local`
   (build time). Sin ella el cliente cae a `http://localhost:8080`; en
   producción inyectarla vía build arg (`STAGING_API_URL`).
2. **CORS**: si el dev server arranca en un puerto distinto de `:3000`, añadirlo
   a `APP_CORS_ALLOWED_ORIGINS` (lista separada por comas). Reiniciar `cmd/api`.

---

## Escenario 5 — Postgres caído o inaccesible

**Síntoma**: `connection refused`, `dial tcp …:5433`, o timeouts en la API.

1. Comprobar el contenedor: `docker compose -f ops/docker/docker-compose.yml ps`.
2. Levantarlo: `docker compose -f ops/docker/docker-compose.yml up -d postgres`.
3. Verificar conectividad con la cadena real:
   `psql "$APP_DATABASE_URL" -c 'select 1;'`.
4. Si el puerto `5433` está ocupado, ajustar el mapeo en `ops/docker/docker-compose.yml`
   y `APP_DATABASE_URL`.
5. Si la base está corrupta y es un entorno de desarrollo, rotar el volumen
   (Escenario 3) y reaplicar migraciones.

La API **no reintenta** la conexión de forma indefinida: si Postgres cae en
caliente, el pool de `pgx` falla las queries; reiniciar `cmd/api` tras recuperar
Postgres.

---

## Escenario 6 — El análisis falla o tarda demasiado

**Síntoma**: la práctica pasa a `failed` con `llm_unavailable`, o el análisis se
queda en `analyzing`.

1. **Timeout**: `APP_LLM_TIMEOUT` (default `180s`). Si el modelo es lento,
   subirlo y reiniciar `cmd/api`.
2. **Truncado de salida**: el adaptador invoca al modelo **una vez por frase**;
   si aun así una respuesta se trunca (`finish_reason=length`), hoy se
   manifiesta como `llm_unavailable` (ver [`bugs/`](bugs/)).
3. **Clave/base URL**: verificar `OPENAI_API_KEY` y `OPENAI_BASE_URL`
   (un `OPENAI_BASE_URL=` vacío en `.env` no debe pisar el default del SDK).
4. **Sonda**: `go run ./cmd/llmcheck` con un texto de referencia permite ver la
   respuesta cruda y los tiempos sin pasar por la API.

---

## Escenario 7 — Jobs del provisioner

```bash
cd apps/backend
go run ./cmd/provisioner refresh-aggregates   # reconstruye error_metrics desde analyses
go run ./cmd/provisioner purge-raw-data       # borra prácticas borradas fuera de retención
go run ./cmd/provisioner execute-deletions    # materializa el derecho al olvido (A9)
```

- `APP_RAW_RETENTION_DAYS` / `APP_DELETION_GRACE_DAYS` (default 30) gobiernan
  las ventanas.
- `refresh-aggregates` es **reconciliación total**: reconstruye `error_metrics`
  desde la fuente de verdad. Es seguro re-ejecutarlo.
- **Clave**: si la pseudonimización se ejecuta con un `APP_PSEUDONYM_SECRET`
  distinto, las métricas se recalculan con la clave nueva (ver
  [`SECRET_ROTATION.md`](SECRET_ROTATION.md)).

---

## Escenario 8 — La caché remota de Turborepo no responde

**Síntoma**: los builds van lentos o `turbo` avisa de que no puede conectar.

1. Comprobar el contenedor `turbo-cache` y su puerto `:3300`.
2. El cliente lee `TURBO_API`/`TURBO_TOKEN`/`TURBO_TEAM`; el token debe coincidir
   con `TURBO_REMOTE_CACHE_TOKEN` del servicio.
3. Si no hay caché remota (p. ej. en un runner *hosted* sin acceso a la red
   local), Turborepo **usa solo la caché local y no falla**: el impacto es
   únicamente de velocidad. Ver `ops/docker/README.md`.

---

## Escenario 9 — Rotación de secretos

Ver el procedimiento por secreto en [`SECRET_ROTATION.md`](SECRET_ROTATION.md).
Recordatorio: rotar `APP_PSEUDONYM_SECRET` invalida los pseudónimos existentes
hasta re-ejecutar `refresh-aggregates`.

---

## Escenario 10 — CI rojo

| Fallo | Causa probable | Acción |
|---|---|---|
| Job `contracts` | `gen_*.go`/`gen.ts` desincronizados del `api.yaml` | `pnpm generate` y commitear |
| `check-version` | `info.version` ≠ `package.json` | alinear ambos (AP-MR6) |
| Job `backend`/`frontend` | lint, test o build | reproducir local: `pnpm lint`, `go test -race ./...`, `pnpm build` |
| Job `security` | `gosec`/`govulncheck` con hallazgos | revisar SARIF; actualizar toolchain/deps o justificar |
| `security_audit.sh` | secreto hardcodeado o docs sin guard | rotar el valor / gatear el render |

---

**Última revisión**: 2026-09-21.
