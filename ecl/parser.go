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
	lexer := grammar.NewECLLexer(antlr.NewInputStream(input))
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := grammar.NewECLParser(stream)

	// Error handling
	errListener := &eclErrorListener{}
	parser.RemoveErrorListeners()
	parser.AddErrorListener(errListener)

	tree := parser.Expressionconstraint()
	if errListener.err != nil {
		return nil, errListener.err
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

type eclErrorListener struct {
	antlr.DefaultErrorListener
	err error
}

func (l *eclErrorListener) SyntaxError(_ antlr.Recognizer, _ any, line, column int, msg string, _ antlr.RecognitionException) {
	l.err = fmt.Errorf("syntax error at %d:%d: %s", line, column, msg)
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
		if c := d.Termfilter(); c != nil {
			if f := v.buildTermFilter(c); f != nil {
				out = append(out, f)
			}
			continue
		}
		if c := d.Typefilter(); c != nil {
			if f := v.buildTypeFilter(c); f != nil {
				out = append(out, f)
			}
			continue
		}
		if c := d.Languagefilter(); c != nil {
			if f := v.buildLanguageFilter(c); f != nil {
				out = append(out, f)
			}
			continue
		}
		if c := d.Dialectfilter(); c != nil {
			// Preserve presence — evaluator will flag as not-yet-implemented.
			out = append(out, &ast.DialectFilter{Op: "=", Dialects: nil})
			_ = c
			continue
		}
		if c := d.Activefilter(); c != nil {
			if f := v.buildActiveFilter(c); f != nil {
				out = append(out, f)
			}
			continue
		}
		if c := d.Modulefilter(); c != nil {
			if f := v.buildModuleFilter(c); f != nil {
				out = append(out, f)
			}
			continue
		}
		if c := d.Effectivetimefilter(); c != nil {
			if f := v.buildEffectiveTimeFilter(c); f != nil {
				out = append(out, f)
			}
			continue
		}
		// descriptionidfilter not modeled — skip.
	}
	return out
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
	var (
		term      string
		matchType = "match"
	)
	if c := concrete.Typedsearchterm(); c != nil {
		term, matchType = extractTypedSearchTerm(c)
	} else if c := concrete.Typedsearchtermset(); c != nil {
		// Take the first term in the set; multi-term sets are treated as the
		// first term for now (evaluator documents the simplification).
		term = stripWrappingQuotes(c.GetText())
	}
	return &ast.TermFilter{Op: op, Term: term, MatchType: matchType}
}

// extractTypedSearchTerm returns the raw search term and its match-type
// modifier ("match" or "wild") from a typedsearchterm context. Falls back to
// "match" when the expected sub-rule shape is not present.
func extractTypedSearchTerm(ctx grammar.ITypedsearchtermContext) (term, matchType string) {
	matchType = "match"
	concrete, ok := ctx.(*grammar.TypedsearchtermContext)
	if !ok {
		return stripWrappingQuotes(ctx.GetText()), matchType
	}
	if s := concrete.Matchsearchtermset(); s != nil {
		return stripWrappingQuotes(s.GetText()), "match"
	}
	if s := concrete.Wildsearchtermset(); s != nil {
		return stripWrappingQuotes(s.GetText()), "wild"
	}
	return stripWrappingQuotes(concrete.GetText()), matchType
}

