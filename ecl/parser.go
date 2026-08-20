package ecl

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/gofhir/ecl/ecl/ast"
	"github.com/gofhir/ecl/ecl/grammar"
)

// Parse parses an ECL expression string and returns the AST.
func Parse(input string) (ast.Expression, error) {
	errListener := &eclErrorListener{}

	lexer := grammar.NewECLLexer(antlr.NewInputStream(input))
	// The lexer needs the listener too. Without this, an unrecognizable
	// character makes ANTLR's default ConsoleErrorListener write to os.Stderr
	// (from inside a library), drop the character from the token stream, and
	// let Parse return a corrupted AST with a nil error.
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(errListener)

	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := grammar.NewECLParser(stream)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errListener)

	tree := parser.Expressionconstraint()

	// The expressionconstraint rule in ECL.g4 does not end in EOF, so ANTLR
	// stops at the first complete parse and discards the rest without
	// reporting anything: "11687002 GARBAGE" parsed as 11687002 with err ==
	// nil, and "A MINUS B MINUS C" silently truncated to "A MINUS B".
	//
	// The message quotes the remaining INPUT, not the token text: the ECL
	// lexer is character-level, so GetText() would be just "G" here.
	if tok := stream.LT(1); tok != nil && tok.GetTokenType() != antlr.TokenEOF {
		// GetStart is a RUNE index, not a byte offset, so slicing the string
		// directly cut multi-byte characters in half and produced mojibake:
		// `404684003 |ááá| GARBAGE` reported trailing input "\xa1| GARBAGE".
		rest := input
		if runes := []rune(input); tok.GetStart() >= 0 && tok.GetStart() < len(runes) {
			rest = string(runes[tok.GetStart():])
		}
		errListener.errs = append(errListener.errs, SyntaxError{
			Line:   tok.GetLine(),
			Column: tok.GetColumn(),
			Msg:    fmt.Sprintf("unexpected trailing input %q", strings.TrimSpace(rest)),
		})
	}

	if len(errListener.errs) > 0 {
		return nil, &ParseError{Errors: errListener.errs}
	}

	visitor := &astBuilder{}
	result := visitor.Visit(tree)
	if expr, ok := result.(ast.Expression); ok {
		return expr, nil
	}
	return nil, fmt.Errorf("unexpected parse result: %T", result)
}

// ---------------------------------------------------------------------------
// Error listener
// ---------------------------------------------------------------------------.

// SyntaxError is a single syntax error reported while parsing an expression.
type SyntaxError struct {
	// Line is the 1-based line the error was reported on.
	Line int
	// Column is the 0-based character offset within the line.
	Column int
	// Msg is the underlying description from the parser or lexer.
	Msg string
}

// Error renders the error in the "syntax error at line:column: msg" form.
func (e SyntaxError) Error() string {
	return fmt.Sprintf("syntax error at %d:%d: %s", e.Line, e.Column, e.Msg)
}

// ParseError collects every syntax error found in one expression. Callers can
// classify a failure with errors.As instead of matching on message text:
//
//	var pe *ecl.ParseError
//	if errors.As(err, &pe) { /* 400 Bad Request, report pe.Errors */ }
type ParseError struct {
	// Errors holds every reported error, in the order the parser found them.
	Errors []SyntaxError
}

// Error renders every collected error, separated by "; ".
//
// The single-error form is byte-identical to what Parse returned before
// ParseError existed ("syntax error at line:column: msg"), so callers that
// match on that text keep working. Callers that add their own prefix (the CLI
// does) stay unaffected too.
func (e *ParseError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	msgs := make([]string, 0, len(e.Errors))
	for _, se := range e.Errors {
		msgs = append(msgs, se.Error())
	}
	return strings.Join(msgs, "; ")
}

type eclErrorListener struct {
	antlr.DefaultErrorListener
	errs []SyntaxError
}

// SyntaxError appends the error. It must accumulate rather than overwrite: the
// previous implementation assigned to a single field, so only the last error of
// a batch survived.
func (l *eclErrorListener) SyntaxError(_ antlr.Recognizer, _ any, line, column int, msg string, _ antlr.RecognitionException) {
	l.errs = append(l.errs, SyntaxError{Line: line, Column: column, Msg: msg})
}

// ---------------------------------------------------------------------------
// AST builder visitor
// ---------------------------------------------------------------------------.

type astBuilder struct {
	grammar.BaseECLVisitor
}

// Visit dispatches to the correct typed Visit method via the Accept pattern.
func (v *astBuilder) Visit(tree antlr.ParseTree) any {
	if tree == nil {
		return nil
	}
	return tree.Accept(v)
}

// visitExpr is a convenience wrapper that casts the result to ast.Expression.
func (v *astBuilder) visitExpr(tree antlr.ParseTree) ast.Expression {
	if tree == nil {
		return nil
	}
	r := v.Visit(tree)
	if expr, ok := r.(ast.Expression); ok {
		return expr
	}
	return nil
}

// ---------------------------------------------------------------------------
// Top-level expression
// ---------------------------------------------------------------------------.

