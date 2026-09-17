# Checklist — Fase 5: Frontend (Next.js)

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 5
> **Predecesora**: Fase 4 (Motor de IA) · **Sucesora**: Fase 6 (Provisioner y privacidad)

---

## 5.1 Base Next.js

- [ ] `5.1.1` Inicializar `apps/frontend/` con Next.js 14+ (App Router), TypeScript `strict: true`, Tailwind CSS.
- [ ] `5.1.2` Crear estructura canónica `src/{app,components,features,lib,types}` (MANIFEST_FRONTEND §3.1).
- [ ] `5.1.3` Configurar `tsconfig.json` con `paths` `@/* → ./src/*`, `noUncheckedIndexedAccess`, `noFallthroughCasesInSwitch`.
- [ ] `5.1.4` Configurar `next.config.js` con `reactStrictMode` y `typedRoutes`.
- [ ] `5.1.5` Env vars: solo `NEXT_PUBLIC_*` expuestas al cliente; secretos server-only (AP-F8).

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
