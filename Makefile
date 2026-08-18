.PHONY: test test-race lint tidy generate conformance check

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
