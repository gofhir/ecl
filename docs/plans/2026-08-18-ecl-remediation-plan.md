# ECL Remediation Plan — semántica, parser y contrato

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Corregir los 7 defectos que hoy producen conjuntos de conceptos incorrectos en silencio, y cerrar el contrato de `DataProvider` para que un tercero pueda implementarlo sin leer el evaluador.

**Origen:** auditoría adversaria del 2026-08-18 (14 agentes, 41 hallazgos, 14 reproducidos a mano). Referencias `B1`–`B10` remiten a los bloqueantes de ese informe.

**Arquitectura de la remediación:** primero la red de pruebas (sin ella ningún arreglo es verificable), después el parser (barato y sin tocar la API), después la semántica del evaluador, y al final el contrato. Cada tarea es TDD: primero el test que falla, luego el arreglo. Ningún commit deja la suite roja.

**Restricción central — la librería está publicada en v1.1.0.** `DataProvider`, `ecl.Set` y `ecl/ast` están bajo la promesa de compatibilidad de Go. El plan se divide en dos fases por eso:

- **Fase A → v1.2.0.** Todo lo que se puede arreglar sin cambiar una firma exportada. Incluye los 5 defectos de semántica y los 2 de parser. Cambios aditivos a structs (nuevos campos) están permitidos; añadir un método a `DataProvider` no.
- **Fase B → ~~v2.0.0~~ interfaces de capacidad opcionales, sin major.** Decisión del 2026-08-22:
  el proyecto no quiere versiones mayores. Eso resultó ser mejor diseño, no una limitación. Todo lo que
  esta fase iba a romper —firmas batch, el método que falta para la cardinalidad reverse, negación a
  nivel de fila— se ofrece como interfaces que el evaluador detecta con type assertion, el patrón que
  la librería estándar usa para `io.ReaderFrom` y `http.Flusher`. Implementar una **nunca es
  obligatorio y nunca rompe nada**, y el evaluador cae al camino anterior o reporta lo que no puede.
  Los campos deprecados se quedan poblados para siempre; y el doble segmento del import path
  (`github.com/gofhir/ecl/ecl`) es permanente, que conviene que sea una decisión consciente.

**Tech Stack:** Go 1.24, ANTLR4 runtime v4.13.x, testify (assert/require)

**Informe de auditoría:** artefacto `Auditoría adversaria de go-ecl`

---

## Revisión adversaria de este plan (2026-08-18)

Este plan fue sometido a 6 revisores adversarios independientes + 6 defensores. Resultado: 26 críticas
en pie, 9 demolidas. Las correcciones ya están **incorporadas** en las tareas de abajo; se listan aquí
para que quien ejecute sepa qué cambió y por qué, y no reintroduzca un error ya corregido.

| Corrección | Qué estaba mal en la v1 de este plan |
|---|---|
| **C1 (fatal)** | La Task 3 hacía `return applyAttrSet(...)` sobre `ref.AttrSet`. Como el parser puebla `AttrSet` y `Conjunction`/`Groups` en el **mismo** nodo, ese `return` descartaba los grupos hermanos. Verificado: `* : A = x , { B = y }` produce `Ungrouped=1, Conjunction=1`, así que el arreglo habría introducido exactamente la clase de bug que este plan existe para erradicar. |
| **C2 (grave)** | La Task 5 ordenaba aplicar `total − matching` a `filterByConcreteValue`. Ahí el operador ya se aplica dentro de `compareFloat` (ver el comentario de `evaluator.go:1101-1103`), así que la inversión rompía un caso verde: `07-concrete.yaml:19` (`1142139005 != #6`). |
| **C3 (grave)** | El `result.Union(acc)` de la Task 3 sobre-genera cuando `collectSubrefinement` ya aplanó un paréntesis. Y la cita del sitio de aplanamiento apuntaba a la rama equivocada. |
| **C4 (grave)** | La Task 2 convertía en error duro toda expresión con caracteres fuera de `UTF8_LETTER` (U+2019, U+00B0, U+00A0 — cualquier copy-paste del navegador SNOMED) sin declararlo como cambio de comportamiento. |
| **Task 0 (nueva)** | Seis helpers de test usados en seis tareas no existían, y `evalFixture` es **imposible** en `package ecl`: `internal/conformance` importa `ecl`, así que un test in-package da `import cycle not allowed in test`. |
| **Task 1b (nueva)** | El fixture no tiene sección `dialects:` ni `memberFields:` ni conceptos inactivos suficientes, así que varias tareas no podían escribir sus casos. Se adelanta desde la Task 11. |
| **Task 6** | El «test que falla» **pasaba hoy**: `!!> (404684003 OR 22298006)` ya devuelve `[404684003]`. El par que discrimina es abuelo/nieto con el intermedio ausente. |
| **Task 17a (nueva)** | `providertest`, `go:embed`, los centinelas y `UnimplementedDataProvider` son aditivos: estaban mal encajonados en el v2, dejando el `conformance` del binario instalado roto hasta un major sin fecha. |

Dos críticas se demolieron con mediciones que corrigen al propio plan y merecen quedar escritas:
la cardinalidad de grupo **sí** es verificable con el fixture actual (hoy `* : [2..*] { 363698007 = 74281007 }`
devuelve `22298006`, no `∅`, así que discrimina antes/después), y excluir el grupo 0 del conteo está
respaldado por el godoc que la librería ya publica (`provider.go:47-48`), no por la spec.

---

## Estado de ejecución

| Task | Estado | Notas |
|---|---|---|
| 0 · Asiento de pruebas | ✅ **hecha** | `ecl/fixture_ext_test.go` (`package ecl_test`, confirmado el ciclo de imports). `nilProvider`/`countingProvider` se posponen a la Task 9, donde se usan: declararlos antes deja tipos sin usar. |
| 1 · Red de conformidad | ✅ **hecha** | `TestRunAllSuites` con un subtest por caso + paso de CLI en CI. Sabotaje verificado: alterar un `expectedIds` de `05-filters.yaml` ahora **falla** `go test ./...`; antes quedaba verde. Cobertura de `internal/conformance`: 41.8 % → 73.9 %. |
| 2 · Parser: EOF + lexer | ✅ **hecha** | `ParseError`/`SyntaxError` públicos, listener en el lexer, errores acumulados, `09-errors.yaml` con 10 casos. `wantTotalCases` 44 → 53. |
| 1b · Ampliar el fixture | ✅ **hecha** | Segundo grupo de relaciones en `22298006`, secciones `dialects:` y `memberFields:`, segundo inactivo, segunda asociación histórica, metadatos de concepto. Un solo caso re-baselineado (dot notation, ahora con dos targets) y documentado en el commit. |
| 2 · Parser: EOF + lexer | ✅ **hecha** | Ver arriba. |
| 3 · Disyunción de atributos | ✅ **hecha** | `ast.AttributeSet` aditivo; `Ungrouped`/`Attrs` deprecados pero poblados. Corrige además la disyunción a nivel de refinamiento, el ámbito de los paréntesis y el `OR` intra-grupo. `explain` ya muestra el operador. 9 casos nuevos; `wantTotalCases` 53 → 62. |
| 4 · Unificar `!=` | ✅ **hecha** | La ruta forward reutiliza `clauseSatisfied` sobre las relaciones del concepto con los grupos aplanados. `conceptMatchesAttribute` → `conceptRelationships`. El test que codificaba el bug reescrito. 4 casos nuevos. |
| 5 · Cardinalidad | ✅ **hecha** | Grupos contados sin `break`, grupo 0 excluido, valores concretos contados. **C2 respetada**: no se invierte `!=` en la ruta concreta, con test anti-regresión. 6 casos nuevos. |
| 6 · Top/Bottom | ✅ **hecha** | `Ancestors`/`Descendants`. El test del plan era vacuo; el par discriminante es abuelo/nieto con el intermedio ausente. `bottomOfSet` no tenía ningún test. |
| 7 · Negación de filtros | ✅ **hecha** | Filtros de concepto por cláusula con `Intersect`/`Minus`; filtros de descripción con `ErrUnsupportedFeature`. Los 3 artefactos verdes que la revisión listó, convertidos en el mismo commit. Resuelto el doble dueño de la negación del dialecto. |
| 8 · Ramas del parser | ✅ **hecha** | `dialectId` con sets y acceptability, `memberField` con literales, `descriptionIdFilter` modelado, escapes decodificados (dejando `\*` en patrones wild), y los operandos de conjunto recogidos completos. Donde el contrato del provider puede llevar la forma nueva (`ModuleIDs`, `DefinitionStatusIDs`) se arregló de punta a punta; donde no (`D id`, conjunto de términos, conjunto de `effectiveTime`), `ErrUnsupportedFeature`. La forma alias `dialect = en-gb` queda con error explícito: mapear alias→refset es dato terminológico. |
| 9 · Contrato del provider | ✅ **hecha** | Preámbulo de contrato en el interface, `nonNil`, `ctx.Err()`, eje `active` redefinido. `nilProvider` con los 18 métodos sobre 8 expresiones que antes panicaban. |
| 10 · Historia | ✅ **hecha** | Dirección invertida, godoc corregido, suite reescrita desde el lado activo. Tres asociaciones en tiers distintos para que MIN/MOD/MAX discriminen — antes los tres devolvían lo mismo. |
| 11 · termMatches y cobertura | ✅ **hecha** | `match` como prefijo de palabra; rama `regex` inalcanzable eliminada. |
| 12 · MRCM | ✅ **hecha** | Dominios en disyunción, `Min` sobre las reglas del modelo (`Model.AllDomains`), regla inválida como `IssueKindInvalidRule`, loader valida el ECL, issues duplicados eliminados. |
| 13 · SCG + SCTID | ✅ **hecha** | Grupos yuxtapuestos (el ejemplo canónico ya parsea), default `===`, particiones SCTID. |
| 14 · CLI | ✅ **hecha** | Diagnósticos a stderr, `-h` con código 0, códigos de salida tipados (3 sintaxis / 4 no soportado). Cobertura 0 % → 55.2 %. |
| 15 · README | ✅ **hecha** | 7 `Example` compilables como fuente de verdad; corregidas las afirmaciones falsas (N+1, `internal/` reutilizable, salida de `explain`); sección de limitaciones conocidas. |
| 16 · Proceso | ✅ **hecha** | Matriz 3×2, gate de `tidy` (fallaba), gate de regeneración del parser **verificado byte a byte**, umbrales de complejidad que muerden, `dependabot.yml`, `release.yml` alineado, `make check`. |
| 17a · providertest | ✅ **hecha** | Paquete público con la suite **embebida**: el binario instalado ya corre las 95 desde cualquier directorio. `Verify` + `UnimplementedDataProvider`. |

**Fase A completa, y la Fase B resuelta sin major.** Las cuatro capacidades opcionales viven en
`ecl/capabilities.go`, las cuatro están conectadas al evaluador, y `providertest.VerifyContract` tiene
un check por capacidad que verifica que **coincida con el método obligatorio que acelera** — el fallo
que importa, porque el evaluador prefiere la capacidad y una discrepancia decidiría cada respuesta en
silencio.

| Capacidad | Qué desbloquea | Medido |
|---|---|---|
| `BatchPropertiesProvider` | El N+1 real de las refinaciones | 27 llamadas → 1 |
| `BatchConcreteValuesProvider` | Los valores concretos, que costaban N×T | — |
| `InboundRelationshipsProvider` | `[m..n] R attr = value` y `R attr != value` | — |
| `NegatingDescriptionProvider` | Los filtros de descripción negados | — |

Lo que sigue pendiente y **no** lo resuelve una capacidad:

- ~~**Las formas reverse de grupo**~~ — **cerrada (2026-08-23), rechazándolas.** No era falta de datos:
  las llaves afirman que las cláusulas comparten un grupo de relaciones *del concepto foco*, y una
  relación inversa pertenece al origen, así que el foco no tiene nada que agrupar. Lo que quedaba era
  redundante (`{ R a = x }` a solas mide idéntico a la forma sin agrupar) o engañoso (en
  `{ R a = x, b = y }` la cláusula hermana restringía el **origen**). Ontoserver también lo rechaza.
  Ahora devuelve `ErrUnsupportedFeature` remitiendo a la forma sin agrupar, y se borraron las cuatro
  funciones que la implementaban. Con esto **la limitación abierta de la Task 3 también queda cerrada**:
  ya no hay ruta reverse dentro de grupos que aplane cláusulas.
- ~~**El dialecto por alias**~~ — **cerrada (2026-08-27).** Se añadió la capacidad
  `DialectAliasResolver`, tal como decía esta línea: el mapeo alias→refset lo aporta el
  provider, no una tabla parcial inventada aquí. Un alias que el provider no resuelve se
  reporta, porque tratarlo como "sin restricción de dialecto" ampliaría la consulta a todos.
