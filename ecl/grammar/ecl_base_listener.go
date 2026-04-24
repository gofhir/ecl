// Code generated from ECL.g4 by ANTLR 4.13.2. DO NOT EDIT.

package grammar // ECL
import "github.com/antlr4-go/antlr/v4"

// BaseECLListener is a complete listener for a parse tree produced by ECLParser.
type BaseECLListener struct{}

var _ ECLListener = &BaseECLListener{}

// VisitTerminal is called when a terminal node is visited.
func (s *BaseECLListener) VisitTerminal(node antlr.TerminalNode) {}

// VisitErrorNode is called when an error node is visited.
func (s *BaseECLListener) VisitErrorNode(node antlr.ErrorNode) {}

// EnterEveryRule is called when any rule is entered.
func (s *BaseECLListener) EnterEveryRule(ctx antlr.ParserRuleContext) {}

// ExitEveryRule is called when any rule is exited.
func (s *BaseECLListener) ExitEveryRule(ctx antlr.ParserRuleContext) {}

// EnterExpressionconstraint is called when production expressionconstraint is entered.
func (s *BaseECLListener) EnterExpressionconstraint(ctx *ExpressionconstraintContext) {}

// ExitExpressionconstraint is called when production expressionconstraint is exited.
func (s *BaseECLListener) ExitExpressionconstraint(ctx *ExpressionconstraintContext) {}

// EnterRefinedexpressionconstraint is called when production refinedexpressionconstraint is entered.
func (s *BaseECLListener) EnterRefinedexpressionconstraint(ctx *RefinedexpressionconstraintContext) {}

// ExitRefinedexpressionconstraint is called when production refinedexpressionconstraint is exited.
func (s *BaseECLListener) ExitRefinedexpressionconstraint(ctx *RefinedexpressionconstraintContext) {}

// EnterCompoundexpressionconstraint is called when production compoundexpressionconstraint is entered.
func (s *BaseECLListener) EnterCompoundexpressionconstraint(ctx *CompoundexpressionconstraintContext) {
}

// ExitCompoundexpressionconstraint is called when production compoundexpressionconstraint is exited.
func (s *BaseECLListener) ExitCompoundexpressionconstraint(ctx *CompoundexpressionconstraintContext) {
}

// EnterConjunctionexpressionconstraint is called when production conjunctionexpressionconstraint is entered.
func (s *BaseECLListener) EnterConjunctionexpressionconstraint(ctx *ConjunctionexpressionconstraintContext) {
}

// ExitConjunctionexpressionconstraint is called when production conjunctionexpressionconstraint is exited.
func (s *BaseECLListener) ExitConjunctionexpressionconstraint(ctx *ConjunctionexpressionconstraintContext) {
}

// EnterDisjunctionexpressionconstraint is called when production disjunctionexpressionconstraint is entered.
func (s *BaseECLListener) EnterDisjunctionexpressionconstraint(ctx *DisjunctionexpressionconstraintContext) {
}

// ExitDisjunctionexpressionconstraint is called when production disjunctionexpressionconstraint is exited.
func (s *BaseECLListener) ExitDisjunctionexpressionconstraint(ctx *DisjunctionexpressionconstraintContext) {
}

// EnterExclusionexpressionconstraint is called when production exclusionexpressionconstraint is entered.
func (s *BaseECLListener) EnterExclusionexpressionconstraint(ctx *ExclusionexpressionconstraintContext) {
}

// ExitExclusionexpressionconstraint is called when production exclusionexpressionconstraint is exited.
func (s *BaseECLListener) ExitExclusionexpressionconstraint(ctx *ExclusionexpressionconstraintContext) {
}

// EnterDottedexpressionconstraint is called when production dottedexpressionconstraint is entered.
func (s *BaseECLListener) EnterDottedexpressionconstraint(ctx *DottedexpressionconstraintContext) {}

// ExitDottedexpressionconstraint is called when production dottedexpressionconstraint is exited.
func (s *BaseECLListener) ExitDottedexpressionconstraint(ctx *DottedexpressionconstraintContext) {}

// EnterDottedexpressionattribute is called when production dottedexpressionattribute is entered.
func (s *BaseECLListener) EnterDottedexpressionattribute(ctx *DottedexpressionattributeContext) {}

// ExitDottedexpressionattribute is called when production dottedexpressionattribute is exited.
func (s *BaseECLListener) ExitDottedexpressionattribute(ctx *DottedexpressionattributeContext) {}

// EnterSubexpressionconstraint is called when production subexpressionconstraint is entered.
func (s *BaseECLListener) EnterSubexpressionconstraint(ctx *SubexpressionconstraintContext) {}

// ExitSubexpressionconstraint is called when production subexpressionconstraint is exited.
func (s *BaseECLListener) ExitSubexpressionconstraint(ctx *SubexpressionconstraintContext) {}

// EnterEclfocusconcept is called when production eclfocusconcept is entered.
func (s *BaseECLListener) EnterEclfocusconcept(ctx *EclfocusconceptContext) {}

// ExitEclfocusconcept is called when production eclfocusconcept is exited.
func (s *BaseECLListener) ExitEclfocusconcept(ctx *EclfocusconceptContext) {}

// EnterDot is called when production dot is entered.
func (s *BaseECLListener) EnterDot(ctx *DotContext) {}

// ExitDot is called when production dot is exited.
func (s *BaseECLListener) ExitDot(ctx *DotContext) {}

// EnterRefsetOperator is called when production refsetOperator is entered.
func (s *BaseECLListener) EnterRefsetOperator(ctx *RefsetOperatorContext) {}

// ExitRefsetOperator is called when production refsetOperator is exited.
func (s *BaseECLListener) ExitRefsetOperator(ctx *RefsetOperatorContext) {}

// EnterMemberof is called when production memberof is entered.
func (s *BaseECLListener) EnterMemberof(ctx *MemberofContext) {}

// ExitMemberof is called when production memberof is exited.
func (s *BaseECLListener) ExitMemberof(ctx *MemberofContext) {}

// EnterRefsetContainingAny is called when production refsetContainingAny is entered.
func (s *BaseECLListener) EnterRefsetContainingAny(ctx *RefsetContainingAnyContext) {}

// ExitRefsetContainingAny is called when production refsetContainingAny is exited.
func (s *BaseECLListener) ExitRefsetContainingAny(ctx *RefsetContainingAnyContext) {}

// EnterRefsetfieldnameset is called when production refsetfieldnameset is entered.
func (s *BaseECLListener) EnterRefsetfieldnameset(ctx *RefsetfieldnamesetContext) {}

// ExitRefsetfieldnameset is called when production refsetfieldnameset is exited.
func (s *BaseECLListener) ExitRefsetfieldnameset(ctx *RefsetfieldnamesetContext) {}

// EnterRefsetfieldname is called when production refsetfieldname is entered.
func (s *BaseECLListener) EnterRefsetfieldname(ctx *RefsetfieldnameContext) {}

// ExitRefsetfieldname is called when production refsetfieldname is exited.
func (s *BaseECLListener) ExitRefsetfieldname(ctx *RefsetfieldnameContext) {}

// EnterEclconceptreference is called when production eclconceptreference is entered.
func (s *BaseECLListener) EnterEclconceptreference(ctx *EclconceptreferenceContext) {}

// ExitEclconceptreference is called when production eclconceptreference is exited.
func (s *BaseECLListener) ExitEclconceptreference(ctx *EclconceptreferenceContext) {}

