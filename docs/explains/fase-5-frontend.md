# Fase 5 — Frontend (Next.js): explicación didáctica

> Documento de estudio sobre todo lo construido en la **Fase 5 (Frontend)** del proyecto LangLint.
> Cubre los bloques `5.0` (cableado HTTP del backend), `5.1` (base Next.js), `5.2` (cliente, TanStack Query, Zustand y Zod), `5.3` (vista diff de 3 columnas y formulario), `5.4` (dashboard de analíticas) y `5.5` (tests).
> Es la continuación de `docs/explains/fase-4-motor-ia.md`.

---

## 1. ¿De qué trata esto? (Explicación para no técnicos)

Hasta la Fase 4 ya teníamos la **cocina** funcionando: el sistema sabe guardar los ejercicios, pedirle a la IA que los corrija y apuntar los errores más frecuentes. Pero una cocina sin **comedor** no le sirve a nadie. La Fase 5 es exactamente eso: **abrir el restaurante al público** montando la sala, los camareros, la carta y el panel de control del jefe.

La imagen es la de un restaurante que pasa de "funcionar por dentro" a "atender clientes":

| Imagen de la vida real | Qué representa en el proyecto |
|---|---|
| **La cocina y los almacenes** | El **backend** (Go): guarda datos, llama a la IA, calcula estadísticas. Ya estaba hecho en las fases 1–4. |
| **La sala y las mesas** | El **frontend** (Next.js): las pantallas que ve el estudiante en el navegador. |
| **El camarero que toma el pedido** | El **cliente HTTP** (`ApiClient`): traduce lo que el estudiante toca en la pantalla a "pedidos" que la cocina entiende, y viceversa. |
| **La carta con los platos exactos** | El **contrato OpenAPI** (`gen.ts`): la lista fija y única de todo lo que se puede pedir y de cómo debe venir servido. Si la cocina cambia un plato, la carta se regenera sola. |
| **La libreta del camarero (pedidos en curso)** | **TanStack Query**: recuerda qué se pidió, qué está cocinándose y cuándo hay que volver a preguntar. |
| **Los post-it pegados en la barra** | **Zustand**: anotaciones rápidas del camarero que no pertenecen a la cocina (p. ej. qué pestaña del panel está abierta). |
| **El corrector de la carta** | **Zod**: comprueba que lo que escribe el estudiante en un formulario tenga la forma correcta antes de mandarlo. |
| **El espejo de tres columnas** | La **vista diff**: muestra lado a lado el texto en español, lo que escribió el alumno y la corrección de la IA, con lo malo tachado y lo bueno resaltado. |
| **El panel del jefe con las estadísticas** | El **dashboard de analíticas**: qué errores se repiten más y cuándo. |
| **El inspector de sanidad** | Los **tests** (Vitest, React Testing Library + axe, Playwright): revisan que cada rincón funcione y que sea accesible para todos. |

**El viaje del estudiante, contado como anécdota:**

1. Entra al local y ve la lista de sus ejercicios (la **home**).
2. Pulsa "Nueva práctica" y rellena un **formulario** (texto en español, su borrador en inglés y la regla que quiere practicar). El camarero (Zod) revisa que no falte nada antes de pasar el pedido.
3. La cocina crea el ejercicio y el camarero le lleva la pantalla del detalle.
4. El estudiante pulsa "Analizar". Mientras la IA trabaja, la pantalla dice "Analizando…" y **se refresca sola** cada poco (sin que nadie recargue la página), hasta que llega la corrección.
5. Entonces ve el **espejo de tres columnas**: qué escribió, qué debería haber escrito y, desplegando cada tarjeta, la explicación gramatical y el tipo de error cometido.
6. En la pestaña "Analíticas" consulta su **panel del jefe**: qué errores comete más a menudo.

**Resultado de la Fase 5**: por primera vez el sistema es **usable por una persona** de principio a fin, con pantallas reales, datos que se cargan y actualizan solos, y una red de seguridad de pruebas que demuestra que todo funciona. Es el cierre del MVP desde el punto de vista del usuario.

---

## 2. Guía de Estudio (Para entender qué hizo la IA)

Vamos por bloques: **qué se creó**, **cómo funciona** y **por qué** se hizo así. La idea es que puedas abrir el código y reconocer cada decisión.

### Bloque 0 — El backend se pone al teléfono (`5.0`)

El frontend no puede hablar con el dominio directamente: necesita un **teléfono** (una API HTTP). En la Fase 5.0 se cerró ese cableado que faltaba:

