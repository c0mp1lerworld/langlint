# Checklist — Fase 4: Motor de Análisis Cognitivo (IA)

> **Estado**: MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12` — Paso 4
> **Predecesora**: Fase 3 (Puertos y adaptadores) · **Sucesora**: Fase 5 (Frontend)

---

## 4.1 Puerto `LLMExtractor` (OpenAI)

- [x] `4.1.1` Implementar `OpenAIExtractor` (adapter) con `openai-go` v1.12.0. _(En `adapters/llm/`; `Extract` en modo JSON simple (parsea `message.content`). El JSON Schema estricto de Structured Outputs se añade en 4.2.1. Runner manual `cmd/llmcheck` verificado contra `gpt-4o-mini`.)_
- [x] `4.1.2` Construir el prompt desde `Practice` (source + draft + target rules) anonimizado. _(`buildPrompt` puro en `prompt.go`; consume el input ya anonimizado de `ExtractRequest`, A8. Fragmentos verbatim, explicaciones en español y glosario de la taxonomía. `PromptText` permite auditarlo. La anonimización en sí es 4.3.1.)_
- [x] `4.1.3` Inyectar el adapter vía el puerto `LLMExtractor`; el dominio **no** conoce al proveedor (A1). _(Constructor + `var _ ports.LLMExtractor`; verificado con `go list`: `openai` no aparece en `internal/domain/...`.)_

## 4.2 Structured Outputs (JSON Mode)

- [x] `4.2.1` Configurar Structured Outputs con el JSON Schema de `Fragment[]` (PRODUCT_DOMAIN §5.2). _(`adapters/llm/schema.go`: `fragmentSchema()` espejo de §5.2 + `fragmentResponseFormat()` con `strict: true`; enums de `code`/`severity` derivados de las constantes del dominio. Nota: Structured Outputs exige raíz `type: "object"`, así que `Fragment[]` se envuelve como `{"fragments":[...]}` y el adapter lo desenvuelve; openai-go v1.12.0 no expone `Message.Parsed`.)_
- [x] `4.2.2` Validar el JSON del LLM contra el schema **antes** de persistir; output inválido → `LLMUnavailableError`. _(`adapters/llm/validate.go`: campos de texto requeridos no vacíos + enums válidos → `*domain.LLMUnavailableError`. Sin librería externa de JSON Schema, manifiesto §2.1.)_
- [x] `4.2.3` La llamada al LLM se ejecuta **fuera** de la transacción (fuera del `UnitOfWork`). _(`services/analysis_service.go`: `Extract` antes de `InTransaction`; test con context marker prueba que corre fuera de la tx.)_
- [x] `4.2.4` Persistir `Analysis` + outbox `AnalysisCompleted` atómicamente tras recibir el resultado (PRODUCT_DOMAIN §4.7 Flujo 2). _(`Analysis` completed + `outbox.Append(AnalysisCompleted)` en la misma `InTransaction`; el camino de fallo (`AnalysisFailed`) queda para 4.3.3.)_

## 4.3 Anonimización, contexto y timeouts

- [x] `4.3.1` Implementar `anonymizer.go` que elimina emails, teléfonos y nombres propios antes de enviar al LLM (A8). _(`services/anonymizer.go`: `Anonymize` (emails → `[email]`, teléfonos → `[phone]`, nombres de lista curada ES/EN → `[name]`) y `AnonymizeSpanish` (+ heurística de mayúsculas a mitad de frase, solo para el source es). `AnalysisService` anonimiza antes de `Extract`.)_
- [x] `4.3.2` Toda goroutine del extractor recibe `context.Context` y chequea `ctx.Done()` (AP6, regla `goroutine_context`). _(Sin cambios: no existen goroutines propias en `internal/` (`grep "go func"` → 0); `Extract` es síncrono y propaga `ctx` al SDK. AP6 se cumple por construcción; la regla `goroutine_context` es de Fase 7.)_
- [x] `4.3.3` Configurar timeout de la llamada; ante fallo emitir `AnalysisFailed` (sin filtrar error crudo). _(`analysis_service.go`: `context.WithTimeout` (default 60s) alrededor del LLM; ante fallo (`Extract`/`Complete`) `Analysis.Fail()` + `Analysis.Save` + `outbox.Append(AnalysisFailed{Reason genérico})` en una tx, retorno `nil` (fallo manejado, sin reintento del relay); el error crudo no se filtra.)_
- [x] `4.3.4` El `PIIHandler` descarta logs con contenido libre (>100 chars) y patrones PII (A8). _(`shared/logger/pii_handler.go`: decorador de `slog.Handler` que descarta mensajes/attrs con texto libre >100 runas, email, teléfono, API key (`sk-`/`pk-`/`AKIA`) o JWT; `WithAttrs` elimina atributos sensibles. No se cablea aún: no existe `cmd/api` (Fase 5).)_

---

## ✅ Gate de salida

- [x] Análisis fragmentado devuelto y validado contra el schema (sin `Fragment` malformado persistido). _(4.2.1/4.2.2.)_
- [x] Test con output inválido del LLM → `LLMUnavailableError` (no se persiste basura). _(4.2.2.)_
- [x] Cambio de proveedor no requiere tocar `internal/domain/` (A1 verificado). _(`go list -deps ./internal/domain/... | grep -c openai` → 0.)_
- [x] `go test -race ./...` en verde en el motor de IA. _(Tier 1 + Tier 3; `go test -race -count=1 ./...` y `-tags=integration`.)_

## Fuente normativa

- **A1, A8**, **AP6**.
- **PRODUCT_DOMAIN.md** §6 (motor de IA) y §4.7 (flujos transaccionales).
