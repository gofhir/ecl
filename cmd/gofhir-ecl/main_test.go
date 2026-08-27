package main

import (
	"bytes"
	"errors"
	"flag"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gofhir/ecl/ecl"
)

// The package was at 0% coverage despite the run*WithOutput seams existing for
// exactly this purpose.

// repoRoot resolves the repository root from this file's path.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	// .../go-ecl/cmd/gofhir-ecl/main_test.go → .../go-ecl
	return filepath.Join(filepath.Dir(file), "..", "..")
}

func standardFixture(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "ecl", "providertest", "testdata", "fixtures", "standard.yaml")
}

func TestExitCode(t *testing.T) {
	parseErr := func() error {
		_, err := ecl.Parse("11687002 GARBAGE")
		require.Error(t, err)
		return err
	}()

	tests := map[string]struct {
		err  error
		want int
	}{
		"nil":                 {nil, exitOK},
		"help is not failure": {flag.ErrHelp, exitOK},
		"usage":               {errUsage, exitUsage},
		"parse error":         {parseErr, exitInvalidECL},
		"wrapped parse error": {errors.Join(errors.New("outer"), parseErr), exitInvalidECL},
		"unsupported":         {ecl.ErrUnsupportedFeature, exitUnsupported},
		"generic":             {errors.New("boom"), exitError},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, exitCode(tc.err))
		})
	}
}

func TestRunValidate(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, runValidateWithOutput([]string{"<< 404684003"}, &out))
	assert.Equal(t, "OK\n", out.String())

	// An invalid expression must fail and print nothing to the result stream.
	out.Reset()
	err := runValidateWithOutput([]string{"11687002 GARBAGE"}, &out)
	require.Error(t, err)
	assert.Empty(t, out.String(), "diagnostics must not go to the result stream")
	var pe *ecl.ParseError
	assert.ErrorAs(t, err, &pe)

	// A missing argument is a usage error, not a runtime one.
	out.Reset()
	err = runValidateWithOutput(nil, &out)
	require.Error(t, err)
	assert.ErrorIs(t, err, errUsage)

	// -h is not a failure.
	out.Reset()
	err = runValidateWithOutput([]string{"-h"}, &out)
	assert.ErrorIs(t, err, flag.ErrHelp)
	assert.Equal(t, exitOK, exitCode(err))
}

func TestRunExplain(t *testing.T) {
	var out bytes.Buffer
	require.NoError(t, runExplainWithOutput([]string{"<< 404684003"}, &out))
	assert.Contains(t, out.String(), "DescendantOrSelfOf")
	assert.Contains(t, out.String(), "404684003")

	// The AST rendering must distinguish AND from OR, which a flat listing
	// could not.
	out.Reset()
	require.NoError(t, runExplainWithOutput([]string{"< 404684003 : 363698007 = 74281007 OR 116676008 = 55641003"}, &out))
	assert.Contains(t, out.String(), "OR")

	out.Reset()
	require.NoError(t, runExplainWithOutput([]string{"< 404684003 : 363698007 = 74281007 , 116676008 = 55641003"}, &out))
	assert.Contains(t, out.String(), "AND")

	out.Reset()
	require.Error(t, runExplainWithOutput([]string{"<< invalid!!!"}, &out))
}

func TestRunEval(t *testing.T) {
	var out bytes.Buffer
	err := runEvalWithOutput([]string{"--fixture", standardFixture(t), "<< 73211009"}, &out)
	require.NoError(t, err)
	lines := strings.Fields(out.String())
	assert.ElementsMatch(t, []string{"11687002", "73211009"}, lines)

	// A feature this build cannot evaluate must be classifiable, not silently
	// answered with the empty set. A reverse attribute inside an attribute group
	// is the example because its rejection is permanent by design — grouping has
	// no meaning there — rather than pending a provider capability, so this
	// assertion cannot be invalidated by a provider growing one.
	out.Reset()
	err = runEvalWithOutput([]string{"--fixture", standardFixture(t), "* : { R 363698007 = 22298006 }"}, &out)
	require.Error(t, err)
	assert.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
	assert.Equal(t, exitUnsupported, exitCode(err))

	// A missing fixture reports the path.
	out.Reset()
	err = runEvalWithOutput([]string{"--fixture", "/nonexistent/fixture.yaml", "<< 73211009"}, &out)
	require.Error(t, err)
}

func TestRunConformance(t *testing.T) {
	// No flags: the suite is embedded, so this must work from any directory --
	// which is the case a `go install`ed binary hits, and which used to fail with
	// "no such file or directory".
	var out bytes.Buffer
	require.NoError(t, runConformanceWithOutput(nil, &out))
	assert.Contains(t, out.String(), "0 failed")

	// A caller may still point at a directory of their own suites.
	out.Reset()
	root := repoRoot(t)
	err := runConformanceWithOutput([]string{
		"-suites", filepath.Join(root, "ecl", "providertest", "testdata", "cases"),
		"-fixtures", filepath.Join(root, "ecl", "providertest", "testdata", "fixtures"),
	}, &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "0 failed")

	// The filter narrows the run.
	out.Reset()
	require.NoError(t, runConformanceWithOutput([]string{"-filter", "^descendantOrSelfOf", "-v"}, &out))
	assert.Contains(t, out.String(), "PASS")
}
