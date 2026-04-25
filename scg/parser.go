package scg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofhir/ecl/sctid"
)

// Parse parses an SCG expression and returns the typed AST.
// Whitespace is flexible — most spaces between tokens are optional.
//
// Errors include the byte position where the failure was detected to aid
// debugging of post-coordinated expressions.
func Parse(input string) (*Expression, error) {
	p := &parser{input: input}
	expr, err := p.parseExpression()
	if err != nil {
		return nil, err
	}
	p.skipWS()
	if p.pos < len(p.input) {
		return nil, p.errorf("unexpected trailing input %q", p.input[p.pos:])
	}
	return expr, nil
}

// parser is a hand-written recursive descent parser for SCG.
type parser struct {
	input string
	pos   int
}

// errorf formats an error with the current position prefix.
func (p *parser) errorf(format string, args ...any) error {
	return fmt.Errorf("at position %d: %s", p.pos, fmt.Sprintf(format, args...))
}

// skipWS advances past any whitespace characters (space, tab, CR, LF).
func (p *parser) skipWS() {
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			p.pos++
			continue
		}
		break
	}
}

// peek returns the byte at the current position, or 0 at EOF.
func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

// peekString reports whether the upcoming input starts with s.
func (p *parser) peekString(s string) bool {
	if p.pos+len(s) > len(p.input) {
		return false
	}
	return p.input[p.pos:p.pos+len(s)] == s
}

// consume advances past s if it matches at the current position.
// Returns true on a successful match.
func (p *parser) consume(s string) bool {
	if p.peekString(s) {
		p.pos += len(s)
		return true
	}
	return false
}

// expect advances past s or returns an error if it doesn't match.
func (p *parser) expect(s string) error {
	if !p.consume(s) {
		return p.errorf("expected %q", s)
	}
	return nil
}

// parseExpression parses the top-level production:
//
//	expression = [definitionStatus] subExpression
func (p *parser) parseExpression() (*Expression, error) {
	p.skipWS()

	// Definition status defaults to subtype.
	defStatus := DefStatusSubtype
	if p.consume(DefStatusEquivalent) {
		defStatus = DefStatusEquivalent
	} else if p.consume(DefStatusSubtype) {
		defStatus = DefStatusSubtype
	}

	expr, err := p.parseSubExpression()
	if err != nil {
		return nil, err
	}
	expr.DefinitionStatus = defStatus
	return expr, nil
}

// parseSubExpression parses:
//
//	subExpression = focusConcept *("+" focusConcept) [":" refinement]
func (p *parser) parseSubExpression() (*Expression, error) {
	expr := &Expression{}

	// First focus concept.
	focus, err := p.parseFocusConcept()
	if err != nil {
		return nil, err
	}
	expr.FocusConcepts = append(expr.FocusConcepts, focus)

	// Additional focus concepts.
	for {
		p.skipWS()
		if !p.consume("+") {
			break
		}
		f, err := p.parseFocusConcept()
		if err != nil {
			return nil, err
		}
		expr.FocusConcepts = append(expr.FocusConcepts, f)
	}

	// Optional refinement.
	p.skipWS()
	if p.consume(":") {
		refs, err := p.parseRefinement()
		if err != nil {
			return nil, err
		}
		expr.Refinements = refs
	}

	return expr, nil
}

// parseFocusConcept parses a single focus concept (just a conceptReference).
func (p *parser) parseFocusConcept() (ConceptRef, error) {
	return p.parseConceptRef()
}

// parseConceptRef parses:
//
//	conceptReference = sctId ["|" term "|"]
func (p *parser) parseConceptRef() (ConceptRef, error) {
	p.skipWS()
	id, err := p.parseSCTID()
	if err != nil {
		return ConceptRef{}, err
	}
	cr := ConceptRef{SCTID: id}

	// Optional term in pipes.
	p.skipWS()
	if p.consume("|") {
		term, err := p.parseTerm()
		if err != nil {
			return ConceptRef{}, err
		}
		cr.Term = term
	}
	return cr, nil
}

