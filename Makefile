# flagwire. Run `make help` for the targets. The acceptance-suite targets
# (test, score, lock-*) live in tests/tests.mk, which is locked.
# Written for GNU Make 3.81 (macOS) as well as newer versions.

GO  ?= go
BIN ?= bin/flagwire

include tests/tests.mk

.DEFAULT_GOAL := help

.PHONY: help build unit lint check fmt

## help: list the targets
help:
	@grep -h -E '^## ' Makefile tests/tests.mk | sed 's/^## /  /'

## build: build bin/flagwire from src/
build:
	@test -f src/go.mod || { echo "src/go.mod does not exist yet (slice S0 creates it)"; exit 1; }
	cd src && $(GO) build -o ../$(BIN) ./cmd/flagwire

## unit: flagwire's own unit tests (src/)
unit:
	@if [ -f src/go.mod ]; then cd src && $(GO) test -count=1 ./...; else echo "unit: no src/go.mod yet"; fi

## lint: gofmt and go vet on src/
lint:
	@if [ -f src/go.mod ]; then \
		out="$$(cd src && gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt: these files in src/ need formatting:"; echo "$$out"; exit 1; fi; \
		cd src && $(GO) vet ./...; \
	else echo "lint: no src/go.mod yet"; fi

## fmt: gofmt -w on src/
fmt:
	@if [ -f src/go.mod ]; then cd src && gofmt -w .; fi

## check: lint, unit tests, the test harness and the lock (run before every commit)
check: lint unit test-harness lock-verify