// EnterEclconceptreferenceset is called when production eclconceptreferenceset is entered.
func (s *BaseECLListener) EnterEclconceptreferenceset(ctx *EclconceptreferencesetContext) {}

// ExitEclconceptreferenceset is called when production eclconceptreferenceset is exited.
func (s *BaseECLListener) ExitEclconceptreferenceset(ctx *EclconceptreferencesetContext) {}

// EnterConceptid is called when production conceptid is entered.
func (s *BaseECLListener) EnterConceptid(ctx *ConceptidContext) {}

// ExitConceptid is called when production conceptid is exited.
func (s *BaseECLListener) ExitConceptid(ctx *ConceptidContext) {}

// EnterTerm is called when production term is entered.
func (s *BaseECLListener) EnterTerm(ctx *TermContext) {}

// ExitTerm is called when production term is exited.
func (s *BaseECLListener) ExitTerm(ctx *TermContext) {}

// EnterAltidentifier is called when production altidentifier is entered.
func (s *BaseECLListener) EnterAltidentifier(ctx *AltidentifierContext) {}

// ExitAltidentifier is called when production altidentifier is exited.
func (s *BaseECLListener) ExitAltidentifier(ctx *AltidentifierContext) {}

// EnterAltidentifierschemealias is called when production altidentifierschemealias is entered.
func (s *BaseECLListener) EnterAltidentifierschemealias(ctx *AltidentifierschemealiasContext) {}

// ExitAltidentifierschemealias is called when production altidentifierschemealias is exited.
func (s *BaseECLListener) ExitAltidentifierschemealias(ctx *AltidentifierschemealiasContext) {}

// EnterAltidentifiercodewithinquotes is called when production altidentifiercodewithinquotes is entered.
func (s *BaseECLListener) EnterAltidentifiercodewithinquotes(ctx *AltidentifiercodewithinquotesContext) {
}

// ExitAltidentifiercodewithinquotes is called when production altidentifiercodewithinquotes is exited.
func (s *BaseECLListener) ExitAltidentifiercodewithinquotes(ctx *AltidentifiercodewithinquotesContext) {
}

// EnterAltidentifiercodewithoutquotes is called when production altidentifiercodewithoutquotes is entered.
func (s *BaseECLListener) EnterAltidentifiercodewithoutquotes(ctx *AltidentifiercodewithoutquotesContext) {
}

// ExitAltidentifiercodewithoutquotes is called when production altidentifiercodewithoutquotes is exited.
func (s *BaseECLListener) ExitAltidentifiercodewithoutquotes(ctx *AltidentifiercodewithoutquotesContext) {
}

// EnterWildcard is called when production wildcard is entered.
func (s *BaseECLListener) EnterWildcard(ctx *WildcardContext) {}

// ExitWildcard is called when production wildcard is exited.
func (s *BaseECLListener) ExitWildcard(ctx *WildcardContext) {}

// EnterConstraintoperator is called when production constraintoperator is entered.
func (s *BaseECLListener) EnterConstraintoperator(ctx *ConstraintoperatorContext) {}

// ExitConstraintoperator is called when production constraintoperator is exited.
func (s *BaseECLListener) ExitConstraintoperator(ctx *ConstraintoperatorContext) {}

// EnterDescendantof is called when production descendantof is entered.
func (s *BaseECLListener) EnterDescendantof(ctx *DescendantofContext) {}

// ExitDescendantof is called when production descendantof is exited.
func (s *BaseECLListener) ExitDescendantof(ctx *DescendantofContext) {}

// EnterDescendantorselfof is called when production descendantorselfof is entered.
func (s *BaseECLListener) EnterDescendantorselfof(ctx *DescendantorselfofContext) {}

// ExitDescendantorselfof is called when production descendantorselfof is exited.
func (s *BaseECLListener) ExitDescendantorselfof(ctx *DescendantorselfofContext) {}

// EnterChildof is called when production childof is entered.
func (s *BaseECLListener) EnterChildof(ctx *ChildofContext) {}

// ExitChildof is called when production childof is exited.
func (s *BaseECLListener) ExitChildof(ctx *ChildofContext) {}

// EnterChildorselfof is called when production childorselfof is entered.
func (s *BaseECLListener) EnterChildorselfof(ctx *ChildorselfofContext) {}

// ExitChildorselfof is called when production childorselfof is exited.
func (s *BaseECLListener) ExitChildorselfof(ctx *ChildorselfofContext) {}

// EnterAncestorof is called when production ancestorof is entered.
func (s *BaseECLListener) EnterAncestorof(ctx *AncestorofContext) {}

// ExitAncestorof is called when production ancestorof is exited.
func (s *BaseECLListener) ExitAncestorof(ctx *AncestorofContext) {}

// EnterAncestororselfof is called when production ancestororselfof is entered.
func (s *BaseECLListener) EnterAncestororselfof(ctx *AncestororselfofContext) {}

// ExitAncestororselfof is called when production ancestororselfof is exited.
func (s *BaseECLListener) ExitAncestororselfof(ctx *AncestororselfofContext) {}

// EnterParentof is called when production parentof is entered.
func (s *BaseECLListener) EnterParentof(ctx *ParentofContext) {}

// ExitParentof is called when production parentof is exited.
func (s *BaseECLListener) ExitParentof(ctx *ParentofContext) {}

// EnterParentorselfof is called when production parentorselfof is entered.
func (s *BaseECLListener) EnterParentorselfof(ctx *ParentorselfofContext) {}

// ExitParentorselfof is called when production parentorselfof is exited.
func (s *BaseECLListener) ExitParentorselfof(ctx *ParentorselfofContext) {}

// EnterTop is called when production top is entered.
func (s *BaseECLListener) EnterTop(ctx *TopContext) {}

// ExitTop is called when production top is exited.
func (s *BaseECLListener) ExitTop(ctx *TopContext) {}

// EnterBottom is called when production bottom is entered.
func (s *BaseECLListener) EnterBottom(ctx *BottomContext) {}

// ExitBottom is called when production bottom is exited.
func (s *BaseECLListener) ExitBottom(ctx *BottomContext) {}

// EnterConjunction is called when production conjunction is entered.
func (s *BaseECLListener) EnterConjunction(ctx *ConjunctionContext) {}

// ExitConjunction is called when production conjunction is exited.
func (s *BaseECLListener) ExitConjunction(ctx *ConjunctionContext) {}

// EnterDisjunction is called when production disjunction is entered.
func (s *BaseECLListener) EnterDisjunction(ctx *DisjunctionContext) {}

// ExitDisjunction is called when production disjunction is exited.
func (s *BaseECLListener) ExitDisjunction(ctx *DisjunctionContext) {}

// EnterExclusion is called when production exclusion is entered.
func (s *BaseECLListener) EnterExclusion(ctx *ExclusionContext) {}

// ExitExclusion is called when production exclusion is exited.
func (s *BaseECLListener) ExitExclusion(ctx *ExclusionContext) {}

// EnterEclrefinement is called when production eclrefinement is entered.
func (s *BaseECLListener) EnterEclrefinement(ctx *EclrefinementContext) {}

// ExitEclrefinement is called when production eclrefinement is exited.
func (s *BaseECLListener) ExitEclrefinement(ctx *EclrefinementContext) {}

