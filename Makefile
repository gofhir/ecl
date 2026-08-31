.PHONY: test test-race lint tidy generate conformance check check-upstream oracle fuzz

# ANTLR_VERSION must match the version the committed parser was generated with,
# which the header of ecl/grammar/ecl_parser.go records. Pin it: generating with
# a different version rewrites 48k lines and makes the diff unreviewable.
ANTLR_VERSION := 4.13.2
ANTLR_JAR := $(HOME)/.cache/antlr/antlr-$(ANTLR_VERSION)-complete.jar
ANTLR_URL := https://www.antlr.org/download/antlr-$(ANTLR_VERSION)-complete.jar

test:
	go test ./...

test-race:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...

lint:
	golangci-lint run

tidy:
	go mod tidy

# Runs the bundled suites through the CLI, which also exercises the binary's own
# path handling.
conformance:
	go run ./cmd/gofhir-ecl conformance

# Everything CI checks, so a failure can be reproduced locally in one command.
#
# vet skips ecl/grammar: ANTLR emits unreachable code by design, and the package
# is regenerated rather than edited. The `generate` target plus the CI diff is
# what guards it.
check: lint test-race conformance
	go mod tidy -diff
	go vet $$(go list ./... | grep -v '/ecl/grammar$$')

# Reports drift between the vendored SNOMED International artefacts (the grammar
# and the official example corpus) and upstream. Needs network. Deliberately NOT
# part of `check`: upstream moving is news, not a local defect, so it runs on a
# schedule in CI instead of blocking a commit.
check-upstream:
	./scripts/check-upstream.sh

# FUZZTIME is how long to fuzz. CI uses 60s as a regression net over the committed
# seed corpus; a real campaign wants minutes or hours.
FUZZTIME ?= 60s

# Fuzzes every entry point that takes untrusted input: both parsers and the MRCM
# loader. A crasher lands in <pkg>/testdata/fuzz/, where it becomes a permanent
# regression case that `make test` replays.
#
# Sequential because -fuzz takes one package at a time, so the wall clock is
# three times FUZZTIME.
fuzz:
	go test ./ecl/ -run '^$$' -fuzz FuzzParse -fuzztime $(FUZZTIME)
	go test ./scg/ -run '^$$' -fuzz FuzzParse -fuzztime $(FUZZTIME)
	go test ./scg/ -run '^$$' -fuzz FuzzParseRenderParse -fuzztime $(FUZZTIME)
	go test ./mrcm/ -run '^$$' -fuzz FuzzLoadFromBytes -fuzztime $(FUZZTIME)

# ORACLE_URL is the terminology server the differential test compares against.
# Any FHIR R4 endpoint serving SNOMED CT works; the CSIRO public server needs no
# credentials, which is why it is the default.
ORACLE_URL ?= https://r4.ontoserver.csiro.au/fhir

# Runs the differential test: every corpus expression is evaluated here AND by the
# server, and the concept sets compared. Needs network. Like check-upstream it is
# NOT part of `check` — it depends on a third party being up, and a divergence
# needs triage rather than a red build.
oracle:
	ECL_ORACLE_URL=$(ORACLE_URL) go test ./internal/oracle/ -run TestDifferential -v -timeout 40m

$(ANTLR_JAR):
	@mkdir -p $(dir $(ANTLR_JAR))
	curl -fsSL -o $(ANTLR_JAR) $(ANTLR_URL)

# Regenerates the parser from ecl/grammar/ECL.g4. Requires a JVM.
#
# The generated files under ecl/grammar/ are committed, so CI can diff them
# against a fresh run: without that gate, an edited grammar that was never
# regenerated, or a hand-edited generated file, would go unnoticed.
generate: $(ANTLR_JAR)
	cd ecl/grammar && java -jar $(ANTLR_JAR) \
		-Dlanguage=Go \
		-package grammar \
		-visitor \
		-listener \
		ECL.g4
	gofmt -w ecl/grammar
