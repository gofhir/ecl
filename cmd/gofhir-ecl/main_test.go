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
	return filepath.Join(repoRoot(t), "testdata", "conformance", "fixtures", "standard.yaml")
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
	// answered with the empty set.
	out.Reset()
	err = runEvalWithOutput([]string{"--fixture", standardFixture(t), "<< 404684003 {{ D dialect = en-gb }}"}, &out)
	require.Error(t, err)
	assert.ErrorIs(t, err, ecl.ErrUnsupportedFeature)
	assert.Equal(t, exitUnsupported, exitCode(err))

	// A missing fixture reports the path.
	out.Reset()
	err = runEvalWithOutput([]string{"--fixture", "/nonexistent/fixture.yaml", "<< 73211009"}, &out)
	require.Error(t, err)
}

func TestRunConformance(t *testing.T) {
	var out bytes.Buffer
	root := repoRoot(t)
	err := runConformanceWithOutput([]string{
		"-suites", filepath.Join(root, "testdata", "conformance", "cases"),
		"-fixtures", filepath.Join(root, "testdata", "conformance", "fixtures"),
	}, &out)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "0 failed")
}
