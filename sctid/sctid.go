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

// IsValid reports whether a SNOMED CT identifier is structurally valid.
//
// It enforces every rule the SCTID ABNF states:
//
//		sctId = digitNonZero 5*17( digit )
//
//	  - 6 to 18 digits, all decimal;
//	  - a non-zero first digit (leading zeros are not permitted);
//	  - a partition identifier of "00", "01", "02", "10", "11" or "12";
//	  - for the long-form partitions ("1x"), at least 11 digits, since those
//	    carry a 7-digit namespace;
//	  - a valid Verhoeff check digit.
//
// It used to check only the length, the digits and the checksum, despite the
// package doc promising partition rules: IsValid("000000001") returned true, and
// roughly one in ten 8-digit identifiers passed with a partition that cannot
// exist.
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
	if id[0] == '0' {
		return false
	}
	if !validPartition(id[n-3 : n-1]) {
		return false
	}
	// Partitions 10, 11 and 12 are the long form: item identifier + 7-digit
	// namespace + 2-digit partition + check digit, so at least 11 digits.
	if id[n-3] == '1' && n < 11 {
		return false
	}
	return verhoeffCheck(id)
}

// validPartition reports whether a 2-digit partition identifier is one the
// specification defines. The first digit selects short form (0, no namespace) or
// long form (1, namespaced); the second selects the component type.
func validPartition(p string) bool {
	switch p {
	case "00", "01", "02", "10", "11", "12":
		return true
	}
	return false
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
