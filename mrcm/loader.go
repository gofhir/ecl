package mrcm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/sctid"
)

// jsonModel is the wire format for an MRCM JSON document.
type jsonModel struct {
	Domains []jsonDomain `json:"domains"`
	Ranges  []jsonRange  `json:"ranges"`
}

type jsonDomain struct {
	AttributeID        string `json:"attributeId"`
	DomainECL          string `json:"domainEcl"`
	Grouped            bool   `json:"grouped"`
	Cardinality        string `json:"cardinality"`
	InGroupCardinality string `json:"inGroupCardinality"`
	RuleStrengthID     string `json:"ruleStrengthId"`
}

type jsonRange struct {
	AttributeID    string `json:"attributeId"`
	RangeECL       string `json:"rangeEcl"`
	RangeRuleECL   string `json:"rangeRuleEcl"`
	RuleStrengthID string `json:"ruleStrengthId"`
}

// LoadFromJSON parses MRCM rules from a JSON document.
//
// Expected JSON structure:
//
//	{
//	  "domains": [
//	    {
//	      "attributeId": "363698007",
//	      "domainEcl": "<< 404684003",
//	      "grouped": true,
//	      "cardinality": "0..*",
//	      "inGroupCardinality": "0..1",
//	      "ruleStrengthId": "723597001"
//	    }
//	  ],
//	  "ranges": [
//	    {
//	      "attributeId": "363698007",
//	      "rangeEcl": "<< 442083009",
//	      "ruleStrengthId": "723597001"
//	    }
//	  ]
//	}
//
// An absent or empty cardinality string means "0..*", i.e. no constraint.
//
// One value is worth calling out: "inGroupCardinality": "0..0" is a NO-OP. The
// validator cannot tell it from the zero value of a Model built in Go, so it
// treats it as unset -- see inGroupSpecified. To say an attribute must never
// appear inside a relationship group, set "grouped": false, which is reported as a
// grouped_violation.
//
// Cardinality strings ("0..*", "1..1", etc.) are parsed into Cardinality
// structs. An empty cardinality string defaults to {0, -1} (i.e. "0..*").
//
// All attributeId values are validated as SCTIDs (Verhoeff check). A
// ruleStrengthId must be empty (mandatory), mandatory or optional; any other
// value is an error, because the validator would otherwise skip the rule as
// not mandatory. A document must hold one JSON value and nothing after it.
func LoadFromJSON(r io.Reader) (*Model, error) {
	var jm jsonModel
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&jm); err != nil {
		return nil, fmt.Errorf("mrcm: decode JSON: %w", err)
	}
	// Decode reads ONE value and stops, so a second model, a concatenated file
	// or stray text after the first would be dropped without a word — and
	// DisallowUnknownFields never sees it.
	if err := onlyWhitespaceLeft(io.MultiReader(dec.Buffered(), r)); err != nil {
		return nil, err
	}

	model := &Model{
		Domains: make([]AttributeDomain, 0, len(jm.Domains)),
		Ranges:  make([]AttributeRange, 0, len(jm.Ranges)),
	}

	for i, jd := range jm.Domains {
		if !sctid.IsValid(jd.AttributeID) {
			return nil, fmt.Errorf("mrcm: domains[%d]: invalid attributeId %q", i, jd.AttributeID)
		}
		// Validate the ECL at load time. Without this a rule with an unparseable
		// (or empty) domainEcl loaded fine and then failed on every validation
		// that touched the attribute -- far from where the mistake was made.
		if err := validateRuleECL(jd.DomainECL); err != nil {
			return nil, fmt.Errorf("mrcm: domains[%d].domainEcl: %w", i, err)
		}
		card, err := parseCardinality(jd.Cardinality)
		if err != nil {
			return nil, fmt.Errorf("mrcm: domains[%d].cardinality: %w", i, err)
		}
		inCard, err := parseCardinality(jd.InGroupCardinality)
		if err != nil {
			return nil, fmt.Errorf("mrcm: domains[%d].inGroupCardinality: %w", i, err)
		}
		strength, err := ruleStrength(jd.RuleStrengthID)
		if err != nil {
			return nil, fmt.Errorf("mrcm: domains[%d]: %w", i, err)
		}
		model.Domains = append(model.Domains, AttributeDomain{
			AttributeID:        jd.AttributeID,
			DomainECL:          jd.DomainECL,
			Grouped:            jd.Grouped,
			Cardinality:        card,
			InGroupCardinality: inCard,
			RuleStrengthID:     strength,
		})
	}

	for i, jr := range jm.Ranges {
		if !sctid.IsValid(jr.AttributeID) {
			return nil, fmt.Errorf("mrcm: ranges[%d]: invalid attributeId %q", i, jr.AttributeID)
		}
		if err := validateRuleECL(jr.RangeECL); err != nil {
			return nil, fmt.Errorf("mrcm: ranges[%d].rangeEcl: %w", i, err)
		}
		strength, err := ruleStrength(jr.RuleStrengthID)
		if err != nil {
			return nil, fmt.Errorf("mrcm: ranges[%d]: %w", i, err)
		}
		model.Ranges = append(model.Ranges, AttributeRange{
			AttributeID:    jr.AttributeID,
			RangeECL:       jr.RangeECL,
			RangeRuleECL:   jr.RangeRuleECL,
			RuleStrengthID: strength,
		})
	}

	return model, nil
}

