package providertest

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gofhir/ecl/ecl"
)

// FixtureSpec is the on-disk YAML schema for an in-memory DataProvider
// fixture. All fields are optional; an empty fixture is a valid (empty)
// terminology.
type FixtureSpec struct {
	Name                   string                         `yaml:"name"`
	CodeSystem             string                         `yaml:"codeSystem"`
	Version                string                         `yaml:"version"`
	Concepts               []FixtureConcept               `yaml:"concepts"`
	Parents                map[string][]string            `yaml:"parents"` // direct parents per concept
	Descriptions           []FixtureDescription           `yaml:"descriptions"`
	Relationships          []FixtureRelationship          `yaml:"relationships"`
	Refsets                map[string][]string            `yaml:"refsets"` // refsetID → memberIDs
	HistoricalAssociations []FixtureHistory               `yaml:"historicalAssociations"`
	AltIdentifiers         map[string]map[string][]string `yaml:"altIdentifiers"` // scheme → code → conceptIDs
	Dialects               []FixtureDialectMember         `yaml:"dialects"`       // (dialectID, acceptabilityID) → conceptIDs as flat list
	MemberFields           []FixtureMemberField           `yaml:"memberFields"`

	// DialectAliases maps a dialect alias to the SCTIDs of its language
	// reference sets, for `{{ D dialect = en-gb }}`. Declaring it is what makes
	// the fixture satisfy ecl.DialectAliasResolver; an alias that is absent stays
	// unresolvable, which is the case the evaluator has to report rather than
	// widen.
	DialectAliases map[string][]string `yaml:"dialectAliases"`
}

// FixtureConcept is one concept declaration in the fixture.
type FixtureConcept struct {
	ID                 string `yaml:"id"`
	Display            string `yaml:"display"`
	Active             *bool  `yaml:"active"` // nil = active by default
	DefinitionStatusID string `yaml:"definitionStatusId"`
	ModuleID           string `yaml:"moduleId"`
	EffectiveTime      string `yaml:"effectiveTime"`
}

// FixtureDescription is one description (designation) row.
type FixtureDescription struct {
	// ID is the description's own SCTID, for `{{ D id = ... }}`. Optional: a
	// description without one simply cannot be selected by that filter, which is
	// the same as a real terminology where the filter names an id nothing has.
	ID string `yaml:"id"`

	Concept       string `yaml:"concept"`
	Language      string `yaml:"language"`
	TypeID        string `yaml:"typeId"`
	Value         string `yaml:"value"`
	Active        *bool  `yaml:"active"`
	ModuleID      string `yaml:"moduleId"`
	EffectiveTime string `yaml:"effectiveTime"`
}

// FixtureRelationship is one attribute relationship. Either Target or
// Concrete is set; never both.
type FixtureRelationship struct {
	Source   string           `yaml:"source"`
	TypeID   string           `yaml:"typeId"`
	Target   string           `yaml:"target"`
	Group    int              `yaml:"group"`
	Concrete *FixtureConcrete `yaml:"concrete"`
}

// FixtureConcrete is a concrete-value attribute payload.
type FixtureConcrete struct {
	Kind  string `yaml:"kind"` // "integer" | "decimal" | "string" | "boolean"
	Value string `yaml:"value"`
}

// FixtureHistory is a historical association row.
type FixtureHistory struct {
	Refset string `yaml:"refset"` // SAME_AS / REPLACED_BY / etc. SCTID
	Source string `yaml:"source"` // inactive concept being replaced
	Target string `yaml:"target"` // replacement concept
}

// FixtureDialectMember binds a description (concept + language) to a
// (dialectID, acceptabilityID) pair.
type FixtureDialectMember struct {
	DialectID       string `yaml:"dialectId"`
	AcceptabilityID string `yaml:"acceptabilityId"`
	ConceptID       string `yaml:"conceptId"`
	Language        string `yaml:"language"`
}

// FixtureMemberField captures a per-member custom field used by member
// filter constraints ({{ M field = ... }}).
type FixtureMemberField struct {
	Refset string `yaml:"refset"`
	Member string `yaml:"member"`

	// Row groups the fields belonging to ONE reference set member row. A
	// reference set may hold several rows for the same member — a complex map has
	// one per map group — and a `{{ M ... }}` filter with several clauses asks for
	// a single row satisfying all of them.
	//
	// Empty means the member has one row, which is every simple reference set.
	Row string `yaml:"row"`

	FieldName string `yaml:"fieldName"`
	Value     string `yaml:"value"`
}

// LoadFixtureFile reads and parses a YAML fixture from disk.
func LoadFixtureFile(path string) (ecl.DataProvider, error) {
	b, err := os.ReadFile(path) //nolint:gosec // CLI tool reads user-provided paths
	if err != nil {
		return nil, fmt.Errorf("read fixture: %w", err)
	}
	return LoadFixture(b)
}

