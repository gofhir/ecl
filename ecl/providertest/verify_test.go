package providertest_test

import (
	"testing"

	"github.com/gofhir/ecl/ecl"
	"github.com/gofhir/ecl/ecl/providertest"
)

// TestVerify_BundledFixture exercises Verify exactly as a third party would, and
// proves the reference fixture satisfies the contract the suite states.
//
// This is the entry point the README used to advertise as reusable while the code
// sat under internal/, where Go forbids importing it.
func TestVerify_BundledFixture(t *testing.T) {
	providertest.Verify(t, func() ecl.DataProvider {
		p, err := providertest.BundledFixture("standard.yaml")
		if err != nil {
			t.Fatalf("loading bundled fixture: %v", err)
		}
		return p
	})
}
