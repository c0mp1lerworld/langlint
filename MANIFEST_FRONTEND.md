# Manifiesto del Frontend (v1.0.0)

> **Guía de arquitectura para la capa frontend en un monorepo polyglot.**
> Este documento establece las reglas, estándares y patrones que rigen la construcción de la aplicación web (TypeScript/Next.js) que consume el backend vía contratos OpenAPI. Es un manifiesto **general y autocontenido**: define *cómo* se construye el frontend, con independencia del dominio de negocio concreto del proyecto. No depende de una línea de producto en particular; aplica a cualquier frontend Next.js dentro del monorepo.
re
> **Audiencia**: desarrolladores frontend, leads técnicos, nuevos devs.
>
> **Prerrequisitos de lectura**: el manifiesto del monorepo (organización de workspaces, orquestación de tasks y contratos compartidos) y, si existe, el documento de contrato del API (`apps/contracts/openapi/`).

---

## Tabla de contenidos

1. [Axiomas de Desarrollo](#1-axiomas-de-desarrollo)
2. [Estándar TypeScript/Next.js y Toolchain](#2-estándar-typescriptnextjs-y-toolchain)
3. [Esqueleto Arquitectónico](#3-esqueleto-arquitectónico)
4. [Anti-patrones](#4-anti-patrones)
5. [Patrones Transversales](#5-patrones-transversales)
6. [Integración y Despliegue en el Monorepo](#6-integración-y-despliegue-en-el-monorepo)
7. [Glosario](#7-glosario)
8. [Documentos autoritativos](#8-documentos-autoritativos)

---

## 1. Axiomas de Desarrollo

Doce principios inquebrantables. Cualquier PR que los viole debe ser rechazado en revisión. No son opiniones: son el precio de mantener un frontend claro, predecible y escalable.

### F1. El Contrato OpenAPI es el Single Source of Truth del Cliente

> Todo tipo de wire (request/response) proviene del contrato OpenAPI y se genera automáticamente. **Nadie edita tipos de wire a mano.**

- **Por qué existe**: backend y frontend comparten un solo contrato. Si el frontend define sus propios tipos a mano, diverge silenciosamente (camelCase vs snake_case, campos faltantes, tipos desalineados) y los bugs aparecen en runtime, no en compilación.
- **Consecuencia de violarlo**: el compilador deja de ser tu red de seguridad. Nulls silenciosos, campos renombrados sin actualizar, contratos que "funcionan" hasta que dejan de hacerlo.
- **Implementación**: `openapi-typescript` genera `src/lib/api/gen.ts` desde `apps/contracts/openapi/api.yaml`. Un cambio de wire se hace primero en el contrato, luego `pnpm generate`, y recién después se usa el tipo en el código de la app.

### F2. Las Dependencias Fluyen Hacia Adentro

> `app/ → features/ → components/ → lib/ → types/`. Las capas externas conocen a las internas; nunca al revés.

```
app/ (rutas y páginas) → features/ (casos de uso) → components/ (UI) → lib/ (cliente, query, store) → types/ (gen.ts)
```

- **Por qué existe**: mantiene los componentes reutilizables y desacoplados de las rutas; mantiene la lógica de datos fuera de los componentes de presentación.
- **Consecuencia de violarlo**: componentes "dios" que saben de todo, ciclos de imports, imposibilidad de testear en aislamiento.
- **Reglas finas**:
  - `types/` no importa a nadie (tipos puros generados).
  - `lib/` importa `types/` (y nada de UI).
  - `components/` importa `lib/` y `types/`; no importa `features/` ni `app/`.
  - `features/` importa `components/`, `lib/`, `types/`.
  - `app/` importa `features/`, `components/`, `lib/`, `types/`.

### F3. Sin Imports Cruzados entre Apps del Monorepo

> El frontend solo se comunica con el backend vía HTTP usando el cliente generado. **Nunca** importa código del backend (ni viceversa).

- **Por qué existe**: frontend y backend son apps separadas en lenguajes distintos. Cualquier acoplamiento de código rompe el build y el modelo de despliegue independiente.
- **Consecuencia de violarlo**: builds rotos, límites de confianza difusos, imposibilidad de desplegar apps por separado.
- **Regla**: si se necesita un tipo del dominio, se consume desde el cliente generado (`gen.ts`), nunca desde un import hacia `apps/backend/`.

### F4. Los Datos Personales se Tratan con Mínimo Privilegio

> Los datos personales (PII) **nunca** se loguean en crudo en el cliente, **nunca** se envían a analytics sin pseudonimizar, y **nunca** se exponen en el bundle más allá de lo estrictamente necesario.

- **Por qué existe**: el navegador es un entorno hostil (DevTools, extensiones, analytics de terceros). Una filtración de PII en el cliente es tan grave como una en el servidor.
- **Consecuencia de violarlo**: exposición de datos personales a terceros (analytics, crash reporting, logs del navegador), incumplimiento regulatorio, daño reputacional.
- **Capas de defensa**:
  - **No loguear** campos sensibles en `console` ni en handlers de error del cliente.
  - **Pseudonimizar** identificadores antes de enviarlos a analytics (hash HMAC con secreto server-only).
  - **Minimizar** qué viaja al cliente: solo lo necesario para renderizar; los datos sensibles se procesan en el servidor (RSC/route handlers) y no se exponen al cliente.

### F5. Los Errores del API se Mapean a UX, No a Consola

> Cada error de dominio del API se traduce a una experiencia de usuario clara. Los detalles técnicos (SQL, stack traces, infraestructura) **jamás** se muestran al usuario.

- **Por qué existe**: el usuario no debe ver `pq: duplicate key...`; debe ver "Ese recurso ya existe". Distintos errores de negocio requieren distintas acciones de UX.
- **Consecuencia de violarlo**: fugas de información, confusión del usuario, errores idempotentes tratados como fallos.
- **Reglas**:
  - Mapear códigos HTTP a mensajes y acciones de UI (404 → "no encontrado", 422 → "revisa los campos", 402 → "saldo insuficiente", etc.).
  - **Idempotencia**: un código de éxito idempotente (p. ej. `409 <recurso>_already_<estado>`) **se trata como éxito**, no como error.
  - Los mensajes de error vienen del contrato (códigos y campos), no se hardcodean textos de negocio en el cliente.

### F6. La Accesibilidad es por Defecto

> Todo componente cumple **WCAG 2.2 AA** desde el primer commit. No es una tarea de "limpieza posterior".

- **Por qué existe**: una interfaz inaccesible excluye usuarios y es un riesgo legal y de producto. La accesibilidad no se puede "añadir" al final sin refactorizar.
- **Consecuencia de violarlo**: rehacer componentes enteros, exclusión de usuarios, incumplimiento normativo.
- **Reglas**: semántica HTML correcta, focus management, contraste, `aria` donde corresponda, navegación por teclado completa, test de accesibilidad en CI (axe-core en tests de componentes).

### F7. Server State y Client State Están Separados

> El **estado del servidor** (datos que vienen del API, con caché y revalidación) vive en **TanStack Query**. El **estado del cliente** (UI, sesión, preferencias locales) vive en **Zustand**. Nunca se mezclan.

- **Por qué existe**: son dos problemas distintos. El estado del servidor necesita cacheo, deduplicación, retries y revalidación; el estado del cliente necesita simplicidad y reactividad local.
- **Consecuencia de violarlo**: caché de API metida en un store global = bugs de staleness, refetchs manuales, invalidación a mano y estado duplicado.
- **Regla**: si un dato proviene del API y puede cambiar en el servidor, es server state (Query). Si es local a la sesión/UI y el servidor no lo conoce, es client state (Zustand).

### F8. Los Formularios se Validan Contra el Contrato

> Los formularios usan **React Hook Form** para el estado y **Zod** para la validación, con schemas derivados de los tipos generados del contrato. No se duplican reglas de validación de negocio en el cliente.

- **Por qué existe**: una sola fuente de verdad para la forma de los datos. La validación de "forma" (tipos, formatos, requeridos) se deriva del contrato; la validación de "negocio" la hace el servidor (que es la autoridad).
- **Consecuencia de violarlo**: reglas duplicadas que divergen, mensajes de validación inconsistentes, doble mantenimiento.
- **Regla**: los schemas Zod se infieren de los tipos OpenAPI para validar la forma; el servidor valida las reglas de negocio y el cliente refleja sus errores (F5).

### F9. Optimistic UI con Retry y Rollback

> Toda mutación optimista (actualizar la UI antes de que confirme el servidor) **debe** tener retry automático y rollback explícito ante fallo.

- **Por qué existe**: la UX optimista es mejor, pero sin retry/rollback produce UI inconsistente con el servidor y estados corruptos invisibles.
- **Consecuencia de violarlo**: el usuario ve un cambio que "desaparece" o "se queda a medias", desconfianza en la app, estados zombies.
- **Regla**: mutación optimista = (1) snapshot del estado previo, (2) aplicar optimistamente, (3) en éxito: confirmar + invalidar queries; (4) en error: rollback + reintentar (con backoff) + notificar.

### F10. Los Tests son un Gate, No un Decorativo

> Pirámide de tres tiers, todos obligatorios en CI, sin `skip()`. Unit (Vitest) → componentes (React Testing Library) → e2e (Playwright).

- **Por qué existe**: un frontend sin tests se pudre. Los tests de componentes atrapan regresiones de UI; los e2e atrapan integraciones rotas con el contrato real.
- **Consecuencia de violarlo**: regresiones que se manifiestan en producción, miedo a refactorizar, "funciona en mi máquina".
- **Convenciones**:
  - Nombres: `describe('Componente', () => { it('escenario → resultado') })`.
  - Unit: lógica pura (helpers, schemas, mappers) con Vitest.
  - Componentes: React Testing Library, comportamiento no implementación.
  - e2e: Playwright contra el contrato (idealmente mockeado o staging), cubriendo flujos críticos.
  - Accesibilidad en tests de componentes (axe-core).

### F11. El Backend es la Fuente de Verdad de las Reglas de Negocio

> El cliente **no reimplementa** lógica de negocio (precios, permisos, cálculos, flujos de estado). Solo refleja lo que el contrato expone.

- **Por qué existe**: duplicar reglas de negocio en el cliente crea dos verdades que inevitablemente divergen. El servidor es la autoridad.
- **Consecuencia de violarlo**: el cliente "decide" algo que el servidor rechaza, o muestra un estado que el servidor no reconoce.
- **Regla**: si una regla puede cambiar por decisión de negocio, pertenece al servidor. El cliente consume estado y expone acciones vía el contrato.

### F12. El Rendimiento es por Construcción

> Se usan React Server Components, streaming, `Suspense` y code-splitting por defecto. Las métricas de Web Vitals son un gate de CI.

- **Por qué existe**: el rendimiento no se "optimiza al final"; se diseña. Un bundle enorme y un render bloqueante son deuda que se paga en conversión y retención.
- **Consecuencia de violarlo**: LCP/INP degradados, malas métricas, refactor de rendimiento doloroso y tardío.
- **Reglas**:
  - Componentes de servidor para lo estático; client components solo donde hace falta interactividad.
  - `Suspense` + `loading.tsx`/`error.tsx` para streaming y límites de error.
  - Code-splitting por ruta (App Router lo hace por defecto) y `dynamic()` para componentes pesados.
  - Web Vitals (LCP, INP, CLS) medidos y como gate en CI.

---

## 2. Estándar TypeScript/Next.js y Toolchain

### 2.1 Stack Tecnológico

| Capa | Tecnología | Versión objetivo | Justificación |
|---|---|---|---|
| Framework | **Next.js** (App Router) | 14+ | RSC, streaming, `Suspense`, rutas por archivos, integración nativa con el monorepo. |
| Lenguaje | **TypeScript** | 5+ (`strict: true`) | Tipado estricto que se apoya en los tipos generados del contrato. |
| Estilos | **Tailwind CSS** | última | Utilidades atómicas, consistencia, árbol de estilos mínimo. |
| Data fetching | **TanStack Query** | última | Server state: caché, deduplicación, retries, invalidación, optimistic updates. |
| Estado global (cliente) | **Zustand** | última | Estado de cliente mínimo, sin boilerplate, selectores finos. |
| Formularios | **React Hook Form** | última | Rendimiento (uncontrolled), mínima re-renderización. |
| Validación | **Zod** | última | Schemas tipados derivables de los tipos OpenAPI. |
| Testing (unit/component) | **Vitest + React Testing Library** | última | Rápido, ergonómico, ESM nativo. |
| Testing (e2e) | **Playwright** | última | Multi-browser, tracing, fixtures robustos. |
| Lint/format | **ESLint + Prettier** | última | Consistencia de estilo y reglas de calidad. |
| Generación de tipos | **openapi-typescript** | última | Tipos TS generados desde el contrato OpenAPI. |
| Accesibilidad en tests | **jest-axe** (o axe-core) | última | Assertions WCAG en tests de componentes. |

### 2.2 Configuración base

```jsonc
// apps/frontend/tsconfig.json (extracto)
{
  "compilerOptions": {
    "strict": true,
    "target": "ES2022",
    "lib": ["dom", "dom.iterable", "esnext"],
    "module": "esnext",
    "moduleResolution": "bundler",
    "jsx": "preserve",
    "paths": { "@/*": ["./src/*"] },
    "noUncheckedIndexedAccess": true,
    "noFallthroughCasesInSwitch": true
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"]
}
```

```js
// apps/frontend/next.config.js
/** @type {import('next').NextConfig} */
module.exports = {
  reactStrictMode: true,
  typedRoutes: true
};
```

> **Nota (Next.js 15.5+)**: `typedRoutes` dejó de ser experimental y se configura en la raíz de `next.config.js`. La ubicación `experimental.typedRoutes` sigue funcionando pero emite un warning de deprecación; en este monorepo se usa la forma estable.

### 2.3 Variables de entorno

Las env vars de cada app del monorepo deben tener un prefijo identificable. Para el frontend:

| Tipo | Prefijo | Ejemplo |
|---|---|---|
| Expuestas al cliente | `NEXT_PUBLIC_` | `NEXT_PUBLIC_API_URL` |
| Internas del frontend (server-only) | prefijo propio del proyecto (p. ej. `FRONTEND_`) | `FRONTEND_API_TOKEN` |

- **Regla**: solo las variables con `NEXT_PUBLIC_` llegan al bundle del navegador. Los secretos (tokens, HMAC keys) viven en variables server-only y **nunca** se exponen con `NEXT_PUBLIC_`.

---

## 3. Esqueleto Arquitectónico

### 3.1 Árbol Canónico

```
apps/frontend/
├── package.json
├── next.config.js
├── tsconfig.json
├── postcss.config.mjs               # Tailwind v4 vía @tailwindcss/postcss (sin tailwind.config)
├── playwright.config.ts
├── vitest.config.ts
├── .eslintrc.json / eslint.config.mjs
├── .prettierrc
│
├── public/                          # assets estáticos
│
└── src/
    ├── app/                         # App Router: rutas, layouts, loading/error/not-found
    │   ├── layout.tsx
    │   ├── page.tsx
    │   └── (routes)/
    │       ├── page.tsx
    │       └── [id]/page.tsx
    │
    ├── components/                   # UI reutilizable (presentacional)
    │   ├── ui/                       # primitivas (Button, Input, Card, ...)
    │   └── feature/                  # componentes de una feature (presentacional)
    │
    ├── features/                     # Casos de uso por dominio (orquestan componentes + lib)
    │   └── <feature>/
    │       ├── components/           # componentes de presentación de la feature
    │       ├── hooks/                # hooks de la feature
    │       └── index.ts              # API pública de la feature
    │
    ├── lib/                          # Infraestructura transversal (no UI)
    │   ├── api/
    │   │   ├── gen.ts                # ⚠️ generado por openapi-typescript (NO editar)
    │   │   └── client.ts             # fetch wrapper con auth + errores (usa gen.ts)
    │   ├── query/                    # TanStack Query: queries y mutaciones
    │   ├── store/                    # Zustand: stores de client state
    │   └── utils/                    # helpers puros
    │
    ├── types/                        # re-exports tipados (gen.ts + derivados)
    │
    └── test/                         # helpers y setup de tests
        ├── setup.ts
        └── mocks/
```

### 3.2 Reglas de Ubicación

| Si estás creando... | Debe vivir en... | NO en... |
|---|---|---|
| Una página o ruta | `src/app/` (App Router) | `components/` ni `features/` |
| Un componente de UI reutilizable | `src/components/ui/` | `app/` ni `features/` |
| Lógica de un caso de uso | `src/features/<feature>/` | `components/ui/` ni `lib/` |
| Una query o mutación del API | `src/lib/query/` | `features/` (salvo wrappers) ni `components/` |
| Un store de client state | `src/lib/store/` | `query/` (no mezclar server/client state) |
| El cliente HTTP generado | `src/lib/api/gen.ts` (regenerable) | nunca editar a mano |
| Un schema Zod de formulario | `src/lib/` o junto a la feature | duplicarlo por página |
| Un hook reutilizable | `src/lib/` o `src/features/<feature>/hooks/` | `components/` |
| Tipos re-exportados | `src/types/` | mezclarlos con lógica |

### 3.3 Capas y Responsabilidades

| Capa | Responsabilidad | NO hace |
|---|---|---|
| `app/` | Composición de rutas, layouts, límites de error/loading. | Lógica de negocio. Fetching directo salvo server components simples. |
| `features/` | Orquestar casos de uso: combinar componentes, hooks, queries y stores. | Definir tipos de wire (usa `gen.ts`). |
| `components/` | UI presentacional, accesible y reutilizable. | Hablar con el API. Conocer rutas o casos de uso. |
| `lib/query/` | Server state: queries, mutaciones, invalidación, optimistic updates. | Guardar estado de UI (eso es `store/`). |
| `lib/store/` | Client state: sesión, preferencias, UI efímera. | Cachear datos del API. |
| `lib/api/` | Cliente HTTP tipado desde `gen.ts`, mapeo de errores. | Lógica de negocio. |
| `types/` | Tipos generados y derivados. | Importar código de runtime. |

---

## 4. Anti-patrones

Errores conocidos que se deben evitar. Si descubres uno nuevo, documéntalo con el mismo formato.

### AP-F1. Definir tipos de wire a mano

- **Qué pasó**: el dev define `interface User { ... }` a mano en `src/types/` en vez de usar `gen.ts`. El backend renombra un campo; el frontend queda con nulls silenciosos.
- **Lección**: viola F1. El wire format vive en el contrato y se genera.
- **Regla**: solo se importan tipos de wire desde `gen.ts`. Cualquier tipo local es de UI o de derivación, nunca duplica el contrato.

### AP-F2. Importar código del backend (o al revés)

- **Qué pasó**: `import type { User } from '../../apps/backend/...'` desde un componente. Build de TS roto.
- **Lección**: viola F3. Las apps se comunican solo por HTTP.
- **Regla**: todo lo que viene del backend llega por el cliente generado.

### AP-F3. `useEffect` para fetching

- **Qué pasó**: `useEffect(() => { fetch(...).then(setState) }, [])` en un componente. Race conditions, refetchs duplicados, sin caché, sin cancelación.
- **Lección**: el fetching es server state (F7). Usar TanStack Query (o server components).
- **Regla**: `useEffect` es para efectos (suscripciones, timers, side-effects), no para cargar datos del API.

### AP-F4. Mezclar estado del API en un store global

- **Qué pasó**: el dev mete la lista de recursos en Zustand y la sincroniza "a mano" con el servidor. Staleness, invalidación manual, bugs de doble fuente de verdad.
- **Lección**: viola F7. El server state lo gestiona TanStack Query.
- **Regla**: si viene del API y puede cambiar en el servidor → Query. Si es local y efímero → Zustand.

### AP-F5. Optimistic UI sin rollback

- **Qué pasó**: se actualiza la lista optimistamente, la petición falla, y la UI queda mostrando un cambio inexistente.
- **Lección**: viola F9. Toda mutación optimista necesita snapshot + rollback + retry.
- **Regla**: ver §5.2 para el patrón completo.

### AP-F6. Tratar un error idempotente como fallo

- **Qué pasó**: el backend responde `409 <recurso>_already_<estado>` en un retry; el frontend muestra "error" al usuario aunque la operación ya tuvo éxito.
- **Lección**: viola F5. Los códigos de éxito idempotente se documentan en el contrato y el cliente los trata como éxito.
- **Regla**: mapear explícitamente los códigos idempotentes a "éxito" en el cliente.

### AP-F7. Loggear PII en el navegador

- **Qué pasó**: `console.log(usuario)` con datos personales, o enviar el payload completo a analytics. Filtración de PII a terceros.
- **Lección**: viola F4. El navegador es hostil; no se loguean datos personales en crudo.
- **Regla**: pseudonimizar antes de analytics; no loguear campos sensibles; minimizar lo que viaja al cliente.

### AP-F8. Variables de entorno sin prefijo

- **Qué pasó**: `API_URL` definida sin `NEXT_PUBLIC_` esperando que llegue al navegador; no llega.
- **Lección**: viola §2.3. Solo `NEXT_PUBLIC_` se expone al cliente.
- **Regla**: prefijos explícitos por app; secretos siempre server-only.

### AP-F9. Duplicar reglas de negocio en el cliente

- **Qué pasó**: el frontend calcula precios o permisos localmente; el backend cambia la regla y el cliente muestra datos incorrectos.
- **Lección**: viola F11. El servidor es la autoridad.
- **Regla**: el cliente consume estado y expone acciones; no decide reglas de negocio.

### AP-F10. Componentes "dios" con toda la lógica

- **Qué pasó**: un componente de 800 líneas que hace fetching, estado, validación y render. Imposible de testear y reutilizar.
- **Lección**: viola F2. Separar presentación (components) de orquestación (features) y datos (lib).
- **Regla**: cada capa una responsabilidad; extraer hooks y componentes presentacionales.

---

## 5. Patrones Transversales

### 5.1 Data Fetching con TanStack Query + Cliente Generado

**Problema**: F1 exige tipos del contrato y F7 exige separar server state. ¿Cómo se combinan?

**Decisión canónica**: un cliente HTTP tipado (`lib/api/client.ts`) que usa los tipos de `gen.ts`, envuelto por TanStack Query en `lib/query/`.

#### Componentes

| Componente | Ubicación | Responsabilidad |
|---|---|---|
| `gen.ts` | `lib/api/gen.ts` | Tipos y paths generados desde OpenAPI. |
| `client.ts` | `lib/api/client.ts` | `fetch` wrapper con auth, base URL y mapeo de errores. |
| `queries/` | `lib/query/` | `useQuery`/`useMutation` tipados por feature o recurso. |
| `queryClient` | `lib/query/query-client.ts` | Instancia de `QueryClient` con defaults (retries, staleTime, gcTime). |

#### Reglas

1. Las keys de query son estables y serializables (`['recurso', id]`).
2. Las mutaciones invalidan las queries afectadas (`queryClient.invalidateQueries`).
3. `staleTime`/`gcTime` se configuran por defecto según el tipo de dato (configurable, no hardcodeado por llamada).
4. Los errores del API se capturan en el cliente y se traducen a errores de dominio tipados (F5).

### 5.2 Mutaciones Optimistas con Retry y Rollback

**Decisión canónica**: patrón snapshot/apply/rollback.

```
[1] Snapshot: onMutate → cancelar queries en vuelo + guardar estado previo.
[2] Apply: setQueryData con el cambio optimista.
[3] Success: invalidar queries afectadas.
[4] Error: rollback (setQueryData con snapshot) + retry con backoff + notificar.
```

#### Reglas

1. Toda mutación optimista implementa `onMutate` (snapshot), `onError` (rollback) y `onSettled` (invalidación).
2. El retry tiene backoff y un límite de intentos.
3. Si el error es idempotente (F5/AP-F6), se trata como éxito y no se hace rollback.
4. El snapshot solo captura las queries afectadas, no todo el store.

### 5.3 Mapeo de Errores de Dominio → UI

**Problema**: el API devuelve códigos y campos (F5). ¿Cómo se traduce a UX?

**Decisión canónica**: un mapper centralizado en `lib/api/errors.ts` que convierte respuestas del API en tipos de error de UI, y componentes de error que los renderizan.

#### Reglas

1. El `client.ts` normaliza los errores HTTP a una estructura tipada (`code`, `message`, `fields`).
2. Un mapper traduce `code` a mensaje/acción de UI (o delega a un diccionario de mensajes).
3. Los códigos idempotentes se marcan como `isIdempotentSuccess` y se tratan como éxito.
4. Nunca se muestra al usuario el mensaje crudo de infraestructura (stack, SQL, etc.).

### 5.4 Autenticación y Sesión

**Decisión canónica**: la sesión es **client state** (Zustand) + **server state** (la identidad del usuario, si viene del API).

- Token de sesión en una cookie `httpOnly` (server-only) cuando es posible; si es un SPA puro, en memoria con refresh vía `NEXT_PUBLIC_` solo para la URL del API.
- El `client.ts` inyecta el header de autorización desde el store de sesión.
- Un guard de rutas (middleware o layout) redirige cuando no hay sesión.
- El logout limpia el store de sesión **y** invalida todo el `QueryClient`.

### 5.5 Generación de Código desde el Contrato

**Regla operativa** (coherente con el monorepo):

1. Cambio en el wire format → primero `apps/contracts/openapi/api.yaml`.
2. `pnpm generate` → `openapi-typescript` escribe `src/lib/api/gen.ts`.
3. `gen.ts` está **commiteado** para builds offline; no se edita a mano.
4. CI verifica que el `gen.ts` commiteado coincida con el contrato (sin drift).

### 5.6 Documentación Interactiva del API

- La UI de docs (Swagger UI / Scalar) se sirve desde el frontend en producción (no desde el backend), con control de acceso vía sesión.
- En desarrollo, se puede montar localmente apuntando al `api.yaml`.
- Los endpoints marcados como internos (`x-internal: true`) se filtran del render.

---

## 6. Integración y Despliegue en el Monorepo

### 6.1 Tasks del Frontend (orquestadas por el monorepo)

| Task | Comando | Notas |
|---|---|---|
| `build` | `next build` | Depende de `generate` (tipos actualizados). |
| `lint` | `eslint .` | Reglas de estilo + reglas custom. |
| `typecheck` | `tsc --noEmit` | Gate estricto de tipos. |
| `test` | `vitest run` | Unit + componentes. |
| `test:e2e` | `playwright test` | e2e contra staging o contrato mockeado. |
| `generate` | `openapi-typescript .../api.yaml -o src/lib/api/gen.ts` | Regenera tipos desde el contrato. |
| `dev` | `next dev` | Desarrollo local. |

### 6.2 Dependencias entre apps

El frontend depende de `apps/contracts/` para generar tipos. En el orquestador:

```jsonc
// turbo.json (extracto)
{
  "tasks": {
    "build": { "dependsOn": ["^build"] }   // contracts se valida antes que frontend
  }
}
```

### 6.3 Versionado del Contrato

- Un cambio **breaking** en el contrato → bump major de `apps/contracts/` + entrada BREAKING en `CHANGELOG.md` + notificación al equipo.
- Un cambio **no-breaking** → bump minor.
- El frontend regenera (`pnpm generate`) y actualiza su dependencia en el mismo PR.

### 6.4 Convenciones de Despliegue

- El frontend se despliega como una app independiente dentro del monorepo (build de Next.js a un artefacto estático o un runtime serverless).
- Las env vars `NEXT_PUBLIC_` se inyectan en build time; las server-only en runtime.
- Los secretos nunca se exponen al navegador.

---

## 7. Glosario

| Término | Significado |
|---|---|
| **Monorepo** | Repositorio único con múltiples apps (backend, frontend, contracts). |
| **App Router** | Sistema de enrutamiento de Next.js basado en archivos (`src/app/`). |
| **RSC (React Server Components)** | Componentes renderizados en el servidor, sin JS en el cliente. |
| **Streaming** | Envío progresivo de HTML/UI al cliente conforme se resuelve. |
| **Suspense** | Límite declarativo para contenido asíncrono en React. |
| **Server State** | Datos que provienen del servidor y se cachean/revalidan (TanStack Query). |
| **Client State** | Estado local de UI/sesión que el servidor no conoce (Zustand). |
| **gen.ts** | Archivo de tipos generado por `openapi-typescript`. No editable a mano. |
| **Optimistic UI** | Actualizar la UI antes de que el servidor confirme, con rollback. |
| **SSOT (Single Source of Truth)** | Fuente única de verdad; en el frontend, el contrato OpenAPI. |
| **Web Vitals** | Métricas de experiencia: LCP, INP, CLS. |
| **WCAG 2.2 AA** | Estándar de accesibilidad web. |
| **PII** | Información personal identificable. |
| **Pseudonimización** | Sustituir identificadores por un hash/HMAC antes de analytics. |
| **Idempotencia** | Una operación repetida produce el mismo resultado sin efectos adversos. |

---

## 8. Documentos autoritativos

Este manifiesto **condensa** pero **nunca contradice** los documentos canónicos. Si hay conflicto, la jerarquía es:

| Pregunta | Documento autoritativo |
|---|---|
| ¿Cómo se orquesta el monorepo (workspaces, turbo)? | Manifiesto del monorepo (organización y orquestación). |
| ¿Cuál es el contrato HTTP (SSOT del wire)? | `apps/contracts/openapi/api.yaml`. |
| ¿Cómo se genera el cliente tipado? | `MANIFEST_FRONTEND.md §5.5`. |
| ¿Qué reglas de negocio aplican? | El backend (contrato) — el frontend no las reimplementa (F11). |
| ¿Qué env vars usa el frontend? | `MANIFEST_FRONTEND.md §2.3`. |
| ¿Qué bugs de frontend están abiertos/cerrados? | Registro de bugs del proyecto. |

---

**Versión**: 1.0.0
**Mantenedor**: equipo frontend (Frontend Lead).
**Última revisión**: 2026-09-15.
**Próxima revisión**: tras cada release mayor o cuando se añadan 3+ axiomas/anti-patrones/patrones.
