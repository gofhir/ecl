package ecl_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
)

// officialExamplesDir holds a verbatim copy of the examples/ tree of
// IHTSDO/snomed-expression-constraint-language. See its PROVENANCE.md.
const officialExamplesDir = "testdata/official-examples"

// wantOfficialExamples is how many example files the vendored tree has. It is
// asserted so that a copy that silently loses files — or a file quietly deleted
// to make a failure go away — is caught. Raise it when upstream adds examples.
const wantOfficialExamples = 121

// TestParse_OfficialExamples parses every example SNOMED International publishes
// as valid ECL.
//
// This is the only test in the repository whose expectations were not written
// here. Everything else — including the conformance suite — encodes this
// project's reading of the specification, and a misreading therefore produces a
// green test; that is exactly how several bugs survived earlier review rounds. A
// corpus authored by the body that defines the language cannot be talked into
// agreeing with us: these expressions are declared valid upstream, so a parse
// failure is our defect by construction.
//
// It proves nothing about semantics. No file states what its expression should
// return, so a build that parses all 121 and evaluates each one wrongly is still
// green here.
func TestParse_OfficialExamples(t *testing.T) {
	var files []string
	err := filepath.WalkDir(officialExamplesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".txt") {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err, "the vendored example tree must be readable")
	require.Len(t, files, wantOfficialExamples,
		"example count changed; update wantOfficialExamples after re-copying upstream")

	for _, path := range files {
		t.Run(filepath.ToSlash(strings.TrimPrefix(path, officialExamplesDir+string(filepath.Separator))), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			require.NoError(t, err)

			// One expression per file, some spanning several lines and some
			// carrying /* comments */ — both are valid ECL, so the whole file is
			// the input.
			expr := strings.TrimSpace(string(raw))
			require.NotEmpty(t, expr, "an empty example file is a broken copy, not a passing case")

			tree, err := ecl.Parse(expr)
			require.NoErrorf(t, err, "upstream declares this expression valid:\n%s", expr)
			require.NotNil(t, tree)
		})
	}
}