// LoadFixture parses YAML bytes into an in-memory DataProvider.
func LoadFixture(data []byte) (ecl.DataProvider, error) {
	var spec FixtureSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parse fixture YAML: %w", err)
	}
	return NewInMemoryProvider(&spec), nil
}

// NewInMemoryProvider builds an in-memory DataProvider from a fixture spec.
// Parents are expanded into a transitive closure for hierarchy queries.
func NewInMemoryProvider(spec *FixtureSpec) ecl.DataProvider {
	p := &inMemoryProvider{
		spec:            spec,
		concepts:        make(map[string]FixtureConcept, len(spec.Concepts)),
		descByConcept:   make(map[string][]FixtureDescription),
		relsBySource:    make(map[string][]ecl.Relationship),
		refsetMembers:   make(map[string]map[string]struct{}),
		refsetsByMember: make(map[string]map[string]struct{}),
		altIDs:          spec.AltIdentifiers,
	}

	for _, c := range spec.Concepts {
		p.concepts[c.ID] = c
	}
	for _, d := range spec.Descriptions {
		p.descByConcept[d.Concept] = append(p.descByConcept[d.Concept], d)
	}
	for _, r := range spec.Relationships {
		rel := ecl.Relationship{
			TypeID:   r.TypeID,
			TargetID: r.Target,
			GroupNum: r.Group,
		}
		if r.Concrete != nil {
			rel.ConcreteValue = &ecl.ConcreteValue{Kind: r.Concrete.Kind, Value: r.Concrete.Value}
		}
		p.relsBySource[r.Source] = append(p.relsBySource[r.Source], rel)
	}
	addMember := func(refsetID, member string) {
		if p.refsetMembers[refsetID] == nil {
			p.refsetMembers[refsetID] = make(map[string]struct{})
		}
		p.refsetMembers[refsetID][member] = struct{}{}
		if p.refsetsByMember[member] == nil {
			p.refsetsByMember[member] = make(map[string]struct{})
		}
		p.refsetsByMember[member][refsetID] = struct{}{}
	}
	for refsetID, members := range spec.Refsets {
		for _, m := range members {
			addMember(refsetID, m)
		}
	}
	// An association declared under historicalAssociations IS a reference set
	// member, and its source is the referencedComponentId. Without this,
	// `^ 900000000000527005` returned nothing while
	// `^ [referencedComponentId] 900000000000527005` returned the member — and
	// those two are the same question, since referencedComponentId is the default
	// projection. Declaring the rows a second time under refsets would have left
	// two copies to drift apart instead.
	for _, h := range spec.HistoricalAssociations {
		addMember(h.Refset, h.Source)
	}

	// Build transitive closure from direct parents.
	p.closure = buildClosure(spec.Parents)

	return p
}

// buildClosure expands the direct-parent map into a transitive (ancestor,
// descendant) relation. Self-references are NOT included; callers add them
// when includeSelf is requested.
func buildClosure(parents map[string][]string) closure {
	// First build the direct relation as ancestors[d] = set of direct
	// parents. Then iteratively expand until fixed point.
	ancestors := make(map[string]map[string]struct{})
	for child, ps := range parents {
		if ancestors[child] == nil {
			ancestors[child] = make(map[string]struct{})
		}
		for _, p := range ps {
			ancestors[child][p] = struct{}{}
		}
	}
	// Iterate to fix point. n^2 in the worst case but fine for fixture sizes.
	changed := true
	for changed {
		changed = false
		for child, anc := range ancestors {
			for parent := range anc {
				for grandparent := range ancestors[parent] {
					if _, ok := ancestors[child][grandparent]; !ok {
						ancestors[child][grandparent] = struct{}{}
						changed = true
					}
				}
			}
		}
	}
	descendants := make(map[string]map[string]struct{})
	for child, anc := range ancestors {
		for a := range anc {
			if descendants[a] == nil {
				descendants[a] = make(map[string]struct{})
			}
			descendants[a][child] = struct{}{}
		}
	}
	// Direct children/parents map (depth=1) for Children/Parents calls.
	directChildren := make(map[string]map[string]struct{})
	directParents := make(map[string]map[string]struct{})
	for child, ps := range parents {
		for _, p := range ps {
			if directChildren[p] == nil {
				directChildren[p] = make(map[string]struct{})
			}
			directChildren[p][child] = struct{}{}
			if directParents[child] == nil {
				directParents[child] = make(map[string]struct{})
			}
			directParents[child][p] = struct{}{}
		}
	}
	return closure{
		ancestors:      ancestors,
		descendants:    descendants,
		directChildren: directChildren,
		directParents:  directParents,
	}
}

type closure struct {
	ancestors      map[string]map[string]struct{}
	descendants    map[string]map[string]struct{}
	directChildren map[string]map[string]struct{}
	directParents  map[string]map[string]struct{}
}

