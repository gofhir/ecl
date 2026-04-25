package mrcm

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_BasicModel(t *testing.T) {
	data := `{
		"domains": [
			{
				"attributeId": "363698007",
				"domainEcl": "<< 404684003",
				"grouped": true,
				"cardinality": "0..*",
				"inGroupCardinality": "0..1",
				"ruleStrengthId": "723597001"
			}
		],
		"ranges": [
			{
				"attributeId": "363698007",
				"rangeEcl": "<< 442083009",
				"ruleStrengthId": "723597001"
			}
		]
	}`
	m, err := LoadFromBytes([]byte(data))
	require.NoError(t, err)
	require.NotNil(t, m)

	require.Len(t, m.Domains, 1)
	d := m.Domains[0]
	assert.Equal(t, "363698007", d.AttributeID)
	assert.Equal(t, "<< 404684003", d.DomainECL)
	assert.True(t, d.Grouped)
	assert.Equal(t, Cardinality{Min: 0, Max: -1}, d.Cardinality)
	assert.Equal(t, Cardinality{Min: 0, Max: 1}, d.InGroupCardinality)
	assert.Equal(t, RuleStrengthMandatory, d.RuleStrengthID)

	require.Len(t, m.Ranges, 1)
	r := m.Ranges[0]
	assert.Equal(t, "363698007", r.AttributeID)
	assert.Equal(t, "<< 442083009", r.RangeECL)
	assert.Equal(t, RuleStrengthMandatory, r.RuleStrengthID)
}

func TestLoad_MultipleAttributes(t *testing.T) {
	data := `{
		"domains": [
			{"attributeId": "363698007", "domainEcl": "<< 404684003", "grouped": true,  "cardinality": "0..*", "inGroupCardinality": "0..1"},
			{"attributeId": "246075003", "domainEcl": "<< 404684003", "grouped": false, "cardinality": "0..1", "inGroupCardinality": "0..1"},
			{"attributeId": "116676008", "domainEcl": "<< 404684003", "grouped": false, "cardinality": "0..*", "inGroupCardinality": "0..1"}
		],
		"ranges": [
			{"attributeId": "363698007", "rangeEcl": "<< 442083009"},
			{"attributeId": "246075003", "rangeEcl": "<< 410607006"},
			{"attributeId": "116676008", "rangeEcl": "<< 49755003"}
		]
	}`
	m, err := LoadFromBytes([]byte(data))
	require.NoError(t, err)

	assert.Len(t, m.Domains, 3)
	assert.Len(t, m.Ranges, 3)

	// All ranges should default to mandatory rule strength.
	for _, r := range m.Ranges {
		assert.Equal(t, RuleStrengthMandatory, r.RuleStrengthID)
	}
}

func TestLoad_Cardinality(t *testing.T) {
	cases := []struct {
		in   string
		want Cardinality
	}{
		{"0..*", Cardinality{Min: 0, Max: -1}},
		{"1..1", Cardinality{Min: 1, Max: 1}},
		{"0..1", Cardinality{Min: 0, Max: 1}},
		{"1..*", Cardinality{Min: 1, Max: -1}},
		{"", Cardinality{Min: 0, Max: -1}},
		{"  2..5  ", Cardinality{Min: 2, Max: 5}},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, err := parseCardinality(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLoad_Cardinality_Invalid(t *testing.T) {
	cases := []string{
		"abc",
		"1..x",
		"x..1",
		"-1..1",
		"5..2",
		"1",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			_, err := parseCardinality(in)
			assert.Error(t, err)
		})
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	data := `{"domains": [{"attributeId": "363698007"`
	_, err := LoadFromJSON(strings.NewReader(data))
	assert.Error(t, err)
}

func TestLoad_InvalidSCTID(t *testing.T) {
	// 12345678 fails the Verhoeff check.
	data := `{
		"domains": [
			{"attributeId": "12345678", "domainEcl": "<< 404684003"}
		]
	}`
	_, err := LoadFromBytes([]byte(data))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid attributeId")

	dataR := `{
		"ranges": [
			{"attributeId": "12345678", "rangeEcl": "<< 442083009"}
		]
	}`
	_, err = LoadFromBytes([]byte(dataR))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid attributeId")
}

func TestFindDomains(t *testing.T) {
	m := &Model{
		Domains: []AttributeDomain{
			{AttributeID: "363698007", DomainECL: "<< 404684003"},
			{AttributeID: "246075003", DomainECL: "<< 404684003"},
			{AttributeID: "363698007", DomainECL: "<< 71388002"}, // second rule for same attr
		},
	}
	got := m.FindDomains("363698007")
	assert.Len(t, got, 2)

	got = m.FindDomains("246075003")
	assert.Len(t, got, 1)

	got = m.FindDomains("999999999")
	assert.Empty(t, got)

	// Nil safety.
	var nm *Model
	assert.Nil(t, nm.FindDomains("363698007"))
}

func TestFindRanges(t *testing.T) {
	m := &Model{
		Ranges: []AttributeRange{
			{AttributeID: "363698007", RangeECL: "<< 442083009"},
			{AttributeID: "246075003", RangeECL: "<< 410607006"},
		},
	}
	got := m.FindRanges("363698007")
	assert.Len(t, got, 1)
	assert.Equal(t, "<< 442083009", got[0].RangeECL)

	got = m.FindRanges("999999999")
	assert.Empty(t, got)

	var nm *Model
	assert.Nil(t, nm.FindRanges("363698007"))
}
