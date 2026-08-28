package oracle

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gofhir/ecl/ecl"
)

// Sentinels for the three ways this harness fails to produce a comparison. They
// are distinct because they call for different responses: fix the corpus, triage
// the server, or ignore a flake.
var (
	// ErrNotAnswerable means the expression needs a DataProvider method this
	// harness cannot answer through a terminology API — see the table on
	// Provider. A corpus case that hits it is out of scope, not failing.
	ErrNotAnswerable = errors.New("oracle: not answerable through a FHIR terminology API")

	// ErrTooLarge means an intermediate set exceeded maxExpansion. Bound the
	// corpus expression; do not raise the cap to make it pass.
	ErrTooLarge = errors.New("oracle: intermediate set too large")

	// ErrServerRejected means the server refused or answered unintelligibly.
	ErrServerRejected = errors.New("oracle: server rejected the request")

	// ErrUnreachable means the network or the server failed. Not a finding.
	ErrUnreachable = errors.New("oracle: server unreachable")
)

// Provider is an ecl.DataProvider answered by a FHIR terminology server.
//
// Each primitive becomes a small ECL query the server evaluates, which is what
// makes the differential test meaningful: the server supplies the FACTS and this
// library supplies only the COMPOSITION. A divergence therefore localizes to
// composition — refinement, cardinality, negation, grouping, set algebra — which
// is where every semantic bug this project has found actually lived.
//
// # What it cannot answer
//
// A terminology API is not a SNOMED CT release file, so several methods have no
// faithful translation. They return ErrNotAnswerable rather than an approximation,
// because an approximation would produce divergences that look like defects:
//
//	AllConcepts               half a million codes; also makes `*` unbounded
//	MatchDescription          $expand can filter terms, but not with ECL's
//	                          word-prefix semantics, per-row negation or the
//	                          type/language/acceptability axes combined
//	FilterConcepts            module and effectiveTime are not exposed per concept
//	RefsetsContainingMembers  the server rejects the ^R operator
//	ResolveIdentifier         needs alternate identifier refsets
//	MatchDialect              acceptability is not exposed per description
//	RefsetMembersFiltered     member fields are not exposed
//
// So the corpus covers hierarchy, refinement, cardinality, negation, grouping,
// reverse attributes, dot notation and set algebra. That is the part of the
// language whose semantics are contested; filters are mostly a matter of passing
// options through to a provider.
type Provider struct {
	client *Client
}

// NewProvider returns a Provider backed by the given client.
func NewProvider(c *Client) *Provider { return &Provider{client: c} }

// Compile-time proof that the harness exercises the real interface and the two
// capabilities that make it affordable.
var (
	_ ecl.DataProvider                = (*Provider)(nil)
	_ ecl.BatchPropertiesProvider     = (*Provider)(nil)
	_ ecl.BatchConcreteValuesProvider = (*Provider)(nil)
)

// ── Hierarchy ────────────────────────────────────────────────────────────────.

// Descendants is `<< ids` on the server, or `< ids` without self.
func (p *Provider) Descendants(ctx context.Context, conceptIDs []string, includeSelf bool) (ecl.Set, error) {
	return p.hierarchy(ctx, conceptIDs, "<<", "<", includeSelf)
}

// Ancestors is `>> ids`, or `> ids` without self.
func (p *Provider) Ancestors(ctx context.Context, conceptIDs []string, includeSelf bool) (ecl.Set, error) {
	return p.hierarchy(ctx, conceptIDs, ">>", ">", includeSelf)
}

// Children asks the server for the depth-1 child operator, with or without self.
func (p *Provider) Children(ctx context.Context, conceptIDs []string, includeSelf bool) (ecl.Set, error) {
	return p.hierarchy(ctx, conceptIDs, "<<!", "<!", includeSelf)
}

// Parents asks the server for the depth-1 parent operator, with or without self.
func (p *Provider) Parents(ctx context.Context, conceptIDs []string, includeSelf bool) (ecl.Set, error) {
	return p.hierarchy(ctx, conceptIDs, ">>!", ">!", includeSelf)
}

// hierarchy runs one operator over a disjunction of the inputs. ECL's "OrSelf"
// variants carry includeSelf, so it never has to be emulated with a union.
func (p *Provider) hierarchy(ctx context.Context, conceptIDs []string, withSelf, withoutSelf string, includeSelf bool) (ecl.Set, error) {
	if len(conceptIDs) == 0 {
		return ecl.NewSet(), nil
	}
	op := withoutSelf
	if includeSelf {
		op = withSelf
	}
	return p.expand(ctx, op+" "+disjunction(conceptIDs))
}

// ── Concepts ─────────────────────────────────────────────────────────────────.