// EnterConjunctionrefinementset is called when production conjunctionrefinementset is entered.
func (s *BaseECLListener) EnterConjunctionrefinementset(ctx *ConjunctionrefinementsetContext) {}

// ExitConjunctionrefinementset is called when production conjunctionrefinementset is exited.
func (s *BaseECLListener) ExitConjunctionrefinementset(ctx *ConjunctionrefinementsetContext) {}

// EnterDisjunctionrefinementset is called when production disjunctionrefinementset is entered.
func (s *BaseECLListener) EnterDisjunctionrefinementset(ctx *DisjunctionrefinementsetContext) {}

// ExitDisjunctionrefinementset is called when production disjunctionrefinementset is exited.
func (s *BaseECLListener) ExitDisjunctionrefinementset(ctx *DisjunctionrefinementsetContext) {}

// EnterSubrefinement is called when production subrefinement is entered.
func (s *BaseECLListener) EnterSubrefinement(ctx *SubrefinementContext) {}

// ExitSubrefinement is called when production subrefinement is exited.
func (s *BaseECLListener) ExitSubrefinement(ctx *SubrefinementContext) {}

// EnterEclattributeset is called when production eclattributeset is entered.
func (s *BaseECLListener) EnterEclattributeset(ctx *EclattributesetContext) {}

// ExitEclattributeset is called when production eclattributeset is exited.
func (s *BaseECLListener) ExitEclattributeset(ctx *EclattributesetContext) {}

// EnterConjunctionattributeset is called when production conjunctionattributeset is entered.
func (s *BaseECLListener) EnterConjunctionattributeset(ctx *ConjunctionattributesetContext) {}

// ExitConjunctionattributeset is called when production conjunctionattributeset is exited.
func (s *BaseECLListener) ExitConjunctionattributeset(ctx *ConjunctionattributesetContext) {}

// EnterDisjunctionattributeset is called when production disjunctionattributeset is entered.
func (s *BaseECLListener) EnterDisjunctionattributeset(ctx *DisjunctionattributesetContext) {}

// ExitDisjunctionattributeset is called when production disjunctionattributeset is exited.
func (s *BaseECLListener) ExitDisjunctionattributeset(ctx *DisjunctionattributesetContext) {}

// EnterSubattributeset is called when production subattributeset is entered.
func (s *BaseECLListener) EnterSubattributeset(ctx *SubattributesetContext) {}

// ExitSubattributeset is called when production subattributeset is exited.
func (s *BaseECLListener) ExitSubattributeset(ctx *SubattributesetContext) {}

// EnterEclattributegroup is called when production eclattributegroup is entered.
func (s *BaseECLListener) EnterEclattributegroup(ctx *EclattributegroupContext) {}

// ExitEclattributegroup is called when production eclattributegroup is exited.
func (s *BaseECLListener) ExitEclattributegroup(ctx *EclattributegroupContext) {}

// EnterEclattribute is called when production eclattribute is entered.
func (s *BaseECLListener) EnterEclattribute(ctx *EclattributeContext) {}

// ExitEclattribute is called when production eclattribute is exited.
func (s *BaseECLListener) ExitEclattribute(ctx *EclattributeContext) {}

// EnterCardinality is called when production cardinality is entered.
func (s *BaseECLListener) EnterCardinality(ctx *CardinalityContext) {}

// ExitCardinality is called when production cardinality is exited.
func (s *BaseECLListener) ExitCardinality(ctx *CardinalityContext) {}

// EnterMinvalue is called when production minvalue is entered.
func (s *BaseECLListener) EnterMinvalue(ctx *MinvalueContext) {}

// ExitMinvalue is called when production minvalue is exited.
func (s *BaseECLListener) ExitMinvalue(ctx *MinvalueContext) {}

// EnterTo is called when production to is entered.
func (s *BaseECLListener) EnterTo(ctx *ToContext) {}

// ExitTo is called when production to is exited.
func (s *BaseECLListener) ExitTo(ctx *ToContext) {}

// EnterMaxvalue is called when production maxvalue is entered.
func (s *BaseECLListener) EnterMaxvalue(ctx *MaxvalueContext) {}

// ExitMaxvalue is called when production maxvalue is exited.
func (s *BaseECLListener) ExitMaxvalue(ctx *MaxvalueContext) {}

// EnterMany is called when production many is entered.
func (s *BaseECLListener) EnterMany(ctx *ManyContext) {}

// ExitMany is called when production many is exited.
func (s *BaseECLListener) ExitMany(ctx *ManyContext) {}

// EnterReverseflag is called when production reverseflag is entered.
func (s *BaseECLListener) EnterReverseflag(ctx *ReverseflagContext) {}

// ExitReverseflag is called when production reverseflag is exited.
func (s *BaseECLListener) ExitReverseflag(ctx *ReverseflagContext) {}

// EnterEclattributename is called when production eclattributename is entered.
func (s *BaseECLListener) EnterEclattributename(ctx *EclattributenameContext) {}

// ExitEclattributename is called when production eclattributename is exited.
func (s *BaseECLListener) ExitEclattributename(ctx *EclattributenameContext) {}

// EnterExpressioncomparisonoperator is called when production expressioncomparisonoperator is entered.
func (s *BaseECLListener) EnterExpressioncomparisonoperator(ctx *ExpressioncomparisonoperatorContext) {
}

// ExitExpressioncomparisonoperator is called when production expressioncomparisonoperator is exited.
func (s *BaseECLListener) ExitExpressioncomparisonoperator(ctx *ExpressioncomparisonoperatorContext) {
}

// EnterNumericcomparisonoperator is called when production numericcomparisonoperator is entered.
func (s *BaseECLListener) EnterNumericcomparisonoperator(ctx *NumericcomparisonoperatorContext) {}

// ExitNumericcomparisonoperator is called when production numericcomparisonoperator is exited.
func (s *BaseECLListener) ExitNumericcomparisonoperator(ctx *NumericcomparisonoperatorContext) {}

// EnterTimecomparisonoperator is called when production timecomparisonoperator is entered.
func (s *BaseECLListener) EnterTimecomparisonoperator(ctx *TimecomparisonoperatorContext) {}

// ExitTimecomparisonoperator is called when production timecomparisonoperator is exited.
func (s *BaseECLListener) ExitTimecomparisonoperator(ctx *TimecomparisonoperatorContext) {}

// EnterStringcomparisonoperator is called when production stringcomparisonoperator is entered.
func (s *BaseECLListener) EnterStringcomparisonoperator(ctx *StringcomparisonoperatorContext) {}

// ExitStringcomparisonoperator is called when production stringcomparisonoperator is exited.
func (s *BaseECLListener) ExitStringcomparisonoperator(ctx *StringcomparisonoperatorContext) {}

// EnterBooleancomparisonoperator is called when production booleancomparisonoperator is entered.
func (s *BaseECLListener) EnterBooleancomparisonoperator(ctx *BooleancomparisonoperatorContext) {}

// ExitBooleancomparisonoperator is called when production booleancomparisonoperator is exited.
func (s *BaseECLListener) ExitBooleancomparisonoperator(ctx *BooleancomparisonoperatorContext) {}

// EnterIdcomparisonoperator is called when production idcomparisonoperator is entered.
func (s *BaseECLListener) EnterIdcomparisonoperator(ctx *IdcomparisonoperatorContext) {}

// ExitIdcomparisonoperator is called when production idcomparisonoperator is exited.
func (s *BaseECLListener) ExitIdcomparisonoperator(ctx *IdcomparisonoperatorContext) {}