// parseSCTID reads consecutive digits (6-18 chars) and validates with the
// sctid package's basic length/digit check. The Verhoeff check digit is NOT
// enforced here (validation against the terminology happens in Validate).
//
// We do enforce the length constraint to fail fast on obviously malformed
// inputs like "123" or 25-digit numbers.
func (p *parser) parseSCTID() (string, error) {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.pos++
	}
	if p.pos == start {
		return "", p.errorf("expected SCTID (digits)")
	}
	id := p.input[start:p.pos]
	if len(id) < 6 || len(id) > 18 {
		return "", fmt.Errorf("at position %d: invalid SCTID length %d (expected 6-18 digits): %q", start, len(id), id)
	}
	return id, nil
}

// parseTerm reads characters until the next "|" and trims surrounding spaces.
// The opening "|" has already been consumed.
func (p *parser) parseTerm() (string, error) {
	start := p.pos
	for p.pos < len(p.input) && p.input[p.pos] != '|' {
		p.pos++
	}
	if p.pos >= len(p.input) {
		return "", fmt.Errorf("at position %d: unterminated term (missing closing '|')", start)
	}
	term := p.input[start:p.pos]
	// Consume the closing pipe.
	p.pos++
	return strings.TrimSpace(term), nil
}

// parseRefinement parses the part after ":". A refinement may be either:
//   - a leading ungrouped attributeSet optionally followed by attributeGroups
//   - one or more attributeGroups (brace-enclosed)
//
// The result is a flat list where ungrouped attributes form a single
// AttributeGroup with Grouped=false.
func (p *parser) parseRefinement() ([]AttributeGroup, error) {
	p.skipWS()
	var groups []AttributeGroup

	// First element: either an ungrouped attribute set or a group.
	if p.peek() == '{' {
		grp, err := p.parseAttributeGroup()
		if err != nil {
			return nil, err
		}
		groups = append(groups, grp)
	} else {
		// Ungrouped attribute set: one or more attributes separated by ",".
		attrs, err := p.parseAttributeSet()
		if err != nil {
			return nil, err
		}
		groups = append(groups, AttributeGroup{Grouped: false, Attributes: attrs})
	}

	// Additional groups, each preceded by ",".
	for {
		p.skipWS()
		// We only continue if the comma is followed by a "{" — otherwise the
		// comma belonged to the ungrouped attributeSet (already consumed by
		// parseAttributeSet) and we should stop.
		save := p.pos
		if !p.consume(",") {
			break
		}
		p.skipWS()
		if p.peek() != '{' {
			// Not actually a group — rewind.
			p.pos = save
			break
		}
		grp, err := p.parseAttributeGroup()
		if err != nil {
			return nil, err
		}
		groups = append(groups, grp)
	}

	return groups, nil
}

// parseAttributeGroup parses a brace-enclosed attribute set.
//
//	attributeGroup = "{" attributeSet "}"
func (p *parser) parseAttributeGroup() (AttributeGroup, error) {
	p.skipWS()
	if err := p.expect("{"); err != nil {
		return AttributeGroup{}, err
	}
	attrs, err := p.parseAttributeSet()
	if err != nil {
		return AttributeGroup{}, err
	}
	p.skipWS()
	if err := p.expect("}"); err != nil {
		return AttributeGroup{}, fmt.Errorf("at position %d: unclosed attribute group (expected '}')", p.pos)
	}
	return AttributeGroup{Grouped: true, Attributes: attrs}, nil
}

// parseAttributeSet parses one or more attributes separated by ",".
func (p *parser) parseAttributeSet() ([]Attribute, error) {
	var attrs []Attribute
	first, err := p.parseAttribute()
	if err != nil {
		return nil, err
	}
	attrs = append(attrs, first)

	for {
		p.skipWS()
		// Stop if the next "," is followed by "{" — that's a new attributeGroup.
		save := p.pos
		if !p.consume(",") {
			break
		}
		p.skipWS()
		if p.peek() == '{' {
			// Rewind, let parseRefinement handle the group.
			p.pos = save
			break
		}
		a, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		attrs = append(attrs, a)
	}
	return attrs, nil
}

// parseAttribute parses:
//
//	attribute = attributeName "=" attributeValue
func (p *parser) parseAttribute() (Attribute, error) {
	name, err := p.parseConceptRef()
	if err != nil {
		return Attribute{}, err
	}
	p.skipWS()
	if err := p.expect("="); err != nil {
		return Attribute{}, err
	}
	val, err := p.parseAttributeValue()
	if err != nil {
		return Attribute{}, err
	}
	return Attribute{Name: name, Value: val}, nil
}

