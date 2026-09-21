# Fase 6 — Provisioner y Privacidad: explicación didáctica

> Documento de estudio sobre todo lo construido en la **Fase 6 (Provisioner y Privacidad)** del proyecto LangLint.
> Cubre los bloques `6.1` (jobs batch del provisioner), `6.2` (endpoints A9: exportar, olvidar y auditar) y `6.3` (retención, pseudonimización y guard de PII en logs).
> Es la continuación de `docs/explains/fase-5-frontend.md`.

---

## 1. ¿De qué trata esto? (Explicación para no técnicos)

En la Fase 5 abriste el **restaurante al público**: los clientes ya pueden entrar, pedir y ver su corrección. Pero un restaurante de verdad no solo atiende mesas: necesita una **trastienda** que mantenga el local ordenado, cumpla la ley y proteja los datos de los clientes. La Fase 6 es exactamente eso: **el departamento de archivo y protección de datos**.

La imagen es la de un **restaurante con su personal de noche y su oficina legal**:

| Imagen de la vida real | Qué representa en el proyecto |
|---|---|
| **El personal de mantenimiento nocturno** | El **provisioner**: un programa que corre por la noche y hace tareas de limpieza y contabilidad. |
| **Recontar la caja y corregir la pizarra** | El job **`refresh-aggregates`**: vuelve a calcular las estadísticas desde los datos originales, por si la pizarra se desincronizó. |
| **Tirar papeles viejos de la papelera** | El job **`purge-raw-data`**: borra de verdad los ejercicios que el cliente ya eliminó hace tiempo. |
| **Destruir el expediente de un cliente que lo pidió** | El job **`execute-deletions`**: ejecuta el "derecho al olvido" cuando pasa el plazo de gracia. |
| **La ventanilla de derechos del cliente** | Los endpoints **A9**: pedir copia de tu expediente, pedir que te olviden, y ver quién ha mirado tu expediente. |
| **El número de casillero anónimo** | La **pseudonimización**: en el panel de estadísticas no se escribe el nombre del cliente, sino un código secreto que solo el restaurante sabe descifrar. |
| **La trituradora junto a la impresora** | El **`PIIHandler`** y **`pii_audit.sh`**: dos redes de seguridad que impiden que un dato personal acabe escrito en los informes internos (los *logs*). |

**La historia completa, contada como anécdota:**

1. El restaurante ya funciona de día (Fase 5). Pero de noche, alguien tiene que **ordenar la trastienda**.
2. Un empleado nocturno **revisa la pizarra de estadísticas**: si está mal (por un error, un reinicio…), la borra y la vuelve a rellenar leyendo los expedientes originales. No "parchea" números sueltos: **reconstruye**.
3. Otro empleado mira la papelera: los expedientes que el cliente borró hace más de 30 días se **trituran de verdad** (ya no se pueden recuperar).
4. Y un tercero revisa la bandeja de "olvidarme, por favor": si el cliente pidió que lo borraran hace más de 30 días, **se destruye todo lo suyo** (ejercicios, correcciones y su fila en la pizarra de estadísticas). Antes de esos 30 días, puede arrepentirse.
5. De cara al público, hay una **ventanilla de derechos**: el cliente puede pedir una **copia de su expediente** (`export`), solicitar **que lo olviden** (`delete`, que solo deja la petición apuntada) y consultar **quién ha accedido a su expediente** (un cuaderno de visitas que nunca se borra).
6. En todo momento, en el **panel de estadísticas** no aparece el nombre real: aparece un **código anónimo** derivado de una clave secreta (la pseudonimización con HMAC-SHA256). Así, aunque alguien robe la pizarra, no puede saber de quién es cada dato.
7. Y como última red de seguridad, hay una **trituradora automática**: si cualquier informe interno intentara contener un correo, un teléfono o una clave, se **descarta antes de imprimirse**. Además, un auditor revisa los informes guardados buscando esos patrones (la auditoría `pii_audit.sh`).

**Resultado de la Fase 6**: el sistema ya no solo "funciona", sino que **se mantiene solo y protege a sus usuarios**: cumple los derechos de portabilidad y olvido (RGPD), no conserva datos personales más de lo necesario y garantiza que las estadísticas nunca revelen quién es cada persona. Con esto, el **Gate de salida de la Fase 6 queda en verde**.

---

## 2. Guía de Estudio (Para entender qué hizo la IA)

Vamos por bloques: **qué se creó**, **cómo funciona** y **por qué** se hizo así. La idea es que puedas abrir el código y reconocer cada decisión.

