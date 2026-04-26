package conformance

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Suite is the on-disk YAML schema for a set of conformance cases.
type Suite struct {
	// Name is the suite's human-readable label.
	Name string `yaml:"name"`

	// Fixture names the fixture file (relative to the suite file's
	// directory) that the cases will run against. May be omitted if cases
	// override the fixture per case.
	Fixture string `yaml:"fixture"`

	// SpecSection optionally documents the section of the ECL spec the
	// suite focuses on (e.g. "5.1 Hierarchy operators").
	SpecSection string `yaml:"specSection"`

	Cases []Case `yaml:"cases"`
}

// Case is one conformance case.
type Case struct {
	// Name uniquely identifies the case within its suite. Used in reports.
	Name string `yaml:"name"`

	// Expression is the ECL string under test.
	Expression string `yaml:"expression"`

	// ExpectedIDs is the canonical sorted set of concept IDs the evaluator
	// must return. nil means "any result is acceptable" (use sparingly,
	// only for syntax-only cases routed through the parser).
	ExpectedIDs []string `yaml:"expectedIds"`

	// ExpectError, when true, declares that ecl.Evaluate must return a
	// non-nil error. ExpectedIDs is ignored when this is set.
	ExpectError bool `yaml:"expectError"`

	// Skip lets a case opt out of the run with a documented reason.
	Skip string `yaml:"skip"`

	// SpecSection optionally pins the case to a spec subsection.
	SpecSection string `yaml:"specSection"`
}

// LoadSuite reads and parses a YAML conformance suite from disk.
func LoadSuite(path string) (*Suite, error) {
	b, err := os.ReadFile(path) //nolint:gosec // CLI tool reads bundled paths
	if err != nil {
		return nil, fmt.Errorf("read suite: %w", err)
	}
	var s Suite
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse suite YAML: %w", err)
	}
	return &s, nil
}
