package mrcm

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/gofhir/ecl/scg"
)

// concreteRange is the range of a concrete attribute, written in the MRCM
// Attribute Range refset with the concreteValueReplacement rule of the SNOMED
// CT Template Syntax (v1.1) rather than in ECL: `int(>#0..)`, `dec(#0.5..#10)`,
// `str("mg" "mL")`, `bool(true)`. With no set, every value of the type is in
// range.
type concreteRange struct {
	kind    string // "integer", "decimal", "string" or "boolean", as scg.ConcreteValue.Kind
	any     bool   // no set: every value of the kind
	numbers []numberItem
	strings []string
	bools   []bool
}

// numberItem is one value of an integer or decimal set, or one range. A nil
// bound is open.
type numberItem struct {
	min, max         *big.Rat
	minExcl, maxExcl bool
}

// concreteKeywords maps the Template Syntax keywords to scg kinds.
var concreteKeywords = map[string]string{"int": kindInteger, "dec": kindDecimal, "str": kindString, "bool": kindBoolean}

// The kinds of scg.ConcreteValue.
const (
	kindInteger = "integer"
	kindDecimal = "decimal"
	kindString  = "string"
	kindBoolean = "boolean"
)

// parseConcreteRange parses a range constraint in the concrete form. The
// second result reports whether s is in that form at all: an ECL range never
// starts with one of the keywords, so a false is an ECL range for the caller
// to evaluate, and an error is a concrete range that is malformed.
//
// One leniency: the ABNF writes a decimal bound as `#0.0`, but the MRCM refsets
// SNOMED International distributes write `dec(>#0..)`, so a decimal set also
// takes integers.
func parseConcreteRange(s string) (concreteRange, bool, error) {
	p := &rangeParser{s: s}
	p.ws()
	start := p.i
	for p.i < len(p.s) && (p.s[p.i] >= 'a' && p.s[p.i] <= 'z' || p.s[p.i] >= 'A' && p.s[p.i] <= 'Z') {
		p.i++
	}
	// An ABNF literal is case-insensitive.
	kind, ok := concreteKeywords[strings.ToLower(p.s[start:p.i])]
	if !ok {
		return concreteRange{}, false, nil
	}
	r := concreteRange{kind: kind}
	keyword := p.s[start:p.i]
	p.ws()
	if p.eof() {
		r.any = true
		return r, true, p.err
	}
	if !p.take("(") {
		return r, true, p.errorf("expected ( after %q", keyword)
	}
	for first := true; ; first = false {
		hadSpace := p.ws()
		if p.take(")") {
			break
		}
		if p.err != nil {
			return r, true, p.err
		}
		if p.eof() {
			return r, true, p.errorf("expected )")
		}
		if !first && !hadSpace {
			return r, true, p.errorf("values of a set are separated by white space")
		}
		if err := p.item(&r); err != nil {
			return r, true, err
		}
	}
	if len(r.numbers)+len(r.strings)+len(r.bools) == 0 {
		return r, true, p.errorf("an empty set")
	}
	p.ws()
	if !p.eof() {
		return r, true, p.errorf("unexpected %q", p.s[p.i:])
	}
	return r, true, p.err
}

type rangeParser struct {
	s   string
	i   int
	err error // an unterminated comment, which ws cannot return
}

func (p *rangeParser) eof() bool { return p.i >= len(p.s) }

func (p *rangeParser) errorf(format string, args ...any) error {
	return fmt.Errorf("concrete range %q at %d: %s", p.s, p.i, fmt.Sprintf(format, args...))
}

func (p *rangeParser) take(tok string) bool {
	if strings.HasPrefix(p.s[p.i:], tok) {
		p.i += len(tok)
		return true
	}
	return false
}

// ws skips white space and /* comments */, reporting whether there was any.
func (p *rangeParser) ws() bool {
	start := p.i
	for !p.eof() {
		switch {
		case strings.ContainsRune(" \t\r\n", rune(p.s[p.i])):
			p.i++
		case p.take("/*"):
			end := strings.Index(p.s[p.i:], "*/")
			if end < 0 {
				p.err = p.errorf("unterminated comment")
				p.i = len(p.s)
				return true
			}
			p.i += end + 2
		default:
			return p.i > start
		}
	}
	return p.i > start
}

// item parses one value or range of the set, by the range's kind.
func (p *rangeParser) item(r *concreteRange) error {
	switch r.kind {
	case kindString:
		v, err := p.quoted()
		if err != nil {
			return err
		}
		r.strings = append(r.strings, v)
	case kindBoolean:
		switch {
		case p.takeFold("true"):
			r.bools = append(r.bools, true)
		case p.takeFold("false"):
			r.bools = append(r.bools, false)
		default:
			return p.errorf("expected true or false")
		}
	default:
		n, err := p.numberItem(r.kind == kindDecimal)
		if err != nil {
			return err
		}
		r.numbers = append(r.numbers, n)
	}
	return nil
}

func (p *rangeParser) takeFold(word string) bool {
	if len(p.s)-p.i >= len(word) && strings.EqualFold(p.s[p.i:p.i+len(word)], word) {
		p.i += len(word)
		return true
	}
	return false
}

// quoted parses a "string", whose escapes are \" and \\.
func (p *rangeParser) quoted() (string, error) {
	if !p.take(`"`) {
		return "", p.errorf("expected a quoted string")
	}
	var b strings.Builder
	for !p.eof() {
		c := p.s[p.i]
		switch {
		case c == '"' && b.Len() == 0:
			return "", p.errorf("an empty string")
		case c == '"':
			p.i++
			return b.String(), nil
		case c == '\\' && p.i+1 < len(p.s) && (p.s[p.i+1] == '"' || p.s[p.i+1] == '\\'):
			b.WriteByte(p.s[p.i+1])
			p.i += 2
		case c == '\\':
			return "", p.errorf(`only \" and \\ are escapes`)
		default:
			b.WriteByte(c)
			p.i++
		}
	}
	return "", p.errorf("unterminated string")
}

