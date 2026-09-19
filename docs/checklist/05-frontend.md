# Checklist — Fase 5: Frontend (Next.js)

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 5
> **Predecesora**: Fase 4 (Motor de IA) · **Sucesora**: Fase 6 (Provisioner y privacidad)

---

## 5.0 Cableado HTTP del backend (prerequisito de 5.3/5.4)

> Bloque de cierre del gap detectado: las Fases 1–4 nunca cubrieron `cmd/api`, los handlers chi ni los services de práctica. Se implementa solo la superficie que consume el frontend; `/analytics/progress` y `/me/*` (A9) se difieren a Fase 6.

- [x] `5.0.1` A12: `ErrorResponse.code` + `not_implemented` e `internal`; respuestas `501` en `/me/*` y `/analytics/progress`. _(`pnpm generate` idempotente; `gen_types.go`/`gen.ts` regenerados. `internal` cierra el drift de §4.5.)_
- [x] `5.0.2` Dominio: evento `AnalysisRequested` (trigger del análisis) + registro `NewEvent` + tests. _(`domain/events.go`; stdlib-only, A1/A3. Distinto de `PracticeCreated`.)_
- [x] `5.0.3` `shared/db` (pool pgx + ping) + `cmd/migrate` (goose embebido). _(`migrations.Up` reutiliza el runner existente; migración idempotente.)_
- [x] `5.0.4` `shared/config` server + `shared/logger` cableando `PIIHandler`. _(`APP_DATABASE_URL`/`APP_HTTP_ADDR`/`APP_USER_ID`/`APP_LLM_TIMEOUT`; `logger.New` es el único constructor.)_
- [x] `5.0.5` `internal/api/handlers/errors.go` (map A5 → HTTP/`ErrorResponse`) + `shared/httpx` (router chi + middlewares + resolvedor de usuario). _(El mapper vive en `handlers/` para respetar AP-MR2: los tipos de wire solo se importan ahí. Usuario único vía `APP_USER_ID`; sin auth en el MVP.)_
- [x] `5.0.6` Services de práctica: `CreatePracticeService`, get/list/update/delete y `AnalyzePractice`. _(`PracticeService`: create/analyze con `outbox.Append` en tx (AP7/AP8); ownership ajeno → `NotFoundError`; analyze ya en curso → `AnalysisPendingError` (AP4).)_
- [x] `5.0.7` Handlers de práctica que implementan `ServerInterface` + conversión dominio↔wire. _(`handlers/server.go`; `Retry-After` en el 202; `/me/*` y progress → 501 `not_implemented`.)_
- [x] `5.0.8` Event handlers: `AnalysisRequested` → `RunAnalysis`; `AnalysisCompleted` → `MarkCompleted` + `Upsert ErrorMetric` (day/week/month); `AnalysisFailed` → `MarkFailed`. _(Idempotentes; cobertura 91.4%.)_
- [x] `5.0.9` Handler de analytics `error-patterns` (`ErrorMetricRepository.ListByUser`) + stubs 501. _(`progress`, `access-log`, `export`, `delete` diferidos a Fase 6.)_
- [x] `5.0.10` `di/` (Uber Fx v1.24.0) + `cmd/api` (migrate → router → outbox relay → serve). _(`newPool` migra en el arranque; dispatcher + relay con el ciclo de vida; `cmd/api` con config y `PIIHandler`.)_
- [x] `5.0.11` Tests + gate. _(services **97.5%**, event_handlers **91.4%**, dominio 100%; e2e Tier 3 del flujo create→analyze→poll→completed→analytics; smoke del grafo Fx. `go build/vet/test -race` (+ `-tags=integration`) en verde.)_

## 5.1 Base Next.js

