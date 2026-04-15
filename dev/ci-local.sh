#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLS_BIN="$ROOT_DIR/.tools/bin"
GOLANGCI_LINT_VERSION="${GOLANGCI_LINT_VERSION:-v2.11.4}"
GOVULNCHECK_VERSION="${GOVULNCHECK_VERSION:-v1.1.4}"

trap 'rm -f "$ROOT_DIR/coverage.out"' EXIT

export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.26.2}"
export PATH="$TOOLS_BIN:$PATH"

step() {
	printf '\n==> %s\n' "$1"
}

install_go_tool() {
	local binary="$1"
	local package="$2"
	local version="$3"
	local version_output

	if [[ -x "$TOOLS_BIN/$binary" ]]; then
		version_output="$("$TOOLS_BIN/$binary" version 2>&1 || true)"
		if grep -Fq "$version" <<<"$version_output"; then
			return
		fi
	fi

	mkdir -p "$TOOLS_BIN"
	GOBIN="$TOOLS_BIN" go install "$package@$version"
}

cd "$ROOT_DIR"

step "Installing pinned tooling"
install_go_tool golangci-lint github.com/golangci/golangci-lint/v2/cmd/golangci-lint "$GOLANGCI_LINT_VERSION"
install_go_tool govulncheck golang.org/x/vuln/cmd/govulncheck "$GOVULNCHECK_VERSION"

step "Tidying and verifying modules"
go mod tidy
go mod verify

step "Checking formatting"
if [[ -n "$(gofmt -s -l .)" ]]; then
	echo "gofmt -s found unformatted files"
	gofmt -s -l .
	exit 1
fi

step "Building"
go build ./...

step "Running go vet"
go vet ./...

step "Running tests"
go test ./... -race -shuffle=on -count=1 -covermode=atomic -coverprofile=coverage.out
go tool cover -func=coverage.out | awk -v thr="${COVER_THRESH:-0}" '
/^total:/ {
  gsub(/%/, "", $3)
  cov = $3 + 0
  if (cov < thr) {
    printf "FAIL: coverage %.1f%% < %d%%\n", cov, thr
    exit 1
  }
  printf "Coverage: %.1f%%\n", cov
  exit 0
}
END {
  if (NR == 0) {
    print "ERROR: no coverage data."
    exit 2
  }
}'

step "Running golangci-lint"
GOGC=off golangci-lint config verify
GOGC=off golangci-lint run --timeout 5m

step "Running govulncheck"
govulncheck -test ./...

step "Quality gate complete"
