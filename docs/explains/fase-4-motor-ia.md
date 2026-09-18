# Fase 4 — Motor de Análisis Cognitivo (IA): explicación didáctica

> Documento de estudio sobre todo lo construido en la **Fase 4 (Motor de IA)** del proyecto LangLint.
> Cubre los bloques `4.1` (puerto `LLMExtractor` + `OpenAIExtractor` + prompt), `4.2` (Structured Outputs, validación y `AnalysisService`) y `4.3` (anonimización, timeout/`AnalysisFailed` y `PIIHandler`).
> Es la continuación de `docs/explains/fase-3-puertos-adaptadores.md`.

---

## 1. ¿De qué trata esto? (Explicación para no técnicos)

En la Fase 2 fabricaste las **piezas y el reglamento** del juego. En la Fase 3 montaste el **almacén, los enchufes y el cartero** para que las piezas se guardaran y se avisaran entre sí. Ahora, en la Fase 4, llega el momento de **contratar al profesor corrector**: la inteligencia artificial que de verdad corrige los textos del alumno.

La imagen es la de una **academia con un tutor externo**. El tutor es un experto buenísimo, pero vive **en otro edificio** y cobra por cada consulta. Por eso hay que tratarlo con cuidado y con reglas muy claras:

| Imagen de la vida real | Qué representa en el proyecto |
|---|---|
| **El tutor externo en otro edificio** | El **LLM** (el modelo de lenguaje de OpenAI). Es un servicio de fuera; no sabemos (ni queremos saber) cómo funciona por dentro. |
| **El secretario con rotulador negro** | El **anonimizador**: antes de sacar el ejercicio de casa, tacha nombres, correos y teléfonos para que el tutor no sepa de quién es el texto (privacidad, A8). |
| **Un formulario con casillas fijas** | El **JSON Schema** de *Structured Outputs*: obliga al tutor a devolver la corrección siempre con la misma forma, rellenando todas las casillas. |
| **El revisor que comprueba el formulario** | La **validación**: antes de archivar la corrección, se verifica que no falte ninguna casilla ni haya valores fuera de la lista permitida. |
| **El cajón que no se deja abierto** | La regla de que la llamada al LLM ocurre **fuera de la transacción**: no dejamos el almacén bloqueado mientras el tutor piensa. |
| **El aviso "corrección fallida"** | El evento **`AnalysisFailed`**: si el tutor no contesta a tiempo, se anota el fallo y se avisa, **sin contar los detalles crudos** del error. |
| **La trituradora de papel en la puerta** | El **`PIIHandler`**: destruye cualquier nota que contenga datos personales antes de guardarla en el registro de actividad (los *logs*). |

**El flujo completo, contado como una anécdota:**

1. El alumno deja su ejercicio (texto en español + borrador en inglés) y pide corrección.
2. El secretario **tacha los datos personales** del ejercicio antes de sacarlo de casa.
3. Se le entrega al tutor con una **rúbrica muy precisa** (el *prompt*) y un **formulario en blanco** (el esquema de salida).
4. El tutor puede tardar; mientras tanto, en casa **nadie bloquea el almacén**.
5. Cuando llega la respuesta, el revisor **comprueba que el formulario esté bien rellenado**. Si hay una casilla vacía o un código inventado, se descarta como si el tutor hubiera fallado.
6. Si todo está bien, se archiva la corrección **y** se deja el aviso "corrección terminada" en el cuaderno de recados, **los dos en la misma operación** (para que nunca se pierda uno sin el otro).
7. Si el tutor no responde a tiempo, se archiva "corrección fallida" y se deja el aviso correspondiente. Nunca se copian los detalles vergonzosos del error.

**Resultado de la Fase 4**: el sistema ya es capaz de **hablar con la IA y obtener una corrección estructurada**, protegiendo los datos personales en el camino y sin romper ninguna de las reglas arquitectónicas construidas en las fases anteriores. Todavía no hay pantallas para el usuario (eso es la Fase 5), pero el "motor" ya funciona de punta a punta y está probado contra la IA real.

---

## 2. Guía de Estudio (Para entender qué hizo la IA)