- ~~**Conjuntos de términos y de `effectiveTime`**~~ — **cerrada (2026-08-27), sin tocar el
  contrato.** Son any-of, o sea la unión de los filtros de un solo valor, así que el
  evaluador emite una llamada por valor. Añadir campos `Terms`/`EffectiveTimes` habría sido
  peor: todo provider existente los ignoraría en silencio y el filtro se aplicaría sobre un
  valor del conjunto. Bajo `!=` el conjunto de `effectiveTime` significa "ninguno de estos",
  y se intersecta. Sigue sin soportarse la forma negada del conjunto de términos: una sola
  fila de descripción tiene que fallar todos los valores, y eso no se descompone.
- ~~**`{{ D id }}`**~~ — **cerrada (2026-08-30)** con la capacidad `DescriptionIDProvider`. Es por
  fila: el id y las cláusulas hermanas tienen que cumplirse en la misma descripción.
- ~~**La proyección `^[field]`**~~ — **cerrada (2026-08-30)** con `RefsetFieldProjector`. La
  limitación estaba mal diagnosticada acá: se anotó como "necesita un tipo de retorno distinto de
  `Set`", y eso solo es cierto de `^[a,b]` y `^[*]`, que son tabulares. Un campo con id de componente
  —que es lo que usa el ejemplo oficial— cabe en un `Set`. De paso apareció que `^[*]` se evaluaba
  en silencio como `^`.
- **Opciones funcionales en `Evaluate`** (`WithMaxDepth`, `WithCache`), que son aditivas y quedan
  disponibles cuando se necesiten. Nunca hicieron falta.

**Tabla de aceptación: 13 de 13 filas pasan.**

**Cerrado después del plan** (nada de esto estaba en él): parser cuadrático → lineal con entrada
acotada y fuzzing, `InGroupCardinality` del MRCM más el conteo por valores distintos, renderer de SCG
con propiedad de ida y vuelta, el corpus oficial del IHTSDO en CI, el oráculo diferencial contra un
servidor real, y los dos defectos de proceso del release (`include-component-in-tag` mal configurado
y `ECL.g4` sin procedencia).

**Métricas al cierre del plan:** 44 → **123** casos de conformidad, todos ejecutados por CI (antes 8).
Cobertura de `ecl` 61.8 % → 76.9 %; `providertest` (antes `internal/conformance`) 41.8 % → 81.7 %.
`golangci-lint` en 0 issues y `-race` limpio en todos los commits. Ningún cambio rompe la API: los
campos viejos siguen poblados y deprecados.

**Estado al 2026-08-31**, tras el trabajo posterior al plan: **153** casos de conformidad, más 121
ejemplos oficiales del IHTSDO que parsean y 39 casos de oráculo diferencial contra un servidor real.
Cobertura: `ecl` 77 %, `providertest` 82 %, `mrcm` 95 %, `scg` 92 %, `sctid` 97 %, `cmd` 57 %. Siete
capacidades opcionales. Publicado hasta v1.6.0. Los números de arriba se dejan como quedaron el día
que el plan se cerró, para que se pueda leer qué aportó el plan y qué vino después.

**Cambios de comportamiento a agrupar en las notas del v1.2.0** (cuatro): errores del lexer y anclaje
`EOF` (Task 2), negación de filtros de descripción (Task 7), eje `active` en `AllConcepts` (Task 9),
dirección de `HistoricalAssociations` (Task 10).

**Nota de la Task 3, para quien siga.** El refactor extrajo `clauseSatisfied`, que ya implementa
`count = totalOfType − matching` para `!=` y **no** lo aplica a los valores concretos (donde el
operador vive dentro de `matchConcreteValue`). Es exactamente la corrección que pide la Task 4 y el
aviso de la C2: la Task 4 consiste ahora en hacer que la ruta forward sin agrupar
(`filterByAttribute`) use esa misma fórmula en lugar de invertir el booleano de cardinalidad.

**Limitación abierta que introducía la Task 3 — cerrada.** La ruta reverse dentro de grupos aplanaba
las hojas, así que `{ R a = x OR b = y }` se evaluaba como `AND`. No se arregló: se eliminó la ruta,
porque el constructo no tiene significado (ver arriba).

**Decisión C4, por escrito (la exigía el Step 6 de la Task 2).** Se eligió **no normalizar** la entrada:
ni NFKC ni colapso de espacios Unicode. Motivo: cualquier normalización transforma en silencio el
término que después se compara contra el provider — el mismo pecado que este plan combate — y no hay
forma de limitarla al texto del término sin ser consciente del lexer. Un carácter que el lexer no
reconoce significa que el parse **no es fiel a la entrada**, así que reportarlo es la conducta correcta:
convierte una corrupción silenciosa (`Crohn’s disease` → `Crohns disease`) en un fallo clasificable.
El arreglo de fondo es ampliar `UTF8_LETTER` en `ECL.g4`, y queda en el Step 3 de la Task 16, detrás del
gate de regeneración. Los casos de `09-errors.yaml` fijan el comportamiento actual y habrá que
convertirlos en `expectedIds` cuando la gramática se amplíe.

---

## Task 0: Asiento de pruebas — bloquea todas las demás

Seis tareas de este plan usan helpers de test que no existen en el repo. Sin este asiento, cada tarea
los reinventa de forma incompatible y los conjuntos esperados se calculan contra fixtures distintos.

**Files:**
- Add: `ecl/fixture_ext_test.go` (**`package ecl_test`**, no `package ecl`)
- Modify: `ecl/provider_test.go:10` (reutilizar el `mockProvider` existente)

**Step 1: El paquete importa**

`evalFixture` necesita `internal/conformance` para cargar `standard.yaml`, y `internal/conformance`
importa `github.com/gofhir/ecl/ecl`. Un test in-package produce `import cycle not allowed in test`
(demostrado). Por eso el archivo va en `package ecl_test`, que sí compila.

```go
package ecl_test

// evalFixture parses and evaluates expr against testdata/conformance/fixtures/standard.yaml.
// Lives in package ecl_test on purpose: internal/conformance imports ecl, so an in-package test
// file that loads the fixture is an import cycle.
func evalFixture(t *testing.T, expr string) ecl.Set { … }

// evalFixtureErr is evalFixture but returns the error instead of failing the test.
func evalFixtureErr(t *testing.T, expr string) (ecl.Set, error) { … }

// isActiveInFixture reports the active flag standard.yaml declares for id.
func isActiveInFixture(t *testing.T, id string) bool { … }

// mustParse fails the test if expr does not parse.
func mustParse(t *testing.T, expr string) ast.Expression { … }

// nilProvider returns (nil, nil) from every method — the idiomatic Go shape for
// "nothing found" that provider.go does not currently forbid. Used by Task 9.
type nilProvider struct{}

// countingProvider wraps a provider and counts calls per method. Used by Task 9
// (cancellation) and to measure the N+1 claims.
type countingProvider struct{ … }
```

**Step 2: Un fixture por tarea, declarado**

Hay **dos** fixtures en juego y no son intercambiables: `testdata/conformance/fixtures/standard.yaml`
(usado por el CLI, el runner y los helpers de arriba) y el `newFixture()` de `ecl/evaluator_test.go`,
que tiene conceptos y grupos distintos (p. ej. `404684004` con relaciones en los grupos 1 y 2).

Regla para todo el plan: **cada test declara en un comentario contra qué fixture están calculados sus
`expectedIds`**. Los conjuntos esperados de las Tasks 3-8 de este documento están calculados contra
`standard.yaml` salvo donde se diga lo contrario.

```bash
go test ./ecl/ -run TestFixtureHelpers -v
```

---

## Task 1b: Ampliar el fixture — antes de las Tasks 5, 8, 9 y 11

Adelantada desde la Task 11. Varias tareas necesitan datos que el fixture no tiene, y descubrirlo a
mitad de la tarea obliga a re-baselinear casos ya escritos.

**Files:**
- Modify: `testdata/conformance/fixtures/standard.yaml`
- Modify: `internal/conformance/fixture.go` (cargar las secciones nuevas)

**Step 1: Lo que falta**

- **Relaciones en ≥ 2 grupos.** Hoy las 4 están en `group: 1` (verificado). Necesario para que los
  casos de cardinalidad de grupo cubran también el caso de sobre-emparejamiento.
- **Sección `dialects:`** con acceptability. Sin ella, el caso de dialecto de la Task 8 no existe.
- **Sección `memberFields:`** con valores string, para el `memberField` de la Task 8.
- **≥ 2 conceptos inactivos**, para `{{ C active = false }}` (Task 9) y para que la historia discrimine
  perfiles (Task 10).
- **`definitionStatusId`, `moduleId`, `effectiveTime`** poblados, para la Task 11.

**Step 2: Re-baseline explícito**

Ampliar el fixture puede mover `expectedIds` de casos existentes. Hoy ningún caso usa `*` como foco
(`grep 'expression: "\*' testdata/conformance/cases/*.yaml` → 0 coincidencias) y los esperados están
anclados en `<<` o en IDs explícitos, así que el riesgo es bajo — pero **listar en el commit** todo
`expectedIds` que cambie, con el motivo. Un re-baseline silencioso aquí anula la red de la Task 1.

```bash
go run ./cmd/gofhir-ecl conformance   # debe seguir en 0 failed
```

---

## Fase A — v1.2.0, sin romper la API

### Task 1: Red de conformidad que pueda fallar

Va primera porque hoy CI ejecuta 8 de los 44 casos. Sin esto, ninguna de las tareas siguientes queda protegida y no hay forma de saber si un arreglo rompió otro camino. **(B8)**

**Files:**
- Modify: `internal/conformance/runner_test.go:26-60` (reemplazar `TestRunSuite_Hierarchy`)
- Modify: `.github/workflows/ci.yml` (añadir paso del binario)

**Step 1: Test que recorre todas las suites**

Sustituir `TestRunSuite_Hierarchy` por un test que haga glob del directorio. El conteo explícito es deliberado: si alguien añade un archivo de casos y olvida registrarlo, el test lo detecta; si alguien borra casos, también.

```go
// TestRunAllSuites ejecuta TODAS las suites de conformidad empaquetadas.
// El conteo esperado es explícito para que añadir o borrar casos sea un
// cambio visible en el diff, no un silencio.
func TestRunAllSuites(t *testing.T) {
	root := repoRoot(t)
	casesDir := filepath.Join(root, "testdata", "conformance", "cases")
	fixtureDir := filepath.Join(root, "testdata", "conformance", "fixtures")

	paths, err := filepath.Glob(filepath.Join(casesDir, "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, paths, "no se encontró ninguna suite en %s", casesDir)

	suites := make([]*Suite, 0, len(paths))
	for _, p := range paths {
		s, err := LoadSuite(p)
		require.NoErrorf(t, err, "LoadSuite(%s)", p)
		suites = append(suites, s)
	}

	rep, err := RunSuites(context.Background(), suites, RunOptions{FixtureDir: fixtureDir})
	require.NoError(t, err)

	for _, r := range rep.Results {
		t.Run(r.Case.Name, func(t *testing.T) {
			if r.Skipped {
				t.Skip(r.Reason)
			}
			if !r.Passed {
				t.Errorf("%s\n  expresión: %s\n  motivo: %s", r.Case.Name, r.Case.Expression, r.Reason)
			}
		})
	}

	passed, failed, skipped, total := rep.Summary()
	t.Logf("%d passed, %d failed, %d skipped, %d total", passed, failed, skipped, total)
	require.Zero(t, failed)
	require.Equal(t, wantTotalCases, total, "cambió el número de casos; actualiza wantTotalCases")
}
```

Declarar la constante junto al test, con el número actual:

```go
// wantTotalCases es el número de casos de conformidad empaquetados.
// Súbelo al añadir casos — nunca lo bajes sin explicar por qué en el commit.
const wantTotalCases = 44
```

**Step 2: Paso de CI que ejecuta el binario**

En `.github/workflows/ci.yml`, después del paso de tests:

```yaml
      - name: Conformance suite (binario)
        run: go run ./cmd/gofhir-ecl conformance
```

**Step 3: Verificar que la red muerde**

Sabotaje deliberado, obligatorio antes de cerrar la tarea: alterar un `expectedIds` en `testdata/conformance/cases/05-filters.yaml` y comprobar que `go test ./...` **falla** ahora. Revertir después.

```bash
go test ./internal/conformance/ -run TestRunAllSuites -v   # 44 subtests
go test ./...
```

---

### Task 2: Anclar EOF, registrar el ErrorListener del lexer, acumular errores

