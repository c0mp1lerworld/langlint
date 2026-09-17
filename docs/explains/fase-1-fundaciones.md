# Fase 1 — Fundaciones: explicación didáctica

> Documento de estudio sobre todo lo construido en la **Fase 1 (Fundaciones)** del proyecto LangLint.
> Cubre los bloques `1.1` (monorepo base), `1.2` (árbol de directorios), `1.3` (contrato OpenAPI) y `1.4` (pipeline de generación).

---

## 1. ¿De qué trata esto? (Explicación para no técnicos)

Imagina que quieres construir una casa con **dos equipos de obra** que hablan idiomas distintos: un equipo que hace la fontanería (el "backend", que piensa y procesa) y otro que hace la decoración y la cara visible de la casa (el "frontend", lo que el usuario ve y toca). Para que la casa funcione, ambos equipos tienen que conectarse en los mismos puntos: la tubería del baño tiene que medir exactamente lo mismo que el agujero que deja el otro equipo.

El problema clásico es que, si cada equipo decide por su cuenta cómo son las conexiones, cuando llegan a la obra **no encajan**: uno hizo un tubo de 2 cm y el otro un agujero de 3 cm. Y el error se descubre tarde y caro.

**¿Qué hace esta Fase 1?** Antes de poner un solo ladrillo, monta tres cosas:

1. **El terreno y las oficinas** (el "monorepo"): una sola nave donde trabajan los dos equipos, con las herramientas y la organización ya puestas.
2. **Un plano oficial de las conexiones** (el "contrato" o `api.yaml`): un único documento donde se define, de forma exacta, qué información se va a intercambiar y con qué forma (nombres, tipos, cuántos campos...). Este plano es la **única fuente de la verdad**: si algo no está aquí, no existe.
3. **Una imprenta automática** (el "pipeline de generación"): una máquina que, leyendo el plano, fabrica automáticamente los "formularios" que cada equipo necesita, cada uno en su idioma. Así el equipo de fontanería recibe los formularios en su idioma y el de decoración en el suyo, pero **ambos hablan de lo mismo** porque salen del mismo plano.

Traduciendo a la vida real: en vez de que un equipo traduzca a mano un documento para el otro (y se equivoque), existe **un único documento maestro** y una **fotocopiadora** que saca copias perfectas adaptadas a cada idioma. Si un día cambia un dato del plano, se pulsa un botón y todos los formularios se actualizan solos.

**Resultado de la Fase 1**: el proyecto todavía no "hace" nada visible para un usuario final (no hay pantallas ni análisis de texto todavía), pero **los cimientos están puestos y verificados**. Es como tener la casa nivelada, las conexiones estandarizadas y la imprenta funcionando: a partir de aquí, construir el resto es mucho más seguro y rápido.

---

## 2. Guía de Estudio (Para entender qué hizo la IA)

Vamos a trocear la Fase 1 en **bloques lógicos**. Para cada uno veremos: qué es, qué se creó, y **por qué** se hizo así.

### Bloque 1 — La raíz del monorepo (`1.1`)

**Qué se creó**

| Archivo | Para qué sirve |
|---|---|
| `package.json` (raíz) | Lista de comandos globales (`build`, `lint`, `test`, `generate`...) y herramientas comunes. |
| `pnpm-workspace.yaml` | Declara que cada carpeta dentro de `apps/*` es un "proyecto" independiente dentro del conjunto. |
| `turbo.json` | Define las "tareas" y el orden en que se ejecutan entre proyectos. |
| `.nvmrc`, `.tool-versions` | Fijan las versiones exactas de Node, Go y pnpm que usa el proyecto. |
| `.editorconfig` | Reglas de estilo (tabulaciones, saltos de línea) para que todos editen igual. |
| `.gitignore` | Le dice a Git qué NO debe guardar (por ejemplo, `node_modules/`). |

**El `package.json` raíz**

```json
{
  "name": "langlint",
  "private": true,
  "packageManager": "pnpm@9.15.4",
  "scripts": {
    "build": "turbo run build",
    "generate": "turbo run generate --filter=@langlint/contracts"
  },
  "devDependencies": {
    "turbo": "^2.10.13",
    "typescript": "^7.0.2"
  }
}
```

