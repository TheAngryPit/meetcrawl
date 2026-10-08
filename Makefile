.DEFAULT_GOAL := help

.PHONY: help test test-race vet tidy tidy-check fmt check

help:
	@printf '%s\n' \
		'Available targets:' \
		'  help        Print available targets (default).' \
		'  test        Run unit tests.' \
		'  test-race   Run tests with the race detector.' \
		'  vet         Run go vet.' \
		'  tidy        Tidy go.mod and go.sum.' \
		'  tidy-check  Fail if go.mod or go.sum need tidying.' \
		'  fmt         Fail if any file needs gofmt.' \
		'  check       Run tidy, fmt, vet, unit tests, and race tests (GOWORK=off).'

test:
	GOWORK=off go test ./...

test-race:
	GOWORK=off go test -race ./...

vet:
	GOWORK=off go vet ./...

tidy:
	GOWORK=off go mod tidy

tidy-check: tidy
	git diff --exit-code -- go.mod go.sum

fmt:
	@set -e; \
	unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then echo "$$unformatted"; exit 1; fi

check: tidy-check fmt vet test test-race