// ConceptExists asks the server which of the inputs it knows, by expanding them
// as a disjunction: a code the server does not have contributes nothing.
func (p *Provider) ConceptExists(ctx context.Context, conceptIDs []string) (ecl.Set, error) {
	if len(conceptIDs) == 0 {
		return ecl.NewSet(), nil
	}
	return p.expand(ctx, disjunction(conceptIDs))
}

// AllConcepts is not answerable; see the table on Provider.
func (p *Provider) AllConcepts(context.Context) (ecl.Set, error) {
	return nil, fmt.Errorf("%w: AllConcepts would expand the whole of SNOMED CT; bound the corpus expression instead of using a bare wildcard focus", ErrNotAnswerable)
}

// ── Attributes ───────────────────────────────────────────────────────────────.

// RelationshipTargets uses ECL dot notation, which is exactly this method:
// `(sources).type` is the set of targets of `type` on `sources`.
func (p *Provider) RelationshipTargets(ctx context.Context, sourceIDs, typeIDs ecl.Set) (ecl.Set, error) {
	if sourceIDs == nil {
		return nil, fmt.Errorf("%w: a wildcard source set means dereferencing every attribute of every concept", ErrNotAnswerable)
	}
	if sourceIDs.Len() == 0 || typeIDs == nil || typeIDs.Len() == 0 {
		return ecl.NewSet(), nil
	}

	// One query per type, unioned: dot notation takes a single attribute, and
	// `(S).(a OR b)` is not the same expression.
	out := ecl.NewSet()
	for _, typeID := range sorted(typeIDs) {
		got, err := p.expand(ctx, "("+disjunction(sorted(sourceIDs))+")."+typeID)
		if err != nil {
			return nil, err
		}
		out = out.Union(got)
	}
	return out, nil
}

// RelationshipSources is the reverse direction, which ECL writes as a refinement
// over a wildcard focus. The wildcard is the SERVER's to resolve here, so it
// costs nothing — unlike AllConcepts, which would have to cross the wire.
func (p *Provider) RelationshipSources(ctx context.Context, targetIDs, typeIDs ecl.Set) (ecl.Set, error) {
	if targetIDs == nil {
		return nil, fmt.Errorf("%w: a wildcard target set", ErrNotAnswerable)
	}
	if targetIDs.Len() == 0 || typeIDs == nil || typeIDs.Len() == 0 {
		return ecl.NewSet(), nil
	}

	out := ecl.NewSet()
	for _, typeID := range sorted(typeIDs) {
		got, err := p.expand(ctx, "* : "+typeID+" = ("+disjunction(sorted(targetIDs))+")")
		if err != nil {
			return nil, err
		}
		out = out.Union(got)
	}
	return out, nil
}

// ── Grouped attributes ───────────────────────────────────────────────────────.

// PropertiesByGroup returns a concept's own inferred relationships, grouped.
//
// The facts come from $lookup with property=*, which reports raw relationship
// targets and the relationship groups they sit in — the same data the server uses
// to evaluate ECL itself. That matters for the harness: both sides of the
// comparison see the same relationships, so a divergence cannot be blamed on the
// provider having been fed something different.
//
// Group numbers are assigned 1..N over the reported groups and 0 to the ungrouped
// set, per the PropertiesByGroup contract. The numbers are arbitrary — only
// co-occurrence within one group is meaningful, which is all the evaluator reads.
func (p *Provider) PropertiesByGroup(ctx context.Context, conceptID string) (map[int][]ecl.Relationship, error) {
	byConcept, err := p.PropertiesByGroupBatch(ctx, []string{conceptID})
	if err != nil {
		return nil, err
	}
	return byConcept[conceptID], nil
}

// PropertiesByGroupBatch implements ecl.BatchPropertiesProvider.
//
// Without it a focus of 300 concepts would be 300 round trips, which is not a
// reasonable thing to do to a public server; with it, it is six batch Bundles.
// The harness therefore also serves as a worked example of why that capability
// exists.
func (p *Provider) PropertiesByGroupBatch(ctx context.Context, conceptIDs []string) (map[string]map[int][]ecl.Relationship, error) {
	ids := append([]string(nil), conceptIDs...)
	sort.Strings(ids)
	if len(ids) == 0 {
		return map[string]map[int][]ecl.Relationship{}, nil
	}
	if len(ids) > maxExpansion {
		return nil, fmt.Errorf("%w: %d concepts need their relationships (cap %d)", ErrTooLarge, len(ids), maxExpansion)
	}

	byConcept, err := p.client.Properties(ctx, ids)
	if err != nil {
		return nil, err
	}

	out := make(map[string]map[int][]ecl.Relationship, len(byConcept))
	for id, props := range byConcept {
		groups := map[int][]ecl.Relationship{}
		for _, a := range props.Ungrouped {
			groups[0] = append(groups[0], relationshipOf(a, 0))
		}
		for i, g := range props.Groups {
			num := i + 1
			for _, a := range g {
				groups[num] = append(groups[num], relationshipOf(a, num))
			}
		}
		out[id] = groups
	}
	return out, nil
}

