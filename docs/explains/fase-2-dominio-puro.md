# Fase 2 — Dominio Puro: explicación didáctica

> Documento de estudio sobre todo lo construido en la **Fase 2 (Dominio Puro)** del proyecto LangLint.
> Cubre los bloques `2.1` (`identity/`), `2.2` (`practice/`), `2.3` (`analysis/`), `2.4` (`analytics/`) y `2.5` (raíz compartida del dominio).
> Es la continuación de `docs/explains/fase-1-fundaciones.md`.

---

## 1. ¿De qué trata esto? (Explicación para no técnicos)

Imagina que vas a fabricar un **juego de mesa** de aprendizaje de inglés. Antes de imprimir el tablero, las cartas y la caja, necesitas dos cosas: **(1) las piezas** (las fichas, los dados, las tarjetas) y **(2) el reglamento** que dice qué es cada pieza y qué movimientos son válidos. Si empiezas a jugar sin reglamento, cada jugador inventará sus propias normas y el juego será un caos.

**La Fase 2 construye exactamente eso para LangLint: las piezas y el reglamento del producto.** Todavía no hay pantallas, ni base de datos, ni llamadas a la inteligencia artificial. Solo está escrito, con todo detalle y verificado, *qué es cada cosa* y *qué se puede hacer con cada cosa*.

Las "piezas" se agrupan en cuatro cajas, cada una con su propio mundo:

| Caja | Qué contiene (en palabras llanas) |
|---|---|
| **`identity`** | La **ficha del estudiante**: quién es (su correo), sin depender de ninguna escuela. |
| **`practice`** | La **hoja de ejercicio**: el texto en español, el borrador en inglés del alumno y las reglas que quiere practicar. Tiene un **estado**: borrador → analizando → terminado o fallido. |
| **`analysis`** | La **corrección del profesor**: el texto troceado frase por frase, con la corrección, la explicación gramatical y los errores detectados. |
| **`analytics`** | El **cuaderno de estadísticas**: cuántas veces el alumno falla cada tipo de error, semana a semana, para ver si mejora. |

Y el "reglamento" son las **reglas del juego**:

- Una hoja de ejercicio **no puede editarse** una vez que el profesor ha empezado a corregirla (solo se edita en estado "borrador").
- Una hoja **no puede borrarse mientras se está corrigiendo** (sería como arrancarle el examen al profesor de las manos).
- Una corrección **no puede darse por terminada si está vacía** (no tiene sentido "corregir" sin ninguna frase).
- Los textos **no pueden estar en blanco**, y cada ejercicio necesita **al menos una regla objetivo**.
- Existe una lista cerrada de **tipos de error** (preposiciones, falsos amigos, tiempos verbales...) y de **gravedades** (leve, moderada, grave).

Además, el sistema tiene un pequeño **libro de quejas predefinidas** (los errores). En vez de decir "algo salió mal", usa quejas concretas y con nombre: "no encontrado", "estado inválido", "el proveedor de IA no responde". Así, cuando en el futuro el producto hable con un usuario por internet, sabrá exactamente qué responder en cada caso.

**Resultado de la Fase 2**: el juego todavía no se puede jugar (no hay tablero ni pantallas), pero **las piezas están fabricadas, el reglamento está escrito y todo está probado**. Es como tener todas las cartas impresas y las reglas revisadas: construir el tablero y empezar a jugar será mucho más rápido y seguro.

---

## 2. Guía de Estudio (Para entender qué hizo la IA)

Vamos a trocear la Fase 2 en **bloques lógicos**. Primero entenderemos la idea general ("el dominio puro"), y luego iremos caja por caja. Para cada bloque veremos: qué se creó, cómo funciona y **por qué** se hizo así.

### Bloque 0 — ¿Qué es un "dominio puro" y por qué separar "contextos"?

**La idea principal.** Cuando se programa un producto grande, se separa el código en capas. La capa más interna y más importante es el **dominio**: la lógica de negocio pura (las reglas del juego), escrita sin saber nada de bases de datos, de internet o de pantallas.

```text
   Fuera  ───────────────────────────────────────────────►  Dentro
   cmd → adapters → services → ports → domain
   (arrancar)  (DB/LLM)  (casos de uso)  (interfaces)  (REGLAS PURAS)
```

