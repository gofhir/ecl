package oracle

// Case is one differential comparison: an ECL expression this library and the
// server both evaluate.
type Case struct {
	// Expr must have a BOUNDED focus. `*` is not usable: resolving it would mean
	// pulling half a million codes across the wire, and the harness refuses
	// (AllConcepts returns ErrNotAnswerable). Every expression below is therefore
	// anchored on a small subhierarchy, which is also what keeps a case to a
	// handful of requests.
	Expr string

	// Exercises names the composition being tested. It is the reason the case is
	// in the corpus: a case that only exercises a primitive compares the server
	// against itself and proves nothing.
	Exercises string
}

// Corpus is the set of expressions the differential test runs.
//
// Chosen on one principle: each must make THIS LIBRARY do the composing. The
// server answers "what are the descendants of X" and "what are the targets of
// attribute A on set S"; everything between those answers and the final set —
// refinement, cardinality, negation, grouping, reverse, set algebra — is code in
// this repository, and is where every semantic bug found so far has lived.
//
// Two SCTIDs anchor most of it: 22298006 (myocardial infarction, ~135
// descendants) and 74281007 (myocardium structure). Both are small enough to fetch
// per-concept relationships for, and both carry the grouped
// finding-site/morphology attributes that make grouping observable.
var Corpus = []Case{
	// ── Set algebra ──────────────────────────────────────────────────────────
	// The plain hierarchy operators are deliberately absent: `<< X` is answered by
	// a single passthrough query, so comparing it to the server's answer compares
	// the server with itself.
	{
		Expr:      "(< 22298006) MINUS (< 57054005)",
		Exercises: "exclusion, computed here as Set.Minus",
	},
	{
		Expr:      "(<< 22298006) AND (<< 414545008)",
		Exercises: "conjunction as an intersection",
	},
	{
		Expr:      "(<< 57054005) OR (<< 194802003)",
		Exercises: "disjunction as a union",
	},
	{
		Expr:      "(<< 22298006) MINUS (<< 22298006 : 363698007 = 74281007)",
		Exercises: "exclusion whose right operand is itself a refinement",
	},

	// ── Refinement ───────────────────────────────────────────────────────────
	{
		Expr:      "<< 22298006 : 363698007 = 74281007",
		Exercises: "attribute value as a single concept",
	},
	{
		Expr:      "<< 22298006 : 363698007 = << 74281007",
		Exercises: "attribute value as a nested constraint",
	},
	{
		Expr:      "<< 22298006 : 116676008 = 55641003",
		Exercises: "a second attribute type, to catch a hardcoded one",
	},
	{
		Expr:      "<< 22298006 : 363698007 = 74281007, 116676008 = 55641003",
		Exercises: "conjunction of attributes — both must hold, ungrouped",
	},
	{
		Expr:      "<< 22298006 : 363698007 = 74281007 OR 116676008 = 55641003",
		Exercises: "DISJUNCTION of attributes, which this library used to evaluate as AND",
	},
	{
		Expr:      "<< 22298006 : 363698007 = *",
		Exercises: "attribute presence with a wildcard value",
	},

	// ── Negation ─────────────────────────────────────────────────────────────
	// "!=" negates the VALUE, not the existence of the attribute: a concept with a
	// finding site other than myocardium qualifies, one with no finding site at
	// all does not. This library had the other reading.
	{
		Expr:      "<< 22298006 : 363698007 != 74281007",
		Exercises: `"!=" on a concept-valued attribute`,
	},
	{
		// Deliberately a value set that only SOME of the focus concepts avoid.
		// `363698007 != << 74281007` looks like a better test and is worthless:
		// every descendant of myocardial infarction has a finding site under
		// myocardium structure, so both sides return the empty set and agree
		// without deciding anything.
		Expr:      "<< 22298006 : 116676008 != (55641003 OR 55470003)",
		Exercises: `"!=" against a set of values`,
	},

	// ── Cardinality ──────────────────────────────────────────────────────────
	// [0..0] is the discriminating one. It used to be indistinguishable from
	// [1..1] and [2..*] because matching stopped at the first hit.
	{
		Expr:      "<< 22298006 : [1..1] 363698007 = *",
		Exercises: "exact cardinality 1",
	},
	{
		Expr:      "<< 22298006 : [2..*] 363698007 = *",
		Exercises: "at least 2 — must not coincide with [1..*]",
	},
	{
		// On attribute 42752001 rather than 363698007: every descendant of
		// myocardial infarction has a finding site, so `[0..0] 363698007 = *` is
		// empty on both sides and proves nothing. "Due to" is present on some and
		// absent on others, which is what makes the comparison bite.
		Expr:      "<< 22298006 : [0..0] 42752001 = *",
		Exercises: "zero cardinality, the complement of attribute presence",
	},
	{
		Expr:      "<< 22298006 : [1..*] 116676008 = *",
		Exercises: "the default cardinality stated explicitly",
	},

	// ── Attribute groups ─────────────────────────────────────────────────────
	// Grouped cases are the ones that read relationship GROUPS out of the server's
	// per-concept response, so they are also what would catch the harness
	// misreading that response.
	{
		Expr:      "<< 22298006 : { 363698007 = 74281007, 116676008 = 55641003 }",
		Exercises: "co-occurrence in ONE relationship group",
	},
	{
		Expr:      "<< 22298006 : { 363698007 = 74281007 }, { 116676008 = 55641003 }",
		Exercises: "two groups, which need not be the same group",
	},
	{
		Expr:      "<< 22298006 : [2..*] { 363698007 = * }",
		Exercises: "GROUP cardinality — counting groups, not attributes",
	},
	{
		Expr:      "<< 22298006 : [0..0] { 363698007 = 74281007 }",
		Exercises: "zero group cardinality",
	},

	// ── Ungrouped attributes and concrete values ─────────────────────────────
	// A different focus, because myocardial infarction has neither. These two are
	// the only cases that exercise a top-level FHIR property (an ungrouped
	// attribute) and a non-concept value; the rest of the corpus would pass with
	// both paths broken.
	{
		Expr:      "<< 322236009 : 411116001 = *",
		Exercises: "an UNGROUPED attribute, which the server reports differently from a grouped one",
	},
	{
		Expr:      "<< 322236009 : 1142139005 >= #1",
		Exercises: "a CONCRETE value with a numeric comparison",
	},

	// ── Dot notation ─────────────────────────────────────────────────────────
	{
		Expr:      "(<< 22298006).363698007",
		Exercises: "dereferencing an attribute over a set",
	},
	{
		Expr:      "(<< 22298006).116676008",
		Exercises: "dereferencing a second attribute type",
	},

	// ── Reverse attributes ───────────────────────────────────────────────────
	// The focus is bounded on the TARGET side here, since the reverse form selects
	// concepts that are values of an attribute on the given set.
	{
		Expr:      "<< 74281007 : R 363698007 = 22298006",
		Exercises: "reverse attribute, bounded focus",
	},
	{
		Expr:      "<< 74281007 : R 363698007 = << 22298006",
		Exercises: "reverse attribute whose source set is a constraint",
	},
	{
		Expr:      "<< 74281007 : [1..1] R 363698007 = << 22298006",
		Exercises: "reverse CARDINALITY — needs InboundRelationshipsProvider, so this case is expected to report unsupported rather than diverge",
	},

	// ── Nesting ──────────────────────────────────────────────────────────────
	{
		Expr:      "<< 22298006 : 363698007 = ((<< 74281007) MINUS (<< 74281007 : 363698007 = *))",
		Exercises: "a compound expression as an attribute value",
	},
	{
		Expr:      "(<< 22298006 : 363698007 = 74281007) : 116676008 = 55641003",
		Exercises: "refining the result of a refinement",
	},
}