### Bloque 0 — Recapitulación: la cocina ya existe

Recuerda la forma del backend (axioma **A2**):

```text
   Fuera  ─────────────────────────────────────────────►  Dentro
   cmd → adapters → services → ports → domain
   (arranque)  (Postgres/LLM)  (casos de uso)  (interfaces)  (REGLAS PURAS)
```

Hasta la Fase 5 ya existía **un** programa (`cmd/api`, el servidor HTTP). La Fase 6 introduce **un segundo programa**: `cmd/provisioner`, el "empleado nocturno". Ambos comparten el mismo `go.mod`, el mismo dominio y la misma base de datos, pero **no se importan entre sí** (axioma **A3**). Se comunican solo a través de la base de datos y de las tablas que ambos entienden.

### Bloque 1 — El provisioner: el empleado de noche (`6.1`)

`cmd/provisioner` es un **programa de línea de comandos** con tres tareas (subcomandos). No es un servidor: se lanza, hace su trabajo y termina.

```go
// apps/backend/internal/provisioner/handlers/jobs.go (conceptual)
func (r *Runner) Run(ctx context.Context, command string) error {
    switch command {
    case "refresh-aggregates":  // recontar la pizarra de estadísticas
    case "purge-raw-data":      // triturar expedientes vencidos
    case "execute-deletions":   // ejecutar el derecho al olvido
    }
}
```

**Tarea 1 — `refresh-aggregates` (reconciliar, A4).** La tabla `error_metrics` (las estadísticas de errores) es **datos derivados**: se calculan a partir de las correcciones (`analyses`) y de los ejercicios (`practices`). Si por cualquier motivo se desincronizan, este job las **reconstruye desde cero**: lee todas las correcciones completadas, vuelve a contar cada tipo de error por usuario y por ventana (día/semana/mes), borra la tabla y la rellena de nuevo **en una sola transacción**. La clave del axioma **A4** es que no se "parcha" un número suelto: se **reconstruye la historia completa** desde la fuente de verdad.

**Tarea 2 — `purge-raw-data` (retención limitada, A8).** Cuando un usuario borra un ejercicio, el sistema solo lo marca como "borrado" (`deleted_at`) en vez de eliminarlo al instante. Este job **borra físicamente** los ejercicios que llevan marcados más de 30 días (configurable con `APP_RAW_RETENTION_DAYS`) **junto con sus correcciones**, en una sola transacción para que nunca quede un texto huérfano.

**Tarea 3 — `execute-deletions` (derecho al olvido, A9).** Cuando el usuario pide que lo olviden (ver Bloque 2), el sistema solo **apunta la petición** en la tabla `deletion_requests`. Este job la **materializa** cuando pasan 30 días de gracia (`APP_DELETION_GRACE_DAYS`): borra sus ejercicios, sus correcciones y su fila en las estadísticas. Durante la gracia, el usuario puede cambiar de opinión.

**¿Por qué un programa aparte y no dentro del servidor?** Porque son tareas **pesadas y periódicas**: no deben ralentizar las peticiones de los usuarios ni competir por los mismos recursos. Y, como los jobs son batch, la decisión fue que **la transacción viva en el adaptador** (el repositorio), no en el servicio — así se respeta el anti-patrón **AP8** (los servicios nunca abren transacciones SQL) sin duplicar la maquinaria del API.

> **Concepto — Binario / entry point**: un programa ejecutable independiente. `cmd/api` y `cmd/provisioner` son dos binarios que comparten código pero corren por separado.
>
> **Concepto — Job batch**: tarea que procesa muchos datos de golpe (a diferencia de una petición web, que atiende de una en una).
>
> **Concepto — Soft-delete vs hard-delete**: "borrado suave" = solo marcar como borrado (reversible); "borrado duro" = eliminarlo de verdad (irreversible).
>
> **Concepto — Periodo de gracia**: tiempo de espera entre que se pide algo (el olvido) y que se ejecuta, para permitir arrepentimiento.
>
> **Concepto — Tabla derivada**: datos que se pueden recalcular a partir de otros (por eso `error_metrics` se puede borrar y reconstruir sin perder nada).

### Bloque 2 — Los endpoints A9: tus datos son tuyos (`6.2`)

El axioma **A9** dice que el usuario es dueño de sus datos. En el MVP no hay login (un único usuario fijo desde config), pero igualmente se implementan los tres "derechos sagrados":

