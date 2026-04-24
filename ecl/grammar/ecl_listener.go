// Code generated from ECL.g4 by ANTLR 4.13.2. DO NOT EDIT.

package grammar // ECL
import "github.com/antlr4-go/antlr/v4"

// ECLListener is a complete listener for a parse tree produced by ECLParser.
type ECLListener interface {
	antlr.ParseTreeListener

	// EnterExpressionconstraint is called when entering the expressionconstraint production.
	EnterExpressionconstraint(c *ExpressionconstraintContext)

	// EnterRefinedexpressionconstraint is called when entering the refinedexpressionconstraint production.
	EnterRefinedexpressionconstraint(c *RefinedexpressionconstraintContext)

	// EnterCompoundexpressionconstraint is called when entering the compoundexpressionconstraint production.
	EnterCompoundexpressionconstraint(c *CompoundexpressionconstraintContext)

	// EnterConjunctionexpressionconstraint is called when entering the conjunctionexpressionconstraint production.
	EnterConjunctionexpressionconstraint(c *ConjunctionexpressionconstraintContext)

	// EnterDisjunctionexpressionconstraint is called when entering the disjunctionexpressionconstraint production.
	EnterDisjunctionexpressionconstraint(c *DisjunctionexpressionconstraintContext)

	// EnterExclusionexpressionconstraint is called when entering the exclusionexpressionconstraint production.
	EnterExclusionexpressionconstraint(c *ExclusionexpressionconstraintContext)

	// EnterDottedexpressionconstraint is called when entering the dottedexpressionconstraint production.
	EnterDottedexpressionconstraint(c *DottedexpressionconstraintContext)

	// EnterDottedexpressionattribute is called when entering the dottedexpressionattribute production.
	EnterDottedexpressionattribute(c *DottedexpressionattributeContext)

	// EnterSubexpressionconstraint is called when entering the subexpressionconstraint production.
	EnterSubexpressionconstraint(c *SubexpressionconstraintContext)

	// EnterEclfocusconcept is called when entering the eclfocusconcept production.
	EnterEclfocusconcept(c *EclfocusconceptContext)

	// EnterDot is called when entering the dot production.
	EnterDot(c *DotContext)

	// EnterRefsetOperator is called when entering the refsetOperator production.
	EnterRefsetOperator(c *RefsetOperatorContext)

	// EnterMemberof is called when entering the memberof production.
	EnterMemberof(c *MemberofContext)

	// EnterRefsetContainingAny is called when entering the refsetContainingAny production.
	EnterRefsetContainingAny(c *RefsetContainingAnyContext)

	// EnterRefsetfieldnameset is called when entering the refsetfieldnameset production.
	EnterRefsetfieldnameset(c *RefsetfieldnamesetContext)

	// EnterRefsetfieldname is called when entering the refsetfieldname production.
	EnterRefsetfieldname(c *RefsetfieldnameContext)

	// EnterEclconceptreference is called when entering the eclconceptreference production.
	EnterEclconceptreference(c *EclconceptreferenceContext)

	// EnterEclconceptreferenceset is called when entering the eclconceptreferenceset production.
	EnterEclconceptreferenceset(c *EclconceptreferencesetContext)

	// EnterConceptid is called when entering the conceptid production.
	EnterConceptid(c *ConceptidContext)

	// EnterTerm is called when entering the term production.
	EnterTerm(c *TermContext)

	// EnterAltidentifier is called when entering the altidentifier production.
	EnterAltidentifier(c *AltidentifierContext)

	// EnterAltidentifierschemealias is called when entering the altidentifierschemealias production.
	EnterAltidentifierschemealias(c *AltidentifierschemealiasContext)

	// EnterAltidentifiercodewithinquotes is called when entering the altidentifiercodewithinquotes production.
	EnterAltidentifiercodewithinquotes(c *AltidentifiercodewithinquotesContext)

	// EnterAltidentifiercodewithoutquotes is called when entering the altidentifiercodewithoutquotes production.
	EnterAltidentifiercodewithoutquotes(c *AltidentifiercodewithoutquotesContext)

	// EnterWildcard is called when entering the wildcard production.
	EnterWildcard(c *WildcardContext)

	// EnterConstraintoperator is called when entering the constraintoperator production.
	EnterConstraintoperator(c *ConstraintoperatorContext)

	// EnterDescendantof is called when entering the descendantof production.
	EnterDescendantof(c *DescendantofContext)

	// EnterDescendantorselfof is called when entering the descendantorselfof production.
	EnterDescendantorselfof(c *DescendantorselfofContext)

	// EnterChildof is called when entering the childof production.
	EnterChildof(c *ChildofContext)

	// EnterChildorselfof is called when entering the childorselfof production.
	EnterChildorselfof(c *ChildorselfofContext)

	// EnterAncestorof is called when entering the ancestorof production.
	EnterAncestorof(c *AncestorofContext)

	// EnterAncestororselfof is called when entering the ancestororselfof production.
	EnterAncestororselfof(c *AncestororselfofContext)

	// EnterParentof is called when entering the parentof production.
	EnterParentof(c *ParentofContext)

	// EnterParentorselfof is called when entering the parentorselfof production.
	EnterParentorselfof(c *ParentorselfofContext)

	// EnterTop is called when entering the top production.
	EnterTop(c *TopContext)

	// EnterBottom is called when entering the bottom production.
	EnterBottom(c *BottomContext)

	// EnterConjunction is called when entering the conjunction production.
	EnterConjunction(c *ConjunctionContext)

	// EnterDisjunction is called when entering the disjunction production.
	EnterDisjunction(c *DisjunctionContext)

	// EnterExclusion is called when entering the exclusion production.
	EnterExclusion(c *ExclusionContext)

	// EnterEclrefinement is called when entering the eclrefinement production.
	EnterEclrefinement(c *EclrefinementContext)

	// EnterConjunctionrefinementset is called when entering the conjunctionrefinementset production.
	EnterConjunctionrefinementset(c *ConjunctionrefinementsetContext)

	// EnterDisjunctionrefinementset is called when entering the disjunctionrefinementset production.
	EnterDisjunctionrefinementset(c *DisjunctionrefinementsetContext)

	// EnterSubrefinement is called when entering the subrefinement production.
	EnterSubrefinement(c *SubrefinementContext)

	// EnterEclattributeset is called when entering the eclattributeset production.
	EnterEclattributeset(c *EclattributesetContext)

	// EnterConjunctionattributeset is called when entering the conjunctionattributeset production.
	EnterConjunctionattributeset(c *ConjunctionattributesetContext)

	// EnterDisjunctionattributeset is called when entering the disjunctionattributeset production.
	EnterDisjunctionattributeset(c *DisjunctionattributesetContext)

	// EnterSubattributeset is called when entering the subattributeset production.
	EnterSubattributeset(c *SubattributesetContext)

	// EnterEclattributegroup is called when entering the eclattributegroup production.
	EnterEclattributegroup(c *EclattributegroupContext)

	// EnterEclattribute is called when entering the eclattribute production.
	EnterEclattribute(c *EclattributeContext)

	// EnterCardinality is called when entering the cardinality production.
	EnterCardinality(c *CardinalityContext)

	// EnterMinvalue is called when entering the minvalue production.
	EnterMinvalue(c *MinvalueContext)

	// EnterTo is called when entering the to production.
	EnterTo(c *ToContext)

	// EnterMaxvalue is called when entering the maxvalue production.
	EnterMaxvalue(c *MaxvalueContext)

	// EnterMany is called when entering the many production.
	EnterMany(c *ManyContext)

	// EnterReverseflag is called when entering the reverseflag production.
	EnterReverseflag(c *ReverseflagContext)

	// EnterEclattributename is called when entering the eclattributename production.
	EnterEclattributename(c *EclattributenameContext)

	// EnterExpressioncomparisonoperator is called when entering the expressioncomparisonoperator production.
	EnterExpressioncomparisonoperator(c *ExpressioncomparisonoperatorContext)

	// EnterNumericcomparisonoperator is called when entering the numericcomparisonoperator production.
	EnterNumericcomparisonoperator(c *NumericcomparisonoperatorContext)

	// EnterTimecomparisonoperator is called when entering the timecomparisonoperator production.
	EnterTimecomparisonoperator(c *TimecomparisonoperatorContext)

	// EnterStringcomparisonoperator is called when entering the stringcomparisonoperator production.
	EnterStringcomparisonoperator(c *StringcomparisonoperatorContext)

	// EnterBooleancomparisonoperator is called when entering the booleancomparisonoperator production.
	EnterBooleancomparisonoperator(c *BooleancomparisonoperatorContext)

	// EnterIdcomparisonoperator is called when entering the idcomparisonoperator production.
	EnterIdcomparisonoperator(c *IdcomparisonoperatorContext)

	// EnterDescriptionfilterconstraint is called when entering the descriptionfilterconstraint production.
	EnterDescriptionfilterconstraint(c *DescriptionfilterconstraintContext)

	// EnterDescriptionfilter is called when entering the descriptionfilter production.
	EnterDescriptionfilter(c *DescriptionfilterContext)

	// EnterDescriptionidfilter is called when entering the descriptionidfilter production.
	EnterDescriptionidfilter(c *DescriptionidfilterContext)

	// EnterDescriptionidkeyword is called when entering the descriptionidkeyword production.
	EnterDescriptionidkeyword(c *DescriptionidkeywordContext)

	// EnterDescriptionid is called when entering the descriptionid production.
	EnterDescriptionid(c *DescriptionidContext)

	// EnterDescriptionidset is called when entering the descriptionidset production.
	EnterDescriptionidset(c *DescriptionidsetContext)

	// EnterTermfilter is called when entering the termfilter production.
	EnterTermfilter(c *TermfilterContext)

	// EnterTermkeyword is called when entering the termkeyword production.
	EnterTermkeyword(c *TermkeywordContext)

	// EnterTypedsearchterm is called when entering the typedsearchterm production.
	EnterTypedsearchterm(c *TypedsearchtermContext)

	// EnterTypedsearchtermset is called when entering the typedsearchtermset production.
	EnterTypedsearchtermset(c *TypedsearchtermsetContext)

	// EnterConcretestringset is called when entering the concretestringset production.
	EnterConcretestringset(c *ConcretestringsetContext)

	// EnterConcretestring is called when entering the concretestring production.
	EnterConcretestring(c *ConcretestringContext)

	// EnterConcretestringcharacters is called when entering the concretestringcharacters production.
	EnterConcretestringcharacters(c *ConcretestringcharactersContext)

	// EnterWild is called when entering the wild production.
	EnterWild(c *WildContext)

	// EnterMatchkeyword is called when entering the matchkeyword production.
	EnterMatchkeyword(c *MatchkeywordContext)

	// EnterMatchsearchterm is called when entering the matchsearchterm production.
	EnterMatchsearchterm(c *MatchsearchtermContext)

	// EnterMatchsearchtermset is called when entering the matchsearchtermset production.
	EnterMatchsearchtermset(c *MatchsearchtermsetContext)

	// EnterWildsearchterm is called when entering the wildsearchterm production.
	EnterWildsearchterm(c *WildsearchtermContext)

	// EnterWildsearchtermset is called when entering the wildsearchtermset production.
	EnterWildsearchtermset(c *WildsearchtermsetContext)

	// EnterLanguagefilter is called when entering the languagefilter production.
	EnterLanguagefilter(c *LanguagefilterContext)

	// EnterLanguage is called when entering the language production.
	EnterLanguage(c *LanguageContext)

	// EnterLanguagecode is called when entering the languagecode production.
	EnterLanguagecode(c *LanguagecodeContext)

	// EnterLanguagecodeset is called when entering the languagecodeset production.
	EnterLanguagecodeset(c *LanguagecodesetContext)

	// EnterTypefilter is called when entering the typefilter production.
	EnterTypefilter(c *TypefilterContext)

	// EnterTypeidfilter is called when entering the typeidfilter production.
	EnterTypeidfilter(c *TypeidfilterContext)

	// EnterTypeid is called when entering the typeid production.
	EnterTypeid(c *TypeidContext)

	// EnterTypetokenfilter is called when entering the typetokenfilter production.
	EnterTypetokenfilter(c *TypetokenfilterContext)

	// EnterType is called when entering the type production.
	EnterType(c *TypeContext)

	// EnterTypetoken is called when entering the typetoken production.
	EnterTypetoken(c *TypetokenContext)

	// EnterTypetokenset is called when entering the typetokenset production.
	EnterTypetokenset(c *TypetokensetContext)

	// EnterSynonym is called when entering the synonym production.
	EnterSynonym(c *SynonymContext)

	// EnterFullyspecifiedname is called when entering the fullyspecifiedname production.
	EnterFullyspecifiedname(c *FullyspecifiednameContext)

	// EnterDefinition is called when entering the definition production.
	EnterDefinition(c *DefinitionContext)

	// EnterDialectfilter is called when entering the dialectfilter production.
	EnterDialectfilter(c *DialectfilterContext)

	// EnterDialectidfilter is called when entering the dialectidfilter production.
	EnterDialectidfilter(c *DialectidfilterContext)

	// EnterDialectid is called when entering the dialectid production.
	EnterDialectid(c *DialectidContext)

	// EnterDialectaliasfilter is called when entering the dialectaliasfilter production.
	EnterDialectaliasfilter(c *DialectaliasfilterContext)

	// EnterDialect is called when entering the dialect production.
	EnterDialect(c *DialectContext)

	// EnterDialectalias is called when entering the dialectalias production.
	EnterDialectalias(c *DialectaliasContext)

	// EnterDialectaliasset is called when entering the dialectaliasset production.
	EnterDialectaliasset(c *DialectaliassetContext)

	// EnterDialectidset is called when entering the dialectidset production.
	EnterDialectidset(c *DialectidsetContext)

	// EnterAcceptabilityset is called when entering the acceptabilityset production.
	EnterAcceptabilityset(c *AcceptabilitysetContext)

	// EnterAcceptabilityconceptreferenceset is called when entering the acceptabilityconceptreferenceset production.
	EnterAcceptabilityconceptreferenceset(c *AcceptabilityconceptreferencesetContext)

	// EnterAcceptabilitytokenset is called when entering the acceptabilitytokenset production.
	EnterAcceptabilitytokenset(c *AcceptabilitytokensetContext)

	// EnterAcceptabilitytoken is called when entering the acceptabilitytoken production.
	EnterAcceptabilitytoken(c *AcceptabilitytokenContext)

	// EnterAcceptable is called when entering the acceptable production.
	EnterAcceptable(c *AcceptableContext)

	// EnterPreferred is called when entering the preferred production.
	EnterPreferred(c *PreferredContext)

	// EnterConceptfilterconstraint is called when entering the conceptfilterconstraint production.
	EnterConceptfilterconstraint(c *ConceptfilterconstraintContext)

	// EnterConceptfilter is called when entering the conceptfilter production.
	EnterConceptfilter(c *ConceptfilterContext)

	// EnterDefinitionstatusfilter is called when entering the definitionstatusfilter production.
	EnterDefinitionstatusfilter(c *DefinitionstatusfilterContext)

	// EnterDefinitionstatusidfilter is called when entering the definitionstatusidfilter production.
	EnterDefinitionstatusidfilter(c *DefinitionstatusidfilterContext)

	// EnterDefinitionstatusidkeyword is called when entering the definitionstatusidkeyword production.
	EnterDefinitionstatusidkeyword(c *DefinitionstatusidkeywordContext)

	// EnterDefinitionstatustokenfilter is called when entering the definitionstatustokenfilter production.
	EnterDefinitionstatustokenfilter(c *DefinitionstatustokenfilterContext)

	// EnterDefinitionstatuskeyword is called when entering the definitionstatuskeyword production.
	EnterDefinitionstatuskeyword(c *DefinitionstatuskeywordContext)

	// EnterDefinitionstatustoken is called when entering the definitionstatustoken production.
	EnterDefinitionstatustoken(c *DefinitionstatustokenContext)

	// EnterDefinitionstatustokenset is called when entering the definitionstatustokenset production.
	EnterDefinitionstatustokenset(c *DefinitionstatustokensetContext)

	// EnterPrimitivetoken is called when entering the primitivetoken production.
	EnterPrimitivetoken(c *PrimitivetokenContext)

	// EnterDefinedtoken is called when entering the definedtoken production.
	EnterDefinedtoken(c *DefinedtokenContext)

	// EnterModulefilter is called when entering the modulefilter production.
	EnterModulefilter(c *ModulefilterContext)

	// EnterModuleidkeyword is called when entering the moduleidkeyword production.
	EnterModuleidkeyword(c *ModuleidkeywordContext)

	// EnterEffectivetimefilter is called when entering the effectivetimefilter production.
	EnterEffectivetimefilter(c *EffectivetimefilterContext)

	// EnterEffectivetimekeyword is called when entering the effectivetimekeyword production.
	EnterEffectivetimekeyword(c *EffectivetimekeywordContext)

	// EnterTimevalue is called when entering the timevalue production.
	EnterTimevalue(c *TimevalueContext)

	// EnterTimevalueset is called when entering the timevalueset production.
	EnterTimevalueset(c *TimevaluesetContext)

	// EnterYear is called when entering the year production.
	EnterYear(c *YearContext)

	// EnterMonth is called when entering the month production.
	EnterMonth(c *MonthContext)

	// EnterDay is called when entering the day production.
	EnterDay(c *DayContext)

	// EnterActivefilter is called when entering the activefilter production.
	EnterActivefilter(c *ActivefilterContext)

	// EnterActivekeyword is called when entering the activekeyword production.
	EnterActivekeyword(c *ActivekeywordContext)

	// EnterActivevalue is called when entering the activevalue production.
	EnterActivevalue(c *ActivevalueContext)

	// EnterActivetruevalue is called when entering the activetruevalue production.
	EnterActivetruevalue(c *ActivetruevalueContext)

	// EnterActivefalsevalue is called when entering the activefalsevalue production.
	EnterActivefalsevalue(c *ActivefalsevalueContext)

	// EnterMemberfilterconstraint is called when entering the memberfilterconstraint production.
	EnterMemberfilterconstraint(c *MemberfilterconstraintContext)

	// EnterMemberfilter is called when entering the memberfilter production.
	EnterMemberfilter(c *MemberfilterContext)

	// EnterMemberfieldfilter is called when entering the memberfieldfilter production.
	EnterMemberfieldfilter(c *MemberfieldfilterContext)

	// EnterHistorysupplement is called when entering the historysupplement production.
	EnterHistorysupplement(c *HistorysupplementContext)

	// EnterHistorykeyword is called when entering the historykeyword production.
	EnterHistorykeyword(c *HistorykeywordContext)

	// EnterHistoryprofilesuffix is called when entering the historyprofilesuffix production.
	EnterHistoryprofilesuffix(c *HistoryprofilesuffixContext)

	// EnterHistoryminimumsuffix is called when entering the historyminimumsuffix production.
	EnterHistoryminimumsuffix(c *HistoryminimumsuffixContext)

	// EnterHistorymoderatesuffix is called when entering the historymoderatesuffix production.
	EnterHistorymoderatesuffix(c *HistorymoderatesuffixContext)

	// EnterHistorymaximumsuffix is called when entering the historymaximumsuffix production.
	EnterHistorymaximumsuffix(c *HistorymaximumsuffixContext)

	// EnterHistorysubset is called when entering the historysubset production.
	EnterHistorysubset(c *HistorysubsetContext)

	// EnterNumericvalue is called when entering the numericvalue production.
	EnterNumericvalue(c *NumericvalueContext)

	// EnterStringvalue is called when entering the stringvalue production.
	EnterStringvalue(c *StringvalueContext)

	// EnterIntegervalue is called when entering the integervalue production.
	EnterIntegervalue(c *IntegervalueContext)

	// EnterDecimalvalue is called when entering the decimalvalue production.
	EnterDecimalvalue(c *DecimalvalueContext)

	// EnterBooleanvalue is called when entering the booleanvalue production.
	EnterBooleanvalue(c *BooleanvalueContext)

	// EnterTrue_1 is called when entering the true_1 production.
	EnterTrue_1(c *True_1Context)

	// EnterFalse_1 is called when entering the false_1 production.
	EnterFalse_1(c *False_1Context)

	// EnterNonnegativeintegervalue is called when entering the nonnegativeintegervalue production.
	EnterNonnegativeintegervalue(c *NonnegativeintegervalueContext)

	// EnterSctid is called when entering the sctid production.
	EnterSctid(c *SctidContext)

	// EnterWs is called when entering the ws production.
	EnterWs(c *WsContext)

	// EnterMws is called when entering the mws production.
	EnterMws(c *MwsContext)

	// EnterComment is called when entering the comment production.
	EnterComment(c *CommentContext)

	// EnterNonstarchar is called when entering the nonstarchar production.
	EnterNonstarchar(c *NonstarcharContext)

	// EnterStarwithnonfslash is called when entering the starwithnonfslash production.
	EnterStarwithnonfslash(c *StarwithnonfslashContext)

	// EnterNonfslash is called when entering the nonfslash production.
	EnterNonfslash(c *NonfslashContext)

	// EnterSp is called when entering the sp production.
	EnterSp(c *SpContext)

	// EnterHtab is called when entering the htab production.
	EnterHtab(c *HtabContext)

	// EnterCr is called when entering the cr production.
	EnterCr(c *CrContext)

	// EnterLf is called when entering the lf production.
	EnterLf(c *LfContext)

	// EnterQm is called when entering the qm production.
	EnterQm(c *QmContext)

	// EnterBs is called when entering the bs production.
	EnterBs(c *BsContext)

	// EnterStar is called when entering the star production.
	EnterStar(c *StarContext)

	// EnterDigit is called when entering the digit production.
	EnterDigit(c *DigitContext)

	// EnterZero is called when entering the zero production.
	EnterZero(c *ZeroContext)

	// EnterDigitnonzero is called when entering the digitnonzero production.
	EnterDigitnonzero(c *DigitnonzeroContext)

	// EnterNonwsnonpipe is called when entering the nonwsnonpipe production.
	EnterNonwsnonpipe(c *NonwsnonpipeContext)

	// EnterAnynonescapedchar is called when entering the anynonescapedchar production.
	EnterAnynonescapedchar(c *AnynonescapedcharContext)

	// EnterEscapedchar is called when entering the escapedchar production.
	EnterEscapedchar(c *EscapedcharContext)

	// EnterEscapedwildchar is called when entering the escapedwildchar production.
	EnterEscapedwildchar(c *EscapedwildcharContext)

	// EnterNonwsnonescapedchar is called when entering the nonwsnonescapedchar production.
	EnterNonwsnonescapedchar(c *NonwsnonescapedcharContext)

	// EnterAlpha is called when entering the alpha production.
	EnterAlpha(c *AlphaContext)

	// EnterDash is called when entering the dash production.
	EnterDash(c *DashContext)

	// ExitExpressionconstraint is called when exiting the expressionconstraint production.
	ExitExpressionconstraint(c *ExpressionconstraintContext)

	// ExitRefinedexpressionconstraint is called when exiting the refinedexpressionconstraint production.
	ExitRefinedexpressionconstraint(c *RefinedexpressionconstraintContext)

	// ExitCompoundexpressionconstraint is called when exiting the compoundexpressionconstraint production.
	ExitCompoundexpressionconstraint(c *CompoundexpressionconstraintContext)

	// ExitConjunctionexpressionconstraint is called when exiting the conjunctionexpressionconstraint production.
	ExitConjunctionexpressionconstraint(c *ConjunctionexpressionconstraintContext)

	// ExitDisjunctionexpressionconstraint is called when exiting the disjunctionexpressionconstraint production.
	ExitDisjunctionexpressionconstraint(c *DisjunctionexpressionconstraintContext)

	// ExitExclusionexpressionconstraint is called when exiting the exclusionexpressionconstraint production.
	ExitExclusionexpressionconstraint(c *ExclusionexpressionconstraintContext)

	// ExitDottedexpressionconstraint is called when exiting the dottedexpressionconstraint production.
	ExitDottedexpressionconstraint(c *DottedexpressionconstraintContext)

	// ExitDottedexpressionattribute is called when exiting the dottedexpressionattribute production.
	ExitDottedexpressionattribute(c *DottedexpressionattributeContext)

	// ExitSubexpressionconstraint is called when exiting the subexpressionconstraint production.
	ExitSubexpressionconstraint(c *SubexpressionconstraintContext)

	// ExitEclfocusconcept is called when exiting the eclfocusconcept production.
	ExitEclfocusconcept(c *EclfocusconceptContext)

	// ExitDot is called when exiting the dot production.
	ExitDot(c *DotContext)

	// ExitRefsetOperator is called when exiting the refsetOperator production.
	ExitRefsetOperator(c *RefsetOperatorContext)

	// ExitMemberof is called when exiting the memberof production.
	ExitMemberof(c *MemberofContext)

	// ExitRefsetContainingAny is called when exiting the refsetContainingAny production.
	ExitRefsetContainingAny(c *RefsetContainingAnyContext)

	// ExitRefsetfieldnameset is called when exiting the refsetfieldnameset production.
	ExitRefsetfieldnameset(c *RefsetfieldnamesetContext)

	// ExitRefsetfieldname is called when exiting the refsetfieldname production.
	ExitRefsetfieldname(c *RefsetfieldnameContext)

	// ExitEclconceptreference is called when exiting the eclconceptreference production.
	ExitEclconceptreference(c *EclconceptreferenceContext)

	// ExitEclconceptreferenceset is called when exiting the eclconceptreferenceset production.
	ExitEclconceptreferenceset(c *EclconceptreferencesetContext)

	// ExitConceptid is called when exiting the conceptid production.
	ExitConceptid(c *ConceptidContext)

	// ExitTerm is called when exiting the term production.
	ExitTerm(c *TermContext)

	// ExitAltidentifier is called when exiting the altidentifier production.
	ExitAltidentifier(c *AltidentifierContext)

	// ExitAltidentifierschemealias is called when exiting the altidentifierschemealias production.
	ExitAltidentifierschemealias(c *AltidentifierschemealiasContext)

	// ExitAltidentifiercodewithinquotes is called when exiting the altidentifiercodewithinquotes production.
	ExitAltidentifiercodewithinquotes(c *AltidentifiercodewithinquotesContext)

	// ExitAltidentifiercodewithoutquotes is called when exiting the altidentifiercodewithoutquotes production.
	ExitAltidentifiercodewithoutquotes(c *AltidentifiercodewithoutquotesContext)

	// ExitWildcard is called when exiting the wildcard production.
	ExitWildcard(c *WildcardContext)

	// ExitConstraintoperator is called when exiting the constraintoperator production.
	ExitConstraintoperator(c *ConstraintoperatorContext)

	// ExitDescendantof is called when exiting the descendantof production.
	ExitDescendantof(c *DescendantofContext)

	// ExitDescendantorselfof is called when exiting the descendantorselfof production.
	ExitDescendantorselfof(c *DescendantorselfofContext)

	// ExitChildof is called when exiting the childof production.
	ExitChildof(c *ChildofContext)

	// ExitChildorselfof is called when exiting the childorselfof production.
	ExitChildorselfof(c *ChildorselfofContext)

	// ExitAncestorof is called when exiting the ancestorof production.
	ExitAncestorof(c *AncestorofContext)

	// ExitAncestororselfof is called when exiting the ancestororselfof production.
	ExitAncestororselfof(c *AncestororselfofContext)

	// ExitParentof is called when exiting the parentof production.
	ExitParentof(c *ParentofContext)

	// ExitParentorselfof is called when exiting the parentorselfof production.
	ExitParentorselfof(c *ParentorselfofContext)

	// ExitTop is called when exiting the top production.
	ExitTop(c *TopContext)

	// ExitBottom is called when exiting the bottom production.
	ExitBottom(c *BottomContext)

	// ExitConjunction is called when exiting the conjunction production.
	ExitConjunction(c *ConjunctionContext)

	// ExitDisjunction is called when exiting the disjunction production.
	ExitDisjunction(c *DisjunctionContext)

	// ExitExclusion is called when exiting the exclusion production.
	ExitExclusion(c *ExclusionContext)

	// ExitEclrefinement is called when exiting the eclrefinement production.
	ExitEclrefinement(c *EclrefinementContext)

	// ExitConjunctionrefinementset is called when exiting the conjunctionrefinementset production.
	ExitConjunctionrefinementset(c *ConjunctionrefinementsetContext)

	// ExitDisjunctionrefinementset is called when exiting the disjunctionrefinementset production.
	ExitDisjunctionrefinementset(c *DisjunctionrefinementsetContext)

	// ExitSubrefinement is called when exiting the subrefinement production.
	ExitSubrefinement(c *SubrefinementContext)

	// ExitEclattributeset is called when exiting the eclattributeset production.
	ExitEclattributeset(c *EclattributesetContext)

	// ExitConjunctionattributeset is called when exiting the conjunctionattributeset production.
	ExitConjunctionattributeset(c *ConjunctionattributesetContext)

	// ExitDisjunctionattributeset is called when exiting the disjunctionattributeset production.
	ExitDisjunctionattributeset(c *DisjunctionattributesetContext)

	// ExitSubattributeset is called when exiting the subattributeset production.
	ExitSubattributeset(c *SubattributesetContext)

	// ExitEclattributegroup is called when exiting the eclattributegroup production.
	ExitEclattributegroup(c *EclattributegroupContext)

	// ExitEclattribute is called when exiting the eclattribute production.
	ExitEclattribute(c *EclattributeContext)

	// ExitCardinality is called when exiting the cardinality production.
	ExitCardinality(c *CardinalityContext)

	// ExitMinvalue is called when exiting the minvalue production.
	ExitMinvalue(c *MinvalueContext)

	// ExitTo is called when exiting the to production.
	ExitTo(c *ToContext)

	// ExitMaxvalue is called when exiting the maxvalue production.
	ExitMaxvalue(c *MaxvalueContext)

	// ExitMany is called when exiting the many production.
	ExitMany(c *ManyContext)

	// ExitReverseflag is called when exiting the reverseflag production.
	ExitReverseflag(c *ReverseflagContext)

	// ExitEclattributename is called when exiting the eclattributename production.
	ExitEclattributename(c *EclattributenameContext)

	// ExitExpressioncomparisonoperator is called when exiting the expressioncomparisonoperator production.
	ExitExpressioncomparisonoperator(c *ExpressioncomparisonoperatorContext)

	// ExitNumericcomparisonoperator is called when exiting the numericcomparisonoperator production.
	ExitNumericcomparisonoperator(c *NumericcomparisonoperatorContext)

	// ExitTimecomparisonoperator is called when exiting the timecomparisonoperator production.
	ExitTimecomparisonoperator(c *TimecomparisonoperatorContext)

	// ExitStringcomparisonoperator is called when exiting the stringcomparisonoperator production.
	ExitStringcomparisonoperator(c *StringcomparisonoperatorContext)

	// ExitBooleancomparisonoperator is called when exiting the booleancomparisonoperator production.
	ExitBooleancomparisonoperator(c *BooleancomparisonoperatorContext)

	// ExitIdcomparisonoperator is called when exiting the idcomparisonoperator production.
	ExitIdcomparisonoperator(c *IdcomparisonoperatorContext)

	// ExitDescriptionfilterconstraint is called when exiting the descriptionfilterconstraint production.
	ExitDescriptionfilterconstraint(c *DescriptionfilterconstraintContext)

	// ExitDescriptionfilter is called when exiting the descriptionfilter production.
	ExitDescriptionfilter(c *DescriptionfilterContext)

	// ExitDescriptionidfilter is called when exiting the descriptionidfilter production.
	ExitDescriptionidfilter(c *DescriptionidfilterContext)

	// ExitDescriptionidkeyword is called when exiting the descriptionidkeyword production.
	ExitDescriptionidkeyword(c *DescriptionidkeywordContext)

	// ExitDescriptionid is called when exiting the descriptionid production.
	ExitDescriptionid(c *DescriptionidContext)

	// ExitDescriptionidset is called when exiting the descriptionidset production.
	ExitDescriptionidset(c *DescriptionidsetContext)

	// ExitTermfilter is called when exiting the termfilter production.
	ExitTermfilter(c *TermfilterContext)

	// ExitTermkeyword is called when exiting the termkeyword production.
	ExitTermkeyword(c *TermkeywordContext)

	// ExitTypedsearchterm is called when exiting the typedsearchterm production.
	ExitTypedsearchterm(c *TypedsearchtermContext)

	// ExitTypedsearchtermset is called when exiting the typedsearchtermset production.
	ExitTypedsearchtermset(c *TypedsearchtermsetContext)

	// ExitConcretestringset is called when exiting the concretestringset production.
	ExitConcretestringset(c *ConcretestringsetContext)

	// ExitConcretestring is called when exiting the concretestring production.
	ExitConcretestring(c *ConcretestringContext)

	// ExitConcretestringcharacters is called when exiting the concretestringcharacters production.
	ExitConcretestringcharacters(c *ConcretestringcharactersContext)

	// ExitWild is called when exiting the wild production.
	ExitWild(c *WildContext)

	// ExitMatchkeyword is called when exiting the matchkeyword production.
	ExitMatchkeyword(c *MatchkeywordContext)

	// ExitMatchsearchterm is called when exiting the matchsearchterm production.
	ExitMatchsearchterm(c *MatchsearchtermContext)

	// ExitMatchsearchtermset is called when exiting the matchsearchtermset production.
	ExitMatchsearchtermset(c *MatchsearchtermsetContext)

	// ExitWildsearchterm is called when exiting the wildsearchterm production.
	ExitWildsearchterm(c *WildsearchtermContext)

	// ExitWildsearchtermset is called when exiting the wildsearchtermset production.
	ExitWildsearchtermset(c *WildsearchtermsetContext)

	// ExitLanguagefilter is called when exiting the languagefilter production.
	ExitLanguagefilter(c *LanguagefilterContext)

	// ExitLanguage is called when exiting the language production.
	ExitLanguage(c *LanguageContext)

	// ExitLanguagecode is called when exiting the languagecode production.
	ExitLanguagecode(c *LanguagecodeContext)

	// ExitLanguagecodeset is called when exiting the languagecodeset production.
	ExitLanguagecodeset(c *LanguagecodesetContext)

	// ExitTypefilter is called when exiting the typefilter production.
	ExitTypefilter(c *TypefilterContext)

	// ExitTypeidfilter is called when exiting the typeidfilter production.
	ExitTypeidfilter(c *TypeidfilterContext)

	// ExitTypeid is called when exiting the typeid production.
	ExitTypeid(c *TypeidContext)

	// ExitTypetokenfilter is called when exiting the typetokenfilter production.
	ExitTypetokenfilter(c *TypetokenfilterContext)

	// ExitType is called when exiting the type production.
	ExitType(c *TypeContext)

	// ExitTypetoken is called when exiting the typetoken production.
	ExitTypetoken(c *TypetokenContext)

	// ExitTypetokenset is called when exiting the typetokenset production.
	ExitTypetokenset(c *TypetokensetContext)

	// ExitSynonym is called when exiting the synonym production.
	ExitSynonym(c *SynonymContext)

	// ExitFullyspecifiedname is called when exiting the fullyspecifiedname production.
	ExitFullyspecifiedname(c *FullyspecifiednameContext)

	// ExitDefinition is called when exiting the definition production.
	ExitDefinition(c *DefinitionContext)

	// ExitDialectfilter is called when exiting the dialectfilter production.
	ExitDialectfilter(c *DialectfilterContext)

	// ExitDialectidfilter is called when exiting the dialectidfilter production.
	ExitDialectidfilter(c *DialectidfilterContext)

	// ExitDialectid is called when exiting the dialectid production.
	ExitDialectid(c *DialectidContext)

	// ExitDialectaliasfilter is called when exiting the dialectaliasfilter production.
	ExitDialectaliasfilter(c *DialectaliasfilterContext)

	// ExitDialect is called when exiting the dialect production.
	ExitDialect(c *DialectContext)

	// ExitDialectalias is called when exiting the dialectalias production.
	ExitDialectalias(c *DialectaliasContext)

	// ExitDialectaliasset is called when exiting the dialectaliasset production.
	ExitDialectaliasset(c *DialectaliassetContext)

	// ExitDialectidset is called when exiting the dialectidset production.
	ExitDialectidset(c *DialectidsetContext)

	// ExitAcceptabilityset is called when exiting the acceptabilityset production.
	ExitAcceptabilityset(c *AcceptabilitysetContext)

	// ExitAcceptabilityconceptreferenceset is called when exiting the acceptabilityconceptreferenceset production.
	ExitAcceptabilityconceptreferenceset(c *AcceptabilityconceptreferencesetContext)

	// ExitAcceptabilitytokenset is called when exiting the acceptabilitytokenset production.
	ExitAcceptabilitytokenset(c *AcceptabilitytokensetContext)

	// ExitAcceptabilitytoken is called when exiting the acceptabilitytoken production.
	ExitAcceptabilitytoken(c *AcceptabilitytokenContext)

	// ExitAcceptable is called when exiting the acceptable production.
	ExitAcceptable(c *AcceptableContext)

	// ExitPreferred is called when exiting the preferred production.
	ExitPreferred(c *PreferredContext)

	// ExitConceptfilterconstraint is called when exiting the conceptfilterconstraint production.
	ExitConceptfilterconstraint(c *ConceptfilterconstraintContext)

	// ExitConceptfilter is called when exiting the conceptfilter production.
	ExitConceptfilter(c *ConceptfilterContext)

	// ExitDefinitionstatusfilter is called when exiting the definitionstatusfilter production.
	ExitDefinitionstatusfilter(c *DefinitionstatusfilterContext)

	// ExitDefinitionstatusidfilter is called when exiting the definitionstatusidfilter production.
	ExitDefinitionstatusidfilter(c *DefinitionstatusidfilterContext)

	// ExitDefinitionstatusidkeyword is called when exiting the definitionstatusidkeyword production.
	ExitDefinitionstatusidkeyword(c *DefinitionstatusidkeywordContext)

	// ExitDefinitionstatustokenfilter is called when exiting the definitionstatustokenfilter production.
	ExitDefinitionstatustokenfilter(c *DefinitionstatustokenfilterContext)

	// ExitDefinitionstatuskeyword is called when exiting the definitionstatuskeyword production.
	ExitDefinitionstatuskeyword(c *DefinitionstatuskeywordContext)

	// ExitDefinitionstatustoken is called when exiting the definitionstatustoken production.
	ExitDefinitionstatustoken(c *DefinitionstatustokenContext)

	// ExitDefinitionstatustokenset is called when exiting the definitionstatustokenset production.
	ExitDefinitionstatustokenset(c *DefinitionstatustokensetContext)

	// ExitPrimitivetoken is called when exiting the primitivetoken production.
	ExitPrimitivetoken(c *PrimitivetokenContext)

	// ExitDefinedtoken is called when exiting the definedtoken production.
	ExitDefinedtoken(c *DefinedtokenContext)

	// ExitModulefilter is called when exiting the modulefilter production.
	ExitModulefilter(c *ModulefilterContext)

	// ExitModuleidkeyword is called when exiting the moduleidkeyword production.
	ExitModuleidkeyword(c *ModuleidkeywordContext)

	// ExitEffectivetimefilter is called when exiting the effectivetimefilter production.
	ExitEffectivetimefilter(c *EffectivetimefilterContext)

	// ExitEffectivetimekeyword is called when exiting the effectivetimekeyword production.
	ExitEffectivetimekeyword(c *EffectivetimekeywordContext)

	// ExitTimevalue is called when exiting the timevalue production.
	ExitTimevalue(c *TimevalueContext)

	// ExitTimevalueset is called when exiting the timevalueset production.
	ExitTimevalueset(c *TimevaluesetContext)

	// ExitYear is called when exiting the year production.
	ExitYear(c *YearContext)

	// ExitMonth is called when exiting the month production.
	ExitMonth(c *MonthContext)

	// ExitDay is called when exiting the day production.
	ExitDay(c *DayContext)

	// ExitActivefilter is called when exiting the activefilter production.
	ExitActivefilter(c *ActivefilterContext)

	// ExitActivekeyword is called when exiting the activekeyword production.
	ExitActivekeyword(c *ActivekeywordContext)

	// ExitActivevalue is called when exiting the activevalue production.
	ExitActivevalue(c *ActivevalueContext)

	// ExitActivetruevalue is called when exiting the activetruevalue production.
	ExitActivetruevalue(c *ActivetruevalueContext)

	// ExitActivefalsevalue is called when exiting the activefalsevalue production.
	ExitActivefalsevalue(c *ActivefalsevalueContext)

	// ExitMemberfilterconstraint is called when exiting the memberfilterconstraint production.
	ExitMemberfilterconstraint(c *MemberfilterconstraintContext)

	// ExitMemberfilter is called when exiting the memberfilter production.
	ExitMemberfilter(c *MemberfilterContext)

	// ExitMemberfieldfilter is called when exiting the memberfieldfilter production.
	ExitMemberfieldfilter(c *MemberfieldfilterContext)

	// ExitHistorysupplement is called when exiting the historysupplement production.
	ExitHistorysupplement(c *HistorysupplementContext)

	// ExitHistorykeyword is called when exiting the historykeyword production.
	ExitHistorykeyword(c *HistorykeywordContext)

	// ExitHistoryprofilesuffix is called when exiting the historyprofilesuffix production.
	ExitHistoryprofilesuffix(c *HistoryprofilesuffixContext)

	// ExitHistoryminimumsuffix is called when exiting the historyminimumsuffix production.
	ExitHistoryminimumsuffix(c *HistoryminimumsuffixContext)

	// ExitHistorymoderatesuffix is called when exiting the historymoderatesuffix production.
	ExitHistorymoderatesuffix(c *HistorymoderatesuffixContext)

	// ExitHistorymaximumsuffix is called when exiting the historymaximumsuffix production.
	ExitHistorymaximumsuffix(c *HistorymaximumsuffixContext)

	// ExitHistorysubset is called when exiting the historysubset production.
	ExitHistorysubset(c *HistorysubsetContext)

	// ExitNumericvalue is called when exiting the numericvalue production.
	ExitNumericvalue(c *NumericvalueContext)

	// ExitStringvalue is called when exiting the stringvalue production.
	ExitStringvalue(c *StringvalueContext)

	// ExitIntegervalue is called when exiting the integervalue production.
	ExitIntegervalue(c *IntegervalueContext)

	// ExitDecimalvalue is called when exiting the decimalvalue production.
	ExitDecimalvalue(c *DecimalvalueContext)

	// ExitBooleanvalue is called when exiting the booleanvalue production.
	ExitBooleanvalue(c *BooleanvalueContext)

	// ExitTrue_1 is called when exiting the true_1 production.
	ExitTrue_1(c *True_1Context)

	// ExitFalse_1 is called when exiting the false_1 production.
	ExitFalse_1(c *False_1Context)

	// ExitNonnegativeintegervalue is called when exiting the nonnegativeintegervalue production.
	ExitNonnegativeintegervalue(c *NonnegativeintegervalueContext)

	// ExitSctid is called when exiting the sctid production.
	ExitSctid(c *SctidContext)

	// ExitWs is called when exiting the ws production.
	ExitWs(c *WsContext)

	// ExitMws is called when exiting the mws production.
	ExitMws(c *MwsContext)

	// ExitComment is called when exiting the comment production.
	ExitComment(c *CommentContext)

	// ExitNonstarchar is called when exiting the nonstarchar production.
	ExitNonstarchar(c *NonstarcharContext)

	// ExitStarwithnonfslash is called when exiting the starwithnonfslash production.
	ExitStarwithnonfslash(c *StarwithnonfslashContext)

	// ExitNonfslash is called when exiting the nonfslash production.
	ExitNonfslash(c *NonfslashContext)

	// ExitSp is called when exiting the sp production.
	ExitSp(c *SpContext)

	// ExitHtab is called when exiting the htab production.
	ExitHtab(c *HtabContext)

	// ExitCr is called when exiting the cr production.
	ExitCr(c *CrContext)

	// ExitLf is called when exiting the lf production.
	ExitLf(c *LfContext)

	// ExitQm is called when exiting the qm production.
	ExitQm(c *QmContext)

	// ExitBs is called when exiting the bs production.
	ExitBs(c *BsContext)

	// ExitStar is called when exiting the star production.
	ExitStar(c *StarContext)

	// ExitDigit is called when exiting the digit production.
	ExitDigit(c *DigitContext)

	// ExitZero is called when exiting the zero production.
	ExitZero(c *ZeroContext)

	// ExitDigitnonzero is called when exiting the digitnonzero production.
	ExitDigitnonzero(c *DigitnonzeroContext)

	// ExitNonwsnonpipe is called when exiting the nonwsnonpipe production.
	ExitNonwsnonpipe(c *NonwsnonpipeContext)

	// ExitAnynonescapedchar is called when exiting the anynonescapedchar production.
	ExitAnynonescapedchar(c *AnynonescapedcharContext)

	// ExitEscapedchar is called when exiting the escapedchar production.
	ExitEscapedchar(c *EscapedcharContext)

	// ExitEscapedwildchar is called when exiting the escapedwildchar production.
	ExitEscapedwildchar(c *EscapedwildcharContext)

	// ExitNonwsnonescapedchar is called when exiting the nonwsnonescapedchar production.
	ExitNonwsnonescapedchar(c *NonwsnonescapedcharContext)

	// ExitAlpha is called when exiting the alpha production.
	ExitAlpha(c *AlphaContext)

	// ExitDash is called when exiting the dash production.
	ExitDash(c *DashContext)
}
