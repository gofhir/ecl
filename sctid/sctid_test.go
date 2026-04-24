package sctid

import "testing"

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
		"12345",              // too short
		"abc",
		"404684004",          // wrong check digit
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