- `"private": true` es **obligatorio** en la raíz. Evita que alguien publique por accidente este paquete en npm (el registro público de librerías de JavaScript). Es solo un orquestador, no un producto que se instale.
- `"packageManager": "pnpm@9.15.4"` fija la versión de pnpm. Si alguien usa otra, se lo recuerda.
- `"scripts"` son atajos: en vez de escribir comandos largos, escribes `pnpm build`.

**El `turbo.json` (el "director de orquesta")**

```json
{
  "tasks": {
    "build": { "dependsOn": ["^build"], "outputs": ["dist/**", "bin/**", "gen/**"] },
    "lint": { "dependsOn": ["^build"] },
    "generate": { "cache": false, "outputs": ["gen/**", "**/gen_*.go"] }
  }
}
```

- `dependsOn: ["^build"]` significa: "antes de construir este proyecto, construye primero los proyectos de los que depende" (el símbolo `^` = "los de arriba", los proveedores).
- `outputs` indica qué carpetas produce una tarea, para poder guardar el resultado en caché.
- `"generate": { "cache": false }` desactiva la caché para esa tarea: la generación de código siempre debe ejecutarse de nuevo (nunca "reutilizar" un resultado viejo).

> **Concepto — Monorepo**: un solo repositorio de código que contiene varios "proyectos" o aplicaciones. En lugar de tener un repositorio por aplicación, todo vive junto. Ventaja: un cambio que afecta a varios sitios se hace de una vez y de forma consistente.
>
> **Concepto — Workspace**: cada aplicación dentro del monorepo. `pnpm-workspace.yaml` con `packages: ['apps/*']` le dice a pnpm que mire dentro de `apps/` y trate cada subcarpeta con `package.json` como un proyecto.
>
> **Concepto — Turborepo (`turbo`)**: una herramienta que ejecuta tareas (build, test, lint...) en varios proyectos a la vez, en el orden correcto y con caché. Es el "director de orquesta".
>
> **Concepto — `devDependencies`**: librerías que solo se necesitan para desarrollar (no forman parte del producto final).
>
> **Concepto — Versión con `^` (`^2.10.13`)**: acepta actualizaciones compatibles (2.x.x). Sin el `^` sería exactamente esa versión.

### Bloque 2 — El árbol canónico de directorios (`1.2`)

**Qué se creó**

```text
langlint/
├── apps/
│   ├── contracts/          # El "plano" y las herramientas de generación (fuente de la verdad)
│   │   ├── openapi/        # Aquí vive api.yaml
│   │   └── scripts/        # Aquí vive generate.sh
│   ├── backend/            # El cerebro (Go): aquí irá la lógica del servidor
│   └── frontend/           # La cara visible (Next.js/TypeScript)
├── ops/                    # Infraestructura y despliegue
│   ├── docker/
│   └── scripts/
├── docs/                   # Documentación del proyecto
└── .github/workflows/      # Automatizaciones (CI), placeholder para la Fase 7
```

- Se dejó un archivo vacío llamado **`.gitkeep`** dentro de las carpetas que aún no tenían contenido.
- El directorio `docs/` ya existía (contiene `PRODUCT_DOMAIN.md`, `DEVLOG.md`, etc.).

> **Concepto — `.gitkeep`**: Git es un sistema de control de versiones que **no guarda carpetas vacías**. Si quieres que una carpeta "exista" desde ya en el repositorio (aunque esté vacía porque la llenarás en otra fase), metes dentro un archivo testimonial llamado `.gitkeep`. No hace nada; solo evita que Git borre la carpeta.
>
> **Concepto — Árbol canónico de directorios**: la estructura de carpetas "oficial y acordada". Tenerla fijada desde el principio evita que cada desarrollador invente su propio orden.

**¿Por qué se hizo así?** Porque el manifiesto del proyecto (`MANIFEST_MONOREPO.md`) define una estructura estándar. Crear primero el esqueleto evita que en fases posteriores cada pieza caiga en un sitio distinto. La separación `apps/contracts` / `apps/backend` / `apps/frontend` refleja la regla de oro: **el backend y el frontend no se hablan directamente, se hablan a través del contrato**.

