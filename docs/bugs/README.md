# Registro de Bugs Conocidos

> Registro de **bugs conocidos** (abiertos y cerrados) con el **formato de los
> anti-patrones de los manifiestos** (`MANIFEST_MONOREPO.md §4`,
> `MANIFEST_FRONTEND.md §4`): *Qué pasó / Lección / Regla*.
>
> Un bug abierto es deuda visible y rastreable; uno cerrado deja la lección para
> no repetirlo. Los gaps contrato↔dominio viven en
> [`../API_CONTRACT.md`](../API_CONTRACT.md).

---

## Índice

| ID | Título | Estado |
|---|---|---|
| [BUG-001](BUG-001-run-ons-no-subdivididos.md) | Run-ons largos no se subdividen en fragmentos | Abierto |
| [BUG-002](BUG-002-finish-reason-length-como-llm-unavailable.md) | `finish_reason=length` se reporta como `llm_unavailable` | Abierto |
| [BUG-003](BUG-003-analytics-progress-501.md) | `GET /analytics/progress` responde `501` | Abierto |

---

## Formato de una entrada

Cada bug se documenta en su propio fichero con:

- **Qué pasó** — síntoma, causa raíz y contexto.
- **Lección** — qué aprendimos (el riesgo o la trampa).
- **Regla** — qué se hace a partir de ahora, o el plan de cierre.
- **Estado** — abierto / cerrado (con referencia al commit que lo cerró).
- **Referencias** — DEVLOG, issues relacionados, tests.

---

**Última revisión**: 2026-09-21.