Vamos por bloques. Para cada uno veremos **qué se creó**, **cómo funciona** y, sobre todo, **por qué** se hizo así. La idea es que puedas leer el código y reconocer cada decisión.

### Bloque 0 — El puerto `LLMExtractor`: el dominio no debe conocer a OpenAI

**Recuerda de la Fase 3** (arquitectura de puertos y adaptadores):

```text
   Fuera  ───────────────────────────────────────────────►  Dentro
   cmd → adapters → services → ports → domain
   (arranque)  (OpenAI/DB)  (casos de uso)  (interfaces)  (REGLAS PURAS)
```

- Un **puerto** es una interfaz (un "enchufe estándar") que dice *qué* operaciones existen, sin decir *cómo*.
- Un **adaptador** es la implementación concreta para una tecnología.

Aquí el puerto es `LLMExtractor` y el adaptador es `OpenAIExtractor` (usa el SDK oficial `openai-go`).

```go
// apps/backend/internal/api/ports/llm_extractor.go (conceptual)
package ports

type ExtractRequest struct {
	PracticeID  domain.ID
	SourceText  string   // ya anonimizado
	DraftText   string   // ya anonimizado
	TargetRules []practice.TargetRule
}

type LLMExtractor interface {
	Extract(ctx context.Context, req ExtractRequest) ([]analysis.Fragment, error)
	Model() string
	ModelVersion() string
}
```

El adaptador concreto declara explícitamente que cumple el contrato:

```go
// apps/backend/internal/api/adapters/llm/openai_extractor.go
var _ ports.LLMExtractor = (*OpenAIExtractor)(nil)
```

**¿Por qué tanta ceremonia?** Por el axioma **A1**: el dominio (`internal/domain/`) es la capa más estable y **solo puede usar la librería estándar de Go**. Si mañana se cambia OpenAI por otro proveedor (Anthropic, un modelo local...), las reglas del negocio **no se tocan**: solo se escribe otro adaptador que cumpla el mismo puerto. Los tests lo comprueban con un comando concreto:

```text
go list -deps ./internal/domain/... | grep -c openai   →   debe dar 0
```

**`Model()` y `ModelVersion()`** se añadieron en `4.2` porque el `Analysis` guarda con qué modelo se corrigió (trazabilidad). Como el servicio solo conoce el **puerto**, el puerto mismo debe exponer esos dos datos.

> **Concepto — Puerto (port)**: interfaz que define un conjunto de operaciones. Es el "enchufe" que el resto del código conoce.
>
> **Concepto — Adaptador (adapter)**: implementación real del puerto para una tecnología concreta (aquí, el SDK de OpenAI).
>
> **Concepto — Inyección por puerto**: quien usa la funcionalidad no crea el adaptador; se lo "inyectan" desde fuera (en el arranque del programa). Por eso `AnalysisService` no importa `openai` en ningún sitio.

### Bloque 1 — El prompt: las instrucciones para el tutor (`4.1`)

Un **prompt** es el texto de instrucciones que se le envía al modelo. Se construye en `adapters/llm/prompt.go` con una función **pura** (sin red, sin efectos), lo que la hace fácil de testear y auditar.

```go
// apps/backend/internal/api/adapters/llm/prompt.go (conceptual)
type prompt struct {
	System string // "quién eres y cómo debes responder"
	User   string // "este es el ejercicio concreto"
}

func buildPrompt(req ports.ExtractRequest) prompt {
	return prompt{
		System: fmt.Sprintf(systemPrompt, errorPatternCatalog(), joinValues(errorPatternSeverities)),
		User:   buildUserPrompt(req),
	}
}
```

El **system prompt** es el "contrato de salida": le dice al modelo que actúe como profesor nativo, que trabaje **frase por frase**, que copie los fragmentos **verbatim** (`source_es`, `user_draft` tal cual), que escriba las explicaciones **en español** (el idioma nativo del alumno) pero la corrección en inglés, y que use **solo** los códigos de la taxonomía.

El **user prompt** incluye los datos del ejercicio: el texto fuente, el borrador y las reglas objetivo.

Dos detalles finos que merecen atención:

1. **El glosario de la taxonomía se deriva de las constantes del dominio**, no de un texto escrito a mano:

   ```go
   var errorPatternDescriptions = []struct {
       code        domain.ErrorPatternCode
       description string
   }{
       {domain.ErrorPatternCodeInfinitiveConjugation, "confusion between an infinitive and a conjugated verb"},
       // ...
       {domain.ErrorPatternCodeLexicalChoice, "wrong vocabulary choice or collocation that is not a false friend"},
       // ...
   }
   ```

   Así, si mañana se añade un código nuevo al dominio, el prompt **no puede quedarse desactualizado**.

2. **`PromptText`** es una función exportada que devuelve el prompt exacto **sin llamar a la API**. Sirve para auditarlo (y la usa la herramienta de desarrollo `cmd/llmcheck` con la opción `-show-prompt`).

**¿Por qué el prompt es una función pura?** Porque no queremos gastar dinero ni depender de la red para comprobar que el texto de instrucciones es correcto. Los tests verifican que el prompt contiene la taxonomía y los datos del ejercicio **sin abrir una conexión**.

> **Concepto — Prompt**: el texto de instrucciones que recibe un modelo de lenguaje. Es "el enunciado del problema".
>
> **Concepto — System vs User**: el mensaje *system* define el **papel y las reglas** (estable); el mensaje *user* trae **los datos concretos** de esta petición (variable).
>
> **Concepto — Función pura**: una función que, dados los mismos datos, siempre devuelve lo mismo y no produce efectos secundarios (no llama a la red, no escribe ficheros).

### Bloque 2 — Structured Outputs: el formulario con casillas (`4.2.1`)

Si solo pidiéramos "devuélveme un JSON", el modelo podría devolver el JSON con otra forma, olvidar campos o inventar códigos de error. Para evitarlo, se usa **Structured Outputs** de OpenAI: se le entrega al modelo un **JSON Schema** (un "formulario con casillas") y se activa el modo `strict: true`, que le obliga a respetarlo.

```go
// apps/backend/internal/api/adapters/llm/schema.go (conceptual)
func fragmentResponseFormat() openai.ChatCompletionNewParamsResponseFormatUnion {
	return openai.ChatCompletionNewParamsResponseFormatUnion{
		OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
			JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        fragmentResponseSchemaName, // "fragment_analysis"
				Description: openai.String(fragmentResponseSchemaDescription),
				Schema:      fragmentSchema(),
				Strict:      openai.Bool(true), // no acepta inventos
			},
		},
	}
}
```

Y `Extract` lo envía en la petición:

```go
completion, err := e.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
	Model:          e.model,
	Messages:       []openai.ChatCompletionMessageParamUnion{ /* system + user */ },
	ResponseFormat: fragmentResponseFormat(), // <- el formulario
})
```

#### 2.1 La lección del "root object" (¡importante!)

En la primera versión, el esquema raíz era directamente una **lista** de fragmentos (`type: "array"`). Al probarlo contra la IA real, la API respondió:

```text
400 Bad Request: Invalid schema for response_format 'fragment_analysis':
schema must be a JSON Schema of 'type: "object"', got 'type: "array"'.
```

**Structured Outputs exige que la raíz del esquema sea un objeto**, no un array. La solución fue **envolver** la lista en un objeto:

```json
{
  "type": "object",
  "properties": {
    "fragments": {
      "type": "array",
      "items": { /* ... el esquema de un Fragment ... */ }
    }
  },
  "required": ["fragments"],
  "additionalProperties": false
}
```

Y el adaptador "desenvuelve" la respuesta con un tipo auxiliar:

```go
type fragmentEnvelope struct {
	Fragments []analysis.Fragment `json:"fragments"`
}

var envelope fragmentEnvelope
json.Unmarshal(..., &envelope)
return envelope.Fragments, nil
```

**Lo importante**: el contrato público (el *wire format*) sigue siendo `Fragment[]`; el envoltorio es un detalle **interno** del diálogo con el modelo. Esta es una lección que conviene recordar: *las APIs externas tienen sus propias reglas y, a veces, hay que adaptar la forma sin cambiar el modelo de negocio.*

#### 2.2 Los enums del esquema salen del dominio

Las casillas `code` y `severity` de cada error no son texto libre: son listas cerradas que se construyen a partir de las constantes del dominio.

