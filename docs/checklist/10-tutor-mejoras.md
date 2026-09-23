# Checklist — Fase 10: Tutor — mejoras post-MVP (backlog)

> **Estado**: BACKLOG — post-MVP, para después (no implementado).
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12.1` (tutor adaptativo).
> **Predecesora**: Fase 8 (Tutor Adaptativo, MVP) y Fase 9 (`9.7` analytics del quiz) · **Sucesora**: —.

> ⚠️ **Ninguno de estos items está implementado.** Este checklist evidencia el trabajo pendiente del tutor: la Fase 8 entregó el ciclo mínimo (detección de debilidades + generación de sesión + endpoint `POST /v1/study-sessions`), pero el tutor aún es un generador "bajo demanda" sin historial, sin elección de tema y sin repetición espaciada real. Son mejoras aditivas sobre lo ya construido (A3, A4), no una reescritura.

---

## 10.1 Historial de sesiones

- [ ] `10.1.1` `GET /v1/study-sessions` (listar las sesiones del usuario, paginado `{items, total}`).
- [ ] `10.1.2` `GET /v1/study-sessions/{sessionId}` (detalle de una sesión).
- [ ] `10.1.3` `StudySessionRepository` gana `ListByUser`/`GetByID` (hoy solo expone `Save`); la tabla `study_sessions` ya existe (migración `000005`).

## 10.2 Elección del tema intensivo

- [ ] `10.2.1` `POST /v1/study-sessions` acepta un `code` (`ErrorPatternCode`) opcional para reforzar un tema concreto; si se omite, mantiene el comportamiento actual (la debilidad más frecuente).
- [ ] `10.2.2` El `StudySessionService` filtra el `WeaknessProfile` por el código elegido antes de generar.
- [ ] `10.2.3` El frontend expone un selector del tema (derivado de `GET /analytics/error-patterns`, F11: sin lógica de negocio duplicada).

## 10.3 Ciclo de vida completo de la sesión

- [ ] `10.3.1` Exponer las transiciones del dominio (`generated → active → completed`): `PATCH /v1/study-sessions/{sessionId}/start` y `.../complete`.
- [ ] `10.3.2` Validar transiciones inválidas → `409 invalid_state` (el dominio ya lo hace con `Start()`/`Complete()`).

## 10.4 Repetición espaciada (spaced repetition)

- [ ] `10.4.1` Programar la próxima revisión: campo `next_review_at` (o intervalo) derivado de los aciertos de la sesión.
- [ ] `10.4.2` Recordar al usuario "es hora de repasar X" (señal de repaso, coherente con `WeaknessDetected`).

## 10.5 Integración quiz → repaso

- [ ] `10.5.1` Usar `quiz_attempts` (9.7) para ajustar dificultad/frecuencia de los ejercicios de la sesión, cerrando el bucle tutor↔práctica activa.
- [ ] `10.5.2` Decidir la fuente de verdad de esa señal (no debe entrar en `refresh-aggregates`, que reconstruye solo desde `analyses`; mismo criterio que el diferido de 9.7).

## 10.6 Frontend

- [ ] `10.6.1` Página de historial de sesiones (lista + detalle).
- [ ] `10.6.2` Vista de sesión activa (start/complete) + selector de tema (10.2).

---

## ✅ Gate de salida (para cuando se aborde)

- [ ] Historial y elección de tema operativos end-to-end (A12 + F11).
- [ ] `tutor/` sigue sin importar `identity`/`practice`/`analysis`/`analytics` (A3).
- [ ] Cobertura del dominio nuevo 100 % y services ≥90 % (A10).
- [ ] `go build/vet/test` (+ `-tags=integration`), `pnpm generate` idempotente, `pnpm test`/`typecheck`/`build`/`test:e2e` en verde.

## Fuente normativa

- **A3, A4, A8, A10, A12**, **F11**.
- **PRODUCT_DOMAIN.md** §12.1 (Roadmap post-MVP) y §4.6 (taxonomía de `ErrorPattern`).