- **Regla A1**: el dominio (`apps/backend/internal/domain/`) **solo** puede usar herramientas del lenguaje estándar de Go (la "caja de herramientas" que viene con el lenguaje). Cero librerías externas.
- **Regla A2**: las dependencias van **hacia dentro**. El dominio no conoce a nadie; los demás conocen al dominio.
- **Regla A3**: el dominio se divide en **cuatro "contextos"** (`identity`, `practice`, `analysis`, `analytics`) que **no se hablan entre sí**. Se comunican por eventos (ver Bloque 1).

**¿Por qué tanto aislamiento?** Porque las reglas del negocio son lo que menos cambia. Si mañana se cambia la base de datos, el proveedor de IA o el servidor web, **las reglas no deberían tener que tocarse**. Un dominio puro es más fácil de probar (no necesita base de datos ni internet para testearse) y más fácil de leer.

**¿Qué se creó en la raíz (`domain/`)?** Un paquete "raíz" con las piezas **compartidas** por los cuatro contextos: los identificadores (`ID`), los errores, los eventos y el `ErrorPattern`. Eso es el Bloque 1.

> **Concepto — DDD (Domain-Driven Design)**: una forma de diseñar software poniendo el modelo del negocio en el centro, con un lenguaje común entre negocio y código (aquí, "Practice", "Analysis", "ErrorPattern"...).
>
> **Concepto — Dominio**: la parte del programa que contiene las reglas del negocio, aislada de la tecnología.
>
> **Concepto — Bounded context ("contexto delimitado")**: cada uno de los "mundos" con su propio modelo y vocabulario (`identity`, `practice`, `analysis`, `analytics`). Se mantienen separados para no enredarse.
>
> **Concepto — Agregado**: un grupo de datos que se trata como una sola unidad con reglas propias (p. ej. una `Practice` con sus textos y reglas). Su "raíz" es el objeto principal por el que se accede al resto.
>
> **Concepto — Entidad**: un objeto con **identidad propia** que perdura en el tiempo (un `User` concreto, una `Practice` concreta), aunque sus datos cambien.
>
> **Concepto — Value Object (objeto valor)**: un objeto definido solo por **sus datos**, sin identidad propia. Un `Email` o un `SourceText` no "son" nadie en concreto; son válidos o no lo son. Se validan al construirse.

### Bloque 1 — La raíz del dominio (lo compartido)

Aquí viven cuatro archivos: `identifiers.go` (los IDs), `errors.go` (las quejas), `events.go` (los anuncios) y `error_pattern.go` (la taxonomía de errores).

#### 1.1 `identifiers.go` — los identificadores UUID v7 hechos a mano

Cada entidad importante necesita un identificador único. En vez de usar una librería externa, se implementó **dentro** del dominio:

```go
// ID is an RFC 9562 UUID v7 generated with the standard library only (A7).
type ID [16]byte

func NewID() (ID, error) {
	var id ID
	if _, err := randRead(id[:]); err != nil {
		return ID{}, fmt.Errorf("domain: generate id: %w", err)
	}

	ms := uint64(time.Now().UnixMilli())
	id[0] = byte(ms >> 40)
	// ... (se colocan los bytes del tiempo) ...
	id[6] = (id[6] & 0x0f) | 0x70   // marca de versión 7
	id[8] = (id[8] & 0x3f) | 0x80   // marca de variante
	return id, nil
}
```

- Se genera con `crypto/rand` (números aleatorios seguros) y se le **inyecta la hora actual** en los primeros bytes.
- El resultado se expone como texto tipo `123e4567-e89b-7ddd-...` mediante `String()` y `MarshalJSON()`.
- La API pública es: `NewID`, `MustNewID`, `ParseID`, `IsValid`, `Version`, `IsZero`, `String`.

**¿Por qué un UUID "v7" y no uno cualquiera?** Porque v7 **empieza por la fecha**, así que los IDs **crecen en orden temporal**: los más nuevos siempre van "después". Eso ayuda a las bases de datos a ordenar y también **evita que se puedan adivinar** (no son "user-1", "user-2"...). Es un requisito de seguridad del proyecto (A7).

