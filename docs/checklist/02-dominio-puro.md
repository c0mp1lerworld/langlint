# Checklist — Fase 2: Dominio Puro (DDD)

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 2
> **Predecesora**: Fase 1 (Fundaciones) · **Sucesora**: Fase 3 (Puertos y adaptadores)

---

## 2.1 Bounded context `identity/`

- [ ] `2.1.1` Crear entidad `User` (identidad portable) con `ID` (UUID v7), `Email`, `CreatedAt` (PRODUCT_DOMAIN §4.2.1).
- [ ] `2.1.2` Crear value object `Email` con validación.
- [ ] `2.1.3` Definir evento `IdentityIssued` (payload `UserID`).
- [ ] `2.1.4` La identidad portable **sin** `tenant_id` (AP1).

## 2.2 Bounded context `practice/`

- [ ] `2.2.1` Crear agregado `Practice` (`SourceText`, `DraftText`, `TargetRules`, `Status`, timestamps).
- [ ] `2.2.2` Crear value objects `SourceText`, `DraftText`, `TargetRule`, `PracticeStatus`.
- [ ] `2.2.3` Implementar invariantes: textos no vacíos, ≥1 `TargetRule`, transición `draft → analyzing → completed|failed`.
- [ ] `2.2.4` Definir evento `PracticeCreated` (payload `PracticeID`, `UserID`).

## 2.3 Bounded context `analysis/`

- [ ] `2.3.1` Crear agregado `Analysis` (`PracticeID` por referencia, `Fragments`, `Model`, `ModelVersion`, `Status`).
- [ ] `2.3.2` Crear value object `Fragment` (7 campos del schema §5.2).
- [ ] `2.3.3` Crear value object `ErrorPattern` (`code` enum + `severity` + `note`).
- [ ] `2.3.4` Invariante: `Fragments` no vacío cuando `Status == completed`.
- [ ] `2.3.5` Definir eventos `AnalysisCompleted` y `AnalysisFailed`.

## 2.4 Bounded context `analytics/`

- [ ] `2.4.1` Crear agregados `ErrorMetric` (`Code`, `Window`, `Count`, `LastSeenAt`) y `ProgressMetric` (`TotalFragments`, `ErrorCount`, `Accuracy`).
- [ ] `2.4.2` Crear value object `Window` (`day|week|month`).

## 2.5 Errores, identificadores y eventos compartidos (raíz del dominio)

- [ ] `2.5.1` Crear `identifiers.go` con UUID v7 usando **solo** `crypto/rand`, `time`, `fmt` (A7). API: `NewID`, `MustNewID`, `IsValid`, `Version`.
- [ ] `2.5.2` Crear `errors.go` con `ValidationError`, `NotFoundError`, `AnalysisPendingError`, `AnalysisFailedError`, `LLMUnavailableError` (A5).
- [ ] `2.5.3` Crear `events.go` con los structs `DomainEvent` inmutables + campo `Version`.
- [ ] `2.5.4` Verificar que **ningún** archivo del dominio importa fuera de stdlib (regla `domain_purity`, A1).
- [ ] `2.5.5` Verificar que ningún bounded context importa a otro (regla `forbidden_imports`, A3).
- [ ] `2.5.6` Todos los fields exportados con tag `json:"snake_case"` explícito (AP2).

---

## ✅ Gate de salida

- [ ] Cobertura del dominio **100%** (`go test -cover` en verde, gate CI, A10).
- [ ] `go test -race ./...` sin fallos en el dominio.
- [ ] Smoke check `purity` (grep de imports) sin violaciones.
- [ ] Nombres de test según convención `TestXxx_Method_Condition_ExpectedResult` (A10).

## Fuente normativa

- **A1, A3, A5, A7, A10**, **AP1, AP2, AP3**.
- **PRODUCT_DOMAIN.md** §4 (modelo de dominio).