### Bloque 3 — El contrato OpenAPI (`1.3`)

**Qué se creó**: `apps/contracts/openapi/api.yaml`, un documento en formato **OpenAPI 3.1.0**.

**Estructura del documento**

```yaml
openapi: 3.1.0
info:
  title: LangLint API
  version: 1.0.0
  license: { name: Apache-2.0, identifier: Apache-2.0 }
servers:
  - url: /v1
security: []        # El MVP no tiene login (decisión explícita)
tags: [...]
paths:              # Los "endpoints": las direcciones a las que se puede llamar
  /practices: ...
components:
  schemas:          # La "forma" de los datos que se intercambian
    Fragment: ...
    ErrorPattern: ...
    Analysis: ...
    Practice: ...
    PracticeStatus: ...
    ...
```

**Contenido principal**

- **9 endpoints** (direcciones de la API). Por ejemplo:
  - `POST /practices` → crear una práctica de escritura.
  - `GET /practices/{practiceId}` → obtener una práctica con su análisis.
  - `POST /practices/{practiceId}/analyze` → pedir el análisis del texto.
  - `GET /analytics/error-patterns`, `GET /v1/me/data/export`, etc.
- **Schemas** (la "forma" de cada dato): `Fragment` (un trozo de frase analizada), `ErrorPattern` (tipo de error: preposición, falso amigo...), `Analysis`, `Practice`, `PracticeStatus`, etc.
- **Enumeraciones** (listas cerradas de valores permitidos):
  - `ErrorPattern.code`: 8 tipos de error gramatical.
  - `severity`: `minor | moderate | critical`.
  - `PracticeStatus`: `draft → analyzing → completed | failed`.

**Decisiones importantes y su por qué**

1. **Todo en `snake_case`** (`source_text`, `practice_id`). `snake_case` = palabras en minúscula separadas por guion bajo. Se fija un único estilo para que Go y TypeScript no se confundan al leer los datos.
2. **`409 analysis_pending` como éxito idempotente**. Cuando pides un análisis que ya está en marcha, el servidor responde con el código 409. Normalmente un 409 es un "error", pero aquí se documenta como "éxito": significa "tranquilo, ya se está haciendo". El frontend debe tratarlo como bueno, no como fallo.
3. **`security: []` explícito**. El MVP no tiene login (es una decisión de producto). En vez de dejar la seguridad "sin definir" (que parece un olvido), se declara **vacía a propósito**: es una decisión consciente.
4. **`servers: /v1` y rutas relativas**. La versión de la API (`/v1`) se declara una sola vez y las rutas son relativas. Así se evita el error clásico de escribir `/v1/v1/practices`.

> **Concepto — OpenAPI**: un estándar para describir APIs (interfaces de comunicación) en un archivo legible. Define qué direcciones existen, qué datos reciben y qué devuelven.
>
> **Concepto — Endpoint**: una "puerta" concreta de la API (una dirección + un método como GET/POST). Por ejemplo, `POST /practices` es la puerta para "crear práctica".
>
> **Concepto — Schema**: el molde o plantilla que describe cómo es un dato (qué campos tiene, de qué tipo, cuáles son obligatorios).
>
> **Concepto — Enum / enumeración**: un campo que solo puede tomar unos pocos valores concretos de una lista. Por ejemplo, `severity` solo puede ser `minor`, `moderate` o `critical`.
>
> **Concepto — SSOT (Single Source of Truth)**: "única fuente de la verdad". Un solo lugar donde está definido algo; el resto se deriva de él. Aquí, `api.yaml` es el SSOT del "idioma" entre backend y frontend.
>
> **Concepto — `snake_case`**: estilo de nombrar con guiones bajos (`error_pattern`). Se contrapone a `camelCase` (`errorPattern`).
>
> **Concepto — Idempotente**: una operación que, si se repite, no cambia el resultado. Pulsar dos veces "analizar" no debe causar dos análisis; el segundo intento devuelve "ya en marcha".
>
> **Concepto — UUID**: un identificador único universal (una cadena larga tipo `123e4567-...`). Se usa para los `id`.
>
> **Concepto — Linter (Redocly)**: una herramienta que revisa el documento del contrato y avisa de errores o malas prácticas. Se usa `@redocly/cli`.