func (v *astBuilder) VisitExpressionconstraint(ctx *grammar.ExpressionconstraintContext) any {
	if c := ctx.Refinedexpressionconstraint(); c != nil {
		return v.Visit(c)
	}
	if c := ctx.Compoundexpressionconstraint(); c != nil {
		return v.Visit(c)
	}
	if c := ctx.Dottedexpressionconstraint(); c != nil {
		return v.Visit(c)
	}
	if c := ctx.Subexpressionconstraint(); c != nil {
		return v.Visit(c)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Compound expressions (AND / OR / MINUS)
// ---------------------------------------------------------------------------.

func (v *astBuilder) VisitCompoundexpressionconstraint(ctx *grammar.CompoundexpressionconstraintContext) any {
	if c := ctx.Conjunctionexpressionconstraint(); c != nil {
		return v.Visit(c)
	}
	if c := ctx.Disjunctionexpressionconstraint(); c != nil {
		return v.Visit(c)
	}
	if c := ctx.Exclusionexpressionconstraint(); c != nil {
		return v.Visit(c)
	}
	return nil
}

func (v *astBuilder) VisitConjunctionexpressionconstraint(ctx *grammar.ConjunctionexpressionconstraintContext) any {
	subs := ctx.AllSubexpressionconstraint()
	if len(subs) == 0 {
		return nil
	}
	result := v.visitExpr(subs[0])
	for i := 1; i < len(subs); i++ {
		result = &ast.And{Left: result, Right: v.visitExpr(subs[i])}
	}
	return result
}

func (v *astBuilder) VisitDisjunctionexpressionconstraint(ctx *grammar.DisjunctionexpressionconstraintContext) any {
	subs := ctx.AllSubexpressionconstraint()
	if len(subs) == 0 {
		return nil
	}
	result := v.visitExpr(subs[0])
	for i := 1; i < len(subs); i++ {
		result = &ast.Or{Left: result, Right: v.visitExpr(subs[i])}
	}
	return result
}

func (v *astBuilder) VisitExclusionexpressionconstraint(ctx *grammar.ExclusionexpressionconstraintContext) any {
	subs := ctx.AllSubexpressionconstraint()
	if len(subs) < 2 {
		return nil
	}
	return &ast.Minus{
		Left:  v.visitExpr(subs[0]),
		Right: v.visitExpr(subs[1]),
	}
}

// ---------------------------------------------------------------------------
// Refined expression
// ---------------------------------------------------------------------------.

func (v *astBuilder) VisitRefinedexpressionconstraint(ctx *grammar.RefinedexpressionconstraintContext) any {
	focus := v.visitExpr(ctx.Subexpressionconstraint())
	refinement := v.visitRefinement(ctx.Eclrefinement())
	return &ast.Refined{
		Focus:      focus,
		Refinement: refinement,
	}
}

// ---------------------------------------------------------------------------
// Dotted expression
// ---------------------------------------------------------------------------.

func (v *astBuilder) VisitDottedexpressionconstraint(ctx *grammar.DottedexpressionconstraintContext) any {
	result := v.visitExpr(ctx.Subexpressionconstraint())
	for _, dattr := range ctx.AllDottedexpressionattribute() {
		attrName := v.visitDottedExpressionAttribute(dattr)
		result = &ast.DotExpression{
			Source:    result,
			Attribute: attrName,
		}
	}
	return result
}

func (v *astBuilder) visitDottedExpressionAttribute(ctx grammar.IDottedexpressionattributeContext) ast.Expression {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.DottedexpressionattributeContext)
	if !ok {
		return nil
	}
	nameCtx := concrete.Eclattributename()
	if nameCtx == nil {
		return nil
	}
	return v.visitExpr(nameCtx)
}

func (v *astBuilder) VisitDottedexpressionattribute(ctx *grammar.DottedexpressionattributeContext) any {
	if ctx.Eclattributename() != nil {
		return v.Visit(ctx.Eclattributename())
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sub-expression constraint (the workhorse)
// ---------------------------------------------------------------------------.

func (v *astBuilder) VisitSubexpressionconstraint(ctx *grammar.SubexpressionconstraintContext) any {
	// 1. Determine the focus concept or nested expression
	var focusExpr ast.Expression

	// Check for nested parenthesised expression
	if ctx.LEFT_PAREN() != nil && ctx.Expressionconstraint() != nil {
		inner := v.visitExpr(ctx.Expressionconstraint())
		focusExpr = &ast.Nested{Inner: inner}
	} else if ctx.Eclfocusconcept() != nil {
		focusExpr = v.visitExpr(ctx.Eclfocusconcept())
	}

	// 2. Apply refset operator (memberOf or ^R)
	if ctx.RefsetOperator() != nil {
		refOp := ctx.RefsetOperator().(*grammar.RefsetOperatorContext)
		if refOp.Memberof() != nil {
			mo := &ast.MemberOf{Operand: focusExpr}
			// Check for field names ^[field1,field2]
			memberCtx := refOp.Memberof().(*grammar.MemberofContext)
			if memberCtx.Refsetfieldnameset() != nil {
				fsCtx := memberCtx.Refsetfieldnameset().(*grammar.RefsetfieldnamesetContext)
				for _, fn := range fsCtx.AllRefsetfieldname() {
					mo.Fields = append(mo.Fields, fn.GetText())
				}
			}
			focusExpr = mo
		} else if refOp.RefsetContainingAny() != nil {
			focusExpr = &ast.RefsetContainingAny{Operand: focusExpr}
		}
	}

	// 3. Apply constraint operator (hierarchy)
	if ctx.Constraintoperator() != nil {
		focusExpr = v.applyConstraintOperator(ctx.Constraintoperator(), focusExpr)
	}

	// 4. Apply description/concept/member filter constraints. Each constraint
	// can contain multiple sub-clauses (term, type, language, active, module,
	// etc.) which we extract into typed ast.Filter nodes.
	filters := make([]ast.Filter, 0,
		len(ctx.AllDescriptionfilterconstraint())+
			len(ctx.AllConceptfilterconstraint())+
			len(ctx.AllMemberfilterconstraint()))
	for _, fc := range ctx.AllDescriptionfilterconstraint() {
		filters = append(filters, v.buildDescriptionFilterClauses(fc)...)
	}
	for _, fc := range ctx.AllConceptfilterconstraint() {
		filters = append(filters, v.buildConceptFilterClauses(fc)...)
	}
	for _, fc := range ctx.AllMemberfilterconstraint() {
		filters = append(filters, v.buildMemberFilterClauses(fc)...)
	}
	if len(filters) > 0 {
		focusExpr = &ast.Filtered{Operand: focusExpr, Filters: filters}
	}

	// 5. Apply history supplement
	if ctx.Historysupplement() != nil {
		focusExpr = v.visitHistorySupplement(ctx.Historysupplement(), focusExpr)
	}

	return focusExpr
}

func (v *astBuilder) applyConstraintOperator(ctx grammar.IConstraintoperatorContext, operand ast.Expression) ast.Expression {
	if ctx == nil {
		return operand
	}
	concrete, ok := ctx.(*grammar.ConstraintoperatorContext)
	if !ok {
		return operand
	}
	if concrete.Descendantof() != nil {
		return &ast.DescendantOf{Operand: operand}
	}
	if concrete.Descendantorselfof() != nil {
		return &ast.DescendantOrSelfOf{Operand: operand}
	}
	if concrete.Childof() != nil {
		return &ast.ChildOf{Operand: operand}
	}
	if concrete.Childorselfof() != nil {
		return &ast.ChildOrSelfOf{Operand: operand}
	}
	if concrete.Ancestorof() != nil {
		return &ast.AncestorOf{Operand: operand}
	}
	if concrete.Ancestororselfof() != nil {
		return &ast.AncestorOrSelfOf{Operand: operand}
	}
	if concrete.Parentof() != nil {
		return &ast.ParentOf{Operand: operand}
	}
	if concrete.Parentorselfof() != nil {
		return &ast.ParentOrSelfOf{Operand: operand}
	}
	if concrete.Top() != nil {
		return &ast.Top{Operand: operand}
	}
	if concrete.Bottom() != nil {
		return &ast.Bottom{Operand: operand}
	}
	return operand
}

// ---------------------------------------------------------------------------
// Focus concept
// ---------------------------------------------------------------------------.

func (v *astBuilder) VisitEclfocusconcept(ctx *grammar.EclfocusconceptContext) any {
	if ctx.Eclconceptreference() != nil {
		return v.Visit(ctx.Eclconceptreference())
	}
	if ctx.Wildcard() != nil {
		return &ast.Any{}
	}
	if ctx.Altidentifier() != nil {
		return v.Visit(ctx.Altidentifier())
	}
	return nil
}

func (v *astBuilder) VisitEclconceptreference(ctx *grammar.EclconceptreferenceContext) any {
	ref := &ast.ConceptRef{}
	if ctx.Conceptid() != nil {
		ref.ID = ctx.Conceptid().GetText()
	}
	if ctx.Term() != nil {
		ref.Term = strings.TrimSpace(ctx.Term().GetText())
	}
	return ref
}

func (v *astBuilder) VisitWildcard(_ *grammar.WildcardContext) any {
	return &ast.Any{}
}

// buildDescriptionFilterClauses extracts typed filter clauses from a
// descriptionfilterconstraint context. Supported sub-rules:
//   - termfilter             → *ast.TermFilter
//   - typefilter             → *ast.TypeFilter
//   - languagefilter         → *ast.LanguageFilter
//   - dialectfilter          → *ast.DialectFilter (best-effort — see evaluator)
//   - activefilter           → *ast.ActiveFilter
//   - modulefilter           → *ast.ModuleFilter
//   - effectivetimefilter    → *ast.EffectiveTimeFilter
//
// descriptionidfilter and acceptabilityfilter sub-rules are currently not
// emitted — they require additional AST types and are deferred.
func (v *astBuilder) buildDescriptionFilterClauses(fc grammar.IDescriptionfilterconstraintContext) []ast.Filter {
	if fc == nil {
		return nil
	}
	concrete, ok := fc.(*grammar.DescriptionfilterconstraintContext)
	if !ok {
		return nil
	}

	var out []ast.Filter
	for _, df := range concrete.AllDescriptionfilter() {
		d, ok := df.(*grammar.DescriptionfilterContext)
		if !ok {
			continue
		}
		// One entry per sub-rule of `descriptionfilter`. A table rather than a
		// chain of ifs so the set of handled branches is visible at a glance:
		// every sub-rule the grammar defines must appear here, or its clause
		// disappears from the AST and the query silently widens.
		for _, branch := range []struct {
			present bool
			build   func() ast.Filter
		}{
			{d.Termfilter() != nil, func() ast.Filter { return v.buildTermFilter(d.Termfilter()) }},
			{d.Typefilter() != nil, func() ast.Filter { return v.buildTypeFilter(d.Typefilter()) }},
			{d.Languagefilter() != nil, func() ast.Filter { return v.buildLanguageFilter(d.Languagefilter()) }},
			{d.Dialectfilter() != nil, func() ast.Filter { return v.buildDialectFilter(d.Dialectfilter()) }},
			{d.Activefilter() != nil, func() ast.Filter { return v.buildActiveFilter(d.Activefilter()) }},
			{d.Modulefilter() != nil, func() ast.Filter { return v.buildModuleFilter(d.Modulefilter()) }},
			{d.Effectivetimefilter() != nil, func() ast.Filter { return v.buildEffectiveTimeFilter(d.Effectivetimefilter()) }},
			{d.Descriptionidfilter() != nil, func() ast.Filter { return v.buildDescriptionIDFilter(d.Descriptionidfilter()) }},
		} {
			if !branch.present {
				continue
			}
			if f := branch.build(); f != nil && !isNilFilter(f) {
				out = append(out, f)
			}
			break
		}
	}
	return out
}

// isNilFilter reports whether f is a typed nil, which a builder returns when the
// clause carried nothing usable. Appending it would put a nil-valued interface in
// the filter list and panic later.
func isNilFilter(f ast.Filter) bool {
	switch x := f.(type) {
	case *ast.TermFilter:
		return x == nil
	case *ast.TypeFilter:
		return x == nil
	case *ast.LanguageFilter:
		return x == nil
	case *ast.DialectFilter:
		return x == nil
	case *ast.ActiveFilter:
		return x == nil
	case *ast.ModuleFilter:
		return x == nil
	case *ast.EffectiveTimeFilter:
		return x == nil
	case *ast.DescriptionIDFilter:
		return x == nil
	}
	return false
}

// buildConceptFilterClauses extracts typed clauses from a conceptfilterconstraint.
func (v *astBuilder) buildConceptFilterClauses(fc grammar.IConceptfilterconstraintContext) []ast.Filter {
	if fc == nil {
		return nil
	}
	concrete, ok := fc.(*grammar.ConceptfilterconstraintContext)
	if !ok {
		return nil
	}
	var out []ast.Filter
	for _, cf := range concrete.AllConceptfilter() {
		c, ok := cf.(*grammar.ConceptfilterContext)
		if !ok {
			continue
		}
		if x := c.Definitionstatusfilter(); x != nil {
			if f := v.buildDefinitionStatusFilter(x); f != nil {
				out = append(out, f)
			}
			continue
		}
		if x := c.Modulefilter(); x != nil {
			if f := v.buildModuleFilter(x); f != nil {
				out = append(out, f)
			}
			continue
		}
		if x := c.Effectivetimefilter(); x != nil {
			if f := v.buildEffectiveTimeFilter(x); f != nil {
				out = append(out, f)
			}
			continue
		}
		if x := c.Activefilter(); x != nil {
			if f := v.buildActiveFilter(x); f != nil {
				out = append(out, f)
			}
			continue
		}
	}
	return out
}

// buildMemberFilterClauses extracts clauses from a memberfilterconstraint.
func (v *astBuilder) buildMemberFilterClauses(fc grammar.IMemberfilterconstraintContext) []ast.Filter {
	if fc == nil {
		return nil
	}
	concrete, ok := fc.(*grammar.MemberfilterconstraintContext)
	if !ok {
		return nil
	}
	var out []ast.Filter
	for _, mf := range concrete.AllMemberfilter() {
		m, ok := mf.(*grammar.MemberfilterContext)
		if !ok {
			continue
		}
		if x := m.Modulefilter(); x != nil {
			if f := v.buildModuleFilter(x); f != nil {
				out = append(out, f)
			}
			continue
		}
		if x := m.Effectivetimefilter(); x != nil {
			if f := v.buildEffectiveTimeFilter(x); f != nil {
				out = append(out, f)
			}
			continue
		}
		if x := m.Activefilter(); x != nil {
			if f := v.buildActiveFilter(x); f != nil {
				out = append(out, f)
			}
			continue
		}
		if x := m.Memberfieldfilter(); x != nil {
			if f := v.buildMemberFieldFilter(x); f != nil {
				out = append(out, f)
			}
			continue
		}
	}
	return out
}

// --- Individual clause builders ---------------------------------------------.

func (v *astBuilder) buildTermFilter(ctx grammar.ITermfilterContext) *ast.TermFilter {
	concrete, ok := ctx.(*grammar.TermfilterContext)
	if !ok {
		return nil
	}
	op := "="
	if concrete.Stringcomparisonoperator() != nil {
		op = v.extractComparisonOp(concrete.Stringcomparisonoperator().GetText())
	}
	var terms []ast.SearchTerm
	if c := concrete.Typedsearchterm(); c != nil {
		terms = append(terms, searchTermOf(c))
	} else if set, ok := concrete.Typedsearchtermset().(*grammar.TypedsearchtermsetContext); ok && set != nil {
		// A set has any-of semantics and each member declares its own search
		// style. Taking GetText() of the whole set used to produce one "term"
		// containing the parentheses and the inner quotes, which matched nothing.
		for _, tst := range set.AllTypedsearchterm() {
			terms = append(terms, searchTermOf(tst))
		}
	}
	if len(terms) == 0 {
		return nil
	}

	tf := &ast.TermFilter{Op: op, Terms: terms}
	// Deprecated scalars, kept populated for readers written against v1.1.
	tf.Term = terms[0].Text           //nolint:staticcheck // deprecated field kept populated for v1 readers
	tf.MatchType = terms[0].MatchType //nolint:staticcheck // deprecated field kept populated for v1 readers
	return tf
}

// searchTermOf extracts one typedsearchterm with its search style.
func searchTermOf(ctx grammar.ITypedsearchtermContext) ast.SearchTerm {
	text, matchType := extractTypedSearchTerm(ctx)
	return ast.SearchTerm{Text: text, MatchType: matchType}
}

// extractTypedSearchTerm returns the raw search term and its match-type
// modifier ("match" or "wild") from a typedsearchterm context. Falls back to
// "match" when the expected sub-rule shape is not present.
func extractTypedSearchTerm(ctx grammar.ITypedsearchtermContext) (term, matchType string) {
	matchType = "match"
	concrete, ok := ctx.(*grammar.TypedsearchtermContext)
	if !ok {
		return unescapeECLString(stripWrappingQuotes(ctx.GetText()), false), matchType
	}
	if s := concrete.Matchsearchtermset(); s != nil {
		return unescapeECLString(stripWrappingQuotes(s.GetText()), false), "match"
	}
	if s := concrete.Wildsearchtermset(); s != nil {
		return unescapeECLString(stripWrappingQuotes(s.GetText()), true), "wild"
	}
	return unescapeECLString(stripWrappingQuotes(concrete.GetText()), false), matchType
}

// unescapeECLString decodes the escape sequences the ECL grammar defines:
//
//	escapedchar     : (bs qm) | (bs bs)              -- \" and \\
//	escapedwildchar : (bs qm) | (bs bs) | (bs star)  -- also \*
//
// The raw token text keeps the backslashes, so a term that reached the provider
// undecoded never matched: `{{ term = "a\"b" }}` searched for the six characters
// a \ " b rather than the three a " b.
//
// Note that keepWildEscape leaves `\*` untouched for a wild pattern: decoding it
// to a bare asterisk would turn a LITERAL asterisk into a wildcard, so the glob
// matcher is the one responsible for reading `\*` as a literal.
func unescapeECLString(s string, keepWildEscape bool) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 >= len(s) {
			b.WriteByte(s[i])
			continue
		}
		switch s[i+1] {
		case '"', '\\':
			b.WriteByte(s[i+1])
			i++
		case '*':
			if keepWildEscape {
				b.WriteString(`\*`)
			} else {
				b.WriteByte('*')
			}
			i++
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// stripWrappingQuotes removes a single layer of surrounding quotes if present and
// trims the whitespace that commonly surrounds the quoted content.
func stripWrappingQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}

func (v *astBuilder) buildTypeFilter(ctx grammar.ITypefilterContext) *ast.TypeFilter {
	concrete, ok := ctx.(*grammar.TypefilterContext)
	if !ok {
		return nil
	}
	if tid := concrete.Typeidfilter(); tid != nil {
		tc, ok := tid.(*grammar.TypeidfilterContext)
		if !ok {
			return nil
		}
		op := "="
		if tc.Booleancomparisonoperator() != nil {
			op = v.extractComparisonOp(tc.Booleancomparisonoperator().GetText())
		}
		tf := &ast.TypeFilter{Op: op}
		if sc := tc.Subexpressionconstraint(); sc != nil {
			if e := v.visitExpr(sc); e != nil {
				tf.Types = []ast.Expression{e}
			}
		} else if crs := tc.Eclconceptreferenceset(); crs != nil {
			if crsCtx, ok := crs.(*grammar.EclconceptreferencesetContext); ok {
				for _, cr := range crsCtx.AllEclconceptreference() {
					if e := v.visitExpr(cr); e != nil {
						tf.Types = append(tf.Types, e)
					}
				}
			}
		}
		return tf
	}
	if ttf := concrete.Typetokenfilter(); ttf != nil {
		tc, ok := ttf.(*grammar.TypetokenfilterContext)
		if !ok {
			return nil
		}
		op := "="
		if tc.Booleancomparisonoperator() != nil {
			op = v.extractComparisonOp(tc.Booleancomparisonoperator().GetText())
		}
		tf := &ast.TypeFilter{Op: op}
		// Map tokens to SCTIDs: syn→900000000000013009, fsn→900000000000003001,
		// def→900000000000550004. We model these as ConceptRef so consumers
		// can dispatch on ID.
		addToken := func(tok grammar.ITypetokenContext) {
			if tok == nil {
				return
			}
			tt, ok := tok.(*grammar.TypetokenContext)
			if !ok {
				return
			}
			var id string
			switch {
			case tt.Synonym() != nil:
				id = "900000000000013009"
			case tt.Fullyspecifiedname() != nil:
				id = "900000000000003001"
			case tt.Definition() != nil:
				id = "900000000000550004"
			}
			if id != "" {
				tf.Types = append(tf.Types, &ast.ConceptRef{ID: id})
			}
		}
		if single := tc.Typetoken(); single != nil {
			addToken(single)
		}
		if set := tc.Typetokenset(); set != nil {
			if setCtx, ok := set.(*grammar.TypetokensetContext); ok {
				for _, tok := range setCtx.AllTypetoken() {
					addToken(tok)
				}
			}
		}
		return tf
	}
	return nil
}

func (v *astBuilder) buildLanguageFilter(ctx grammar.ILanguagefilterContext) *ast.LanguageFilter {
	concrete, ok := ctx.(*grammar.LanguagefilterContext)
	if !ok {
		return nil
	}
	op := "="
	if concrete.Booleancomparisonoperator() != nil {
		op = v.extractComparisonOp(concrete.Booleancomparisonoperator().GetText())
	}
	lf := &ast.LanguageFilter{Op: op}
	if lc := concrete.Languagecode(); lc != nil {
		lf.Languages = append(lf.Languages, lc.GetText())
	}
	if lcs := concrete.Languagecodeset(); lcs != nil {
		if lcsCtx, ok := lcs.(*grammar.LanguagecodesetContext); ok {
			for _, code := range lcsCtx.AllLanguagecode() {
				lf.Languages = append(lf.Languages, code.GetText())
			}
		}
	}
	return lf
}

// buildDialectFilter builds an *ast.DialectFilter from a dialectfilter context.
//
// The grammar offers two forms:
//
//	dialectidfilter    : dialectId = <SCTID> | (set of SCTIDs)
//	dialectaliasfilter : dialect   = en-gb   | (set of aliases)
//
// Only the SCTID form is populated here. An alias like "en-gb" has to be mapped
// to the SCTID of a language reference set, and that mapping is terminology data:
// only the two international English aliases are universal, while national
// dialects use namespace-specific refset IDs. Inventing a partial table inside
// the parser would resolve some expressions and silently mis-resolve others, so
// the alias form is left with an empty Dialects slice and the evaluator reports
// it through ErrUnsupportedFeature.
//
// This node used to be emitted with Dialects always nil and Op forced to "=",
// which made every dialect expression evaluate to the empty set without a word.
func (v *astBuilder) buildDialectFilter(ctx grammar.IDialectfilterContext) *ast.DialectFilter {
	concrete, ok := ctx.(*grammar.DialectfilterContext)
	if !ok {
		return nil
	}

	df := &ast.DialectFilter{Op: "="}

	// A filter-level acceptability applies to every entry that declares none of
	// its own.
	var acceptability []ast.Expression
	if as := concrete.Acceptabilityset(); as != nil {
		acceptability = v.buildAcceptabilities(as)
	}

	if idf, ok := concrete.Dialectidfilter().(*grammar.DialectidfilterContext); ok && idf != nil {
		if op := idf.Booleancomparisonoperator(); op != nil {
			df.Op = v.extractComparisonOp(op.GetText())
		}
		if sub := idf.Subexpressionconstraint(); sub != nil {
			if expr := v.visitExpr(sub); expr != nil {
				entry := ast.DialectEntry{Dialect: expr, Acceptabilities: acceptability}
				if len(acceptability) > 0 {
					entry.Acceptability = acceptability[0] //nolint:staticcheck // deprecated field kept populated for v1 readers
				}
				df.Dialects = append(df.Dialects, entry)
			}
		}
		if set, ok := idf.Dialectidset().(*grammar.DialectidsetContext); ok && set != nil {
			df.Dialects = append(df.Dialects, v.dialectEntriesOf(set, acceptability)...)
		}
		return df
	}

	// Alias form: record the operator so the error message is accurate, but
	// leave Dialects empty — see the comment above.
	if af, ok := concrete.Dialectaliasfilter().(*grammar.DialectaliasfilterContext); ok && af != nil {
		if op := af.Booleancomparisonoperator(); op != nil {
			df.Op = v.extractComparisonOp(op.GetText())
		}
	}
	return df
}

// dialectEntriesOf pairs each dialect of a dialectidset with the acceptability
// that follows it, walking the children IN ORDER.
//
// The grammar makes acceptability optional per entry:
//
//	dialectidset : LEFT_PAREN ws eclconceptreference (ws acceptabilityset)?
//	               (mws eclconceptreference (ws acceptabilityset)? )* ws RIGHT_PAREN
//
// so ANTLR's flat AllEclconceptreference() and AllAcceptabilityset() lists cannot
// be zipped by index. Doing that attached the first acceptability to the first
// dialect regardless of where it appeared: `(A B (X))` gave X to A and left B
// bare, and `(A (X) B)` produced the identical AST.
func (v *astBuilder) dialectEntriesOf(set *grammar.DialectidsetContext, fallback []ast.Expression) []ast.DialectEntry {
	var (
		entries []ast.DialectEntry
		pending *ast.DialectEntry
	)
	flush := func() {
		if pending == nil {
			return
		}
		if len(pending.Acceptabilities) == 0 {
			pending.Acceptabilities = fallback
		}
		if len(pending.Acceptabilities) > 0 {
			pending.Acceptability = pending.Acceptabilities[0] //nolint:staticcheck // deprecated field kept populated for v1 readers
		}
		entries = append(entries, *pending)
		pending = nil
	}

	for _, child := range set.GetChildren() {
		switch c := child.(type) {
		case grammar.IEclconceptreferenceContext:
			flush()
			if expr := v.visitExpr(c); expr != nil {
				pending = &ast.DialectEntry{Dialect: expr}
			}
		case grammar.IAcceptabilitysetContext:
			if pending != nil {
				pending.Acceptabilities = v.buildAcceptabilities(c)
			}
		}
	}
	flush()
	return entries
}

// buildAcceptabilities renders an acceptabilityset as concept references, with
// any-of semantics.
//
// It used to return on the first reference, so `(preferred acceptable)` silently
// became `preferred` and narrowed the result.
func (v *astBuilder) buildAcceptabilities(as grammar.IAcceptabilitysetContext) []ast.Expression {
	concrete, ok := as.(*grammar.AcceptabilitysetContext)
	if !ok {
		return nil
	}
	var out []ast.Expression
	if crs, ok := concrete.Acceptabilityconceptreferenceset().(*grammar.AcceptabilityconceptreferencesetContext); ok && crs != nil {
		for _, ref := range crs.AllEclconceptreference() {
			if expr := v.visitExpr(ref); expr != nil {
				out = append(out, expr)
			}
		}
	}
	// The token form. Handling only the SCTID set above dropped `(prefer)` and
	// `(accept)` entirely, and an empty AcceptabilityIDs means "any acceptability"
	// to the provider — so the query silently widened instead of narrowing.
	if ts, ok := concrete.Acceptabilitytokenset().(*grammar.AcceptabilitytokensetContext); ok && ts != nil {
		for _, tok := range ts.AllAcceptabilitytoken() {
			if id := acceptabilityTokenID(tok); id != "" {
				out = append(out, &ast.ConceptRef{ID: id})
			}
		}
	}
	return out
}

// acceptabilityTokenID maps an acceptability token to its well-known SCTID.
func acceptabilityTokenID(tok grammar.IAcceptabilitytokenContext) string {
	concrete, ok := tok.(*grammar.AcceptabilitytokenContext)
	if !ok {
		return ""
	}
	switch {
	case concrete.Preferred() != nil:
		return "900000000000548007" // preferred
	case concrete.Acceptable() != nil:
		return "900000000000549004" // acceptable
	}
	return ""
}

// buildDescriptionIDFilter builds an *ast.DescriptionIDFilter.
//
// This branch used to be skipped without emitting anything, and when the filter
// was a constraint's only clause the ast.Filtered node was never created at all:
// `< 404684003 {{ D id = 123456789012 }}` produced an AST identical to
// `< 404684003` and the query silently returned every descendant.
func (v *astBuilder) buildDescriptionIDFilter(ctx grammar.IDescriptionidfilterContext) *ast.DescriptionIDFilter {
	concrete, ok := ctx.(*grammar.DescriptionidfilterContext)
	if !ok {
		return nil
	}
	op := "="
	if c := concrete.Idcomparisonoperator(); c != nil {
		op = v.extractComparisonOp(c.GetText())
	}
	f := &ast.DescriptionIDFilter{Op: op}
	if id := concrete.Descriptionid(); id != nil {
		f.IDs = append(f.IDs, strings.TrimSpace(id.GetText()))
	} else if set, ok := concrete.Descriptionidset().(*grammar.DescriptionidsetContext); ok && set != nil {
		for _, id := range set.AllDescriptionid() {
			f.IDs = append(f.IDs, strings.TrimSpace(id.GetText()))
		}
	}
	if len(f.IDs) == 0 {
		return nil
	}
	return f
}

func (v *astBuilder) buildActiveFilter(ctx grammar.IActivefilterContext) *ast.ActiveFilter {
	concrete, ok := ctx.(*grammar.ActivefilterContext)
	if !ok {
		return nil
	}
	active := true
	if av := concrete.Activevalue(); av != nil {
		if avCtx, ok := av.(*grammar.ActivevalueContext); ok {
			switch {
			case avCtx.Activetruevalue() != nil:
				active = true
			case avCtx.Activefalsevalue() != nil:
				active = false
			case avCtx.Wildcard() != nil:
				// Wildcard means "either" — represented as true here with a
				// note; evaluator may treat wildcard specially in the future.
				active = true
			}
		}
	}
	// The grammar allows "!=" but the AST ActiveFilter only carries a Value.
	// If operator is "!=", we invert the recorded boolean so downstream code
	// still gets the correct semantics for the common case (active = false).
	if concrete.Booleancomparisonoperator() != nil {
		if v.extractComparisonOp(concrete.Booleancomparisonoperator().GetText()) == "!=" {
			active = !active
		}
	}
	return &ast.ActiveFilter{Value: active}
}

func (v *astBuilder) buildModuleFilter(ctx grammar.IModulefilterContext) *ast.ModuleFilter {
	concrete, ok := ctx.(*grammar.ModulefilterContext)
	if !ok {
		return nil
	}
	op := "="
	if concrete.Booleancomparisonoperator() != nil {
		op = v.extractComparisonOp(concrete.Booleancomparisonoperator().GetText())
	}
	mf := &ast.ModuleFilter{Op: op}
	if sc := concrete.Subexpressionconstraint(); sc != nil {
		if expr := v.visitExpr(sc); expr != nil {
			mf.Modules = append(mf.Modules, expr)
		}
	} else if crs, ok := concrete.Eclconceptreferenceset().(*grammar.EclconceptreferencesetContext); ok && crs != nil {
		// The set has any-of semantics. Only the first reference used to be
		// kept, so `moduleId = (A B)` silently became `moduleId = A`.
		for _, ref := range crs.AllEclconceptreference() {
			if expr := v.visitExpr(ref); expr != nil {
				mf.Modules = append(mf.Modules, expr)
			}
		}
	}
	if len(mf.Modules) == 0 {
		return nil
	}
	// Deprecated scalar, kept populated for readers written against v1.1.
	mf.Module = mf.Modules[0] //nolint:staticcheck // deprecated field kept populated for v1 readers
	return mf
}

func (v *astBuilder) buildEffectiveTimeFilter(ctx grammar.IEffectivetimefilterContext) *ast.EffectiveTimeFilter {
	concrete, ok := ctx.(*grammar.EffectivetimefilterContext)
	if !ok {
		return nil
	}
	op := "="
	if concrete.Timecomparisonoperator() != nil {
		op = concrete.Timecomparisonoperator().GetText()
	}
	ef := &ast.EffectiveTimeFilter{Op: op}
	if tv := concrete.Timevalue(); tv != nil {
		ef.Values = append(ef.Values, stripWrappingQuotes(tv.GetText()))
	} else if tvs, ok := concrete.Timevalueset().(*grammar.TimevaluesetContext); ok && tvs != nil {
		// GetText() of the whole set used to be stored as ONE value, parentheses
		// and inner quotes included, which no provider could match.
		for _, tv := range tvs.AllTimevalue() {
			ef.Values = append(ef.Values, stripWrappingQuotes(tv.GetText()))
		}
	}
	if len(ef.Values) == 0 {
		return nil
	}
	// Deprecated scalar, kept populated for readers written against v1.1.
	ef.Value = ef.Values[0] //nolint:staticcheck // deprecated field kept populated for v1 readers
	return ef
}

func (v *astBuilder) buildDefinitionStatusFilter(ctx grammar.IDefinitionstatusfilterContext) *ast.DefinitionStatusFilter {
	concrete, ok := ctx.(*grammar.DefinitionstatusfilterContext)
	if !ok {
		return nil
	}
	if idf := concrete.Definitionstatusidfilter(); idf != nil {
		if idCtx, ok := idf.(*grammar.DefinitionstatusidfilterContext); ok {
			op := "="
			if idCtx.Booleancomparisonoperator() != nil {
				op = v.extractComparisonOp(idCtx.Booleancomparisonoperator().GetText())
			}
			f := &ast.DefinitionStatusFilter{Op: op}
			if sc := idCtx.Subexpressionconstraint(); sc != nil {
				if expr := v.visitExpr(sc); expr != nil {
					f.Values = append(f.Values, expr)
				}
			} else if crs, ok := idCtx.Eclconceptreferenceset().(*grammar.EclconceptreferencesetContext); ok && crs != nil {
				// Any-of. Only the first reference used to survive.
				for _, ref := range crs.AllEclconceptreference() {
					if expr := v.visitExpr(ref); expr != nil {
						f.Values = append(f.Values, expr)
					}
				}
			}
			if len(f.Values) == 0 {
				return nil
			}
			f.Value = f.Values[0] //nolint:staticcheck // deprecated field kept populated for v1 readers
			return f
		}
	}
	if tf := concrete.Definitionstatustokenfilter(); tf != nil {
		if tCtx, ok := tf.(*grammar.DefinitionstatustokenfilterContext); ok {
			op := "="
			if tCtx.Booleancomparisonoperator() != nil {
				op = v.extractComparisonOp(tCtx.Booleancomparisonoperator().GetText())
			}
			f := &ast.DefinitionStatusFilter{Op: op}
			addToken := func(tok grammar.IDefinitionstatustokenContext) {
				dt, ok := tok.(*grammar.DefinitionstatustokenContext)
				if !ok {
					return
				}
				var id string
				switch {
				case dt.Primitivetoken() != nil:
					id = "900000000000074008" // primitive
				case dt.Definedtoken() != nil:
					id = "900000000000073002" // defined
				}
				if id != "" {
					f.Values = append(f.Values, &ast.ConceptRef{ID: id})
				}
			}
			if single := tCtx.Definitionstatustoken(); single != nil {
				addToken(single)
			} else if set, ok := tCtx.Definitionstatustokenset().(*grammar.DefinitionstatustokensetContext); ok && set != nil {
				// `definitionStatus = (primitive defined)` kept only the first.
				for _, tok := range set.AllDefinitionstatustoken() {
					addToken(tok)
				}
			}
			if len(f.Values) == 0 {
				return nil
			}
			f.Value = f.Values[0] //nolint:staticcheck // deprecated field kept populated for v1 readers
			return f
		}
	}
	return nil
}

func (v *astBuilder) buildMemberFieldFilter(ctx grammar.IMemberfieldfilterContext) *ast.MemberFieldFilter {
	concrete, ok := ctx.(*grammar.MemberfieldfilterContext)
	if !ok {
		return nil
	}
	f := &ast.MemberFieldFilter{}
	if fn := concrete.Refsetfieldname(); fn != nil {
		f.FieldName = fn.GetText()
	}
	switch {
	case concrete.Expressioncomparisonoperator() != nil:
		f.Op = v.extractComparisonOp(concrete.Expressioncomparisonoperator().GetText())
		if sc := concrete.Subexpressionconstraint(); sc != nil {
			f.Value = v.visitExpr(sc)
		}
	case concrete.Numericcomparisonoperator() != nil:
		f.Op = v.extractComparisonOp(concrete.Numericcomparisonoperator().GetText())
		if nv := concrete.Numericvalue(); nv != nil {
			f.Value = v.visitNumericValue(nv)
		}
	case concrete.Stringcomparisonoperator() != nil:
		f.Op = v.extractComparisonOp(concrete.Stringcomparisonoperator().GetText())
		if ts := concrete.Typedsearchterm(); ts != nil {
			term, _ := extractTypedSearchTerm(ts)
			f.Value = &ast.StringValue{Value: term}
		}
	case concrete.Booleancomparisonoperator() != nil:
		f.Op = v.extractComparisonOp(concrete.Booleancomparisonoperator().GetText())
		if bv := concrete.Booleanvalue(); bv != nil {
			f.Value = v.visitBooleanValue(bv)
		}
	case concrete.Timecomparisonoperator() != nil:
		f.Op = concrete.Timecomparisonoperator().GetText()
		if tv := concrete.Timevalue(); tv != nil {
			f.Value = &ast.StringValue{Value: stripWrappingQuotes(tv.GetText())}
		}
	}
	return f
}

func (v *astBuilder) VisitAltidentifier(ctx *grammar.AltidentifierContext) any {
	alt := &ast.AltIdentifier{}
	if ctx.Altidentifierschemealias() != nil {
		alt.Scheme = ctx.Altidentifierschemealias().GetText()
	}
	if ctx.Altidentifiercodewithinquotes() != nil {
		alt.Code = ctx.Altidentifiercodewithinquotes().GetText()
	} else if ctx.Altidentifiercodewithoutquotes() != nil {
		alt.Code = ctx.Altidentifiercodewithoutquotes().GetText()
	}
	if ctx.Term() != nil {
		alt.Term = strings.TrimSpace(ctx.Term().GetText())
	}
	return alt
}

// ---------------------------------------------------------------------------
// Eclattributename — delegates to subexpressionconstraint
// ---------------------------------------------------------------------------.

func (v *astBuilder) VisitEclattributename(ctx *grammar.EclattributenameContext) any {
	if ctx.Subexpressionconstraint() != nil {
		return v.Visit(ctx.Subexpressionconstraint())
	}
	return nil
}

// ---------------------------------------------------------------------------
// Refinement
// ---------------------------------------------------------------------------.

func (v *astBuilder) visitRefinement(ctx grammar.IEclrefinementContext) *ast.Refinement {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.EclrefinementContext)
	if !ok {
		return nil
	}
	ref := &ast.Refinement{}

	// Parse the first subrefinement
	if concrete.Subrefinement() != nil {
		v.collectSubrefinement(concrete.Subrefinement(), ref)
	}

	// Check for conjunction/disjunction refinement sets
	if concrete.Conjunctionrefinementset() != nil {
		conj := concrete.Conjunctionrefinementset().(*grammar.ConjunctionrefinementsetContext)
		for _, sub := range conj.AllSubrefinement() {
			subRef := &ast.Refinement{}
			v.collectSubrefinement(sub, subRef)
			ref.Conjunction = append(ref.Conjunction, subRef)
		}
	}
	if concrete.Disjunctionrefinementset() != nil {
		disj := concrete.Disjunctionrefinementset().(*grammar.DisjunctionrefinementsetContext)
		for _, sub := range disj.AllSubrefinement() {
			subRef := &ast.Refinement{}
			v.collectSubrefinement(sub, subRef)
			ref.Disjunction = append(ref.Disjunction, subRef)
		}
	}

	return ref
}

func (v *astBuilder) VisitEclrefinement(ctx *grammar.EclrefinementContext) any {
	return v.visitRefinement(ctx)
}

func (v *astBuilder) collectSubrefinement(ctx grammar.ISubrefinementContext, ref *ast.Refinement) {
	if ctx == nil {
		return
	}
	concrete, ok := ctx.(*grammar.SubrefinementContext)
	if !ok {
		return
	}
	switch {
	case concrete.Eclattributegroup() != nil:
		grp := v.visitAttributeGroup(concrete.Eclattributegroup())
		if grp != nil {
			ref.Groups = append(ref.Groups, grp)
		}
	case concrete.Eclattributeset() != nil:
		set := v.collectAttributeSet(concrete.Eclattributeset())
		ref.AttrSet = mergeAttrSet(ref.AttrSet, set)
		ref.Ungrouped = append(ref.Ungrouped, flattenAttrSet(set)...) //nolint:staticcheck // deprecated field kept populated for v1 readers
	case concrete.Eclrefinement() != nil:
		// A parenthesised refinement is a SCOPE: keep it as its own node.
		//
		// This used to merge inner.Groups/Ungrouped/Conjunction/Disjunction into
		// the parent, which destroyed the parentheses: the parent then held both
		// a Conjunction and a Disjunction with no record of which operands
		// belonged to the inner scope, so `({A} OR {B}) , C` became
		// indistinguishable from `{A} , ({B} OR C)`.
		//
		// It goes in its own field rather than in Conjunction. Reusing
		// Conjunction conflated "the first sub-refinement was parenthesised" with
		// "there is a conjunction set", which made the legitimate shape
		// `(<refinement>) OR <clause>` look like a node holding both a
		// conjunction and a disjunction.
		if inner := v.visitRefinement(concrete.Eclrefinement()); inner != nil {
			ref.Nested = inner
		}
	}
}

// mergeAttrSet combines two attribute trees with AND, which is the operator
// between successive sub-refinements of the same refinement.
func mergeAttrSet(a, b *ast.AttributeSet) *ast.AttributeSet {
	switch {
	case a == nil:
		return b
	case b == nil:
		return a
	default:
		return &ast.AttributeSet{Op: ast.AttrSetAnd, Items: []*ast.AttributeSet{a, b}}
	}
}

// ---------------------------------------------------------------------------
// Attribute group
// ---------------------------------------------------------------------------.

func (v *astBuilder) visitAttributeGroup(ctx grammar.IEclattributegroupContext) *ast.AttributeGroup {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.EclattributegroupContext)
	if !ok {
		return nil
	}
	grp := &ast.AttributeGroup{}

	if concrete.Cardinality() != nil {
		grp.Cardinality = v.visitCardinality(concrete.Cardinality())
	}

	if concrete.Eclattributeset() != nil {
		grp.AttrSet = v.collectAttributeSet(concrete.Eclattributeset())
		grp.Attrs = flattenAttrSet(grp.AttrSet) //nolint:staticcheck // deprecated field kept populated for v1 readers
	}

	return grp
}

func (v *astBuilder) VisitEclattributegroup(ctx *grammar.EclattributegroupContext) any {
	return v.visitAttributeGroup(ctx)
}

// ---------------------------------------------------------------------------
// Attribute set — collects all attributes (possibly with conjunction/disjunction)
// ---------------------------------------------------------------------------.

// collectAttributeSet builds the boolean tree of an eclattributeset, preserving
// whether the clauses were joined by AND (",") or OR.
//
// The grammar rule is
//
//	eclattributeset : subattributeset ws (conjunctionattributeset | disjunctionattributeset)?;
//
// so a level is either a conjunction or a disjunction, never both, and the
// operator is unambiguous. Flattening both into one slice — which is what
// collectAttributes below still does for the deprecated field — made `a = x OR
// b = y` and `a = x, b = y` produce byte-identical ASTs, and the evaluator then
// intersected the disjuncts.
func (v *astBuilder) collectAttributeSet(ctx grammar.IEclattributesetContext) *ast.AttributeSet {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.EclattributesetContext)
	if !ok {
		return nil
	}

	var items []*ast.AttributeSet
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

	switch len(items) {
	case 0:
		return nil
	case 1:
		return items[0] // a lone leaf needs no boolean wrapper
	default:
		return &ast.AttributeSet{Op: op, Items: items}
	}
}

