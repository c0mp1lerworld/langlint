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

- [ ] `4.2.1` Configurar Structured Outputs con el JSON Schema de `Fragment[]` (PRODUCT_DOMAIN §5.2).
- [ ] `4.2.2` Validar el JSON del LLM contra el schema **antes** de persistir; output inválido → `LLMUnavailableError`.
- [ ] `4.2.3` La llamada al LLM se ejecuta **fuera** de la transacción (fuera del `UnitOfWork`).
- [ ] `4.2.4` Persistir `Analysis` + outbox `AnalysisCompleted` atómicamente tras recibir el resultado (PRODUCT_DOMAIN §4.7 Flujo 2).

## 4.3 Anonimización, contexto y timeouts

- [ ] `4.3.1` Implementar `anonymizer.go` que elimina emails, teléfonos y nombres propios antes de enviar al LLM (A8).
- [ ] `4.3.2` Toda goroutine del extractor recibe `context.Context` y chequea `ctx.Done()` (AP6, regla `goroutine_context`).
- [ ] `4.3.3` Configurar timeout de la llamada; ante fallo emitir `AnalysisFailed` (sin filtrar error crudo).
- [ ] `4.3.4` El `PIIHandler` descarta logs con contenido libre (>100 chars) y patrones PII (A8).

---

## ✅ Gate de salida

- [ ] Análisis fragmentado devuelto y validado contra el schema (sin `Fragment` malformado persistido).
- [ ] Test con output inválido del LLM → `LLMUnavailableError` (no se persiste basura).
- [ ] Cambio de proveedor no requiere tocar `internal/domain/` (A1 verificado).
- [ ] `go test -race ./...` en verde en el motor de IA.

## Fuente normativa

- **A1, A8**, **AP6**.
- **PRODUCT_DOMAIN.md** §6 (motor de IA) y §4.7 (flujos transaccionales).
