#!/bin/sh
set -eu

project_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
temp_root=$(mktemp -d "${TMPDIR:-/tmp}/gremlins-testpackage.XXXXXX")
trap 'rm -rf "$temp_root"' EXIT HUP INT TERM
# Keep disposable linter runs isolated from hk's parallel repository lint step.
export GOLANGCI_LINT_CACHE="$temp_root/golangci-cache"

fixture="$temp_root/project"
mkdir -p "$fixture"
tar -C "$project_root" --exclude=.git --exclude=dist -cf - . | tar -C "$fixture" -xf -
mkdir -p "$fixture/internal/fixture"
cat >"$fixture/internal/fixture/fixture.go" <<'EOF'
// Package fixture provides a testpackage lint fixture.
package fixture

// Value returns a stable fixture value.
func Value() int { return 1 }
EOF

# An internal_test.go filename must not receive testpackage's implicit exemption.
cat >"$fixture/internal/fixture/internal_test.go" <<'EOF'
package fixture

import "testing"

func TestValue(t *testing.T) {
	if Value() != 1 {
		t.Fatal("unexpected value")
	}
}
EOF
if (cd "$fixture" && golangci-lint run ./internal/fixture) >"$temp_root/same-package.log" 2>&1; then
    echo "testpackage accepted a same-package internal_test.go fixture" >&2
    exit 1
fi
if ! grep -q 'testpackage' "$temp_root/same-package.log"; then
    echo "same-package fixture failed for an unexpected reason:" >&2
    cat "$temp_root/same-package.log" >&2
    exit 1
fi

# The linter's default exemption for package main is also disabled.
sed 's/^package fixture$/package main/' "$fixture/internal/fixture/fixture.go" >"$temp_root/main.go"
mv "$temp_root/main.go" "$fixture/internal/fixture/fixture.go"
sed 's/^package fixture$/package main/' "$fixture/internal/fixture/internal_test.go" >"$temp_root/main_test.go"
mv "$temp_root/main_test.go" "$fixture/internal/fixture/internal_test.go"
if (cd "$fixture" && golangci-lint run ./internal/fixture) >"$temp_root/main-package.log" 2>&1; then
    echo "testpackage accepted a same-package test in package main" >&2
    exit 1
fi
if ! grep -q 'testpackage' "$temp_root/main-package.log"; then
    echo "main-package fixture failed for an unexpected reason:" >&2
    cat "$temp_root/main-package.log" >&2
    exit 1
fi

# External-package tests are accepted by the same pinned lint configuration.
sed 's/^package main$/package fixture/' "$fixture/internal/fixture/fixture.go" >"$temp_root/fixture.go"
mv "$temp_root/fixture.go" "$fixture/internal/fixture/fixture.go"
cat >"$fixture/internal/fixture/external_test.go" <<'EOF'
package fixture_test

import (
	"testing"

	"github.com/kejne/gremlins/internal/fixture"
)

func TestValue(t *testing.T) {
	if fixture.Value() != 1 {
		t.Fatal("unexpected value")
	}
}
EOF
rm "$fixture/internal/fixture/internal_test.go"
(cd "$fixture" && golangci-lint run ./internal/fixture)

# A file's testpackage allowance must not suppress diagnostics from another linter.
sed '/^package exclusion$/a import "fmt"' "$fixture/internal/exclusion/rules_test.go" >"$temp_root/rules_test.go"
mv "$temp_root/rules_test.go" "$fixture/internal/exclusion/rules_test.go"
cat >>"$fixture/internal/exclusion/rules_test.go" <<'EOF'

func lintDiagnostic() {
	fmt.Printf("%d", "not a number")
}
EOF
if (cd "$fixture" && golangci-lint run --enable-only govet ./internal/exclusion) >"$temp_root/unrelated-linter.log" 2>&1; then
    echo "testpackage allowance suppressed or missed the injected govet diagnostic" >&2
    exit 1
fi
if ! grep -q 'Printf' "$temp_root/unrelated-linter.log"; then
    echo "unrelated-linter fixture failed for an unexpected reason:" >&2
    cat "$temp_root/unrelated-linter.log" >&2
    exit 1
fi

# Removing a single baseline entry restores enforcement for that exact file.
sed '/rules_test/,+1d' "$fixture/.golangci.yml" >"$fixture/.golangci-without-rules-allowance.yml"
if (cd "$fixture" && golangci-lint run --config .golangci-without-rules-allowance.yml ./internal/exclusion) >"$temp_root/removed-allowance.log" 2>&1; then
    echo "testpackage accepted internal/exclusion/rules_test.go after its allowance was removed" >&2
    exit 1
fi
if ! grep -q 'testpackage' "$temp_root/removed-allowance.log"; then
    echo "removed-allowance fixture failed for an unexpected reason:" >&2
    cat "$temp_root/removed-allowance.log" >&2
    exit 1
fi

echo "testpackage regression demonstrations passed: implicit exemption disabled, external package accepted, unrelated diagnostics visible under an allowance, and removing one exact allowance restores enforcement."