// EnterDescriptionfilterconstraint is called when production descriptionfilterconstraint is entered.
func (s *BaseECLListener) EnterDescriptionfilterconstraint(ctx *DescriptionfilterconstraintContext) {}

// ExitDescriptionfilterconstraint is called when production descriptionfilterconstraint is exited.
func (s *BaseECLListener) ExitDescriptionfilterconstraint(ctx *DescriptionfilterconstraintContext) {}

// EnterDescriptionfilter is called when production descriptionfilter is entered.
func (s *BaseECLListener) EnterDescriptionfilter(ctx *DescriptionfilterContext) {}

// ExitDescriptionfilter is called when production descriptionfilter is exited.
func (s *BaseECLListener) ExitDescriptionfilter(ctx *DescriptionfilterContext) {}

// EnterDescriptionidfilter is called when production descriptionidfilter is entered.
func (s *BaseECLListener) EnterDescriptionidfilter(ctx *DescriptionidfilterContext) {}

// ExitDescriptionidfilter is called when production descriptionidfilter is exited.
func (s *BaseECLListener) ExitDescriptionidfilter(ctx *DescriptionidfilterContext) {}

// EnterDescriptionidkeyword is called when production descriptionidkeyword is entered.
func (s *BaseECLListener) EnterDescriptionidkeyword(ctx *DescriptionidkeywordContext) {}

// ExitDescriptionidkeyword is called when production descriptionidkeyword is exited.
func (s *BaseECLListener) ExitDescriptionidkeyword(ctx *DescriptionidkeywordContext) {}

// EnterDescriptionid is called when production descriptionid is entered.
func (s *BaseECLListener) EnterDescriptionid(ctx *DescriptionidContext) {}

// ExitDescriptionid is called when production descriptionid is exited.
func (s *BaseECLListener) ExitDescriptionid(ctx *DescriptionidContext) {}

// EnterDescriptionidset is called when production descriptionidset is entered.
func (s *BaseECLListener) EnterDescriptionidset(ctx *DescriptionidsetContext) {}

// ExitDescriptionidset is called when production descriptionidset is exited.
func (s *BaseECLListener) ExitDescriptionidset(ctx *DescriptionidsetContext) {}

// EnterTermfilter is called when production termfilter is entered.
func (s *BaseECLListener) EnterTermfilter(ctx *TermfilterContext) {}

// ExitTermfilter is called when production termfilter is exited.
func (s *BaseECLListener) ExitTermfilter(ctx *TermfilterContext) {}

// EnterTermkeyword is called when production termkeyword is entered.
func (s *BaseECLListener) EnterTermkeyword(ctx *TermkeywordContext) {}

// ExitTermkeyword is called when production termkeyword is exited.
func (s *BaseECLListener) ExitTermkeyword(ctx *TermkeywordContext) {}

// EnterTypedsearchterm is called when production typedsearchterm is entered.
func (s *BaseECLListener) EnterTypedsearchterm(ctx *TypedsearchtermContext) {}

// ExitTypedsearchterm is called when production typedsearchterm is exited.
func (s *BaseECLListener) ExitTypedsearchterm(ctx *TypedsearchtermContext) {}

// EnterTypedsearchtermset is called when production typedsearchtermset is entered.
func (s *BaseECLListener) EnterTypedsearchtermset(ctx *TypedsearchtermsetContext) {}

// ExitTypedsearchtermset is called when production typedsearchtermset is exited.
func (s *BaseECLListener) ExitTypedsearchtermset(ctx *TypedsearchtermsetContext) {}

// EnterConcretestringset is called when production concretestringset is entered.
func (s *BaseECLListener) EnterConcretestringset(ctx *ConcretestringsetContext) {}

// ExitConcretestringset is called when production concretestringset is exited.
func (s *BaseECLListener) ExitConcretestringset(ctx *ConcretestringsetContext) {}

// EnterConcretestring is called when production concretestring is entered.
func (s *BaseECLListener) EnterConcretestring(ctx *ConcretestringContext) {}

// ExitConcretestring is called when production concretestring is exited.
func (s *BaseECLListener) ExitConcretestring(ctx *ConcretestringContext) {}

// EnterConcretestringcharacters is called when production concretestringcharacters is entered.
func (s *BaseECLListener) EnterConcretestringcharacters(ctx *ConcretestringcharactersContext) {}

// ExitConcretestringcharacters is called when production concretestringcharacters is exited.
func (s *BaseECLListener) ExitConcretestringcharacters(ctx *ConcretestringcharactersContext) {}

// EnterWild is called when production wild is entered.
func (s *BaseECLListener) EnterWild(ctx *WildContext) {}

// ExitWild is called when production wild is exited.
func (s *BaseECLListener) ExitWild(ctx *WildContext) {}

// EnterMatchkeyword is called when production matchkeyword is entered.
func (s *BaseECLListener) EnterMatchkeyword(ctx *MatchkeywordContext) {}

// ExitMatchkeyword is called when production matchkeyword is exited.
func (s *BaseECLListener) ExitMatchkeyword(ctx *MatchkeywordContext) {}

// EnterMatchsearchterm is called when production matchsearchterm is entered.
func (s *BaseECLListener) EnterMatchsearchterm(ctx *MatchsearchtermContext) {}

// ExitMatchsearchterm is called when production matchsearchterm is exited.
func (s *BaseECLListener) ExitMatchsearchterm(ctx *MatchsearchtermContext) {}

// EnterMatchsearchtermset is called when production matchsearchtermset is entered.
func (s *BaseECLListener) EnterMatchsearchtermset(ctx *MatchsearchtermsetContext) {}

// ExitMatchsearchtermset is called when production matchsearchtermset is exited.
func (s *BaseECLListener) ExitMatchsearchtermset(ctx *MatchsearchtermsetContext) {}

// EnterWildsearchterm is called when production wildsearchterm is entered.
func (s *BaseECLListener) EnterWildsearchterm(ctx *WildsearchtermContext) {}

// ExitWildsearchterm is called when production wildsearchterm is exited.
func (s *BaseECLListener) ExitWildsearchterm(ctx *WildsearchtermContext) {}

// EnterWildsearchtermset is called when production wildsearchtermset is entered.
func (s *BaseECLListener) EnterWildsearchtermset(ctx *WildsearchtermsetContext) {}

// ExitWildsearchtermset is called when production wildsearchtermset is exited.
func (s *BaseECLListener) ExitWildsearchtermset(ctx *WildsearchtermsetContext) {}

// EnterLanguagefilter is called when production languagefilter is entered.
func (s *BaseECLListener) EnterLanguagefilter(ctx *LanguagefilterContext) {}

// ExitLanguagefilter is called when production languagefilter is exited.
func (s *BaseECLListener) ExitLanguagefilter(ctx *LanguagefilterContext) {}

// EnterLanguage is called when production language is entered.
func (s *BaseECLListener) EnterLanguage(ctx *LanguageContext) {}

// ExitLanguage is called when production language is exited.
func (s *BaseECLListener) ExitLanguage(ctx *LanguageContext) {}

// EnterLanguagecode is called when production languagecode is entered.
func (s *BaseECLListener) EnterLanguagecode(ctx *LanguagecodeContext) {}

// ExitLanguagecode is called when production languagecode is exited.
func (s *BaseECLListener) ExitLanguagecode(ctx *LanguagecodeContext) {}