Dos defectos con un mismo síntoma: `Parse` acepta entradas inválidas con `err == nil` y devuelve un AST truncado. **(B2)**

**Files:**
- Modify: `ecl/parser.go:15-50` (`Parse`, `eclErrorListener`)
- Modify: `ecl/parser_test.go` (casos nuevos)

**Step 1: Tests que fallan primero**

```go
func TestParse_RejectsTrailingInput(t *testing.T) {
	for _, in := range []string{
		"11687002 TOTALGARBAGE",
		"<< 404684003 ESTO SOBRA",
		"404684003 MINUS 11687002 MINUS 73211009", // MINUS no es asociativo sin paréntesis
		"1234567890 OR 2234567890 AND 3234567890", // mezclar AND/OR sin paréntesis es inválido
	} {
		t.Run(in, func(t *testing.T) {
			_, err := Parse(in)
			require.Error(t, err, "se aceptó una expresión inválida y se truncó el AST")
		})
	}
}

func TestParse_RejectsLexerErrors(t *testing.T) {
	// U+2019 queda fuera de todos los rangos UTF8_LETTER de ECL.g4.
	_, err := Parse("404684003 |Crohn’s disease|")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "Crohns", "el carácter se borró del stream en silencio")
}

func TestParse_ErrorIsTyped(t *testing.T) {
	_, err := Parse("<< invalid!!!")
	var pe *ParseError
	require.ErrorAs(t, err, &pe)
	require.NotEmpty(t, pe.Errors)
	require.Positive(t, pe.Errors[0].Line)
}
```

**Step 2: Tipo de error público**

En `ecl/parser.go`, antes de `Parse`. Se introduce ahora porque ya estamos reescribiendo el camino de errores; hacerlo en dos pasos obligaría a tocarlo dos veces.

```go
// SyntaxError is a single syntax error reported while parsing.
type SyntaxError struct {
	Line, Column int
	Msg          string
}

func (e SyntaxError) Error() string {
	return fmt.Sprintf("syntax error at %d:%d: %s", e.Line, e.Column, e.Msg)
}

// ParseError collects every syntax error found in one expression. Callers can
// classify failures with errors.As instead of matching on strings.
type ParseError struct {
	Errors []SyntaxError
}

func (e *ParseError) Error() string {
	if len(e.Errors) == 1 {
		return "invalid ECL: " + e.Errors[0].Error()
	}
	msgs := make([]string, 0, len(e.Errors))
	for _, se := range e.Errors {
		msgs = append(msgs, se.Error())
	}
	return "invalid ECL: " + strings.Join(msgs, "; ")
}
```

**Step 3: Acumular en lugar de sobreescribir**

`eclErrorListener.SyntaxError` hoy asigna `l.err` en cada llamada, así que de un lote solo sobrevive el último:

```go
type eclErrorListener struct {
	antlr.DefaultErrorListener
	errs []SyntaxError
}

func (l *eclErrorListener) SyntaxError(_ antlr.Recognizer, _ any, line, column int, msg string, _ antlr.RecognitionException) {
	l.errs = append(l.errs, SyntaxError{Line: line, Column: column, Msg: msg})
}
```

**Step 4: `Parse` con lexer instrumentado y EOF exigido**

```go
func Parse(input string) (ast.Expression, error) {
	errListener := &eclErrorListener{}

	lexer := grammar.NewECLLexer(antlr.NewInputStream(input))
	lexer.RemoveErrorListeners()      // sin esto, ANTLR escribe a os.Stderr
	lexer.AddErrorListener(errListener)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := grammar.NewECLParser(stream)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errListener)

	tree := parser.Expressionconstraint()

	// La regla expressionconstraint de ECL.g4 no termina en EOF, así que ANTLR
	// para en el primer parse completo y descarta el resto sin avisar.
	//
	// El mensaje usa el resto de la ENTRADA, no el texto del token: el lexer de
	// ECL es a nivel de carácter, así que tok.GetText() sería "T" para
	// "11687002 TOTALGARBAGE" — inútil para el usuario. Verificado.
	if tok := stream.LT(1); tok != nil && tok.GetTokenType() != antlr.TokenEOF {
		rest := strings.TrimSpace(input[tok.GetStart():])
		errListener.errs = append(errListener.errs, SyntaxError{
			Line:   tok.GetLine(),
			Column: tok.GetColumn(),
			Msg:    fmt.Sprintf("unexpected trailing input %q", rest),
		})
	}

	if len(errListener.errs) > 0 {
		return nil, &ParseError{Errors: errListener.errs}
	}

	visitor := &astBuilder{}
	if expr, ok := visitor.Visit(tree).(ast.Expression); ok {
		return expr, nil
	}
	return nil, fmt.Errorf("unexpected parse result")
}
```

**Alternativa preferible a medio plazo:** añadir `root : expressionconstraint EOF ;` a `ECL.g4` y llamar a `parser.Root()`. Se deja para cuando el Step 3 de la **Task 16** establezca el gate de regeneración, para no tener un `.g4` modificado sin forma reproducible de regenerar. El enfoque de `stream.LT(1)` está verificado: para 8 expresiones válidas (incluidos whitespace inicial/final y `/* comentario */`) devuelve `EOF`, y para las 5 inválidas devuelve el primer token no consumido, con cero falsos positivos.

**Step 5: Casos de error en la suite**

Crear `testdata/conformance/cases/09-errors.yaml` con `expectError: true` (el runner ya lo soporta y hoy no hay ni un caso):

```yaml
name: "Errores de sintaxis"
fixture: standard.yaml
cases:
  - name: "trailing input tras una expresión válida"
    expression: "11687002 TOTALGARBAGE"
    expectError: true
  - name: "MINUS encadenado sin paréntesis"
    expression: "404684003 MINUS 11687002 MINUS 73211009"
    expectError: true
  - name: "AND y OR mezclados sin paréntesis"
    expression: "< 404684003 AND < 11687002 OR < 73211009"
    expectError: true
```

Subir `wantTotalCases` en el mismo commit.

**Step 6: Declarar el cambio de comportamiento del ErrorListener del lexer (C4)**

Registrar el listener en el lexer no es un detalle: convierte en error duro **una clase entera** de
expresiones que hoy se aceptan. `UTF8_LETTER` (`ECL.g4:183-191`) arranca en `'À'`, así que todo lo que
cae en los huecos anteriores dispara `token recognition error`, el carácter se borra del stream y hoy
`Parse` devuelve `err == nil`. Verificado:

| Entrada | Carácter | Hoy | Tras el Step 4 |
|---|---|---|---|
| `404684003 \|Crohn’s disease\|` | U+2019 | `err=nil`, término corrupto | `*ParseError` |
| `404684003 \|Temperatura 37°C\|` | U+00B0 | `err=nil` | `*ParseError` |
| `404684003 \|Body structure\|` con NBSP | U+00A0 | `err=nil` | `*ParseError` |
| `404684003 \|hipertensión\|` | U+00F3 | `err=nil` | sigue OK — el corte **no** es ASCII vs no-ASCII |

El NBSP es el caso que importa en la práctica: aparece en cualquier copy-paste del navegador SNOMED.

Tres cosas obligatorias:

1. Entrada de CHANGELOG bajo un encabezado explícito **«Behaviour change»**, con la lista de rangos
   afectados.
2. Casos en `09-errors.yaml` con U+00A0 y U+2013 en un término, para fijar la decisión.
3. Decidir **por escrito** una de dos: **(a)** colapsar los espacios Unicode a U+0020 dentro de `Parse`
   antes de crear el lexer — arregla el NBSP sin tocar el `.g4` ni depender de la Task 16; o **(b)**
   ampliar `UTF8_LETTER` y adelantar el Step 3 de la Task 16 (gate de regeneración) **antes** de esta
   tarea. **No usar NFKC**: mapea `µ`→`μ` y `½`→`1⁄2`, corrompiendo el término que después se compara
   contra el provider.

Nota sobre el test de U+2019: el ABNF oficial de ECL admite `UTF8-2/3/4` en un término, así que
`require.Error` sobre `|Crohn’s disease|` fijaría en la suite una expectativa que **contradice** la
spec. El test debe assertar que el error es de *reconocimiento de token* (es decir, que el problema se
reporta en vez de silenciarse), no que la expresión sea inválida.

```bash
go test ./ecl/ -run TestParse -v
go run ./cmd/gofhir-ecl validate "11687002 TOTALGARBAGE"   # debe fallar
```

---

### Task 3: Disyunción de atributos en el AST

Hoy `collectAttributes` aplana `Subattributeset`, `Conjunctionattributeset` y `Disjunctionattributeset` en una única slice, así que los ASTs de `A , B` y `A OR B` son `reflect.DeepEqual` idénticos y el evaluador no puede distinguirlos: `OR` se computa como `AND` y devuelve vacío. **(B1)**

Se resuelve de forma **aditiva** para no romper v1: se añade `Refinement.AttrSet` y `Refinement.Ungrouped` se sigue poblando (deprecado, retirado en la Task 18).

**Files:**
- Modify: `ecl/ast/nodes.go:89-116`
- Modify: `ecl/parser.go:967-999` (`collectAttributes`), `:905-932` (`collectSubrefinement`)
- Modify: `ecl/evaluator.go:264-312` (`applyRefinement`)
- Modify: `ecl/parser_test.go`, `ecl/evaluator_refined_test.go`

**Step 1: Test que falla**

```go
func TestParse_ConjunctionAndDisjunctionDiffer(t *testing.T) {
	andExpr, err := Parse("< 404684003 : 363698007 = 39057004 , 116676008 = 415582006")
	require.NoError(t, err)
	orExpr, err := Parse("< 404684003 : 363698007 = 39057004 OR 116676008 = 415582006")
	require.NoError(t, err)
	require.False(t, reflect.DeepEqual(andExpr, orExpr),
		"AND y OR producen el mismo AST: la polaridad se pierde en el parser")
}
```

Y en el evaluador, contra el fixture del repo:

```go
func TestEvaluate_UngroupedDisjunction(t *testing.T) {
	// 22298006 tiene 363698007=74281007; 73211009 tiene 363698007=113331007.
	set := evalFixture(t, "* : 363698007 = 74281007 OR 363698007 = 113331007")
	require.ElementsMatch(t, []string{"22298006", "73211009"}, set.Slice())
}
```

**Step 2: Nodo nuevo en el AST**

```go
// AttrSetOp is the boolean operator joining the items of an AttributeSet.
type AttrSetOp string

const (
	AttrSetAnd AttrSetOp = "AND"
	AttrSetOr  AttrSetOp = "OR"
)

// AttributeSet is a boolean tree of attribute clauses. It preserves the
// operator and the parenthesised nesting that the ECL grammar allows, which a
// flat []*Attribute cannot express.
//
// Exactly one of Attr, Group or Items is set:
//   - Attr:  a single attribute clause (leaf)
//   - Group: a curly-brace attribute group (leaf)
//   - Items: a nested set combined with Op
type AttributeSet struct {
	Op    AttrSetOp
	Attr  *Attribute
	Group *AttributeGroup
	Items []*AttributeSet
}

func (*AttributeSet) eclNode() {}
```

Y en `Refinement`:

```go
type Refinement struct {
	// AttrSet is the boolean tree of attribute clauses. Prefer it over the
	// fields below, which are kept for backwards compatibility.
	AttrSet *AttributeSet

	// Deprecated: use AttrSet. Populated for compatibility with v1.1 readers;
	// it cannot represent OR and will be removed in v2.
	Groups      []*AttributeGroup
	// Deprecated: use AttrSet.
	Ungrouped   []*Attribute
	Conjunction []*Refinement
	Disjunction []*Refinement
}
```

**Step 3: Construir el árbol en el parser**

`collectAttributes` devuelve ahora `*ast.AttributeSet`. La regla `eclattributeset` de la gramática admite conjunción **o** disyunción, nunca ambas al mismo nivel, así que el operador es determinable:

```go
func (v *astBuilder) collectAttributeSet(ctx grammar.IEclattributesetContext) *ast.AttributeSet {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.EclattributesetContext)
	if !ok {
		return nil
	}

	items := []*ast.AttributeSet{}
	if concrete.Subattributeset() != nil {
		if s := v.collectSubAttributeSet(concrete.Subattributeset()); s != nil {
			items = append(items, s)
		}
	}

	op := ast.AttrSetAnd
	switch {
	case concrete.Conjunctionattributeset() != nil:
		conj := concrete.Conjunctionattributeset().(*grammar.ConjunctionattributesetContext)
		for _, sub := range conj.AllSubattributeset() {
			if s := v.collectSubAttributeSet(sub); s != nil {
				items = append(items, s)
			}
		}
	case concrete.Disjunctionattributeset() != nil:
		op = ast.AttrSetOr
		disj := concrete.Disjunctionattributeset().(*grammar.DisjunctionattributesetContext)
		for _, sub := range disj.AllSubattributeset() {
			if s := v.collectSubAttributeSet(sub); s != nil {
				items = append(items, s)
			}
		}
	}

	if len(items) == 1 {
		return items[0] // hoja: sin nodo booleano envolvente
	}
	return &ast.AttributeSet{Op: op, Items: items}
}
```

