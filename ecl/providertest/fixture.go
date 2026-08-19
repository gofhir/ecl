package providertest

import (
	"context"
	"fmt"
	"os"
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
	Refset    string `yaml:"refset"`
	Member    string `yaml:"member"`
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
	for refsetID, members := range spec.Refsets {
		set := make(map[string]struct{}, len(members))
		for _, m := range members {
			set[m] = struct{}{}
			if p.refsetsByMember[m] == nil {
				p.refsetsByMember[m] = make(map[string]struct{})
			}
			p.refsetsByMember[m][refsetID] = struct{}{}
		}
		p.refsetMembers[refsetID] = set
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

// globMatch implements '*' glob matching (no '?', no character classes) for
// the in-memory fixture. Sufficient for conformance tests.
func globMatch(s, pattern string) bool {
	if pattern == "" {
		return s == ""
	}
	parts := strings.Split(pattern, "*")
	idx := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		switch i {
		case 0:
			if !strings.HasPrefix(s[idx:], part) {
				return false
			}
			idx += len(part)
		case len(parts) - 1:
			if !strings.HasSuffix(s, part) {
				return false
			}
		default:
			pos := strings.Index(s[idx:], part)
			if pos < 0 {
				return false
			}
			idx += pos + len(part)
		}
	}
	return true
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
		// EffectiveTime intentionally not enforced in fixtures yet.
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
		return []string{
			"900000000000527005", // SAME_AS
			"900000000000528000", // REPLACED_BY
			"900000000000526001", // WAS_A
			"900000000000530003", // MOVED_TO
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

func (p *inMemoryProvider) MatchDialect(_ context.Context, concepts ecl.Set, opts ecl.DialectFilterOpts) (ecl.Set, error) {
	out := ecl.NewSet()
	if concepts == nil {
		return out, nil
	}
	concepts.Iter(func(id string) bool {
		for _, entry := range opts.Dialects {
			for _, d := range p.spec.Dialects {
				if d.ConceptID != id {
					continue
				}
				if !contains(entry.DialectIDs, d.DialectID) {
					continue
				}
				if len(entry.AcceptabilityIDs) > 0 && !contains(entry.AcceptabilityIDs, d.AcceptabilityID) {
					continue
				}
				out = out.Union(ecl.NewSetFromSlice([]string{id}))
				return true
			}
		}
		return true
	})
	return out, nil
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
