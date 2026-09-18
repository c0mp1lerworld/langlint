# Checklist — Fase 3: Puertos y Adaptadores (Postgres)

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 3
> **Predecesora**: Fase 2 (Dominio puro) · **Sucesora**: Fase 4 (Motor de IA)

---

## 3.1 Puertos (interfaces)

- [x] `3.1.1` Crear puertos de repositorio en `internal/api/ports/storage/` (`PracticeRepository`, `AnalysisRepository`, `ErrorMetricRepository`).
- [x] `3.1.2` Crear puerto `UnitOfWork` (`InTransaction(ctx, fn) error`) en `ports/storage/unit_of_work.go`.
- [x] `3.1.3` Crear puerto `LLMExtractor` (`Extract(ctx, req) ([]analysis.Fragment, error)`) en `ports/` (`Fragment` vive en el BC `analysis/`, no en la raíz `domain`).
- [x] `3.1.4` Crear puertos de eventos en `ports/events/`: `EventDispatcher`, `EventHandler`, `Outbox`.
- [x] `3.1.5` Generar mocks con `mockgen` en `ports/mocks/` (A10).

## 3.2 Repositorios Postgres

- [ ] `3.2.1` Implementar `PostgresPracticeRepository`, `PostgresAnalysisRepository`, `PostgresErrorMetricRepository` en `adapters/postgres/repositories/`.
- [ ] `3.2.2` Cada método detecta la transacción vía helper `ctxTx(ctx)` (usa tx si existe, o conexión del pool) (§5.3).
- [ ] `3.2.3` Crear migraciones `goose` (`migrations/NNNNNN_<slug>.up.sql`/`.down.sql`) para `practices`, `analyses`, `error_metrics`, `outbox_events`.
- [ ] `3.2.4` Mapear errores de infraestructura (`pgx.ErrNoRows`) a errores de dominio (A5). **Sin** filtrar `pgx` al service.

## 3.3 Unit of Work

- [ ] `3.3.1` Implementar `PostgresUnitOfWork` (`pool.Begin` → `fn(txCtx)` → `Commit`/`Rollback`) (§5.3).
- [ ] `3.3.2` Crear `tx_context.go` (tipo `txKey` privado + helper `ctxTx`).
- [ ] `3.3.3` Verificar que el service **nunca** importa `pgx` ni inicia transacciones (AP8, A1/A2).

## 3.4 Outbox + Event Bus

- [ ] `3.4.1` Implementar `InMemoryEventDispatcher` (channel buffered + worker pool + backpressure) (§5.1).
- [ ] `3.4.2` Implementar `OutboxRelay` (lee `outbox_events` no publicados, entrega al dispatcher, marca `published_at`/`attempts`).
- [ ] `3.4.3` Los services publican vía `outbox.Append(...)` **dentro** de la transacción; nunca `dispatcher.Dispatch(...)` directo (AP7).
- [ ] `3.4.4` Handlers idempotentes (pueden ejecutarse 2 veces sin efecto adverso).

---

## ✅ Gate de salida

- [ ] Cobertura de services **≥90%** y adapters **≥70%** (A10).
- [ ] Test Tier 3 (`//go:build integration`) con `testcontainers-go` (Postgres 16) en verde: outbox relay end-to-end y rollback del UoW.
- [ ] `pnpm test-integration --filter=backend` sin `t.Skip()`.
- [ ] Race detector ON en todos los tiers.

## Fuente normativa

- **A2, A4, A6, A10**, **AP7, AP8**, **AP-MR8** (testcontainers obligatorio).
- **MANIFEST_MONOREPO.md** §5.1 (outbox), §5.2 (testing), §5.3 (UoW).