`collectSubAttributeSet` devuelve una hoja `{Attr: …}` o recursivamente el árbol del `eclattributeset`
entre paréntesis (`parser.go:1015-1017`). **No** puede devolver una hoja `{Group: …}`: ver el Step 4b.

Aparte, y en el mismo Step: la rama `case concrete.Eclrefinement() != nil` de `collectSubrefinement`
(`parser.go:922-931`) hoy **funde** `inner.Groups/Ungrouped/Conjunction/Disjunction` en el padre,
destruyendo el ámbito del paréntesis. Debe dejar de aplanar y guardar el refinamiento anidado como un
sub-nodo único, propagando también `inner.AttrSet`:

```go
	case concrete.Eclrefinement() != nil:
		if inner := v.visitRefinement(concrete.Eclrefinement()); inner != nil {
			// Un paréntesis es un ámbito: guardarlo entero, no fundirlo.
			ref.Conjunction = append(ref.Conjunction, inner)
		}
```

> Son **dos sitios distintos** de aplanamiento y la v1 de este plan los confundía: el paréntesis de
> `eclattributeset` se aplana en `collectSubAttributes` (`parser.go:1016`), y el de `eclrefinement` en
> `collectSubrefinement` (`:922-931`). Ir al segundo buscando el primero hace perder el tiempo.

En `collectSubrefinement`, poblar los dos mundos:

```go
case concrete.Eclattributeset() != nil:
	set := v.collectAttributeSet(concrete.Eclattributeset())
	ref.AttrSet = mergeAttrSet(ref.AttrSet, set) // AND si ya había algo
	ref.Ungrouped = append(ref.Ungrouped, flattenAttrs(set)...) // deprecado
```

`flattenAttrs` replica el comportamiento v1.1 exactamente, de forma que un consumidor que lea `Ungrouped` vea lo mismo que antes. `mergeAttrSet(a, b)` devuelve `b` si `a == nil`, y en otro caso `&AttributeSet{Op: AttrSetAnd, Items: []*AttributeSet{a, b}}`.

**Step 4: Evaluar el árbol contra el foco original — como etapa del pipeline, NO con `return`**

> **C1 — el error fatal de la v1 de este plan.** La versión anterior escribía
> `return applyAttrSet(ctx, focus, ref.AttrSet, provider)`. Eso es incorrecto porque
> `visitRefinement` (`parser.go:875-895`) llama a `collectSubrefinement` sobre el **mismo** `ref` y
> después le añade `Conjunction`/`Disjunction`; el Step 3 pone `AttrSet` en ese primer
> sub-refinamiento, así que `AttrSet != nil` **coexiste** con `Groups`/`Conjunction`/`Disjunction` y
> el `return` las saltaba. Medido: `* : 363698007 = 74281007 , { 116676008 = 55641003 }` produce
> `Ungrouped=1, Groups=0, Conjunction=1` (el grupo vive dentro de `Conjunction[0].Groups`), porque
> `subattributeset : eclattribute | (LEFT_PAREN ws eclattributeset ws RIGHT_PAREN)` (`ECL.g4:48`) no
> admite grupos y la coma sube forzosamente al nivel de `eclrefinement`. Con el `return`, la
> restricción del grupo se descartaba en silencio y la expresión devolvía un superconjunto.

`AttrSet` sustituye **solo** al bucle de `ref.Ungrouped`. Todo lo demás sigue igual:

```go
	result := focus

	// Camino nuevo: árbol booleano de atributos. Sustituye al bucle de
	// ref.Ungrouped, no al resto del pipeline.
	if ref.AttrSet != nil {
		r, err := applyAttrSet(ctx, result, ref.AttrSet, provider)
		if err != nil {
			return nil, err
		}
		result = r
	} else {
		for _, attr := range ref.Ungrouped { // camino deprecado, idéntico a hoy
			filtered, err := filterByAttribute(ctx, result, attr, provider)
			if err != nil {
				return nil, err
			}
			result = filtered
		}
	}

	// … y a continuación, SIN CAMBIOS: ref.Groups, ref.Conjunction, y
	// ref.Disjunction con la corrección del Step 5.
```

**Step 4b: `AttributeSet.Group` es inalcanzable — bórralo o justifícalo**

La gramática distingue los dos niveles: `subrefinement` admite `eclattributegroup` (`ECL.g4:44`),
pero `subattributeset` **no** (`:48`). Por tanto `collectSubAttributeSet` no puede producir una hoja
`{Group: …}` y la rama `case set.Group != nil` de `applyAttrSet` es código muerto. Eliminar el campo
`Group` del tipo, o dejarlo documentado como reservado con un comentario que explique por qué no se
puebla. La v1 de este plan afirmaba que sí se poblaba: era falso.

**Step 4c: `OR` dentro de un grupo — decisión requerida**

`{ A = x OR B = y }` está hoy roto por la misma causa (medido: `* : { 363698007 = 113331007 OR
116676008 = 55641003 }` → `∅`) y el árbol del Step 3 no lo alcanza, porque
`AttributeGroup.Attrs` sigue siendo `[]*Attribute`. Añadir de forma aditiva:

```go
type AttributeGroup struct {
	// AttrSet is the boolean tree of the group's clauses. Prefer it over Attrs.
	AttrSet     *AttributeSet
	// Deprecated: use AttrSet. Cannot represent OR.
	Attrs       []*Attribute
	Cardinality *Cardinality
}
```

y mantener `collectAttributes` con su firma actual para no romper `parser.go:953` y `:1016`,
poblando ambos campos con `flattenAttrs`. Si se decide **no** arreglar el `OR` intra-grupo en esta
tarea, hay que declararlo como limitación conocida en el CHANGELOG en lugar de dejarlo silencioso.

```go
// applyAttrSet filters focus by a boolean tree of attribute clauses.
// Every branch is evaluated against the SAME incoming focus — that is what
// makes OR a union instead of a chained intersection.
func applyAttrSet(ctx context.Context, focus Set, set *ast.AttributeSet, provider DataProvider) (Set, error) {
	switch {
	case set == nil:
		return focus, nil
	case set.Attr != nil:
		return filterByAttribute(ctx, focus, set.Attr, provider)
	case set.Group != nil:
		return filterByAttributeGroup(ctx, focus, set.Group, provider)
	}

	if set.Op == ast.AttrSetOr {
		acc := NewSet()
		for _, item := range set.Items {
			sub, err := applyAttrSet(ctx, focus, item, provider) // focus, NO acc
			if err != nil {
				return nil, err
			}
			acc = acc.Union(sub)
		}
		return acc, nil
	}

	result := focus
	for _, item := range set.Items {
		sub, err := applyAttrSet(ctx, result, item, provider)
		if err != nil {
			return nil, err
		}
		result = sub
	}
	return result, nil
}
```

**Step 5: El mismo defecto a nivel de sub-refinamiento**

`applyRefinement:299-309` toma la unión de `ref.Disjunction` sobre un `result` **ya filtrado** por `ref.Groups`/`ref.Ungrouped`, dando `focus ∩ A ∩ B`. Corregir capturando el foco entrante:

```go
	if len(ref.Disjunction) > 0 {
		acc := NewSet()
		for _, sub := range ref.Disjunction {
			subResult, err := applyRefinement(ctx, focus, sub, provider) // focus, no result
			if err != nil {
				return nil, err
			}
			acc = acc.Union(subResult)
		}
		// Los disyuntos son alternativas al primer sub-refinamiento, que ya
		// filtró `result`: la unión incluye ese resultado.
		result = result.Union(acc)
	}
```

> **C3 — invariante obligatorio.** Ese `Union` solo es correcto si el nodo **no** tiene `Conjunction`
> poblada a la vez. Sin la corrección del Step 3, un paréntesis aplanado deja `Groups`,
> `Conjunction` y `Disjunction` en el mismo nodo y el `Union` sobre-genera: medido,
> `< 404684003 : ({ 363698007 = 74281007 } OR { 116676008 = 55641003 }) , 1142139005 = #2` da hoy
> `[22298006]` (correcto: `(gA ∪ gB) ∩ C`), y con el `Union` sin la corrección del Step 3 daría
> `[22298006 404684004]`, colando un concepto que no cumple `C`. Con el Step 3 corregido el invariante
> queda garantizado; añadir además un guard defensivo:
>
> ```go
> if len(ref.Conjunction) > 0 && len(ref.Disjunction) > 0 {
> 	return nil, fmt.Errorf("refinamiento con conjunción y disyunción en el mismo nodo: el ámbito del paréntesis se perdió al parsear")
> }
> ```
>
> Y el caso de conformidad de «disyunción anidada entre paréntesis» del Step 6 debe escribirse **con un
> grupo dentro** del paréntesis: es la única forma que alcanza la rama `Eclrefinement`.

**Step 6: Casos de conformidad**

Añadir a `testdata/conformance/cases/04-refinement.yaml`: disyunción sin agrupar, disyunción de grupos, disyunción anidada entre paréntesis, y la conjunción equivalente para probar que ya no coinciden.

```bash
go test ./ecl/ -run 'TestParse_Conjunction|TestEvaluate_Ungrouped' -v
go test ./...
```

---

### Task 4: Unificar la semántica de `!=`

`filterByAttribute:389-392` invierte el booleano de cardinalidad (`keep = !keep`), lo que implementa `[0..0] attr = X` en lugar de `attr != X`: incluye los conceptos que no tienen el atributo y excluye los que tienen otro valor. La ruta agrupada del mismo archivo (`groupSatisfiesClauses:589-597`) ya lo hace bien. **(B3)**

**Files:**
- Modify: `ecl/evaluator.go:357-423` (incluye la rama reverse de `:363-378`, no solo la forward)
- Modify: `ecl/evaluator_refined_test.go:31-42` (codifica el bug)

> **Alcance real: hay CUATRO rutas de `!=`, no dos.** Forward sin agrupar (`:389-392`, la que esta tarea
> arregla), agrupada (`groupSatisfiesClauses:589-597`, ya correcta), reverse suelta
> (`:363-378`, `return focus.Minus(inbound)` — implementa la semántica que este plan declara equivocada)
> y `conceptMatchesGroupWithReverse`, que **ignora `c.op` por completo**. Medido:
> `<< 138875005 : R 363698007 != 22298006` devuelve los 7 conceptos del foco. Esta tarea **no** arregla
> las dos rutas reverse: requieren el método de provider de la Task 19. Por tanto el CHANGELOG **no**
> puede anunciar `!=` como «unificado», y `R attr != X` va a la lista de limitaciones conocidas.

**Step 1: Test que falla — nombre honesto sobre lo que compara**

```go
// Compara la ruta forward sin agrupar con la agrupada, que ya es correcta y sirve
// de oráculo. NO cubre las dos rutas reverse: ver la nota de alcance de esta tarea.
func TestEvaluate_NotEqualsUngroupedMatchesGroupedOracle(t *testing.T) {
	ungrouped := evalFixture(t, "<< 404684003 : 363698007 != 74281007")
	grouped := evalFixture(t, "<< 404684003 : { 363698007 != 74281007 }")
	require.ElementsMatch(t, grouped.Slice(), ungrouped.Slice(),
		"las rutas agrupada y sin agrupar dan conjuntos distintos para la misma cláusula")
	// 73211009 tiene 363698007=113331007 y 363698007=74281007: tiene al menos
	// un finding site que no es 74281007, así que debe estar.
	require.Contains(t, ungrouped.Slice(), "73211009")
	// 404684003 no tiene ninguna relación 363698007: no debe estar.
	require.NotContains(t, ungrouped.Slice(), "404684003")
}
```

**Step 2: Devolver también el total del tipo**

```go
// conceptMatchesAttribute counts the relationships of the concept whose type is
// in typeIDs, returning both the number that match the value set and the total
// of that type. The total is what "!=" needs: the ECL negation is over the
// VALUE, so it counts relationships of the type whose target is NOT in the set.
func conceptMatchesAttribute(ctx context.Context, conceptID string, typeIDs, valueSet Set, valueIsAny bool, provider DataProvider) (matching, totalOfType int, err error) {
	groups, err := provider.PropertiesByGroup(ctx, conceptID)
	if err != nil {
		return 0, 0, fmt.Errorf("PropertiesByGroup(%s): %w", conceptID, err)
	}
	for _, rels := range groups {
		for _, r := range rels {
			if !typeIDs.Contains(r.TypeID) {
				continue
			}
			totalOfType++
			if valueIsAny || valueSet.Contains(r.TargetID) {
				matching++
			}
		}
	}
	return matching, totalOfType, nil
}
```

