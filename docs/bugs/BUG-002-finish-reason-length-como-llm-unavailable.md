# BUG-002 — `finish_reason=length` se reporta como `llm_unavailable`

- **Estado**: Abierto (mitigado parcialmente)
- **Área**: Motor de IA (adaptador) · `internal/api/adapters/llm`
- **Detectado**: 2026-09-20

## Qué pasó

Cuando el modelo alcanzaba el tope de tokens de salida, OpenAI devolvía
`finish_reason=length` con un JSON truncado. El adaptador lo trataba como un
JSON inválido y lo mapeaba al error de dominio **`llm_unavailable`**, un cajón de
sastre que **oculta la causa real** (respuesta truncada ≠ proveedor caído).

En textos de ~2 KB llegó a ocurrir ~50 % de las veces con el prompt anterior
(exhaustivo, una sola llamada).

## Lección

Un error de dominio debe ser **específico**: "el proveedor no está disponible" y
"la respuesta se truncó por el tope de salida" son incidentes distintos, con
respuestas operativas distintas (reintentar vs. reducir el trabajo por llamada).

## Mitigación ya aplicada

- Extracción **una vez por frase** (acota la salida por llamada).
- Prompt **acotado** (máximo de entradas por categoría, campos de una frase).
- Observación de `finish_reason` con una sonda temporal (ya retirada); medición:
  `finish=stop`, ~3k tokens (antes: `finish=length`, 16 384).
- Timeout cableado (`APP_LLM_TIMEOUT`, default `180s`).

## Regla / plan de cierre

- **Candidato de hardening**: detectar `finish_reason=length` en el adaptador y
  devolver un **error de dominio específico** (p. ej. `llm_output_truncated`) en
  vez de `llm_unavailable`, para que la API/UX pueda distinguirlo.
- Añadir un test del adaptador que cubra la respuesta truncada.

## Referencias

- `docs/DEVLOG.md` — 2026-09-20, "Fix: el análisis fallaba por desbordar el límite de salida del LLM".
- `apps/backend/internal/api/adapters/llm/openai_extractor.go`.
- Error de dominio: `internal/domain` (`LLMUnavailableError` / mapeo HTTP).