```go
"code":     map[string]any{"type": "string", "enum": errorPatternCodeEnum()},
"severity": map[string]any{"type": "string", "enum": severityEnum()},
```

`errorPatternCodeEnum()` recorre `errorPatternDescriptions` (el mismo catálogo del prompt) y `severityEnum()` recorre `errorPatternSeverities`. De nuevo: **una sola fuente de verdad**, cero posibilidad de *drift* (desincronización).

> **Concepto — Structured Outputs**: modo de un LLM que garantiza que la respuesta JSON cumple un esquema predefinido.
>
> **Concepto — JSON Schema**: un "plano" que describe la forma válida de un JSON (qué campos hay, de qué tipo, cuáles son obligatorios, qué valores admite cada enum).
>
> **Concepto — `strict: true`**: activa el modo estricto; el modelo está obligado a rellenar exactamente las casillas del esquema. Exige que **todos** los campos estén en `required` y que `additionalProperties` sea `false`.
>
> **Concepto — `enum`**: una casilla que solo admite un conjunto cerrado de valores (p. ej. `minor | moderate | critical`).
>
> **Concepto — Drift (deriva)**: cuando dos partes del sistema que deberían coincidir se desincronizan sin que nadie se dé cuenta. Aquí se combate derivando todo de una única fuente.

### Bloque 3 — La validación: el revisor comprueba el formulario (`4.2.2`)

Aunque el modo estricto empuja al modelo a responder bien, **nunca se confía ciegamente en él**. Antes de persistir nada, el adaptador valida la respuesta en `adapters/llm/validate.go`.

```go
// apps/backend/internal/api/adapters/llm/validate.go
func validateFragments(fragments []analysis.Fragment) error {
	for _, fragment := range fragments {
		if err := validateFragment(fragment); err != nil {
			return err
		}
	}
	return nil
}

func validateFragment(fragment analysis.Fragment) error {
	required := []string{
		fragment.SourceES,
		fragment.UserDraft,
		fragment.Correction,
		fragment.TargetVerbReview,
		fragment.LexicalClarification,
		fragment.GrammarExplanation,
	}
	for _, value := range required {
		if strings.TrimSpace(value) == "" {
			return invalidOutput()
		}
	}
	for _, pattern := range fragment.ErrorPatterns {
		if !pattern.Code.IsValid() || !pattern.Severity.IsValid() {
			return invalidOutput()
		}
	}
	return nil
}
```

Se comprueban dos cosas:

1. Que **ningún campo de texto obligatorio esté vacío**.
2. Que **cada error use un código y una gravedad de la lista del dominio** (`IsValid()`).

Si algo falla, se convierte en un error de dominio genérico:

```go
func invalidOutput() error {
	return &domain.LLMUnavailableError{Message: "llm unavailable"}
}
```

**¿Por qué no usar una librería externa de validación de JSON Schema?** Porque el manifiesto del proyecto prohíbe paquetes externos de validación (se prioriza la librería estándar). La validación semántica en Go (campos no vacíos + enums válidos) cubre lo que de verdad nos importa y mantiene el sistema ligero.

**¿Por qué todo error de salida inválida se trata como "IA no disponible"?** Porque, desde el punto de vista del negocio, una respuesta inservible del modelo es equivalente a que el modelo haya fallado. Así el resto del sistema solo tiene que saber manejar **una** situación ("no pude obtener una corrección válida"), y nunca se persiste "basura".

> **Concepto — Validar antes de persistir**: comprobar que un dato es correcto **antes** de guardarlo. Si se guarda primero y se valida después, ya hay datos corruptos en el almacén.
>
> **Concepto — `IsValid()`**: método que comprueba si un valor pertenece a una lista cerrada. Es la forma de blindar los enumerados.

### Bloque 4 — El `AnalysisService`: orquestar sin romper las reglas (`4.2.3`, `4.2.4`, `4.3.3`)

Este es el corazón de la Fase 4. El `AnalysisService` (`services/analysis_service.go`) es el **caso de uso** que une todas las piezas: cargar la práctica, anonimizarla, pedirle la corrección a la IA, validar y guardar el resultado.

