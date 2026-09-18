# Fase 3 — Puertos y Adaptadores: explicación didáctica

> Documento de estudio sobre todo lo construido en la **Fase 3 (Puertos y Adaptadores)** del proyecto LangLint.
> Cubre los bloques `3.1` (puertos/interfaces + mocks), `3.2` (repositorios Postgres + migraciones), `3.3` (Unit of Work) y `3.4` (Outbox + Event Bus).
> Es la continuación de `docs/explains/fase-2-dominio-puro.md`.

---

## 1. ¿De qué trata esto? (Explicación para no técnicos)

Imagina que en la fase anterior fabricaste un **juego de mesa** completo: las fichas, las tarjetas y el reglamento que dice qué movimientos son válidos. El problema es que todo eso quedó dentro de una **habitación sellada**: era perfecto y estaba probadísimo, pero no se podía guardar, ni avisar a nadie, ni sobrevivir a un apagón. La Fase 3 construye **las puertas, el almacén y el sistema de mensajería** para sacar ese juego al mundo real.

Cuatro imágenes de la vida cotidiana resumen todo lo que se hizo:

| Imagen de la vida real | Qué representa en el proyecto |
|---|---|
| **Un almacén con estanterías** | La base de datos **PostgreSQL**, donde las piezas se guardan de forma permanente y ordenada. |
| **Enchufes y adaptadores de viaje** | Los **puertos** (la "toma estándar" que todos entienden) y los **adaptadores** (el enchufe concreto que encaja en esa toma). Así las piezas no dependen de la marca del almacén. |
| **Un cuaderno de recados + un cartero** | El **outbox** (cuaderno duradero de avisos) y el **relay** (el cartero que los reparte). Se anota el aviso *dentro* de la misma operación en que se guarda la pieza; así, aunque el sistema se caiga justo después, el recado no se pierde. |
| **Una cesta de la compra** | El **Unit of Work**: una operación "todo o nada". Si un solo paso falla, se cancela todo (como devolver la compra entera al carrito si la tarjeta es rechazada). |

También hay una imagen para los avisos internos: un **contestador automático con buzón limitado**. Si el buzón se llena, la máquina **descarta** el aviso más nuevo en lugar de dejar de atender el teléfono. Eso se llama *backpressure* (control de presión) y evita que un pico de trabajo bloquee el sistema.

**La base de datos** guarda cuatro "cajones": las prácticas del alumno, sus análisis, las estadísticas de errores y el cuaderno de avisos pendientes. Cada vez que el alumno crea o modifica algo, el cambio se guarda **y** se deja el aviso en el cuaderno, los dos en la misma operación. Después, el cartero lee el cuaderno y avisa a quien corresponda (por ejemplo, "hay que analizar esta práctica").

**Resultado de la Fase 3**: para el usuario todavía **no cambia nada visible** (no hay pantallas nuevas), pero el juego ya está conectado a un almacén real y a un sistema de mensajes que sobrevive a fallos. Es la fontanería profesional: aburrida de mirar, imprescindible para que nada se pierda.

---

## 2. Guía de Estudio (Para entender qué hizo la IA)

Vamos por bloques. Primero recordaremos la arquitectura ("puertos y adaptadores"), y luego iremos pieza por pieza. Para cada bloque veremos **qué se creó**, **cómo funciona** y **por qué** se hizo así.

### Bloque 0 — La idea de "puertos y adaptadores"

Cuando un programa grande necesita guardar datos o hablar con un servicio externo, hay dos opciones: acoplar la lógica del negocio a esa tecnología concreta (malo), o **definir primero una interfaz** que describa *qué* se necesita, y dejar la implementación concreta aparte (bueno). Eso es la arquitectura de **puertos y adaptadores**.

```text
   Fuera  ───────────────────────────────────────────────►  Dentro
   cmd → adapters → services → ports → domain
   (arranque)  (DB/LLM)  (casos de uso)  (interfaces)  (REGLAS PURAS)
```

