# AGENTS.md — Instrucciones para Agentes de IA

> Archivo de onboarding y memoria de trabajo para cualquier agente de IA que trabaje en este repositorio. **Leer SIEMPRE al inicio de cada sesión.**

## Estado del proyecto

- **Fase actual**: Fase 7 — Hardening; bloque CI/CD `7.1` (`7.1.1`–`7.1.5`) **completado**: `ci.yml` (detect-changes + jobs contracts/backend/frontend, Tier 3 en nightly/push/label), `contracts.yml` (regenera Go+TS y falla por drift; valida `info.version == package.json`), `deploy-staging.yml` + Dockerfiles (backend distroless `nonroot`, frontend `standalone`) y remote cache **self-hosted** (`ops/docker/`, `ducktors/turborepo-remote-cache`). El *Gate de salida* de Fase 7 sigue abierto: faltan `7.2`/`7.3` y el primer run real de los workflows (no corren en local). Historial: Fase 6 — Provisioner y privacidad **completada; Gate de salida en verde**. Bloques `6.1` (jobs batch `refresh-aggregates`, `purge-raw-data`, `execute-deletions`), `6.2` (endpoints A9 `GET /me/data/export`, `DELETE /me/data`, `GET /me/access-log`) y `6.3` (pseudonimización HMAC-SHA256 de analytics con `APP_PSEUDONYM_SECRET`, separación de datos crudos y auditoría `ops/scripts/pii_audit.sh`) completados. Mejora post-MVP **Fase 9 — feedback profundo y práctica activa** completada en `9.1`–`9.6` (explicaciones estructuradas + quiz IA); `9.7` (analytics del quiz) queda pendiente por diseño (ver checklist 09).
- **Siguiente item**: Fase 7.2 — Seguridad (`security.yml` con `gosec`/`govulncheck`, `ops/scripts/security_audit.sh`, `/docs` no expuesto en producción y `x-internal: true` filtrado del render). Acción humana pendiente: crear los secrets `TURBO_API`/`TURBO_TEAM`/`TURBO_TOKEN` (y la variable `STAGING_API_URL`) para activar la caché remota y el inlining del API en staging. Gap conocido: `/analytics/progress` sigue en `501` (no tiene item en el checklist 06). Nota: el checklist 07 (`7.2.2`/`7.3.3`) da por pendientes `pii_audit.sh` y `README.md`, que **ya existen**; reconciliar antes de ejecutar. Gap conocido adicional (candidato de hardening): la extracción es por frase del borrador, así que una frase run-on sin `. ! ?` internos va en un único fragmento grande; falta subdividir run-ons por sub-cláusulas deterministas validadas contra la frase original.
- **Estado**: monorepo base, contrato OpenAPI (`apps/contracts/openapi/api.yaml`), pipeline `pnpm generate`, módulo Go (`apps/backend/go.mod`, module path `github.com/c0mp1lerworld/langlint/backend`), Fases 3–4 (adaptadores Postgres, UoW, outbox; motor de IA), Fase 5.0–5.5 (frontend Next.js 15 con diff de 3 columnas, dashboard de analíticas y suite de tests: Vitest 95 tests + Playwright e2e mockeado), Fase 6.1 (binario `cmd/provisioner` con sus jobs batch, tabla `deletion_requests`, migración `000002` y harness Tier 3 en `internal/shared/testdb`) y Fase 6.2 (endpoints A9: `IdentityService`, `access_events`/migración `000003`, middleware `internal/api/accesslog`, `APP_USER_EMAIL`) y Fase 6.3 (pseudonimización HMAC-SHA256 de `error_metrics.user_id` con `APP_PSEUDONYM_SECRET`/migración `000004`, `internal/shared/pseudonymizer` y `ops/scripts/pii_audit.sh`). Contrato `apps/contracts` en **`3.0.0`**: `Fragment` expone `target_verb_reviews`/`lexical_clarifications`/`grammar_explanations` como arrays (varias entradas por fragmento, lista vacía permitida). Añadidos en la verificación de 5.4: CORS configurable (`APP_CORS_ALLOWED_ORIGINS`, default `http://localhost:3000`) y fix del `fetchImpl` del cliente. El frontend requiere `apps/frontend/.env.local` con `NEXT_PUBLIC_API_URL` (gitignored). El provisioner usa `APP_DATABASE_URL` + `APP_RAW_RETENTION_DAYS`/`APP_DELETION_GRACE_DAYS` (default 30); `cmd/api` requiere `APP_USER_EMAIL` (además de `APP_USER_ID`); ambos requieren `APP_PSEUDONYM_SECRET`. `gen_*.go`/`gen.ts` commiteados. Dominio puro stdlib-only; `go build/vet/test` en verde; `pnpm build`/`pnpm lint` (3 apps), `pnpm typecheck --filter=frontend`, `pnpm test`, `pnpm test-integration --filter=backend` y `pnpm test:e2e` en verde.

