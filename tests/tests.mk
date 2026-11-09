# Make targets for the locked acceptance suite. The root Makefile includes
# this file; CI runs it directly (make -f tests/tests.mk ...), so these
# recipes are the ones that count. Written for GNU Make 3.81 (macOS).

TESTS_GO     ?= go
T            ?=
RESULTS      ?= .score/results.json
SCORE_CSV    ?= docs/metrics/score.csv
RATCHET      ?= docs/metrics/ratchet.txt
SUMMARY      ?=
LOCK_FROM    ?=

TEST_RUN := TestAcceptance$(if $(T),/$(T),)

.PHONY: test score score-update test-harness lock-verify lock-update lock-audit hooks

## test: run the acceptance scripts (T=feature or T=feature/level to narrow)
test:
	cd tests && $(TESTS_GO) test -count=1 -run '$(TEST_RUN)' .

## score: run every script, print the score, append docs/metrics/score.csv, check the ratchet
score:
	@mkdir -p .score
	-cd tests && FLAGWIRE_RESULTS=$(CURDIR)/$(RESULTS) $(TESTS_GO) test -count=1 -run 'TestAcceptance' . > $(CURDIR)/.score/test.log 2>&1
	cd tests && $(TESTS_GO) run ./cmd/score -results $(CURDIR)/$(RESULTS) \
		-csv $(CURDIR)/$(SCORE_CSV) -commit "$$(git rev-parse --short HEAD 2>/dev/null || echo none)" \
		-ratchet $(CURDIR)/$(RATCHET) $(if $(SUMMARY),-md $(SUMMARY),)

## score-update: like score, and add newly passing tests to the ratchet
score-update:
	@mkdir -p .score
	-cd tests && FLAGWIRE_RESULTS=$(CURDIR)/$(RESULTS) $(TESTS_GO) test -count=1 -run 'TestAcceptance' . > $(CURDIR)/.score/test.log 2>&1
	cd tests && $(TESTS_GO) run ./cmd/score -results $(CURDIR)/$(RESULTS) \
		-ratchet $(CURDIR)/$(RATCHET) -update-ratchet

## test-harness: the runner's own tests, go vet, and a lint of every script
test-harness:
	@cd tests && out="$$(gofmt -l .)" && if [ -n "$$out" ]; then echo "gofmt: these files in tests/ need formatting:"; echo "$$out"; exit 1; fi
	cd tests && $(TESTS_GO) vet ./...
	cd tests && $(TESTS_GO) test -count=1 ./internal/... && $(TESTS_GO) test -count=1 -run 'TestScripts' .

## lock-verify: locked files match tests/SHA256SUMS
lock-verify:
	cd tests && $(TESTS_GO) run ./cmd/lock verify

## lock-update: rewrite tests/SHA256SUMS (a logged intervention; commit with a Locked-change: trailer)
lock-update:
	cd tests && $(TESTS_GO) run ./cmd/lock update

## lock-audit: every commit after LOCK_FROM that changes a locked file has a Locked-change: trailer
lock-audit:
	cd tests && $(TESTS_GO) run ./cmd/lock audit $(if $(LOCK_FROM),$(LOCK_FROM),$$(git rev-list --max-parents=0 HEAD | tail -1)) HEAD

## hooks: use .githooks/ (the pre-commit hook refuses changes to locked files)
hooks:
	git config core.hooksPath .githooks