// inMemoryProvider implements ecl.DataProvider against fixture data.
// The reference provider implements DataProvider and every optional capability,
// which is what makes it a worked example and what lets the bundled suite exercise
// the paths those capabilities unlock.
//
// Asserted at compile time because the failure is otherwise SILENT: rename a
// method or change a signature and the provider quietly stops satisfying the
// interface, VerifyContract skips that check with "the provider does not implement
// …", the suite stays green, and the README goes on claiming all seven. Nothing
// enforced the claim until this block existed.
var (
	_ ecl.DataProvider = (*inMemoryProvider)(nil)

	_ ecl.BatchPropertiesProvider      = (*inMemoryProvider)(nil)
	_ ecl.BatchConcreteValuesProvider  = (*inMemoryProvider)(nil)
	_ ecl.InboundRelationshipsProvider = (*inMemoryProvider)(nil)
	_ ecl.NegatingDescriptionProvider  = (*inMemoryProvider)(nil)
	_ ecl.DialectAliasResolver         = (*inMemoryProvider)(nil)
	_ ecl.DescriptionIDProvider        = (*inMemoryProvider)(nil)
	_ ecl.RefsetFieldProjector         = (*inMemoryProvider)(nil)
)

type inMemoryProvider struct {
	spec            *FixtureSpec
	concepts        map[string]FixtureConcept
	descByConcept   map[string][]FixtureDescription
	relsBySource    map[string][]ecl.Relationship
	refsetMembers   map[string]map[string]struct{} // refsetID → set of memberIDs
	refsetsByMember map[string]map[string]struct{} // memberID → set of refsetIDs
	altIDs          map[string]map[string][]string
	closure         closure
}

func (p *inMemoryProvider) Descendants(_ context.Context, conceptIDs []string, includeSelf bool) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, id := range conceptIDs {
		if includeSelf {
			out = out.Union(ecl.NewSetFromSlice([]string{id}))
		}
		for d := range p.closure.descendants[id] {
			out = out.Union(ecl.NewSetFromSlice([]string{d}))
		}
	}
	return out, nil
}

func (p *inMemoryProvider) Ancestors(_ context.Context, conceptIDs []string, includeSelf bool) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, id := range conceptIDs {
		if includeSelf {
			out = out.Union(ecl.NewSetFromSlice([]string{id}))
		}
		for a := range p.closure.ancestors[id] {
			out = out.Union(ecl.NewSetFromSlice([]string{a}))
		}
	}
	return out, nil
}

func (p *inMemoryProvider) Children(_ context.Context, conceptIDs []string, includeSelf bool) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, id := range conceptIDs {
		if includeSelf {
			out = out.Union(ecl.NewSetFromSlice([]string{id}))
		}
		for c := range p.closure.directChildren[id] {
			out = out.Union(ecl.NewSetFromSlice([]string{c}))
		}
	}
	return out, nil
}

func (p *inMemoryProvider) Parents(_ context.Context, conceptIDs []string, includeSelf bool) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, id := range conceptIDs {
		if includeSelf {
			out = out.Union(ecl.NewSetFromSlice([]string{id}))
		}
		for par := range p.closure.directParents[id] {
			out = out.Union(ecl.NewSetFromSlice([]string{par}))
		}
	}
	return out, nil
}

// ConceptExists reports presence of concepts in the terminology regardless of
// their active flag. ECL references inactive concepts directly (e.g. through
// HistorySupplement) so the existence check must not gate on active. Refset
// SCTIDs declared via the `refsets:` map are also considered to exist even if
// not duplicated in the `concepts:` list.
func (p *inMemoryProvider) ConceptExists(_ context.Context, conceptIDs []string) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, id := range conceptIDs {
		if _, ok := p.concepts[id]; ok {
			out = out.Union(ecl.NewSetFromSlice([]string{id}))
			continue
		}
		if _, ok := p.refsetMembers[id]; ok {
			out = out.Union(ecl.NewSetFromSlice([]string{id}))
		}
	}
	return out, nil
}

// AllConcepts returns every declared concept, active or not.
//
// It deliberately does NOT filter by the active flag: the wildcard is resolved
// before filters are applied, so filtering here would make
// `* {{ C active = false }}` unable to return anything. Restricting the active
// axis is FilterConcepts' job. See the DataProvider contract.
func (p *inMemoryProvider) AllConcepts(_ context.Context) (ecl.Set, error) {
	ids := make([]string, 0, len(p.concepts))
	for id := range p.concepts {
		ids = append(ids, id)
	}
	return ecl.NewSetFromSlice(ids), nil
}

func (p *inMemoryProvider) RelationshipTargets(_ context.Context, sources, types ecl.Set) (ecl.Set, error) {
	if types == nil || types.Len() == 0 {
		return ecl.NewSet(), nil
	}
	out := ecl.NewSet()
	visit := func(srcID string) {
		for _, r := range p.relsBySource[srcID] {
			if types.Contains(r.TypeID) && r.TargetID != "" {
				out = out.Union(ecl.NewSetFromSlice([]string{r.TargetID}))
			}
		}
	}
	if sources == nil {
		// Wildcard: scan all sources.
		for src := range p.relsBySource {
			visit(src)
		}
	} else {
		sources.Iter(func(id string) bool { visit(id); return true })
	}
	return out, nil
}