// EnterLanguagecodeset is called when production languagecodeset is entered.
func (s *BaseECLListener) EnterLanguagecodeset(ctx *LanguagecodesetContext) {}

// ExitLanguagecodeset is called when production languagecodeset is exited.
func (s *BaseECLListener) ExitLanguagecodeset(ctx *LanguagecodesetContext) {}

// EnterTypefilter is called when production typefilter is entered.
func (s *BaseECLListener) EnterTypefilter(ctx *TypefilterContext) {}

// ExitTypefilter is called when production typefilter is exited.
func (s *BaseECLListener) ExitTypefilter(ctx *TypefilterContext) {}

// EnterTypeidfilter is called when production typeidfilter is entered.
func (s *BaseECLListener) EnterTypeidfilter(ctx *TypeidfilterContext) {}

// ExitTypeidfilter is called when production typeidfilter is exited.
func (s *BaseECLListener) ExitTypeidfilter(ctx *TypeidfilterContext) {}

// EnterTypeid is called when production typeid is entered.
func (s *BaseECLListener) EnterTypeid(ctx *TypeidContext) {}

// ExitTypeid is called when production typeid is exited.
func (s *BaseECLListener) ExitTypeid(ctx *TypeidContext) {}

// EnterTypetokenfilter is called when production typetokenfilter is entered.
func (s *BaseECLListener) EnterTypetokenfilter(ctx *TypetokenfilterContext) {}

// ExitTypetokenfilter is called when production typetokenfilter is exited.
func (s *BaseECLListener) ExitTypetokenfilter(ctx *TypetokenfilterContext) {}

// EnterType is called when production type is entered.
func (s *BaseECLListener) EnterType(ctx *TypeContext) {}

// ExitType is called when production type is exited.
func (s *BaseECLListener) ExitType(ctx *TypeContext) {}

// EnterTypetoken is called when production typetoken is entered.
func (s *BaseECLListener) EnterTypetoken(ctx *TypetokenContext) {}

// ExitTypetoken is called when production typetoken is exited.
func (s *BaseECLListener) ExitTypetoken(ctx *TypetokenContext) {}

// EnterTypetokenset is called when production typetokenset is entered.
func (s *BaseECLListener) EnterTypetokenset(ctx *TypetokensetContext) {}

// ExitTypetokenset is called when production typetokenset is exited.
func (s *BaseECLListener) ExitTypetokenset(ctx *TypetokensetContext) {}

// EnterSynonym is called when production synonym is entered.
func (s *BaseECLListener) EnterSynonym(ctx *SynonymContext) {}

// ExitSynonym is called when production synonym is exited.
func (s *BaseECLListener) ExitSynonym(ctx *SynonymContext) {}

// EnterFullyspecifiedname is called when production fullyspecifiedname is entered.
func (s *BaseECLListener) EnterFullyspecifiedname(ctx *FullyspecifiednameContext) {}

// ExitFullyspecifiedname is called when production fullyspecifiedname is exited.
func (s *BaseECLListener) ExitFullyspecifiedname(ctx *FullyspecifiednameContext) {}

// EnterDefinition is called when production definition is entered.
func (s *BaseECLListener) EnterDefinition(ctx *DefinitionContext) {}

// ExitDefinition is called when production definition is exited.
func (s *BaseECLListener) ExitDefinition(ctx *DefinitionContext) {}

// EnterDialectfilter is called when production dialectfilter is entered.
func (s *BaseECLListener) EnterDialectfilter(ctx *DialectfilterContext) {}

// ExitDialectfilter is called when production dialectfilter is exited.
func (s *BaseECLListener) ExitDialectfilter(ctx *DialectfilterContext) {}

// EnterDialectidfilter is called when production dialectidfilter is entered.
func (s *BaseECLListener) EnterDialectidfilter(ctx *DialectidfilterContext) {}

// ExitDialectidfilter is called when production dialectidfilter is exited.
func (s *BaseECLListener) ExitDialectidfilter(ctx *DialectidfilterContext) {}

// EnterDialectid is called when production dialectid is entered.
func (s *BaseECLListener) EnterDialectid(ctx *DialectidContext) {}

// ExitDialectid is called when production dialectid is exited.
func (s *BaseECLListener) ExitDialectid(ctx *DialectidContext) {}

// EnterDialectaliasfilter is called when production dialectaliasfilter is entered.
func (s *BaseECLListener) EnterDialectaliasfilter(ctx *DialectaliasfilterContext) {}

// ExitDialectaliasfilter is called when production dialectaliasfilter is exited.
func (s *BaseECLListener) ExitDialectaliasfilter(ctx *DialectaliasfilterContext) {}

// EnterDialect is called when production dialect is entered.
func (s *BaseECLListener) EnterDialect(ctx *DialectContext) {}

// ExitDialect is called when production dialect is exited.
func (s *BaseECLListener) ExitDialect(ctx *DialectContext) {}

// EnterDialectalias is called when production dialectalias is entered.
func (s *BaseECLListener) EnterDialectalias(ctx *DialectaliasContext) {}

// ExitDialectalias is called when production dialectalias is exited.
func (s *BaseECLListener) ExitDialectalias(ctx *DialectaliasContext) {}

// EnterDialectaliasset is called when production dialectaliasset is entered.
func (s *BaseECLListener) EnterDialectaliasset(ctx *DialectaliassetContext) {}

// ExitDialectaliasset is called when production dialectaliasset is exited.
func (s *BaseECLListener) ExitDialectaliasset(ctx *DialectaliassetContext) {}

// EnterDialectidset is called when production dialectidset is entered.
func (s *BaseECLListener) EnterDialectidset(ctx *DialectidsetContext) {}

// ExitDialectidset is called when production dialectidset is exited.
func (s *BaseECLListener) ExitDialectidset(ctx *DialectidsetContext) {}

// EnterAcceptabilityset is called when production acceptabilityset is entered.
func (s *BaseECLListener) EnterAcceptabilityset(ctx *AcceptabilitysetContext) {}

// ExitAcceptabilityset is called when production acceptabilityset is exited.
func (s *BaseECLListener) ExitAcceptabilityset(ctx *AcceptabilitysetContext) {}

// EnterAcceptabilityconceptreferenceset is called when production acceptabilityconceptreferenceset is entered.
func (s *BaseECLListener) EnterAcceptabilityconceptreferenceset(ctx *AcceptabilityconceptreferencesetContext) {
}

// ExitAcceptabilityconceptreferenceset is called when production acceptabilityconceptreferenceset is exited.
func (s *BaseECLListener) ExitAcceptabilityconceptreferenceset(ctx *AcceptabilityconceptreferencesetContext) {
}

// EnterAcceptabilitytokenset is called when production acceptabilitytokenset is entered.
func (s *BaseECLListener) EnterAcceptabilitytokenset(ctx *AcceptabilitytokensetContext) {}

// ExitAcceptabilitytokenset is called when production acceptabilitytokenset is exited.
func (s *BaseECLListener) ExitAcceptabilitytokenset(ctx *AcceptabilitytokensetContext) {}

// EnterAcceptabilitytoken is called when production acceptabilitytoken is entered.
func (s *BaseECLListener) EnterAcceptabilitytoken(ctx *AcceptabilitytokenContext) {}

// ExitAcceptabilitytoken is called when production acceptabilitytoken is exited.
func (s *BaseECLListener) ExitAcceptabilitytoken(ctx *AcceptabilitytokenContext) {}