func relationshipOf(a Attr, groupNum int) ecl.Relationship {
	r := ecl.Relationship{TypeID: a.TypeID, TargetID: a.ValueCode, GroupNum: groupNum}
	if a.Concrete != nil {
		r.ConcreteValue = &ecl.ConcreteValue{Kind: a.Concrete.Kind, Value: a.Concrete.Value}
	}
	return r
}

// ConcreteValues reads concrete values out of the same relationships.
func (p *Provider) ConcreteValues(ctx context.Context, sourceID, typeID string) ([]ecl.ConcreteValue, error) {
	byConcept, err := p.ConcreteValuesBatch(ctx, []string{sourceID}, []string{typeID})
	if err != nil {
		return nil, err
	}
	return byConcept[sourceID][typeID], nil
}

// ConcreteValuesBatch implements ecl.BatchConcreteValuesProvider.
func (p *Provider) ConcreteValuesBatch(ctx context.Context, conceptIDs, typeIDs []string) (map[string]map[string][]ecl.ConcreteValue, error) {
	byConcept, err := p.PropertiesByGroupBatch(ctx, conceptIDs)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]bool, len(typeIDs))
	for _, t := range typeIDs {
		wanted[t] = true
	}

	out := map[string]map[string][]ecl.ConcreteValue{}
	for id, groups := range byConcept {
		for _, rels := range groups {
			for _, r := range rels {
				if r.ConcreteValue == nil || !wanted[r.TypeID] {
					continue
				}
				if out[id] == nil {
					out[id] = map[string][]ecl.ConcreteValue{}
				}
				out[id][r.TypeID] = append(out[id][r.TypeID], *r.ConcreteValue)
			}
		}
	}
	return out, nil
}

// ── Everything a terminology API does not expose ─────────────────────────────.

// MatchDescription is not answerable; see the table on Provider.
func (p *Provider) MatchDescription(context.Context, ecl.DescriptionFilterOpts) (ecl.Set, error) {
	return nil, fmt.Errorf("%w: $expand filters terms, but not with ECL's word-prefix semantics, per-row negation, or the type and language axes combined", ErrNotAnswerable)
}

// FilterConcepts is not answerable; see the table on Provider.
func (p *Provider) FilterConcepts(context.Context, ecl.Set, ecl.ConceptFilterOpts) (ecl.Set, error) {
	return nil, fmt.Errorf("%w: FilterConcepts (module and effectiveTime are not exposed per concept)", ErrNotAnswerable)
}

// RefsetMembers is the "^" operator, which ECL expresses directly.
//
// Several reference sets become a disjunction of "^" terms rather than one "^"
// over a parenthesised disjunction: the obvious `^ (a OR b)` is refused by
// Ontoserver with HTTP 422, and so is `^ (a)` for a single one.
func (p *Provider) RefsetMembers(ctx context.Context, refsetIDs []string) (ecl.Set, error) {
	if len(refsetIDs) == 0 {
		return ecl.NewSet(), nil
	}
	ids := append([]string(nil), refsetIDs...)
	sort.Strings(ids)

	terms := make([]string, 0, len(ids))
	for _, id := range ids {
		terms = append(terms, "^ "+id)
	}
	return p.expand(ctx, strings.Join(terms, " OR "))
}

// RefsetsContainingMembers is not answerable; see the table on Provider.
func (p *Provider) RefsetsContainingMembers(context.Context, []string) (ecl.Set, error) {
	return nil, fmt.Errorf("%w: RefsetsContainingMembers (the server rejects the ^R operator)", ErrNotAnswerable)
}

// historyAssociationParent is the parent of every historical association
// reference set. Expanding its descendants is how the MAX profile discovers the
// full list, rather than hardcoding one: an edition may carry association refsets
// this file has never heard of, and the International release has added two since
// the profiles were specified (PARTIALLY EQUIVALENT TO, POSSIBLY REPLACED BY).
const historyAssociationParent = "900000000000522004"

// historyRefsetsForProfile mirrors the mapping in
// ecl/providertest/fixture.go — this project's recommendation to provider
// implementors, since the profile is resolved provider-side by contract.
//
// Duplicating it here is deliberate and is the point of the exercise: the server
// resolves `{{ +HISTORY-MOD }}` with ITS OWN mapping, so a corpus case comparing
// the two checks the recommendation against a real terminology server rather than
// against itself.
//
// The coupling is MANUAL, and nothing enforces it. The differential test compares
// THIS list against the server; the fixture's list takes no part in the run, so
// editing one and not the other goes unnoticed until someone reads both. Said
// plainly because the first draft of this comment claimed the test would catch
// it, which is not true and is the kind of false reassurance that stops people
// checking.
//
// An empty result means the MAX profile: every association reference set.
func historyRefsetsForProfile(profile string) []string {
	switch profile {
	case "MIN", "HISTORY-MIN":
		return []string{"900000000000527005"} // SAME AS
	case "MOD", "HISTORY-MOD":
		return []string{
			"900000000000527005", // SAME AS
			"900000000000526001", // REPLACED BY
			"900000000000528000", // WAS A
			"900000000000530003", // ALTERNATIVE
		}
	default:
		return nil // MAX, empty, or unknown
	}
}