**Step 3: Aplicar la cardinalidad sobre el conteo correcto**

```go
	focus.Iter(func(id string) bool {
		matching, totalOfType, err := conceptMatchesAttribute(ctx, id, typeIDs, valueSet, valueIsAny, provider)
		if err != nil {
			iterErr = err
			return false
		}
		count := matching
		if attr.Op == "!=" {
			// "attr != X" selecciona conceptos con al menos una relación de ese
			// tipo cuyo valor NO está en X. No es el complemento del concepto.
			count = totalOfType - matching
		}
		if cardinalitySatisfied(attr.Cardinality, count) {
			out.m[id] = struct{}{}
		}
		return true
	})
```

**Step 4: Corregir el test que codifica el bug**

`ecl/evaluator_refined_test.go:31-42` justifica el resultado erróneo con «Per our AND-across-focus semantics». Reescribir la expectativa según la spec y dejar en el comentario por qué cambió, citando esta tarea.

**Step 5: Casos de conformidad**

`04-refinement.yaml` no tiene ni un caso con `!=`. Añadir: `!=` sin agrupar, `!=` agrupado, `!=` con cardinalidad explícita, y `!=` contra un concepto sin ninguna relación de ese tipo.

```bash
go test ./ecl/ -run TestEvaluate_NotEquals -v && go test ./...
```

---

### Task 5: Cardinalidad en grupos y en valores concretos

`AttributeGroup.Cardinality` no se lee ni una vez en el evaluador aunque el parser lo rellena, y `filterByConcreteValue` ignora `attr.Cardinality`. `[0..0]` devuelve por tanto el conjunto exactamente inverso al pedido. **(B5, parcial)**

**Files:**
- Modify: `ecl/evaluator.go:443-538` (`filterByAttributeGroup`)
- Modify: `ecl/evaluator.go:1020-1113` (`filterByConcreteValue`)
- Modify: `ecl/evaluator_refined_test.go`

**Step 1: Tests que fallan**

```go
func TestEvaluate_GroupCardinalityZero(t *testing.T) {
	with := evalFixture(t, "* : { 363698007 = 74281007 }")
	without := evalFixture(t, "* : [0..0] { 363698007 = 74281007 }")
	require.NotEqual(t, with.Slice(), without.Slice(), "[0..0] devuelve el mismo conjunto que [1..*]")
	for _, id := range with.Slice() {
		require.NotContains(t, without.Slice(), id)
	}
}

func TestEvaluate_ConcreteCardinalityZero(t *testing.T) {
	with := evalFixture(t, "* : 1142139005 = #2")
	without := evalFixture(t, "* : [0..0] 1142139005 = #2")
	for _, id := range with.Slice() {
		require.NotContains(t, without.Slice(), id)
	}
}
```

**Step 2: Contar grupos coincidentes en lugar de cortar en el primero**

El `break` de `:514-519` hace imposible cualquier cardinalidad de grupo. Nótese que `[0..0]` exige recorrer **todos** los grupos, y que un concepto sin ningún grupo también satisface `[0..0]`:

```go
		if !hasReverse {
			groups, err := provider.PropertiesByGroup(ctx, id)
			if err != nil {
				iterErr = fmt.Errorf("PropertiesByGroup(%s): %w", id, err)
				return false
			}
			matchingGroups := 0
			for gnum, rels := range groups {
				if gnum == 0 {
					continue // el grupo 0 es "sin agrupar": no es un grupo de relación
				}
				if groupSatisfiesClauses(rels, clauses) {
					matchingGroups++
				}
			}
			// nil se trata como [1..*], igual que en cardinalitySatisfied.
			anyGroupSatisfies = cardinalitySatisfied(grp.Cardinality, matchingGroups)
		}
```

> **Decisión a documentar:** excluir el grupo 0 del conteo cambia el comportamiento actual, que lo
> incluía. La spec de ECL **no se pronuncia** sobre si los atributos sin agrupar cuentan como grupo a
> efectos de cardinalidad, así que esto es una decisión de esta librería, no una lectura normativa: se
> alinea con el contrato que ya publicamos en `provider.go:47-48` («*Group 0 means "ungrouped"*») y con
> que los atributos sin agrupar se evalúan por la ruta de `filterByAttribute`. Añadir un caso de
> conformidad que lo fije.

**Step 3: La ruta reverse dentro de grupos no puede dar un conteo real**

`conceptMatchesGroupWithReverse:609+` devuelve un `bool`. Cambiar la firma a
`(matchingGroups int, err error)` y aplicar `cardinalitySatisfied` en el llamador, de modo que las dos
ramas de `filterByAttributeGroup` compartan la decisión.

> **Limitación que hay que escribir en el código, no descubrir después.** Esa función recorre los grupos
> de los conceptos **origen** (`RelationshipSources` en `:615`, `PropertiesByGroup(srcID)` en `:629-640`),
> no los del foco. Cualquier `matchingGroups` derivado de ahí cuenta grupos ajenos, así que
> `[2..*] { R … }` daría resultados arbitrarios. Devolver `1`/`0` (equivalente al `bool` actual) y
> añadir un comentario que diga explícitamente que no es un conteo de grupos del foco; la cardinalidad
> de grupo con cláusulas reverse queda en limitaciones conocidas hasta la Task 19.

**Step 4: Contar valores concretos — sin invertir `!=`**

`filterByConcreteValue` decide hoy con `if matched { out.m[id] = struct{}{} }`. Contar los valores que
satisfacen la comparación y pasar **ese** número a `cardinalitySatisfied(attr.Cardinality, count)`.

> **C2 — NO aplicar aquí la corrección de `!=` de la Task 4.** La v1 de este plan lo ordenaba y era un
> error. A diferencia de la ruta concept-valued, donde el conteo es de pertenencia al conjunto, en la
> ruta concreta el operador **ya está dentro** de la comparación: `compareFloat(f, attr.Op, numeric)`
> devuelve `f != numeric` cuando el operador es `!=` (`evaluator.go:1116-1132`; igual `compareString` y
> `compareBool`), y el propio código lo documenta en `:1101-1103`: *«No extra inversion for "!="»*.
> Restar del total da el conteo **inverso**. Verificado que hoy es correcto y que la inversión lo
> rompería, incluido el caso verde `07-concrete.yaml:19` (`1142139005 != #6` → `["73211009"]`):
>
> ```
> * : 1142139005 = #5    → 73211009   ✅
> * : 1142139005 != #5   → ∅          ✅   (con la inversión daría 73211009)
> * : 1142139005 != #2   → 73211009   ✅   (con la inversión daría ∅)
> ```
>
> Tests obligatorios en el Step 1: los tres de arriba más `* : [0..0] 1142139005 != #2` (sin `73211009`).
> Además `<`, `<=`, `>`, `>=` no se invertirían nunca, así que la inversión dejaría dos convenciones de
> conteo incoherentes dentro de la misma función.

**Step 5: Casos de conformidad**

No hay ni un caso de cardinalidad de grupo en todo `testdata/`. Añadir a `04-refinement.yaml`: `[0..0]`, `[1..1]`, `[2..*]` sobre grupos, y `[0..0]` sobre un valor concreto. Requiere ampliar el fixture (Task 11): hoy las 4 relaciones están todas en el mismo grupo, así que la cardinalidad de grupo es estructuralmente inverificable.

```bash
go test ./ecl/ -run 'TestEvaluate_.*Cardinality' -v && go test ./...
```

---

### Task 6: Top y Bottom transitivos, en una llamada por lote

`topOfSet`/`bottomOfSet` consultan `Parents`/`Children` (profundidad 1) donde la spec exige supertipos y subtipos transitivos, y lo hacen **una llamada por concepto**. Arreglar la semántica elimina además dos de los tres N+1 medidos. **(B7)**

**Files:**
- Modify: `ecl/evaluator.go:1160-1208`
- Modify: `ecl/evaluator_advanced_test.go`

**Step 1: Test que falla — el par tiene que ser abuelo/nieto, no padre/hijo**

> **El test de la v1 de este plan pasaba hoy**, así que habría cerrado la tarea sin evidencia. Su
> comentario también era falso: el padre directo de `22298006` es `404684003`, no `73211009`
> (`standard.yaml:43`). Como `404684003` **sí** está en el conjunto, `Parents` ya lo excluye y
> `!!> (404684003 OR 22298006)` devuelve `[404684003]` hoy. Para que el bug se manifieste hace falta que
> el eslabón intermedio **falte** en el conjunto: la cadena del fixture es
> `138875005 → 404684003 → 22298006`, así que el par discriminante es abuelo y nieto sin el padre.

```go
func TestEvaluate_TopUsesTransitiveAncestors(t *testing.T) {
	// Cadena del fixture: 138875005 → 404684003 → 22298006. El conjunto omite el
	// eslabón intermedio, así que el padre directo de 22298006 no está dentro pero
	// su abuelo sí: con Parents (profundidad 1) se cuela como top.
	// Hoy devuelve [138875005 22298006]; correcto: [138875005].
	set := evalFixture(t, "!!> (138875005 OR 22298006)")
	require.ElementsMatch(t, []string{"138875005"}, set.Slice())
}

func TestEvaluate_BottomUsesTransitiveDescendants(t *testing.T) {
	// Simétrico: bottomOfSet está igual de roto y la v1 de este plan no le
	// proponía ni un test. Hoy devuelve [138875005 22298006]; correcto: [22298006].
	set := evalFixture(t, "!!< (138875005 OR 22298006)")
	require.ElementsMatch(t, []string{"22298006"}, set.Slice())
}
```

Añadir los dos como casos de conformidad en `01-hierarchy.yaml` — es la suite que CI ya ejecutaba antes
de la Task 1, así que quedan protegidos de inmediato.

**Step 2: Una sola llamada, `Ancestors` en vez de `Parents`**

```go
// topOfSet returns the members of baseSet that have no proper ancestor inside
// baseSet. Uses Ancestors (transitive) because a member whose direct parent is
// outside the set may still have a grandparent inside it.
func topOfSet(ctx context.Context, baseSet Set, provider DataProvider) (Set, error) {
	if baseSet == nil || baseSet.Len() == 0 {
		return baseSet, nil
	}
	ids := baseSet.Slice()
	out := newMapSet()
	for _, id := range ids {
		anc, err := provider.Ancestors(ctx, []string{id}, false)
		if err != nil {
			return nil, fmt.Errorf("Ancestors(%s): %w", id, err)
		}
		if nonNil(anc).Intersect(baseSet).Len() == 0 {
			out.m[id] = struct{}{}
		}
	}
	return out, nil
}
```

> **Nota de rendimiento:** sigue siendo una llamada por concepto porque `Ancestors` colapsa la unión de los ancestros de todos los inputs y se pierde a quién pertenece cada uno. Resolverlo de verdad exige una firma que devuelva `map[string]Set`, y eso rompe la API: queda en la Task 19 (v2). Este paso corrige la **semántica** ya; el N+1 se reduce de dos sitios a uno.

`bottomOfSet` es simétrico con `Descendants`.

```bash
go test ./ecl/ -run TestEvaluate_Top -v && go test ./...
```

---

### Task 7: Negación de filtros por cláusula

`buildDescriptionFilterOpts` y `buildConceptFilterOpts` colapsan todas las cláusulas de un `{{ }}` en unas únicas `Opts` más **un solo** `bool negate`, activado por cualquier cláusula con `!=`. El evaluador entonces resta el conjunto que satisface la conjunción completa, así que las cláusulas positivas hermanas se pierden. **(B4)**

La corrección se **divide por familia**, y la distinción es la clave de esta tarea:

- **Filtros de concepto** (`{{ C … }}`): cada cláusula identifica un conjunto de *conceptos*. Componer por cláusula con `Intersect`/`Minus` es exactamente correcto y **no necesita tocar el provider**. Se arregla ahora.
- **Filtros de descripción** (`{{ D … }}`): la negación es a nivel de *fila de descripción*. `{{ D type != fsn }}` significa «tiene alguna descripción cuyo tipo no es FSN», que **no** es `Minus(conceptos con FSN)` — eso excluiría a un concepto que tiene FSN y además un sinónimo. Requiere que el provider niegue en la fila, y eso es un cambio de contrato. Aquí se hace **ruidoso**, y se implementa en la Task 19 (v2).