**¿Por qué implementarlo a mano?** Porque el dominio debe ser independiente: si mañana se quiere cambiar de generador, se cambia **una sola función** sin tocar el resto del sistema.

> **Concepto — UUID**: "identificador único universal", una cadena larga que (en la práctica) nunca se repite.
>
> **Concepto — UUID v7**: una versión concreta de UUID que incluye la fecha al principio, logrando orden temporal + aleatoriedad.
>
> **Concepto — `[16]byte`**: un array fijo de 16 bytes. Al ser de tamaño fijo, es **comparable** (`==`), por lo que se puede usar como clave de un `map`.
>
> **Concepto — `crypto/rand`**: la fuente de números aleatorios "de verdad" de la librería estándar (a diferencia de un aleatorio predecible).

#### 1.2 `errors.go` — errores con nombre y apellidos

El proyecto prohíbe devolver un error genérico ("algo falló"). Cada situación tiene **su propio tipo de error**:

```go
// ValidationError signals invalid input to a domain operation (HTTP 422).
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
```

Los seis errores y su significado:

| Error | Cuándo se usa | HTTP (futuro) |
|---|---|---|
| `ValidationError` | Entrada inválida (texto vacío, sin reglas, etc.) | 422 |
| `NotFoundError` | La práctica o el análisis no existe | 404 |
| `AnalysisPendingError` | Piden el resultado antes de que termine el análisis | 409 |
| `InvalidStateError` | La operación no es válida para el estado actual | 409 |
| `AnalysisFailedError` | El análisis terminó mal (el cliente lo ve como `status == failed`) | — (reservado) |
| `LLMUnavailableError` | El proveedor de IA está caído o tardó demasiado | 503 |

**¿Por qué tipos en vez de un `error` genérico?** Porque la capa que habla con internet (aún no construida) necesitará traducir cada error a una **respuesta HTTP distinta**. Con tipos concretos, esa capa puede preguntar "¿eres un `NotFoundError`?" y responder 404. Con un error genérico, tendría que adivinar. Esta idea se resume en el axioma **A5** y en el anti-patrón **AP3** (no reutilizar el mismo tipo para conceptos distintos).

> **Concepto — `error` (interfaz de Go)**: cualquier tipo que tenga un método `Error() string` "es" un error. Por eso cada struct del dominio define su `Error()`.
>
> **Concepto — Error tipado**: un error que es un tipo concreto (no un texto suelto), lo que permite distinguirlo en el código.
>
> **Concepto — `errors.As`**: función de Go que comprueba si un error es de un tipo concreto (p. ej. "¿este error es un `*InvalidStateError`?"). Se usa muchísimo en los tests.

#### 1.3 `events.go` — los "anuncios" de lo que ocurre

Un **evento de dominio** es un hecho pasado, inmutable, que ha ocurrido en el negocio: "se creó una práctica", "el análisis terminó", "el análisis falló". Se definen en la raíz (no en cada contexto) para que los contextos puedan reaccionar a ellos **sin importarse entre sí** (A3).

```go
// DomainEvent is a pure, immutable business fact (A4).
type DomainEvent interface {
	EventName() string
}

const EventNamePracticeCreated = "practice.created"

type PracticeCreated struct {
	PracticeID ID  `json:"practice_id"`
	UserID     ID  `json:"user_id"`
	Version    int `json:"version"`
}

func (PracticeCreated) EventName() string { return EventNamePracticeCreated }
```

Los cuatro eventos del MVP:

| Evento | Lo produce | Lo consumirá |
|---|---|---|
| `IdentityIssued` | `identity` | `practice` |
| `PracticeCreated` | `practice` | `analysis` (dispara el análisis) |
| `AnalysisCompleted` | `analysis` | `analytics` (actualiza estadísticas) |
| `AnalysisFailed` | `analysis` | `practice` (marca `failed`) |

**¿Por qué eventos y no llamadas directas?** Porque si `analysis` llamara directamente al código de `analytics`, los dos contextos quedarían pegados y romperían A3. Los eventos son "anuncios" que cada contexto escucha por separado. El campo `Version` permite cambiar el formato del anuncio en el futuro sin romper a quien lo escucha.