```go
// apps/backend/internal/api/services/analysis_service.go (conceptual)
func (s *AnalysisService) RunAnalysis(ctx context.Context, practiceID domain.ID) error {
	// 1) Cargar la práctica
	p, err := s.practices.GetByID(ctx, practiceID)
	if err != nil {
		return err
	}

	// 2) Crear el Analysis en estado "pending" (aún sin corregir)
	result, err := analysis.NewAnalysis(practiceID, s.extractor.Model(), s.extractor.ModelVersion(), s.now())
	if err != nil {
		return err
	}

	// 3) Llamar al LLM con un límite de tiempo... ¡FUERA de la transacción!
	llmCtx, cancel := context.WithTimeout(ctx, s.llmTimeout)
	defer cancel()

	fragments, err := s.extractor.Extract(llmCtx, ports.ExtractRequest{
		PracticeID:  practiceID,
		SourceText:  AnonymizeSpanish(p.SourceText.String()), // privacidad
		DraftText:   Anonymize(p.DraftText.String()),
		TargetRules: p.TargetRules,
	})
	if err != nil {
		return s.failAnalysis(ctx, result, reasonLLMUnavailable)
	}

	// 4) Validar el resultado y marcarlo como completado
	if err := result.Complete(fragments); err != nil {
		return s.failAnalysis(ctx, result, reasonInvalidResult)
	}

	// 5) Guardar el Analysis y dejar el aviso, TODO en la misma transacción
	event := domain.AnalysisCompleted{ /* ... */ }
	return s.uow.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.analyses.Save(txCtx, result); err != nil {
			return err
		}
		return s.outbox.Append(txCtx, event)
	})
}
```

Hay **cuatro decisiones de diseño** que conviene entender bien:

#### 4.1 El LLM se llama FUERA de la transacción (`4.2.3`)

Una transacción de base de datos es como un "cajón abierto": mientras está abierta, bloquea recursos. La llamada al LLM es **lenta** (puede tardar segundos). Si la metiéramos dentro, tendríamos el cajón abierto todo ese tiempo. Por eso se llama primero al LLM y solo **después** se abre la transacción para guardar. Esto implementa el anti-patrón **AP7/AP8** al revés: nunca se hace trabajo lento dentro de la transacción.

#### 4.2 Guardado atómico + outbox (`4.2.4`)

El `Analysis` (el resultado) y el aviso `AnalysisCompleted` se guardan **en la misma transacción** (`InTransaction`). Si una de las dos cosas falla, ninguna se guarda. Así es **imposible** tener una corrección guardada cuyo aviso se haya perdido, o un aviso sin su corrección. Esto es el **patrón outbox**, explicado en la Fase 3.

> **Ojo con la regla clave**: nunca se llama al "cartero" (`dispatcher.Dispatch`) dentro de la transacción; solo se **anota el aviso** (`outbox.Append`). El cartero lo repartirá después.

#### 4.3 El fallo es un "estado terminal", no un error que se retorna

Cuando el LLM falla, `RunAnalysis` **no** devuelve el error hacia arriba: en su lugar, marca el `Analysis` como `failed`, lo guarda y emite `AnalysisFailed`, y **retorna `nil`**.

```go
func (s *AnalysisService) failAnalysis(ctx context.Context, result *analysis.Analysis, reason string) error {
	result.Fail() // pending -> failed

	event := domain.AnalysisFailed{
		AnalysisID: result.ID,
		PracticeID: result.PracticeID,
		Reason:     reason, // siempre genérico, nunca el error crudo
		Version:    analysisEventVersion,
	}

	return s.uow.InTransaction(ctx, func(txCtx context.Context) error {
		if err := s.analyses.Save(txCtx, result); err != nil {
			return err
		}
		return s.outbox.Append(txCtx, event)
	})
}
```

**¿Por qué?** Porque el "cartero" (el *outbox relay*) entrega los avisos **al menos una vez** y **reintenta** los que fallan. Si devolviéramos el error, el relay pensaría "esto no se ha completado, lo intento otra vez" y reintentaría **para siempre** un fallo que ya es definitivo. Al convertirlo en un estado terminal (`failed`) + un aviso nuevo (`AnalysisFailed`), el sistema lo da por cerrado y sigue con su vida.

