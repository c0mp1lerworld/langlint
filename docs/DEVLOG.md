# DEVLOG — Bitácora de Desarrollo

> Bitácora **append-only** del proyecto. Registrar aquí decisiones, avances, bloqueos y próximos pasos. Cada sesión la actualiza al cerrar. Las entradas más recientes van arriba.

---

## 2026-09-17 — Fase 1 (inicio): base del monorepo (items 1.1.1–1.1.6)

**Estado**: Fase 1 en curso. `pnpm install --frozen-lockfile && pnpm build` en verde.

**Hecho**:
- `package.json` raíz: `"private": true`, `"packageManager": "pnpm@9.15.4"`, scripts globales (`build`, `lint`, `test`, `test-integration`, `generate`, `typecheck`, `dev:backend`, `dev:frontend`).
- `pnpm-workspace.yaml` con `packages: ['apps/*']`.
- `turbo.json` con tasks `build` (dependsOn `^build`), `lint`, `test`, `test-integration`, `typecheck`, `dev`, `generate` (según manifiesto §2.2).
- `.nvmrc` (Node 22), `.tool-versions` (golang 1.26.2, nodejs 22.13.0, pnpm 9.15.4), `.editorconfig`, `.gitignore`.
- `devDependencies`: `turbo@^2.10.13`, `typescript@^7.0.2` (versiones concretas para reproducibilidad con `--frozen-lockfile`, en lugar de `latest`).
- Placeholder `apps/contracts/package.json` (`@langlint/contracts`, `build` stub) para que `turbo run build` tenga objetivo.
- `git init` + commit inicial (primer commit del repositorio).

**Decisiones**:
- Se usa un placeholder mínimo de `apps/contracts` con `build` stub para que el gate `pnpm build` quede verde sin adelantar `1.2.1`/`1.3`/`1.4` (los dirs `openapi/` y `scripts/generate.sh` siguen pendientes).
- `typescript@7.0.2` (native) y `turbo@2.10.13` resueltos por pnpm como `latest` (manifiesto decía `latest`); se pinnean con caret.

**Bloqueos**: ninguno.

**Nota**: `turbo` emite `WARNING no output files found for task @langlint/contracts#build` (el `build` stub no produce artefactos y `turbo.json` declara `outputs`). Inofensivo; desaparecerá cuando contracts genere outputs reales (1.4).

**Próximo paso**: items `1.2.x` (árbol canónico de directorios) y `1.3.x` (contrato OpenAPI `api.yaml`).

---

## 2026-09-15 — Fase 0: Preparación y fundamentos documentales

**Estado**: antes de Fase 1. Sin código.

**Hecho**:
- `MANIFEST_MONOREPO.md` (v2.0.0) y `MANIFEST_FRONTEND.md` (v1.0.0) como base normativa.
- `docs/PRODUCT_DOMAIN.md` (v1.0.0): visión, modelo de dominio (4 bounded contexts + futuro `tutor`), contrato, motor IA, analítica, privacidad, matriz de trazabilidad, DoD, roadmap.
- `docs/checklist/` con 8 fases (7 MVP + 1 post-MVP), cada una con subfases `x.x`, procedimientos `x.x.x` y `Gate de salida`.
- `AGENTS.md` (memoria de trabajo para agentes de IA).
- `docs/GUIDE_WORK_IA.md` (metodología de colaboración humano-IA).

**Decisiones de diseño bloqueadas**:
- Bounded contexts propios: `identity`, `practice`, `analysis`, `analytics` (se descartan `organization`/`billing` del template B2B).
- **Single-user MVP**: identidad portable mínima (A9), sin login real.
- Tutor adaptativo (`tutor/`) como **fase 8 post-MVP** (§12.1 del PRODUCT_DOMAIN).
- Proveedor IA: OpenAI (`openai-go`) detrás del puerto `LLMExtractor`.

**Bloqueos**: ninguno.

**Próximo paso**: Fase 1 — Fundaciones (`docs/checklist/01-fundaciones.md`): inicializar monorepo + contrato OpenAPI fundacional.