// collectSubAttributeSet builds the tree of a subattributeset: either a single
// attribute leaf, or recursively the parenthesised eclattributeset. Note it can
// never return a group — `subattributeset` does not admit eclattributegroup.
func (v *astBuilder) collectSubAttributeSet(ctx grammar.ISubattributesetContext) *ast.AttributeSet {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.SubattributesetContext)
	if !ok {
		return nil
	}
	if concrete.Eclattribute() != nil {
		if attr := v.visitAttribute(concrete.Eclattribute()); attr != nil {
			return &ast.AttributeSet{Attr: attr}
		}
		return nil
	}
	// Parenthesised nested set: recurse so the scope survives.
	return v.collectAttributeSet(concrete.Eclattributeset())
}

// collectAttributes flattens an attribute set into a slice, losing the AND/OR
// distinction.
//
// Deprecated: it exists only to keep ast.Refinement.Ungrouped and
// ast.AttributeGroup.Attrs populated for readers written against v1.1. Use
// collectAttributeSet.
func (v *astBuilder) collectAttributes(ctx grammar.IEclattributesetContext) []*ast.Attribute {
	return flattenAttrSet(v.collectAttributeSet(ctx))
}

// flattenAttrSet collects every attribute leaf of a set, in order.
func flattenAttrSet(set *ast.AttributeSet) []*ast.Attribute {
	if set == nil {
		return nil
	}
	if set.Attr != nil {
		return []*ast.Attribute{set.Attr}
	}
	var out []*ast.Attribute
	for _, item := range set.Items {
		out = append(out, flattenAttrSet(item)...)
	}
	return out
}

