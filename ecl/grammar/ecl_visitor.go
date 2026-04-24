// Code generated from ECL.g4 by ANTLR 4.13.2. DO NOT EDIT.

package grammar // ECL
import "github.com/antlr4-go/antlr/v4"

// A complete Visitor for a parse tree produced by ECLParser.
type ECLVisitor interface {
	antlr.ParseTreeVisitor

	// Visit a parse tree produced by ECLParser#expressionconstraint.
	VisitExpressionconstraint(ctx *ExpressionconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#refinedexpressionconstraint.
	VisitRefinedexpressionconstraint(ctx *RefinedexpressionconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#compoundexpressionconstraint.
	VisitCompoundexpressionconstraint(ctx *CompoundexpressionconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#conjunctionexpressionconstraint.
	VisitConjunctionexpressionconstraint(ctx *ConjunctionexpressionconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#disjunctionexpressionconstraint.
	VisitDisjunctionexpressionconstraint(ctx *DisjunctionexpressionconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#exclusionexpressionconstraint.
	VisitExclusionexpressionconstraint(ctx *ExclusionexpressionconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#dottedexpressionconstraint.
	VisitDottedexpressionconstraint(ctx *DottedexpressionconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#dottedexpressionattribute.
	VisitDottedexpressionattribute(ctx *DottedexpressionattributeContext) interface{}

	// Visit a parse tree produced by ECLParser#subexpressionconstraint.
	VisitSubexpressionconstraint(ctx *SubexpressionconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#eclfocusconcept.
	VisitEclfocusconcept(ctx *EclfocusconceptContext) interface{}

	// Visit a parse tree produced by ECLParser#dot.
	VisitDot(ctx *DotContext) interface{}

	// Visit a parse tree produced by ECLParser#refsetOperator.
	VisitRefsetOperator(ctx *RefsetOperatorContext) interface{}

	// Visit a parse tree produced by ECLParser#memberof.
	VisitMemberof(ctx *MemberofContext) interface{}

	// Visit a parse tree produced by ECLParser#refsetContainingAny.
	VisitRefsetContainingAny(ctx *RefsetContainingAnyContext) interface{}

	// Visit a parse tree produced by ECLParser#refsetfieldnameset.
	VisitRefsetfieldnameset(ctx *RefsetfieldnamesetContext) interface{}

	// Visit a parse tree produced by ECLParser#refsetfieldname.
	VisitRefsetfieldname(ctx *RefsetfieldnameContext) interface{}

	// Visit a parse tree produced by ECLParser#eclconceptreference.
	VisitEclconceptreference(ctx *EclconceptreferenceContext) interface{}

	// Visit a parse tree produced by ECLParser#eclconceptreferenceset.
	VisitEclconceptreferenceset(ctx *EclconceptreferencesetContext) interface{}

	// Visit a parse tree produced by ECLParser#conceptid.
	VisitConceptid(ctx *ConceptidContext) interface{}

	// Visit a parse tree produced by ECLParser#term.
	VisitTerm(ctx *TermContext) interface{}

	// Visit a parse tree produced by ECLParser#altidentifier.
	VisitAltidentifier(ctx *AltidentifierContext) interface{}

	// Visit a parse tree produced by ECLParser#altidentifierschemealias.
	VisitAltidentifierschemealias(ctx *AltidentifierschemealiasContext) interface{}

	// Visit a parse tree produced by ECLParser#altidentifiercodewithinquotes.
	VisitAltidentifiercodewithinquotes(ctx *AltidentifiercodewithinquotesContext) interface{}

	// Visit a parse tree produced by ECLParser#altidentifiercodewithoutquotes.
	VisitAltidentifiercodewithoutquotes(ctx *AltidentifiercodewithoutquotesContext) interface{}

	// Visit a parse tree produced by ECLParser#wildcard.
	VisitWildcard(ctx *WildcardContext) interface{}

	// Visit a parse tree produced by ECLParser#constraintoperator.
	VisitConstraintoperator(ctx *ConstraintoperatorContext) interface{}

	// Visit a parse tree produced by ECLParser#descendantof.
	VisitDescendantof(ctx *DescendantofContext) interface{}

	// Visit a parse tree produced by ECLParser#descendantorselfof.
	VisitDescendantorselfof(ctx *DescendantorselfofContext) interface{}

	// Visit a parse tree produced by ECLParser#childof.
	VisitChildof(ctx *ChildofContext) interface{}

	// Visit a parse tree produced by ECLParser#childorselfof.
	VisitChildorselfof(ctx *ChildorselfofContext) interface{}

	// Visit a parse tree produced by ECLParser#ancestorof.
	VisitAncestorof(ctx *AncestorofContext) interface{}

	// Visit a parse tree produced by ECLParser#ancestororselfof.
	VisitAncestororselfof(ctx *AncestororselfofContext) interface{}

	// Visit a parse tree produced by ECLParser#parentof.
	VisitParentof(ctx *ParentofContext) interface{}

	// Visit a parse tree produced by ECLParser#parentorselfof.
	VisitParentorselfof(ctx *ParentorselfofContext) interface{}

	// Visit a parse tree produced by ECLParser#top.
	VisitTop(ctx *TopContext) interface{}

	// Visit a parse tree produced by ECLParser#bottom.
	VisitBottom(ctx *BottomContext) interface{}

	// Visit a parse tree produced by ECLParser#conjunction.
	VisitConjunction(ctx *ConjunctionContext) interface{}

	// Visit a parse tree produced by ECLParser#disjunction.
	VisitDisjunction(ctx *DisjunctionContext) interface{}

	// Visit a parse tree produced by ECLParser#exclusion.
	VisitExclusion(ctx *ExclusionContext) interface{}

	// Visit a parse tree produced by ECLParser#eclrefinement.
	VisitEclrefinement(ctx *EclrefinementContext) interface{}

	// Visit a parse tree produced by ECLParser#conjunctionrefinementset.
	VisitConjunctionrefinementset(ctx *ConjunctionrefinementsetContext) interface{}

	// Visit a parse tree produced by ECLParser#disjunctionrefinementset.
	VisitDisjunctionrefinementset(ctx *DisjunctionrefinementsetContext) interface{}

	// Visit a parse tree produced by ECLParser#subrefinement.
	VisitSubrefinement(ctx *SubrefinementContext) interface{}

	// Visit a parse tree produced by ECLParser#eclattributeset.
	VisitEclattributeset(ctx *EclattributesetContext) interface{}

	// Visit a parse tree produced by ECLParser#conjunctionattributeset.
	VisitConjunctionattributeset(ctx *ConjunctionattributesetContext) interface{}

	// Visit a parse tree produced by ECLParser#disjunctionattributeset.
	VisitDisjunctionattributeset(ctx *DisjunctionattributesetContext) interface{}

	// Visit a parse tree produced by ECLParser#subattributeset.
	VisitSubattributeset(ctx *SubattributesetContext) interface{}

	// Visit a parse tree produced by ECLParser#eclattributegroup.
	VisitEclattributegroup(ctx *EclattributegroupContext) interface{}

	// Visit a parse tree produced by ECLParser#eclattribute.
	VisitEclattribute(ctx *EclattributeContext) interface{}

	// Visit a parse tree produced by ECLParser#cardinality.
	VisitCardinality(ctx *CardinalityContext) interface{}

	// Visit a parse tree produced by ECLParser#minvalue.
	VisitMinvalue(ctx *MinvalueContext) interface{}

	// Visit a parse tree produced by ECLParser#to.
	VisitTo(ctx *ToContext) interface{}

	// Visit a parse tree produced by ECLParser#maxvalue.
	VisitMaxvalue(ctx *MaxvalueContext) interface{}

	// Visit a parse tree produced by ECLParser#many.
	VisitMany(ctx *ManyContext) interface{}

	// Visit a parse tree produced by ECLParser#reverseflag.
	VisitReverseflag(ctx *ReverseflagContext) interface{}

	// Visit a parse tree produced by ECLParser#eclattributename.
	VisitEclattributename(ctx *EclattributenameContext) interface{}

	// Visit a parse tree produced by ECLParser#expressioncomparisonoperator.
	VisitExpressioncomparisonoperator(ctx *ExpressioncomparisonoperatorContext) interface{}

	// Visit a parse tree produced by ECLParser#numericcomparisonoperator.
	VisitNumericcomparisonoperator(ctx *NumericcomparisonoperatorContext) interface{}

	// Visit a parse tree produced by ECLParser#timecomparisonoperator.
	VisitTimecomparisonoperator(ctx *TimecomparisonoperatorContext) interface{}

	// Visit a parse tree produced by ECLParser#stringcomparisonoperator.
	VisitStringcomparisonoperator(ctx *StringcomparisonoperatorContext) interface{}

	// Visit a parse tree produced by ECLParser#booleancomparisonoperator.
	VisitBooleancomparisonoperator(ctx *BooleancomparisonoperatorContext) interface{}

	// Visit a parse tree produced by ECLParser#idcomparisonoperator.
	VisitIdcomparisonoperator(ctx *IdcomparisonoperatorContext) interface{}

	// Visit a parse tree produced by ECLParser#descriptionfilterconstraint.
	VisitDescriptionfilterconstraint(ctx *DescriptionfilterconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#descriptionfilter.
	VisitDescriptionfilter(ctx *DescriptionfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#descriptionidfilter.
	VisitDescriptionidfilter(ctx *DescriptionidfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#descriptionidkeyword.
	VisitDescriptionidkeyword(ctx *DescriptionidkeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#descriptionid.
	VisitDescriptionid(ctx *DescriptionidContext) interface{}

	// Visit a parse tree produced by ECLParser#descriptionidset.
	VisitDescriptionidset(ctx *DescriptionidsetContext) interface{}

	// Visit a parse tree produced by ECLParser#termfilter.
	VisitTermfilter(ctx *TermfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#termkeyword.
	VisitTermkeyword(ctx *TermkeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#typedsearchterm.
	VisitTypedsearchterm(ctx *TypedsearchtermContext) interface{}

	// Visit a parse tree produced by ECLParser#typedsearchtermset.
	VisitTypedsearchtermset(ctx *TypedsearchtermsetContext) interface{}

	// Visit a parse tree produced by ECLParser#concretestringset.
	VisitConcretestringset(ctx *ConcretestringsetContext) interface{}

	// Visit a parse tree produced by ECLParser#concretestring.
	VisitConcretestring(ctx *ConcretestringContext) interface{}

	// Visit a parse tree produced by ECLParser#concretestringcharacters.
	VisitConcretestringcharacters(ctx *ConcretestringcharactersContext) interface{}

	// Visit a parse tree produced by ECLParser#wild.
	VisitWild(ctx *WildContext) interface{}

	// Visit a parse tree produced by ECLParser#matchkeyword.
	VisitMatchkeyword(ctx *MatchkeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#matchsearchterm.
	VisitMatchsearchterm(ctx *MatchsearchtermContext) interface{}

	// Visit a parse tree produced by ECLParser#matchsearchtermset.
	VisitMatchsearchtermset(ctx *MatchsearchtermsetContext) interface{}

	// Visit a parse tree produced by ECLParser#wildsearchterm.
	VisitWildsearchterm(ctx *WildsearchtermContext) interface{}

	// Visit a parse tree produced by ECLParser#wildsearchtermset.
	VisitWildsearchtermset(ctx *WildsearchtermsetContext) interface{}

	// Visit a parse tree produced by ECLParser#languagefilter.
	VisitLanguagefilter(ctx *LanguagefilterContext) interface{}

	// Visit a parse tree produced by ECLParser#language.
	VisitLanguage(ctx *LanguageContext) interface{}

	// Visit a parse tree produced by ECLParser#languagecode.
	VisitLanguagecode(ctx *LanguagecodeContext) interface{}

	// Visit a parse tree produced by ECLParser#languagecodeset.
	VisitLanguagecodeset(ctx *LanguagecodesetContext) interface{}

	// Visit a parse tree produced by ECLParser#typefilter.
	VisitTypefilter(ctx *TypefilterContext) interface{}

	// Visit a parse tree produced by ECLParser#typeidfilter.
	VisitTypeidfilter(ctx *TypeidfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#typeid.
	VisitTypeid(ctx *TypeidContext) interface{}

	// Visit a parse tree produced by ECLParser#typetokenfilter.
	VisitTypetokenfilter(ctx *TypetokenfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#type.
	VisitType(ctx *TypeContext) interface{}

	// Visit a parse tree produced by ECLParser#typetoken.
	VisitTypetoken(ctx *TypetokenContext) interface{}

	// Visit a parse tree produced by ECLParser#typetokenset.
	VisitTypetokenset(ctx *TypetokensetContext) interface{}

	// Visit a parse tree produced by ECLParser#synonym.
	VisitSynonym(ctx *SynonymContext) interface{}

	// Visit a parse tree produced by ECLParser#fullyspecifiedname.
	VisitFullyspecifiedname(ctx *FullyspecifiednameContext) interface{}

	// Visit a parse tree produced by ECLParser#definition.
	VisitDefinition(ctx *DefinitionContext) interface{}

	// Visit a parse tree produced by ECLParser#dialectfilter.
	VisitDialectfilter(ctx *DialectfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#dialectidfilter.
	VisitDialectidfilter(ctx *DialectidfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#dialectid.
	VisitDialectid(ctx *DialectidContext) interface{}

	// Visit a parse tree produced by ECLParser#dialectaliasfilter.
	VisitDialectaliasfilter(ctx *DialectaliasfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#dialect.
	VisitDialect(ctx *DialectContext) interface{}

	// Visit a parse tree produced by ECLParser#dialectalias.
	VisitDialectalias(ctx *DialectaliasContext) interface{}

	// Visit a parse tree produced by ECLParser#dialectaliasset.
	VisitDialectaliasset(ctx *DialectaliassetContext) interface{}

	// Visit a parse tree produced by ECLParser#dialectidset.
	VisitDialectidset(ctx *DialectidsetContext) interface{}

	// Visit a parse tree produced by ECLParser#acceptabilityset.
	VisitAcceptabilityset(ctx *AcceptabilitysetContext) interface{}

	// Visit a parse tree produced by ECLParser#acceptabilityconceptreferenceset.
	VisitAcceptabilityconceptreferenceset(ctx *AcceptabilityconceptreferencesetContext) interface{}

	// Visit a parse tree produced by ECLParser#acceptabilitytokenset.
	VisitAcceptabilitytokenset(ctx *AcceptabilitytokensetContext) interface{}

	// Visit a parse tree produced by ECLParser#acceptabilitytoken.
	VisitAcceptabilitytoken(ctx *AcceptabilitytokenContext) interface{}

	// Visit a parse tree produced by ECLParser#acceptable.
	VisitAcceptable(ctx *AcceptableContext) interface{}

	// Visit a parse tree produced by ECLParser#preferred.
	VisitPreferred(ctx *PreferredContext) interface{}

	// Visit a parse tree produced by ECLParser#conceptfilterconstraint.
	VisitConceptfilterconstraint(ctx *ConceptfilterconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#conceptfilter.
	VisitConceptfilter(ctx *ConceptfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#definitionstatusfilter.
	VisitDefinitionstatusfilter(ctx *DefinitionstatusfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#definitionstatusidfilter.
	VisitDefinitionstatusidfilter(ctx *DefinitionstatusidfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#definitionstatusidkeyword.
	VisitDefinitionstatusidkeyword(ctx *DefinitionstatusidkeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#definitionstatustokenfilter.
	VisitDefinitionstatustokenfilter(ctx *DefinitionstatustokenfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#definitionstatuskeyword.
	VisitDefinitionstatuskeyword(ctx *DefinitionstatuskeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#definitionstatustoken.
	VisitDefinitionstatustoken(ctx *DefinitionstatustokenContext) interface{}

	// Visit a parse tree produced by ECLParser#definitionstatustokenset.
	VisitDefinitionstatustokenset(ctx *DefinitionstatustokensetContext) interface{}

	// Visit a parse tree produced by ECLParser#primitivetoken.
	VisitPrimitivetoken(ctx *PrimitivetokenContext) interface{}

	// Visit a parse tree produced by ECLParser#definedtoken.
	VisitDefinedtoken(ctx *DefinedtokenContext) interface{}

	// Visit a parse tree produced by ECLParser#modulefilter.
	VisitModulefilter(ctx *ModulefilterContext) interface{}

	// Visit a parse tree produced by ECLParser#moduleidkeyword.
	VisitModuleidkeyword(ctx *ModuleidkeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#effectivetimefilter.
	VisitEffectivetimefilter(ctx *EffectivetimefilterContext) interface{}

	// Visit a parse tree produced by ECLParser#effectivetimekeyword.
	VisitEffectivetimekeyword(ctx *EffectivetimekeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#timevalue.
	VisitTimevalue(ctx *TimevalueContext) interface{}

	// Visit a parse tree produced by ECLParser#timevalueset.
	VisitTimevalueset(ctx *TimevaluesetContext) interface{}

	// Visit a parse tree produced by ECLParser#year.
	VisitYear(ctx *YearContext) interface{}

	// Visit a parse tree produced by ECLParser#month.
	VisitMonth(ctx *MonthContext) interface{}

	// Visit a parse tree produced by ECLParser#day.
	VisitDay(ctx *DayContext) interface{}

	// Visit a parse tree produced by ECLParser#activefilter.
	VisitActivefilter(ctx *ActivefilterContext) interface{}

	// Visit a parse tree produced by ECLParser#activekeyword.
	VisitActivekeyword(ctx *ActivekeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#activevalue.
	VisitActivevalue(ctx *ActivevalueContext) interface{}

	// Visit a parse tree produced by ECLParser#activetruevalue.
	VisitActivetruevalue(ctx *ActivetruevalueContext) interface{}

	// Visit a parse tree produced by ECLParser#activefalsevalue.
	VisitActivefalsevalue(ctx *ActivefalsevalueContext) interface{}

	// Visit a parse tree produced by ECLParser#memberfilterconstraint.
	VisitMemberfilterconstraint(ctx *MemberfilterconstraintContext) interface{}

	// Visit a parse tree produced by ECLParser#memberfilter.
	VisitMemberfilter(ctx *MemberfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#memberfieldfilter.
	VisitMemberfieldfilter(ctx *MemberfieldfilterContext) interface{}

	// Visit a parse tree produced by ECLParser#historysupplement.
	VisitHistorysupplement(ctx *HistorysupplementContext) interface{}

	// Visit a parse tree produced by ECLParser#historykeyword.
	VisitHistorykeyword(ctx *HistorykeywordContext) interface{}

	// Visit a parse tree produced by ECLParser#historyprofilesuffix.
	VisitHistoryprofilesuffix(ctx *HistoryprofilesuffixContext) interface{}

	// Visit a parse tree produced by ECLParser#historyminimumsuffix.
	VisitHistoryminimumsuffix(ctx *HistoryminimumsuffixContext) interface{}

	// Visit a parse tree produced by ECLParser#historymoderatesuffix.
	VisitHistorymoderatesuffix(ctx *HistorymoderatesuffixContext) interface{}

	// Visit a parse tree produced by ECLParser#historymaximumsuffix.
	VisitHistorymaximumsuffix(ctx *HistorymaximumsuffixContext) interface{}

	// Visit a parse tree produced by ECLParser#historysubset.
	VisitHistorysubset(ctx *HistorysubsetContext) interface{}

	// Visit a parse tree produced by ECLParser#numericvalue.
	VisitNumericvalue(ctx *NumericvalueContext) interface{}

	// Visit a parse tree produced by ECLParser#stringvalue.
	VisitStringvalue(ctx *StringvalueContext) interface{}

	// Visit a parse tree produced by ECLParser#integervalue.
	VisitIntegervalue(ctx *IntegervalueContext) interface{}

	// Visit a parse tree produced by ECLParser#decimalvalue.
	VisitDecimalvalue(ctx *DecimalvalueContext) interface{}

	// Visit a parse tree produced by ECLParser#booleanvalue.
	VisitBooleanvalue(ctx *BooleanvalueContext) interface{}

	// Visit a parse tree produced by ECLParser#true_1.
	VisitTrue_1(ctx *True_1Context) interface{}

	// Visit a parse tree produced by ECLParser#false_1.
	VisitFalse_1(ctx *False_1Context) interface{}

	// Visit a parse tree produced by ECLParser#nonnegativeintegervalue.
	VisitNonnegativeintegervalue(ctx *NonnegativeintegervalueContext) interface{}

	// Visit a parse tree produced by ECLParser#sctid.
	VisitSctid(ctx *SctidContext) interface{}

	// Visit a parse tree produced by ECLParser#ws.
	VisitWs(ctx *WsContext) interface{}

	// Visit a parse tree produced by ECLParser#mws.
	VisitMws(ctx *MwsContext) interface{}

	// Visit a parse tree produced by ECLParser#comment.
	VisitComment(ctx *CommentContext) interface{}

	// Visit a parse tree produced by ECLParser#nonstarchar.
	VisitNonstarchar(ctx *NonstarcharContext) interface{}

	// Visit a parse tree produced by ECLParser#starwithnonfslash.
	VisitStarwithnonfslash(ctx *StarwithnonfslashContext) interface{}

	// Visit a parse tree produced by ECLParser#nonfslash.
	VisitNonfslash(ctx *NonfslashContext) interface{}

	// Visit a parse tree produced by ECLParser#sp.
	VisitSp(ctx *SpContext) interface{}

	// Visit a parse tree produced by ECLParser#htab.
	VisitHtab(ctx *HtabContext) interface{}

	// Visit a parse tree produced by ECLParser#cr.
	VisitCr(ctx *CrContext) interface{}

	// Visit a parse tree produced by ECLParser#lf.
	VisitLf(ctx *LfContext) interface{}

	// Visit a parse tree produced by ECLParser#qm.
	VisitQm(ctx *QmContext) interface{}

	// Visit a parse tree produced by ECLParser#bs.
	VisitBs(ctx *BsContext) interface{}

	// Visit a parse tree produced by ECLParser#star.
	VisitStar(ctx *StarContext) interface{}

	// Visit a parse tree produced by ECLParser#digit.
	VisitDigit(ctx *DigitContext) interface{}

	// Visit a parse tree produced by ECLParser#zero.
	VisitZero(ctx *ZeroContext) interface{}

	// Visit a parse tree produced by ECLParser#digitnonzero.
	VisitDigitnonzero(ctx *DigitnonzeroContext) interface{}

	// Visit a parse tree produced by ECLParser#nonwsnonpipe.
	VisitNonwsnonpipe(ctx *NonwsnonpipeContext) interface{}

	// Visit a parse tree produced by ECLParser#anynonescapedchar.
	VisitAnynonescapedchar(ctx *AnynonescapedcharContext) interface{}

	// Visit a parse tree produced by ECLParser#escapedchar.
	VisitEscapedchar(ctx *EscapedcharContext) interface{}

	// Visit a parse tree produced by ECLParser#escapedwildchar.
	VisitEscapedwildchar(ctx *EscapedwildcharContext) interface{}

	// Visit a parse tree produced by ECLParser#nonwsnonescapedchar.
	VisitNonwsnonescapedchar(ctx *NonwsnonescapedcharContext) interface{}

	// Visit a parse tree produced by ECLParser#alpha.
	VisitAlpha(ctx *AlphaContext) interface{}

	// Visit a parse tree produced by ECLParser#dash.
	VisitDash(ctx *DashContext) interface{}
}