## Primeros pasos obligatorios al iniciar una sesión

1. Leer este archivo (`AGENTS.md`).
2. Leer el checklist de la fase actual (`docs/checklist/XX-*.md`).
3. Ejecutar `git status` y `git log --oneline -10`.
4. Leer `docs/DEVLOG.md` (últimas entradas, bloqueos, próximos pasos).

## Comandos canónicos

> La mayoría se definen en Fase 1. Antes de que existan, **no inventar** comandos alternativos.

| Comando | Propósito | Disponible |
|---|---|---|
| `pnpm install --frozen-lockfile` | Instalar dependencias del monorepo | Fase 1+ |
| `pnpm generate` | Regenerar tipos Go + TS desde el contrato (A12) | Fase 1+ |
| `pnpm build` | Build de todo el monorepo (Turborepo) | Fase 1+ |
| `pnpm lint` | Lint del monorepo | Fase 1+ |
| `pnpm test` | Tests Tier 1+2 | Fase 2+ |
| `pnpm test-integration --filter=backend` | Tests Tier 3 (testcontainers) | Fase 3+ |
| `pnpm typecheck` | `tsc --noEmit` del frontend | Fase 5+ |
| `make test` / `make generate` | Targets Go del backend | Fase 2+ |
| `go test -race -count=1 ./...` | Unit Go con race detector | Fase 2+ |
| `go run ./cmd/migrate` | Aplica las migraciones goose (backend) | Fase 5.0+ |
| `go run ./cmd/api` | Arranca el servidor HTTP (requiere Postgres + `.env`) | Fase 5.0+ |

## Reglas inquebrantables (resumen de los manifiestos)

1. **Contract-first (A12)**: todo cambio de wire empieza en `apps/contracts/openapi/api.yaml`, luego `pnpm generate`, luego código. Nunca editar `gen_*.go` ni `gen.ts` a mano.
2. **Dominio puro (A1)**: `apps/backend/internal/domain/` solo importa stdlib. Nada de LLM, DB, HTTP.
3. **Dirección de dependencias (A2)**: `cmd → adapters → services → ports → domain`.
4. **Bounded contexts aislados (A3)**: `identity`, `practice`, `analysis`, `analytics` (y futuro `tutor`) no se importan entre sí.
5. **Errores de dominio (A5)**: los services nunca retornan `error` genérico.
6. **Outbox (AP7)**: nunca `dispatcher.Dispatch()` dentro de un `UnitOfWork`; siempre `outbox.Append()`.
7. **UoW (AP8)**: los services nunca inician transacciones SQL.
8. **PII (A8)**: anonimizar antes del LLM, pseudonimizar en analytics, no loguear PII.
9. **Sin imports cruzados (F3)**: frontend solo habla con backend vía HTTP.
10. **Sin `t.Skip()`, sin `useEffect` para fetching** (A10, AP-F3).

## Convenciones de código

- **Go**: tags `json:"snake_case"` explícitos; nombres de test `TestXxx_Method_Condition_ExpectedResult`; mocks con `mockgen`; un solo `go.mod` (A11).
- **TypeScript**: tipos de wire solo desde `gen.ts` (F1); capas `app → features → components → lib → types` (F2); server state en TanStack Query, client state en Zustand (F7).
- **Idioma**: código y comentarios de negocio en inglés; documentación y commits en español (consistente con `docs/`).

## Protocolo de trabajo (obligatorio)

1. Trabajar **una fase a la vez**; no pasar de fase sin el `Gate de salida` en verde.
2. Incrementos atómicos: 1–3 items del checklist → verificar → commit.
3. Preferir **TDD**: escribir el test fallante primero (especificación objetiva).
4. Al terminar cada sesión: actualizar `docs/DEVLOG.md` y dejar el repo con commit limpio.
5. Si un comando de verificación no existe aún, **preguntar** en lugar de inventar.
6. No asumir librerías: verificar en `go.mod` / `package.json` antes de importar.
7. No commitear secretos ni PII.

## Fuentes autoritativas

- `MANIFEST_MONOREPO.md` (axiomas A1–A12, anti-patrones, patrones transversales).
- `MANIFEST_FRONTEND.md` (axiomas F1–F12).
- `docs/PRODUCT_DOMAIN.md` (modelo de dominio).
- `docs/GUIDE_WORK_IA.md` (metodología de colaboración humano-IA).
- `docs/checklist/` (fases con gate de salida).
