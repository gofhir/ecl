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
// ---------------------------------------------------------------------------

type eclErrorListener struct {
	antlr.DefaultErrorListener
	err error
}

func (l *eclErrorListener) SyntaxError(_ antlr.Recognizer, _ interface{}, line, column int, msg string, _ antlr.RecognitionException) {
	l.err = fmt.Errorf("syntax error at %d:%d: %s", line, column, msg)
}

// ---------------------------------------------------------------------------
// AST builder visitor
// ---------------------------------------------------------------------------

type astBuilder struct {
	grammar.BaseECLVisitor
}

// Visit dispatches to the correct typed Visit method via the Accept pattern.
func (v *astBuilder) Visit(tree antlr.ParseTree) interface{} {
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
// ---------------------------------------------------------------------------

func (v *astBuilder) VisitExpressionconstraint(ctx *grammar.ExpressionconstraintContext) interface{} {
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
// ---------------------------------------------------------------------------

func (v *astBuilder) VisitCompoundexpressionconstraint(ctx *grammar.CompoundexpressionconstraintContext) interface{} {
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

func (v *astBuilder) VisitConjunctionexpressionconstraint(ctx *grammar.ConjunctionexpressionconstraintContext) interface{} {
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

func (v *astBuilder) VisitDisjunctionexpressionconstraint(ctx *grammar.DisjunctionexpressionconstraintContext) interface{} {
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

func (v *astBuilder) VisitExclusionexpressionconstraint(ctx *grammar.ExclusionexpressionconstraintContext) interface{} {
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
// ---------------------------------------------------------------------------

func (v *astBuilder) VisitRefinedexpressionconstraint(ctx *grammar.RefinedexpressionconstraintContext) interface{} {
	focus := v.visitExpr(ctx.Subexpressionconstraint())
	refinement := v.visitRefinement(ctx.Eclrefinement())
	return &ast.Refined{
		Focus:      focus,
		Refinement: refinement,
	}
}

// ---------------------------------------------------------------------------
// Dotted expression
// ---------------------------------------------------------------------------

func (v *astBuilder) VisitDottedexpressionconstraint(ctx *grammar.DottedexpressionconstraintContext) interface{} {
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

func (v *astBuilder) VisitDottedexpressionattribute(ctx *grammar.DottedexpressionattributeContext) interface{} {
	if ctx.Eclattributename() != nil {
		return v.Visit(ctx.Eclattributename())
	}
	return nil
}

// ---------------------------------------------------------------------------
// Sub-expression constraint (the workhorse)
// ---------------------------------------------------------------------------

func (v *astBuilder) VisitSubexpressionconstraint(ctx *grammar.SubexpressionconstraintContext) interface{} {
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

	// 4. Apply description/concept filter constraints
	// (Not implemented in detail for tests — filters would wrap focusExpr in ast.Filtered)

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
// ---------------------------------------------------------------------------

func (v *astBuilder) VisitEclfocusconcept(ctx *grammar.EclfocusconceptContext) interface{} {
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

func (v *astBuilder) VisitEclconceptreference(ctx *grammar.EclconceptreferenceContext) interface{} {
	ref := &ast.ConceptRef{}
	if ctx.Conceptid() != nil {
		ref.ID = ctx.Conceptid().GetText()
	}
	if ctx.Term() != nil {
		ref.Term = strings.TrimSpace(ctx.Term().GetText())
	}
	return ref
}

func (v *astBuilder) VisitWildcard(_ *grammar.WildcardContext) interface{} {
	return &ast.Any{}
}

func (v *astBuilder) VisitAltidentifier(ctx *grammar.AltidentifierContext) interface{} {
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
// ---------------------------------------------------------------------------

func (v *astBuilder) VisitEclattributename(ctx *grammar.EclattributenameContext) interface{} {
	if ctx.Subexpressionconstraint() != nil {
		return v.Visit(ctx.Subexpressionconstraint())
	}
	return nil
}

// ---------------------------------------------------------------------------
// Refinement
// ---------------------------------------------------------------------------

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

func (v *astBuilder) VisitEclrefinement(ctx *grammar.EclrefinementContext) interface{} {
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
	if concrete.Eclattributegroup() != nil {
		grp := v.visitAttributeGroup(concrete.Eclattributegroup())
		if grp != nil {
			ref.Groups = append(ref.Groups, grp)
		}
	} else if concrete.Eclattributeset() != nil {
		attrs := v.collectAttributes(concrete.Eclattributeset())
		ref.Ungrouped = append(ref.Ungrouped, attrs...)
	} else if concrete.Eclrefinement() != nil {
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
// ---------------------------------------------------------------------------

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

func (v *astBuilder) VisitEclattributegroup(ctx *grammar.EclattributegroupContext) interface{} {
	return v.visitAttributeGroup(ctx)
}

// ---------------------------------------------------------------------------
// Attribute set — collects all attributes (possibly with conjunction/disjunction)
// ---------------------------------------------------------------------------

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

func (v *astBuilder) VisitEclattributeset(ctx *grammar.EclattributesetContext) interface{} {
	return v.collectAttributes(ctx)
}

// ---------------------------------------------------------------------------
// Single attribute
// ---------------------------------------------------------------------------

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

func (v *astBuilder) VisitEclattribute(ctx *grammar.EclattributeContext) interface{} {
	return v.visitAttribute(ctx)
}

// ---------------------------------------------------------------------------
// Cardinality
// ---------------------------------------------------------------------------

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

func (v *astBuilder) VisitCardinality(ctx *grammar.CardinalityContext) interface{} {
	return v.visitCardinality(ctx)
}

// ---------------------------------------------------------------------------
// Concrete values
// ---------------------------------------------------------------------------

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
// ---------------------------------------------------------------------------

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
		if suffix.Historyminimumsuffix() != nil {
			hs.Profile = "HISTORY-MIN"
		} else if suffix.Historymoderatesuffix() != nil {
			hs.Profile = "HISTORY-MOD"
		} else if suffix.Historymaximumsuffix() != nil {
			hs.Profile = "HISTORY-MAX"
		}
	}

	return hs
}

func (v *astBuilder) VisitHistorysupplement(ctx *grammar.HistorysupplementContext) interface{} {
	// This is typically called from visitHistorySupplement with an operand.
	// If called standalone, wrap with nil operand.
	return v.visitHistorySupplement(ctx, nil)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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