#### 4.4 Trazabilidad del modelo

`analysis.NewAnalysis(...)` recibe `s.extractor.Model()` y `s.extractor.ModelVersion()`. Cuando el service depende del **puerto**, esos dos datos vienen del adaptador inyectado: si mañana cambia el proveedor, el `Analysis` seguirá guardando con qué modelo se corrigió, sin que el dominio sepa qué es "OpenAI".

> **Concepto — Orquestación**: coordinar varias piezas (cargar datos, llamar a un servicio, validar, persistir) siguiendo un orden correcto.
>
> **Concepto — Unit of Work (UoW)**: una operación "todo o nada". Se pasa una función y el UoW la ejecuta dentro de una única transacción.
>
> **Concepto — Outbox**: patrón para guardar el cambio **y** el aviso en la misma transacción; un proceso externo (el *relay*) los publica después.
>
> **Concepto — Idempotencia**: propiedad por la que ejecutar una operación dos veces tiene el mismo efecto que ejecutarla una vez. Los handlers del outbox deben ser idempotentes.
>
> **Concepto — Estado terminal**: un estado del que ya no se sale (por ejemplo `failed` o `completed`). Volver a intentarlo no tiene sentido.

### Bloque 5 — La anonimización: el rotulador negro (`4.3.1`)

Antes de sacar el texto de casa (mandarlo al LLM), hay que **borrar los datos personales**. Eso hace `services/anonymizer.go`. En vez de *eliminar* los datos (lo que dejaría frases cojas y confundiría al corrector), los **sustituye por etiquetas neutrales**:

| Dato personal | Se convierte en |
|---|---|
| `maria@example.com` | `[email]` |
| `+34 600 123 456` | `[phone]` |
| `Carmen` (nombre de la lista) | `[name]` |
| `Barcelona` (mayúscula a mitad de frase) | `[name]` |

```go
func Anonymize(text string) string {
	return redactNames(redactPhones(redactEmails(text)))
}

func AnonymizeSpanish(text string) string {
	return redactCapitalized(Anonymize(text)) // heurística extra para el español
}
```

Hay **dos funciones** porque el español y el inglés se comportan distinto con las mayúsculas:

- **`Anonymize`** (para el borrador en inglés): aplica emails, teléfonos y una **lista curada** de nombres comunes (españoles e ingleses).
- **`AnonymizeSpanish`** (para el texto fuente en español): además, redacta palabras **capitalizadas a mitad de frase** (que en español son casi siempre nombres propios, porque los sustantivos comunes van en minúscula).

**¿Por qué la lista de nombres solo redacta los que empiezan en mayúscula?** Porque hay palabras comunes que coinciden con nombres (el verbo inglés *mark* y el nombre *Mark*). Si redactáramos cualquier coincidencia, corromperíamos el borrador con falsos positivos. Esa mejora se detectó justo en una **revisión manual** de esta fase.

**¿Por qué solo la heurística de mayúsculas en español?** Porque en inglés se capitalizan muchas palabras legítimas ("English", "Monday"): aplicarla al borrador inglés borraría palabras normales y estropearía la corrección.

> **Concepto — Anonimización**: eliminar o sustituir los datos personales de un texto antes de enviarlo a un tercero.
>
> **Concepto — PII**: *Personally Identifiable Information*; datos que permiten identificar a una persona (nombre, correo, teléfono...).
>
> **Concepto — Token de redacción**: la etiqueta que sustituye al dato (`[name]`, `[email]`...). Conserva la estructura de la frase para que el corrector siga entendiéndola.
>
> **Concepto — Heurística**: una regla práctica y aproximada (no infalible) que resuelve bien la mayoría de los casos. Aquí: "mayúscula a mitad de frase en español ⇒ probable nombre propio".
>
> **Concepto — Falso positivo**: marcar como PII algo que no lo es. En privacidad, un falso negativo (dejar pasar un dato real) es peor que un falso positivo; aun así, se busca no estropear el texto.

### Bloque 6 — El timeout y la degradación (`4.3.3`)