```text
   Navegador (Next.js) ──HTTP/JSON──► cmd/api (Go) ──► services ──► domain
```

- **`cmd/api`**: el programa que arranca el servidor. Usa **Uber Fx** para "enchufar" todas las piezas (config, base de datos, servicios) automáticamente al arrancar.
- **Handlers chi**: cada ruta (`GET /practices`, `POST /practices`, `POST /analyze`…) tiene una función que recibe la petición, llama al servicio y devuelve JSON.
- **`handlers/errors.go`**: un traductor que convierte los errores del dominio (los de la Fase 2) en respuestas HTTP con su `code` y su `message` (F5: el error correcto, no detalles internos).

En el MVP no hay login: hay **un único usuario** fijo que viene de la variable `APP_USER_ID`. Las rutas de privacidad (`/me/*`) y `/analytics/progress` se dejan "en obra" devolviendo `501 not_implemented` hasta la Fase 6.

> **Concepto — API HTTP**: la "ventanilla" por la que el navegador pide y recibe datos, en formato JSON.
>
> **Concepto — Router**: la centralita que mira la URL y el método y decide qué función atiende cada petición.
>
> **Concepto — Inyección de dependencias (Uber Fx)**: en lugar de que cada pieza se construya a sí misma, un "montador" las construye y las conecta en el arranque.

### Bloque 1 — La base: Next.js, TypeScript y Tailwind (`5.1`)

Se creó la app `apps/frontend/` con **Next.js 15** (App Router), **React 19**, **TypeScript en modo estricto** y **Tailwind v4**.

- **Next.js (App Router)**: cada carpeta dentro de `src/app/` es una ruta. `src/app/practices/[id]/page.tsx` crea automáticamente la página de detalle `/practices/{id}`.
- **TypeScript estricto** (`strict: true`, `noUncheckedIndexedAccess`): el compilador te avisa **antes de ejecutar** si un campo podría no existir. Es la red de seguridad contra los *null* silenciosos.
- **Tailwind**: clases cortas de estilos (`text-gray-900`, `rounded-md`) escritas directamente en el HTML, sin hojas de estilo a mano.

Estructura de capas (axioma **F2**, de fuera hacia dentro):

```text
app/ (páginas) → features/ (casos de uso) → components/ (UI) → lib/ (cliente, query, store) → types/ (gen.ts)
```

La regla es unidireccional: una página conoce las features, pero una feature **nunca** importa de una página. Esto evita los "componentes dios" y los ciclos.

> **Concepto — Monorepo**: un único repositorio con varias apps (`contracts`, `backend`, `frontend`) que comparten herramientas.
>
> **Concepto — App Router**: sistema de rutas de Next.js basado en carpetas de archivos.
>
> **Concepto — `strict`**: el modo más exigente del compilador de TypeScript; caza más errores a cambio de exigir más rigor.

### Bloque 2 — La carta única: el contrato genera los tipos (`5.2.1`)

El axioma **F1** dice: *todo tipo de datos que viaja por la red nace del contrato OpenAPI*. Nadie escribe tipos a mano.

```text
apps/contracts/openapi/api.yaml  ──openapi-typescript──►  apps/frontend/src/lib/api/gen.ts
```

- `gen.ts` es **generado y no se edita a mano** (regla `contract_drift`). Si el backend cambia un campo, se edita `api.yaml`, se lanza `pnpm generate` y todo el frontend queda alineado.
- Por eso, si el compilador dice "el tipo no tiene ese campo", es que **el contrato no lo tiene**: no se fuerza con un `as any`.

> **Concepto — Single Source of Truth (SSOT)**: una única fuente de la verdad. Aquí, el `api.yaml`; los tipos de Go y de TypeScript se derivan de él.
>
> **Concepto — Generación de código**: producir ficheros a partir de una especificación, para que código y contrato no se desincronicen.

### Bloque 3 — El camarero: `ApiClient` y el mapeo de errores (`5.2.2`, `5.2.3`)

`lib/api/client.ts` es un envoltorio del `fetch` nativo:

```ts
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
export const apiClient = new ApiClient({ baseUrl: API_BASE_URL });

// uso:
apiClient.get<PracticeList>("/practices");
apiClient.post<Practice>("/practices", body);
apiClient.get<PracticeDetail>(buildPath("/practices/{practiceId}", { practiceId: id }));
```

Qué aporta frente a usar `fetch` a pelo:

1. **Tipado**: `apiClient.get<PracticeList>(...)` te devuelve un objeto con la forma exacta del contrato.
2. **`buildPath`**: sustituye `{practiceId}` por su valor y lo codifica.
3. **Query params y headers**: los monta por ti (auth, `Content-Type`).
4. **Mapeo de errores** (`errors.ts`): convierte cualquier respuesta de error en un `ApiError{status, code}` limpio, sin filtrar stack traces ni SQL (**F5**).

Y una joya de diseño: **la idempotencia** (anti-patrón AP-F6). Si el análisis ya está en curso, el backend responde `409 analysis_pending`. Eso **no es un error**, es "ya lo estás haciendo". El cliente lo reconoce y lo trata como éxito:

```ts
export function isIdempotentSuccess(status: number, code?: ErrorCode): boolean {
  return status === 409 && code !== undefined && IDEMPOTENT_SUCCESS_CODES.has(code);
}
```

> **Concepto — `fetch`**: la función del navegador para hacer peticiones HTTP.
>
> **Concepto — Idempotencia (aquí)**: repetir una acción no rompe nada; el `409 analysis_pending` se interpreta como "éxito", no como fallo.
>
> **Concepto — Mapeo de errores**: traducir un código/estado técnico a una experiencia de usuario clara.

### Bloque 4 — La libreta y los post-it: server state vs client state (`5.2.4`, `5.2.5`)

El axioma **F7** separa **dos tipos de estado** que se confunden a menudo:

| Tipo | ¿Qué es? | ¿Dónde vive? |
|---|---|---|
| **Server state** | Datos que vienen del API y pueden cambiar en el servidor (la lista de prácticas, el análisis). | **TanStack Query** (`lib/query/`) |
| **Client state** | Cosas locales de la pantalla que el servidor no conoce (pestaña abierta, ventana elegida). | **Zustand** (`lib/store/`) |

**TanStack Query** aporta, sin escribir nada a mano: caché, deduplicación de peticiones, reintentos, invalidación y el refresco automático. Su configuración base está en `query-client.ts`:

```ts
return new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, gcTime: 5 * 60_000, refetchOnWindowFocus: false, retry: 1 },
    mutations: { retry: 0 },
  },
});
```

**Zustand** es un "almacén" mínimo para lo efímero. La regla de oro: **nunca** meter datos del API en Zustand (AP-F4), porque acabarías con dos verdades y cachés a mano.

> **Concepto — Server state**: datos cuya fuente de verdad es el servidor; se cachean y revalidan.
>
> **Concepto — Client state**: estado local de la interfaz, sin vida en el servidor.
>
> **Concepto — `staleTime` / `gcTime`**: cuánto tiempo un dato se considera "fresco" y cuánto se guarda en caché antes de tirarlo.

### Bloque 5 — El corrector de la carta: Zod + React Hook Form (`5.2.6`, `5.3.5`)

Los formularios usan **React Hook Form** (gestiona el estado) + **Zod** (valida la forma), axioma **F8**.

```ts
export const createPracticeSchema = z.object({
  source_text: z.string().min(1, "El texto en español es obligatorio"),
  draft_text: z.string().min(1, "El borrador en inglés es obligatorio"),
  target_rules: z.array(targetRuleSchema).min(1, "Añade al menos una regla objetivo"),
});
```

Los mensajes están en español y el schema se deriva del tipo del contrato con una **aserción de compilación**:

```ts
type CreatePracticeContract = AssertAssignable<CreatePracticeInput, Schemas["CreatePracticeRequest"]>;
```

Es decir: si mañana el contrato cambia `source_text` a otro nombre, **el proyecto deja de compilar**, avisándote de que el formulario está desalineado.

Además, en lugar de añadir la librería `@hookform/resolvers`, se escribió un **resolver propio** (`lib/utils/zod-rhf.ts`) que traduce los errores de Zod al formato que React Hook Form espera. Menos dependencias, más control.

> **Concepto — Validación**: comprobar que la entrada tiene la forma correcta **antes** de enviarla. La validación de "negocio" la hace el servidor (F11); el cliente solo valida la "forma".
>
> **Concepto — Resolver**: puente que adapta los errores de una librería (Zod) al formato de otra (React Hook Form).
>
> **Concepto — `AssertAssignable`**: tipo que solo compila si `A` es asignable a `B`; una "alarma" en tiempo de compilación.

### Bloque 6 — El espejo de tres columnas: word-diff (`5.3.1`, `5.3.2`)