func (p *inMemoryProvider) RelationshipSources(_ context.Context, targets, types ecl.Set) (ecl.Set, error) {
	if types == nil || types.Len() == 0 {
		return ecl.NewSet(), nil
	}
	out := ecl.NewSet()
	for src, rels := range p.relsBySource {
		for _, r := range rels {
			if !types.Contains(r.TypeID) {
				continue
			}
			if targets == nil || targets.Contains(r.TargetID) {
				out = out.Union(ecl.NewSetFromSlice([]string{src}))
				break
			}
		}
	}
	return out, nil
}

func (p *inMemoryProvider) ConcreteValues(_ context.Context, sourceID, typeID string) ([]ecl.ConcreteValue, error) {
	var out []ecl.ConcreteValue
	for _, r := range p.relsBySource[sourceID] {
		if r.TypeID == typeID && r.ConcreteValue != nil {
			out = append(out, *r.ConcreteValue)
		}
	}
	return out, nil
}

func (p *inMemoryProvider) PropertiesByGroup(_ context.Context, conceptID string) (map[int][]ecl.Relationship, error) {
	rels := p.relsBySource[conceptID]
	if len(rels) == 0 {
		return nil, nil
	}
	out := make(map[int][]ecl.Relationship)
	for _, r := range rels {
		out[r.GroupNum] = append(out[r.GroupNum], r)
	}
	return out, nil
}

func (p *inMemoryProvider) MatchDescription(_ context.Context, opts ecl.DescriptionFilterOpts) (ecl.Set, error) {
	out := ecl.NewSet()
	needle := strings.ToLower(opts.Term)
	for conceptID, descs := range p.descByConcept {
		for _, d := range descs {
			if !descriptionMatches(d, opts, needle) {
				continue
			}
			out = out.Union(ecl.NewSetFromSlice([]string{conceptID}))
			break
		}
	}
	return out, nil
}

func descriptionMatches(d FixtureDescription, opts ecl.DescriptionFilterOpts, needle string) bool {
	if opts.Term != "" && !termMatches(d.Value, opts.Term, opts.MatchType, needle) {
		return false
	}
	if len(opts.TypeIDs) > 0 && !contains(opts.TypeIDs, d.TypeID) {
		return false
	}
	if len(opts.Languages) > 0 && !contains(opts.Languages, d.Language) {
		return false
	}
	if opts.Active != nil && isActive(d.Active) != *opts.Active {
		return false
	}
	if len(opts.ModuleIDs) > 0 && !contains(opts.ModuleIDs, d.ModuleID) {
		return false
	}
	return true
}

func termMatches(value, term, matchType, lowerTerm string) bool {
	switch matchType {
	case "wild":
		// Glob → simplified: '*' anywhere, case-insensitive.
		return globMatch(strings.ToLower(value), strings.ToLower(term))
	default: // "match" or empty
		return wordPrefixMatch(value, lowerTerm)
	}
}