> **Concepto — Evento de dominio**: algo que ocurrió y que otros podrían querer saber.
>
> **Concepto — Inmutable**: que no se puede modificar una vez creado. Un evento, por definición, ya pasó.
>
> **Concepto — Outbox (avanzado)**: patrón (Fase 3) para guardar los eventos en la misma transacción que el cambio y publicarlos **después** de confirmar. Aquí solo se definen los eventos; su publicación es de otra fase.

#### 1.4 `error_pattern.go` — la taxonomía de errores (¡y por qué vive en la raíz!)

`ErrorPattern` describe *un* error gramatical: su **código** (qué tipo), su **gravedad** y una **nota** explicativa. Los códigos son una lista cerrada de 8:

```go
type ErrorPatternCode string

const (
	ErrorPatternCodeInfinitiveConjugation   ErrorPatternCode = "infinitive_conjugation"
	ErrorPatternCodePassiveVoiceMisuse      ErrorPatternCode = "passive_voice_misuse"
	// ... 6 más ...
)

func (c ErrorPatternCode) IsValid() bool { /* valida contra la lista */ }
```

**¿Por qué está en la raíz y no dentro de `analysis/`, como decía el checklist?** Es una decisión fina y elegante que conviene entender:

- `ErrorPattern` lo usan **dos** contextos: `analysis` (cuando la IA describe el error) y `analytics` (cuando cuenta errores por tipo).
- Además, el evento `AnalysisCompleted` (que vive en la raíz) **transporta una lista de `ErrorPattern`**.
- Si `ErrorPattern` viviera en `analysis/`, entonces `analytics` tendría que importar `analysis` (rompe A3), o la raíz tendría que importar `analysis` (rompe A2). **Imposible sin violar un axioma.**
- La solución: es un tipo **compartido** y por eso vive en la raíz, junto a `ID` y los errores.

> **Concepto — Enumerado (enum)**: un campo que solo admite unos pocos valores de una lista. Aquí el "tipo nominal" `type ErrorPatternCode string` obliga a usar constantes con nombre en vez de textos sueltos, y `IsValid()` comprueba que el valor pertenece a la lista.

### Bloque 2 — `identity/` (la ficha del estudiante)

**Archivos**: `identity/user.go` y `identity/email.go`.