// HistoricalAssociations returns the INACTIVE concepts that were replaced by the
// given ones, which the specification defines as
//
//	^ 900000000000527005 {{ M targetComponentId = (X) }}
//
// per association reference set of the profile. The direction matters and is the
// opposite of what one might assume: the input concepts are the association
// TARGETS, and the result is the historical concepts pointing at them. A provider
// that expands active concepts to their replacements instead makes
// `{{ +HISTORY }}` a silent no-op for every realistic input, which is a bug this
// project shipped once.
func (p *Provider) HistoricalAssociations(ctx context.Context, conceptIDs ecl.Set, profile string) (ecl.Set, error) {
	if conceptIDs == nil || conceptIDs.Len() == 0 {
		return ecl.NewSet(), nil
	}

	refsets := historyRefsetsForProfile(profile)
	if len(refsets) == 0 {
		all, err := p.expand(ctx, "< "+historyAssociationParent)
		if err != nil {
			return nil, err
		}
		refsets = sorted(all)

		// Discovering nothing is not the same as there being nothing. A server
		// whose edition lacks the association reference set metadata would
		// otherwise make this return the empty set, and the differential test
		// would read that as "this concept has no historical associations" and
		// compare it against the server's real answer — reporting a divergence
		// whose cause is a missing lookup, not the evaluator.
		if len(refsets) == 0 {
			return nil, fmt.Errorf("%w: no association reference sets found under %s, so the %s profile cannot be resolved",
				ErrNotAnswerable, historyAssociationParent, profile)
		}
	}

	targets := disjunction(sorted(conceptIDs))
	out := ecl.NewSet()
	for _, refset := range refsets {
		got, err := p.expand(ctx, "^ "+refset+" {{ M targetComponentId = ("+targets+") }}")
		if err != nil {
			// Named, because at least one server cannot answer this form for
			// every association reference set and the failure is otherwise
			// indistinguishable from a general outage. Measured on
			// r4.ontoserver.csiro.au: 1186924009 |PARTIALLY EQUIVALENT TO| and
			// 1186921001 |POSSIBLY REPLACED BY| — the two most recently added —
			// answer HTTP 500 with a NullPointerException, while the same server
			// evaluates `{{ +HISTORY-MAX }}` itself without trouble.
			//
			// Dropping the failed reference set and carrying on would be worse
			// than reporting: the result would silently omit whatever it holds,
			// and the differential test would then compare a partial answer
			// against a complete one and blame the evaluator.
			return nil, fmt.Errorf("association reference set %s: %w", refset, err)
		}
		out = out.Union(got)
	}
	return out, nil
}

// ResolveIdentifier is not answerable; see the table on Provider.
func (p *Provider) ResolveIdentifier(context.Context, string, string) (ecl.Set, error) {
	return nil, fmt.Errorf("%w: ResolveIdentifier", ErrNotAnswerable)
}

// MatchDialect is not answerable; see the table on Provider.
func (p *Provider) MatchDialect(context.Context, ecl.Set, ecl.DialectFilterOpts) (ecl.Set, error) {
	return nil, fmt.Errorf("%w: MatchDialect (acceptability is not exposed per description)", ErrNotAnswerable)
}

// RefsetMembersFiltered is not answerable; see the table on Provider.
func (p *Provider) RefsetMembersFiltered(context.Context, []string, ecl.MemberFilterOpts) (ecl.Set, error) {
	return nil, fmt.Errorf("%w: RefsetMembersFiltered (member fields are not exposed)", ErrNotAnswerable)
}

// ── helpers ──────────────────────────────────────────────────────────────────.

func (p *Provider) expand(ctx context.Context, expr string) (ecl.Set, error) {
	codes, err := p.client.ExpandECL(ctx, expr)
	if err != nil {
		return nil, err
	}
	return ecl.NewSetFromSlice(codes), nil
}

// disjunction renders concept IDs as an ECL disjunction, sorted so that the
// client's cache sees one key per set rather than one per iteration order.
func disjunction(ids []string) string {
	s := append([]string(nil), ids...)
	sort.Strings(s)
	return strings.Join(s, " OR ")
}

func sorted(s ecl.Set) []string {
	if s == nil {
		return nil
	}
	out := s.Slice()
	sort.Strings(out)
	return out
}
