# BUG-001 — Run-ons largos no se subdividen en fragmentos

- **Estado**: Abierto
- **Área**: Motor de IA (extracción) · `internal/api/adapters/llm`
- **Detectado**: 2026-09-20

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

- **Candidato de hardening**: subdividir los run-ons por **sub-cláusulas
  deterministas** (conjunciones, relativos) y **validar** que las sub-cadenas
  reconstruyan exactamente la frase original (invariante de cobertura), antes de
  invocar al modelo por sub-fragmento.
- Mientras no se cierre, queda registrado aquí y en el DEVLOG.

## Referencias

- `docs/DEVLOG.md` — 2026-09-20, "Fix: alineación español↔borrador".
- `apps/backend/internal/api/adapters/llm/prompt.go` (`SplitSentences`, `buildSentencePrompt`).
- `AGENTS.md` — gap conocido.