```go
type User struct {
	ID        domain.ID `json:"id"`
	Email     Email     `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func NewUser(email Email, createdAt time.Time) (*User, error) {
	if email == "" {
		return nil, &domain.ValidationError{Field: "email", Message: "must not be empty"}
	}
	id, err := generateID()   // genera el UUID v7
	// ...
	return &User{ID: id, Email: email, CreatedAt: createdAt}, nil
}
```

- `User` es la **entidad** del estudiante: tiene `ID` (UUID v7), `Email` y `CreatedAt`.
- `Email` es un **value object** (`type Email string`) que se valida con `net/mail` de la librería estándar. Si el correo no es válido, `NewEmail` devuelve `ValidationError`.
- **Importante**: `User` **no** tiene `tenant_id` (identificador de "escuela" o inquilino). LangLint es de un solo usuario y su identidad es **portable** por diseño (A9). Meter un `tenant_id` aquí sería el anti-patrón AP1. Hay un test que verifica que el JSON del usuario **no** contiene ese campo.

**¿Por qué el correo es un objeto y no un simple texto?** Porque así es **imposible** construir un correo inválido: la única forma de tener un `Email` es pasar por `NewEmail`, que valida. Esto se llama "hacer ilegal lo imposible".

> **Concepto — PII (Personal Identifiable Information)**: datos personales (como el correo). El proyecto obliga a no registrarlos en logs en crudo (A8). Por eso `Email` es un tipo propio y cuidadoso.
>
> **Concepto — `net/mail`**: herramienta de la librería estándar de Go para validar direcciones de correo.
>
> **Concepto — Portable**: que la identidad no pertenece a ningún "inquilino" y puede viajar con el usuario (requisito del usuario dueño de sus datos, A9).

### Bloque 3 — `practice/` (la hoja de ejercicio)

**Archivos**: `practice.go`, `source_text.go`, `draft_text.go`, `target_rule.go`, `practice_status.go`.

#### 3.1 El agregado `Practice` y sus value objects

```go
type Practice struct {
	ID          domain.ID      `json:"id"`
	UserID      domain.ID      `json:"user_id"`
	SourceText  SourceText     `json:"source_text"`
	DraftText   DraftText      `json:"draft_text"`
	TargetRules []TargetRule   `json:"target_rules"`
	Status      PracticeStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   *time.Time     `json:"-"`   // soft-delete, no viaja al frontend
}
```

| Pieza | Tipo | Regla |
|---|---|---|
| `SourceText` | Value Object | No vacío. Conserva el texto tal cual. |
| `DraftText` | Value Object | No vacío. |
| `TargetRule` | Value Object | `{verb, tense, note}`; el `verb` es obligatorio. |
| `PracticeStatus` | Enumerado | `draft` / `analyzing` / `completed` / `failed`. |

#### 3.2 Las invariantes (las reglas que siempre se cumplen)

Una **invariante** es una condición que debe ser cierta siempre. En `NewPractice`:

```go
func NewPractice(userID domain.ID, source SourceText, draft DraftText, rules []TargetRule, createdAt time.Time) (*Practice, error) {
	if len(rules) == 0 {
		return nil, &domain.ValidationError{Field: "target_rules", Message: "must contain at least one rule"}
	}
	id, err := generateID()
	// ...
	return &Practice{ /* ... */ Status: PracticeStatusDraft, CreatedAt: createdAt, UpdatedAt: createdAt }, nil
}
```

Reglas del agregado:
1. Los textos no vacíos (lo garantizan los propios value objects).
2. **Al menos una** `TargetRule` (lo garantiza `NewPractice`).
3. La **transición de estado** solo puede ser `draft → analyzing → completed|failed`.

#### 3.3 La máquina de estados y sus métodos

```text
   draft ──StartAnalysis──► analyzing ──MarkCompleted──► completed
                                  └──── MarkFailed ────► failed
```

```go
func (p *Practice) StartAnalysis(now time.Time) error {
	if p.Status != PracticeStatusDraft {
		return &domain.InvalidStateError{Field: "status", Message: "must be draft to start analysis"}
	}
	p.Status = PracticeStatusAnalyzing
	p.UpdatedAt = now
	return nil
}
```

Cada método comprueba primero que el estado actual sea el correcto; si no, devuelve `InvalidStateError` (equivale a un 409: "no puedes hacer eso ahora").

#### 3.4 Editar y borrar (con matices importantes)

```go
func (p *Practice) Edit(source *SourceText, draft *DraftText, rules []TargetRule, now time.Time) error {
	if p.Status != PracticeStatusDraft {
		return &domain.InvalidStateError{Field: "status", Message: "must be draft to be edited"}
	}
	if rules != nil && len(rules) == 0 {
		return &domain.ValidationError{Field: "target_rules", Message: "must contain at least one rule"}
	}
	// Solo cambia lo que llega "no nulo"; un nil deja el campo intacto.
	p.SourceText = /* ... */
	// ...
}