- **Puerto**: una **interfaz** (un "contrato" o enchufe estándar). Dice *qué* operaciones existen, sin decir *cómo* se hacen. Vive en `internal/api/ports/`.
- **Adaptador**: una **implementación concreta** de ese puerto para una tecnología. Vive en `internal/api/adapters/`. Por ejemplo, `PostgresPracticeRepository` implementa el puerto `PracticeRepository` usando Postgres.
- **Regla A2 (dirección de dependencias)**: las dependencias van **hacia dentro**. El dominio no conoce a nadie; los adaptadores conocen los puertos y el dominio.

**¿Por qué tanto rodeo?** Porque así se puede cambiar Postgres por otra base de datos, o el proveedor de IA por otro, **sin tocar las reglas del negocio**. Y sobre todo: se puede **probar sin infraestructura real** usando *mocks* (imitaciones del puerto).

> **Concepto — Puerto (port)**: interfaz que define un conjunto de operaciones. Es el "enchufe" que el resto del código conoce.
>
> **Concepto — Adaptador (adapter)**: implementación real del puerto para una tecnología concreta (Postgres, OpenAI, etc.).
>
> **Concepto — Interfaz**: en Go, un contrato que lista métodos; cualquier tipo que los implemente "encaja" en ella.
>
> **Concepto — Inyección de dependencias (DI)**: entregar a un servicio las piezas que necesita (por ejemplo el repositorio) desde fuera, en vez de que él las fabrique.
>
> **Concepto — Mock**: imitación controlada de un puerto que se usa en los tests; permite simular respuestas y errores sin base de datos.

### Bloque 1 — Los puertos y los mocks (`3.1`)

#### 1.1 Los contratos del almacén

Se definieron tres puertos de repositorio (uno por tipo de dato guardable) y el contrato de transacciones:

```go
// internal/api/ports/storage/practice_repository.go
type PracticeRepository interface {
	Save(ctx context.Context, practice *practice.Practice) error
	GetByID(ctx context.Context, id domain.ID) (*practice.Practice, error)
	ListByUser(ctx context.Context, userID domain.ID, limit, offset int) ([]practice.Practice, int, error)
}
```

```go
// internal/api/ports/storage/unit_of_work.go
type UnitOfWork interface {
	InTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

Además: `AnalysisRepository`, `ErrorMetricRepository` (mismo estilo), el puerto **`LLMExtractor`** (el futuro "llamador" de la IA) y los tres puertos de eventos (`EventDispatcher`, `EventHandler`, `Outbox`, del bloque 4).

**¿Por qué interfaces tan pequeñas y específicas?** Porque un puerto pequeño es fácil de implementar y de imitar en tests. Cuanto más pequeño el contrato, menos acoplamiento.

> **Concepto — `context.Context`**: objeto que viaja con cada operación y transporta plazos (timeouts), cancelaciones y algunos valores compartidos. Es la forma idiomática de decir "esto puede cancelarse".
>
> **Concepto — Repositorio**: objeto que se encarga de guardar/leer datos, ocultando si por debajo hay SQL, un fichero u otra fuente.

#### 1.2 Mocks generados con `mockgen`

Para poder probar los servicios sin base de datos, se generan **mocks** automáticamente a partir de los puertos con la herramienta `mockgen` (librería `go.uber.org/mock`, versión `v0.6.0`). El `Makefile` del backend tiene un objetivo `mocks` que los regenera de forma reproducible:

```makefile
mocks: tools
	$(MOCKGEN) -package mocks -destination $(MOCKS_DIR)/storage.go $(MODULE)/internal/api/ports/storage PracticeRepository,AnalysisRepository,ErrorMetricRepository,UnitOfWork
	...