**Files:**
- Modify: `ecl/evaluator.go:779-843` (`evaluateFiltered`)
- Modify: `ecl/evaluator.go:866-959` (`build*FilterOpts`)
- Modify: `ecl/evaluator_filters_test.go:271` (`{{ term != "infarction" }}`), `:281` (`{{ language != es }}`)
- Move: caso `term filter negated` de `testdata/conformance/cases/05-filters.yaml:34-36` → `09-errors.yaml` con `expectError: true`

> **Tres artefactos verdes hoy que esta tarea pone en rojo.** El guard del Step 3 alcanza a **toda** la
> familia de descripción, no solo a `type !=`. Verificado que hoy pasan: los dos tests de
> `evaluator_filters_test.go` y el caso `<< 73211009 {{ D term != "Type 1 diabetes mellitus" }}` →
> `["73211009"]` de `05-filters.yaml`, dentro de los `44 passed, 0 failed`. Convertirlos en el **mismo
> commit** que el guard, con entrada de CHANGELOG bajo «Behaviour change»: pasar de un resultado
> incorrecto a un error es una mejora, pero es observable. Si no se mueven en el mismo commit, la Task 1
> los acaba de congelar y CI queda rojo.

**Step 1: Tests que fallan**

```go
func TestEvaluate_ConceptFilterMixedPolarity(t *testing.T) {
	// active = true debe seguir aplicándose aunque la otra cláusula sea !=.
	set := evalFixture(t, "< 138875005 {{ C active = true, moduleId != 900000000000207008 }}")
	for _, id := range set.Slice() {
		require.True(t, isActiveInFixture(t, id), "sobrevivió un concepto inactivo: %s", id)
	}
}

func TestEvaluate_DescriptionNegationIsLoud(t *testing.T) {
	_, err := evalFixtureErr(t, "<< 404684003 {{ D type != fsn }}")
	require.ErrorIs(t, err, ErrUnsupportedFeature,
		"la negación a nivel de descripción devolvió un conjunto en lugar de un error")
}
```

**Step 2: Una `Opts` por cláusula, no una acumulada**

Cambiar `buildConceptFilterOpts` para que devuelva `[]conceptClause`, donde cada elemento lleva su propia `ConceptFilterOpts` y su operador:

```go
// conceptClause is a single concept filter clause with its own polarity.
type conceptClause struct {
	opts   ConceptFilterOpts
	negate bool
}
```

Y en `evaluateFiltered`, componer:

```go
	for _, cl := range conceptClauses {
		matched, err := provider.FilterConcepts(ctx, result, cl.opts)
		if err != nil {
			return nil, fmt.Errorf("FilterConcepts: %w", err)
		}
		if cl.negate {
			result = result.Minus(nonNil(matched))
		} else {
			result = nonNil(matched)
		}
	}
```

Coste: una llamada al provider por cláusula en lugar de una por filtro. Es aceptable — los filtros de concepto rara vez pasan de tres cláusulas, y la alternativa es un resultado incorrecto.

**Step 3: Centinela de error y rechazo explícito en descripción**

En `ecl/evaluator.go`, junto a los tipos:

```go
// ErrUnsupportedFeature marks an ECL construct the evaluator recognises but
// cannot evaluate correctly yet. Callers can classify it with errors.Is and
// return 501/422 instead of a wrong result set.
var ErrUnsupportedFeature = errors.New("unsupported ECL feature")
```

En la rama de filtros de descripción, si alguna cláusula tiene `Op == "!="` — incluyendo
explícitamente `*ast.DialectFilter`, que pertenece a la familia `desc` (`evaluator.go:851`) pero
`buildDescriptionFilterOpts` salta en `:906-908` y se evalúa en su propio bucle:

```go
		return nil, fmt.Errorf("%w: negated description filter (%s) requires row-level negation in the DataProvider; see DescriptionFilterOpts in v2", ErrUnsupportedFeature, kind)
```

> **El dialecto tiene hoy dos dueños de la misma negación** y esta tarea debe elegir uno.
> `buildDialectFilterOpts:964` ya pide la negación al provider (`DialectFilterOpts{Negate: df.Op == "!="}`),
> mientras `evaluator.go:817-818` la vuelve a aplicar con `result.Minus(matches)` y
> `internal/conformance/fixture.go:566-590` ignora el flag. Cuando la Task 8 pueble `Op` de verdad,
> esa doble negación se activa. Decidir: o el provider niega (y se borra el `Minus` de `:817`), o el
> evaluador niega (y se borra el flag de las `Opts`) — y alinear el fixture con la elección.

Esto sustituye un resultado incorrecto silencioso por un error clasificable. Es el patrón que el propio repo ya aplica bien en `evaluator.go:152-154` para `^[field]`.

**Step 4: Casos de conformidad**

`09-errors.yaml` gana los casos de negación de descripción con `expectError: true`; `05-filters.yaml` gana los de polaridad mixta en filtros de concepto.

```bash
go test ./ecl/ -run 'TestEvaluate_.*Filter' -v && go test ./...
```

---

### Task 8: Ramas del parser sin modelar: poblarlas o fallar en voz alta

Tres formas del mismo patrón: el parser descarta información y el evaluador no lo señala. **(B6)**

**Files:**
- Modify: `ecl/parser.go:387-391` (dialecto), `:411` (descriptionIdFilter), `:519` (operandos de conjunto), `:547` (escapes)
- Modify: `ecl/ast/nodes.go:146-198` (campos escalares → slices)
- Modify: `ecl/evaluator.go:802-822`

**Step 1: Tests que fallan**

```go
func TestParse_DialectFilterIsPopulated(t *testing.T) {
	e, err := Parse("<< 404684003 {{ D dialect = en-gb }}")
	require.NoError(t, err)
	f := e.(*ast.Filtered).Filters[0].(*ast.DialectFilter)
	require.NotEmpty(t, f.Dialects, "el contenido del filtro de dialecto se descartó")
}

func TestParse_DescriptionIDFilterIsNotDropped(t *testing.T) {
	withFilter, err := Parse("< 404684003 {{ D id = 123456789012 }}")
	if err == nil {
		bare, _ := Parse("< 404684003")
		require.False(t, reflect.DeepEqual(withFilter, bare),
			"el filtro se evaporó: la consulta devuelve un superconjunto")
	}
}

func TestParse_TermSetAndEscapes(t *testing.T) {
	e, err := Parse(`< 404684003 {{ term = ("heart" "attack") }}`)
	require.NoError(t, err)
	tf := e.(*ast.Filtered).Filters[0].(*ast.TermFilter)
	require.ElementsMatch(t, []string{"heart", "attack"}, tf.Terms)

	e, err = Parse(`< 404684003 {{ term = "a\"b" }}`)
	require.NoError(t, err)
	tf = e.(*ast.Filtered).Filters[0].(*ast.TermFilter)
	require.Equal(t, `a"b`, tf.Terms[0], "el escape no se decodificó")
}
```

**Step 2: Poblar el filtro de dialecto**

Reemplazar el placeholder de `:387-391` por la visita real de `dialectidfilter` / `dialectaliasfilter`, incluyendo operador y acceptability, hacia `ast.DialectEntry` — el tipo ya existe y `buildDialectFilterOpts` ya sabe consumirlo. Con `Op` correctamente poblado, la rama `!=` de `evaluator.go:817` deja de ser código muerto.

**Step 3: Campos escalares a slices**

Aditivo: añadir `Terms []string`, `Values []string`, `Modules []Expression`, `DefinitionStatuses []Expression`, dejando los escalares como deprecados y poblados con el primer elemento. Recorrer las slices en `build*FilterOpts` con semántica any-of, que las `Opts` ya documentan.

**Step 4: Decodificar escapes**

Un helper en el parser, aplicado en todo literal de string y en los términos:

```go
// unescapeECLString decodes the \" \\ \* escapes the ECL grammar defines.
// The raw token text keeps the backslashes, so a term filter that reaches the
// provider un-decoded never matches.
func unescapeECLString(s string) string { … }
```

**Step 5: Fallar en voz alta lo que quede sin modelar**

Auditar cada rama de `buildDescriptionFilterClauses` y del visitor: toda rama de la gramática reconocida y no modelada debe devolver `fmt.Errorf("%w: …", ErrUnsupportedFeature)` en lugar de no emitir nada. Y en `evaluateFiltered`, un `DialectFilter` con `Dialects` vacío es un error, no una intersección con vacío.

**Step 6: `memberField` con valor literal — hoy no tiene dueño en el plan**

La fila 10 de la tabla de aceptación (`{{ M mapTarget = "X" }}`) es un fallo del **evaluador**, no del
fixture: `evaluator.go:748-752` llama a `Evaluate(ctx, field.Value, provider)` sobre un
`*ast.StringValue` (`parser.go:816-818`), que cae en el default de `:239`. Reproducido:
`error: evaluate: evaluating member filter value: unsupported AST node type: *ast.StringValue`.

Enrutar los literales sin pasar por `Evaluate`:

```go
	switch v := field.Value.(type) {
	case *ast.StringValue, *ast.IntegerValue, *ast.DecimalValue, *ast.BooleanValue:
		// Los campos de miembro no-SCTID comparan valores literales, no conceptos.
		opts.ValueSet = NewSetFromSlice([]string{literalText(v)})
	default:
		opts.ValueSet, err = Evaluate(ctx, field.Value, provider)
	}
```

Y documentar en el godoc de `MemberFilterOpts.ValueSet` que para campos no-SCTID contiene los valores
literales — que es exactamente lo que `internal/conformance/fixture.go:595-599` ya implementa, así que
el cambio es aditivo y el fixture no necesita tocarse.

**Step 7: Casos de conformidad**

`05-filters.yaml` gana casos de conjunto de términos y de escapes. El caso de dialecto requiere la
sección `dialects:` del fixture, ya creada en la **Task 1b**. El caso de `memberField` con string va a
`11-memberfilters.yaml`.

```bash
go test ./ecl/ -run TestParse -v && go test ./...
```

---

### Task 9: Defensa contra `nil`, contrato en el godoc, y cancelación

Tres huecos pequeños del mismo contrato. Ninguno cambia una firma. **(B9, parcial)**

**Files:**
- Modify: `ecl/provider.go:5-89` (godoc)
- Modify: `ecl/evaluator.go` (helper `nonNil` en cada call site; `ctx.Err()`)
- Modify: `ecl/provider_test.go`

**Step 1: Test que falla**

```go
// nilProvider devuelve (nil, nil) para todo, que es lo idiomático en Go para
// "no hay nada" y lo que el godoc actual no prohíbe.
func TestEvaluate_NilSetsFromProviderDoNotPanic(t *testing.T) {
	p := &nilProvider{}
	for _, expr := range []string{"^ 900000000000509007 AND << 404684003", "<< 404684003", "* : 363698007 = *"} {
		ast, err := Parse(expr)
		require.NoError(t, err)
		set, err := Evaluate(context.Background(), ast, p) // hoy: panic
		require.NoError(t, err)
		require.NotNil(t, set, "Evaluate devolvió un Set nil al llamador")
		require.Zero(t, set.Len())
	}
}

