# Changelog — `apps/contracts`

Formato basado en [Keep a Changelog](https://keepachangelog.com/es/1.1.0/).
El contrato OpenAPI es la SSOT del wire format (A12); todo cambio breaking sube
la versión major y se anota aquí (AP-MR6).

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
