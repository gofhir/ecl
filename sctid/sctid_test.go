package sctid

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValid(t *testing.T) {
	valid := []string{
		"404684003",          // Clinical finding
		"22298006",           // Myocardial infarction
		"363698007",          // Finding site
		"74281007",           // Myocardium
		"116680003",          // IS-A
		"138875005",          // SNOMED CT root
		"900000000000003001", // Fully specified name (description)
		"900000000000013009", // Synonym (description)
	}
	for _, id := range valid {
		if !IsValid(id) {
			t.Errorf("IsValid(%q) = false, want true", id)
		}
	}

	invalid := []string{
		"",
		"12345", // too short
		"abc",
		"404684004",           // wrong check digit
		"0000000000000000000", // too long (19 digits)
	}
	for _, id := range invalid {
		if IsValid(id) {
			t.Errorf("IsValid(%q) = true, want false", id)
		}
	}
}

func TestPartitionID(t *testing.T) {
	tests := []struct{ id, want string }{
		{"404684003", "00"},          // concept (short format, partition 00)
		{"2674034019", "01"},         // description (short format, partition 01)
		{"900000000000003017", "01"}, // description (short format, partition 01)
		{"", ""},
	}
	for _, tt := range tests {
		if got := PartitionID(tt.id); got != tt.want {
			t.Errorf("PartitionID(%q) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestIsConcept(t *testing.T) {
	if !IsConcept("404684003") {
		t.Error("expected concept")
	}
	if IsConcept("2674034019") {
		t.Error("expected not concept")
	}
}

func TestIsDescription(t *testing.T) {
	if !IsDescription("2674034019") {
		t.Error("expected description")
	}
	if IsDescription("404684003") {
		t.Error("expected not description")
	}
}

// TestIsValid_PartitionRules covers the rules the package doc promises and that
// IsValid used to ignore entirely: it checked only length, digits and checksum.
func TestIsValid_PartitionRules(t *testing.T) {
	// Real SCTIDs must keep passing.
	for _, id := range []string{"404684003", "73211009", "138875005", "22298006", "900000000000003001"} {
		assert.Truef(t, IsValid(id), "%s is a real SCTID", id)
	}

	// A leading zero is forbidden by the ABNF (sctId = digitNonZero 5*17 digit).
	assert.False(t, IsValid("000000001"), "leading zero must be rejected")

	// Partition "99" does not exist; only 00/01/02/10/11/12 do.
	assert.False(t, IsValid("123995"), "partition 99 does not exist")

	// The long-form partitions carry a 7-digit namespace, so they need >= 11
	// digits. An 8-digit id claiming partition 10 is structurally impossible.
	assert.False(t, IsConcept("100108"), "partition 10 requires a 7-digit namespace")
}

// TestPartitionAccessors checks the component-type helpers against the partition
// of a real concept identifier.
func TestPartitionAccessors(t *testing.T) {
	assert.Equal(t, "00", PartitionID("404684003"))
	assert.True(t, IsConcept("404684003"))
	assert.False(t, IsDescription("404684003"))
	assert.False(t, IsRelationship("404684003"))

	// An invalid id has no partition.
	assert.Empty(t, PartitionID("000000001"))
	assert.False(t, IsConcept("000000001"))
}