```

**¿Por qué generarlos con una herramienta y no escribirlos a mano?** Porque si mañana cambia el puerto, se regeneran y **nunca quedan desincronizados** con el contrato.

> **Concepto — `mockgen`**: herramienta que lee una interfaz y escribe un "doble" de prueba con métodos que puedes programar para devolver lo que quieras.

### Bloque 2 — Repositorios Postgres y migraciones (`3.2`)

#### 2.1 Las "migraciones": el plano de las estanterías

Una **migración** es un archivo con instrucciones SQL que construye el esquema de la base de datos. Se aplican en orden y llevan control de versión: la primera vez crean las tablas; las siguientes, las amplían. El archivo es `apps/backend/migrations/000001_init.sql` y crea cuatro tablas:

| Tabla | Guarda | Detalle |
|---|---|---|
| `practices` | Las prácticas del alumno | Incluye `deleted_at` para el borrado lógico (*soft-delete*). |
| `analyses` | La corrección de la IA | Relación 1-a-1 con `practices` (clave foránea única). |
| `error_metrics` | Estadísticas de errores | Clave compuesta por usuario + código de error + ventana temporal. |
| `outbox_events` | El cuaderno de avisos | Se explica en el bloque 4. |

El runner (`migrations.go`) **incrusta** los `.sql` en el propio binario con `//go:embed` y los aplica con la librería `goose`:

```go
//go:embed *.sql
var FS embed.FS

func Up(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	goose.SetBaseFS(FS)
	if err := goose.SetDialect("postgres"); err != nil { return err }
	return goose.Up(db, ".")
}
```

**¿Por qué incrustar los archivos?** Porque el binario viaja con sus migraciones dentro: no hay que copiar archivos aparte ni depender de la carpeta del proyecto al desplegar.

> **Concepto — Migración**: cambio versionado del esquema de la base de datos (crear una tabla, añadir una columna...).
>
> **Concepto — `//go:embed`**: directiva de Go que mete archivos (aquí `.sql`) dentro del binario compilado.
>
> **Concepto — `goose`**: herramienta que aplica migraciones SQL en orden y recuerda cuáles ya se aplicaron.

#### 2.2 El truco `querier`: la transacción o el pool

Un "repositorio" puede trabajar de dos formas: **dentro de una transacción** (cuando forma parte de una operación compuesta) o **suelto**, cogiendo una conexión del *pool* (el conjunto de conexiones disponibles). Para no duplicar cada consulta, se definió una interfaz privada mínima que **ambos** tipos satisfacen:

```go
// internal/api/adapters/postgres/repositories/querier.go
type querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func conn(ctx context.Context, pool *pgxpool.Pool) querier {
	if tx := txctx.From(ctx); tx != nil {
		return tx   // hay una transacción en el contexto: nos unimos a ella
	}
	return pool     // no hay: usamos una conexión del pool
}
```

Cada método del repositorio empieza con `conn(ctx, r.pool)` y ejecuta la misma consulta, ya sea dentro o fuera de transacción. **Esto es lo que conecta el bloque 2 con el bloque 3.**

#### 2.3 La "frontera SQL": convertir tipos

El dominio tiene tipos propios (`domain.ID`, `practice.SourceText`, `[]practice.TargetRule`...) que **no son** los tipos de la base de datos. El adaptador es el único que traduce:

- **Identificadores**: `domain.ID` es un `[16]byte`; se convierte a texto UUID con `id.String()` al guardar y con `domain.ParseID(...)` al leer.
- **Listas y objetos anidados** (`target_rules`, `fragments`): se serializan a JSON con `json.Marshal` y se insertan con un *cast* `$n::jsonb`; al leer, se hace `json.Unmarshal`.

```go
// guardar
INSERT INTO practices (id, user_id, source_text, draft_text, target_rules, status, ...)
VALUES ($1, $2, $3, $4, $5::jsonb, $6, ...)
ON CONFLICT (id) DO UPDATE SET ...
```

