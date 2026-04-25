// Package sctid validates SNOMED CT identifiers (SCTIDs) including Verhoeff checksum and partition rules.
package sctid

// Verhoeff multiplication table (d).
var d = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 2, 3, 4, 0, 6, 7, 8, 9, 5},
	{2, 3, 4, 0, 1, 7, 8, 9, 5, 6},
	{3, 4, 0, 1, 2, 8, 9, 5, 6, 7},
	{4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
	{5, 9, 8, 7, 6, 0, 4, 3, 2, 1},
	{6, 5, 9, 8, 7, 1, 0, 4, 3, 2},
	{7, 6, 5, 9, 8, 2, 1, 0, 4, 3},
	{8, 7, 6, 5, 9, 3, 2, 1, 0, 4},
	{9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

// Verhoeff permutation table (p).
var p = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9},
	{1, 5, 7, 6, 2, 8, 3, 0, 9, 4},
	{5, 8, 0, 3, 7, 9, 6, 1, 4, 2},
	{8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
	{9, 4, 5, 3, 1, 2, 6, 8, 7, 0},
	{4, 2, 8, 6, 5, 7, 3, 9, 0, 1},
	{2, 7, 9, 3, 8, 0, 6, 4, 1, 5},
	{7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}

// verhoeffCheck returns true if the digit string passes the Verhoeff check
// (i.e. the check digit is valid, meaning the final accumulator equals 0).
func verhoeffCheck(s string) bool {
	c := 0
	n := len(s)
	for i := 0; i < n; i++ {
		digit := int(s[n-1-i] - '0')
		c = d[c][p[i%8][digit]]
	}
	return c == 0
}

// IsValid checks if a SNOMED CT identifier is valid.
// An SCTID is 6-18 digits and passes the Verhoeff check digit algorithm.
func IsValid(id string) bool {
	n := len(id)
	if n < 6 || n > 18 {
		return false
	}
	for i := 0; i < n; i++ {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
	}
	return verhoeffCheck(id)
}

// PartitionID returns the 2-digit partition identifier from an SCTID.
// The partition is the penultimate 2 digits before the check digit.
// Returns empty string for invalid IDs.
func PartitionID(id string) string {
	if !IsValid(id) {
		return ""
	}
	n := len(id)
	return id[n-3 : n-1]
}

// IsConcept returns true if the SCTID is a concept identifier (partition "00" or "10").
func IsConcept(id string) bool {
	p := PartitionID(id)
	return p == "00" || p == "10"
}

// IsDescription returns true if the SCTID is a description identifier (partition "01" or "11").
func IsDescription(id string) bool {
	p := PartitionID(id)
	return p == "01" || p == "11"
}

// IsRelationship returns true if the SCTID is a relationship identifier (partition "02" or "12").
func IsRelationship(id string) bool {
	p := PartitionID(id)
	return p == "02" || p == "12"
}
