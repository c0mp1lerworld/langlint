# Changelog — `apps/contracts`

Formato basado en [Keep a Changelog](https://keepachangelog.com/es/1.1.0/).
El contrato OpenAPI es la SSOT del wire format (A12); todo cambio breaking sube
la versión major y se anota aquí (AP-MR6).

## [3.2.0] — 2026-09-21

### Añadido

- **`ErrorResponse.code`**: nuevo código `llm_output_truncated`, distinto de
  `llm_unavailable`, para cuando el proveedor alcanza el tope de tokens de salida
  y devuelve una respuesta truncada (BUG-002). No es breaking: es un valor nuevo
  de un enum que el cliente ya tolera como desconocido.

## [3.1.0] — 2026-09-21

### Cambiado

- **`GET /v1/analytics/progress`**: deja de declarar `501 NotImplemented`; la
  operación responde `200 ProgressSeries` con la serie temporal real
  (PRODUCT_DOMAIN §7.1). Se retira el componente de respuesta `NotImplemented`,
  ya sin uso (no es breaking: el `200` ya estaba declarado).

## [3.0.0] — 2026-09-20

### BREAKING

- **`Fragment`**: `target_verb_review`, `lexical_clarification` y
  `grammar_explanation` dejan de ser un objeto único y pasan a ser listas
  (`target_verb_reviews`, `lexical_clarifications`, `grammar_explanations`).
  Un mismo fragmento puede acumular varios verbos objetivo, varias
  aclaraciones léxicas y varias reglas gramaticales; cada lista puede ir vacía
  cuando la categoría no aplica (PRODUCT_DOMAIN §1.2, "Beyond Correction"). Los
  clientes que leían un solo elemento deben regenerar y adaptar el render.

## [2.1.0] — 2026-09-19

### Añadido

- Operaciones `POST /v1/practices/{practiceId}/quiz` (genera una pregunta de
  práctica activa anclada al error) y
  `POST /v1/practices/{practiceId}/quiz/answer` (evalúa la respuesta y ofrece
  una pregunta de seguimiento). Esquemas `QuizQuestion`, `QuizQuestionRequest`,
  `QuizAnswerRequest` y `QuizEvaluation` (PRODUCT_DOMAIN §1.2, §12.1).

## [2.0.0] — 2026-09-19

### BREAKING

- **`Fragment`**: los campos `target_verb_review`, `lexical_clarification` y
  `grammar_explanation` dejan de ser `string` y pasan a ser objetos
  estructurados (`TargetVerbReview`, `LexicalClarification`,
  `GrammarExplanation`) para forzar feedback profundo y renderizable como
  tarjetas (PRODUCT_DOMAIN §1.2, "Beyond Correction"). Los clientes que leían
  esos campos como texto deben regenerar y adaptar el render.

### Añadido

- Esquemas `TargetVerbReview`, `LexicalClarification` y `GrammarExplanation`.

## [1.0.0]

- Versión inicial del contrato (endpoints de prácticas, análisis, analíticas y
  A9 `/me/*`).