**¿Por qué `ON CONFLICT ... DO UPDATE`?** Es un *upsert*: "inserta; si ya existe ese `id`, actualízalo". Así `Save` sirve tanto para crear como para modificar con un solo método.

> **Concepto — `jsonb`**: tipo de columna de Postgres que guarda JSON de forma indexable.
>
> **Concepto — Upsert**: inserción que, si la fila ya existe, la actualiza en vez de fallar.
>
> **Concepto — Pool de conexiones**: conjunto reutilizable de conexiones a la base de datos; abrir/cerrar una por consulta sería lentísimo.

#### 2.4 Errores: traducir la infraestructura al dominio (A5)

Cuando la base de datos dice "no encontré nada", el servicio **no debe enterarse** de que eso es SQL. El adaptador traduce:

```go
// internal/api/adapters/postgres/repositories/errors.go
func mapError(field string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return &domain.NotFoundError{Field: field, Message: "not found"}
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {   // violación de unicidad
		return &domain.ValidationError{Field: field, Message: "already exists"}
	}
	return &domain.InternalError{Field: field, Message: "database error"}
}
```

**¿Por qué?** Porque el dominio (y mañana el HTTP) necesitan errores **con nombre**: "no encontrado" (404), "entrada inválida" (422), "fallo interno" (500). Sin esta traducción, se filtrarían detalles de Postgres al usuario.

> **Concepto — `pgx.ErrNoRows`**: error que devuelve la librería cuando una consulta no encuentra filas.
>
> **Concepto — Código `23505`**: el identificador que Postgres usa para "clave duplicada".

#### 2.5 Dos sorpresas reales resueltas

- **`window` es palabra reservada en Postgres** (se usa para funciones de ventana). Por eso la columna se escribe `"window"` **entre comillas** en el SQL. Se descubrió porque el test (valga la redundancia) falló con `syntax error at or near "window"`.
- **`goose` no admite archivos separados `.up.sql` / `.down.sql`**: los interpreta como dos migraciones de la misma versión y aborta. Se usó su formato nativo: **un solo archivo** con secciones `-- +goose Up` y `-- +goose Down`.

### Bloque 3 — Unit of Work (`3.3`)

#### 3.1 El problema: varias escrituras, una sola operación

Crear una práctica significa: **guardar la práctica** *y* **anotar el aviso**. Si se guarda la práctica pero se cae antes de anotar el aviso, el aviso se pierde. Necesitamos que ambas cosas ocurran **juntas o ninguna**.

#### 3.2 La solución: una transacción que viaja en el contexto

El `PostgresUnitOfWork` abre una transacción, la **mete en el `context.Context`** y ejecuta dentro la función del servicio. Si la función devuelve error, hace *rollback*; si no, *commit*:

```go
func (u *PostgresUnitOfWork) InTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := u.pool.Begin(ctx)
	if err != nil { return &domain.InternalError{Field: "transaction", Message: "cannot begin transaction"} }

	if err := fn(txctx.With(ctx, tx)); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil { /* ... */ }
		return err
	}
	if err := tx.Commit(ctx); err != nil { return &domain.InternalError{Field: "commit", Message: "cannot commit transaction"} }
	return nil
}
```

La transacción se guarda en el contexto con una clave privada, en un paquete compartido por el UoW y los repositorios:

```go
// internal/api/adapters/postgres/txctx/tx_context.go
type txKey struct{}

func With(ctx context.Context, tx pgx.Tx) context.Context { return context.WithValue(ctx, txKey{}, tx) }
func From(ctx context.Context) pgx.Tx {
	tx, _ := ctx.Value(txKey{}).(pgx.Tx)
	return tx
}
```

Así, el repositorio del bloque 2 no recibe la transacción como parámetro: **la descubre en el contexto**.