// EnterAcceptable is called when production acceptable is entered.
func (s *BaseECLListener) EnterAcceptable(ctx *AcceptableContext) {}

// ExitAcceptable is called when production acceptable is exited.
func (s *BaseECLListener) ExitAcceptable(ctx *AcceptableContext) {}

// EnterPreferred is called when production preferred is entered.
func (s *BaseECLListener) EnterPreferred(ctx *PreferredContext) {}

// ExitPreferred is called when production preferred is exited.
func (s *BaseECLListener) ExitPreferred(ctx *PreferredContext) {}

// EnterConceptfilterconstraint is called when production conceptfilterconstraint is entered.
func (s *BaseECLListener) EnterConceptfilterconstraint(ctx *ConceptfilterconstraintContext) {}

// ExitConceptfilterconstraint is called when production conceptfilterconstraint is exited.
func (s *BaseECLListener) ExitConceptfilterconstraint(ctx *ConceptfilterconstraintContext) {}

// EnterConceptfilter is called when production conceptfilter is entered.
func (s *BaseECLListener) EnterConceptfilter(ctx *ConceptfilterContext) {}

// ExitConceptfilter is called when production conceptfilter is exited.
func (s *BaseECLListener) ExitConceptfilter(ctx *ConceptfilterContext) {}

// EnterDefinitionstatusfilter is called when production definitionstatusfilter is entered.
func (s *BaseECLListener) EnterDefinitionstatusfilter(ctx *DefinitionstatusfilterContext) {}

// ExitDefinitionstatusfilter is called when production definitionstatusfilter is exited.
func (s *BaseECLListener) ExitDefinitionstatusfilter(ctx *DefinitionstatusfilterContext) {}

// EnterDefinitionstatusidfilter is called when production definitionstatusidfilter is entered.
func (s *BaseECLListener) EnterDefinitionstatusidfilter(ctx *DefinitionstatusidfilterContext) {}

// ExitDefinitionstatusidfilter is called when production definitionstatusidfilter is exited.
func (s *BaseECLListener) ExitDefinitionstatusidfilter(ctx *DefinitionstatusidfilterContext) {}

// EnterDefinitionstatusidkeyword is called when production definitionstatusidkeyword is entered.
func (s *BaseECLListener) EnterDefinitionstatusidkeyword(ctx *DefinitionstatusidkeywordContext) {}

// ExitDefinitionstatusidkeyword is called when production definitionstatusidkeyword is exited.
func (s *BaseECLListener) ExitDefinitionstatusidkeyword(ctx *DefinitionstatusidkeywordContext) {}

// EnterDefinitionstatustokenfilter is called when production definitionstatustokenfilter is entered.
func (s *BaseECLListener) EnterDefinitionstatustokenfilter(ctx *DefinitionstatustokenfilterContext) {}

// ExitDefinitionstatustokenfilter is called when production definitionstatustokenfilter is exited.
func (s *BaseECLListener) ExitDefinitionstatustokenfilter(ctx *DefinitionstatustokenfilterContext) {}

// EnterDefinitionstatuskeyword is called when production definitionstatuskeyword is entered.
func (s *BaseECLListener) EnterDefinitionstatuskeyword(ctx *DefinitionstatuskeywordContext) {}

// ExitDefinitionstatuskeyword is called when production definitionstatuskeyword is exited.
func (s *BaseECLListener) ExitDefinitionstatuskeyword(ctx *DefinitionstatuskeywordContext) {}

// EnterDefinitionstatustoken is called when production definitionstatustoken is entered.
func (s *BaseECLListener) EnterDefinitionstatustoken(ctx *DefinitionstatustokenContext) {}

// ExitDefinitionstatustoken is called when production definitionstatustoken is exited.
func (s *BaseECLListener) ExitDefinitionstatustoken(ctx *DefinitionstatustokenContext) {}

// EnterDefinitionstatustokenset is called when production definitionstatustokenset is entered.
func (s *BaseECLListener) EnterDefinitionstatustokenset(ctx *DefinitionstatustokensetContext) {}

// ExitDefinitionstatustokenset is called when production definitionstatustokenset is exited.
func (s *BaseECLListener) ExitDefinitionstatustokenset(ctx *DefinitionstatustokensetContext) {}

// EnterPrimitivetoken is called when production primitivetoken is entered.
func (s *BaseECLListener) EnterPrimitivetoken(ctx *PrimitivetokenContext) {}

// ExitPrimitivetoken is called when production primitivetoken is exited.
func (s *BaseECLListener) ExitPrimitivetoken(ctx *PrimitivetokenContext) {}

// EnterDefinedtoken is called when production definedtoken is entered.
func (s *BaseECLListener) EnterDefinedtoken(ctx *DefinedtokenContext) {}

// ExitDefinedtoken is called when production definedtoken is exited.
func (s *BaseECLListener) ExitDefinedtoken(ctx *DefinedtokenContext) {}

// EnterModulefilter is called when production modulefilter is entered.
func (s *BaseECLListener) EnterModulefilter(ctx *ModulefilterContext) {}

// ExitModulefilter is called when production modulefilter is exited.
func (s *BaseECLListener) ExitModulefilter(ctx *ModulefilterContext) {}

// EnterModuleidkeyword is called when production moduleidkeyword is entered.
func (s *BaseECLListener) EnterModuleidkeyword(ctx *ModuleidkeywordContext) {}

// ExitModuleidkeyword is called when production moduleidkeyword is exited.
func (s *BaseECLListener) ExitModuleidkeyword(ctx *ModuleidkeywordContext) {}

// EnterEffectivetimefilter is called when production effectivetimefilter is entered.
func (s *BaseECLListener) EnterEffectivetimefilter(ctx *EffectivetimefilterContext) {}

// ExitEffectivetimefilter is called when production effectivetimefilter is exited.
func (s *BaseECLListener) ExitEffectivetimefilter(ctx *EffectivetimefilterContext) {}

// EnterEffectivetimekeyword is called when production effectivetimekeyword is entered.
func (s *BaseECLListener) EnterEffectivetimekeyword(ctx *EffectivetimekeywordContext) {}

// ExitEffectivetimekeyword is called when production effectivetimekeyword is exited.
func (s *BaseECLListener) ExitEffectivetimekeyword(ctx *EffectivetimekeywordContext) {}

// EnterTimevalue is called when production timevalue is entered.
func (s *BaseECLListener) EnterTimevalue(ctx *TimevalueContext) {}

// ExitTimevalue is called when production timevalue is exited.
func (s *BaseECLListener) ExitTimevalue(ctx *TimevalueContext) {}

// EnterTimevalueset is called when production timevalueset is entered.
func (s *BaseECLListener) EnterTimevalueset(ctx *TimevaluesetContext) {}

// ExitTimevalueset is called when production timevalueset is exited.
func (s *BaseECLListener) ExitTimevalueset(ctx *TimevaluesetContext) {}

// EnterYear is called when production year is entered.
func (s *BaseECLListener) EnterYear(ctx *YearContext) {}

// ExitYear is called when production year is exited.
func (s *BaseECLListener) ExitYear(ctx *YearContext) {}

// EnterMonth is called when production month is entered.
func (s *BaseECLListener) EnterMonth(ctx *MonthContext) {}

// ExitMonth is called when production month is exited.
func (s *BaseECLListener) ExitMonth(ctx *MonthContext) {}