Una llamada a un servicio externo **puede quedarse colgada**. Para que el sistema no espere indefinidamente, se le pone un **límite de tiempo** con `context.WithTimeout`:

```go
const defaultLLMTimeout = 60 * time.Second
// ...
llmCtx, cancel := context.WithTimeout(ctx, s.llmTimeout)
defer cancel()
fragments, err := s.extractor.Extract(llmCtx, ...)
```

- `context.WithTimeout` crea un contexto "hijo" que se **cancela solo** al cabo del tiempo.
- El `defer cancel()` libera los recursos del temporizador en cuanto la función termina.
- El adaptador de OpenAI propaga ese contexto al SDK, que aborta la petición al expirar.

Si el tiempo se agota (o el proveedor falla), caemos en el camino de `failAnalysis` del Bloque 4: se archiva el fallo y se emite `AnalysisFailed` con un **motivo genérico**.

**¿Por qué el error crudo nunca se filtra?** Por dos razones: seguridad (los detalles internos no deben salir) y experiencia de usuario (al usuario se le mostrará un mensaje claro, no un texto técnico). El axioma **A5** y las reglas de PII (**A8**) lo exigen.

> **Concepto — `context.Context`**: objeto de Go que transporta información de la petición (cancelación, plazos, valores) a través de las capas.
>
> **Concepto — Deadline / Timeout**: el "plazo máximo" que se le da a una operación antes de cancelarla.
>
> **Concepto — Degradación (graceful degradation)**: cuando una parte externa falla, el sistema no se cae: responde de una forma controlada y clara.

### Bloque 7 — El `PIIHandler`: la trituradora de papel (`4.3.4`)

Los *logs* (el registro de actividad del sistema) son otro punto por el que se pueden escapar datos personales. Para evitarlo, se creó `shared/logger/pii_handler.go`: un **decorador** que envuelve el escribidor de logs y **descarta** cualquier mensaje que huela a PII.

```go
// apps/backend/internal/shared/logger/pii_handler.go (conceptual)
type PIIHandler struct {
	inner slog.Handler
}

func (h *PIIHandler) Handle(ctx context.Context, record slog.Record) error {
	if containsPII(record) {
		return nil // se descarta el registro
	}
	return h.inner.Handle(ctx, record)
}
```

`containsPII` inspecciona el mensaje y **todos los atributos** (incluso dentro de grupos), y considera sensible un valor si:

- Es **texto libre de más de 100 caracteres** (el contenido largo suele ser texto del usuario).
- Contiene un **email**, un **teléfono**, una **API key** (`sk-...`, `AKIA...`) o un **JWT** (`eyJ...`).

Además, `WithAttrs` **elimina** los atributos sensibles antes de adjuntarlos, porque un valor "enganchado" al logger (`logger.With(...)`) se colaría en todos los mensajes futuros:

```go
func (h *PIIHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		if !attrIsSensitive(attr) {
			clean = append(clean, attr)
		}
	}
	return &PIIHandler{inner: h.inner.WithAttrs(clean)}
}
```

**¿Por qué un "decorador"?** Porque el logger estándar de Go (`log/slog`) está diseñado para permitir envolver unos manejadores dentro de otros. El `PIIHandler` se coloca **delante** de cualquier logger real: así, la protección es centralizada y no hay que acordarse de "limpiar" cada mensaje. Como beneficio extra, sigue usando solo la **librería estándar** (nada de `zap` ni similares).

> **Concepto — `slog`**: el paquete de *logging* estructurado de la librería estándar de Go. Registra eventos como pares clave-valor.
>
> **Concepto — `slog.Handler`**: la interfaz que decide qué hacer con cada registro (escribirlo, filtrarlo, darle formato...).
>
> **Concepto — Decorador (decorator pattern)**: envolver un objeto con otro que añade comportamiento (aquí, filtrar PII) sin que el objeto original se entere.
>
> **Concepto — API key / JWT**: credenciales de acceso. Un JWT es un "pase" firmado; una API key es una "llave" de servicio. Filtran seguridad si aparecen en un log.

### Bloque 8 — La verificación (el "gate de salida")

Cada fase cierra con una **puerta** que debe estar en verde. En la Fase 4 se comprobó:

| Comando | Qué comprueba |
|---|---|
| `go test -race -count=1 ./...` | Todos los tests pasan, incluso detectando *race conditions*. |
| `go test -race -count=1 -tags=integration ./...` | Los tests que necesitan Postgres real (testcontainers) siguen verdes. |
| `go test -cover ./internal/api/adapters/llm/` | El adaptador del LLM al **100%**. |
| `go test -cover ./internal/api/services/` | Los servicios ≥ **90%** (quedó en 96.9–97.1%). |
| `go test -cover ./internal/shared/logger/` | El `PIIHandler` al **100%**. |
| `go list -deps ./internal/domain/... \| grep -c openai` | A1: el dominio no conoce a OpenAI (debe dar `0`). |
| `grep "go func" internal/` | AP6: no hay *goroutines* propias sin contexto (aquí, ninguna). |

Dos detalles de cómo se probó **sin gastar dinero ni depender de la red**:

- **`httptest.Server`**: para el adaptador de OpenAI se levanta un **servidor de mentira** que devuelve respuestas simuladas. Se prueba el contrato (que se envía el modelo, el prompt y el `response_format`) sin llamar a la API real.
- **Mocks con `mockgen`**: para el `AnalysisService` se usan imitaciones de los puertos (`LLMExtractor`, `UnitOfWork`, repositorios, `Outbox`) y se verifica, por ejemplo, que `Extract` corre **fuera** de la transacción y `Save`/`Append` **dentro**.

**El truco del "context marker"** merece mención: en el test, el mock del `UnitOfWork` inyecta una **marca** en el `context` al abrir la transacción; luego se comprueba que la llamada al LLM recibió un contexto **sin** la marca y las escrituras uno **con** la marca. Así se demuestra objetivamente que el LLM corre fuera de la transacción.

Y, como se vio en el Bloque 2, la verificación **manual** contra la IA real (`go run ./cmd/llmcheck`) fue la que destapó el problema del "root object". Es el recordatorio de que los tests con simulaciones y la prueba real se complementan: los primeros son rápidos y baratos; la segunda, la única que garantiza que el proveedor externo acepta lo que le mandamos.

> **Concepto — `httptest.Server`**: servidor HTTP de usar y tirar, para tests, incluido en la librería estándar.
>
> **Concepto — Mock**: imitación controlada de una dependencia, que permite decir "cuando te llamen, devuelve esto" y comprobar qué se llamó.
>
> **Concepto — Gate de salida**: criterio objetivo que marca cuándo una fase está terminada, para no avanzar sobre cimientos a medias.

---

## 3. Resumen de los cambios

- **Motor de IA (`4.1` + `4.2.1` + `4.3.2`)**: se implementó `OpenAIExtractor` (adaptador que cumple el puerto `LLMExtractor`), el constructor de *prompt* con la taxonomía derivada del dominio, y **Structured Outputs** con un JSON Schema estricto (envuelto en un objeto raíz porque la API no acepta arrays en la raíz), con los enums de error tomados de las constantes del dominio. Se confirmó que no hay goroutines propias (AP6 por construcción).
- **`AnalysisService` (`4.2.2` + `4.2.3` + `4.2.4` + `4.3.3`)**: validación del JSON del LLM antes de persistir (campos no vacíos + enums → `LLMUnavailableError`), llamada al LLM **fuera** de la transacción, persistencia **atómica** del `Analysis` y su evento en el outbox (`AnalysisCompleted`), y manejo del fallo como **estado terminal** (`Analysis` `failed` + `AnalysisFailed` con motivo genérico, retornando `nil` para no reintentar), con **timeout** de 60 s.
- **Privacidad (`4.3.1` + `4.3.4`)**: `anonymizer.go` redacta emails, teléfonos y nombres (lista curada + heurística de mayúsculas solo en español, solo nombres capitalizados) antes de enviar el texto al LLM; y `PIIHandler` (decorador de `slog.Handler`) descarta cualquier log con texto libre >100 caracteres, email, teléfono, API key o JWT. Cierre de la fase con el **Gate de salida en verde** (cobertura de servicios ≥90%, adaptador del LLM y logger al 100%, A1 verificado, tests con `-race`).