func (p *Practice) Delete(now time.Time) error {
	if p.Status == PracticeStatusAnalyzing {
		return &domain.InvalidStateError{Field: "status", Message: "must not be analyzing to be deleted"}
	}
	if p.DeletedAt != nil {
		return &domain.InvalidStateError{Field: "deleted_at", Message: "practice is already deleted"}
	}
	p.DeletedAt = &now
	p.UpdatedAt = now
	return nil
}
```

- **`Edit` es una edición parcial** (como el verbo `PATCH` de internet): si un campo llega a `nil`, no se toca. Solo permitida en `draft`.
- **`Delete` es un "borrado suave" (soft-delete)**: no se destruye el dato; se apunta la fecha en `DeletedAt`. Un proceso posterior (Fase 6) lo purgará de verdad tras un tiempo de gracia (A8, retención limitada). Se prohíbe borrar mientras se está analizando.

**¿Por qué `InvalidStateError` y no `ValidationError`?** Porque son conceptos distintos (AP3): "los datos que me diste son inválidos" (422) no es lo mismo que "los datos son válidos, pero no en este estado" (409). Cada concepto merece su propio error.

> **Concepto — Agregado raíz**: el objeto principal que controla un grupo de datos y protege sus reglas. Se entra a los demás objetos a través de él.
>
> **Concepto — Invariante**: condición que nunca debe dejar de cumplirse. Si una operación la rompería, la operación se rechaza.
>
> **Concepto — Transición de estado**: el paso de un estado a otro (borrador → analizando...). No todas las transiciones son válidas.
>
> **Concepto — Soft-delete**: "marcar como borrado" sin borrar físicamente, para poder recuperarlo o purgarlo luego.
>
> **Concepto — Constructor (`New...`)**: función que crea un objeto ya válido. En este proyecto, casi nunca se crea un objeto "a pelo"; se pasa por su constructor.

### Bloque 4 — `analysis/` (la corrección de la IA)

**Archivos**: `analysis.go`, `fragment.go`, `analysis_status.go`.

```go
type Analysis struct {
	ID           domain.ID      `json:"id"`
	PracticeID   domain.ID      `json:"practice_id"`
	Fragments    []Fragment     `json:"fragments"`
	Model        string         `json:"model"`
	ModelVersion string         `json:"model_version"`
	Status       AnalysisStatus `json:"status"`
	CreatedAt    time.Time      `json:"created_at"`
}
```

- **`PracticeID` es solo un identificador**, no un objeto `Practice`. Es clave: `analysis` **no importa** el paquete `practice` (A3). Se relacionan "por número", no "cogiéndose de la mano".
- `Fragment` es el trozo de análisis (7 campos: frase en español, borrador, corrección, revisión del verbo, aclaración léxica, explicación gramatical y lista de `ErrorPattern`). Es un **struct plano**: no valida nada porque ese texto lo produce la IA y lo validará otro componente contra un esquema.
- `AnalysisStatus`: `pending` / `completed` / `failed`.

La invariante más importante:

```go
func (a *Analysis) Complete(fragments []Fragment) error {
	if a.Status != AnalysisStatusPending {
		return &domain.InvalidStateError{Field: "status", Message: "must be pending to be completed"}
	}
	if len(fragments) == 0 {
		return &domain.ValidationError{Field: "fragments", Message: "must not be empty when completed"}
	}
	a.Fragments = fragments
	a.Status = AnalysisStatusCompleted
	return nil
}
```

**¿Por qué "no vacío cuando completed"?** Porque un análisis "terminado" sin ninguna frase analizada sería un resultado falso. La regla impide que exista ese estado incoherente.

> **Concepto — Referencia por ID**: relacionar dos objetos guardando solo su identificador. Es la forma de que dos "mundos" colaboren sin acoplarse.

### Bloque 5 — `analytics/` (el cuaderno de estadísticas)

**Archivos**: `window.go`, `error_metric.go`, `progress_metric.go`.

- `Window` es un enumerado de la ventana de tiempo: `day` / `week` / `month`.
- `ErrorMetric` cuenta **cuántas veces** aparece cada tipo de error. No tiene ID propio: se identifica por la combinación `(UserID, Code, Window)` (su "clave natural"). Nace con `Count = 1` y el método `Record` lo incrementa:

```go
func NewErrorMetric(userID domain.ID, code domain.ErrorPatternCode, window Window, lastSeenAt time.Time) (*ErrorMetric, error) {
	if !code.IsValid()   { return nil, &domain.ValidationError{Field: "code", ...} }
	if !window.IsValid() { return nil, &domain.ValidationError{Field: "window", ...} }
	return &ErrorMetric{UserID: userID, Code: code, Window: window, Count: 1, LastSeenAt: lastSeenAt}, nil
}