- [x] `5.1.1` Inicializar `apps/frontend/` con Next.js 14+ (App Router), TypeScript `strict: true`, Tailwind CSS. _(Next 15.5.25 (App Router), React 19.3, TypeScript 5.9.3 strict y Tailwind v4.3 (CSS-first vía `@tailwindcss/postcss`). Se renombra el paquete `@langlint/frontend` → `frontend` para que `--filter=frontend` (gate) resuelva, alineado con `backend`. `src/app/{layout,page,globals.css}` + `pnpm build` en verde.)_
- [x] `5.1.2` Crear estructura canónica `src/{app,components,features,lib,types}` (MANIFEST_FRONTEND §3.1). _(`src/app`, `src/components/{ui,feature}`, `src/features`, `src/lib{/api(gen.ts),/query,/store,/utils}`, `src/types`. Subcarpetas vacías con `.gitkeep`; `src/types/index.ts` re-exporta `components`/`paths`/`operations` desde `lib/api/gen.ts` (F1).)_
- [x] `5.1.3` Configurar `tsconfig.json` con `paths` `@/* → ./src/*`, `noUncheckedIndexedAccess`, `noFallthroughCasesInSwitch`. _(`strict: true`, `target: ES2022`, `moduleResolution: bundler`, `jsx: preserve`, plugin `next`; `pnpm typecheck --filter=frontend` en verde.)_
- [x] `5.1.4` Configurar `next.config.js` con `reactStrictMode` y `typedRoutes`. _(`reactStrictMode: true` + `typedRoutes: true` **top-level**: Next 15.5 promovió typedRoutes a estable (en `experimental` emite warning de deprecación).)_
- [x] `5.1.5` Env vars: solo `NEXT_PUBLIC_*` expuestas al cliente; secretos server-only (AP-F8). _(`apps/frontend/.env.example` documenta `NEXT_PUBLIC_API_URL` (única expuesta) y el prefijo server-only `FRONTEND_*`; `.env.example` sigue trackeable por `.gitignore`.)_

## 5.2 Cliente, queries y store

- [ ] `5.2.1` Generar `src/lib/api/gen.ts` (openapi-typescript, commiteado, no editable) — F1.
- [ ] `5.2.2` Implementar `lib/api/client.ts` (fetch wrapper con auth + base URL) y `lib/api/errors.ts` (mapper de errores → UX) — F5.
- [ ] `5.2.3` Tratar códigos idempotentes (`409 analysis_pending`) como éxito (AP-F6).
- [ ] `5.2.4` Configurar TanStack Query (`lib/query/`) con `queryClient` y defaults (staleTime/gcTime) — F7.
- [ ] `5.2.5` Configurar Zustand (`lib/store/`) para client state (UI efímera), **sin** cachear datos del API — F7/AP-F4.
- [ ] `5.2.6` Schemas Zod inferidos de los tipos del contrato (formularios) — F8.

## 5.3 Vista diff de 3 columnas

- [ ] `5.3.1` Implementar componente de diff: columna Español | Borrador usuario | Corrección IA (PRODUCT_DOMAIN §8.1).
- [ ] `5.3.2` Resaltar diff rojo/verde y paneles expandibles con `target_verb_review`, `lexical_clarification`, `grammar_explanation`.
- [ ] `5.3.3` Polling del estado de análisis con TanStack Query (refetchInterval) + optimistic update con rollback — F9.
- [ ] `5.3.4` Accesibilidad WCAG 2.2 AA: navegación por teclado, contraste, `aria` en tooltips — F6.

## 5.4 Dashboard de analíticas

- [ ] `5.4.1` Consumir `GET /v1/analytics/error-patterns` y `/progress` (tipos desde `gen.ts`).
- [ ] `5.4.2` Visualizar frecuencia de `ErrorPattern` y alerta del error recurrente (PRODUCT_DOMAIN §8.1).
- [ ] `5.4.3` El cliente **no** reimplementa reglas de clasificación/análisis (F11).

## 5.5 Tests

- [ ] `5.5.1` Vitest (unit) para lógica pura (helpers, schemas, mappers) — F10.
- [ ] `5.5.2` React Testing Library (componentes) con axe-core para accesibilidad — F6/F10.
- [ ] `5.5.3` Playwright (e2e) cubriendo el flujo crítica: crear práctica → análisis → diff.
- [ ] `5.5.4` Sin `skip()` en ningún tier (F10).

---

## ✅ Gate de salida

- [ ] `pnpm typecheck --filter=frontend` (tsc --noEmit) en verde.
- [ ] `pnpm lint` y `pnpm test` en verde (F1–F12 satisfechos).
- [ ] `pnpm build` (next build) en verde con dependencia de `generate`.
- [ ] Sin imports cruzados hacia `apps/backend/` (F3, AP-F2); sin `useEffect` para fetching (AP-F3).

## Fuente normativa

- **F1–F12**, **AP-F1..AP-F10**.
- **PRODUCT_DOMAIN.md** §8 (frontend).