### Bloque 4 — El pipeline de generación (`1.4`)

Esta es la parte más "mágica" y la que probablemente más cuesta entender. Vamos despacio.

**La idea**: `api.yaml` es un texto. A partir de ese texto, una máquina fabrica **código real** en dos idiomas:
- Código **Go** (backend) con los moldes de los datos.
- Código **TypeScript** (frontend) con los mismos moldes.

Como ambos salen del mismo `api.yaml`, es **imposible** que se desincronicen.

**Piezas creadas**

1. **`apps/contracts/scripts/generate.sh`** — el orquestador (el "botón" que lo lanza todo):

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
make -C "$ROOT/apps/backend" generate                      # Go
(cd "$ROOT/apps/frontend" && pnpm run generate)            # TypeScript
```

- `set -euo pipefail` = "modo estricto": si cualquier comando falla, el script se detiene inmediatamente. Evita seguir adelante con errores a medias.
- `ROOT="$(cd "$(dirname ... )" && pwd)"` calcula la carpeta raíz del proyecto **a partir de la ubicación del propio script**, para que funcione sin importar desde dónde lo llames. Es un patrón muy habitual y robusto en scripts.
- Luego delega: en el backend ejecuta el target `generate` del `Makefile`; en el frontend ejecuta `pnpm run generate`.

2. **`apps/backend/Makefile`** — la receta para generar el código Go:

```make
OAPI_CODEGEN_VERSION := v2.8.0
OAPI_CODEGEN := $(shell go env GOPATH)/bin/oapi-codegen
SPEC := ../contracts/openapi/api.yaml
HANDLERS_DIR := internal/api/handlers

generate: tools
	mkdir -p $(HANDLERS_DIR)
	$(OAPI_CODEGEN) -config oapi-codegen-types.yaml $(SPEC)
	$(OAPI_CODEGEN) -config oapi-codegen-server.yaml $(SPEC)