| Endpoint | Derecho (RGPD) | Qué hace |
|---|---|---|
| `GET /v1/me/data/export` | Portabilidad (Art. 15+20) | Devuelve una copia completa de los datos del usuario. |
| `DELETE /v1/me/data` | Olvido (Art. 17) | Apunta una petición de borrado; responde `202` y el job la ejecuta luego. |
| `GET /v1/me/access-log` | Auditoría (A4/A9) | Devuelve quién y cuándo accedió a los datos, paginado. |

Todo vive en `IdentityService` (`internal/api/services/identity_service.go`). Dos detalles de diseño importantes:

- **El borrado es idempotente (AP4):** si pides que te olviden dos veces, no se crean dos peticiones. `RequestDeletion` primero pregunta "¿ya hay una pendiente?" y, si la hay, simplemente no hace nada (éxito):

```go
// internal/api/services/identity_service.go (conceptual)
func (s *IdentityService) RequestDeletion(ctx, userID) error {
    pending, _ := s.deletions.HasPending(ctx, userID)
    if pending { return nil }               // ya pedido: idempotente
    req, _ := identity.NewDeletionRequest(userID, s.now())
    return s.deletions.Append(ctx, req)     // solo apunta la petición
}
```

- **La auditoría es append-only (A4):** la tabla `access_events` solo admite **insertar** (nunca actualizar ni borrar). El middleware `internal/api/accesslog` registra cada petición autenticada *después* de responder, en modo **best-effort**: si falla el registro, la respuesta ya fue enviada y no se rompe el flujo — solo se loguea el fallo (sin PII).

> **Concepto — Idempotencia**: repetir la misma operación muchas veces tiene el mismo efecto que hacerla una vez. Clave para que los reintentos no dupliquen datos.
>
> **Concepto — Append-only**: un registro que solo crece; nunca se edita ni se borra. Es lo que permite demostrar "quién hizo qué y cuándo".
>
> **Concepto — Best-effort**: "haz lo que puedas, pero no rompas lo principal". El registro de auditoría no puede tumbar una respuesta que ya se envió.

### Bloque 3 — Pseudonimización: el código anónimo (`6.3.1`)

Aquí está el corazón de la privacidad en analytics. La idea del axioma **A8**: los datos personales **nunca** deben aparecer tal cual donde no hacen falta. Las estadísticas (`error_metrics`) no necesitan saber *quién* eres para contar tus errores, así que en vez del identificador real (`user_id`, un UUID) se guarda un **pseudónimo**: un código que solo se puede derivar si se conoce un secreto.

La función es un **HMAC-SHA256** (un "hash con llave"):

```go
// apps/backend/internal/shared/pseudonymizer/pseudonymizer.go (conceptual)
type Pseudonymizer struct { key []byte }

func (p *Pseudonymizer) Pseudonymize(id string) string {
    mac := hmac.New(sha256.New, p.key)
    mac.Write([]byte(id))
    return hex.EncodeToString(mac.Sum(nil))   // 64 caracteres hexadecimales
}
```

- Mismo `id` + misma clave → **siempre el mismo pseudónimo** (determinista): por eso la escritura y la lectura coinciden.
- Sin la clave (`APP_PSEUDONYM_SECRET`), **es imposible revertir** el pseudónimo al identificador original (es "one-way").

**¿Dónde se aplica?** La decisión clave fue hacerlo en los **adaptadores** (los repositorios), no en los servicios:

```go
// API: al guardar una métrica, se guarda el pseudónimo, no el UUID.
r.pseudonymizer.Pseudonymize(m.UserID.String())

// API: al leer, se busca por el mismo pseudónimo.
WHERE user_id = $1  -- con $1 = Pseudonymize(userID)
```

Esto toca **cuatro sitios**: el `Upsert` y el `ListByUser` del API, el `ReplaceAll` del provisioner (reconstruir estadísticas) y el borrado de `error_metrics` en `execute-deletions`. Por eso el pseudonimizador vive en `internal/shared/` (compartido), **no** en `api/` como sugería el manifiesto: `api` y `provisioner` no se importan entre sí (**A3**) y ambos necesitan producir el mismo pseudónimo.

Para guardar ese código de 64 caracteres, la columna `error_metrics.user_id` cambió de tipo `uuid` a `text` (migración `000004`). Como la tabla es **derivada**, el cambio es seguro: `refresh-aggregates` la reconstruye.

