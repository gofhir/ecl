package mrcm

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

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
// Cardinality strings ("0..*", "1..1", etc.) are parsed into Cardinality
// structs. An empty cardinality string defaults to {0, -1} (i.e. "0..*").
//
// All attributeId values are validated as SCTIDs (Verhoeff check).
func LoadFromJSON(r io.Reader) (*Model, error) {
	var jm jsonModel
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&jm); err != nil {
		return nil, fmt.Errorf("mrcm: decode JSON: %w", err)
	}

	model := &Model{
		Domains: make([]AttributeDomain, 0, len(jm.Domains)),
		Ranges:  make([]AttributeRange, 0, len(jm.Ranges)),
	}

	for i, jd := range jm.Domains {
		if !sctid.IsValid(jd.AttributeID) {
			return nil, fmt.Errorf("mrcm: domains[%d]: invalid attributeId %q", i, jd.AttributeID)
		}
		card, err := parseCardinality(jd.Cardinality)
		if err != nil {
			return nil, fmt.Errorf("mrcm: domains[%d].cardinality: %w", i, err)
		}
		inCard, err := parseCardinality(jd.InGroupCardinality)
		if err != nil {
			return nil, fmt.Errorf("mrcm: domains[%d].inGroupCardinality: %w", i, err)
		}
		strength := jd.RuleStrengthID
		if strength == "" {
			strength = RuleStrengthMandatory
		} else if !sctid.IsValid(strength) {
			return nil, fmt.Errorf("mrcm: domains[%d]: invalid ruleStrengthId %q", i, strength)
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
		strength := jr.RuleStrengthID
		if strength == "" {
			strength = RuleStrengthMandatory
		} else if !sctid.IsValid(strength) {
			return nil, fmt.Errorf("mrcm: ranges[%d]: invalid ruleStrengthId %q", i, strength)
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

	min, err := strconv.Atoi(minStr)
	if err != nil {
		return Cardinality{}, fmt.Errorf("invalid cardinality min %q: %w", minStr, err)
	}
	if min < 0 {
		return Cardinality{}, fmt.Errorf("cardinality min must be >= 0, got %d", min)
	}

	var max int
	if maxStr == "*" {
		max = -1
	} else {
		m, err := strconv.Atoi(maxStr)
		if err != nil {
			return Cardinality{}, fmt.Errorf("invalid cardinality max %q: %w", maxStr, err)
		}
		if m < min {
			return Cardinality{}, fmt.Errorf("cardinality max %d less than min %d", m, min)
		}
		max = m
	}

	return Cardinality{Min: min, Max: max}, nil
}