func (v *astBuilder) VisitEclattributeset(ctx *grammar.EclattributesetContext) any {
	return v.collectAttributes(ctx)
}

// ---------------------------------------------------------------------------
// Single attribute
// ---------------------------------------------------------------------------.

func (v *astBuilder) visitAttribute(ctx grammar.IEclattributeContext) *ast.Attribute {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.EclattributeContext)
	if !ok {
		return nil
	}
	attr := &ast.Attribute{}

	// Cardinality
	if concrete.Cardinality() != nil {
		attr.Cardinality = v.visitCardinality(concrete.Cardinality())
	}

	// Reverse flag
	if concrete.Reverseflag() != nil {
		attr.Reverse = true
	}

	// Attribute name
	if concrete.Eclattributename() != nil {
		attr.Name = v.visitExpr(concrete.Eclattributename())
	}

	// Expression comparison: = or != with subexpressionconstraint
	if concrete.Expressioncomparisonoperator() != nil {
		attr.Op = v.extractComparisonOp(concrete.Expressioncomparisonoperator().GetText())
		if concrete.Subexpressionconstraint() != nil {
			attr.Value = v.visitExpr(concrete.Subexpressionconstraint())
		}
		return attr
	}

	// Numeric comparison
	if concrete.Numericcomparisonoperator() != nil {
		attr.Op = v.extractComparisonOp(concrete.Numericcomparisonoperator().GetText())
		if concrete.Numericvalue() != nil {
			attr.Value = v.visitNumericValue(concrete.Numericvalue())
		}
		return attr
	}

	// String comparison
	if concrete.Stringcomparisonoperator() != nil {
		attr.Op = v.extractComparisonOp(concrete.Stringcomparisonoperator().GetText())
		if concrete.Concretestring() != nil {
			attr.Value = v.visitConcreteString(concrete.Concretestring())
		}
		return attr
	}

	// Boolean comparison
	if concrete.Booleancomparisonoperator() != nil {
		attr.Op = v.extractComparisonOp(concrete.Booleancomparisonoperator().GetText())
		if concrete.Booleanvalue() != nil {
			attr.Value = v.visitBooleanValue(concrete.Booleanvalue())
		}
		return attr
	}

	return attr
}