func (m *ErrorMetric) Record(now time.Time) {
	m.Count++
	m.LastSeenAt = now
}
```

- `ProgressMetric` guarda el progreso: total de frases, número de errores y una **precisión** (`Accuracy`) derivada.

```go
func accuracy(totalFragments, errorCount int) float64 {
	if totalFragments == 0 {
		return 1   // sin frases, no hay nada que estar mal
	}
	value := 1 - float64(errorCount)/float64(totalFragments)
	if value < 0 {
		return 0   // no permitimos precisiones negativas
	}
	return value
}
```

**¿Por qué el "clamp"?** Porque un fragmento puede contener **varios** errores, así que `errorCount` puede superar a `totalFragments` y la resta daría negativa. Como el contrato exige que la precisión esté entre `0` y `1`, se "recorta" (`clamp`) para que nunca salga del rango. Y si no hay frases, se define precisión `1`.

> **Concepto — Clave natural**: identificar un registro por sus datos (usuario + código + ventana) en vez de por un ID nuevo.
>
> **Concepto — Upsert**: "insertar o actualizar". El handler del evento buscará si ya existe la métrica y, si existe, la incrementará.
>
> **Concepto — Clamp**: forzar un valor a quedarse dentro de un rango mínimo/máximo.
>
> **Concepto — División por cero**: error típico al dividir entre 0. Aquí se evita con el caso especial `totalFragments == 0`.

### Bloque 6 — La verificación (el "gate de salida")

Cada fase termina con una **puerta** que debe estar en verde antes de avanzar. En Fase 2 se comprobó:

| Comando | Qué comprueba |
|---|---|
| `go test -race -count=1 ./...` | Que todos los tests pasan, **incluso con concurrencia** (detector de carreras). |
| `go test -cover ./internal/domain/...` | Que el dominio tiene **100%** de cobertura. |
| `go build ./...` / `go vet ./...` | Que compila y que no hay errores sospechosos. |
| `gofmt -l` | Que el formato del código es el oficial. |
| `grep` de imports | Que ningún archivo del dominio usa librerías externas (A1). |
| `go list` de paquetes | Que ningún contexto importa a otro (A3). |

**El truco para llegar al 100% de cobertura: los "seams".** Una variable como `generateID = domain.NewID` permite, en un test, **sustituirla** por una versión que falla a propósito. Así se prueba también la rama de error ("¿qué pasa si no se puede generar el ID?"), que de otro modo nunca se ejecutaría.

```go
var generateID = domain.NewID   // "costura" que en tests se puede cambiar
```

Los tests siguen una convención de nombre muy estricta: `TestXxx_Método_Condición_ResultadoEsperado` (por ejemplo, `TestPractice_Delete_Analyzing_ReturnsInvalidStateError`). El nombre **documenta** qué se está probando.

> **Concepto — Cobertura (coverage)**: porcentaje de líneas de código que son ejecutadas por los tests. El dominio exige el 100%.
>
> **Concepto — Race detector (`-race`)**: herramienta que detecta errores cuando varios hilos de ejecución tocan los mismos datos a la vez.
>
> **Concepto — Seam ("costura")**: punto del código preparado para sustituir una pieza (p. ej. el generador de IDs) durante un test.
>
> **Concepto — Gate de salida**: criterio objetivo que marca cuándo una fase está terminada. Evita avanzar con cimientos a medias.

---

## 3. Resumen de los cambios

- **Raíz del dominio (`2.5`)**: se construyeron las piezas compartidas: `ID` (UUID v7 generado solo con la librería estándar), los 6 errores tipados (`ValidationError`, `NotFoundError`, `AnalysisPendingError`, `AnalysisFailedError`, `InvalidStateError`, `LLMUnavailableError`), los 4 eventos (`IdentityIssued`, `PracticeCreated`, `AnalysisCompleted`, `AnalysisFailed`) y la taxonomía `ErrorPattern` con sus enums.
- **Los 4 bounded contexts**: `identity/` (`User` portable sin `tenant_id` + VO `Email`), `practice/` (agregado `Practice` + VOs `SourceText`/`DraftText`/`TargetRule`/`PracticeStatus`), `analysis/` (agregado `Analysis` + `Fragment` + `AnalysisStatus`) y `analytics/` (`Window`, `ErrorMetric`, `ProgressMetric`).
- **Reglas y verificación**: se implementaron las invariantes (textos no vacíos, ≥1 regla, transiciones `draft → analyzing → completed|failed`, "fragments no vacío al completar"), `Edit`/`Delete` con `InvalidStateError`, y el **Gate de salida quedó en verde**: 100% de cobertura del dominio, tests con `-race`, pureza A1 (solo librería estándar) y aislamiento A3 (ningún contexto importa a otro).