func TestEvaluate_RespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Evaluate(ctx, mustParse(t, "< 138875005 : 363698007 = *"), countingProvider(t))
	require.ErrorIs(t, err, context.Canceled)
}
```

**Step 2: Normalizar en un solo punto**

```go
// nonNil normalises a Set returned by a DataProvider. The interface documents
// that implementations must return a non-nil Set, but a wrong provider should
// yield an empty result, never a panic inside the evaluator.
func nonNil(s Set) Set {
	if s == nil {
		return NewSet()
	}
	return s
}
```

Envolver cada retorno del provider y el retorno de `Evaluate`. El evaluador ya tolera `nil` en seis sitios de forma dispersa (`:204`, `:265`, `:317`, `:1174`, `:1199`, `toIDSlice`): unificarlos en el helper y borrar los guards ad-hoc.

**Step 3: Escribir el contrato que hoy vive en `internal/`**

En el godoc de `DataProvider`, encabezando el interface:

```go
// Contract for every method:
//
//   - Never return a nil Set. Use ecl.NewSet() for the empty result; the
//     evaluator normalises nil defensively but implementations must not rely
//     on that.
//   - An empty (non-nil) input Set or slice yields the empty Set, never the
//     whole terminology.
//   - Results are unordered; the evaluator sorts for output.
//   - Hierarchy methods are transitive except Children/Parents, which are
//     depth 1 by definition.
```

Corregir además los tres puntos concretos que rompen a un implementador que siga el godoc al pie de la letra:

- `RelationshipTargets`: documentar `nil` como wildcard **en este método** — es el que lo recibe (`evaluator.go:367`), no `RelationshipSources`, donde está escrito hoy y que nunca lo recibe.
- `AllConcepts`: redefinir como «todos los conceptos existentes, activos o no» y dejar que `FilterConcepts` restrinja el eje `active`. Hoy dice «all active concepts», lo que hace que `* {{ C active = false }}` no pueda devolver nada. Actualizar `internal/conformance/fixture.go` en el mismo commit, y añadir el caso de conformidad.
- `HistoricalAssociations`: ver Task 10.

**Step 4: Cancelación**

`ctx.Err()` al entrar en `Evaluate` y dentro de cada callback de `Iter`, abortando con `return false` y propagando por el `iterErr` que ya existe (`:381-400`, `:504-537`, `:1063-1112`, `:1166-1182`).

```bash
go test ./ecl/ -run 'TestEvaluate_Nil|TestEvaluate_Respects' -v && go test ./...
```

---

### Task 10: Invertir la dirección de las asociaciones históricas

`provider.go:72` documenta «expands a set of inactive concepts to their historical replacements» y el fixture recorre `if h.Source != id`. La spec define el suplemento como `(X) OR (^ 900000000000527005 {{ M targetComponentId = (X) }})`: añade los **miembros** (inactivos) cuyo `targetComponentId` cae en X. Con la dirección actual, `{{ +HISTORY }}` es un no-op sobre cualquier conjunto de conceptos activos — el 100 % del uso real.

**Files:**
- Modify: `ecl/provider.go:70-73` (godoc)
- Modify: `internal/conformance/fixture.go:515-537`
- Modify: `testdata/conformance/cases/06-history.yaml`
- Modify: `testdata/conformance/fixtures/standard.yaml`

**Step 1: Test que falla**

```yaml
  - name: "history supplement añade el concepto inactivo reemplazado"
    expression: "22298006 {{ +HISTORY }}"
    expectedIds: ["22298006", "11111111"]   # hoy devuelve solo 22298006
```

**Step 2: Invertir el recorrido y corregir el godoc**

En el fixture, `if h.Target != id { continue }` emitiendo `h.Source`. En el godoc:

```go
	// HistoricalAssociations returns the inactive concepts that were replaced
	// by any of the given concepts, according to the profile (MIN, MOD, MAX,
	// ALL). Direction matters: the input is the set of (typically active)
	// concepts, and the result is the set of historical concepts pointing AT
	// them via targetComponentId.
```

**Step 3: Hacer que los perfiles discriminen**

Los 3 casos actuales parten del concepto inactivo para que la dirección errónea encaje, y el fixture tiene una sola asociación, así que `MIN`/`MOD`/`MAX` devuelven lo mismo: los casos son vacuos (probado por mutación). Añadir al fixture una asociación `SAME_AS`, una `REPLACED_BY` y una `WAS_A`, de forma que los tres perfiles den conjuntos distintos, y reescribir los casos partiendo de conceptos **activos**.

```bash
go run ./cmd/gofhir-ecl conformance -filter '^history' && go test ./...
```

---

### Task 11: Semántica del fixture y cobertura de filtros

Los pasos de ampliación del fixture se movieron a la **Task 1b** (varias tareas anteriores los
necesitan). Aquí quedan la semántica de `termMatches` y la barrida de cobertura.

El fixture es la especificación ejecutable de referencia de esta librería — el README lo ofrece como tal.

**Files:**
- Modify: `internal/conformance/fixture.go:422-435` (`termMatches`), `:488` (`FilterConcepts`)
- Add: `testdata/conformance/cases/10-conceptfilters.yaml`, `11-memberfilters.yaml`

**Step 1: `match` como prefijo de palabra, no como substring**

`termMatches` usa `strings.Contains`, así que `{{ D term = "infarction myocardial" }}` no encuentra nada (el orden es irrelevante en la spec) y `{{ D term = "farct" }}` sí encuentra (no debería). Implementar: tokenizar la búsqueda y exigir que cada token sea prefijo de algún token de la descripción. Eliminar la rama `case "regex":`, que es inalcanzable porque la gramática no acepta ese prefijo.

**Step 2: Cerrar los huecos de cobertura**

Siete de las nueve familias de filtros marcadas ✅ en el README no tienen ni un caso: `dialect`, `D type`, `C module`, `effectiveTime`, `definitionStatus`, `{{ M … }}`. Consistente con `buildTypeFilter`, `buildModuleFilter`, `buildEffectiveTimeFilter`, `buildDefinitionStatusFilter` y `buildDialectFilterOpts` a 0.0 % de cobertura. Añadir casos para cada una y para los 3 tipos concretos sin caso.

Corregir además `03-primitives.yaml:18`: el caso se llama «wildcard returns every active concept» y su expresión es `<< 11687002` — no contiene ningún `*`.

**Step 3: Retirar los ✅ que no queden respaldados**

Cualquier feature del README que al terminar esta tarea siga sin caso de conformidad pierde el ✅ y pasa a una sección de limitaciones conocidas. La tabla del README debe poder derivarse de `testdata/`.

```bash
go run ./cmd/gofhir-ecl conformance && go test ./... && go test -coverprofile=c.out ./ecl/
```

---

### Task 12: MRCM — dominios en OR, cardinalidad mínima, reglas inválidas

**Files:**
- Modify: `mrcm/validator.go:118-210`, `:239`
- Modify: `mrcm/loader.go:76-120`
- Modify: `mrcm/validator_test.go`

**Step 1: Dominios en disyunción, no en conjunción**

Cuando un atributo tiene más de una fila de dominio — la forma normal del refset MRCM Attribute Domain, una fila por dominio y contentTypeId — el validador exige hoy que el foco esté en **todas** y emite un `domain_violation` por cada fila que no lo contenga. Separar aplicabilidad de conformidad: recolectar las filas del atributo, marcar violación solo si **ninguna** fila aplicable contiene al foco, y evaluar `grouped`/cardinalidad solo sobre las filas aplicables.

**Step 2: Cardinalidad mínima sobre las reglas del modelo**

`counts` solo contiene atributos **presentes**, así que `count < r.Cardinality.Min` es inalcanzable para un atributo obligatorio ausente: con `Min:1` y el atributo ausente el resultado es `Valid=true`. Recorrer las reglas del modelo, no el mapa de conteos. Implementar `InGroupCardinality` por índice de grupo o eliminar el campo, que hoy se carga y almacena con cero referencias.

**Step 3: Una regla inválida es un `Issue`, no un abort**

`validator.go:199-202` propaga el error hasta `:96-98`, que hace `return nil, err` y **descarta los issues ya detectados**. Convertirlo en un `Issue{Kind: "invalid_rule"}` y seguir. En el loader, validar `domainEcl` y `rangeEcl` con `ecl.Parse` en carga: hoy `domainEcl: ""` carga sin error y falla en cada validación posterior.

**Step 4: Expresiones anidadas y ruido**

Los valores de atributo que son expresiones anidadas escapan hoy al control de rango. Y cada violación se emite una vez por focus concept porque la recursión en `Value.Nested` está dentro del bucle de `expr.FocusConcepts`: sacarla del bucle.

```bash
go test ./mrcm/ -v && go test ./...
```

---

### Task 13: SCG y SCTID

**Files:**
- Modify: `scg/parser.go:239-259` (grupos), `:93` (definitionStatus), `:404-422` (literales)
- Modify: `sctid/sctid.go:44-55`
- Modify: `scg/parser_test.go:178-182`, `sctid/sctid_test.go`

**Step 1: Grupos de relación yuxtapuestos**

SCG separa los `attributeGroup` solo con whitespace, pero `parseRefinement` exige coma: el ejemplo publicado en la especificación se rechaza con `unexpected trailing input` en la posición 153 (verificado), y con coma parsea bien. Tras el `attributeSet` o el primer grupo, hacer `skipWS` y continuar el bucle mientras `p.peek() == '{'`, aceptando la coma como separador opcional por tolerancia.

**Step 2: El default de definitionStatus está invertido**

`Parse("22298006")` devuelve `DefStatusSubtype` (`<<<`) cuando SCG define `===` (equivalentTo) como default, lo que convierte todo código precoordinado suelto en «algún subtipo de». Inicializar `defStatus = DefStatusEquivalent` en `parseExpression` y en el caso anidado de `:350`, corregir el comentario de `scg/ast.go:16-21`, y actualizar `scg/parser_test.go:178-182`, que codifica la expectativa errónea.

> **Nota de compatibilidad:** cambia el AST observable de una expresión sin prefijo. Es un arreglo de corrección, no una feature: documentarlo en el CHANGELOG bajo un encabezado explícito de cambio de comportamiento. Hay **dos** tests que codifican la expectativa errónea, no uno: `scg/parser_test.go:178-182` y `scg/parser_test.go:14` (`TestParse_SingleConcept`).

**Step 3: Literales concretos según el ABNF**

`scg/parser.go:405` no contempla `+`, así que `#+20` se rechaza siendo válido; y acepta `#.5`, `#5.` y `#007` normalizándolos en silencio. Alinear con el ABNF que el propio repo incluye en `ecl/grammar/ECL.g4:146-149`, y conservar el texto original en `ConcreteValue` para permitir round-trip.

**Step 4: SCTID — cumplir el docstring o corregirlo**

`IsValid` promete «partition rules» y solo valida longitud, dígitos y Verhoeff: `IsValid("000000001")` es `true` (primer dígito cero, prohibido por el ABNF) y 94 de 1000 IDs de 8 dígitos pasan con partición ilegal. Rechazar `id[0] == '0'`, validar la partición `{0,1}×{0,1,2}`, y exigir `len >= 11` para las particiones `1x` (que requieren namespace de 7 dígitos). Añadir tests con vectores conocidos, incluidas las mutaciones de un dígito.

```bash
go test ./scg/ ./sctid/ -v && go test ./...
```

---

### Task 14: CLI — diagnósticos a stderr, `-h` con código 0, tests

`cmd/gofhir-ecl` está a 0.0 % de cobertura pese al comentario «runValidateWithOutput is the testable seam».

**Files:**
- Modify: `cmd/gofhir-ecl/{main,validate,eval,conformance}.go`
- Add: `cmd/gofhir-ecl/main_test.go`

**Step 1: Separar los flujos**

`fs.SetOutput(out)` con `out = os.Stdout` manda usage y errores de flags a stdout, contaminando pipes. Pasar `os.Stderr` a `SetOutput` y dejar stdout solo para resultados. Mismo patrón en `eval.go:20-21` y `conformance.go:21-22`.

**Step 2: `-h` no es un error**

`flag.ContinueOnError` devuelve `flag.ErrHelp` y ningún `errors.Is` lo trata, así que `-h` sale con **código 1** y el prefijo `error: flag: help requested`:

```go
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
```

**Step 3: Tests usando los seams existentes**

`main_test.go` con: ayuda, argumento faltante, expresión inválida (código ≠ 0 y mensaje en stderr), `-suites` inexistente, y una evaluación correcta contra el fixture.

**Step 4: Errores clasificables en el CLI**

Con `ParseError` (Task 2) y `ErrUnsupportedFeature` (Task 7), distinguir el código de salida de un error de sintaxis del de una feature no soportada.

```bash
go test ./cmd/... -v && go run ./cmd/gofhir-ecl -h; echo "exit=$?"
```

---

### Task 15: README ejecutable

El snippet SCG+MRCM del README usa `mrcm.LoadModel` y `mrcm.NewValidator`, que **no existen**: la API real es `LoadFromJSON`/`LoadFromBytes` y la función libre `Validate(ctx, expr, model, provider) (*Result, error)`. Tampoco declara `provider` ni `ctx`. Y no hay ni una función `Example*` en todo el repo, así que nada detecta la deriva.

**Files:**
- Modify: `README.md:84-93`, `:24-32` (tabla de features), `:175`
- Add: `ecl/example_test.go`, `mrcm/example_test.go`

**Step 1: Convertir los snippets en `Example_*` compilables**

Cada bloque de código del README pasa a ser un `Example` en el paquete correspondiente, con un provider mínimo. CI los compila y ejecuta, así que un cambio de API rompe el build en vez de dejar el README mintiendo.

**Step 2: Reescribir el snippet de MRCM contra la API real**

**Step 3: Alinear las afirmaciones con la Task 11**

Retirar el ✅ de lo que no tenga caso de conformidad, y corregir la promesa de `README.md:175` sobre reutilizar `internal/conformance` — se cumple de verdad en la Task 17.

```bash
go test ./... -run Example -v
```

---

### Task 16: Proceso — matriz, tidy, regeneración del parser, umbrales