// EnterDay is called when production day is entered.
func (s *BaseECLListener) EnterDay(ctx *DayContext) {}

// ExitDay is called when production day is exited.
func (s *BaseECLListener) ExitDay(ctx *DayContext) {}

// EnterActivefilter is called when production activefilter is entered.
func (s *BaseECLListener) EnterActivefilter(ctx *ActivefilterContext) {}

// ExitActivefilter is called when production activefilter is exited.
func (s *BaseECLListener) ExitActivefilter(ctx *ActivefilterContext) {}

// EnterActivekeyword is called when production activekeyword is entered.
func (s *BaseECLListener) EnterActivekeyword(ctx *ActivekeywordContext) {}

// ExitActivekeyword is called when production activekeyword is exited.
func (s *BaseECLListener) ExitActivekeyword(ctx *ActivekeywordContext) {}

// EnterActivevalue is called when production activevalue is entered.
func (s *BaseECLListener) EnterActivevalue(ctx *ActivevalueContext) {}

// ExitActivevalue is called when production activevalue is exited.
func (s *BaseECLListener) ExitActivevalue(ctx *ActivevalueContext) {}

// EnterActivetruevalue is called when production activetruevalue is entered.
func (s *BaseECLListener) EnterActivetruevalue(ctx *ActivetruevalueContext) {}

// ExitActivetruevalue is called when production activetruevalue is exited.
func (s *BaseECLListener) ExitActivetruevalue(ctx *ActivetruevalueContext) {}

// EnterActivefalsevalue is called when production activefalsevalue is entered.
func (s *BaseECLListener) EnterActivefalsevalue(ctx *ActivefalsevalueContext) {}

// ExitActivefalsevalue is called when production activefalsevalue is exited.
func (s *BaseECLListener) ExitActivefalsevalue(ctx *ActivefalsevalueContext) {}

// EnterMemberfilterconstraint is called when production memberfilterconstraint is entered.
func (s *BaseECLListener) EnterMemberfilterconstraint(ctx *MemberfilterconstraintContext) {}

// ExitMemberfilterconstraint is called when production memberfilterconstraint is exited.
func (s *BaseECLListener) ExitMemberfilterconstraint(ctx *MemberfilterconstraintContext) {}

// EnterMemberfilter is called when production memberfilter is entered.
func (s *BaseECLListener) EnterMemberfilter(ctx *MemberfilterContext) {}

// ExitMemberfilter is called when production memberfilter is exited.
func (s *BaseECLListener) ExitMemberfilter(ctx *MemberfilterContext) {}

// EnterMemberfieldfilter is called when production memberfieldfilter is entered.
func (s *BaseECLListener) EnterMemberfieldfilter(ctx *MemberfieldfilterContext) {}

// ExitMemberfieldfilter is called when production memberfieldfilter is exited.
func (s *BaseECLListener) ExitMemberfieldfilter(ctx *MemberfieldfilterContext) {}

// EnterHistorysupplement is called when production historysupplement is entered.
func (s *BaseECLListener) EnterHistorysupplement(ctx *HistorysupplementContext) {}

// ExitHistorysupplement is called when production historysupplement is exited.
func (s *BaseECLListener) ExitHistorysupplement(ctx *HistorysupplementContext) {}

// EnterHistorykeyword is called when production historykeyword is entered.
func (s *BaseECLListener) EnterHistorykeyword(ctx *HistorykeywordContext) {}

// ExitHistorykeyword is called when production historykeyword is exited.
func (s *BaseECLListener) ExitHistorykeyword(ctx *HistorykeywordContext) {}

// EnterHistoryprofilesuffix is called when production historyprofilesuffix is entered.
func (s *BaseECLListener) EnterHistoryprofilesuffix(ctx *HistoryprofilesuffixContext) {}

// ExitHistoryprofilesuffix is called when production historyprofilesuffix is exited.
func (s *BaseECLListener) ExitHistoryprofilesuffix(ctx *HistoryprofilesuffixContext) {}

// EnterHistoryminimumsuffix is called when production historyminimumsuffix is entered.
func (s *BaseECLListener) EnterHistoryminimumsuffix(ctx *HistoryminimumsuffixContext) {}

// ExitHistoryminimumsuffix is called when production historyminimumsuffix is exited.
func (s *BaseECLListener) ExitHistoryminimumsuffix(ctx *HistoryminimumsuffixContext) {}

// EnterHistorymoderatesuffix is called when production historymoderatesuffix is entered.
func (s *BaseECLListener) EnterHistorymoderatesuffix(ctx *HistorymoderatesuffixContext) {}

// ExitHistorymoderatesuffix is called when production historymoderatesuffix is exited.
func (s *BaseECLListener) ExitHistorymoderatesuffix(ctx *HistorymoderatesuffixContext) {}

// EnterHistorymaximumsuffix is called when production historymaximumsuffix is entered.
func (s *BaseECLListener) EnterHistorymaximumsuffix(ctx *HistorymaximumsuffixContext) {}

// ExitHistorymaximumsuffix is called when production historymaximumsuffix is exited.
func (s *BaseECLListener) ExitHistorymaximumsuffix(ctx *HistorymaximumsuffixContext) {}

// EnterHistorysubset is called when production historysubset is entered.
func (s *BaseECLListener) EnterHistorysubset(ctx *HistorysubsetContext) {}

// ExitHistorysubset is called when production historysubset is exited.
func (s *BaseECLListener) ExitHistorysubset(ctx *HistorysubsetContext) {}

// EnterNumericvalue is called when production numericvalue is entered.
func (s *BaseECLListener) EnterNumericvalue(ctx *NumericvalueContext) {}

// ExitNumericvalue is called when production numericvalue is exited.
func (s *BaseECLListener) ExitNumericvalue(ctx *NumericvalueContext) {}

// EnterStringvalue is called when production stringvalue is entered.
func (s *BaseECLListener) EnterStringvalue(ctx *StringvalueContext) {}

// ExitStringvalue is called when production stringvalue is exited.
func (s *BaseECLListener) ExitStringvalue(ctx *StringvalueContext) {}

// EnterIntegervalue is called when production integervalue is entered.
func (s *BaseECLListener) EnterIntegervalue(ctx *IntegervalueContext) {}

// ExitIntegervalue is called when production integervalue is exited.
func (s *BaseECLListener) ExitIntegervalue(ctx *IntegervalueContext) {}

// EnterDecimalvalue is called when production decimalvalue is entered.
func (s *BaseECLListener) EnterDecimalvalue(ctx *DecimalvalueContext) {}

// ExitDecimalvalue is called when production decimalvalue is exited.
func (s *BaseECLListener) ExitDecimalvalue(ctx *DecimalvalueContext) {}

// EnterBooleanvalue is called when production booleanvalue is entered.
func (s *BaseECLListener) EnterBooleanvalue(ctx *BooleanvalueContext) {}

// ExitBooleanvalue is called when production booleanvalue is exited.
func (s *BaseECLListener) ExitBooleanvalue(ctx *BooleanvalueContext) {}

// EnterTrue_1 is called when production true_1 is entered.
func (s *BaseECLListener) EnterTrue_1(ctx *True_1Context) {}

// ExitTrue_1 is called when production true_1 is exited.
func (s *BaseECLListener) ExitTrue_1(ctx *True_1Context) {}