// numberItem parses `#v`, `[>]#v..[[<]#v]` or `..[<]#v`.
func (p *rangeParser) numberItem(decimal bool) (numberItem, error) {
	var n numberItem
	if !p.take("..") {
		n.minExcl = p.take(">")
		v, err := p.number(decimal)
		if err != nil {
			return n, err
		}
		n.min = v
		if !p.take("..") {
			if n.minExcl {
				return n, p.errorf("> needs a range: >#v..")
			}
			n.max = v // a single value
			return n, nil
		}
	}
	// After "..": a maximum, unless the item ends here.
	if p.eof() || strings.ContainsRune(" \t\r\n)/", rune(p.s[p.i])) {
		if n.min == nil {
			return n, p.errorf(".. needs a bound")
		}
		return n, nil
	}
	n.maxExcl = p.take("<")
	v, err := p.number(decimal)
	if err != nil {
		return n, err
	}
	n.max = v
	if n.min != nil && n.empty(decimal) {
		// A range that holds no value is a mistake in the rule.
		return n, p.errorf("the range holds no value")
	}
	return n, nil
}

// number parses `#[-+]integer[.digits]`; the fraction only when decimal.
func (p *rangeParser) number(decimal bool) (*big.Rat, error) {
	if !p.take("#") {
		return nil, p.errorf("expected #")
	}
	start := p.i
	if !p.eof() && (p.s[p.i] == '-' || p.s[p.i] == '+') {
		p.i++
	}
	digits := p.i
	for !p.eof() && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
		p.i++
	}
	switch {
	case p.i == digits:
		return nil, p.errorf("expected digits")
	case p.s[digits] == '0' && p.i-digits > 1:
		return nil, p.errorf("an integer has no leading zero")
	}
	signedZero := digits > start && p.s[digits] == '0' && p.i-digits == 1
	// A fraction is a dot and digits; a dot followed by another is the "..".
	if decimal && strings.HasPrefix(p.s[p.i:], ".") && !strings.HasPrefix(p.s[p.i:], "..") {
		p.i++
		fraction := p.i
		for !p.eof() && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
		}
		if p.i == fraction {
			return nil, p.errorf("expected digits after .")
		}
	}
	v, ok := new(big.Rat).SetString(p.s[start:p.i])
	if !ok {
		return nil, p.errorf("not a number")
	}
	if signedZero && v.Sign() == 0 {
		// integerValue is `digitNonZero *digit / zero`, and only the first takes a sign.
		return nil, p.errorf("zero takes no sign")
	}
	return v, nil
}

// empty reports whether a range with both bounds holds no value: none at all,
// or, for an integer range, no integer between exclusive bounds.
func (n numberItem) empty(decimal bool) bool {
	c := n.min.Cmp(n.max)
	if c > 0 || c == 0 && (n.minExcl || n.maxExcl) {
		return true
	}
	if decimal {
		return false
	}
	// Integer bounds: the lowest and the highest integer in range.
	lo, hi := new(big.Int).Set(n.min.Num()), new(big.Int).Set(n.max.Num())
	if n.minExcl {
		lo.Add(lo, big.NewInt(1))
	}
	if n.maxExcl {
		hi.Sub(hi, big.NewInt(1))
	}
	return lo.Cmp(hi) > 0
}

// contains reports whether a concrete value is in the range. A value of
// another kind is not, except that an integer is a decimal number.
func (r concreteRange) contains(v *scg.ConcreteValue) bool {
	switch r.kind {
	case kindString:
		return v.Kind == kindString && (r.any || containsString(r.strings, v.String))
	case kindBoolean:
		if v.Kind != kindBoolean {
			return false
		}
		for _, b := range r.bools {
			if b == v.Bool {
				return true
			}
		}
		return r.any
	}
	if v.Kind != kindInteger && (r.kind == kindInteger || v.Kind != kindDecimal) {
		return false
	}
	x := concreteRat(v)
	if x == nil {
		return false
	}
	if r.any {
		return true
	}
	for _, n := range r.numbers {
		if n.holds(x) {
			return true
		}
	}
	return false
}

func (n numberItem) holds(x *big.Rat) bool {
	if n.min != nil {
		if c := x.Cmp(n.min); c < 0 || (c == 0 && n.minExcl) {
			return false
		}
	}
	if n.max != nil {
		if c := x.Cmp(n.max); c > 0 || (c == 0 && n.maxExcl) {
			return false
		}
	}
	return true
}

// concreteRat is a numeric value exactly. A decimal goes through its shortest
// text, which is the literal as written for any literal float64 can tell
// apart: the float itself sits a little off 0.1, which would put #0.1 outside
// `dec(#0.1)`. The limit is the scg package's, which keeps a decimal as a
// float64: #1.00000000000000001 arrives as 1.
func concreteRat(v *scg.ConcreteValue) *big.Rat {
	if v.Kind == kindInteger {
		return new(big.Rat).SetInt64(v.Int)
	}
	r, ok := new(big.Rat).SetString(strconv.FormatFloat(v.Float, 'g', -1, 64))
	if !ok {
		// Infinity or NaN, which no SCG literal parses to; out of every range.
		return nil
	}
	return r
}

func containsString(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
