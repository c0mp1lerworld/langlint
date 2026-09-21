# Checklist — Fase 9: Feedback profundo y práctica activa

> **Estado**: mejora de calidad del MVP (post-Fase 6, previa a Hardening).
> **Roadmap**: mejora de `PRODUCT_DOMAIN §1.2` ("Beyond Correction") y §12.1 (tutor adaptativo).
> **Predecesora**: Fase 6 (Provisioner y privacidad) · **Sucesora propuesta**: Fase 7 (Hardening).

---

## 9.1–9.4 Feedback profundo y estructurado

- [x] `9.1` Endurecer el prompt con una rúbrica de explicación (regla nombrada, por qué, construcción, contra-ejemplo, excepción, contraste ES→EN, alternativas). _(`systemPrompt` en `internal/api/adapters/llm/prompt.go`; verificado con `go run ./cmd/llmcheck -show-prompt` y una llamada real con el ejemplo "bet on".)_
- [x] `9.2` A12: reemplazar los 3 `string` de `Fragment` por esquemas estructurados (`TargetVerbReview`, `LexicalClarification`, `GrammarExplanation`) en `api.yaml` + bump de `apps/contracts` a 2.0.0 + `CHANGELOG.md`. _(BREAKING; `pnpm generate` idempotente.)_
- [x] `9.3` Dominio `analysis` (structs anidados) + `schema.go` (JSON Schema anidado) + `validate.go` (validar sub-campos) + conversión dominio→wire en `handlers/server.go`. _(Variación de `alternatives` vacía permitida; el resto de campos de texto deben tener contenido.)_
- [x] `9.4` Frontend: tarjetas etiquetadas en `fragment-diff.tsx` + fixtures/tests Vitest + e2e Playwright. _(Accesibilidad: `dl`/`dt`/`dd`, `h3` bajo las columnas `h2`; `axe` sin violaciones.)_

## 9.5 Límites de práctica (punto 1)

- [x] `9.5` Acotar el "engorro" de muchos verbos / textos largos: máximo de `TargetRules` y longitud máxima de `source_text`/`draft_text` (error de dominio A5) + guía en `practice-form.tsx`. _(Máx. 5 reglas y 2000 runes, con `ValidationError`; espejo en Zod y contador de botón "Añadir regla".)_

## 9.6–9.7 Preguntas generadas por IA (práctica activa)

- [ ] `9.6` Puerto `TutorQuestioner` (tipos `open` y `fill`, la IA elige) + endpoints `GET .../question` y `POST .../answer` con evaluación y seguimiento socrático acotado. _(`mcq` descartado: entrena reconocimiento, no producción.)_
- [ ] `9.7` Integración con analytics: registrar aciertos/fallos del quiz para alimentar el repaso (tutor adaptativo futuro).

---

## ✅ Gate de salida

- [ ] `go build/vet/test -race` (+ `-tags=integration`) en verde.
- [ ] `pnpm generate` idempotente y `pnpm lint` (contrato incluido) en verde.
- [ ] `pnpm test-integration --filter=backend`, `pnpm test`, `pnpm typecheck --filter=frontend` y `pnpm build` en verde.
- [ ] Verificación manual: lección estructurada + pregunta/evaluación con el ejemplo de referencia ("bet on").

## Fuente normativa

- **A5, A8, A12**; **PRODUCT_DOMAIN §1.2, §5.2, §12.1**.
