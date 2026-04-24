// Code generated from ECL.g4 by ANTLR 4.13.2. DO NOT EDIT.

package grammar // ECL
import "github.com/antlr4-go/antlr/v4"

type BaseECLVisitor struct {
	*antlr.BaseParseTreeVisitor
}

func (v *BaseECLVisitor) VisitExpressionconstraint(ctx *ExpressionconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitRefinedexpressionconstraint(ctx *RefinedexpressionconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitCompoundexpressionconstraint(ctx *CompoundexpressionconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConjunctionexpressionconstraint(ctx *ConjunctionexpressionconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDisjunctionexpressionconstraint(ctx *DisjunctionexpressionconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitExclusionexpressionconstraint(ctx *ExclusionexpressionconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDottedexpressionconstraint(ctx *DottedexpressionconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDottedexpressionattribute(ctx *DottedexpressionattributeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitSubexpressionconstraint(ctx *SubexpressionconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEclfocusconcept(ctx *EclfocusconceptContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDot(ctx *DotContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitRefsetOperator(ctx *RefsetOperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMemberof(ctx *MemberofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitRefsetContainingAny(ctx *RefsetContainingAnyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitRefsetfieldnameset(ctx *RefsetfieldnamesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitRefsetfieldname(ctx *RefsetfieldnameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEclconceptreference(ctx *EclconceptreferenceContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEclconceptreferenceset(ctx *EclconceptreferencesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConceptid(ctx *ConceptidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTerm(ctx *TermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAltidentifier(ctx *AltidentifierContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAltidentifierschemealias(ctx *AltidentifierschemealiasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAltidentifiercodewithinquotes(ctx *AltidentifiercodewithinquotesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAltidentifiercodewithoutquotes(ctx *AltidentifiercodewithoutquotesContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitWildcard(ctx *WildcardContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConstraintoperator(ctx *ConstraintoperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDescendantof(ctx *DescendantofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDescendantorselfof(ctx *DescendantorselfofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitChildof(ctx *ChildofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitChildorselfof(ctx *ChildorselfofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAncestorof(ctx *AncestorofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAncestororselfof(ctx *AncestororselfofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitParentof(ctx *ParentofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitParentorselfof(ctx *ParentorselfofContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTop(ctx *TopContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitBottom(ctx *BottomContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConjunction(ctx *ConjunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDisjunction(ctx *DisjunctionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitExclusion(ctx *ExclusionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEclrefinement(ctx *EclrefinementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConjunctionrefinementset(ctx *ConjunctionrefinementsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDisjunctionrefinementset(ctx *DisjunctionrefinementsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitSubrefinement(ctx *SubrefinementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEclattributeset(ctx *EclattributesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConjunctionattributeset(ctx *ConjunctionattributesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDisjunctionattributeset(ctx *DisjunctionattributesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitSubattributeset(ctx *SubattributesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEclattributegroup(ctx *EclattributegroupContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEclattribute(ctx *EclattributeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitCardinality(ctx *CardinalityContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMinvalue(ctx *MinvalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTo(ctx *ToContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMaxvalue(ctx *MaxvalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMany(ctx *ManyContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitReverseflag(ctx *ReverseflagContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEclattributename(ctx *EclattributenameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitExpressioncomparisonoperator(ctx *ExpressioncomparisonoperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitNumericcomparisonoperator(ctx *NumericcomparisonoperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTimecomparisonoperator(ctx *TimecomparisonoperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitStringcomparisonoperator(ctx *StringcomparisonoperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitBooleancomparisonoperator(ctx *BooleancomparisonoperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitIdcomparisonoperator(ctx *IdcomparisonoperatorContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDescriptionfilterconstraint(ctx *DescriptionfilterconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDescriptionfilter(ctx *DescriptionfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDescriptionidfilter(ctx *DescriptionidfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDescriptionidkeyword(ctx *DescriptionidkeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDescriptionid(ctx *DescriptionidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDescriptionidset(ctx *DescriptionidsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTermfilter(ctx *TermfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTermkeyword(ctx *TermkeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTypedsearchterm(ctx *TypedsearchtermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTypedsearchtermset(ctx *TypedsearchtermsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConcretestringset(ctx *ConcretestringsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConcretestring(ctx *ConcretestringContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConcretestringcharacters(ctx *ConcretestringcharactersContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitWild(ctx *WildContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMatchkeyword(ctx *MatchkeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMatchsearchterm(ctx *MatchsearchtermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMatchsearchtermset(ctx *MatchsearchtermsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitWildsearchterm(ctx *WildsearchtermContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitWildsearchtermset(ctx *WildsearchtermsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitLanguagefilter(ctx *LanguagefilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitLanguage(ctx *LanguageContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitLanguagecode(ctx *LanguagecodeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitLanguagecodeset(ctx *LanguagecodesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTypefilter(ctx *TypefilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTypeidfilter(ctx *TypeidfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTypeid(ctx *TypeidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTypetokenfilter(ctx *TypetokenfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitType(ctx *TypeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTypetoken(ctx *TypetokenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTypetokenset(ctx *TypetokensetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitSynonym(ctx *SynonymContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitFullyspecifiedname(ctx *FullyspecifiednameContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinition(ctx *DefinitionContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDialectfilter(ctx *DialectfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDialectidfilter(ctx *DialectidfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDialectid(ctx *DialectidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDialectaliasfilter(ctx *DialectaliasfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDialect(ctx *DialectContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDialectalias(ctx *DialectaliasContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDialectaliasset(ctx *DialectaliassetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDialectidset(ctx *DialectidsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAcceptabilityset(ctx *AcceptabilitysetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAcceptabilityconceptreferenceset(ctx *AcceptabilityconceptreferencesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAcceptabilitytokenset(ctx *AcceptabilitytokensetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAcceptabilitytoken(ctx *AcceptabilitytokenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAcceptable(ctx *AcceptableContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitPreferred(ctx *PreferredContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConceptfilterconstraint(ctx *ConceptfilterconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitConceptfilter(ctx *ConceptfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinitionstatusfilter(ctx *DefinitionstatusfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinitionstatusidfilter(ctx *DefinitionstatusidfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinitionstatusidkeyword(ctx *DefinitionstatusidkeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinitionstatustokenfilter(ctx *DefinitionstatustokenfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinitionstatuskeyword(ctx *DefinitionstatuskeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinitionstatustoken(ctx *DefinitionstatustokenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinitionstatustokenset(ctx *DefinitionstatustokensetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitPrimitivetoken(ctx *PrimitivetokenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDefinedtoken(ctx *DefinedtokenContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitModulefilter(ctx *ModulefilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitModuleidkeyword(ctx *ModuleidkeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEffectivetimefilter(ctx *EffectivetimefilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEffectivetimekeyword(ctx *EffectivetimekeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTimevalue(ctx *TimevalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTimevalueset(ctx *TimevaluesetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitYear(ctx *YearContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMonth(ctx *MonthContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDay(ctx *DayContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitActivefilter(ctx *ActivefilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitActivekeyword(ctx *ActivekeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitActivevalue(ctx *ActivevalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitActivetruevalue(ctx *ActivetruevalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitActivefalsevalue(ctx *ActivefalsevalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMemberfilterconstraint(ctx *MemberfilterconstraintContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMemberfilter(ctx *MemberfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMemberfieldfilter(ctx *MemberfieldfilterContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitHistorysupplement(ctx *HistorysupplementContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitHistorykeyword(ctx *HistorykeywordContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitHistoryprofilesuffix(ctx *HistoryprofilesuffixContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitHistoryminimumsuffix(ctx *HistoryminimumsuffixContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitHistorymoderatesuffix(ctx *HistorymoderatesuffixContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitHistorymaximumsuffix(ctx *HistorymaximumsuffixContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitHistorysubset(ctx *HistorysubsetContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitNumericvalue(ctx *NumericvalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitStringvalue(ctx *StringvalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitIntegervalue(ctx *IntegervalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDecimalvalue(ctx *DecimalvalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitBooleanvalue(ctx *BooleanvalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitTrue_1(ctx *True_1Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitFalse_1(ctx *False_1Context) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitNonnegativeintegervalue(ctx *NonnegativeintegervalueContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitSctid(ctx *SctidContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitWs(ctx *WsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitMws(ctx *MwsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitComment(ctx *CommentContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitNonstarchar(ctx *NonstarcharContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitStarwithnonfslash(ctx *StarwithnonfslashContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitNonfslash(ctx *NonfslashContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitSp(ctx *SpContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitHtab(ctx *HtabContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitCr(ctx *CrContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitLf(ctx *LfContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitQm(ctx *QmContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitBs(ctx *BsContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitStar(ctx *StarContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDigit(ctx *DigitContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitZero(ctx *ZeroContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDigitnonzero(ctx *DigitnonzeroContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitNonwsnonpipe(ctx *NonwsnonpipeContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAnynonescapedchar(ctx *AnynonescapedcharContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEscapedchar(ctx *EscapedcharContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitEscapedwildchar(ctx *EscapedwildcharContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitNonwsnonescapedchar(ctx *NonwsnonescapedcharContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitAlpha(ctx *AlphaContext) interface{} {
	return v.VisitChildren(ctx)
}

func (v *BaseECLVisitor) VisitDash(ctx *DashContext) interface{} {
	return v.VisitChildren(ctx)
}