> **Concepto — Hash**: función que convierte cualquier texto en una cadena de tamaño fijo; es "one-way" (no se puede deshacer).
>
> **Concepto — HMAC**: un hash **con clave secreta**. Sin la clave, no puedes reproducir ni invertir el resultado.
>
> **Concepto — Pseudónimo**: identificador sustituto que oculta el real; a diferencia del anonimato total, con la clave se podría volver a enlazar.
>
> **Concepto — Determinista**: misma entrada y misma clave producen siempre la misma salida (necesario para poder consultar después).

### Bloque 4 — Separar los datos crudos (`6.3.2`)

La regla de oro del axioma **A8**: los datos "crudos" (los textos de estudio, que sí son personales) y las estadísticas (agregados) **no deben mezclarse**. En el esquema actual ya están separados:

```text
practices  ─┐   tablas CRUDAS: contienen el texto real del usuario
analyses   ─┘   (se purgan con el tiempo)

error_metrics ── tabla DERIVADA: solo agregados + pseudónimo (sin texto)
```

Lo que faltaba era **demostrarlo**. Se añadió el test Tier 3 `TestPrivacy_RawDataSeparatedFromAnalytics`, que siembra una práctica con datos identificables (un email), la analiza, y comprueba que:
1. El texto crudo vive en `practices`/`analyses`.
2. `error_metrics` solo contiene la clave pseudonimizada, **sin** el UUID real ni ningún texto.
3. Al ejecutar `execute-deletions`, **desaparece todo rastro** (incluida la fila pseudonimizada de estadísticas).

> **Concepto — Tier 3**: tests que levantan una base de datos real (efímera, vía testcontainers) para probar la integración completa, no solo funciones aisladas.

### Bloque 5 — La trituradora y el auditor (`6.3.3`)

Dos defensas complementarias para que ningún dato personal acabe en los *logs* (registros internos):

1. **Guard runtime — `PIIHandler`** (`internal/shared/logger`): ya existía desde la Fase 4. Es un decorador del logger que **descarta** cualquier línea que contenga patrones sospechosos: texto libre muy largo, emails, teléfonos, API keys o JWTs. En la Fase 6 se **verificó** que todos los loggers de producción (`cmd/api` y `cmd/provisioner`) pasan por él (usan `logger.New`, que envuelve el handler con `PIIHandler`); el resto de `slog.New` del proyecto son solo tests que descartan su salida.

2. **Auditoría estática — `ops/scripts/pii_audit.sh`**: un script que **revisa los logs ya emitidos** (por ejemplo en CI) buscando los mismos patrones. Si encuentra uno, imprime la línea **con el valor oculto** (`[REDACTED]`) y termina con código de error para que el pipeline falle. Así, una fuga de PII en un log no pasa desapercibida.

```bash
# ops/scripts/pii_audit.sh (extracto conceptual)
grep -rnE "$EMAIL|$PHONE|$API_KEY|$JWT" "$@" || true   # busca patrones
# ... si hay coincidencias: imprime [REDACTED] y sale con código 1
```

> **Concepto — Guard runtime**: una protección que actúa *mientras* el programa corre (descarta el log antes de escribirlo).
>
> **Concepto — Auditoría estática**: una revisión *a posteriori* sobre archivos ya generados (los logs), como un inspector de sanidad.
>
> **Concepto — Código de salida (exit code)**: número que un programa devuelve al terminar. `0` = éxito; cualquier otro valor = fallo. Es lo que usa el CI para decidir si la build pasa o no.

---

## 3. Resumen de los cambios

- **Provisioner (`6.1`)**: nuevo binario `cmd/provisioner` con tres jobs batch —`refresh-aggregates` (reconstruye las estadísticas desde la fuente de verdad, A4), `purge-raw-data` (borrado físico tras la retención, A8) y `execute-deletions` (derecho al olvido tras la gracia, A9)— con su propio módulo de inyección de dependencias y el harness de tests Tier 3 movido a `internal/shared/testdb`.
- **Endpoints A9 (`6.2`)**: `IdentityService` + `GET /me/data/export` (portabilidad), `DELETE /me/data` (olvido idempotente con petición pendiente) y `GET /me/access-log` (auditoría append-only con middleware best-effort), con las tablas `deletion_requests` y `access_events`.
- **Privacidad y retención (`6.3`)**: pseudonimización HMAC-SHA256 de `error_metrics.user_id` con `APP_PSEUDONYM_SECRET` (aplicada en adaptadores, migración `000004`), test que certifica la separación entre datos crudos y analytics, y la auditoría `ops/scripts/pii_audit.sh` que cierra el **Gate de Fase 6 en verde**.