```

- `tools` instala la herramienta `oapi-codegen` en la versión **v2.8.0 exacta**.
- `generate: tools` significa "antes de generar, asegúrate de tener la herramienta".
- Se usan **dos archivos de configuración** para producir **dos archivos Go** distintos: uno con los tipos (`gen_types.go`) y otro con el esqueleto del servidor (`gen_server.go`).

3. **`apps/frontend/package.json`** — la receta para generar el código TypeScript:

```json
{
  "scripts": {
    "generate": "mkdir -p src/lib/api && openapi-typescript ../contracts/openapi/api.yaml -o src/lib/api/gen.ts"
  },
  "devDependencies": {
    "openapi-typescript": "^7.13.0",
    "typescript": "5.9.3"
  }
}
```

- `openapi-typescript` lee el mismo `api.yaml` y escribe `src/lib/api/gen.ts` (los tipos de TypeScript).

4. **Los archivos generados** se guardan en el repositorio (se "commitean"):
   - `apps/backend/internal/api/handlers/gen_types.go`
   - `apps/backend/internal/api/handlers/gen_server.go`
   - `apps/frontend/src/lib/api/gen.ts`

**¿Por qué se hizo así? (las decisiones clave, explicadas)**

- **¿Por qué guardar el código generado en el repositorio en vez de generarlo siempre?** Para que el proyecto se pueda "construir sin internet" y para que el código generado no cambie sin que nadie se dé cuenta. Al estar guardado, si alguien lo modifica a mano, Git lo detecta.
- **¿Por qué dos configuraciones en Go?** `oapi-codegen` (en su versión actual) genera un archivo por ejecución. Para tener "tipos" por un lado y "servidor" por otro (como pide el manifiesto), se hacen dos ejecuciones.
- **¿Por qué versiones exactas (`v2.8.0`, `7.13.0`, `5.9.3`)?** Para que en cualquier ordenador y momento se genere **exactamente** el mismo código. Si usáramos "la última versión", dos personas podrían generar cosas distintas y aparecerían diferencias fantasma.
- **¿Por qué `typescript: 5.9.3` solo en el frontend?** La versión de TypeScript de la raíz (la 7) es incompatible con `openapi-typescript` (necesita la 5). Se aisló la versión 5 únicamente donde hace falta, sin tocar el resto.
- **¿Por qué el script raíz filtra a `--filter=@langlint/contracts`?** Porque, si no, la tarea de generación del frontend se ejecutaría **dos veces** (una por su cuenta y otra llamada por el orquestador). El filtro hace que **el orquestador sea el único** que manda, evitando trabajo duplicado y posibles choques al escribir el mismo archivo.

**¿Qué es "idempotencia" aquí?** Significa que si ejecutas `pnpm generate` dos veces seguidas, el resultado es idéntico: no hay cambios ("sin drift"). Se verificó con `git diff --exit-code` (una orden que falla si hay diferencias). Esto garantiza que el contrato y el código generado están siempre sincronizados.

> **Concepto — Generación de código**: escribir un programa que escribe código. Se usa para evitar tareas repetitivas y errores humanos.
>
> **Concepto — `oapi-codegen` / `openapi-typescript`**: las dos herramientas generadoras. La primera produce Go; la segunda, TypeScript. Ambas leen el mismo `api.yaml`.
>
> **Concepto — `Makefile`**: un archivo con "recetas" para ejecutar tareas (`make generate`). Muy típico en proyectos Go.
>
> **Concepto — `gen_*.go`**: convención de nombre. El prefijo `gen_` avisa de que ese archivo está generado automáticamente y **no debe editarse a mano** (si lo editas, la próxima generación lo sobrescribe).
>
> **Concepto — Drift**: "deriva" o desincronización. Ocurre cuando el código generado y el contrato dejan de coincidir. El pipeline lo evita.
>
> **Concepto — Lockfile (`pnpm-lock.yaml`)**: archivo que registra las versiones exactas de todas las dependencias instaladas, para que todos instalen lo mismo.
>
> **Concepto — `--frozen-lockfile`**: hace que la instalación falle si el lockfile no coincide con lo declarado. Garantiza reproducibilidad.
>
> **Concepto — `GOPATH`**: la carpeta donde Go instala sus herramientas y dependencias (`go env GOPATH` la muestra).

### Bloque 5 — La verificación (el "gate de salida")

Cada fase termina con una **puerta** que debe estar "en verde" antes de pasar a la siguiente. En Fase 1 se comprobó:

| Comando | Qué comprueba |
|---|---|
| `pnpm install --frozen-lockfile` | Que las dependencias se instalan exactamente como están fijadas. |
| `pnpm generate` (x2) | Que el código se regenera y **no cambia** (idempotencia). |
| `pnpm build` | Que el proyecto "construye" sin errores. |
| `pnpm lint` | Que el contrato OpenAPI es válido (Redocly). |

Resultado: todo en verde. Los únicos avisos (warnings) son de Redocly sugiriendo que algunos endpoints de lectura no declaran un error 4XX; se dejaron **conscientemente** porque en este MVP esos endpoints no tienen errores reales que declarar.

> **Concepto — Gate de salida**: criterio objetivo que marca cuándo una fase está terminada. Evita avanzar con cimientos a medias.

---

## 3. Resumen de los cambios

- **Monorepo base + árbol canónico (`1.1`–`1.2`)**: se montó la raíz con pnpm workspaces y Turborepo (`package.json`, `pnpm-workspace.yaml`, `turbo.json`, `docs/`, `ops/`, `.github/workflows/`) y la estructura de `apps/contracts`, `apps/backend` y `apps/frontend`.
- **Contrato OpenAPI fundacional (`1.3`)**: se creó `apps/contracts/openapi/api.yaml` (OpenAPI 3.1.0) como única fuente de la verdad, con 9 endpoints y los schemas `Fragment`, `ErrorPattern`, `Analysis`, `Practice` y `PracticeStatus`, todo en `snake_case` y con los códigos de error (incluido `409 analysis_pending` como éxito idempotente).
- **Pipeline de generación (`1.4`)**: se implementó `pnpm generate` (orquestador `generate.sh` + `Makefile`/`oapi-codegen` para Go + `openapi-typescript` para TS), generando y commiteando `gen_types.go`, `gen_server.go` y `gen.ts`, con versiones fijadas, validación del contrato (`pnpm lint`) y verificación de idempotencia (`pnpm generate` dos veces sin diferencias).