**Files:**
- Modify: `.github/workflows/ci.yml`, `.github/workflows/release.yml`
- Add: `.github/dependabot.yml`
- Modify: `Makefile`, `.golangci.yml:48-82`, `go.mod`

**Step 1: `go mod tidy -diff` falla hoy**

`gopkg.in/yaml.v3` está marcado `// indirect` siendo import directo en `internal/conformance/`. Ejecutar `go mod tidy`, y añadir el gate:

```yaml
      - name: go.mod está limpio
        run: go mod tidy -diff
```

**Step 2: Matriz**

`strategy: matrix` con `go-version: [1.24.x, 1.25.x]` y `os: [ubuntu-latest, macos-latest, windows-latest]` — la librería se ofrece en el README para *edge devices* y hoy solo se prueba en un Linux con un Go.

**Step 3: El parser generado no puede derivar**

Target `generate` en el Makefile con la versión de ANTLR fijada, un `//go:generate` equivalente, y un paso de CI que regenere y falle si `git diff --exit-code ecl/grammar/` no está limpio. Alinear el runtime a la versión del generador (hoy: generado con 4.13.2, runtime pinneado v4.13.1). Sin esto, la Task 2 no puede migrar a la regla `root : expressionconstraint EOF`.

**Step 4: Umbrales de complejidad que muerdan**

`cyclop: 60`, `gocognit: 80`, `gocyclo: 60` y `maintidx: under: 15` están por encima del peor caso del propio código (`Evaluate`: gocognit 77, gocyclo 51), así que el gate es inerte. Bajar a `gocognit` 30-40, `gocyclo`/`cyclop` 20-25, `maintidx` 20, y excluir por `path`+`text` las funciones que no se refactoricen ahora, como ya se hace en `.golangci.yml:129-139`. Una exclusión documentada es honesta; un umbral de 80 disfraza el problema.

**Step 5: Release y permisos**

`release.yml` ignora `release-please-config.json` — el salto a v1.0.0 fue accidental. Alinearlo, añadir `permissions: contents: read` a `ci.yml` y un `dependabot.yml`.

```bash
go mod tidy -diff && golangci-lint run && go test ./...
```

---

### Task 17a: `providertest`, `go:embed` y centinelas — TODAVÍA EN FASE A

> **Reubicada.** La v1 de este plan metía esto en el v2, dejando el `conformance` del binario instalado
> roto hasta un major sin fecha. Nada de esto rompe a nadie: mover `internal/conformance` es imposible de
> romper (`internal/` no es importable por construcción — el propio plan lo argumenta), y
> `providertest.Verify`, los centinelas y `UnimplementedDataProvider` son **declaraciones nuevas**.
> Además `UnimplementedDataProvider` solo cumple su propósito declarado — «que añadir métodos deje de ser
> una ruptura» — si existe **antes** del release que añade los métodos, o sea antes del v2.

`README.md:175` promete que `internal/conformance` es reutilizable como paquete Go; Go lo prohíbe por
construcción. Y el subcomando `conformance` que `README.md:150` vende para CI usa rutas relativas al cwd:
reproducido, ejecutar el binario desde otro directorio da
`open testdata/conformance/cases: no such file or directory` (causa en `cmd/gofhir-ecl/conformance.go:22-23`).

- Mover `runner` + `fixture` a `ecl/providertest/`, exponiendo `providertest.Verify(t *testing.T, newProvider func() ecl.DataProvider)`.
- Incluir en la batería los casos de contrato que la Fase A documentó: `nil` Set, `nil`-como-wildcard, `AllConcepts` con inactivos, dirección de `HistoricalAssociations`.
- `go:embed` sobre los YAML para que el binario funcione instalado. **Caveat mecánico:** `//go:embed ../../testdata/...` **no compila** (`invalid pattern syntax`): un patrón no puede salir del directorio del paquete. Hay que relocar `testdata/conformance/` dentro del paquete que embebe y actualizar las rutas del runner y del CLI.
- Declarar `ErrProvider` y `ErrInvalidExpression` junto al `ErrUnsupportedFeature` de la Task 7, y añadir `ecl.UnimplementedDataProvider` embebible (patrón gRPC).

---

## Fase B — v2.0.0, romper una sola vez

Todo lo que sigue cambia firmas exportadas. Va en **un único release** con ruta de módulo `github.com/gofhir/ecl/v2`, con una guía de migración en el CHANGELOG.

### Task 17.0: Corte de major — instrumentarlo, no solo enunciarlo

La v1 de este plan enunciaba el `/v2` sin los pasos que lo hacen funcionar. Como está el repo hoy,
`go get github.com/gofhir/ecl/v2` fallaría.

- Rama `release-1.x` protegida antes de tocar nada.
- Commit único: `module github.com/gofhir/ecl` → `/v2`, más la reescritura de los 21 imports internos.
- README: `:11`, `:34`, `:40`, `:50`, `:86-87`. La Task 15 ya los convirtió en `Example*` compilables, así que CI detecta lo que se olvide.
- Alinear `.release-please-manifest.json` (`{".": "1.1.0"}`) con `release.yml`, que hoy pasa `release-type: go` en línea **sin** `config-file` e ignora `release-please-config.json` — el mismo defecto que arregla el Step 5 de la Task 16.
- Aceptación: `go get github.com/gofhir/ecl/v2@v2.0.0` desde un directorio limpio, fuera del repo.
- **Decidir ahora si se elimina el doble `ecl/ecl` del import path.** Es el único momento en que sale gratis: hoy un consumidor escribe `import "github.com/gofhir/ecl/ecl"`.

### Task 18: Retirar los campos deprecados

- `ast.Refinement`: eliminar `Ungrouped`, `Groups`, `Conjunction`, `Disjunction`; queda solo `AttrSet`.
- `ast.TermFilter.Term`, `EffectiveTimeFilter.Value`, `ModuleFilter.Module`, `DefinitionStatusFilter.Value`: eliminar los escalares, quedan las slices.
- Borrar `flattenAttrs` y el doble poblado del parser.

### Task 19: Firmas batch y la cardinalidad reverse

La cardinalidad en la ruta reverse es hoy **estructuralmente imposible**: `RelationshipTargets` devuelve un `Set` y pierde la multiplicidad, así que `[3..*] R attr = *` no puede distinguirse de `[1..*]`. Requiere un método nuevo:

```go
	// InboundRelationships returns, for each concept in targetIDs, the
	// relationships that point AT it with a type in typeIDs. Unlike
	// RelationshipTargets it preserves multiplicity, which cardinality needs.
	InboundRelationships(ctx context.Context, targetIDs Set, typeIDs Set) (map[string][]Relationship, error)
```

Y las dos firmas por-concepto que hacen falso el «batch-shaped to avoid N+1» del `provider.go:9` (20 000 round-trips medidos para una refinación; ~110 000 SELECT contra SNOMED International real):

```go
	PropertiesByGroup(ctx context.Context, conceptIDs []string) (map[string]map[int][]Relationship, error)
	ConcreteValues(ctx context.Context, sourceIDs []string, typeIDs []string) (map[string][]ConcreteValue, error)
```

Más `Ancestors`/`Descendants` con resultado por concepto (`map[string]Set`) para que `topOfSet`/`bottomOfSet` de la Task 6 sean una sola llamada. Y la negación a nivel de fila que la Task 7 dejó ruidosa:

```go
	// TypeIDsNegated inverts the TypeIDs match at the description row level:
	// a concept matches if it has a description whose type is NOT in TypeIDs.
	TypeIDsNegated bool
```

Con el evaluador ajustado, retirar el `ErrUnsupportedFeature` de los filtros de descripción negados.

### Task 20: Errores tipados y contrato extensible

104 `fmt.Errorf` y cero tipos de error: `errors.Is`/`errors.As` es imposible hoy (el llamador recibe un `*errors.errorString` y `Unwrap` devuelve `nil`), así que un servidor que necesite distinguir 400 de 503 solo puede hacer `strings.Contains`.

- Centinelas `ErrProvider`, `ErrUnsupportedFeature` (ya en Fase A), `ErrInvalidExpression`, envueltos con `%w` en los 104 sitios.
- `ecl.UnimplementedDataProvider` embebible, patrón gRPC: cada método devuelve `ErrUnsupportedFeature`, de forma que añadir el método 19 deje de romper a los providers de terceros.
- Opciones funcionales en `Evaluate` (`ecl.WithMaxDepth`, `ecl.WithCache`) para que la superficie pueda crecer sin nuevas firmas.

---

## Orden de ejecución y verificación

Orden de la Fase A tras la revisión adversaria:

```
0 → 1 → 1b → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12 → 13 → 14 → 15 → 16 → 17a
```

- **Task 0 bloquea todo**: sin el asiento de pruebas, cada tarea reinventa helpers incompatibles.
- **Task 1 antes que cualquier arreglo**: es lo que hace que un caso pueda fallar en CI.
- **Task 1b antes de las Tasks 5, 8, 9 y 11**: sin las secciones `dialects:`/`memberFields:`/inactivos, esas tareas no pueden escribir sus casos. Ojo: la cardinalidad de grupo de la Task 5 **sí** es verificable con el fixture actual (hoy `* : [2..*] { … }` devuelve `22298006`, no `∅`), así que la dependencia es de cobertura, no de bloqueo.
- **Tasks 3-5 consecutivas**: comparten el código de cardinalidad y `conceptMatchesAttribute`; intercalarlas produce conflictos.
- **Si se elige ampliar `UTF8_LETTER`** en vez de normalizar espacios (C4), el Step 3 de la Task 16 se adelanta **antes** de la Task 2.

Cada tarea que cambia comportamiento observable (2, 7, 10, 13) exige su entrada de CHANGELOG bajo
«Behaviour change» **en el mismo commit**. Son cuatro; agruparlas todas en el release notes del v1.2.0.

Al cerrar cada tarea:

```bash
go build ./... && go vet ./... && golangci-lint run
go test ./... && go test -race ./...
go run ./cmd/gofhir-ecl conformance
```

Al cerrar la Fase A, la evidencia de que funcionó son estas expresiones, todas verificadas como incorrectas hoy contra `testdata/conformance/fixtures/standard.yaml`:

| Expresión | Hoy | Esperado tras la Fase A | Task |
|---|---|---|---|
| `* : 363698007 = 74281007 OR 363698007 = 113331007` | `∅` | `22298006 73211009` | 3 |
| `* : { 363698007 = 113331007 OR 116676008 = 55641003 }` | `∅` | los que cumplen alguna en un grupo | 3 (Step 4c) |
| `* : 363698007 = 74281007 , { 116676008 = 55641003 }` | `22298006` ✅ | **igual** — no debe regresionar | 3 (C1) |
| `* : 116676008 != 55641003` | 17 conceptos | solo los que tienen el atributo con otro valor | 4 |
| `* : 1142139005 != #2` | `73211009` ✅ | **igual** — no debe regresionar | 5 (C2) |
| `* : [0..0] { 363698007 = 74281007 }` | `22298006` | el complemento | 5 |
| `* : [2..*] { 363698007 = 74281007 }` | `22298006` | `∅` | 5 |
| `!!> (138875005 OR 22298006)` | ambos | `138875005` | 6 |
| `!!< (138875005 OR 22298006)` | ambos | `22298006` | 6 |
| `* {{ C active = false }}` | `∅` | los conceptos inactivos | 9 + 1b |
| `<< 404684003 {{ D dialect = en-gb }}` | `∅` | los conceptos con descripción aceptable en `en-gb` | 8 + 1b |
| `<< 404684003 {{ D type != fsn }}` | `∅` | `ErrUnsupportedFeature` (resultado correcto en v2) | 7 |
| `< 138875005 {{ C active = true, moduleId != … }}` | 4 conceptos, algunos inactivos | solo los activos fuera de ese módulo | 7 |
| `validate "11687002 TOTALGARBAGE"` | `OK` | error de sintaxis | 2 |
| `explain "A MINUS B MINUS C"` | trunca a `A MINUS B` | error de sintaxis | 2 |
| `Parse("404684003 \|Crohn’s disease\|")` | `err=nil`, término corrupto | error reportado, no silenciado | 2 (C4) |
| `22298006 {{ +HISTORY }}` | `22298006` | `22298006 11111111` | 10 |
| `^ … {{ M mapTarget = "X" }}` | `unsupported AST node type` | los miembros con ese mapTarget | 8 (Step 6) |
| `gofhir-ecl conformance` desde otro directorio | `no such file or directory` | corre las 44+ | 17a |

Las tres filas marcadas ✅ son **anti-regresión**: hoy son correctas y la v1 de este plan las habría
roto. Van en la tabla precisamente por eso.