// onlyWhitespaceLeft reads the rest of the input and reports anything but
// JSON whitespace. It reads byte by byte through a small buffer, so a large
// trailing value costs time, not memory — unlike reading it as a token, which
// buffers a whole string — and a stray "}" or "]", which json.Decoder.More
// does not report, is caught too.
func onlyWhitespaceLeft(r io.Reader) error {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		for _, c := range buf[:n] {
			switch c {
			case ' ', '\t', '\n', '\r':
			default:
				return fmt.Errorf("mrcm: decode JSON: unexpected data after the model: %q", c)
			}
		}
		switch {
		case errors.Is(err, io.EOF):
			return nil
		case err != nil:
			return fmt.Errorf("mrcm: decode JSON: reading after the model: %w", err)
		}
	}
}

// ruleStrength resolves a rule's ruleStrengthId: empty is mandatory, and
// otherwise it must be one of the two strengths the validator knows. A
// well-formed SCTID that is neither would load and then switch the rule off,
// since the validator skips every rule that is not mandatory.
func ruleStrength(id string) (string, error) {
	switch id {
	case "":
		return RuleStrengthMandatory, nil
	case RuleStrengthMandatory, RuleStrengthOptional:
		return id, nil
	}
	return "", fmt.Errorf("invalid ruleStrengthId %q: must be mandatory (%s) or optional (%s)", id, RuleStrengthMandatory, RuleStrengthOptional)
}

// LoadFromBytes is a convenience wrapper around LoadFromJSON.
func LoadFromBytes(data []byte) (*Model, error) {
	return LoadFromJSON(strings.NewReader(string(data)))
}

// parseCardinality parses an MRCM cardinality string of the form "min..max"
// where max is either an integer or "*" for unbounded. Whitespace around the
// numbers and operator is tolerated. An empty string defaults to {0, -1}.
func parseCardinality(s string) (Cardinality, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Cardinality{Min: 0, Max: -1}, nil
	}
	parts := strings.SplitN(s, "..", 2)
	if len(parts) != 2 {
		return Cardinality{}, fmt.Errorf("invalid cardinality %q (expected min..max)", s)
	}
	minStr := strings.TrimSpace(parts[0])
	maxStr := strings.TrimSpace(parts[1])

	minVal, err := strconv.Atoi(minStr)
	if err != nil {
		return Cardinality{}, fmt.Errorf("invalid cardinality min %q: %w", minStr, err)
	}
	if minVal < 0 {
		return Cardinality{}, fmt.Errorf("cardinality min must be >= 0, got %d", minVal)
	}

	var maxVal int
	if maxStr == "*" {
		maxVal = -1
	} else {
		m, err := strconv.Atoi(maxStr)
		if err != nil {
			return Cardinality{}, fmt.Errorf("invalid cardinality max %q: %w", maxStr, err)
		}
		if m < minVal {
			return Cardinality{}, fmt.Errorf("cardinality max %d less than min %d", m, minVal)
		}
		maxVal = m
	}

	return Cardinality{Min: minVal, Max: maxVal}, nil
}

// validateRuleECL checks that a rule's ECL constraint parses, so a malformed rule
// is rejected where it is written rather than on every validation that reaches it.
// An empty constraint is a mistake too: it would evaluate to nothing.
func validateRuleECL(expr string) error {
	if strings.TrimSpace(expr) == "" {
		return fmt.Errorf("must not be empty")
	}
	if _, err := ecl.Parse(expr); err != nil {
		return fmt.Errorf("invalid ECL %q: %w", expr, err)
	}
	return nil
}
