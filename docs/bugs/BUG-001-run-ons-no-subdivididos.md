# BUG-001 — Run-ons largos no se subdividen en fragmentos

- **Estado**: Cerrado
- **Área**: Motor de IA (extracción) · `internal/api/adapters/llm`
- **Detectado**: 2026-09-20
- **Cerrado**: 2026-09-21

## Qué pasó

Para garantizar cobertura y orden del borrador, la extracción se hace **una vez
por frase** del borrador, con segmentación determinista en `. ! ?`
(`SplitSentences`). El problema aparece con **run-ons**: una frase larga sin
puntuación interna (`. ! ?`) se trata como **un único fragmento grande**.

Consecuencias:

- La corrección y las explicaciones son menos granulares (todo el run-on cae en
  una sola tarjeta del diff).
- El prompt por frase puede acumular más errores de los que el modelo explica
  bien, y la respuesta es más propensa al truncado por longitud.

El fix de alineación del 2026-09-20 (extracción por frase) **resolvió** el
desfase ES↔borrador, pero dejó este límite de granularidad documentado.

## Lección

Segmentar por frase es **fuerte** para la cobertura y el orden (el `user_draft`
lo fija el backend y la alineación es por construcción), pero traslada la
granularidad a la puntuación que escriba el usuario. Un usuario que no puntúa
obtiene un análisis peor precisamente cuando más lo necesita.

## Regla / plan de cierre

- **Implementado**: `splitSegments`/`splitRunOns` en
  `internal/api/adapters/llm/prompt.go` subdividen los run-ons (≥25 palabras) por
  **sub-cláusulas deterministas** (coma seguida de conjunción/relativo, o punto y
  coma) y validan el **invariante de cobertura** (`reconstructs`: unir las piezas
  con un espacio reproduce la frase original); si no se cumple, se devuelve la
  frase entera. El adaptador llama al modelo por sub-cláusula.
- Tests: `prompt_test.go` (`splitRunOns`/`splitSegments`) y
  `openai_extractor_test.go` (un run-on ⇒ una llamada por sub-cláusula).

## Referencias

- `docs/DEVLOG.md` — 2026-09-20, "Fix: alineación español↔borrador".
- `apps/backend/internal/api/adapters/llm/prompt.go` (`SplitSentences`, `buildSentencePrompt`).
- `AGENTS.md` — gap conocido.
