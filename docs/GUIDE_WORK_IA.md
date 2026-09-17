# Guía de Trabajo con IA — Metodología de Colaboración Humano-IA (v1.0.0)

> **Audiencia**: ingenieros que trabajan en este repositorio asistidos por IA (opencole/agentes).
> **Objetivo**: mantener el enfoque del proyecto, evitar la pérdida de contexto entre sesiones y construir el sistema "sin dolores de cabeza".
>
> **Prerrequisitos**: `AGENTS.md`, `MANIFEST_MONOREPO.md`, `MANIFEST_FRONTEND.md`, `docs/PRODUCT_DOMAIN.md`.

---

## Tabla de contenidos

1. [Principio rector](#1-principio-rector)
2. [Ritual de sesión](#2-ritual-de-sesión)
3. [Protocolo de incremento atómico](#3-protocolo-de-incremento-atómico)
4. [Briefing: cómo dar la instrucción a la IA](#4-briefing-cómo-dar-la-instrucción-a-la-ia)
5. [Reglas de oro (anti-descarrilamiento)](#5-reglas-de-oro-anti-descarrilamiento)
6. [Verificación: quién corrige a quién](#6-verificación-quién-corrige-a-quién)
7. [Ejemplos de interacción](#7-ejemplos-de-interacción)

---

## 1. Principio rector

> **La IA ejecuta; el humano decide el alcance. La máquina corrige; el humano aprueba.**

El mayor riesgo al trabajar con IA es la **pérdida de contexto** (la sesión muere al cerrarse) y el **scope creep** (la IA "avanza" más de lo pedido). Esta guía existe para neutralizar ambos.

- El contexto vive en **archivos** (`AGENTS.md`, `docs/DEVLOG.md`, checklists), no en la memoria de la conversación.
- El alcance vive en los **checklists con gate** (`docs/checklist/`), no en el entusiasmo del momento.

---

## 2. Ritual de sesión

### 2.1 Apertura (lo hace la IA, obligatorio)

1. Leer `AGENTS.md`.
2. Leer el checklist de la fase actual (`docs/checklist/XX-*.md`).
3. Ejecutar `git status` y `git log --oneline -10`.
4. Leer `docs/DEVLOG.md` (últimas entradas y próximos pasos).

El humano, en paralelo, solo comunica el **objetivo de la sesión** (un item o un subconjunto).

### 2.2 Cierre (lo hace la IA, obligatorio)

1. Actualizar `docs/DEVLOG.md` (qué se hizo, decisiones, bloqueos, próximo paso).
2. Dejar `git status` limpio (todo commiteado o claramente identificado).
3. Reportar al humano: resumen + qué verificar + siguiente item sugerido.

> **Regla dura**: nunca terminar una sesión sin `DEVLOG` actualizado y commit. La próxima sesión depende de ese estado.

---

## 3. Protocolo de incremento atómico

Nunca pedir "hazme el backend completo". El bucle correcto:

```
1 trozo coherente (1–3 items del checklist)
  → verificar (test / lint / build)
  → commit (mensaje en español, atómico)
```

| Correcto | Incorrecto |
|---|---|
| "Fase 2, item `2.2.1` (`Practice` + value objects). Verifica con `go test ./internal/domain/practice`." | "Implementa todo el dominio de una vez." |

Por qué funciona: si algo falla, el `git diff` es pequeño y reversible; el commit es un punto de restauración.

---

## 4. Briefing: cómo dar la instrucción a la IA

Formato de 4 campos. Si falta alguno, la IA debe pedirlo.

```
Fase/Item:  Fase 2, item 2.2.1 (bounded context `practice`)
Objetivo:   Crear agregado `Practice` + value objects + invariantes.
Verificar:  go test -race ./internal/domain/practice
Parada:     Al terminar, para y reporta (no avances a 2.2.2 sin aprobar).
```

Campos:
1. **Fase/Item** — ancla el trabajo al checklist (evita divagar).
2. **Objetivo** — qué se espera, en una frase.
3. **Verificar** — el comando exacto que define "hecho".
4. **Parada** — límite explícito (la IA no debe encadenar trabajo no aprobado).

---

## 5. Reglas de oro (anti-descarrilamiento)

1. **Contract-first (A12)**: todo cambio de wire → primero `apps/contracts/openapi/api.yaml` → `pnpm generate` → código. Nunca al revés.
2. **TDD como especificación**: pedir "haz pasar este test" o "escribe el test fallante primero". Un test es una especificación objetiva que **sobrevive** a la pérdida de contexto.
3. **Orden de dependencias (A2)**: construir `domain → ports → adapters → services → handlers`. En ese orden es imposible acoplar mal las capas.
4. **Una fase a la vez**: no se cruza el `Gate de salida` hasta estar en verde.
5. **No inventar comandos**: si un comando aún no existe, la IA **pregunta**, no improvisa (ver `AGENTS.md`).
6. **Scope**: el MVP es single-user. Recordar los non-goals de `PRODUCT_DOMAIN.md §3.2` cuando la IA proponga "y también podríamos…".

---

## 6. Verificación: quién corrige a quién

| Guardrail | Quién lo aplica | Cuándo |
|---|---|---|
| Reglas de lint (`domain_purity`, `forbidden_imports`, `contract_drift`) | La máquina (CI/lint) | Desde Fase 1–2 |
| Gates de cobertura (dominio 100%, services ≥90%) | La máquina (CI) | Desde Fase 2 |
| `pnpm generate` idempotente | La máquina | Cada cambio de contrato |
| Revisiones de alcance y decisiones de producto | El humano | Antes de cada commit/merge |
| `DEVLOG` + commit limpio | La IA (obligado por ritual) | Cierre de sesión |

> La IA no es la fuente de verdad: la fuente de verdad son los **manifiestos**, el **contrato** y los **tests**. La IA ejecuta dentro de esas fronteras; la máquina las hace cumplir.

---

## 7. Ejemplos de interacción

### 7.1 TDD en el dominio

> **Humano**: "Fase 2, item `2.2.3` (invariantes de `Practice`). Escribe primero el test fallante, luego la implementación. Verifica con `go test -race ./internal/domain/practice`. Para y reporta."

### 7.2 Cambio de contrato

> **Humano**: "Necesito un campo `feedback` en `Fragment`. Fase 1, regla A12: primero `api.yaml`, luego `pnpm generate`, luego actualiza el código que consume el tipo. Verifica con `pnpm generate` y `pnpm build`."

### 7.3 Sesión de revisión

> **Humano**: "Revisa el Gate de salida de Fase 3 (`03-puertos-adaptadores.md`). Dime qué items faltan y qué comando verifica cada uno. No escribas código todavía."

---

**Versión**: 1.0.0
**Mantenedor**: equipo de ingeniería.
**Última revisión**: 2026-09-15.
**Próxima revisión**: tras la primera sesión real de implementación (Fase 1), para ajustar el ritual según lo observado.