// stripWrappingQuotes removes a single layer of surrounding " quotes if present
// and trims whitespace that commonly surrounds the quoted content.
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
		mf.Module = v.visitExpr(sc)
	} else if crs := concrete.Eclconceptreferenceset(); crs != nil {
		// Pick the first ref from the set; multi-module sets represented as
		// a single module is a simplification (most ECL uses single module).
		if crsCtx, ok := crs.(*grammar.EclconceptreferencesetContext); ok {
			refs := crsCtx.AllEclconceptreference()
			if len(refs) > 0 {
				mf.Module = v.visitExpr(refs[0])
			}
		}
	}
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
	var value string
	if tv := concrete.Timevalue(); tv != nil {
		value = stripWrappingQuotes(tv.GetText())
	} else if tvs := concrete.Timevalueset(); tvs != nil {
		value = stripWrappingQuotes(tvs.GetText())
	}
	return &ast.EffectiveTimeFilter{Op: op, Value: value}
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
				f.Value = v.visitExpr(sc)
			} else if crs := idCtx.Eclconceptreferenceset(); crs != nil {
				if crsCtx, ok := crs.(*grammar.EclconceptreferencesetContext); ok {
					refs := crsCtx.AllEclconceptreference()
					if len(refs) > 0 {
						f.Value = v.visitExpr(refs[0])
					}
				}
			}
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
				if dt.Primitivetoken() != nil {
					id = "900000000000074008" // primitive
				} else if dt.Definedtoken() != nil {
					id = "900000000000073002" // defined
				}
				if id != "" {
					f.Value = &ast.ConceptRef{ID: id}
				}
			}
			if single := tCtx.Definitionstatustoken(); single != nil {
				addToken(single)
			} else if set := tCtx.Definitionstatustokenset(); set != nil {
				if setCtx, ok := set.(*grammar.DefinitionstatustokensetContext); ok {
					toks := setCtx.AllDefinitionstatustoken()
					if len(toks) > 0 {
						addToken(toks[0])
					}
				}
			}
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
		attrs := v.collectAttributes(concrete.Eclattributeset())
		ref.Ungrouped = append(ref.Ungrouped, attrs...)
	case concrete.Eclrefinement() != nil:
		// Nested refinement in parentheses
		inner := v.visitRefinement(concrete.Eclrefinement())
		if inner != nil {
			ref.Groups = append(ref.Groups, inner.Groups...)
			ref.Ungrouped = append(ref.Ungrouped, inner.Ungrouped...)
			ref.Conjunction = append(ref.Conjunction, inner.Conjunction...)
			ref.Disjunction = append(ref.Disjunction, inner.Disjunction...)
		}
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
		grp.Attrs = v.collectAttributes(concrete.Eclattributeset())
	}

	return grp
}

func (v *astBuilder) VisitEclattributegroup(ctx *grammar.EclattributegroupContext) any {
	return v.visitAttributeGroup(ctx)
}

// ---------------------------------------------------------------------------
// Attribute set — collects all attributes (possibly with conjunction/disjunction)
// ---------------------------------------------------------------------------.

func (v *astBuilder) collectAttributes(ctx grammar.IEclattributesetContext) []*ast.Attribute {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.EclattributesetContext)
	if !ok {
		return nil
	}
	var attrs []*ast.Attribute

	// First sub-attribute
	if concrete.Subattributeset() != nil {
		attrs = append(attrs, v.collectSubAttributes(concrete.Subattributeset())...)
	}

	// Conjunction attributes
	if concrete.Conjunctionattributeset() != nil {
		conj := concrete.Conjunctionattributeset().(*grammar.ConjunctionattributesetContext)
		for _, sub := range conj.AllSubattributeset() {
			attrs = append(attrs, v.collectSubAttributes(sub)...)
		}
	}

	// Disjunction attributes
	if concrete.Disjunctionattributeset() != nil {
		disj := concrete.Disjunctionattributeset().(*grammar.DisjunctionattributesetContext)
		for _, sub := range disj.AllSubattributeset() {
			attrs = append(attrs, v.collectSubAttributes(sub)...)
		}
	}

	return attrs
}

func (v *astBuilder) collectSubAttributes(ctx grammar.ISubattributesetContext) []*ast.Attribute {
	if ctx == nil {
		return nil
	}
	concrete, ok := ctx.(*grammar.SubattributesetContext)
	if !ok {
		return nil
	}
	if concrete.Eclattribute() != nil {
		attr := v.visitAttribute(concrete.Eclattribute())
		if attr != nil {
			return []*ast.Attribute{attr}
		}
	}
	if concrete.Eclattributeset() != nil {
		return v.collectAttributes(concrete.Eclattributeset())
	}
	return nil
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