> **Concepto — Transacción**: grupo de operaciones de base de datos que se aplican todas juntas o ninguna.
>
> **Concepto — Commit / Rollback**: "confirmar" los cambios / "deshacerlos" por completo.
>
> **Concepto — `context.WithValue`**: forma de adjuntar un valor (aquí, la transacción) a un contexto para que viaje por las llamadas.
>
> **Concepto — AP8**: regla del proyecto que prohíbe que un *servicio* inicie transacciones SQL. Solo el adaptador (`PostgresUnitOfWork`) toca `pgx`.

#### 3.3 El error interno y el "diccionario" que faltaba

Se añadió al dominio `domain.InternalError` ("fallo de infraestructura inesperado", 500). **¿Por qué?** Porque la regla A5 dice que los adaptadores deben convertir **todo** error de infraestructura a un error de dominio; para lo imprevisto de la base de datos no había ningún tipo, así que se creó uno.

### Bloque 4 — Outbox + Event Bus (`3.4`)

Este es el patrón estrella de la fase: **cómo avisar a otros módulos sin acoplarlos y sin perder avisos**.

#### 4.1 El patrón "outbox" (buzón de salida)

En vez de que un servicio "grite" el aviso directamente (lo que se puede perder si el proceso cae), se sigue este flujo:

```text
[1] Servicio: uow.InTransaction { guardar entidad; outbox.Append(aviso) }
[2] COMMIT  → entidad y aviso quedan guardados A LA VEZ (atómicos)
[3] El cartero (relay) lee los avisos no publicados de la tabla outbox_events
[4] Los entrega al bus de eventos (dispatcher)
[5] El bus los reparte a los interesados (handlers)
[6] El relay marca el aviso como publicado
```

**¿Por qué así?** Porque el aviso se guarda en la **misma transacción** que el dato (AP7). Si algo falla, no hay aviso ni dato: nunca existe el caso "dato guardado, aviso perdido".

#### 4.2 Las piezas de datos

En el dominio se añadió la entidad del aviso persistido y un **registro** que liga cada nombre de evento con su tipo concreto:

```go
// internal/domain/events.go
type OutboxEvent struct {
	ID          ID         `json:"id"`
	EventType   string     `json:"event_type"`
	Payload     []byte     `json:"payload"`
	CreatedAt   time.Time  `json:"created_at"`
	PublishedAt *time.Time `json:"published_at"`
	Attempts    int        `json:"attempts"`
}

func NewEvent(eventType string) (DomainEvent, bool) {
	switch eventType {
	case EventNameIdentityIssued:     return &IdentityIssued{}, true
	case EventNamePracticeCreated:    return &PracticeCreated{}, true
	case EventNameAnalysisCompleted:  return &AnalysisCompleted{}, true
	case EventNameAnalysisFailed:     return &AnalysisFailed{}, true
	default:                           return nil, false
	}
}
```

**¿Por qué `NewEvent` vive en el dominio?** Porque es la **única lista** de todos los eventos que existen. Si se duplicara en cada adaptador, un evento nuevo se olvidaría en alguno. El codec (`adapters/events/codec.go`) es quien convierte a/desde JSON:

```go
func MarshalEvent(event domain.DomainEvent) (string, []byte, error) {
	payload, err := json.Marshal(event)
	return event.EventName(), payload, err
}
func UnmarshalEvent(eventType string, payload []byte) (domain.DomainEvent, error) {
	event, ok := domain.NewEvent(eventType)
	if !ok { return nil, fmt.Errorf("unknown event type %q", eventType) }
	return event, json.Unmarshal(payload, event)
}
```

#### 4.3 El cuaderno (`PostgresOutbox`)

`PostgresOutbox.Append` codifica el evento y hace un `INSERT` en `outbox_events`, **usando la transacción que venga en el contexto** (o el pool si no hay):

```go
if tx := txctx.From(ctx); tx != nil {
	if _, err := tx.Exec(ctx, q, args...); err != nil { return &domain.InternalError{...} }
	return nil
}
_, err := o.pool.Exec(ctx, q, args...)
```

#### 4.4 El altavoz interno (`InMemoryEventDispatcher`)

