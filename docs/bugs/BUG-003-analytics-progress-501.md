# BUG-003 — `GET /analytics/progress` responde `501`

- **Estado**: Cerrado
- **Área**: Backend (API) + analíticas · `internal/api/handlers/server.go`
- **Detectado**: 2026-09-19
- **Cerrado**: 2026-09-21

## Qué pasó

El contrato declara la operación `get_progress_series` (`GET /analytics/progress`)
con respuesta `200 ProgressSeries`, y el dominio define `analytics.ProgressMetric`
(con tests). Sin embargo, el handler responde `501 not_implemented`:

```go
// GetProgressSeries handles GET /analytics/progress. Deferred to Fase 6
func (s *Server) GetProgressSeries(w http.ResponseWriter, _ *http.Request, _ GetProgressSeriesParams) {
	writeNotImplemented(w)
}
```

Además, el job `refresh-aggregates` **solo** reconstruye `error_metrics`; no hay
repositorio ni agregación de progreso. La Fase 6 no incluyó un item para
materializarlo, así que el endpoint quedó pendiente.

El dashboard (`/analytics`) consume la query y muestra un placeholder "La serie
de progreso estará disponible próximamente".

## Lección

Un endpoint anunciado en el contrato pero no implementado es un **gap visible**
que el client-team descubre en integración. El contrato lo mitiga declarando
`501`, pero persiste la deuda funcional. Debe rastrearse aquí y en
[`../API_CONTRACT.md`](../API_CONTRACT.md) (AP5).

## Regla / plan de cierre

- **Implementado**: la serie se deriva **on-read** de las `analyses` completadas
  (la verdad append-only), sin tabla materializada. `ProgressRepository`
  (`ListSamplesByUser`) + `analytics.BuildProgressSeries` (bucketing puro) y el
  handler devuelve `200 ProgressSeries`. El contrato `3.1.0` retiró el `501`.
- **Sin reimplementar el cálculo en el frontend** (F11): el backend sigue siendo
  la fuente de verdad; el dashboard consume `points` y ya no muestra placeholder.

## Referencias

- [`../API_CONTRACT.md`](../API_CONTRACT.md) — GAP-1.
- `docs/PRODUCT_DOMAIN.md` §4.2.4 (`ProgressMetric`) y §7.1.
- `apps/backend/internal/domain/analytics/progress_metric.go`.
- `apps/frontend/src/lib/query/analytics.ts` (`useProgressSeries`, `retry: false`).