// parseAttributeValue parses:
//
//	attributeValue = conceptReference / "(" subExpression ")" / concreteValue
func (p *parser) parseAttributeValue() (AttributeValue, error) {
	p.skipWS()
	c := p.peek()
	switch {
	case c == 0:
		return AttributeValue{}, p.errorf("expected attribute value")
	case c == '(':
		// Nested expression.
		p.pos++ // consume "("
		sub, err := p.parseSubExpression()
		if err != nil {
			return AttributeValue{}, err
		}
		// Nested expressions inherit the default definition status.
		sub.DefinitionStatus = DefStatusSubtype
		p.skipWS()
		if err := p.expect(")"); err != nil {
			return AttributeValue{}, err
		}
		return AttributeValue{Nested: sub}, nil
	case c == '#' || c == '"' || c == 't' || c == 'f':
		// Concrete value.
		cv, err := p.parseConcreteValue()
		if err != nil {
			return AttributeValue{}, err
		}
		return AttributeValue{Concrete: cv}, nil
	case c >= '0' && c <= '9':
		// Concept reference.
		cr, err := p.parseConceptRef()
		if err != nil {
			return AttributeValue{}, err
		}
		return AttributeValue{Concept: &cr}, nil
	default:
		return AttributeValue{}, p.errorf("unexpected character %q in attribute value", c)
	}
}

// parseConcreteValue parses:
//
//	concreteValue = "#" number / "\"" string "\"" / "true" / "false"
func (p *parser) parseConcreteValue() (*ConcreteValue, error) {
	p.skipWS()
	switch p.peek() {
	case '#':
		p.pos++ // consume "#"
		return p.parseNumber()
	case '"':
		p.pos++ // consume opening quote
		return p.parseString()
	case 't':
		if p.consume("true") {
			return &ConcreteValue{Kind: "boolean", Bool: true}, nil
		}
		return nil, p.errorf("expected 'true'")
	case 'f':
		if p.consume("false") {
			return &ConcreteValue{Kind: "boolean", Bool: false}, nil
		}
		return nil, p.errorf("expected 'false'")
	default:
		return nil, p.errorf("expected concrete value")
	}
}

// parseNumber parses an integer or decimal after the "#" prefix.
// Supports an optional leading "-" sign.
func (p *parser) parseNumber() (*ConcreteValue, error) {
	start := p.pos
	if p.peek() == '-' {
		p.pos++
	}
	hasDot := false
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c >= '0' && c <= '9' {
			p.pos++
			continue
		}
		if c == '.' && !hasDot {
			hasDot = true
			p.pos++
			continue
		}
		break
	}
	raw := p.input[start:p.pos]
	if raw == "" || raw == "-" {
		return nil, p.errorf("expected number after '#'")
	}
	if hasDot {
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return nil, p.errorf("invalid decimal %q: %v", raw, err)
		}
		return &ConcreteValue{Kind: "decimal", Float: f}, nil
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, p.errorf("invalid integer %q: %v", raw, err)
	}
	return &ConcreteValue{Kind: "integer", Int: n}, nil
}

// parseString parses a quoted string with basic escape handling.
// The opening quote has already been consumed. Supported escapes: \" and \\.
func (p *parser) parseString() (*ConcreteValue, error) {
	var b strings.Builder
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == '\\' && p.pos+1 < len(p.input) {
			next := p.input[p.pos+1]
			switch next {
			case '"', '\\':
				b.WriteByte(next)
				p.pos += 2
				continue
			default:
				// Unknown escape — keep as-is.
				b.WriteByte(c)
				p.pos++
				continue
			}
		}
		if c == '"' {
			p.pos++ // consume closing quote
			return &ConcreteValue{Kind: "string", String: b.String()}, nil
		}
		b.WriteByte(c)
		p.pos++
	}
	return nil, p.errorf("unterminated string literal")
}

// IsValidSCTID is a convenience re-export of sctid.IsValid for callers that
// have already imported scg.
func IsValidSCTID(id string) bool {
	return sctid.IsValid(id)
}