Es un **bus de eventos en memoria**: un canal con cola, atendido por varios trabajadores (*worker pool*). Quien quiera enterarse de un evento se **suscribe** con un handler.

```go
func (d *InMemoryEventDispatcher) Dispatch(ctx context.Context, event domain.DomainEvent) error {
	select {
	case d.ch <- event:
	default:   // cola llena: se descarta el aviso (drop policy)
	}
	return nil
}
```

- `DefaultBufferSize = 1024` y `DefaultWorkers = 4`.
- **Drop policy** (*backpressure*): si la cola está llena, se descarta el evento en lugar de bloquear al cartero. Es una decisión de disponibilidad: mejor perder un aviso que congelar el sistema.

> **Concepto — Event bus**: mecanismo que reparte eventos a quien esté suscrito a ellos.
>
> **Concepto — Canal (`chan`)**: tubo por el que pasan datos entre goroutines.
>
> **Concepto — Worker pool**: grupo de goroutines que van sacando trabajo de una cola.
>
> **Concepto — Backpressure**: estrategia para reaccionar cuando llega más trabajo del que se puede procesar.

#### 4.5 El cartero (`OutboxRelay`)

`Tick` publica **un lote** de avisos pendientes:

```sql
SELECT id, event_type, payload, created_at, attempts
FROM outbox_events
WHERE published_at IS NULL
ORDER BY created_at, id
LIMIT $1
FOR UPDATE SKIP LOCKED
```

- `FOR UPDATE SKIP LOCKED` bloquea las filas seleccionadas y **salta** las que otro cartero (u otra ejecución) ya esté tratando. Evita entregar el mismo aviso dos veces y evita bloqueos.
- Por cada aviso: se reconstruye el evento, se entrega al bus y se marca `published_at`.
- Si un aviso no se puede decodificar o entregar, se incrementa `attempts` y queda pendiente para el siguiente intento.
- `Run(ctx, intervalo)` repite `Tick` cada cierto tiempo; `Tick` se usa directamente en los tests.

> **Concepto — `FOR UPDATE SKIP LOCKED`**: instrucción de Postgres para "tomar" filas sin pelearse con otras consultas simultáneas.
>
> **Concepto — AP7**: regla que prohíbe llamar al dispatcher **dentro** de la transacción; siempre se anota en el outbox y el relay publica después del commit.

### Bloque 5 — Cómo se prueba todo esto

El proyecto usa una **pirámide de tres niveles**:

| Tier | Qué prueba | Cómo | ¿Necesita Docker? |
|---|---|---|---|
| **Tier 1** | Lógica pura y adaptadores sin infraestructura | `go test -race ./...` | No |
| **Tier 2** | Handlers HTTP con `httptest` | (fase futura) | No |
| **Tier 3** | SQL real, transacciones, outbox, migraciones | `go test -tags=integration ./...` | **Sí** |

- Los tests Tier 3 llevan la etiqueta `//go:build integration` y usan **testcontainers**: levantan un Postgres 16 real en un contenedor temporal, aplican las migraciones y lo destruyen al terminar. Un contenedor por suite, gracias a `TestMain`.
- La imagen `postgres:16-alpine` **no** está lista al instante; `BasicWaitStrategies()` espera a que el servidor esté disponible de verdad (su log aparece dos veces porque se reinicia durante la instalación). Sin esa espera, los tests fallaban con "connection reset".
- Convención de nombres: `TestXxx_Método_Condición_ResultadoEsperado`. Sin `t.Skip()`.
- Verificación de la fase: `pnpm test-integration` (Tier 3) en verde, cobertura de adaptadores **≥70%** y dominio **100%** con `-race`.

> **Concepto — testcontainers**: librería que arranca servicios reales (Postgres, etc.) dentro de contenedores Docker para los tests.
>
> **Concepto — `go test -race`**: ejecuta los tests con el detector de condiciones de carrera (accesos simultáneos inseguros).
>
> **Concepto — Build tag**: etiqueta en la cabecera del archivo que decide si se compila o no en cada corrida.

