# go-ecl: análisis técnico, dos revisiones adversariales y veredicto

**Fecha:** 12 de octubre de 2026.  
**Código revisado:** `fa3f3ef392754470baadf921468cc4df4ea1282d` (HEAD local; documentación de release v1.6.0).  
**Entorno de ejecución:** Go 1.25.9, Darwin/arm64, Apple M4 Pro.  
**Alcance:** biblioteca ECL, AST, providers, SCG, MRCM, SCTID, CLI, pruebas, oráculo, CI, documentación y estrategia comparativa.

## 1. Veredicto ejecutivo

**El proyecto tiene una arquitectura adecuada y una base de ingeniería sólida para un motor ECL embebible en Go. La evidencia actual no permite presentarlo como una implementación semánticamente completa de ECL v2.2 ni como un validador integral de MRCM.**

La principal fortaleza es separar el lenguaje de la terminología mediante `DataProvider`, acompañarlo de capacidades opcionales y proporcionar herramientas para verificar implementaciones. La principal debilidad es que algunas estructuras internas pierden información semántica que después resulta imposible recuperar: el ámbito y los límites de los bloques de filtros, la identidad de la descripción y la representación exacta de números.

La suite existente pasa con detector de carreras y la CLI confirma **153/153 casos**. Aun así, esta revisión reproduce resultados incorrectos con expresiones válidas y providers pequeños. Por tanto, el siguiente avance de mayor valor es **corregir semántica y fortalecer evidencia externa**, antes de ampliar la lista de funcionalidades.

| Uso | Veredicto | Condición |
|---|---|---|
| Parser ECL y herramientas de inspección | Favorable con reservas | Corregir conversiones numéricas silenciosas; no confundir aceptar sintaxis con preservar significado |
| Evaluación embebida de un subconjunto controlado | Viable | Definir el subconjunto, verificar el provider y excluir expresiones afectadas |
| Consultas ECL arbitrarias recibidas de terceros | No recomendar sin remediación | Hay falsos positivos y negativos confirmados en filtros y números |
| Validación SCG de referencias | Adecuada para ese alcance | No equivale a validación completa del modelo conceptual |
| Validación MRCM integral | Evidencia y alcance insuficientes | Faltan rangos concretos, selección por contexto y otros límites descritos abajo |
| Sustituir directamente un servidor terminológico | Fuera del alcance actual | Faltan almacenamiento productivo, carga de contenido y API operacional |

Estas conclusiones se refieren al código revisado. No se ejecutaron workloads de producción ni una evaluación clínica.

## 2. Método y fuerza de la evidencia

Se realizaron tres pasadas: análisis inicial de arquitectura y pruebas; revisión adversarial centrada en refutar la corrección; revisión adversarial centrada en refutar el veredicto, las mediciones y la comparación competitiva. **Son dos perspectivas críticas realizadas en esta misma revisión, no dictámenes independientes de dos personas o modelos.**

Clasificación utilizada:

- **Confirmado:** lectura de código más ejecución de un contraejemplo, o resultado directo de una herramienta.
- **Limitación explícita:** comportamiento admitido en código/documentación, cuyo alcance debe quedar claro.
- **Riesgo / hipótesis:** inferencia razonable pendiente de una prueba específica. No se contabiliza como defecto confirmado.
- **Propuesta:** cambio o experimento recomendado; no se presenta como implementado.

Se leyó el código escrito a mano y la integración del parser generado. No se auditó línea por línea toda la salida ANTLR. Tampoco se regeneró la gramática, ejecutó una nueva campaña prolongada de fuzzing, consultó el historial remoto de CI ni ejecutó `make oracle` contra un servidor público. Los resultados históricos del README no se presentan como mediciones nuevas.

### Comprobaciones ejecutadas

| Comprobación | Resultado | Qué demuestra |
|---|---|---|
| `go test -race -coverprofile=… ./...` | Pasa | Casos existentes y ausencia de carreras detectadas en esos recorridos |
| `go run ./cmd/gofhir-ecl conformance` | 153 pasan, 0 fallan, 0 omitidos | Acuerdo con las expectativas locales y fixture embebida |
| Ejemplos oficiales | Pasan dentro de `go test` | Aceptación de los 121 archivos oficiales vendorizados |
| `go vet` excluyendo `ecl/grammar` | Pasa | Comprobaciones estáticas sobre paquetes mantenidos manualmente |
| `go mod verify` | Pasa | Integridad de módulos descargados |
| `go mod tidy -diff` | Pasa sin diferencias | Consistencia del módulo; se necesitó acceso adicional a la caché |
| Benchmarks de parser | Ejecutados | Coste local de formas concretas de entrada |
| Programa adversarial del apéndice | Ejecutado | Contraejemplos descritos en los hallazgos |

El lint local usa **golangci-lint 2.13.2**; CI fija **2.11.3**. La ejecución local reportó cinco avisos `goconst` en `providertest/contract.go`, `providertest/fixture.go`, `internal/oracle/corpus.go`, `scg/parser.go` y `scg/render.go`. Esto no prueba que CI falle con su versión fijada. No se declara `make check` completamente verde.

### Cobertura medida

| Paquete | Statements cubiertos |
|---|---:|
| `ecl` | 77,0 % |
| `ecl/providertest` | 82,3 % |
| `mrcm` | 94,6 % |
| `scg` | 91,6 % |
| `sctid` | 96,9 % |
| CLI | 57,3 % |
| `internal/oracle` | 0,0 % en esta ejecución; test diferencial omitido por configuración |

`ast` y `grammar` reportan 0 % con la instrumentación utilizada. No significa que no se invoquen: la cobertura entre paquetes necesita una configuración como `-coverpkg`. Un porcentaje global dominado por archivos generados sería poco útil. Tampoco la cobertura alta de MRCM demuestra que cubra toda la especificación.

## 3. Arquitectura y decisiones correctas

### 3.1 Separación del lenguaje y los datos

[`ecl/provider.go`](../ecl/provider.go) define 18 métodos obligatorios. El evaluador usa un AST propio y recibe los hechos del provider. Es una buena frontera para integrar bases SQL, índices o terminología en memoria, y permite probar el motor sin redistribuir una edición completa de SNOMED CT.

