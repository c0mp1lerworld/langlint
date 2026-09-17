# Manifiesto del Monorepo (v2.0.0)

> **Guía de arquitectura para proyectos monorepo polyglot.**
> Este documento establece las reglas, estándares y patrones que rigen la construcción de un sistema como un monorepo que aloja el ecosistema backend (Go) y, eventualmente, el frontend (TypeScript/Next.js). Es un manifiesto **general y autocontenido**: define *cómo* se construye el monorepo y el backend, con independencia del dominio de negocio concreto. No requiere lectura previa de otros manifiestos para su aplicación.
>
> **Audiencia**: arquitectos, leads técnicos, nuevos devs que se incorporan al equipo.
>
> **Prerrequisitos de lectura**: `docs/ARCHITECTURE.md §1-§3`, `docs/PRODUCT_DOMAIN.md §1-§3`.

---

## Tabla de contenidos

1. [Axiomas de Desarrollo](#1-axiomas-de-desarrollo)
2. [Estándar Go y Toolchain del Monorepo](#2-estándar-go-y-toolchain-del-monorepo)
3. [Esqueleto Arquitectónico](#3-esqueleto-arquitectónico)
4. [Anti-patrones Históricos](#4-anti-patrones-históricos)
5. [Patrones Arquitectónicos Transversales](#5-patrones-arquitectónicos-transversales)
6. [Estrategia de Monorepo y Despliegue](#6-estrategia-de-monorepo-y-despliegue)
7. [Glosario del Monorepo](#7-glosario-del-monorepo)
8. [Documentos autoritativos](#8-documentos-autoritativos)

---

## 1. Axiomas de Desarrollo

Doce principios inquebrantables. Cualquier PR que los viole debe ser rechazado en revisión. No son opiniones: son el precio de mantener el código claro, preciso y escalable.

### A1. La Pureza del Dominio es Sagrada

> El paquete `apps/backend/internal/domain/` y sus sub-paquetes (`identity/`, `organization/`, `billing/`, `audit/`) **solo importan la stdlib de Go**. Nada más.

- **Por qué existe**: el dominio es la capa más estable del sistema. Cambiar de Postgres a MySQL, de chi a gin, de un proveedor LLM a otro, o de UUID externo a otro algoritmo **no debe requerir un solo cambio en `apps/backend/internal/domain/`**. Si necesitamos una operación de infraestructura, primero se define como puerto en `apps/backend/internal/api/ports/` (o `provisioner/ports/`) y se implementa en `apps/backend/internal/<entrypoint>/adapters/`.
- **Consecuencia de violarlo**: cada dependencia externa en el dominio se convierte en una deuda que se paga multiplicada al migrar. Una violación es un bug latente.
- **Validación**: regla custom de `golangci-lint` llamada `domain_purity` (filtra cualquier import fuera de stdlib) más un job `purity` en CI que ejecuta `grep -rh --include='*.go' --exclude='*_test.go' '^[[:space:]]*"[^"]*"' apps/backend/internal/domain/` como smoke check y falla si aparece un import fuera de stdlib.
- **Única excepción documentada**: UUID v7 se implementa internamente con `crypto/rand`, `time` y `fmt`. La API expuesta es `domain.NewID()`, `domain.MustNewID()`, `(domain.ID).IsValid()`, `(domain.ID).Version()`.

### A2. La Dirección de las Dependencias es Hacia Adentro, Siempre

> Las dependencias fluyen en una sola dirección: `cmd → adapters → services → ports → domain`. Cualquier flecha hacia afuera es un bug estructural.

```
cmd/  →  adapters/  →  services/  →  ports/  →  domain/
                                              (núcleo puro)
```

- **Por qué existe**: garantiza que el dominio no se acopla a la infraestructura y que los servicios solo conocen interfaces (`ports/`), no implementaciones (`adapters/`).
- **Consecuencia de violarlo**: una flecha invertida crea un ciclo de imports que rompe `go build`, o peor, una dependencia lógica que solo se manifiesta en runtime (testing doloroso, mocking imposible).
- **Reglas finas**:
  - `apps/backend/internal/domain/` no importa a nadie (raíz pura).
  - `apps/backend/internal/<entrypoint>/services/` importa `ports/` (interfaces) y `domain/`.
  - `apps/backend/internal/<entrypoint>/adapters/` importa `ports/` (las implementa) y `services/` (invoca servicios).
  - `apps/backend/cmd/<entrypoint>/` importa `adapters/`, `services/`, `ports/`, `shared/`, `domain/` y su `di/`. Nada más.

### A3. Los Bounded Contexts Están Aislados por Construcción

> `apps/backend/internal/domain/identity/`, `apps/backend/internal/domain/organization/`, `apps/backend/internal/domain/billing/` y `apps/backend/internal/domain/audit/` **no se importan entre sí**. Se comunican únicamente vía puertos en `apps/backend/internal/<entrypoint>/ports/` o eventos append-only.

- **Por qué existe**: cambiar el modelo de facturación no debe tocar autenticación; cambiar el modelo de identidad no debe tocar billing; testear cada contexto en aislamiento puro sin mocks cruzados.
- **Consecuencia de violarlo**: acoplamiento cruzado que convierte un cambio localizado en una migración de días.
- **Validación**: regla `forbidden_imports` de `golangci-lint`. Los bounded contexts solo comparten tipos primitivos del paquete raíz `apps/backend/internal/domain/` (`ID`, `OpaqueData`, errores sentinela).

### A4. Append-Only donde Vive la Verdad

> Las tablas `billing.transactions` y `audit.events` **no exponen `Update` ni `Delete`** en sus repositorios. La historia es reconstruible; la auditoría es inmutable. Son ledgers append-only de negocio: cada INSERT es un hecho de negocio.

- **Por qué existe**: el ledger financiero y la auditoría de accesos son evidencia legal y contractual. Permitir UPDATE/DELETE abre la puerta a fraude, manipulación y pérdida de trazabilidad. Los flujos de facturación y auditoría requieren reconstruir saldos y eventos con precisión exacta.
- **Consecuencia de violarlo**: un cliente puede alegar "yo tenía más saldo" sin que el sistema tenga cómo rebatirlo. Un usuario puede alegar "nadie accedió a mis datos" sin que el sistema tenga el log para defenderse.
- **Implementación**: las interfaces `TransactionRepository` y `AuditEventRepository` solo exponen `Insert` y `Get*`. Si una query requiere "corregir" algo, se inserta un nuevo registro que corrige (compensating entry).
- **Eventos**: la publicación de `BalanceChanged`/`BalanceReplenished` se hace vía el **outbox genérico** `outbox_events` (ver §5.1), dentro de la misma transacción que el INSERT del ledger. Un relay publica post-commit sin acoplar bounded contexts.

### A5. Los Errores son Ciudadanos del Dominio

> Un service **nunca** retorna `error` genérico. Siempre retorna un tipo sentinela o struct del paquete `domain` (`*domain.ValidationError`, `*domain.NotFoundError`, `*domain.InsufficientBalanceError`, `*domain.ConflictError`, etc.).

- **Por qué existe**: el handler HTTP necesita mapear errores de dominio a status codes (404, 400, 422, 500). Si el service retorna `error` opaco, el adapter tiene que hacer `errors.As` adivinando qué tipo es, lo cual es frágil y propenso a fugas de detalles de infraestructura al cliente.
- **Consecuencia de violarlo**: respuestas HTTP incorrectas, detalles de SQL filtrados al cliente (`pq: duplicate key value violates unique constraint "..."`), 500 donde debería ser 422.
- **Contrato**: los errores se definen en `apps/backend/internal/domain/errors.go`. Los adaptadores convierten errores de infraestructura (`pgx.ErrNoRows`, `stripe.Error`) a errores de dominio. La capa HTTP traduce errores de dominio a status codes vía `apps/backend/internal/api/errors.go`.

### A6. Multi-tenancy por Fila, Nunca por Schema

> Cada entidad de negocio lleva `tenant_id` (excepto la entidad portable de identidad global, que es portable por diseño; ver AP1). El aislamiento se enforce por **tres capas** independientes: middleware HTTP, CHECK constraint en DB, y test que rechaza `tenantID.IsZero()`.

- **Por qué existe**: un tenant nunca debe ver datos de otro tenant. El aislamiento por schema separado no escala (migraciones por tenant, blast radius operativo). El aislamiento por fila con RLS permite multi-tenancy operativo y económico con blast radius de query.
- **Consecuencia de violarlo**: brecha de datos entre tenants. Una sola línea sin RLS es un CVE de nivel crítico.
- **Implementación**:
  - **Capa 1 — Middleware**: extrae `tenant_id` del API key o JWT y lo inyecta en `context.Context` (en `apps/backend/internal/shared/httpx/`).
  - **Capa 2 — DB**: políticas RLS en Postgres (`tenant_isolation`) que comparan `current_setting('app.tenant_id')` con la columna `tenant_id`. Helpers en `apps/backend/internal/shared/db/`.
  - **Capa 3 — Test**: cada repositorio tiene un test que llama `Save(ctx, entity)` con `tenantID.IsZero()` y espera error de dominio.

### A7. Los IDs son Responsabilidad del Dominio

> Los UUIDs se generan internamente en `apps/backend/internal/domain/identifiers.go` usando **solo** `crypto/rand`, `time` y `fmt`. **Prohibido** importar `github.com/google/uuid` u otra librería externa.

- **Por qué existe**: el dominio es dueño de su generación de IDs. Debe seguir funcionando aunque cambie la implementación interna. RFC 9562 (UUID v7) ofrece ordenamiento temporal y no-enumeración, ambos críticos para datos sensibles.
- **Consecuencia de violarlo**: el dominio arrastra una dependencia externa que puede deprecar, romper o tener CVEs; los IDs pierden trazabilidad temporal.
- **API canónica**: `domain.NewID()`, `domain.MustNewID()`, `(domain.ID).IsValid()`, `(domain.ID).Version()`.
- **Prohibición adicional**: IDs predecibles o secuenciales (`user-1`, `org-2`) son vulnerabilidades de enumeración. Los IDs **deben** ser UUID v7 (random + timestamp).

### A8. Los Datos Personales son Inviolables

> Los datos personales del usuario (PII) **nunca** se loguean completos. Se anonimizan antes de enviarlos a servicios externos (LLM). Se pseudonimizan antes de analytics. El `PIIHandler` de `apps/backend/internal/shared/logger/` es guard runtime.

- **Por qué existe**: una filtración de PII no es un bug técnico: es un incidente regulatorio, ético y legal. La auditoría debe ser capaz de demostrar que **nunca** se logueó el contenido crudo de los datos personales del usuario.
- **Consecuencia de violarlo**: incumplimiento de GDPR-equivalente, daño reputacional irreparable, responsabilidad legal del equipo y la empresa.
- **Capas de defensa**:
  - **Anonimización previa al LLM**: `apps/backend/internal/api/services/anonymizer.go` elimina emails, teléfonos y nombres propios antes de enviar a servicios externos.
  - **Pseudonimización en analytics**: `apps/backend/internal/api/services/pseudonymizer.go` aplica HMAC-SHA256 con `APP_PSEUDONYM_SECRET` sobre identificadores antes de materializar datos de analytics.
  - **Retención limitada**: datos crudos separados en tablas dedicadas (RLS restrictiva, purga programada vía `apps/backend/cmd/provisioner`).
  - **Guard runtime**: `PIIHandler` descarta logs cuyos atributos contengan patrones PII (texto libre >100 chars, emails, API keys, JWTs, nombres propios).
  - **Auditoría estática**: `ops/scripts/pii_audit.sh` ejecuta `grep` defensivo en logs de CI.

### A9. El Usuario es Dueño de sus Datos

> La identidad global del usuario es portable por diseño. El usuario es dueño de sus datos. Los tenants son inquilinos, no propietarios. Los grants de acceso son explícitos, revocables en ≤24h, y cada visualización queda registrada como evento append-only.

- **Por qué existe**: la portabilidad de los datos del usuario es un **activo estructural** y un requisito regulatorio (GDPR). Sin ella, los datos quedan "secuestrados" por el tenant; con ella, el usuario confía y el sistema puede operar como plataforma con efecto de red.
- **Consecuencia de violarlo**: pérdida del activo estratégico (la base de usuarios), exposición a fuga de datos si un tenant "secuestra" perfiles, y no conformidad regulatoria.
- **Endpoints sagrados**:
  - `GET /v1/me/data/export` — portabilidad (GDPR Art. 15+20).
  - `DELETE /v1/me/data` — derecho al olvido con gracia de 30 días (GDPR Art. 17).
  - `GET /v1/me/access-log` — auditoría del usuario.

### A10. Los Tests son un Gate, No un Decorativo

> Cobertura mínima por capa: **dominio 100%** (regla dura, gate en CI), **services ≥90%**, **adapters ≥70%**. Sin `t.Skip()` en CI. Sin optimistic UI sin retry manual. Concurrencia cubierta con `go test -race`. **Pirámide de tres tiers** (ver §5.2): unit puro, unit con fakes, integración con testcontainers.

- **Por qué existe**: en un sistema que produce resultados de negocio críticos, un bug no es un crash: es un resultado mal calculado que el cliente usa para decidir. La única defensa es tests exhaustivos en el dominio y muy buenos en los services.
- **Consecuencia de violarlo**: regresiones que se manifiestan en producción, no en CI. Post-mortems que empiezan con "los tests no lo cubrían".
- **Convenciones**:
  - Nombres: `TestXxx_MethodName_Condition_ExpectedResult` (ej: `TestWalletService_Debit_Concurrent10_DoesNotOverspend`).
  - `testify/require` para assertions fatales, `testify/assert` para no fatales.
  - Mocks generados con `mockgen` desde interfaces en `apps/backend/internal/<entrypoint>/ports/` (ver `apps/backend/Makefile`).
  - Golden files para outputs complejos con snapshots en `apps/backend/testdata/`.
  - Race detector ON para tests de concurrencia (`WalletService.Debit` con 10 goroutines).
  - Rollback cubierto (`OrderService.Finalize` cuando la transacción a DB falla a mitad del flujo, ver §5.3).
  - **Build tag** `//go:build integration` para tests de integración con testcontainers (no se ejecutan en cada PR; sí en nightly y pre-release).

### A11. El Backend es un Monolito Lógico y Cohesivo

> Todo el backend Go vive bajo **un único `go.mod`** (`apps/backend/go.mod`). Las refactorizaciones son atómicas: un cambio toca todos los archivos necesarios en un solo commit/PR. Todo el código base compila junto en cada `go build`. Las fronteras internas (`domain/` vs `api/` vs `provisioner/` vs `shared/`) se garantizan por **reglas de lint y convención**, no por barreras físicas de módulos separados con versionado independiente.

- **Por qué existe**: en la etapa actual del proyecto, **modularizar prematuramente** (separar `domain` como módulo publicable, extraer `adapters` como SDK, dividir bounded contexts en microservicios) genera un costo operativo desproporcionado:
  - Infierno de versiones cruzadas (¿qué versión de `domain` usa `server`?).
  - Builds incrementales rotos cuando `domain` cambia y `server` no se recompila.
  - Confusión sobre cuál es la "versión canónica" de una entidad.
  - Refactorings atómicos imposibles (rename de un símbolo requiere PR en N repos).
  - Complejidad del toolchain (multi-module workspaces, replace directives, semver discipline).
- **Regla operativa**: la pureza y el aislamiento se garantizan con:
  - **Reglas de lint** (`domain_purity`, `forbidden_imports`, `contract_drift`).
  - **Code review estricto**: cada PR valida que las fronteras lógicas se respeten.
  - **Pirámide de tests** (§5.2): tests de integración detectan violaciones de capas.
- **Cuándo SÍ se extrae un módulo**: solo cuando hay un consumidor externo real (SDK público para integradores, librería open-source, microservicio deployado independientemente). Hasta entonces, **un solo `go.mod`**.
- **Consecuencia de violarlo**: extraer `apps/backend/internal/domain/` a un módulo separado sin justificación introduce fricción sin beneficio. Si dudas, **deja el código donde está** y agrega una regla de lint.

### A12. Los Contratos son Single Source of Truth

> Los OpenAPI specs viven en `apps/contracts/openapi/` y son la **única fuente de verdad** para el wire format del API. El backend Go genera tipos y handlers desde ellos (`oapi-codegen`); el frontend Next.js (futuro) genera tipos y clientes desde ellos (`openapi-typescript`). **Nadie edita tipos de wire a mano.**

- **Por qué existe**: dos equipos (backend y frontend) consumiendo el mismo contrato evitan la divergencia silenciosa que puede introducir bugs de camelCase vs snake_case, campos faltantes, tipos desalineados. Un cambio en `apps/contracts/openapi/api.yaml` regenera código en ambos lados en el mismo commit, garantizando que **lo que el backend expone es exactamente lo que el frontend consume**.
- **Reglas operativas**:
  - **Cambio en wire format** → primero `apps/contracts/openapi/api.yaml`, luego `pnpm generate`, luego usar el tipo generado en el código de app.
  - **Cambio breaking en OpenAPI** → bump major de `apps/contracts/` + entrada BREAKING en `CHANGELOG.md` + notificación al equipo frontend.
  - **Lint rule `contract_drift`**: CI falla si un handler de Go define un tipo de wire que no proviene del OpenAPI generado.

---

## 2. Estándar Go y Toolchain del Monorepo

### 2.1 Stack Tecnológico (vista monorepo)

| Capa | Tecnología | Versión objetivo | Justificación |
|---|---|---|---|
| **Monorepo orchestrator** | **Turborepo** | última estable | Caching distribuido de tasks (`build`, `lint`, `test`), ejecución paralela entre apps (Go y TS). |
| **Workspace manager** | **pnpm workspaces** (preferido) o npm workspaces | pnpm 9+ | Más rápido, mejor soporte para monorepos, contenido direccionable. |
| **Contratos HTTP** | **OpenAPI 3.1** + **oapi-codegen** (Go) + **openapi-typescript** (TS) | últimas | Specs declarativos; generación bidireccional Go↔TS desde el mismo `api.yaml`. |
| **Documentación interactiva del API** | **Swagger UI** (o **Redoc**, **Scalar**) | últimas | Renderiza `api.yaml` como docs navegables con "Try it out". Default: Scalar. |
| Lenguaje backend | **Go** | 1.26 (o 1.25+) | Rendimiento, tipado fuerte, ergonomía para servicios concurrentes. |
| Inyección de dependencias | **Uber Fx** | v1.24.0 | Estándar de facto en Go para DI modular y lifecycle management. |
| Arquitectura backend | **Hexagonal adaptada** (no pura) | — | Pureza del dominio + organización por entry-point (`api/`, `provisioner/`). |
| HTTP | **net/http + chi router** | chi/v5 v5.3.0 | Router minimalista, idiomático Go, sin overhead de frameworks. |
| JSON | **encoding/json** (stdlib) | — | Sin `go-playground/validator`. Sin paquetes externos de validación. |
| Base de datos | **PostgreSQL 16** | — | Soporte robusto de JSONB para datos estructurados. |
| Driver DB | **pgx v5** | jackc/pgx/v5 v5.10.0 | Mejor performance, tipos nativos para JSONB. **No** ORM. SQL explícito. |
| Migraciones | **pressly/goose v3** | v3.27.2 | Versionado de schema. CLI `apps/backend/cmd/migrate/`. |
| LLM Client | **openai-go** (oficial) | v1.12.0 | Structured Outputs (JSON Schema estricto) nativo. |
| Logging | **log/slog** (stdlib) | — | Único logger. `PIIHandler` bloquea PII. **No** zap. |
| Config | **os.Getenv manual** en `apps/backend/internal/shared/config/` | — | Sin `envconfig` ni librerías externas. |
| UUIDs | **Generación interna con stdlib** | RFC 9562 v7 | `crypto/rand + time + fmt`. Prohibido `google/uuid`. |
| Tracing | **go.opentelemetry.io/otel** | v1.44.0 + OTLP gRPC | Degraded mode si `OTEL_EXPORTER_OTLP_ENDPOINT` vacío. |
| Métricas | **prometheus/client_golang** | v1.20.5 | Endpoint `/metrics`. HTTP, LLM, negocio. |
| Testing | **stretchr/testify** + **go.uber.org/mock** + **testcontainers-go** | v1.11.1 / v0.6.0 | Aserciones legibles, mocks tipados con `mockgen`, Postgres efímero. |
| **Frontend (futuro)** | **Next.js 14+** + TypeScript 5+ | — | Ver `MANIFEST_FRONTEND.md §2`. |

### 2.2 Configuración de turborepo

```json
// turbo.json (raíz del monorepo)
{
  "$schema": "https://turbo.build/schema.json",
  "globalDependencies": ["**/.env.*local"],
  "tasks": {
    "build": {
      "dependsOn": ["^build"],
      "outputs": ["dist/**", "bin/**", "gen/**"]
    },
    "lint": {
      "dependsOn": ["^build"]
    },
    "test": {
      "dependsOn": ["^build"],
      "outputs": ["coverage/**"]
    },
    "test-integration": {
      "dependsOn": ["^build"],
      "outputs": ["coverage/**"]
    },
    "typecheck": {},
    "dev": {
      "cache": false,
      "persistent": true
    },
    "generate": {
      "cache": false,
      "outputs": ["gen/**", "**/gen_*.go"]
    }
  }
}
```

```json
// package.json (raíz)
{
  "name": "monorepo",
  "private": true,
  "packageManager": "pnpm@9.15.4",
  "scripts": {
    "build": "turbo run build",
    "lint": "turbo run lint",
    "test": "turbo run test",
    "test-integration": "turbo run test-integration --filter=backend",
    "generate": "turbo run generate",
    "typecheck": "turbo run typecheck",
    "dev:backend": "turbo run dev --filter=backend",
    "dev:frontend": "turbo run dev --filter=frontend"
  },
  "devDependencies": {
    "turbo": "latest",
    "typescript": "latest"
  }
}
```

### 2.3 Reglas de comunicación entre apps

> **Axioma A12 en acción**: las apps del monorepo **solo se comunican vía contratos OpenAPI**, nunca vía imports cruzados.

| Origen | Destino | Mecanismo |
|---|---|---|
| `apps/backend/` | `apps/frontend/` | OpenAPI HTTP contract (`apps/contracts/openapi/api.yaml`). |
| `apps/backend/cmd/api/` | `apps/backend/cmd/provisioner/` | Comparten `internal/domain/`, `internal/shared/`, `migrations/` (mismo `go.mod`). |
| `apps/backend/internal/api/` | `apps/backend/internal/provisioner/` | Comparten domain y shared; **no importan código entre sí** (van a través de domain events). |
| `apps/frontend/` | `apps/contracts/` | `pnpm generate` → tipos TS en `apps/frontend/src/lib/api/gen.ts`. |
| `apps/backend/` | `apps/contracts/` | `oapi-codegen` → tipos Go en `apps/backend/internal/api/handlers/gen_*.go`. |

**Prohibido**:
- ❌ `apps/frontend/src/lib/api/client.ts` haciendo `import` desde `apps/backend/`.
- ❌ `apps/backend/internal/api/` importando desde `apps/backend/internal/provisioner/` (van vía domain events).
- ❌ Cualquier PR que modifique código de wire (request/response types) sin antes modificar `apps/contracts/openapi/`.

---

## 3. Esqueleto Arquitectónico

### 3.1 Árbol Canónico del Monorepo

```
monorepo/
│
├── turbo.json                       # Orquestador de tasks entre apps
├── package.json                     # Scripts globales + workspaces
├── pnpm-workspace.yaml              # Definición de workspaces
├── pnpm-lock.yaml
├── .gitignore
├── .editorconfig
├── .nvmrc                           # Versión Node para frontend
├── .tool-versions                   # Versiones asdf (Go, Node, pnpm)
│
├── docs/                            # Documentación cross-app
│   ├── ARCHITECTURE.md
│   ├── PRODUCT_DOMAIN.md
│   ├── RUNBOOK.md
│   ├── PRODUCTION_ENV.md
│   ├── SECRET_ROTATION.md
│   ├── SECURITY_DEBT.md
│   ├── API_CONTRACT.md
│   └── ...
│
├── apps/
│   │
│   ├── contracts/                   # OpenAPI specs compartidos (Go↔TS SSOT)
│   │   ├── package.json             # npm package para tooling
│   │   ├── openapi/
│   │   │   ├── api.yaml             # Contrato HTTP principal
│   │   │   ├── events.yaml          # Schemas de eventos de dominio
│   │   │   └── README.md
│   │   └── scripts/
│   │       └── generate.sh          # oapi-codegen + openapi-typescript
│   │
│   ├── backend/                     # Todo el ecosistema Go vive aquí
│   │   ├── go.mod                   # ÚNICO archivo go.mod del backend
│   │   ├── go.sum
│   │   ├── Dockerfile               # multi-stage distroless + nonroot
│   │   ├── .dockerignore
│   │   ├── Makefile                 # targets build, test, lint, migrate, generate
│   │   ├── .golangci.yml            # reglas custom: domain_purity, forbidden_imports, contract_drift, goroutine_context, docs_no_internal_in_prod
│   │   │
│   │   ├── cmd/                     # Entry points (binarios)
│   │   │   ├── api/main.go          # HTTP API server (puerto 8080)
│   │   │   ├── provisioner/main.go  # Jobs batch (purge-raw-data, execute-deletions, refresh-aggregates)
│   │   │   └── migrate/main.go      # CLI migrations (pressly/goose)
│   │   │
│   │   ├── internal/
│   │   │   ├── shared/              # Utilidades transversales (todos las consumen)
│   │   │   │   ├── logger/          # slog + PIIHandler + ctxkeys
│   │   │   │   ├── db/              # pgx pool + RLS helpers + lifecycle
│   │   │   │   ├── config/          # os.Getenv manual + Load()
│   │   │   │   ├── metrics/         # Prometheus + /metrics
│   │   │   │   ├── telemetry/       # OpenTelemetry + OTLP
│   │   │   │   └── httpx/           # chi router compartido + middlewares globales
│   │   │   │
│   │   │   ├── domain/              # ⚠️ PURO. Solo stdlib. Cero imports externos.
│   │   │   │   ├── entity.go        # entidades y agregados
│   │   │   │   ├── value_object.go  # value objects
│   │   │   │   ├── errors.go        # errores de dominio
│   │   │   │   ├── identifiers.go   # UUID v7 + tipos nominales
│   │   │   │   ├── events.go        # DomainEvent structs
│   │   │   │   │
│   │   │   │   ├── identity/        # Bounded context: identidades, cuentas
│   │   │   │   ├── organization/    # Bounded context: tenants, API keys, planes
│   │   │   │   ├── billing/         # Bounded context: ledger, transacciones append-only
│   │   │   │   └── audit/           # Bounded context: access grants + eventos de auditoría
│   │   │   │
│   │   │   ├── api/                 # Todo lo específico del binario HTTP API
│   │   │   │   ├── handlers/        # chi handlers
│   │   │   │   │   └── gen_*.go     # código generado por oapi-codegen (NO editar)
│   │   │   │   ├── services/        # casos de uso
│   │   │   │   │   └── event_handlers/  # Subscribers a DomainEvent
│   │   │   │   ├── ports/           # interfaces (repositories, extractors, ...)
│   │   │   │   │   ├── events/      # EventDispatcher, EventHandler, Outbox
│   │   │   │   │   ├── storage/     # UnitOfWork
│   │   │   │   │   └── mocks/       # mocks generados por mockgen
│   │   │   │   ├── adapters/        # postgres, external services, events
│   │   │   │   │   ├── postgres/    # UnitOfWork + tx helpers + repositories/
│   │   │   │   │   └── events/      # in-memory dispatcher + outbox relay
│   │   │   │   ├── docs/            # Swagger UI / Scalar UI embebido (handler opcional)
│   │   │   │   ├── di/              # fx.Module del API
│   │   │   │   └── errors.go        # mapeo domain.Error → HTTP status
│   │   │   │
│   │   │   └── provisioner/         # Todo lo específico del binario provisioner
│   │   │       ├── handlers/        # subcomandos CLI (purge-raw-data, ...)
│   │   │       ├── services/        # jobs batch
│   │   │       │   └── event_handlers/  # Subscribers a DomainEvent
│   │   │       ├── ports/           # interfaces que necesita
│   │   │       │   ├── events/      # EventDispatcher, EventHandler, Outbox
│   │   │       │   ├── storage/     # UnitOfWork
│   │   │       │   └── mocks/       # mocks generados por mockgen
│   │   │       ├── adapters/        # implementaciones (reusa shared/db + shared/logger)
│   │   │       │   └── postgres/repositories/  # repos específicos del provisioner
│   │   │       └── di/              # fx.Module del provisioner
│   │   │
│   │   ├── migrations/              # SQL versionados (visible desde cmd/migrate)
│   │   │   └── NNNNNN_<slug>.up.sql / .down.sql
│   │   │
│   │   ├── testdata/                # golden files y snapshots
│   │   │
│   │   └── test/                    # tests cross-cutting (e2e, integration end-to-end)
│   │       └── e2e/
│   │
│   └── frontend/                    # FUTURO — Next.js (cuando se integre)
│       ├── package.json
│       ├── next.config.js
│       ├── tsconfig.json
│       ├── tailwind.config.ts
│       └── src/
│           ├── app/
│           ├── components/
│           ├── lib/
│           │   └── api/             # Cliente generado desde apps/contracts/openapi
│           └── types/               # Tipos generados desde OpenAPI
│
├── ops/                             # Infraestructura y despliegue
│   ├── terraform/                   # IaC (DB, Redis, OTel collector)
│   ├── docker/                      # docker-compose para dev (Postgres, etc.)
│   ├── k8s/                         # manifiestos Kubernetes (futuro)
│   └── scripts/                     # Bash ejecutables globales
│       ├── smoke.sh
│       ├── security_audit.sh
│       ├── pii_audit.sh
│       ├── gen_mocks.sh
│       └── ...
│
└── .github/
    └── workflows/
        ├── ci.yml                   # lint + test + build de TODO el monorepo
        ├── contracts.yml            # valida que apps/backend y apps/frontend usen la misma versión del OpenAPI
        ├── deploy-staging.yml       # build + push imágenes
        └── security.yml             # gosec + govulncheck
```

### 3.2 Reglas de Ubicación

| Si estás creando... | Debe vivir en... | NO en... |
|---|---|---|
| Un error nuevo del dominio | `apps/backend/internal/domain/errors.go` | un handler o un service |
| Una entidad de negocio | `apps/backend/internal/domain/` o `apps/backend/internal/domain/<bc>/` | api/ ni provisioner/ ni shared/ |
| Un servicio del API HTTP | `apps/backend/internal/api/services/` | provisioner/ ni shared/ |
| Un handler HTTP | `apps/backend/internal/api/handlers/` | un service ni shared/ |
| Un servicio del provisioner | `apps/backend/internal/provisioner/services/` | api/ ni shared/ |
| Un comando CLI del provisioner | `apps/backend/internal/provisioner/handlers/` (subcomandos) | api/ ni cmd/ raíz |
| Un repositorio PostgreSQL del API | `apps/backend/internal/api/adapters/postgres/repositories/` | provisioner/ ni shared/ |
| Un repositorio PostgreSQL del provisioner | `apps/backend/internal/provisioner/adapters/postgres/repositories/` | api/ ni shared/ |
| El pool de pgx, RLS helpers | `apps/backend/internal/shared/db/` | un handler ni un service |
| El logger | `apps/backend/internal/shared/logger/` | api/ ni provisioner/ |
| La config | `apps/backend/internal/shared/config/` | api/ ni provisioner/ |
| Un middleware HTTP reutilizable | `apps/backend/internal/shared/httpx/middleware/` | api/ ni provisioner/ (los entry-points pueden reusar, no redefinir) |
| Una migración SQL | `apps/backend/migrations/NNNNNN_<slug>.up.sql` | cualquier otra carpeta |
| Un OpenAPI spec | `apps/contracts/openapi/` | apps/backend ni apps/frontend |
| Un mock | `apps/backend/internal/<entrypoint>/ports/mocks/` (generado) | un service ni shared/ |
| Código generado de OpenAPI (Go) | `apps/backend/internal/api/handlers/gen_*.go` (commiteado para reproducibilidad) | nunca editar a mano |
| Código generado de OpenAPI (TS, frontend) | `apps/frontend/src/lib/api/gen.ts` (regenerable) | nunca editar a mano |
| El handler que sirve Swagger UI / Scalar UI | `apps/backend/internal/api/docs/` (montado opcionalmente bajo `APP_API_DOCS_ENABLED=true`) | nunca bajo `/v1/` |
| IaC (Terraform, Helm, k8s) | `ops/` | apps/ ni raíz del monorepo |
| Scripts bash ejecutables | `ops/scripts/` (globales) o `apps/backend/` (específicos del backend) | raíz del monorepo |

### 3.3 Capas y Responsabilidades

| Capa / App | Responsabilidad | NO hace |
|---|---|---|
| `apps/contracts/` | Especificaciones wire (HTTP, eventos). SSOT cross-app. OpenAPI specs + código generado. | Lógica de negocio. Validaciones. |
| `apps/backend/cmd/<entrypoint>/` | Composición de módulos Fx, arrancar/detener procesos. | Lógica de negocio. |
| `apps/backend/internal/shared/` | Infraestructura transversal (db pool, logger, config, metrics, telemetry, router, middlewares). | Decisiones de negocio. Reglas de validación. |
| `apps/backend/internal/domain/` | Reglas de negocio puras. Estructuras de datos. Errores. Domain Events. | Saber de HTTP, DB, LLM, JSON, JSONB. |
| `apps/backend/internal/api/` | Handlers HTTP, services de casos de uso del API, ports y adapters del API. | Compartir lógica con provisioner (van vía domain events). |
| `apps/backend/internal/provisioner/` | Handlers CLI, services de jobs batch, ports y adapters del provisioner. | Compartir lógica con api (van vía domain events). |
| `apps/frontend/` (futuro) | UI, componentes, cliente HTTP generado, tipos generados. | Lógica de negocio backend. Acceso directo a DB. |

**Nota arquitectónica importante**: la separación `api/` ↔ `provisioner/` es **lógica, no física**. Si una pieza de lógica es genuinamente transversal, vive en `shared/`. Si es del dominio, vive en `domain/`. Si es específica de un entry-point, vive en su carpeta.

---

## 4. Anti-patrones Históricos

Cicatrices documentadas. Errores que el equipo **ya cometió** o que son trampas conocidas del monorepo. Si los descubres de nuevo, documéntalos en `docs/bugs/` con el mismo formato.

### AP1. Mezclar el ID público con el ID interno

- **Qué pasó**: durante el diseño de endpoints, varios handlers asumieron que "el ID del usuario" era uno solo. Se pasó el ID público a queries que esperaban el ID interno (y viceversa). Resultado: 500 internal errors por policies RLS que rechazaban el query (el ID público no tiene `tenant_id`), o peor, brechas cross-tenant porque se indexaba por la columna equivocada.
- **Lección**: son **dos entidades distintas** del dominio, no dos nombres para lo mismo:
  - ID público — **portable**, cross-tenant, vive en la tabla de identidades SIN `tenant_id`. Es el identificador global del usuario.
  - ID interno — **tenant-scoped**, vive en la tabla de recursos CON `tenant_id`. Es el ID local en un tenant.
- **Regla**: el `path param` debe nombrarse según lo que acepta. Si el endpoint acepta el ID público, el path es `/v1/identities/{publicId}/*`. Si acepta el ID interno, el path es `/v1/resources/{internalId}`. **Nunca mezclar**.

### AP2. Asumir camelCase por defecto en structs de Go

- **Qué pasó**: structs se serializaban con campos en camelCase porque `encoding/json` aplica el nombre del field Go como tag por defecto. El resto del API usa snake_case. El frontend asumía snake_case. Resultado: nulls silenciosos en el deserializador del frontend.
- **Lección**: **siempre** declarar `json:"snake_case"` explícitamente en cada field exportado de un struct que cruza el wire. Los `json:"..."` tags son **metadata del struct** (string literals procesados por `reflect`), NO importan paquetes externos. Por tanto, **NO violan la purity rule** del dominio (A1).
- **Regla**: el contrato del API manda. Con A12 y el uso de `oapi-codegen`, los wire types se generan desde `apps/contracts/openapi/api.yaml` y ya vienen con los tags correctos. **Nunca editar a mano un archivo `gen_*.go`.**

### AP3. Mapear un Error de Negocio como `ValidationError`

- **Qué pasó**: el service retornaba `*domain.ValidationError` cuando el tenant no tenía saldo. El `error_mapper.go` lo traducía a `422 validation_error` con `field="tenant_id"`. El frontend no podía distinguir "tu tenant no tiene saldo" de "enviaste un JSON malformado".
- **Lección**: los errores de negocio deben tener su propio tipo (`*domain.InsufficientBalanceError`) y mapear a códigos HTTP específicos (`402 Payment Required` con `code="insufficient_balance"`). Reutilizar `ValidationError` para errores conceptualmente distintos es un antipatrón de A5.
- **Regla**: si un error merece un código HTTP distinto, merece un tipo de error distinto. Nunca dos conceptos de negocio distintos compartiendo el mismo tipo de error.

### AP4. Idempotencia mal entendida vía `X-Idempotency-Key`

- **Qué pasó**: el frontend asumió que enviar `X-Idempotency-Key` garantizaría "exactamente-una-ejecución" durante retries agresivos. El backend implementó idempotencia **por estado** (`IsTerminal()` + Unique constraints) pero **NO** cacheaba respuestas por clave. Resultado: reintentos agresivos del frontend causaban `409 resource_already_finalized` que el frontend no manejaba como éxito.
- **Lección**: idempotencia por estado + Unique constraints es más fuerte que cache por clave (no requiere almacenamiento adicional, es correcta por construcción). Pero el cliente debe ser informado de los códigos de respuesta que **son éxito idempotente** (`409 resource_already_finalized`).
- **Regla**: documentar explícitamente en el contrato OpenAPI (response descriptions) qué status codes son éxito idempotente. El cliente debe tratarlos como éxito, no como error.

### AP5. Campo del contrato ausente del modelo de dominio

- **Qué pasó**: el contrato documentaba `expires_at` en el response. El modelo del dominio no incluía ese campo. Resultado: el frontend no podía mostrar el countdown real; lo calculaba localmente (`started_at + 24h`). Cuando el dominio cambió el modelo de expiración, el frontend quedó con datos stale.
- **Lección**: cuando un contrato documenta un campo, ese campo **debe existir** en el modelo de dominio o documentarse explícitamente como gap.
- **Regla**: cualquier gap entre contrato y modelo de dominio se documenta en `docs/API_CONTRACT.md` (Gaps conocidos) con plan de cierre. No se deja divergir silenciosamente.

### AP6. Goroutines sin `context.Context` cancelable

- **Qué pasó**: en un extractor externo (LLM), se lanzaban goroutines para paralelizar llamadas sin pasarles un contexto cancelable. Cuando el cliente HTTP desconectaba, las goroutines seguían ejecutando y consumiendo recursos hasta terminar.
- **Lección**: **toda** goroutine debe recibir un `context.Context` y chequear `ctx.Done()` en cada iteración. Sin excepción.
- **Regla**: `go func() { ... }` sin `ctx` es un bug latente. CI falla el linter si detecta `go func(` sin `ctx` en los argumentos (regla custom `goroutine_context`).

### AP7. Service publica evento sin esperar commit de la transacción

- **Qué pasó**: un service publicaba un DomainEvent llamando `eventDispatcher.Publish(...)` directamente, sin pasar por el outbox. El handler del evento intentaba leer el recurso de la DB y no lo encontraba, porque la transacción principal aún no había hecho commit. Race condition clásico.
- **Lección**: **publicar un evento antes de que la transacción que lo causó haya commiteado es un bug**. La solución canónica es el patrón outbox (ver §5.1): el evento se inserta en la misma transacción que el cambio de estado, y un relay lo publica post-commit.
- **Regla**: **nunca** publicar un DomainEvent desde un service que está dentro de un `UnitOfWork.InTransaction(...)` directamente al dispatcher. Publicar al outbox dentro de la transacción. El relay se encarga después.

### AP8. Transacción "implícita" abriendo conexiones por doquier

- **Qué pasó**: un service necesitaba atomicidad entre dos repositorios. El dev pensó "lo más fácil es iniciar una transacción SQL aquí mismo" y expuso `*pgxpool.Pool.Begin(ctx)` desde el service. Resultado: el service ahora depende de `pgx` directamente (rompe A1/A2), los tests requieren mockear pgx (no mocks tipados de puertos), y migrar a otro driver es un infierno.
- **Lección**: el service **nunca** debe iniciar transacciones SQL. Solo el adapter de UnitOfWork puede hacerlo. El service recibe el `UnitOfWork` port por inyección y lo invoca como una abstracción.
- **Regla**: ver §5.3 para el patrón completo. El service tiene código Go puro; la transacción es un detalle del adapter.

### AP-MR1. Múltiples `go.mod` en `apps/backend/`

- **Qué pasó**: la tentación de "modularizar" creando un `go.mod` para `apps/backend/internal/domain/`, otro para `apps/backend/internal/api/`, etc. Con la idea de "reusar `domain` desde el SDK".
- **Lección**: viola A11. Mientras no haya un consumidor externo real, un solo `go.mod` evita el infierno de versiones y simplifica refactorings atómicos.
- **Regla**: si quieres reusar `domain` desde otro lugar dentro del mismo `apps/backend/`, déjalo en el mismo módulo. Solo extrae cuando haya un `cmd/sdk/main.go` o un import desde **fuera** de `apps/backend/`.

### AP-MR2. Handlers que importan tipos de wire desde código de aplicación

- **Qué pasó**: el handler usaba structs de request/response definidos en un service. Resultado: el cliente no puede generar tipos porque el wire format vive mezclado con la lógica.
- **Lección**: viola A12. El wire format vive en `apps/contracts/openapi/api.yaml` y se genera.
- **Regla**: el handler hace **dos conversiones explícitas** — de wire (generado) a dominio al recibir, y de dominio a wire (generado) al responder. La capa `internal/api/handlers/` es el **único lugar** donde se importan los tipos `gen_*.go`.

### AP-MR3. Frontend importando código de Go directamente

- **Qué pasó**: dev junior intenta `import type { Entity } from '../../apps/backend/internal/domain/...'` desde un componente React. Resultado: el build de TS falla estrepitosamente.
- **Lección**: viola la regla fundamental de monorepos polyglot. Frontend y backend son apps separadas que se comunican solo por HTTP (vía OpenAPI).
- **Regla**: si frontend necesita un tipo del dominio, lo consume vía el cliente generado desde `apps/contracts/openapi/`. Nunca via import cruzado.

### AP-MR4. Variables de entorno compartidas entre apps sin namespace

- **Qué pasó**: el backend lee `APP_DATABASE_URL` y el frontend lee `NEXT_PUBLIC_API_URL` sin un prefijo claro. Un dev junior define `DATABASE_URL` esperando que funcione en el backend; falla porque el backend busca `APP_DATABASE_URL`.
- **Lección**: las env vars de cada app deben tener un prefijo identificable.
- **Regla**:
  - Backend (Go): prefijo del proyecto (`APP_`).
  - Frontend (Next.js): prefijo `NEXT_PUBLIC_` para vars expuestas al cliente + `APP_FRONTEND_` para vars internas del frontend.
  - Contratos: prefijo `APP_CONTRACTS_` (si los specs tienen metadata runtime, ej. URLs de validación).

### AP-MR5. `package.json` raíz sin `private: true`

- **Qué pasó**: alguien publica accidentalmente el `package.json` raíz a npm registry porque le faltaba `"private": true`.
- **Lección**: el `package.json` raíz de un monorepo NUNCA debe publicarse. Es solo orquestador.
- **Regla**: `"private": true` es **obligatorio** en `package.json` raíz. Verificar en CI.

### AP-MR6. Cambios en `apps/contracts/` sin bump de versión

- **Qué pasó**: dev edita `api.yaml` agregando un campo nuevo, hace commit. El backend regenera y todo funciona. Pero el frontend no regenera y empieza a tener nulls donde espera un nuevo campo.
- **Lección**: un cambio en el contrato es un **evento de release**. No basta con commitear el YAML.
- **Regla**:
  1. Editar `apps/contracts/openapi/api.yaml`.
  2. `pnpm generate` en ambos lados.
  3. **Si es breaking**: bump major de `apps/contracts/` + entrada BREAKING en `CHANGELOG.md` + tag `contracts-v2.0.0`.
  4. **Si es no-breaking**: bump minor + tag `contracts-v1.x.0`.
  5. CI verifica que el hash del código generado en backend y frontend coincida.

### AP-MR7. Turborepo sin cache hit en CI

- **Qué pasó**: CI tarda 25 min en cada PR porque `turbo` no cachea entre runs (porque no hay remote cache configurado).
- **Lección**: el caching distribuido es el **principal valor** de turborepo. Sin él, es solo un orquestador paralelo.
- **Regla**:
  - Configurar `TURBO_TOKEN` + `TURBO_TEAM` (Vercel Remote Cache) en secrets de GitHub Actions desde el día 1.
  - Alternativa OSS: `turborepo-remote-cache` self-hosted en un contenedor del cluster.
  - Métrica objetivo: cache hit ratio > 80% después de la primera semana.

### AP-MR8. Tests de integración que requieren Docker pero no usan testcontainers

- **Qué pasó**: dev define un test Tier 3 que necesita Postgres real, pero usa `os.Getenv("DATABASE_URL")` apuntando al Postgres local. Resultado: el test falla en CI porque no hay Postgres local en el runner.
- **Lección**: cualquier test que requiera infraestructura externa debe usar testcontainers (o un mock equivalente). Nunca depender de servicios preinstalados en el runner de CI.
- **Regla**: tests con `//go:build integration` DEBEN usar `testcontainers-go` para Postgres 16. Sin excepciones.

### AP-MR9. Swagger UI / Scalar UI expuesto en producción sin auth

- **Qué pasó**: dev monta el UI de Swagger/Scalar en producción bajo `/docs` sin autenticación, para "facilitar el debugging". Resultado: cualquier persona con la URL puede ver todo el contrato del API, incluyendo endpoints internos de admin, y ejecutar requests reales contra el backend (botón "Try it out").
- **Lección**: el UI renderiza el OpenAPI spec completo. Si contiene endpoints sensibles (admin, debug, métricas internas) o permite ejecutar requests reales, exponerlo sin auth es un riesgo de seguridad y de información.
- **Regla**:
  - En **desarrollo y staging**: el UI se monta bajo `/docs` (o ruta similar) **con auth de admin** (bearer token de `APP_API_DOCS_TOKEN`).
  - En **producción**: el UI **NO se expone** desde el backend. Se sirve desde `apps/frontend/` (cuando exista), que controla acceso via sesión de usuario admin.
  - El UI **NO incluye endpoints marcados como `x-internal: true`** en el spec (filtrado en build del HTML estático).
  - Lint rule `docs_no_internal_in_prod`: CI falla si `apps/contracts/openapi/*.yaml` contiene operaciones con `x-internal: true` y `APP_API_DOCS_ENABLED=true` en el deploy de producción.

---

## 5. Patrones Arquitectónicos Transversales

Tres patrones que cruzan capas y que todo proyecto monorepo debe resolver de manera canónica. Aquí se destila la decisión y la implementación; la validación automatizada vive en los tests golden y en las reglas de lint.

### 5.1 Comunicación Asíncrona entre Bounded Contexts (Outbox + In-Memory Bus)

**Problema**: A3 obliga a que `identity`, `organization`, `billing` y `audit` no se importen entre sí. Sin embargo, hay hechos de negocio que **un** bounded context produce y **otros** deben consumir:

- `billing` produce `BalanceChanged` cuando se modifica el saldo de un tenant.
- `audit` consume `BalanceChanged` para registrar quién accedió a qué recurso.
- `identity` produce `IdentityIssued` cuando se emite un nuevo ID global.
- `organization` consume `IdentityIssued` para asociar el ID al tenant.

**Decisión canónica: Outbox Pattern + In-Memory Event Bus (default).**

#### Componentes

| Componente | Ubicación | Responsabilidad |
|---|---|---|
| `DomainEvent` (structs) | `apps/backend/internal/domain/events.go` | Tipos puros de hechos de negocio. Inmutables. |
| `EventDispatcher` (port) | `apps/backend/internal/<entrypoint>/ports/events/event_dispatcher.go` | Interfaz: `Dispatch(ctx, event) error` + `Subscribe(eventType, handler)`. |
| `EventHandler` (port) | `apps/backend/internal/<entrypoint>/ports/events/event_handler.go` | Interfaz: `Handle(ctx, event) error`. Un handler por evento. |
| `Outbox` (port) | `apps/backend/internal/<entrypoint>/ports/events/outbox.go` | Interfaz: `Append(ctx, event) error`. Persiste el evento en la misma transacción. |
| `OutboxEvent` (entity) | `apps/backend/internal/domain/events.go` | `(ID, EventType, Payload, CreatedAt, PublishedAt, Attempts)`. El relay actualiza `published_at`/`attempts` post-commit. |
| `InMemoryEventDispatcher` (adapter) | `apps/backend/internal/<entrypoint>/adapters/events/in_memory_dispatcher.go` | Implementación default con `chan DomainEvent` (buffer 1024) + worker pool. |
| `OutboxRelay` (adapter) | `apps/backend/internal/<entrypoint>/adapters/events/outbox_relay.go` | Background goroutine que lee `outbox_events` no publicados y los entrega al dispatcher. |
| Handlers del API | `apps/backend/internal/api/services/event_handlers/` | Implementaciones de `EventHandler` para cada evento que el API consume. |
| Handlers del provisioner | `apps/backend/internal/provisioner/services/event_handlers/` | Implementaciones de `EventHandler` para cada evento que el provisioner consume. |

**Nota**: los ports de eventos (`EventDispatcher`, `EventHandler`, `Outbox`) se **duplican por entry-point** (`api/ports/events/` y `provisioner/ports/events/`), porque `api/` y `provisioner/` no se importan entre sí. La interfaz es idéntica; solo el paquete que la define difiere.

#### Flujo de un evento

```
[1] Service ejecuta caso de uso dentro de uow.InTransaction(ctx, func(txCtx) {
[2]   entityRepo.Save(txCtx, entity)
[3]   outbox.Append(txCtx, ResourceCreated{...})  // misma transacción
[4] })
[5] uow hace COMMIT → datos + evento visibles atómicamente
[6] OutboxRelay.tick():
[7]   SELECT * FROM outbox_events WHERE published_at IS NULL ORDER BY id LIMIT 100;
[8]   for each event: dispatcher.Dispatch(ctx, event); UPDATE SET published_at = NOW();
[9] InMemoryEventDispatcher.Dispatch:
[10]  encuentra handlers suscritos a eventType
[11]  invoca cada handler.Handle(ctx, event) en goroutine (con backpressure)
[12] handler de audit: auditRepo.Insert(ctx, AuditEvent{...})  // nuevo append-only
```

#### Reglas

1. **Nunca** `dispatcher.Dispatch(...)` directamente desde un service dentro de un UnitOfWork. **Siempre** `outbox.Append(...)` dentro de la transacción. El relay publica post-commit.
2. **Nunca** un bounded context importa el paquete de eventos de otro bounded context. Solo el paquete raíz `apps/backend/internal/domain/` define los `DomainEvent`.
3. **Handlers son idempotentes**: deben poder ejecutarse 2 veces sin efecto adverso.
4. **Backpressure**: el `InMemoryEventDispatcher` usa un channel buffered + drop policy si el buffer está lleno.
5. **Handlers son services de aplicación**, no adapters. Pueden invocar otros services, repositorios y el UnitOfWork.
6. **Versionado de eventos**: cada `DomainEvent` lleva un campo `Version`. Si el payload cambia de forma breaking, se sube la versión.
7. **Broker externo (futuro)**: cuando el outbox supere ~10k eventos/día sostenidos o se necesite multi-binary, se introduce NATS JetStream o Kafka. El `EventDispatcher` cambia de adapter; el código de los services **no cambia**.

#### Por qué in-memory y no Kafka desde día uno

- Costo cero operacional.
- Bus in-memory **por proceso**: cada binario (`cmd/api`, `cmd/provisioner`) corre su propio `InMemoryEventDispatcher` + `OutboxRelay`. La entrega de eventos **entre binarios** va vía `outbox_events` en DB, nunca por un `chan` compartido.
- Latencia sub-ms dentro del mismo proceso.
- Misma garantía transaccional vía outbox.
- Migración trivial cuando se necesite.

### 5.2 Estrategia de Testing de Integración (Pirámide 3 Tiers + Testcontainers)

**Problema**: A10 exige 100% de cobertura en el dominio y ≥90% en services. Pero los adapters tienen comportamiento real que los mocks no pueden validar: SQL real con RLS, constraints, índices.

**Decisión canónica: Pirámide de Tres Tiers + Testcontainers.**

#### Tier 1 — Unit Puro (sin infraestructura)

- **Qué prueba**: lógica de negocio pura. Dominio 100%. Services con mocks de puertos.
- **Archivos**: `apps/backend/internal/**/*_test.go` (sin sufijo `_integration`).
- **Sin build tag**. Se ejecuta en cada PR.
- **Mockgen** genera los mocks desde `apps/backend/internal/<entrypoint>/ports/`.
- **Sin Docker, sin red, sin DB**: se ejecuta en CI en <30s.
- **Cobertura objetivo**: dominio 100%, services ≥90%.

#### Tier 2 — Adapter Unit con Fakes (sin Docker)

- **Qué prueba**: handlers HTTP con `httptest`, JSON marshalling, validación de input, status codes.
- **Archivos**: `apps/backend/internal/api/handlers/*_test.go` (sin sufijo).
- **Usa `httptest.NewRecorder()` + `chi.NewRouter()`** para testear handlers end-to-end sin red.
- **Mockea el service** vía mocks de las interfaces de servicio.
- **Cobertura objetivo**: adapters ≥70%.

#### Tier 3 — Integration con Testcontainers (Postgres 16 real)

- **Qué prueba**: SQL real, RLS, constraints, transacciones, índices, migraciones, outbox relay.
- **Archivos**: `apps/backend/internal/**/*_integration_test.go` con build tag `//go:build integration`.
- **Comando**: `pnpm test-integration --filter=backend` (NO se ejecuta por defecto).
- **CI nightly + pre-release**: `.github/workflows/ci.yml` corre en `cron: '0 3 * * *'` (3am UTC) y en todo push a `main` con label `ready-for-release`.
- **Infraestructura**: `testcontainers-go` levanta un contenedor Docker de `postgres:16-alpine` por suite de tests.

#### Reglas

1. **Todo test que requiera Postgres real lleva `//go:build integration`**. Sin excepción.
2. **Tier 1 + Tier 2 son obligatorios en cada PR**.
3. **Tier 3 es obligatorio antes de release**.
4. **No `t.Skip()`** en ningún tier.
5. **Race detector ON** en todos los tiers: `go test -race -tags=integration ./...`.
6. **Tier 3 incluye un test del outbox relay** end-to-end.

#### Make targets por app

```makefile
# apps/backend/Makefile
test:                    ## Tier 1 + 2 (cada PR)
    go test -race -count=1 ./...

test-integration:        ## Tier 3 (nightly + pre-release)
    go test -race -count=1 -tags=integration ./...

test-coverage:           ## Reporte de cobertura por capa
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | grep -E "(domain|services|adapters)"

generate:
    oapi-codegen -package httpapi -generate types ../contracts/openapi/api.yaml > internal/api/handlers/gen_types.go
    oapi-codegen -package httpapi -generate chi-server ../contracts/openapi/api.yaml > internal/api/handlers/gen_server.go
```

### 5.3 Gestión de Transacciones Cross-Repository (Unit of Work)

**Problema**: muchos casos de uso requieren atomicidad entre múltiples repositorios que pertenecen a diferentes bounded contexts:

- `OrderService.Finalize` debe: guardar la entidad + debitar el saldo + emitir `OrderFinalized`. Si cualquiera falla, todo debe rollback.
- `AccessGrantService.Revoke` debe: marcar el grant como revocado + emitir `AccessRevoked`. Atómico.

**La pregunta clave**: ¿cómo logra el service atomicidad multi-repo **sin filtrar SQL ni tipos de DB al service** (rompiendo A1 y A2)?

**Decisión canónica: Unit of Work Pattern.**

#### Componentes

| Componente | Ubicación | Responsabilidad |
|---|---|---|
| `UnitOfWork` (port) | `apps/backend/internal/<entrypoint>/ports/storage/unit_of_work.go` | Interfaz: `InTransaction(ctx, fn func(txCtx context.Context) error) error`. |
| `contextKey` (privado) | `apps/backend/internal/<entrypoint>/adapters/postgres/tx_context.go` | Tipo `txKey struct{}` para identificar la transacción en el contexto. |
| `txFromContext` (helper) | `apps/backend/internal/<entrypoint>/adapters/postgres/tx_context.go` | Helper privado: `func ctxTx(ctx) pgx.Tx { ... }`. Devuelve `nil` si no hay tx. |
| `PostgresUnitOfWork` (adapter) | `apps/backend/internal/<entrypoint>/adapters/postgres/unit_of_work.go` | Implementación: `pool.Begin(ctx)`, llama a `fn(ctxWithTx)`, COMMIT o ROLLBACK. |
| Repositorios con soporte tx | `apps/backend/internal/<entrypoint>/adapters/postgres/repositories/*.go` | Cada método detecta `txKey` en el contexto y usa la tx si existe, o adquiere una conexión nueva del pool si no. |

#### Flujo del UoW

```
[1] Service: uow.InTransaction(ctx, func(txCtx context.Context) error {
[2]   entityRepo.Save(txCtx, entity)             // txCtx lleva la tx
[3]   walletRepo.Debit(txCtx, tenantID, 1)       // misma tx
[4]   outbox.Append(txCtx, OrderFinalized{...})  // misma tx
[5]   return nil
[6] })
[7] PostgresUnitOfWork.InTransaction:
[8]   tx, _ := pool.Begin(ctx)
[9]   txCtx := context.WithValue(ctx, txKey{}, tx)
[10]  err := fn(txCtx)
[11]  if err != nil { tx.Rollback(ctx); return err }
[12]  return tx.Commit(ctx)  // todo o nada
```

#### Helper en repositorios (patrón canónico)

```go
// apps/backend/internal/api/adapters/postgres/repositories/entity_repository.go
func (r *PostgresEntityRepository) Save(ctx context.Context, e *domain.Entity) error {
    tx := ctxTx(ctx)  // nil si no hay tx
    if tx != nil {
        return r.saveTx(ctx, tx, e)   // join transacción existente
    }
    return r.savePool(ctx, e)         // adquiere conexión del pool
}
```

#### Por qué este patrón respeta A1 y A2

- El service NO importa `pgx`: solo conoce el port `UnitOfWork` y los ports de repositorios.
- El adapter `PostgresUnitOfWork` es el único que toca `pgxpool.Pool`.
- Los tests del service usan un mock de `UnitOfWork` que simplemente ejecuta `fn(ctx)` sin transacción.

#### Reglas del patrón UoW

1. El service siempre llama `uow.InTransaction(ctx, fn)` para operaciones multi-repo atómicas.
2. Dentro de `fn(txCtx)`, todos los repositorios reciben `txCtx`.
3. El helper `ctxTx(ctx)` es privado del adapter.
4. El outbox se appenda dentro del UoW. El relay publica post-commit.
5. El `Rollback` ocurre automáticamente si `fn(txCtx)` retorna error.
6. Una sola transacción por request HTTP.
7. El timeout de la transacción es responsabilidad del adapter (`SET LOCAL statement_timeout`).
8. Tests Tier 3 del UoW verifican end-to-end con testcontainers.

#### Anti-patrones prohibidos

- ❌ Service que importa `github.com/jackc/pgx/v5` directamente.
- ❌ Service que recibe `*pgxpool.Pool` o `pgx.Tx` por inyección.
- ❌ Repository que retorna `error` con `pgx.PgError` envuelto sin mapear a `*domain.*Error`.
- ❌ Handler que abre transacción antes de invocar el service.
- ❌ Service que retorna `(value, pgx.Tx, error)` para que el handler haga commit.

---

## 6. Estrategia de Monorepo y Despliegue

### 6.1 Por qué monorepo y no multi-repo

- **Atomicidad**: un cambio que toca backend + contracts + frontend se hace en un solo commit, un solo PR, una sola revisión.
- **Visibilidad**: cualquier dev ve todo el código; las decisiones arquitectónicas se comparten.
- **DX (Developer Experience)**: un solo `git clone`, un solo IDE workspace, una sola búsqueda de símbolos.
- **Refactorings seguros**: rename de un símbolo toca todos los call sites en una sola operación atómica.
- **Alineación con A11**: monolito lógico y cohesivo, no fragmentación prematura.

### 6.2 Turborepo como orquestador

`turborepo` ejecuta tasks (`build`, `lint`, `test`, `typecheck`, `generate`) en orden de dependencias entre apps, con **cache distribuido** por hash de inputs.

**Tasks por app**:

| Task | `apps/contracts/` | `apps/backend/` | `apps/frontend/` |
|---|---|---|---|
| `build` | (n/a, solo specs) | `go build ./...` | `next build` |
| `lint` | (n/a) | `golangci-lint run` | `eslint .` |
| `test` | (n/a) | `go test ./...` | `vitest run` |
| `test-integration` | (n/a) | `go test -tags=integration ./...` | (playwright e2e opcional) |
| `typecheck` | (n/a) | (no aplica, Go es tipado) | `tsc --noEmit` |
| `generate` | `bash scripts/generate.sh` | `oapi-codegen` (en make) | `openapi-typescript` |
| `dev` | (n/a) | `go run ./cmd/api` | `next dev` |

**Dependencias entre apps** (en `turbo.json`):

```json
{
  "tasks": {
    "build": { "dependsOn": ["^build"] }
  }
}
```

Esto garantiza que `apps/contracts/` se "construya" (valide) antes que `apps/backend/` y `apps/frontend/` que dependen de sus artefactos generados.

### 6.3 Contratos compartidos (OpenAPI SSOT)

El módulo `apps/contracts/` es el **único lugar** donde se modifica el wire format.

**Estructura de `apps/contracts/openapi/api.yaml`** (extracto):

```yaml
openapi: 3.1.0
info:
  title: API
  version: 1.0.0
servers:
  - url: https://api.example.com/v1
paths:
  /v1/resources:
    post:
      summary: Crear un nuevo recurso
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateResourceRequest'
      responses:
        '201':
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Resource'
components:
  schemas:
    CreateResourceRequest:
      type: object
      required: [owner_id]
      properties:
        owner_id: { type: string, format: uuid }
    Resource:
      type: object
      required: [id, owner_id, status]
      properties:
        id: { type: string, format: uuid }
        owner_id: { type: string, format: uuid }
        status:
          type: string
          enum: [in_progress, completed, abandoned, expired]
```

**Generación Go (backend)** — vía `oapi-codegen`:

```bash
# apps/backend/Makefile
generate:
    oapi-codegen -package httpapi -generate types \
      ../contracts/openapi/api.yaml > internal/api/handlers/gen_types.go
    oapi-codegen -package httpapi -generate chi-server \
      ../contracts/openapi/api.yaml > internal/api/handlers/gen_server.go
```

**Generación TS (frontend futuro)** — vía `openapi-typescript`:

```bash
# apps/frontend/package.json
"scripts": {
  "generate": "openapi-typescript ../contracts/openapi/api.yaml -o src/lib/api/gen.ts"
}
```

**Reglas operativas**:
- Si necesitas un nuevo campo en el wire format, primero lo agregas a `api.yaml`, ejecutas `pnpm generate`, y recién después lo usas en tu código de app.
- `gen_*.go` y `gen.ts` están **commiteados al repo** para reproducibilidad de builds offline.
- Cualquier modificación manual a `gen_*` será rechazada por el lint rule `contract_drift`.

#### Documentación interactiva del API (Swagger UI / Redoc / Scalar)

El OpenAPI spec en `apps/contracts/openapi/api.yaml` se renderiza automáticamente como **documentación navegable** con un componente que soporte "Try it out" (ejecutar requests reales contra el backend). Esto permite a frontend devs, integradores externos y QA explorar el contrato sin necesidad de leer el YAML crudo ni de configurar Postman.

**Opciones soportadas** (elegible vía `APP_API_DOCS_STYLE`):

| Herramienta | Mejor para | Notas |
|---|---|---|
| **Swagger UI** | Estándar de facto, máxima compatibilidad. | Soporta OpenAPI 3.1 completo. UI clásica. Es el proyecto que originalmente creó la especificación (donada luego a Linux Foundation como OpenAPI). |
| **Redoc** | Documentación pública, integradores externos. | UI limpia tipo "docs site". Sin "Try it out" interactivo en su versión open-source. |
| **Scalar** | DX moderna, temas configurables. | Reemplazo moderno de Swagger UI con mejor UX y "Try it out" potente. |

**Decisión por defecto**: **Scalar** (mejor balance entre modernidad, DX y compatibilidad con OpenAPI 3.1).

**Cómo se sirve**:

| Entorno | Mecanismo | URL |
|---|---|---|
| **Desarrollo local** | `cmd/api` monta el UI en una ruta interna | `http://localhost:8080/docs` |
| **Staging** | UI servido por `cmd/api` con auth de admin (bearer `APP_API_DOCS_TOKEN`) | `https://api-staging.example.com/docs` |
| **Producción** | UI **NO expuesto** desde el backend. Se sirve desde el frontend (futuro) en `https://app.example.com/docs` | — |

**Reglas operativas**:
- El UI **lee el spec desde `apps/contracts/openapi/api.yaml`** en build time. No se sirve el YAML directamente en runtime; se sirve el HTML+JS del renderer con el spec embebido.
- En **producción, `/docs` NO debe estar expuesto** desde el backend (ver AP-MR9). Se sirve desde el frontend que controla quién tiene acceso.
- El renderer se elige en build via `APP_API_DOCS_STYLE=scalar|swagger|redoc`.
- Los cambios en `api.yaml` invalidan el cache del UI en el siguiente build.
- Endpoints marcados como `x-internal: true` se filtran del HTML servido en producción.

### 6.4 Convenciones por entry-point

- **`apps/backend/cmd/api/main.go`**: arranca el servidor HTTP en `APP_HTTP_PORT`. Carga `internal/api/di/`. Opcionalmente monta el UI de docs bajo `/docs` si `APP_API_DOCS_ENABLED=true`.
- **`apps/backend/cmd/provisioner/main.go`**: CLI con subcomandos (`purge-raw-data`, `execute-deletions`, `refresh-aggregates`). Carga `internal/provisioner/di/`.
- **`apps/backend/cmd/migrate/main.go`**: CLI de migraciones. No carga dominio (solo `internal/shared/db/`).

Cada entry-point tiene su propio `fx.Module` en `internal/<entrypoint>/di/`. **No se comparten módulos Fx entre entry-points** (evita acoplamiento y reduce blast radius).

### 6.5 CI/CD por monorepo

```yaml
# .github/workflows/ci.yml (extracto)
name: ci
on:
  pull_request:
    paths:
      - 'apps/**'
      - 'docs/**'
      - 'ops/**'
      - '.github/workflows/**'
      - 'turbo.json'
      - 'package.json'
      - 'pnpm-lock.yaml'
      - 'pnpm-workspace.yaml'
  schedule:
    - cron: '0 3 * * *'   # nightly: integración Tier 3
  push:
    branches: [main]

jobs:
  detect-changes:
    runs-on: ubuntu-latest
    outputs:
      contracts: ${{ steps.filter.outputs.contracts }}
      backend: ${{ steps.filter.outputs.backend }}
      frontend: ${{ steps.filter.outputs.frontend }}
    steps:
      - uses: actions/checkout@v4
      - uses: dorny/paths-filter@v3
        id: filter
        with:
          filters: |
            contracts:
              - 'apps/contracts/**'
            backend:
              - 'apps/backend/**'
              - 'apps/contracts/**'
            frontend:
              - 'apps/frontend/**'
              - 'apps/contracts/**'

  validate-contracts:
    # Si cambió apps/contracts/, regenera y valida que ambos lados compilen.
    needs: detect-changes
    if: needs.detect-changes.outputs.contracts == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v3
      - run: pnpm install --frozen-lockfile
      - run: pnpm turbo run generate

  backend:
    needs: detect-changes
    if: needs.detect-changes.outputs.backend == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - uses: pnpm/action-setup@v3
      - run: pnpm install --frozen-lockfile
      - run: pnpm turbo run build lint test --filter=backend
      - run: pnpm turbo run test-integration --filter=backend
        if: github.event_name == 'schedule' || github.event_name == 'push' || contains(github.event.pull_request.labels.*.name, 'ready-for-release')

  frontend:
    needs: detect-changes
    if: needs.detect-changes.outputs.frontend == 'true'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
      - uses: pnpm/action-setup@v3
      - run: pnpm install --frozen-lockfile
      - run: pnpm turbo run build lint typecheck test --filter=frontend
```

**Triggers inteligentes** (vía `dorny/paths-filter`):
- Cambios solo en `apps/backend/**` → corren jobs de backend.
- Cambios en `apps/contracts/**` → corren jobs de backend y frontend (porque regeneran código).
- Cambios solo en `apps/frontend/**` → corren jobs de frontend.
- Cambios en `turbo.json`, `package.json`, `pnpm-lock.yaml`, `pnpm-workspace.yaml` → corren todos los jobs.
- Integración Tier 3 (`test-integration`) corre en: `schedule` (nightly), push a `main`, o PR con label `ready-for-release`.

**Remote cache**: configurar `TURBO_TOKEN` + `TURBO_TEAM` en secrets de GitHub Actions desde el día 1.

### 6.6 Versionado y Release

- **Monorepo raíz**: tags `v1.0.0` (SemVer) para releases globales coordinados.
- **`apps/backend/`**: tags independientes `backend-v1.x.y` cuando se requiera canary de backend-only.
- **`apps/contracts/`**: tags `contracts-v1.x.y`. Cualquier breaking change del OpenAPI requiere bump major.
- **`apps/frontend/`**: tags `frontend-v1.x.y` cuando exista.
- **`CHANGELOG.md`** raíz incluye secciones por app en cada release.

---

## 7. Glosario del Monorepo

| Término | Significado |
|---|---|
| **Monorepo** | Repositorio único que contiene múltiples proyectos/apps relacionados, cada uno con su propio build/test/deploy. |
| **Polyglot monorepo** | Monorepo con apps en **distintos lenguajes** (Go + TypeScript, por ejemplo). |
| **Workspace** | Conjunto de paquetes/apps declarados en `pnpm-workspace.yaml` que comparten `node_modules` raíz y se resuelven entre sí. |
| **Turborepo** | Orquestador de tasks para monorepos JS/TS/Go. Cache distribuido por hash de inputs. |
| **Remote Cache** | Backend de cache de Turborepo compartido entre CI y devs (Vercel o self-hosted). |
| **SSOT (Single Source of Truth)** | Una única fuente canónica de verdad para un dato. En el monorepo, `apps/contracts/openapi/api.yaml` es el SSOT del wire format. |
| **OpenAPI 3.1** | Estándar de especificación de APIs HTTP. Es la fuente declarativa del wire format. Antes conocido como "Swagger Specification" (donada a Linux Foundation). |
| **Swagger** | Ecosistema original de herramientas para APIs (creado por SmartBear en 2011). El nombre "Swagger" se refiere hoy al tooling: Swagger Editor, Swagger UI, Swagger Codegen. La **especificación** se llama ahora OpenAPI. |
| **Swagger UI** | Herramienta open-source que renderiza un OpenAPI spec como documentación HTML navegable con botón "Try it out". Es el estándar de facto para docs de APIs REST. Mantenido por SmartBear. |
| **Redoc** | Alternativa moderna a Swagger UI para documentación de APIs. UI limpia tipo "docs site" optimizada para documentación pública. Sin "Try it out" interactivo en su versión open-source. Mantenido por Redocly. |
| **Scalar** | Alternativa moderna a Swagger UI con mejor UX, temas configurables y "Try it out" potente. Se elige como default por su balance entre modernidad y compatibilidad. |
| **`oapi-codegen`** | Generador de tipos Go + servidor HTTP desde un OpenAPI spec. Usado por el backend. |
| **`openapi-typescript`** | Generador de tipos TS + cliente desde un OpenAPI spec. Usado por el frontend. |
| **`gen_*.go`** | Convención para archivos Go generados por `oapi-codegen`. Prefijo `gen_` los marca como no-editables. |
| **Entry point** | Binario ejecutable del backend. En el monorepo: `cmd/api`, `cmd/provisioner`, `cmd/migrate`. |
| **App** | Unidad independiente del monorepo (`apps/backend`, `apps/contracts`, `apps/frontend`). Tiene su propio package manager y ciclo de release. |
| **Apps contracts** | App `apps/contracts/` que contiene los OpenAPI specs. Es **dependencia** de las otras apps. |
| **Provisioner** | Binario de jobs batch (purge-raw-data, execute-deletions, refresh-aggregates). |
| **Shared layer** | `apps/backend/internal/shared/`. Utilidades transversales consumidas por api/ y provisioner/. NO contiene lógica de negocio. |
| **`pnpm-workspace.yaml`** | Archivo que declara los workspaces del monorepo. Ej: `packages: ['apps/*']`. |
| **`turbo.json`** | Configuración de tasks y dependencias del orquestador Turborepo. |
| **Vercel Remote Cache** | Servicio gestionado de cache distribuido de Turborepo (gratis para OSS). |
| **Try it out** | Botón en Swagger UI / Scalar UI que permite ejecutar requests reales contra el backend desde la documentación. Útil para QA y para integradores externos. |
| **`x-internal: true`** | Extensión OpenAPI que marca un endpoint como interno (no debe aparecer en docs públicos). Usada por el filtro de AP-MR9. |
| **Dominio** | Capa con las reglas de negocio puras y entidades. La raíz del sistema. |
| **Puerto** | Interfaz que define una dependencia hacia afuera (`EntityRepository`, `Extractor`, `PaymentGateway`). |
| **Adaptador** | Implementación concreta de un puerto (`PostgresEntityRepository`, `LLMExtractor`, `StripePaymentGateway`). |
| **Servicio de aplicación** | Caso de uso que orquesta puertos (`OrderService`, `WalletService`). |
| **Bounded Context** | Sub-paquete del dominio con modelo aislado. No importa otros contexts. |
| **Append-Only** | Tabla que solo crece (INSERT). Sin UPDATE/DELETE. `billing.transactions`, `audit.events`. |
| **PII** | Personally Identifiable Information. Datos personales del usuario: emails, nombres, teléfonos, contenido libre. |
| **Identidad portable** | Identificador global del usuario que sobrevive al cambio de tenant. |
| **Tenant** | Espacio de datos aislado de una organización. Cada organización = un tenant. |
| **RLS** | Row-Level Security en Postgres. Policy que filtra filas según `current_setting('app.tenant_id')`. |
| **PII Handler** | `slog.Handler` que descarta atributos con patrones PII. Guard runtime, no de diseño. |
| **Golden File** | Snapshot de output esperado en `testdata/`. Se regenera manualmente cuando cambia el comportamiento intencionalmente. |
| **Domain Event** | Struct inmutable en `apps/backend/internal/domain/events.go` que representa un hecho de negocio (`ResourceCreated`, `BalanceChanged`). |
| **EventDispatcher** | Puerto que recibe eventos y los entrega a handlers registrados. |
| **Outbox** | Tabla `outbox_events` donde se insertan eventos en la misma transacción que el cambio de estado. Un relay los lee post-commit (marcando `published_at`/`attempts`) y los publica al dispatcher. A diferencia de `billing.transactions`/`audit.events`, **no** es append-only de negocio. |
| **UnitOfWork** | Puerto que encapsula una transacción lógica. El service lo invoca; el adapter (Postgres) lo implementa con BEGIN/COMMIT/ROLLBACK. |
| **In-Memory Event Bus** | Implementación por defecto del EventDispatcher basada en channels de Go con buffer + backpressure. |
| **Testcontainers** | Librería para levantar contenedores Docker efímeros por test suite (Postgres 16, Redis futuro). |
| **Render UI** | Componente HTML+JS que visualiza un OpenAPI spec. Ejemplos: Swagger UI, Redoc, Scalar. |

---

## 8. Documentos autoritativos

Este manifiesto **condensa** pero **nunca contradice** los documentos canónicos. Si una regla está en conflicto entre dos documentos, la jerarquía es:

| Pregunta | Documento autoritativo |
|---|---|
| ¿Cuál es el contrato arquitectónico global? | `docs/ARCHITECTURE.md` |
| ¿Cuál es el dominio del producto? | `docs/PRODUCT_DOMAIN.md` |
| ¿Qué env vars existen? | `docs/ARCHITECTURE.md §5.1` (resumen) + `docs/PRODUCTION_ENV.md` (detalle operativo) |
| ¿Cuándo rotar secretos? | `docs/SECRET_ROTATION.md §1.1` |
| ¿Qué hacer si Postgres está caído? | `docs/RUNBOOK.md` Escenario 5 |
| ¿Cómo se comparte wire format entre Go y TS? | `MANIFEST_MONOREPO.md §6.3` |
| ¿Cómo se renderiza la documentación interactiva del API? | `MANIFEST_MONOREPO.md §6.3` |
| ¿Cómo se orquesta el monorepo? | `MANIFEST_MONOREPO.md §6.2` |
| ¿Qué reglas rigen el frontend? | `MANIFEST_FRONTEND.md` |
| ¿Qué waivers de seguridad son legítimos? | `docs/SECURITY_DEBT.md` |
| ¿Qué bugs están abiertos o cerrados? | `docs/bugs/` |

---

**Versión**: 2.0.0
**Mantenedor**: equipo de arquitectura (Principal Architect).
**Última revisión**: 2026-09-15.
**Próxima revisión**: tras cada release mayor o cuando se añadan 3+ axiomas/anti-patrones/patrones transversales.