// EnterFalse_1 is called when production false_1 is entered.
func (s *BaseECLListener) EnterFalse_1(ctx *False_1Context) {}

// ExitFalse_1 is called when production false_1 is exited.
func (s *BaseECLListener) ExitFalse_1(ctx *False_1Context) {}

// EnterNonnegativeintegervalue is called when production nonnegativeintegervalue is entered.
func (s *BaseECLListener) EnterNonnegativeintegervalue(ctx *NonnegativeintegervalueContext) {}

// ExitNonnegativeintegervalue is called when production nonnegativeintegervalue is exited.
func (s *BaseECLListener) ExitNonnegativeintegervalue(ctx *NonnegativeintegervalueContext) {}

// EnterSctid is called when production sctid is entered.
func (s *BaseECLListener) EnterSctid(ctx *SctidContext) {}

// ExitSctid is called when production sctid is exited.
func (s *BaseECLListener) ExitSctid(ctx *SctidContext) {}

// EnterWs is called when production ws is entered.
func (s *BaseECLListener) EnterWs(ctx *WsContext) {}

// ExitWs is called when production ws is exited.
func (s *BaseECLListener) ExitWs(ctx *WsContext) {}

// EnterMws is called when production mws is entered.
func (s *BaseECLListener) EnterMws(ctx *MwsContext) {}

// ExitMws is called when production mws is exited.
func (s *BaseECLListener) ExitMws(ctx *MwsContext) {}

// EnterComment is called when production comment is entered.
func (s *BaseECLListener) EnterComment(ctx *CommentContext) {}

// ExitComment is called when production comment is exited.
func (s *BaseECLListener) ExitComment(ctx *CommentContext) {}

// EnterNonstarchar is called when production nonstarchar is entered.
func (s *BaseECLListener) EnterNonstarchar(ctx *NonstarcharContext) {}

// ExitNonstarchar is called when production nonstarchar is exited.
func (s *BaseECLListener) ExitNonstarchar(ctx *NonstarcharContext) {}

// EnterStarwithnonfslash is called when production starwithnonfslash is entered.
func (s *BaseECLListener) EnterStarwithnonfslash(ctx *StarwithnonfslashContext) {}

// ExitStarwithnonfslash is called when production starwithnonfslash is exited.
func (s *BaseECLListener) ExitStarwithnonfslash(ctx *StarwithnonfslashContext) {}

// EnterNonfslash is called when production nonfslash is entered.
func (s *BaseECLListener) EnterNonfslash(ctx *NonfslashContext) {}

// ExitNonfslash is called when production nonfslash is exited.
func (s *BaseECLListener) ExitNonfslash(ctx *NonfslashContext) {}

// EnterSp is called when production sp is entered.
func (s *BaseECLListener) EnterSp(ctx *SpContext) {}

// ExitSp is called when production sp is exited.
func (s *BaseECLListener) ExitSp(ctx *SpContext) {}

// EnterHtab is called when production htab is entered.
func (s *BaseECLListener) EnterHtab(ctx *HtabContext) {}

// ExitHtab is called when production htab is exited.
func (s *BaseECLListener) ExitHtab(ctx *HtabContext) {}

// EnterCr is called when production cr is entered.
func (s *BaseECLListener) EnterCr(ctx *CrContext) {}

// ExitCr is called when production cr is exited.
func (s *BaseECLListener) ExitCr(ctx *CrContext) {}

// EnterLf is called when production lf is entered.
func (s *BaseECLListener) EnterLf(ctx *LfContext) {}

// ExitLf is called when production lf is exited.
func (s *BaseECLListener) ExitLf(ctx *LfContext) {}

// EnterQm is called when production qm is entered.
func (s *BaseECLListener) EnterQm(ctx *QmContext) {}

// ExitQm is called when production qm is exited.
func (s *BaseECLListener) ExitQm(ctx *QmContext) {}

// EnterBs is called when production bs is entered.
func (s *BaseECLListener) EnterBs(ctx *BsContext) {}

// ExitBs is called when production bs is exited.
func (s *BaseECLListener) ExitBs(ctx *BsContext) {}

// EnterStar is called when production star is entered.
func (s *BaseECLListener) EnterStar(ctx *StarContext) {}

// ExitStar is called when production star is exited.
func (s *BaseECLListener) ExitStar(ctx *StarContext) {}

// EnterDigit is called when production digit is entered.
func (s *BaseECLListener) EnterDigit(ctx *DigitContext) {}

// ExitDigit is called when production digit is exited.
func (s *BaseECLListener) ExitDigit(ctx *DigitContext) {}

// EnterZero is called when production zero is entered.
func (s *BaseECLListener) EnterZero(ctx *ZeroContext) {}

// ExitZero is called when production zero is exited.
func (s *BaseECLListener) ExitZero(ctx *ZeroContext) {}

// EnterDigitnonzero is called when production digitnonzero is entered.
func (s *BaseECLListener) EnterDigitnonzero(ctx *DigitnonzeroContext) {}

// ExitDigitnonzero is called when production digitnonzero is exited.
func (s *BaseECLListener) ExitDigitnonzero(ctx *DigitnonzeroContext) {}

// EnterNonwsnonpipe is called when production nonwsnonpipe is entered.
func (s *BaseECLListener) EnterNonwsnonpipe(ctx *NonwsnonpipeContext) {}

// ExitNonwsnonpipe is called when production nonwsnonpipe is exited.
func (s *BaseECLListener) ExitNonwsnonpipe(ctx *NonwsnonpipeContext) {}

// EnterAnynonescapedchar is called when production anynonescapedchar is entered.
func (s *BaseECLListener) EnterAnynonescapedchar(ctx *AnynonescapedcharContext) {}

// ExitAnynonescapedchar is called when production anynonescapedchar is exited.
func (s *BaseECLListener) ExitAnynonescapedchar(ctx *AnynonescapedcharContext) {}

// EnterEscapedchar is called when production escapedchar is entered.
func (s *BaseECLListener) EnterEscapedchar(ctx *EscapedcharContext) {}

// ExitEscapedchar is called when production escapedchar is exited.
func (s *BaseECLListener) ExitEscapedchar(ctx *EscapedcharContext) {}

// EnterEscapedwildchar is called when production escapedwildchar is entered.
func (s *BaseECLListener) EnterEscapedwildchar(ctx *EscapedwildcharContext) {}

// ExitEscapedwildchar is called when production escapedwildchar is exited.
func (s *BaseECLListener) ExitEscapedwildchar(ctx *EscapedwildcharContext) {}

// EnterNonwsnonescapedchar is called when production nonwsnonescapedchar is entered.
func (s *BaseECLListener) EnterNonwsnonescapedchar(ctx *NonwsnonescapedcharContext) {}

// ExitNonwsnonescapedchar is called when production nonwsnonescapedchar is exited.
func (s *BaseECLListener) ExitNonwsnonescapedchar(ctx *NonwsnonescapedcharContext) {}

// EnterAlpha is called when production alpha is entered.
func (s *BaseECLListener) EnterAlpha(ctx *AlphaContext) {}

// ExitAlpha is called when production alpha is exited.
func (s *BaseECLListener) ExitAlpha(ctx *AlphaContext) {}

// EnterDash is called when production dash is entered.
func (s *BaseECLListener) EnterDash(ctx *DashContext) {}

// ExitDash is called when production dash is exited.
func (s *BaseECLListener) ExitDash(ctx *DashContext) {}