// wordPrefixMatch implements the default "match" search type: every word of the
// search term must be a prefix of some word of the description, in any order.
//
// This replaced strings.Contains, which was wrong in both directions and, being
// the reference implementation the README points implementors at, taught the
// wrong semantics:
//
//	{{ D term = "infarction myocardial" }}  Contains: no match (order matters)
//	                                        correct:  matches (order is irrelevant)
//	{{ D term = "farct" }}                  Contains: matches (mid-word)
//	                                        correct:  no match (not a word prefix)
func wordPrefixMatch(value, lowerTerm string) bool {
	needles := strings.Fields(lowerTerm)
	if len(needles) == 0 {
		return true
	}
	haystack := strings.Fields(strings.ToLower(value))
	for _, needle := range needles {
		found := false
		for _, word := range haystack {
			if strings.HasPrefix(word, needle) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// globMatch implements the wild search type for the in-memory fixture: a '*'
// matches any run of characters and anything else matches literally. There is no
// '?' and no character classes, which the grammar does not offer either.
//
// A pattern with no '*' must match the WHOLE string. The previous
// implementation split on '*' and treated a single-segment pattern as case 0,
// which only checked HasPrefix -- so `wild: "Diabet"` matched
// "Diabetes mellitus" even though the pattern asks for an exact match.
func globMatch(s, pattern string) bool {
	segments := splitGlob(pattern)
	if len(segments) == 1 {
		return s == segments[0]
	}

	// First segment is anchored at the start, last at the end, the rest match in
	// order anywhere between.
	if !strings.HasPrefix(s, segments[0]) {
		return false
	}
	s = s[len(segments[0]):]

	last := segments[len(segments)-1]
	for _, part := range segments[1 : len(segments)-1] {
		idx := strings.Index(s, part)
		if idx < 0 {
			return false
		}
		s = s[idx+len(part):]
	}

	if len(s) < len(last) {
		return false
	}
	return strings.HasSuffix(s, last)
}

// splitGlob splits a wild pattern on its unescaped '*' separators, decoding `\*`
// into a literal asterisk inside the segments.
//
// The parser deliberately leaves `\*` encoded so a literal asterisk stays
// distinguishable from a wildcard; splitting on a plain "*" would have turned
// every escaped asterisk into one.
func splitGlob(pattern string) []string {
	var (
		segments []string
		cur      strings.Builder
	)
	for i := 0; i < len(pattern); i++ {
		switch {
		case pattern[i] == '\\' && i+1 < len(pattern) && pattern[i+1] == '*':
			cur.WriteByte('*') // escaped: a literal asterisk
			i++
		case pattern[i] == '*':
			segments = append(segments, cur.String())
			cur.Reset()
		default:
			cur.WriteByte(pattern[i])
		}
	}
	return append(segments, cur.String())
}

func (p *inMemoryProvider) FilterConcepts(_ context.Context, concepts ecl.Set, opts ecl.ConceptFilterOpts) (ecl.Set, error) {
	out := ecl.NewSet()
	if concepts == nil {
		return out, nil
	}
	concepts.Iter(func(id string) bool {
		c, ok := p.concepts[id]
		if !ok {
			return true
		}
		if opts.Active != nil && isActive(c.Active) != *opts.Active {
			return true
		}
		if len(opts.DefinitionStatusIDs) > 0 && !contains(opts.DefinitionStatusIDs, c.DefinitionStatusID) {
			return true
		}
		if len(opts.ModuleIDs) > 0 && !contains(opts.ModuleIDs, c.ModuleID) {
			return true
		}
		if !effectiveTimeMatches(c.EffectiveTime, opts.EffectiveTime, opts.EffectiveTimeOp) {
			return true
		}
		out = out.Union(ecl.NewSetFromSlice([]string{id}))
		return true
	})
	return out, nil
}

func (p *inMemoryProvider) RefsetMembers(_ context.Context, refsetIDs []string) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, rid := range refsetIDs {
		for m := range p.refsetMembers[rid] {
			out = out.Union(ecl.NewSetFromSlice([]string{m}))
		}
	}
	return out, nil
}

func (p *inMemoryProvider) RefsetsContainingMembers(_ context.Context, conceptIDs []string) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, id := range conceptIDs {
		for rs := range p.refsetsByMember[id] {
			out = out.Union(ecl.NewSetFromSlice([]string{rs}))
		}
	}
	return out, nil
}

func (p *inMemoryProvider) HistoricalAssociations(_ context.Context, conceptIDs ecl.Set, profile string) (ecl.Set, error) {
	if conceptIDs == nil {
		return ecl.NewSet(), nil
	}
	allowed := historyRefsetsForProfile(profile)
	out := ecl.NewSet()
	conceptIDs.Iter(func(id string) bool {
		for _, h := range p.spec.HistoricalAssociations {
			// The input concepts are the TARGETS of the associations, and the
			// result is the inactive concepts pointing at them.
			//
			// The spec defines the supplement as
			//   (X) OR (^ 900000000000527005 {{ M targetComponentId = (X) }})
			// so it adds the association members whose targetComponentId falls in
			// X. Walking it the other way round (matching h.Source against id and
			// emitting h.Target) made {{ +HISTORY }} a silent no-op for any set of
			// active concepts, which is the whole point of the operator.
			if h.Target != id {
				continue
			}
			if allowed != nil && !contains(allowed, h.Refset) {
				continue
			}
			out = out.Union(ecl.NewSetFromSlice([]string{h.Source}))
		}
		return true
	})
	return out, nil
}

// historyRefsetsForProfile returns the SCTIDs allowed for a given history
// profile. Returns nil for MAX/empty (no filter).
func historyRefsetsForProfile(profile string) []string {
	switch profile {
	case "MIN", "HISTORY-MIN":
		return []string{"900000000000527005"} // SAME_AS only
	case "MOD", "HISTORY-MOD":
		// The SCTIDs are right and three of the four labels used to be wrong:
		// 900000000000528000 is WAS A (not REPLACED BY), 900000000000526001 is
		// REPLACED BY (not WAS A), and 900000000000530003 is ALTERNATIVE (not
		// MOVED TO, which is 900000000000524003 and belongs to MAX only).
		// Confirmed against the terminology: `< 900000000000522004` lists all
		// eleven association reference sets with their names.
		//
		// internal/oracle mirrors this list so the differential test can check it
		// against a real terminology server. The copy is manual: change both.
		return []string{
			"900000000000527005", // SAME AS
			"900000000000526001", // REPLACED BY
			"900000000000528000", // WAS A
			"900000000000530003", // ALTERNATIVE
		}
	default:
		return nil // MAX / empty / unknown → all
	}
}

func (p *inMemoryProvider) ResolveIdentifier(_ context.Context, scheme, code string) (ecl.Set, error) {
	if p.altIDs == nil {
		return ecl.NewSet(), nil
	}
	if byCode, ok := p.altIDs[scheme]; ok {
		if ids, ok := byCode[code]; ok {
			return ecl.NewSetFromSlice(ids), nil
		}
	}
	return ecl.NewSet(), nil
}

// MatchDialect returns the concepts whose descriptions satisfy the dialect
// filter.
//
// Note that opts.Negate inverts the match at the DESCRIPTION ROW level, which is
// why the evaluator hands the flag over rather than subtracting sets: a concept
// with a preferred term in en-gb and another in en-us satisfies
// `dialectId != <en-gb refset>` through the en-us row. Ignoring the flag made the
// negated form return exactly the same set as the positive one.
func (p *inMemoryProvider) MatchDialect(_ context.Context, concepts ecl.Set, opts ecl.DialectFilterOpts) (ecl.Set, error) {
	out := ecl.NewSet()
	if concepts == nil {
		return out, nil
	}
	concepts.Iter(func(id string) bool {
		for _, d := range p.spec.Dialects {
			if d.ConceptID != id {
				continue
			}
			if dialectRowMatches(d, opts.Dialects) != opts.Negate {
				out = out.Union(ecl.NewSetFromSlice([]string{id}))
				return true
			}
		}
		return true
	})
	return out, nil
}

// dialectRowMatches reports whether one dialect membership row satisfies any entry
// of the filter, which is the any-of semantics DialectEntryOpts documents.
func dialectRowMatches(d FixtureDialectMember, entries []ecl.DialectEntryOpts) bool {
	for _, entry := range entries {
		if !contains(entry.DialectIDs, d.DialectID) {
			continue
		}
		if len(entry.AcceptabilityIDs) > 0 && !contains(entry.AcceptabilityIDs, d.AcceptabilityID) {
			continue
		}
		return true
	}
	return false
}

func (p *inMemoryProvider) RefsetMembersFiltered(_ context.Context, refsetIDs []string, opts ecl.MemberFilterOpts) (ecl.Set, error) {
	out := ecl.NewSet()
	for _, rid := range refsetIDs {
		for _, mf := range p.spec.MemberFields {
			if mf.Refset != rid || mf.FieldName != opts.FieldName {
				continue
			}
			matched := opts.ValueSet != nil && opts.ValueSet.Contains(mf.Value)
			if opts.Op == "!=" {
				matched = !matched
			}
			if matched {
				out = out.Union(ecl.NewSetFromSlice([]string{mf.Member}))
			}
		}
	}
	return out, nil
}

// isActive reports whether a fixture entity is active (nil pointer = active
// by default, matching SNOMED authoring expectations).
func isActive(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

// contains reports whether haystack contains needle.
func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Optional capabilities
// ---------------------------------------------------------------------------.
//
// The reference provider implements all three so the bundled suite exercises the
// paths they unlock, and so an implementor can read a working example. Each is
// optional: a provider that omits them still conforms, and the evaluator falls
// back or reports the forms it cannot answer.

// PropertiesByGroupBatch answers PropertiesByGroup for many concepts at once,
// collapsing the evaluator's per-concept loop into one call.
func (p *inMemoryProvider) PropertiesByGroupBatch(ctx context.Context, conceptIDs []string) (map[string]map[int][]ecl.Relationship, error) {
	out := make(map[string]map[int][]ecl.Relationship, len(conceptIDs))
	for _, id := range conceptIDs {
		groups, err := p.PropertiesByGroup(ctx, id)
		if err != nil {
			return nil, err
		}
		if len(groups) > 0 {
			out[id] = groups
		}
	}
	return out, nil
}

// ConcreteValuesBatch answers ConcreteValues for many (concept, type) pairs at
// once.
func (p *inMemoryProvider) ConcreteValuesBatch(ctx context.Context, conceptIDs, typeIDs []string) (map[string]map[string][]ecl.ConcreteValue, error) {
	out := make(map[string]map[string][]ecl.ConcreteValue, len(conceptIDs))
	for _, id := range conceptIDs {
		for _, typeID := range typeIDs {
			values, err := p.ConcreteValues(ctx, id, typeID)
			if err != nil {
				return nil, err
			}
			if len(values) == 0 {
				continue
			}
			if out[id] == nil {
				out[id] = make(map[string][]ecl.ConcreteValue)
			}
			out[id][typeID] = values
		}
	}
	return out, nil
}

// InboundRelationships returns the relationships pointing AT each target,
// preserving multiplicity — which is what a cardinality or "!=" on a reverse
// attribute needs and what RelationshipTargets cannot supply.
func (p *inMemoryProvider) InboundRelationships(_ context.Context, targetIDs, typeIDs ecl.Set) (map[string][]ecl.InboundRelationship, error) {
	out := map[string][]ecl.InboundRelationship{}
	for srcID, rels := range p.relsBySource {
		for _, r := range rels {
			if r.TargetID == "" {
				continue // concrete value, not a concept-valued relationship
			}
			if targetIDs != nil && !targetIDs.Contains(r.TargetID) {
				continue
			}
			if typeIDs != nil && typeIDs.Len() > 0 && !typeIDs.Contains(r.TypeID) {
				continue
			}
			out[r.TargetID] = append(out[r.TargetID], ecl.InboundRelationship{
				SourceID: srcID,
				TypeID:   r.TypeID,
			})
		}
	}
	return out, nil
}

// MatchDescriptionNegated evaluates a description filter in which each dimension
// carries its own polarity, negating at the DESCRIPTION ROW level.
//
// Row-level is the whole point. A concept matches when it has at least one
// description satisfying every dimension as written, so a concept holding both an
// FSN and a Spanish synonym satisfies `language != es` through its FSN — whereas
// subtracting the concepts that have a Spanish description would remove it. That
// difference is why the evaluator refuses to compose this from sets.
func (p *inMemoryProvider) MatchDescriptionNegated(_ context.Context, filter ecl.NegatedDescriptionFilterOpts) (ecl.Set, error) {
	opts := filter.Opts
	needle := strings.ToLower(opts.Term)

	out := ecl.NewSet()
	for conceptID, descs := range p.descByConcept {
		for _, d := range descs {
			if !descriptionMatchesNegated(d, filter, needle) {
				continue
			}
			out = out.Union(ecl.NewSetFromSlice([]string{conceptID}))
			break
		}
	}
	return out, nil
}

// descriptionMatchesNegated reports whether ONE description satisfies every
// dimension of the filter, reading each through its Negate flag.
func descriptionMatchesNegated(d FixtureDescription, filter ecl.NegatedDescriptionFilterOpts, needle string) bool {
	opts := filter.Opts

	if opts.Term != "" {
		if termMatches(d.Value, opts.Term, opts.MatchType, needle) == filter.TermNegated {
			return false
		}
	}
	if len(opts.TypeIDs) > 0 {
		if contains(opts.TypeIDs, d.TypeID) == filter.TypeIDsNegated {
			return false
		}
	}
	if len(opts.Languages) > 0 {
		if contains(opts.Languages, d.Language) == filter.LanguagesNegated {
			return false
		}
	}
	// The dimensions with no polarity of their own keep the positive reading.
	if opts.Active != nil && isActive(d.Active) != *opts.Active {
		return false
	}
	if len(opts.ModuleIDs) > 0 && !contains(opts.ModuleIDs, d.ModuleID) {
		return false
	}
	return true
}

// effectiveTimeMatches compares a concept's effectiveTime against a filter.
//
// YYYYMMDD sorts chronologically as text, so the comparison is a plain string
// comparison and needs no date parsing. A concept that declares no effectiveTime
// cannot satisfy any effectiveTime comparison, including "!=": the filter asks
// about a value the concept does not have, and answering true would make
// `effectiveTime != X` select concepts with no effectiveTime at all.
func effectiveTimeMatches(have, want, op string) bool {
	if want == "" {
		return true // no filter on this dimension
	}
	if have == "" {
		return false
	}
	switch op {
	case "", "=":
		return have == want
	case "!=":
		return have != want
	case "<":
		return have < want
	case "<=":
		return have <= want
	case ">":
		return have > want
	case ">=":
		return have >= want
	default:
		return false
	}
}

// ResolveDialectAliases implements ecl.DialectAliasResolver from the fixture's
// dialectAliases declarations.
//
// Lookup is case-insensitive on the alias, because "en-GB" and "en-gb" name the
// same reference set in practice and the capability leaves the choice to the
// implementation. An alias the fixture does not declare is left OUT of the result
// rather than mapped to an empty slice: absent means "cannot resolve", which the
// evaluator reports, while an empty slice would read as "no dialect constraint"
// and widen the query to every dialect.
func (p *inMemoryProvider) ResolveDialectAliases(_ context.Context, aliases []string) (map[string][]string, error) {
	out := map[string][]string{}
	for _, alias := range aliases {
		for declared, ids := range p.spec.DialectAliases {
			if strings.EqualFold(declared, alias) && len(ids) > 0 {
				out[alias] = append([]string(nil), ids...)
				break
			}
		}
	}
	return out, nil
}

// MatchDescriptionByID implements ecl.DescriptionIDProvider.
//
// The id constraint is per description ROW, not per concept: the SAME description
// must carry one of the ids AND satisfy the sibling clauses. Checking the two
// against different descriptions of the same concept would make
// `{{ D id = X, term = "y" }}` match a concept whose FSN has the id and whose
// synonym has the term, which is a different question from the one asked.
func (p *inMemoryProvider) MatchDescriptionByID(_ context.Context, filter ecl.DescriptionIDFilterOpts) (ecl.Set, error) {
	opts := filter.Opts
	needle := strings.ToLower(opts.Term)

	wanted := make(map[string]bool, len(filter.DescriptionIDs))
	for _, id := range filter.DescriptionIDs {
		wanted[id] = true
	}

	out := ecl.NewSet()
	for conceptID, descs := range p.descByConcept {
		for _, d := range descs {
			// A description with no id declared can satisfy "!=" — its id is not
			// among the listed ones — but never "=".
			if wanted[d.ID] == filter.Negate {
				continue
			}
			if !descriptionMatches(d, opts, needle) {
				continue
			}
			out = out.Union(ecl.NewSetFromSlice([]string{conceptID}))
			break
		}
	}
	return out, nil
}

// componentIDFields are the member fields that hold a component id, and so can
// be returned as a Set of concept ids.
//
// Every reference set has referencedComponentId as its first column, and
// association reference sets add targetComponentId. Any other field — mapTarget, mapAdvice,
// mapPriority — holds a value that is not a concept, and projecting it would
// produce a Set of things that are not concept ids.
var componentIDFields = map[string]bool{
	"referencedComponentId": true,
	"targetComponentId":     true,
}

// ProjectRefsetField implements ecl.RefsetFieldProjector.
//
// Filters are applied to the member ROW, then the requested field of the rows
// that survive is returned. That ordering is the whole point: the filter and the
// projection are different columns of the same row, so filtering the projected
// values afterwards would compare the wrong column.
//
// A field this fixture cannot express as a concept id is an ERROR rather than an
// empty result. Empty would read as "no members matched", and a caller would take
// that for an answer.
func (p *inMemoryProvider) ProjectRefsetField(_ context.Context, opts ecl.RefsetProjectionOpts) (ecl.Set, error) {
	if !componentIDFields[opts.Field] {
		// Wrapping the sentinel is what the capability asks for: the evaluator
		// passes it through without adding ErrProvider, so this reads as "the
		// expression cannot be answered" rather than "the backend is unwell".
		return nil, fmt.Errorf("%w: member field %q does not hold a component id, so it cannot be projected into a Set of concept ids",
			ecl.ErrUnsupportedFeature, opts.Field)
	}

	out := ecl.NewSet()
	for _, refsetID := range opts.RefsetIDs {
		for _, row := range p.memberRows(refsetID) {
			if !row.matches(opts.Filters) {
				continue
			}
			if value := row.field(opts.Field); value != "" {
				out = out.Union(ecl.NewSetFromSlice([]string{value}))
			}
		}
	}
	return out, nil
}

// memberRow is ONE reference set member row with the fields declared for it. A
// member with several rows produces several of these.
type memberRow struct {
	member string
	fields map[string]string
}

// field returns one column of the row. The referencedComponentId column is the
// member itself — every reference set's first column, not an extra field someone
// has to declare.
func (r memberRow) field(name string) string {
	if name == "referencedComponentId" {
		return r.member
	}
	return r.fields[name]
}

// matches reports whether every filter holds on this row.
func (r memberRow) matches(filters []ecl.MemberFilterOpts) bool {
	for _, f := range filters {
		matched := f.ValueSet != nil && f.ValueSet.Contains(r.field(f.FieldName))
		if f.Op == "!=" {
			matched = !matched
		}
		if !matched {
			return false
		}
	}
	return true
}

// memberRows assembles the members of one reference set from whichever fixture
// section declares them.
//
// Association reference sets are declared under historicalAssociations, not under
// refsets: the source is the member and the target is its targetComponentId. That
// is where the projection form is actually useful, so reading it from there keeps
// ONE source of truth. Declaring the same rows a second time under refsets and
// memberFields — which is what the first attempt at this did — leaves two copies
// to drift apart, and the copy was already wrong: it named concepts the fixture
// does not have.
func (p *inMemoryProvider) memberRows(refsetID string) []memberRow {
	// Keyed by member AND row, so a member with several rows produces several.
	// Keying by member alone — which the first version did — silently merged the
	// rows of a complex map into one, and a `{{ M ... }}` filter with several
	// clauses would then match a member whose different rows satisfy different
	// clauses. That is the exact bug this whole path exists to avoid, so the
	// fixture must be able to express the shape that exposes it.
	type key struct{ member, row string }
	fields := map[key]map[string]string{}
	get := func(k key) map[string]string {
		if fields[k] == nil {
			fields[k] = map[string]string{}
		}
		return fields[k]
	}

	for _, member := range p.spec.Refsets[refsetID] {
		get(key{member: member})
	}
	for _, mf := range p.spec.MemberFields {
		if mf.Refset == refsetID {
			get(key{member: mf.Member, row: mf.Row})[mf.FieldName] = mf.Value
		}
	}
	for _, h := range p.spec.HistoricalAssociations {
		if h.Refset == refsetID {
			get(key{member: h.Source})["targetComponentId"] = h.Target
		}
	}

	// Sorted, so rows are built in a stable order. The Set does not care, but a
	// fixture that iterates a map is a fixture whose bugs come and go.
	keys := make([]key, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].member != keys[j].member {
			return keys[i].member < keys[j].member
		}
		return keys[i].row < keys[j].row
	})

	rows := make([]memberRow, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, memberRow{member: k.member, fields: fields[k]})
	}
	return rows
}