La pieza estrella es la **vista diff**: tres columnas por fragmento (Español | tu borrador | corrección IA), con lo borrado tachado en rojo y lo corregido resaltado en verde.

El corazón es `lib/utils/diff.ts`, que compara **palabra a palabra** usando **LCS** (*longest common subsequence*):

```ts
export function diffWords(before: string, after: string): DiffToken[] {
  // 1) parte en palabras
  // 2) calcula la subsecuencia común más larga (LCS)
  // 3) recorre y emite tokens "equal" | "del" | "ins"
}
```

**¿Por qué escribir el diff a mano y no usar una librería?** Por dos motivos: (1) es **lógica pura de presentación**, perfecta para testear sin dependencias; (2) el manifiesto prioriza pocas dependencias. De hecho, un test unitario destapó un bug real de *backtracking* antes del commit.

Cada `FragmentDiff` es una tarjeta expandible con la explicación (`target_verb_review`, `lexical_clarification`, `grammar_explanation`) y los badges del tipo de error.

> **Concepto — Diff**: comparación visual entre "antes" y "después" resaltando lo que cambió.
>
> **Concepto — LCS**: algoritmo clásico que encuentra la secuencia más larga de elementos comunes entre dos listas; es la base de los diffs de texto.
>
> **Concepto — Token**: cada unidad comparada (aquí, una palabra con su espacio).

### Bloque 7 — La pantalla se refresca sola: polling y optimistic update (`5.3.3`)

El análisis es **asíncrono** (la IA tarda). El frontend lo gestiona con dos técnicas de TanStack Query:

1. **Polling**: mientras la práctica esté en `analyzing`, la query se re-ejecuta sola cada 2 segundos:

```ts
refetchInterval: (query) =>
  query.state.data?.status === "analyzing" ? POLL_INTERVAL_MS : false,
```

2. **Optimistic update con rollback** (axioma **F9**): al pulsar "Analizar", la UI cambia a "Analizando…" **antes** de que el servidor confirme; si algo falla, se revierte al estado anterior:

```ts
onMutate: async () => { /* guarda el estado previo y pone "analyzing" */ },
onError: (error, _vars, context) => { /* si no es "analysis_pending", restaura el estado previo */ },
onSettled: () => { /* invalida para recargar la verdad del servidor */ },
```

**¿Por qué importa el rollback?** Porque una UI optimista sin red de seguridad deja al usuario viendo un cambio que "desaparece" o se queda a medias. Y el caso `409 analysis_pending` se respeta como éxito idempotente (Bloque 3).

> **Concepto — Polling**: preguntar periódicamente hasta que el estado cambie.
>
> **Concepto — Optimistic update**: aplicar el cambio en la UI antes de la confirmación del servidor, y revertirlo si falla.
>
> **Concepto — Rollback**: restaurar el estado anterior ante un fallo.

### Bloque 8 — Accesible por defecto (`5.3.4`, F6)

Cada componente cumple **WCAG 2.2 AA** desde el primer commit: semántica (`section`, `h1`–`h3`, `ul`), botones nativos, `aria-expanded`/`aria-controls` en los paneles, `role="status"`/`role="alert"` para anunciar cambios, tooltips con `role="tooltip"` y `aria-describedby`, y **doble señal** en el diff (color **y** tachado/negrita) para no depender solo del color.

Un detalle fino que capturó el test de accesibilidad: los encabezados de columna del diff eran `h3` bajo un `h1`, saltándose un nivel (`h1 → h3`). Se corrigió a `h2` porque axe (`heading-order`) lo marcó.

> **Concepto — WCAG**: estándar internacional de accesibilidad web.
>
> **Concepto — `aria-*`**: atributos que describen a los lectores de pantalla lo que la vista no puede.
>
> **Concepto — Doble cue (doble señal)**: comunicar algo por dos vías (color + forma) para que nadie se quede fuera.

### Bloque 9 — El panel del jefe: dashboard de analíticas (`5.4`)

El dashboard (`/analytics`) consume `GET /analytics/error-patterns` y muestra:

- **Alerta del error recurrente**: el patrón con más `count` ("Tu error más frecuente es …").
- **Frecuencia por patrón**: barras horizontales con etiqueta, veces y última fecha.
- **Selector de ventana** (`day`/`week`/`month`) guardado en **Zustand** (client state, F7).
- **Progreso en obra**: `/analytics/progress` responde `501 not_implemented` (Fase 6); el dashboard lo muestra con un *placeholder* elegante.