El coste de esa flexibilidad es significativo: la corrección del producto resulta de **parser + evaluador + provider + contenido + configuración de edición**, no del evaluador por sí solo. El contrato debería declarar también el sustrato: versión/edición, vista stated/inferred, estado de relaciones, descripciones y miembros. La [referencia oficial de ECL](https://docs.snomed.org/snomed-ct-specifications/snomed-ct-expression-constraint-language/appendices/appendix-d-ecl-quick-reference) distingue los conceptos incluidos del estado activo de los otros componentes.

### 3.2 Capacidades opcionales y compatibilidad

[`ecl/capabilities.go`](../ecl/capabilities.go) permite mejorar rendimiento o habilitar semántica sin ampliar la interfaz obligatoria. `UnimplementedDataProvider` ofrece un punto de entrada razonable y evita devolver conjuntos vacíos cuando falta implementación.

El diseño favorece evolución compatible. Conviene conservarlo, pero evitar que cada nuevo caso introduzca otra interfaz incapaz de componer con las anteriores: los filtros de descripción ya muestran ese límite.

### 3.3 Contratos y fixtures reutilizables

`providertest.VerifyContract` verifica invariantes contra los datos del consumidor; `VerifyFixture` verifica resultados del motor con datos propios. La separación es acertada. Los checks omitidos con motivo también son una buena práctica: no todos los datasets permiten probar todas las propiedades.

Mejora necesaria: publicar cuántos checks fueron efectivamente ejercitados por capacidad. «El contrato pasa» es una afirmación débil si casi todo se omitió.

### 3.4 Parser y manejo de entrada

El parser emplea SLL primero y LL como fallback, comprueba EOF, acumula errores de lexer/parser y aplica límites de bytes y profundidad. Son medidas concretas contra truncamiento y entradas costosas. Las pruebas de regresión reflejan aprendizaje de defectos anteriores.

No hay un presupuesto temporal duro del parser. Los límites de tamaño y profundidad acotan dimensiones de entrada, pero no prueban que toda forma admitida tenga un coste aceptable. El fallback LL y las entradas inválidas necesitan mediciones propias.

### 3.5 Pruebas y CI

Hay ejemplos externos, casos de evaluación locales, fuzzing, race detector, matriz de sistemas operativos/Go, regeneración de gramática y detección programada de cambios upstream. La combinación es mucho mejor que depender únicamente de unit tests.

El oráculo diferencial es especialmente valioso porque separa los hechos obtenidos del servidor de la composición ejecutada localmente. Sin embargo, varias primitivas también se resuelven mediante ECL remoto: puede haber errores correlacionados. Hace falta complementar ese enfoque con datos pequeños independientes y RF2 versionado.

### 3.6 SCG, SCTID y MRCM

SCG tiene parser, representación y renderer con round trip; SCTID valida estructura y checksum. Son piezas útiles y delimitables. El renderer conserva el orden y no promete normalización semántica, decisión correctamente explicada en las notas de v1.6.0.

MRCM memoiza evaluaciones durante una validación, trata reglas de dominio como alternativas y distingue fallos de infraestructura de reglas inválidas. Son decisiones razonables. Su modelo JSON simplifica el contenido RF2 y pierde dimensiones relevantes para una aplicación general; debe anunciarse como un subconjunto de validación.

## 4. Hallazgos priorizados

**P1:** puede alterar resultados o aceptar validación incompleta silenciosamente. **P2:** inconsistencia de API, carga o evidencia que requiere corrección, sin equivaler por sí sola a un fallo general del motor. No se confirmó un incidente P0.

### H1 — Se pierde el ámbito D/C de filtros de metadatos

**P1 · Confirmado.**

El parser convierte filtros de descripción y de concepto en los mismos tipos `ActiveFilter`, `ModuleFilter` y `EffectiveTimeFilter`. Después, `categorizeFilters` los asigna siempre a la familia de conceptos. El prefijo explícito `D` ya no puede recuperarse.

Reproducción: concepto activo `22298006` con una descripción inactiva y otra descripción activa cuyo módulo es `404684003`; el concepto no tiene ese módulo.

```ecl
* {{ D active = false }}
* {{ D moduleId = 404684003 }}
```

Ambas expresiones devuelven vacío, aunque la descripción declarada satisface el filtro correspondiente. El primer caso contrasta directamente el estado de concepto y descripción. El segundo confirma que se consulta la columna del componente equivocado.

**Causa:** [`parser.go`, construcción de filtros](../ecl/parser.go) y [`categorizeFilters`](../ecl/evaluator.go). La [sintaxis oficial](https://docs.snomed.org/snomed-ct-specifications/snomed-ct-expression-constraint-language/design/5-syntax-specification) incluye esos tres filtros dentro de los filtros de descripción.

**Mejora:** preservar el ámbito en el AST y transportar los metadatos al predicado de la misma descripción. Agregar pares de pruebas D/C donde los estados, módulos y fechas difieran.

### H2 — Las cláusulas repetidas dejan de ser una conjunción

**P1 · Confirmado.**

```ecl
* {{ D language = en, language = es }}
* {{ D term = "nonexistent", term = "heart" }}
```

Con dos descripciones, una inglesa `heart` y otra española `corazon`, ambas expresiones incluyen `22298006`. En el primer caso se concatenan los idiomas y el provider recibe un filtro any-of; en el segundo, el último término sobrescribe al anterior.

La primera expresión exige dos comparaciones sobre una misma columna y debería ser insatisfacible. La segunda exige ambos términos en una misma descripción; ninguna los tiene. Una lista de valores dentro de una cláusula tiene semántica diferente de varias cláusulas unidas por comas.

**Causa:** `buildDescriptionFilterOpts`, `termCalls` y la representación de filtros. **Mejora:** conservar cláusulas como predicados individuales. Mientras una forma no pueda representarse fielmente, devolver `ErrUnsupportedFeature` en lugar de ensanchar la consulta.

### H3 — Término y dialecto pueden satisfacerse en descripciones distintas

**P1 · Confirmado en el fixture y en la composición del evaluador.**

```ecl
* {{ D term = "heart", dialectId = 900000000000508004 }}
```

El programa adversarial declara el término inglés en una descripción y la membresía de dialecto en la descripción de otra lengua del mismo concepto. El resultado incluye el concepto.

`MatchDescription` devuelve conceptos y `MatchDialect` recibe esos conceptos, sin el término ni la identidad de las descripciones que lo cumplieron. La intersección posterior no puede garantizar una misma fila. El problema persiste con un provider real que implemente correctamente ambos métodos por separado: no recibe información suficiente para unir ambos predicados.

Además, la fixture identifica membresías de dialecto por concepto y lengua, no por description ID; dos descripciones de una misma lengua tampoco pueden distinguirse con exactitud. El dialecto escogido para la prueba es deliberadamente sintético: se prueba el mecanismo de correlación, no el contenido lingüístico de ese refset.

**Mejora:** una capacidad que reciba el bloque completo de predicados de descripción, incluida la membresía de dialecto, y lo resuelva sobre una descripción identificable. Mantener los métodos antiguos para las formas simples que puedan contestar correctamente.

### H4 — Números grandes cambian de significado

**P1 · Confirmado; dos mecanismos diferentes.**

1. Un valor almacenado `9007199254740992` coincide con una consulta por `#9007199254740993`. Ambos se convierten al mismo `float64` antes de compararse.
2. `#99999999999999999999999999999` se acepta sin error y queda representado en el AST como `9223372036854775807` en este entorno. `strconv.Atoi` devuelve un valor saturado y un error que se descarta.

`visitCardinality` también descarta errores de `Atoi`; el riesgo de cardinalidades alteradas está visible en código, aunque el apéndice reproduce específicamente el overflow de valores concretos.

**Mejora:** conservar el literal original y comprobar cada conversión. Para enteros, comparar sin conversión a float; para decimales, definir representación exacta y comparación coherente entre integer/decimal. Si se impone un rango, rechazar explícitamente lo que queda fuera. Usar `int` en el AST introduce además dependencia de arquitectura.

Cambiar campos públicos del AST requiere una decisión de compatibilidad; se puede comenzar por rechazar overflow y añadir una representación exacta compatible antes de una eventual v2.

### H5 — `ErrProvider` no clasifica todas las fallas de provider

**P2 · Confirmado.**

Un `ConceptExists` que retorna `errors.New("backend down")` produce ese error sin `ErrProvider`. Otros caminos sí agregan el sentinel. El consumidor no puede confiar uniformemente en `errors.Is(err, ecl.ErrProvider)` como promete su documentación.

**Mejora:** centralizar el tratamiento de errores de provider, preservando cancelación, causas originales y `ErrUnsupportedFeature`. Añadir una tabla que fuerce errores por cada método, no solo por uno representativo.

### H6 — Carga MRCM acepta documento incompleto o fuerza de regla desconocida

**P2; P1 cuando es la barrera de validación · Confirmado.**

```json
{} {"unexpected": true}
```

El loader acepta el primer objeto e ignora el segundo: `DisallowUnknownFields` no comprueba EOF después de `Decode`. También acepta `null` como modelo vacío.

Un `ruleStrengthId` con SCTID válido pero ajeno a las dos fuerzas admitidas, por ejemplo `404684003`, se carga. El validador omite toda regla cuya fuerza no sea vacía o mandatory: un identificador válido en estructura puede desactivar una regla por error de configuración.

**Mejora:** exigir fin de documento, decidir explícitamente si `null` es admisible y validar la enumeración de fuerzas. La política para modelos construidos directamente en Go debe ser igual de clara. Un modelo vacío puede ser válido por diseño; el defecto es aceptar por accidente uno que no expresa la configuración pretendida.

### H7 — `Valid: true` de MRCM no implica validación completa de rangos

**P1 de alcance de API/producto · Limitación explícita y reproducida.**

Un modelo con rango conceptual `74281007` para el atributo `363698007` acepta `22298006 : 363698007 = #12` y devuelve `Valid:true`. El validador solo comprueba rango si `attr.Value.Concept != nil`. Los valores concretos y las expresiones anidadas no reciben esa comprobación externa.

Esto está documentado en el godoc; no es un fallo oculto del algoritmo prometido allí. Sí impide interpretar el resultado como conformidad general con MRCM. La recursión sobre una expresión anidada tampoco demuestra por sí sola que esa expresión satisfaga el rango del atributo que la contiene.

El modelo tampoco representa `contentTypeId`, ni ofrece carga RF2 directa. Tratar todas las filas de diferentes contextos como alternativas puede resultar demasiado permisivo si el consumidor no filtra antes.

**Mejora:** exponer el alcance o completitud del resultado y rechazar formas no verificables cuando se solicite validación estricta. Incorporar contexto de contenido y rangos concretos antes de anunciar validación integral.

### H8 — El estado verde del oráculo puede aportar cero evidencia

**P2 · Confirmado por lectura.**

[`TestDifferential`](../internal/oracle/oracle_test.go) omite casos por servidor inaccesible, capacidades no implementadas y operaciones rechazadas. Distingue acuerdos vacíos, lo cual es bueno, pero no exige un mínimo de acuerdos no vacíos. Una ejecución puede acabar satisfactoriamente sin comparar nada útil.

El harness no implementa varias primitivas de filtros de descripción/concepto. Los defectos H1–H3 no quedan cubiertos simplemente por ejecutar el corpus diferencial actual.

**Mejora:** mantener el modo exploratorio tolerante y añadir un modo de cualificación con umbrales de cobertura y skips permitidos por motivo. Distinguir explícitamente «no hubo divergencias observadas» de «se cualificaron estas capacidades».

## 5. Rendimiento y límites operacionales

### 5.1 Mediciones del parser

Resultados de `-benchtime=200ms`, sin race detector:

| Forma | Tamaño | Tiempo | Bytes asignados/op | Allocs/op |
|---|---:|---:|---:|---:|
| Expresión típica | ~170 B | 100 µs | 122.274 | 1.626 |
| 10 cláusulas | 222 B | 232 µs | 266.296 | 3.324 |
| 50 cláusulas | 1.102 B | 646 µs | 696.582 | 9.797 |
| 100 cláusulas | 2.202 B | 1,27 ms | 1.241.497 | 18.251 |
| 200 cláusulas | 4.402 B | 2,50 ms | 2.385.117 | 35.156 |

La tendencia de las conjunciones planas es aproximadamente lineal en ese rango. Las asignaciones siguen siendo relevantes para un endpoint de alto tráfico; son bytes acumulados asignados por operación, **no RSS ni memoria retenida**.

Un primer benchmark corto a profundidad 100 dio 40,1 ms/op y ~60 MB/op. Una repetición aislada de 1 segundo, tres veces, dio **0,817 / 0,485 / 0,488 ms/op**, con asignaciones estabilizadas cercanas a 219 KB/op en las últimas pasadas. No es evidencia suficiente para declarar una regresión sostenida: la diferencia es compatible con costes de inicialización/caché amortizados de forma distinta, sin que aquí se haya perfilado la causa.

**Conclusión corregida:** publicar arranque frío y régimen estable por separado, con procesos nuevos y repeticiones. Una única cifra de profundidad 100 no prueba el coste máximo de todas las entradas admitidas. Tampoco se midió aquí el comportamiento cerca de 1 MiB.

### 5.2 El evaluador depende del plan de acceso a datos

Las capacidades batch reducen viajes al backend, pero no necesariamente el volumen de datos cargados. Descargar todas las relaciones de un foco grande puede consumir mucha memoria aunque se haga en una sola llamada. Hay que medir bytes, cantidad de filas y tamaño de lote.

`Top` y `Bottom` consultan jerarquía por concepto. Sin memoización o implementación especializada del provider, escalan en número de llamadas. Las operaciones AND/OR materializan conjuntos y evalúan ambos lados; no hay un planificador general que use selectividad o empuje toda la consulta al backend.

No recomiendo comenzar con un optimizador completo: primero instrumentar el evaluador y crear un provider productivo medible. Después decidir si convienen lotes limitados, intersecciones tempranas, caché por evaluación o una capacidad de ejecución delegada.

### 5.3 Riesgos que deben medirse, no afirmarse como incidentes

- SCG carece de límites explícitos equivalentes a los de ECL; su recursión profunda depende del stack y memoria disponibles.
- MRCM procesa JSON sin un límite interno de tamaño y puede validar muchas reglas ECL. El consumidor puede limitar el reader, pero falta una política explícita.
- La comprobación de tiempo en fuzzing ocurre después del retorno: detecta lentitud cuando termina, no interrumpe un bloqueo.
- La cancelación del evaluador está presente en pasos recursivos y varios loops, pero el provider debe cooperar. Las operaciones de conjuntos no reciben contexto.
- El AST público puede construirse manualmente sin pasar por límites del parser. No se confirmó aquí un problema de ciclos o de typed nil; conviene definir si el API exige AST válido.

## 6. Primera revisión adversarial: intentar refutar «el motor es correcto»

**Tesis inicial atacada:** la amplitud de funcionalidades y los tests verdes sugieren que el proyecto implementa correctamente su alcance ECL.

| Ataque | Experimento o evidencia | Resultado | Corrección de la conclusión |
|---|---|---|---|
| Los tests pueden compartir la misma interpretación equivocada | Contraejemplos H1–H4 sobre fixture mínima | Refutación lograda | 153/153 no equivale a conformidad semántica completa |
| El AST puede borrar diferencias significativas | Ámbito D/C y bloques convertidos a una lista plana | Información perdida confirmada | Corregir representación antes de añadir más condiciones |
| Dos métodos correctos pueden componer un resultado incorrecto | Término y dialecto resueltos sobre conjuntos de conceptos | Refutación lograda | Hace falta correlación por descripción, no más intersecciones |
| Un número aceptado puede dejar de ser el número escrito | Overflow del literal y colisión float64 | Refutación lograda | Validar conversiones y preservar exactitud |
| Cobertura alta puede esconder alcance incompleto | MRCM acepta valor concreto contra rango conceptual | Refutación lograda para el claim integral | Mantener el mérito del subconjunto y explicitar lo no comprobado |

**Observación adicional pendiente:** la lista plana también borra fronteras entre bloques `{{ D … }}`. Hay que probar separadamente condiciones que deben compartir una descripción dentro de un bloque y condiciones de bloques distintos que pueden satisfacer descripciones diferentes. No se presenta una reproducción adicional aquí.

**Resultado de esta revisión:** la base técnica sigue siendo valiosa, pero la tesis de completitud queda refutada. La prioridad pasa de ampliar cobertura de funcionalidades a preservar estructura semántica y generar casos discriminantes.

## 7. Segunda revisión adversarial: intentar refutar este análisis y la estrategia

Esta pasada critica tanto el optimismo como el exceso de severidad del primer análisis.

### Objeción A: «Los enteros mayores de 2^53 casi nunca aparecen en datos clínicos»

Es una objeción válida sobre frecuencia e impacto práctico. No elimina el defecto: se acepta una consulta y se cambia el resultado sin advertencia. La solución puede ser un rango explícito, no necesariamente aritmética arbitraria. La prioridad concreta depende de los datos del consumidor; H1–H3 afectan formas más habituales y deben ir primero.

### Objeción B: «Una fixture artificial no demuestra que un servidor real falle»

Correcto: el dialecto del ejemplo es sintético y no se declara divergencia contra Ontoserver/Snowstorm. El hallazgo demuestra pérdida de correlación en la interfaz y composición; la validación externa siguiente debe reproducirlo con dos descripciones reales y memberships identificadas por descripción. Para H1/H2, los contraejemplos no requieren una terminología grande.

### Objeción C: «MRCM nunca prometió validar rangos concretos»

El godoc delimita ese alcance. Por ello H7 se clasifica como limitación explícita de API/producto, no como incumplimiento oculto del contrato documentado. El veredicto debe permitir usar el subconjunto y exigir precisión en el significado de `Valid`.

### Objeción D: «El benchmark de 40 ms demuestra una regresión grave»

La repetición aislada no la sostiene. Se retiró esa conclusión: hay que separar inicialización y estado estable. Este ejemplo ilustra por qué el análisis adversarial debe revisar también sus propias mediciones.

### Objeción E: «El oráculo usa el mismo servidor como fuente de hechos y como juez»

La comparación sigue teniendo valor para descubrir errores de composición local. No es independiente para primitivas ya calculadas mediante ECL remoto. Tampoco detecta errores que ambos lados comparten. Se necesita una tercera base: fixtures con verdad explícita y datos RF2 importados sin usar ECL para producir las expectativas.

### Objeción F: «Comparar una biblioteca con un servidor es injusto»

Correcto si se mezclan latencia de parseo local y latencia HTTP total. La comparación debe separar parser, evaluador con datos y producto operacional. La biblioteca puede ganar en integración Go y a la vez perder en coste total si cada usuario debe implementar almacenamiento.

### Objeción G: «El proyecto tiene tests, así que producción es una decisión del consumidor»

El consumidor debe verificar su provider, pero no puede corregir desde él información que nunca recibe. H1–H3 son responsabilidades del motor/contrato, no una transferencia legítima al backend.

**Resultado de esta revisión:** mantener los defectos reproducidos, rebajar afirmaciones sobre rendimiento y alcance MRCM, y evitar un rechazo global del proyecto. El veredicto final es **adopción condicionada y posicionamiento estrecho**, con una ruta concreta hacia mayor confianza.

## 8. Estrategia de comparación con otras bibliotecas y motores

### 8.1 Qué comparar y con quién

Fuentes primarias consultadas para identificar alternativas; capacidades declaradas no equivalen a capacidades verificadas en esta revisión.

| Alternativa | Naturaleza | Comparación válida | Precaución |
|---|---|---|---|
| [SNOMED International `snomed-ecl-parser`](https://github.com/IHTSDO/snomed-ecl-parser) | Biblioteca Java de parsing y objetos | Aceptación/rechazo, preservación del árbol y diagnósticos | No usarla como evaluador completo |
| [IQVIA `snomed-ecl`](https://github.com/iqvia-hds-oss/snomed-ecl) | Implementación basada en Xtext | Parsing y tooling del lenguaje; verificar API concreta de la versión elegida | No inferir capacidades de ejecución solo por el nombre |
| [Snowstorm](https://github.com/IHTSDO/snowstorm) | Servidor SNOMED CT sobre Elasticsearch | Resultados ECL, comportamiento semántico y coste de despliegue | Comparar el motor y el servicio en pistas distintas |
| [Ontoserver](https://ontoserver.csiro.au/site/technical-documentation/ontoserver-technical-documentation/) | Servidor terminológico con API FHIR | Oráculo diferencial y workload FHIR | Un servidor público variable no es una plataforma de benchmark controlado |
| [Snow Owl](https://github.com/iqvia-hds-oss/snow-owl) | Servidor terminológico y authoring | ECL y plataforma operacional | Sus capacidades exceden ampliamente las de una biblioteca |
| [Rust `snomed-ecl`](https://docs.rs/crate/snomed-ecl/latest) | Crate con parser/evaluación y capacidades documentadas | Candidato para evaluación embebida entre lenguajes | Fijar versión y comprobar cada capacidad declarada |

Los antiguos repositorios de B2i anuncian migración a IQVIA: usar las ubicaciones actuales y fijar commits. No se asignan rankings ni se afirma superioridad de rendimiento sin ejecutar las comparaciones.

La afirmación del README «hasta ahora Go requería Snowstorm» es demasiado absoluta sin un estudio exhaustivo del ecosistema. Una formulación defendible sería: **«Motor ECL embebible en Go con provider intercambiable y herramientas para verificar su contrato»**.

### 8.2 Tres pistas separadas

**Pista A — Lenguaje.** Mismo corpus, parseo sin I/O y comparación de aceptación/rechazo. Canonicalizar a un formato neutral los nodos significativos: ámbito, bloques, operadores, cardinalidad y literales originales. Medir diagnóstico, tiempo de arranque, tiempo estable y asignaciones. Los ejemplos positivos oficiales deben acompañarse de negativos y mutaciones que alteren significado.

**Pista B — Semántica.** Misma edición/versionado y vista de relaciones. Comparar conjuntos completos de IDs mediante diferencia simétrica. Incluir consultas por familia y combinaciones entre familias; documentar los requisitos de cada provider. Mantener casos con resultados positivos y negativos conocidos, incluidos vacíos esperados: un vacío es evidencia útil cuando la verdad esperada es vacío, aunque un acuerdo vacío sin explicación no lo sea.

**Pista C — Producto y operación.** Workload equivalente de FHIR o servicio de consulta, con costes de importación, almacenamiento, memoria, arranque, actualización e integración. Separar la latencia HTTP del coste interno del motor y publicar ambos. Medir también el trabajo necesario para construir y mantener el provider.

### 8.3 Condiciones para una comparación reproducible

Guardar junto con cada ejecución un manifiesto:

```yaml
run:
  timestamp: "<UTC>"
  corpus_commit: "<sha>"
  seed: 42
terminology:
  edition_uri: "<edition>"
  version_uri: "<version>"
  rf2_sha256: "<hash>"
  view: inferred
  substrate_policy: "<estado de conceptos, relaciones, descripciones y miembros>"
engines:
  go_ecl: "<commit>"
  provider: "<commit + capacidades>"
  reference: "<producto + versión + configuración>"
environment:
  cpu: "<modelo>"
  memory_limit: "<bytes>"
  concurrency: 1
  cache_mode: "cold|warm"
```

Fijar los estados activos por tipo de componente; no basta con escribir `active=true` globalmente. Incluir edición, extensiones, módulos, lenguaje, dialecto, history profiles y política stated/inferred. Completar paginación antes de comparar; detectar truncamiento, duplicados y respuestas parciales.

### 8.4 Corpus propuesto

Primero conservar los 121 ejemplos y 153 casos existentes. Después añadir un corpus semántico independiente de aproximadamente **300–500 casos como objetivo de trabajo**, no como número mágico de conformidad:

- Jerarquía, álgebra y Top/Bottom sobre conjuntos abiertos y no cerrados.
- Agrupación frente a atributos distribuidos en grupos distintos; grupo 0.
- Cardinalidades 0, exactas y no acotadas, duplicados y valores redundantes.
- Reverse con y sin capacidad inbound, incluidas cardinalidad y negación.
- Descripciones: D/C, varias cláusulas de la misma dimensión, términos por conjunto, bloques separados, dialecto y description ID sobre la misma fila.
- Números alrededor de 2^53, límites del tipo entero, overflow y decimales cercanos.
- Membresías de refset con varias filas para el mismo componente; proyecciones y campos concretos.
- Historial MIN/MOD/MAX y contenido nacional.
- MRCM: tipo de contenido, reglas opcionales, cardinalidades, rangos concretos y valores anidados.
- Errores tipificados: sintaxis, no soportado, backend y cancelación.

La meta es que cada caso discrimine entre una implementación correcta y una plausible incorrecta. Añadir propiedades metamórficas como `A AND A = A`, `A OR B = B OR A`, `A MINUS A = vacío` y equivalencia batch/no batch, sin tratarlas como sustituto de la especificación.

### 8.5 Métricas y política de divergencias

Publicar por capacidad: casos compatibles, acuerdos exactos, diferencias, errores de infraestructura, no soportados y omisiones. No convertir unsupported o timeout en vacío. Para desempeño: p50/p95/p99, throughput, CPU, asignaciones, RSS, llamadas al provider, filas/bytes retornados y tiempos de importación. Ejecutar procesos nuevos para frío y repeticiones suficientes para estable.

Cada divergencia debe producir una expresión mínima, dataset mínimo, resultados de ambos motores, configuración y referencia normativa. Dos servidores que coinciden no reemplazan la especificación. Si dos interpretaciones son posibles, documentar la decisión y mantener un caso de regresión.

**Criterio propuesto de publicación:** cero diferencias semánticas sin explicación en el subconjunto soportado, un mínimo pactado de casos útiles por familia y un máximo explícito de omisiones. El umbral debe aprobarse según el producto; no inventar un porcentaje universal de conformidad.

## 9. Mejoras y secuencia recomendada

| Orden | Entregable | Criterio de aceptación |
|---|---|---|
| 1 | Casos de regresión para H1–H6 | Fallan contra el commit revisado y cada uno prueba significado/error concreto |
| 2 | AST conserva ámbito y bloques | D/C se distinguen; dentro/fuera de bloque tiene pruebas independientes |
| 3 | Predicados de descripción correlacionados | Término, idioma, tipo, estado, ID y dialecto se aplican a una descripción |
| 4 | Números y cardinalidades comprobados | Sin saturación silenciosa; igualdad exacta o rechazo explícito |
| 5 | Errores y loader consistentes | Matriz de métodos del provider; EOF JSON y fuerzas desconocidas cubiertos |
| 6 | Alcance MRCM visible | Resultado distingue lo validado de lo no verificable; contexto definido |
| 7 | Oráculo de cualificación | No queda verde con cero evidencia; edición y skips registrados |
| 8 | Provider productivo de referencia | Importación/versionado repetible y contrato realmente ejercitado |
| 9 | Comparativa pública reproducible | Corpus, configuración y resultados descargables, con límites explícitos |

No empezaría migrando el parser, cambiando mapas por bitmaps ni construyendo un optimizador general. Primero corregir la información que el AST pierde; después medir el provider y las asignaciones para decidir dónde invertir.

En documentación: reemplazar «full coverage» por matriz parser/evaluador/capacidad/limitación; corregir la referencia inexistente a `internal/conformance`; aclarar que la CLI de conformance usa fixtures embebidas y por sí sola no certifica el backend del consumidor; separar estadísticas históricas de resultados reproducidos y conservar enlaces versionados.

## 10. Conclusión final

**Continuar el proyecto y estrechar sus afirmaciones públicas.** Hay una propuesta útil y diferenciada para Go, con decisiones correctas y pruebas que merecen conservarse. Los contraejemplos encontrados justifican una fase de corrección semántica antes de promover consultas arbitrarias o validación MRCM integral.

La ventaja competitiva defendible es la integración embebida con un contrato verificable. Para convertirla en una ventaja práctica, el próximo hito debería combinar **filtros correctos, números exactos o acotados, un provider productivo y una comparación publicada sobre datos/versiones fijos**. No hay evidencia todavía para afirmar que sea más rápido, más completo o más barato que las alternativas.

## 11. Addenda del 13 de octubre de 2026: pruebas de regresión y hallazgos nuevos

Esta addenda registra lo que cambió al convertir los hallazgos en pruebas. Las secciones anteriores quedan como estaban el 12 de octubre; donde esta addenda las corrige, prevalece la addenda.

### 11.1 Cómo se implementó el paso 1 de la sección 9

La sección 9 pide casos de regresión que «fallan contra el commit revisado». Se implementaron como **fallos esperados estrictos**, para no dejar CI en rojo:

- Cada caso declara el resultado que exige la especificación (`want`) y, mientras el defecto exista, el resultado incorrecto exacto que produce hoy (`now`).
- El caso pasa mientras el evaluador devuelva `now`. Falla si devuelve `want` («el defecto parece corregido: quitar la marca») y también si devuelve cualquier otra cosa («algo más cambió»).
- Quitar la marca convierte el caso en una prueba de regresión ordinaria.

| Archivo | Cubre |
|---|---|
| `ecl/semantics_test.go` | H1, H2, H2b, H3 y H4, con base normativa citada (§6.8 y ejemplos oficiales 8.1.8 y 8.2.2) |
| `ecl/metamorphic_test.go` | Relaciones que valen sin resultado esperado: orden de bloques y de cláusulas, bloques frente a `AND`, álgebra de conjuntos y orden numérico |
| `ecl/provider_errors_test.go` | H5, método por método |
| `mrcm/loader_strict_test.go` | H6 |
| `ecl/providertest/description_filter_test.go` | El defecto del provider descrito en 11.3 |

Las relaciones metamórficas detectan H2, H2b y H4 sin ningún resultado escrito a mano. H1 y H3 no: un ámbito equivocado o una correlación perdida pueden ser perfectamente coherentes consigo mismos, así que sólo los detectan los casos escritos a mano.

H7 y H8 no tienen pruebas.

### 11.2 H2b: los bloques separados tampoco son independientes

**P1 · Confirmado. No figuraba en la sección 4;** es la observación que la sección 6 dejó pendiente.

El evaluador junta todos los bloques `{{ D … }}` en un solo predicado:

- **El último término gana.** `{{ D term = "nonexistent" }} {{ D term = "corazon" }}` devuelve el concepto que tiene «corazon», cuando debería devolver vacío. El resultado depende del orden de los bloques.
- **Los idiomas se suman como «cualquiera de».** `{{ D language = es }} {{ D language = en }}` incluye conceptos sin ninguna descripción en español.
- **Término e idioma de bloques distintos se exigen a una misma descripción.** `{{ D term = "heart" }} {{ D language = es }}` devuelve vacío para un concepto con «heart» en inglés y otra descripción en español.

La forma del ejemplo oficial 8.2.2 (`{{ term = …, language = sv }} {{ term = …, language = en }}`) da un falso positivo. Hoy ese ejemplo sólo se prueba como parseo.

Según los ejemplos 8.1.8 y 8.2.2, cada bloque puede satisfacerse con una descripción distinta. Al corregir H3 hay que conservar esa independencia entre bloques.

### 11.3 Ampliaciones de hallazgos existentes

- **H1 alcanza también `effectiveTime`.** `{{ D effectiveTime = X }}` compara con la fecha del concepto. Además, el provider en memoria de `providertest` ignoraba `DescriptionFilterOpts.EffectiveTime`, aunque el contrato lo define; se corrigió en este cambio. Ese código queda inalcanzable desde ECL hasta que el arreglo de H1 en `categorizeFilters` envíe esos campos. `DescriptionFilterOpts` tampoco puede representar todavía varios valores de `effectiveTime` ni `moduleId !=`.
- **H4 también ocurre en MaxInt64:** `= #9223372036854775806` coincide con un valor guardado de `9223372036854775807`. Un literal que desborda coincide con un valor guardado en MaxInt64. En plataformas de 32 bits todos estos límites se trasladan a MaxInt32. Las pruebas numéricas asumen `int` de 64 bits y se omiten en otro caso.
- **H5 es más amplio de lo descrito.** Devuelven el error sin `ErrProvider`:
  - los cuatro métodos de jerarquía;
  - `ConceptExists`;
  - `AllConcepts`;
  - `RelationshipTargets`;
  - `RefsetsContainingMembers` (`^R`);
  - `ResolveIdentifier`.

  Los otros ocho métodos obligatorios que el evaluador llama sí agregan `ErrProvider`. Hay métodos, como `Descendants`, con llamadas que agregan el sentinel y otras que no; conviene envolver en un único punto.
- **`RelationshipSources` es obligatorio en el contrato, pero el evaluador nunca lo llama.** El atributo reverso usa `RelationshipTargets` con el conjunto de valores como fuentes. Cada implementador escribe un método que el motor no usa.
- **H3: la fixture no puede verificar un arreglo completo.** Las membresías de dialecto se asocian a concepto e idioma, no a una descripción. Un arreglo que correlacione por idioma pasaría las pruebas actuales. Cerrarlo exige aceptar un ID de descripción en `FixtureDialectMember`.

### 11.4 Decisiones que siguen abiertas

- **Comparación entre literal decimal y valor entero.** No se encontró un texto normativo que la defina, así que no hay casos que la fijen.
- **MRCM `null`.** No está decidido si significa un modelo sin reglas o la ausencia de documento.

### 11.5 Corrección a pruebas existentes

`(en OR es)` no es un conjunto de dos idiomas: `languageCodeSet` separa los códigos con espacios, y `OR`, de dos letras, se lee como un tercer código. Dos pruebas existentes usaban esa forma y no distinguían «cualquiera de» de «sólo el primer valor». Se reemplazaron por los pares `(es fr)` y `(fr es)`, que distinguen además «sólo el último valor» y «filtro ignorado». La suite de conformidad pasa de 153 a 154 casos. El README decía que 107 casos fijan IDs concretos; el recuento actual es 118.

### 11.6 Primera comparación contra un motor de referencia

Se cargó el mismo RF2 sintético, generado desde un `FixtureSpec`, en Hermes v1.4.1629:

- **Confirmó de forma independiente** H1, H2 y H3 en filtros de un solo bloque.
- **Comparte H4:** el acuerdo con go-ecl en números cercanos a 2^53 incluyó tres resultados en que ambos motores estaban equivocados.
- **Se equivoca con varios bloques:** exige una sola descripción para todos ellos, en contra de los ejemplos 8.1.8 y 8.2.2.
- **No soporta** `moduleId` ni `language` en filtros de descripción.

Conclusión para la sección 8: una referencia sirve como segunda opinión dentro de su subconjunto confiable, nunca como juez. Coincidir con ella no prueba corrección, y cada caso necesita un resultado derivado de la especificación.

### 11.7 Estado tras corregir el ámbito de los filtros

El parser emite un `ast.Filtered` por bloque `{{ D … }}` o `{{ C … }}`, anidados en orden, y cada uno indica su familia en el campo nuevo `Family`. Los bloques `{{ M … }}` siguen juntos sobre el `MemberOf` que filtran.

- **H1 y H2b quedan corregidos.**
- **H2 queda corregido para `type`, `language` y `moduleId`**, cuyos valores se intersectan dentro del bloque. Dos cláusulas `term` en un mismo bloque se informan como `ErrUnsupportedFeature`: dar la respuesta exacta exige una capacidad nueva del provider, igual que H3.
- **Cambio de comportamiento:** `{{ active = … }}`, `{{ moduleId = … }}` y `{{ effectiveTime = … }}` sin prefijo filtran descripciones, como indica la gramática, que hace opcional la `D`. Antes filtraban conceptos. Como el evaluador no enviaba esos campos de `DescriptionFilterOpts` hasta ahora, exigen la capacidad nueva `DescriptionMetadataProvider`; sin ella se informan como `ErrUnsupportedFeature`. Un filtro `{{ M … }}` cuyo foco no tiene `^` también se informa como no soportado.
- **Filtros de miembro sobre la fila:** `{{ M active/moduleId/effectiveTime … }}` se envían como comparaciones de campo con el nombre de la columna RF2 (`active` como `"1"`/`"0"`, `moduleId`, `effectiveTime` como `YYYYMMDD`). Antes filtraban el concepto referenciado. Las comparaciones de orden sobre `effectiveTime` de miembro se informan como `ErrUnsupportedFeature`, porque `MemberFilterOpts` compara por igualdad. Un provider tiene que declarar que responde esas columnas implementando la capacidad nueva `MemberRowColumnsProvider`; sin ella el filtro se informa como no soportado, porque un provider anterior buscaría las columnas como campos adicionales y respondería vacío sin avisar. `VerifyContract` comprueba que todo miembro aparezca con `active = 1` o con `active = 0`, y que un valor que ninguna fila tiene no devuelva miembros. El provider de `providertest` implementa la capacidad y considera activa una fila que no declara `active`.
- **Filas inactivas:** `^ R` significa las filas activas. Cuando el provider implementa `MemberRowColumnsProvider` y `RefsetFieldProjector`, un filtro M sin `active` recibe una cláusula `active = 1` sobre la misma fila, así que el valor de una fila inactiva ya no califica al miembro a través de una fila activa hermana. Sin esas capacidades se mantiene la intersección con los miembros activos, como antes.
- **Formas que ahora se informan como `ErrUnsupportedFeature`** en lugar de evaluarse más amplias de lo escrito:
  - en descripciones, `moduleId !=`, dos cláusulas `effectiveTime` en un bloque y un conjunto de `effectiveTime` con `!=`. Un conjunto positivo se responde con una llamada por valor.
- **Corregido de paso:** la ruta negada de filtros de descripción consultaba sólo el primer valor de un conjunto de términos; `active = *` se leía como `active = true` (ahora `ActiveFilter.Any`: una descripción en cualquier estado, ninguna restricción en conceptos, y en miembros las filas activas e inactivas por igual); un operando sin tipos en un conjunto de `typeId` anulaba los demás.
- **Pendiente, H2m:** los bloques `{{ M … }}` separados siguen exigiendo una misma fila, porque comparten un nodo sobre el `MemberOf`. Corregirlo toca la proyección `^ [campo]`, donde varios bloques no tienen una semántica clara. Queda como fallo esperado estricto. (Corregido en §11.10.)
- **H3 se extiende a las cláusulas de metadatos:** `active` y `dialectId` en un mismo bloque se resuelven con dos llamadas y pueden satisfacerse en descripciones distintas.

H4, H5 y H6 no cambian.

### 11.8 Estado tras corregir H4, H5 y H6

- **H4 queda corregido.** Las comparaciones numéricas son exactas: el literal guarda su texto (`IntegerValue.Text`, `DecimalValue.Text`). Enteros y decimales se siguen comparando por valor, como antes, pero sin pasar por `float64` cuando la conversión pierde precisión. Un entero que cabe en `int64`, o un valor de hasta 15 dígitos significativos, se compara sin asignar memoria; el resto, con `math/big.Rat`. Los valores guardados se leen sólo como decimal (no `"1/2"` ni `"0x10"`). Un literal de la expresión admite hasta 4000 dígitos y de exponente; un valor guardado, que es dato de la terminología y no entrada del usuario, hasta 100 000, y sólo más allá se compara por su `float64`. En filtros de miembro, un literal viaja en forma canónica (`#1.50` como `"1.5"`, `#+1` como `"1"`). `Text` cuenta mientras coincide con `Value`: si un nodo parseado se edita, manda el `Value` nuevo. Un entero o una cardinalidad que no cabe en `int` es ahora un error de parseo que nombra el literal, en lugar de saturar en silencio. En plataformas de 32 bits ese límite es `MaxInt32`.
- **H5 queda corregido.** Todo error que devuelve el provider pasa por una única función, `providerError`, que agrega `ErrProvider` y el nombre del método. Las dos excepciones son `ErrUnsupportedFeature`, que una capacidad usa para rechazar una forma, y el contexto de quien llama ya terminado: ninguna de las dos es un backend caído. La segunda se decide por `ctx.Err()`, no por la forma del error: un timeout propio del backend con el contexto vivo sí es `ErrProvider`, y un driver que responde a una cancelación con su propio error no lo es.
- **Límite nuevo:** un bloque que combina un conjunto de términos con uno de `effectiveTime` se rechaza si el producto supera 256 llamadas al provider. Un conjunto de términos solo sigue sin límite propio.
- **H6 queda corregido.** El loader de MRCM rechaza datos después del modelo y cualquier `ruleStrengthId` distinto de las dos fuerzas que el validador conoce; `Validate` aplica la misma regla a un modelo construido en Go. `null` sigue abierto como decisión de política.

Quedan abiertos H2 (dos cláusulas `term` en un bloque), H2m y H3, que comparten la necesidad de una capacidad que resuelva un bloque completo sobre una sola descripción o fila, además de H7 y H8. (H2 y H3 se corrigen en §11.9; H2m, en §11.10, sin capacidad nueva.)

### 11.9 Bloque de descripción completo: `DescriptionRowMatcher`

Capacidad opcional nueva. El evaluador envía el bloque `{{ D … }}` entero, como condiciones que deben cumplirse todas sobre **una misma fila de descripción**: términos, tipos, idiomas, módulos, IDs, `effectiveTime`, `active` y membresías de dialecto, cada una con su propia negación. Cuando el provider la implementa, todo bloque de descripción pasa por ella y no se reparte en llamadas ni se intersecta a nivel de concepto.

- **H2 queda corregido** para providers con la capacidad: dos cláusulas `term` en un bloque deben cumplirse en la misma descripción.
- **H3 queda corregido** para providers con la capacidad: término, o `active`, y dialecto en un bloque se cumplen sobre la misma descripción. `FixtureDialectMember.DescriptionID` permite que la fixture asigne una membresía a una descripción concreta, en vez de a concepto e idioma.
- También se responden formas que la ruta por dimensión rechaza: `moduleId !=`, varias cláusulas `effectiveTime`, un conjunto de términos negado y un filtro de ID con una cláusula negada al lado.
- **Sin la capacidad** se mantiene la ruta anterior, y sus marcas H2 y H3 siguen registradas.
- La negación de dialecto conserva la lectura que ya tenía `MatchDialect`: `dialectId != X` se cumple por una membresía en otro dialecto, y una descripción sin membresías no cumple ninguna de las dos formas.
- Implementar la capacidad declara que se respetan **todas** las condiciones; `VerifyContract` lo comprueba con un valor imposible por dimensión y con pares que ninguna fila puede cumplir a la vez (dos idiomas, dos tipos).
- La conformance corre en las dos rutas: con la capacidad y, con `providertest.WithoutRowMatcher`, sin ella. Una cláusula se interpreta igual en ambas mediante un normalizador común.
- **Cambio en el provider de referencia:** `MatchDialect` lee ahora las membresías por descripción, como la ruta nueva. Una membresía cuyo concepto no tiene descripción en ese idioma ya no coincide con nada, porque una fila RF2 de refset de idioma siempre nombra una descripción.

En ese momento quedaba abierto H2m, corregido en §11.10.

### 11.10 Bloques de miembro separados (H2m)

- **H2m queda corregido sin capacidad nueva.** El parser emite un `Filtered` por bloque `{{ M … }}`, anidados en orden de aparición directamente sobre el `MemberOf`, como ya hacía con D y C. Sobre un `^ R` simple, cada bloque se responde por su cuenta con lo que el contrato ya tiene, igual que un bloque solo: una cláusula va a `RefsetMembersFiltered`, y varias van a `ProjectRefsetField` con sus `Filters`. Con `MemberRowColumnsProvider` y `RefsetFieldProjector`, el bloque recibe además `active = 1` sobre la misma fila, así que también va a `ProjectRefsetField`. Luego se intersectan los miembros de los bloques. Todos los bloques se validan antes de enviar ninguno, y los refsets y sus miembros se piden una sola vez. `^ 447562003 {{ M mapGroup = "2" }} {{ M mapTarget = "I10" }}` devuelve `95566004`, que cumple cada bloque en una fila distinta.
- **Proyección:** sobre `^ [campo] R`, todos los bloques se cumplen sobre la fila proyectada, como si fueran uno. Una proyección devuelve una columna de una fila, así que "cada bloque en su fila" no deja una fila de la cual proyectar. Es la lectura elegida y queda documentada en `ast.Filtered`.
- **Cambio de comportamiento:** `{{ M active = true }} {{ M active = false }}` ya no es una contradicción. Lo cumple un miembro con una fila activa y otra inactiva. Dentro de un mismo bloque sigue sin cumplirse.
- **Árbol armado a mano:** un filtro de miembro cuyo operando no es un `^`, en cualquier familia (incluida `""`), se informa como `ErrUnsupportedFeature`. Antes, nodos `""` anidados leían los miembros del nodo interior como refsets y devolvían vacío sin avisar.
- **Cambio en el AST:** quien recorra el árbol ve ahora un nodo por bloque M, en vez de uno compartido. En este repositorio sólo lo consumen el evaluador y `explain` del CLI; `gofhir/server` no importa `ecl/ast`.

## Apéndice A. Reproducción de los hallazgos

Guardar el siguiente programa fuera del paquete, por ejemplo en `/tmp/go-ecl-audit-probe.go`, y ejecutar desde la raíz del repositorio:

```sh
go run /tmp/go-ecl-audit-probe.go
```

Es un programa diagnóstico deliberadamente pequeño, no un provider productivo ni un test permanente. Su salida se observó contra el commit indicado. Los datos son sintéticos; los números y filtros se escogen para separar casos que no deberían confundirse.

```go
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/ast"
	"github.com/gofhir/ecl/ecl/providertest"
	"github.com/gofhir/ecl/mrcm"
	"github.com/gofhir/ecl/scg"
)

type broken struct{ ecl.UnimplementedDataProvider }

func (broken) ConceptExists(context.Context, []string) (ecl.Set, error) {
	return nil, errors.New("backend down")
}

func main() {
	ctx := context.Background()
	inactive := false
	p := providertest.NewInMemoryProvider(&providertest.FixtureSpec{
		Concepts:      []providertest.FixtureConcept{{ID: "22298006"}, {ID: "363698007"}},
		Descriptions:  []providertest.FixtureDescription{{Concept: "22298006", Language: "en", Value: "heart"}, {Concept: "22298006", Language: "es", Value: "corazon", ModuleID: "404684003"}, {Concept: "22298006", Language: "es", Value: "old", Active: &inactive}},
		Relationships: []providertest.FixtureRelationship{{Source: "22298006", TypeID: "363698007", Concrete: &providertest.FixtureConcrete{Kind: "integer", Value: "9007199254740992"}}},
		Dialects:      []providertest.FixtureDialectMember{{ConceptID: "22298006", Language: "es", DialectID: "900000000000508004"}},
	})
	for _, s := range []string{
		`* : 363698007 = #9007199254740993`,
		`* : 363698007 = #99999999999999999999999999999`,
		`* {{ D language = en, language = es }}`,
		`* {{ D term = "nonexistent", term = "heart" }}`,
		`* {{ D term = "heart", dialectId = 900000000000508004 }}`,
		`* {{ D active = false }}`,
		`* {{ D moduleId = 404684003 }}`,
	} {
		a, err := ecl.Parse(s)
		fmt.Printf("%s\nparse: %v\n", s, err)
		if err == nil {
			r, e := ecl.Evaluate(ctx, a, p)
			fmt.Printf("result: %v error: %v\n", r, e)
			if ref, ok := a.(*ast.Refined); ok {
				fmt.Printf("literal: %+v\n", ref.Refinement.Ungrouped[0].Value)
			}
		}
	}
	for _, s := range []string{`{} {"unexpected":true}`, `null`, `{"domains":[{"attributeId":"363698007","domainEcl":"*","ruleStrengthId":"404684003"}]}`} {
		m, err := mrcm.LoadFromBytes([]byte(s))
		fmt.Printf("MRCM %s => model=%+v error=%v\n", s, m, err)
	}
	a, _ := ecl.Parse("22298006")
	_, err := ecl.Evaluate(ctx, a, broken{})
	fmt.Printf("provider error: %v; ErrProvider=%v\n", err, errors.Is(err, ecl.ErrProvider))
	model := &mrcm.Model{Domains: []mrcm.AttributeDomain{{AttributeID: "363698007", DomainECL: "*", Cardinality: mrcm.Cardinality{Max: -1}}}, Ranges: []mrcm.AttributeRange{{AttributeID: "363698007", RangeECL: "74281007"}}}
	expr, _ := scg.Parse("22298006 : 363698007 = #12")
	result, err := mrcm.Validate(ctx, expr, model, p)
	fmt.Printf("concrete value against concept range: %+v error=%v\n", result, err)
}
```