func (v *astBuilder) VisitEclattribute(ctx *grammar.EclattributeContext) any {
	return v.visitAttribute(ctx)
}

// ---------------------------------------------------------------------------
// Cardinality
// ---------------------------------------------------------------------------.

func (v *astBuilder) visitCardinality(ctx grammar.ICardinalityContext) *ast.Cardinality {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.CardinalityContext)
	if !ok {
		return nil
	}
	card := &ast.Cardinality{}
	if concrete.Minvalue() != nil {
		card.Min, _ = strconv.Atoi(concrete.Minvalue().GetText())
	}
	if concrete.Maxvalue() != nil {
		maxCtx := concrete.Maxvalue().(*grammar.MaxvalueContext)
		if maxCtx.Many() != nil {
			card.Max = -1
		} else {
			card.Max, _ = strconv.Atoi(maxCtx.GetText())
		}
	}
	return card
}

func (v *astBuilder) VisitCardinality(ctx *grammar.CardinalityContext) any {
	return v.visitCardinality(ctx)
}

// ---------------------------------------------------------------------------
// Concrete values
// ---------------------------------------------------------------------------.

func (v *astBuilder) visitNumericValue(ctx grammar.INumericvalueContext) ast.Expression {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.NumericvalueContext)
	if !ok {
		return nil
	}
	text := concrete.GetText()
	if concrete.Decimalvalue() != nil {
		val, _ := strconv.ParseFloat(text, 64)
		return &ast.DecimalValue{Value: val}
	}
	val, _ := strconv.Atoi(text)
	return &ast.IntegerValue{Value: val}
}