Regla **F11** respetada: el cliente **no recalcula** la clasificación de errores. "El más frecuente" es solo elegir el máximo `count` de datos que el backend ya agregó; los *labels* (código → texto) son presentación, no reglas.

Durante la **verificación manual en navegador** de este bloque se destaparon dos bugs reales de cableado que conviene recordar:

1. **CORS**: el navegador (`:3000`) no podía leer las respuestas del backend (`:8080`). Se añadió un middleware configurable (`httpx.CORS` + `APP_CORS_ALLOWED_ORIGINS`).
2. **`fetch` desvinculado**: guardar `this.fetchImpl = fetch` y llamarlo como `this.fetchImpl(...)` producía `TypeError: Illegal invocation` (el `this` del `fetch` nativo debe ser el `window`). Se arregló con `(input, init) => fetch(input, init)`.

> **Concepto — CORS**: mecanismo del navegador que autoriza a una web a leer respuestas de otro origen (otro puerto/dominio).
>
> **Concepto — Origen (origin)**: combinación de protocolo + host + puerto. `localhost:3000` y `localhost:8080` son orígenes distintos.
>
> **Concepto — Placeholder graceful**: mostrar un "próximamente" limpio en vez de un error crudo cuando una función aún no existe.

### Bloque 10 — El inspector de sanidad: los tests (`5.5`)

El bloque de cierre `5.5` monta la **pirámide de tres niveles** del axioma **F10**:

| Nivel | Herramienta | Qué cubre | Ejemplo |
|---|---|---|---|
| **Unit** | **Vitest** | Lógica pura | `diffWords`, `mostFrequentPattern`, schemas Zod, `ApiClient`, `errors`. |
| **Componente** | **React Testing Library + axe-core** | Comportamiento y accesibilidad | `FragmentDiff`, `PracticeForm`, `AnalyticsDashboard`, con `axe.run()` para WCAG. |
| **e2e** | **Playwright** | Flujo completo en un navegador real | crear práctica → analizar → diff. |

Detalles que merecen atención:

- **`ApiClient` se testea con `fetch` inyectado** (`fetchImpl`), sin red: se comprueba la URL, los headers, los query params y el mapeo de errores. (Este test habría atrapado el bug del `fetch` desvinculado del Bloque 9.)
- **Los tests de componente mockean los hooks de query** (`vi.mock`), no el HTTP: así se aísla la UI del dato.
- **`axe-core` directo, no `jest-axe`**: `jest-axe@11` no publica tipos TypeScript; la checklist pide "axe-core", así que se usa `axe.run()` y se verifica `violations == []`.
- **El e2e no necesita Postgres ni OpenAI**: `playwright.config.ts` levanta `next dev` en un puerto propio con la API **en el mismo origen**, y `page.route` **mockea** el backend (crear → analizar → detalle completado). Es la interpretación del manifiesto F10: *"e2e idealmente mockeado o contra staging"*.
- **Sin `skip()` ni `only()`** en ningún nivel (5.5.4).

> **Concepto — Mock**: imitación controlada de una dependencia para probar sin efectos reales.
>
> **Concepto — `axe`**: herramienta que inspecciona el HTML y reporta violaciones de accesibilidad.
>
> **Concepto — e2e (end-to-end)**: prueba que recorre el sistema completo como lo haría un usuario real.

---

## 3. Resumen de los cambios

- **Base y capa de datos (`5.0`–`5.2`)**: se cableó la API HTTP del backend (handlers chi + DI con Fx + mapeo de errores) y se levantó la app Next.js 15 con TypeScript estricto y Tailwind; se añadió el cliente tipado `ApiClient` (con idempotencia del `409 analysis_pending`), el manejo de errores→UX, TanStack Query (server state), Zustand (client state) y los schemas Zod derivados del contrato.
- **Interfaz del usuario (`5.3`–`5.4`)**: vista diff de 3 columnas con word-diff propio (LCS), formulario de creación (React Hook Form + Zod), polling + optimistic update con rollback, accesibilidad WCAG AA, y dashboard de analíticas (frecuencia, error recurrente y placeholder de progreso), más el fix de CORS configurable y del `fetch` desvinculado detectados en la verificación en navegador.
- **Suite de pruebas (`5.5`)**: Vitest (unit), React Testing Library + axe-core (componentes y accesibilidad) y Playwright (e2e con backend mockeado vía `page.route`); scripts `test`/`test:e2e` integrados en el monorepo y **Gate de salida de Fase 5 en verde** (typecheck, lint, test, test:e2e y build).
