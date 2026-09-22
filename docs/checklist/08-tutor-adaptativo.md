# Checklist — Fase 8: Tutor Adaptativo (Spaced Repetition)

> **Estado**: POST-MVP
> **Roadmap**: `docs/PRODUCT_DOMAIN.md §12.1`
> **Predecesora**: Fase 7 (Hardening) · **Sucesora**: —

> ⚠️ **Esta fase no se implementa en el MVP.** Se mantiene en el radar a futuro. Su diseño se anticipa para que su incorporación sea **aditiva** (sin romper los 4 bounded contexts existentes).

---

## 8.1 Bounded context `tutor/`

- [x] `8.1.1` Crear `internal/domain/tutor/` como **quinto** bounded context, aislado por A3 (sin imports cruzados).
- [x] `8.1.2` Definir agregado `StudySession` (teoría resumida, trampas comunes, ejercicios interactivos).
- [x] `8.1.3` Definir value object `WeaknessProfile` (agrupación de `ErrorPattern` históricos).
- [x] `8.1.4` Verificar que `tutor/` consume agregados de `analytics/` (no datos crudos).

## 8.2 Detección de debilidades

- [ ] `8.2.1` `analytics/` emite evento `WeaknessDetected` (payload: `UserID`, `[]ErrorPattern`, `Window`) vía outbox.
- [ ] `8.2.2` `tutor/` se suscribe a `WeaknessDetected` como `EventHandler` (mismo patrón del relay, sin tocar el dispatcher existente).
- [ ] `8.2.3` Regla de detección: umbral de frecuencia (p. ej. "5 prácticas con el mismo `ErrorPattern.code`").

## 8.3 Generación de sesión de estudio

- [ ] `8.3.1` Generar la sesión con IA (Structured Outputs, reusando el patrón del `LLMExtractor`).
- [ ] `8.3.2` Contenido de la sesión: teoría resumida + 3 trampas comunes del usuario + 5 ejercicios interactivos personalizados.
- [ ] `8.3.3` Anonimizar antes de enviar al LLM (A8) y validar el output contra su schema.

## 8.4 Integración sin rupturas

- [ ] `8.4.1` Los 4 bounded contexts del MVP (`identity`, `practice`, `analysis`, `analytics`) **no cambian** (cero rupturas).
- [ ] `8.4.2` Nuevo endpoint `POST /v1/study-sessions` (o equivalente) declarado primero en `api.yaml` (A12).
- [ ] `8.4.3` Frontend: nueva feature `study-session` consumiendo el cliente generado, sin lógica de negocio duplicada (F11).

---

## ✅ Gate de salida

- [ ] `tutor/` funciona sin modificar `identity`, `practice`, `analysis`, `analytics`.
- [ ] Evento `WeaknessDetected` entregado end-to-end vía outbox (test Tier 3).
- [ ] Sesión generada con Structured Outputs validada; PII anonimizado (A8).
- [ ] Cobertura del nuevo dominio 100% y services ≥90% (A10).

## Fuente normativa

- **A3, A4, A8, A10, A12**, **F11**.
- **PRODUCT_DOMAIN.md** §12.1 (Roadmap post-MVP) y §4.6 (taxonomía de `ErrorPattern`).