### Bloque 6 — Decisiones y desviaciones (para que nada sorprenda)

- **Sin `tenant_id` ni RLS**: el manifiesto A6 habla de multi-inquilino (varios "tenants"), pero LangLint es **single-user** y el dominio no tiene ese campo. Se aisló por `user_id`. Añadir RLS sería un cambio aditivo cuando exista multi-usuario.
- **Goose en un solo archivo**: explicado en 2.5; es una desviación consciente del nombre `.up.sql`/`.down.sql`.
- **`ErrorMetric.Upsert` absoluto**: guarda el contador como valor final (no lo incrementa). Así, si el mismo aviso se procesa dos veces, el resultado no cambia (idempotencia).
- **Services diferidos a Fase 4**: los ítems `3.4.3` ("los services publican vía outbox") y `3.4.4` ("handlers idempotentes") hablan de *services*, que aún no existen (llegan con el motor de IA). En esta fase se dejó **el mecanismo** verificado con un test de extremo a extremo; el "enforce" real y la cobertura de *services* (gate ≥90%) se cierran en Fase 4.
- **`cmd/migrate` y `shared/db` diferidos**: no eran ítems de esta fase; el runner incrustado ya sirve a los tests y al futuro CLI.

**Glosario rápido de piezas**

| Pieza | Archivo | En una frase |
|---|---|---|
| Puertos | `internal/api/ports/**` | Los "enchufes" estándar. |
| Repositorios | `adapters/postgres/repositories/**` | Guardan/leen en Postgres. |
| Migraciones | `migrations/000001_init.sql` | El plano de las tablas. |
| `txctx` | `adapters/postgres/txctx/**` | La transacción viajando en el contexto. |
| `PostgresUnitOfWork` | `adapters/postgres/unit_of_work.go` | Abre/cierra transacciones "todo o nada". |
| `PostgresOutbox` | `adapters/postgres/outbox.go` | Escribe avisos en la misma transacción. |
| `InMemoryEventDispatcher` | `adapters/events/in_memory_dispatcher.go` | Reparte eventos en memoria. |
| `OutboxRelay` | `adapters/events/outbox_relay.go` | El cartero que publica los avisos. |
| `OutboxEvent` / `NewEvent` | `internal/domain/events.go` | El aviso persistido y su registro. |

---

## 3. Resumen de los cambios

- **Puertos e interfaces (`3.1`)**: se definieron los contratos de `PracticeRepository`, `AnalysisRepository`, `ErrorMetricRepository`, `UnitOfWork`, `LLMExtractor` y los puertos de eventos (`EventDispatcher`, `EventHandler`, `Outbox`), y se generaron sus **mocks** con `mockgen` (`go.uber.org/mock v0.6.0`, objetivo `make mocks`).
- **Repositorios Postgres, migraciones y Unit of Work (`3.2`–`3.3`)**: se creó el esquema con `goose` (`practices`, `analyses`, `error_metrics`, `outbox_events`), los tres repositorios sobre un helper `querier` que une la transacción o el pool, el `PostgresUnitOfWork` con `txctx` (transacción en el contexto) y el mapeo de errores de infraestructura a errores de dominio (`NotFound`, `Validation`, `Internal`).
- **Outbox + Event Bus (`3.4`)**: se añadieron `OutboxEvent` y el registro `NewEvent`, el codec `Marshal`/`Unmarshal`, el `PostgresOutbox` (aviso y dato en la misma transacción), el `InMemoryEventDispatcher` (canal + workers + *drop policy*) y el `OutboxRelay` (`Tick`/`Run` con `FOR UPDATE SKIP LOCKED`); todo verificado con tests Tier 1 y Tier 3 (testcontainers) — adaptadores ≥70% y dominio 100%.
