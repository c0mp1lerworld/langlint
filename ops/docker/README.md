# `ops/docker` — infraestructura de desarrollo

`docker-compose.yml` levanta la infraestructura local del proyecto:

| Servicio | Puerto host | Propósito |
|---|---|---|
| `postgres` | `5433` | Postgres 16 de desarrollo (`APP_DATABASE_URL`). |
| `turbo-cache` | `3300` | Remote cache self-hosted de Turborepo (7.1.4, AP-MR7). |

```bash
docker compose -f ops/docker/docker-compose.yml up -d
```

## Remote cache de Turborepo (`turbo-cache`)

Implementación OSS [`ducktors/turborepo-remote-cache`](https://github.com/ducktors/turborepo-remote-cache)
con `STORAGE_PROVIDER=local` (volumen `langlint_turbo_cache`). El token del
servidor se toma de `TURBO_REMOTE_CACHE_TOKEN` (default `local-dev-token`).

### Uso local

El cliente lee `TURBO_API`, `TURBO_TOKEN` y `TURBO_TEAM` del entorno. Para usar
la caché local:

```bash
export TURBO_API=http://localhost:3300
export TURBO_TOKEN=local-dev-token   # debe coincidir con TURBO_REMOTE_CACHE_TOKEN
export TURBO_TEAM=langlint
pnpm build
```

Sin esas variables, Turborepo usa solo la caché local (comportamiento por
defecto) y no falla.

### CI (GitHub Actions)

Los workflows `.github/workflows/ci.yml` y `contracts.yml` leen
`TURBO_API`/`TURBO_TEAM`/`TURBO_TOKEN` de **secrets del repositorio**. Un humano
con permiso de admin debe crearlos (Settings → Secrets and variables → Actions).
Si no existen, quedan vacíos y la caché remota simplemente se omite.

> **Caveat operativo**: los runners *hosted* de GitHub solo alcanzan la caché si
> el endpoint es públicamente accesible. Para una caché en la red local/cluster
> hay que usar **self-hosted runners** (o exponer el servicio con TLS + token).
> Objetivo de AP-MR7: *cache hit ratio* > 80 % tras la primera semana.

### Staging / cluster

En staging se despliega la misma imagen con almacenamiento S3 (`STORAGE_PROVIDER=s3`,
`STORAGE_PATH=<bucket>`, credenciales AWS) detrás de un endpoint con TLS. Luego
se apuntan los secrets `TURBO_API`/`TURBO_TOKEN`/`TURBO_TEAM` a ese endpoint.

Opcional (endurecimiento): fijar `TURBO_REMOTE_CACHE_SIGNATURE_KEY` **idéntico**
en servidor y clientes para firmar/verificar los artefactos de caché.