func (v *astBuilder) visitConcreteString(ctx grammar.IConcretestringContext) ast.Expression {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.ConcretestringContext)
	if !ok {
		return nil
	}
	// The text includes surrounding quotes, strip them
	text := concrete.GetText()
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		text = text[1 : len(text)-1]
	}
	return &ast.StringValue{Value: text}
}

func (v *astBuilder) visitBooleanValue(ctx grammar.IBooleanvalueContext) ast.Expression {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.BooleanvalueContext)
	if !ok {
		return nil
	}
	if concrete.True_1() != nil {
		return &ast.BooleanValue{Value: true}
	}
	return &ast.BooleanValue{Value: false}
}

// ---------------------------------------------------------------------------
// History supplement
// ---------------------------------------------------------------------------.

func (v *astBuilder) visitHistorySupplement(ctx grammar.IHistorysupplementContext, operand ast.Expression) ast.Expression {
	if ctx == nil {
		return operand
	}
	concrete, ok := ctx.(*grammar.HistorysupplementContext)
	if !ok {
		return operand
	}
	hs := &ast.HistorySupplement{Operand: operand}

	if concrete.Historyprofilesuffix() != nil {
		suffix := concrete.Historyprofilesuffix().(*grammar.HistoryprofilesuffixContext)
		switch {
		case suffix.Historyminimumsuffix() != nil:
			hs.Profile = "HISTORY-MIN"
		case suffix.Historymoderatesuffix() != nil:
			hs.Profile = "HISTORY-MOD"
		case suffix.Historymaximumsuffix() != nil:
			hs.Profile = "HISTORY-MAX"
		}
	}

	return hs
}

func (v *astBuilder) VisitHistorysupplement(ctx *grammar.HistorysupplementContext) any {
	// This is typically called from visitHistorySupplement with an operand.
	// If called standalone, wrap with nil operand.
	return v.visitHistorySupplement(ctx, nil)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------.

// extractComparisonOp normalises the raw token text to a comparison operator string.
func (v *astBuilder) extractComparisonOp(text string) string {
	switch text {
	case "=":
		return "="
	case "!=":
		return "!="
	case "<":
		return "<"
	case "<=":
		return "<="
	case ">":
		return ">"
	case ">=":
		return ">="
	default:
		return text
	}
}
