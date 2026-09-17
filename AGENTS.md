# AGENTS.md — Instrucciones para Agentes de IA

> Archivo de onboarding y memoria de trabajo para cualquier agente de IA que trabaje en este repositorio. **Leer SIEMPRE al inicio de cada sesión.**

## Estado del proyecto

- **Fase actual**: Fase 2 — Dominio puro **completada** (gate de salida en verde). Raíz del dominio (`ID` UUID v7, errores A5, `ErrorPattern` + enums, eventos `IdentityIssued`/`PracticeCreated`/`AnalysisCompleted`/`AnalysisFailed`) y bounded contexts `identity/`, `practice/`, `analysis/` y `analytics/` con **100%** de cobertura.
- **Siguiente item**: Fase 3 — Puertos y adaptadores (`docs/checklist/03-*.md`): puertos `LLMExtractor`, repositorios, `UnitOfWork` y outbox en `internal/api/ports/` + adaptadores Postgres (testcontainers).
- **Estado**: monorepo base, contrato OpenAPI (`apps/contracts/openapi/api.yaml`), pipeline `pnpm generate` y ahora el módulo Go (`apps/backend/go.mod`, module path `github.com/c0mp1lerworld/langlint/backend`). `gen_*.go`/`gen.ts` commiteados. Dominio puro stdlib-only; `go build/vet/test` en verde.

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
