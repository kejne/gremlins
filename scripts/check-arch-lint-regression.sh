#!/bin/sh
set -eu

project_root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
temp_root=$(mktemp -d "${TMPDIR:-/tmp}/gremlins-arch-lint.XXXXXX")
trap 'rm -rf "$temp_root"' EXIT HUP INT TERM

# Run both checks on an isolated copy so the regression fixture never enters the checkout.
mkdir -p "$temp_root/project"
tar -C "$project_root" --exclude=.git --exclude=dist -cf - . | tar -C "$temp_root/project" -xf -

if (cd "$temp_root/project" && go-arch-lint check --output-color=false) >"$temp_root/baseline.log" 2>&1; then
    :
else
    status=$?
    echo "go-arch-lint rejected the unmodified source-tree copy (exit $status):" >&2
    cat "$temp_root/baseline.log" >&2
    exit 1
fi

cat >"$temp_root/project/internal/log/architecture_regression.go" <<'EOF'
package log

import _ "github.com/kejne/gremlins/internal/configuration"
EOF

if (cd "$temp_root/project" && go-arch-lint check --output-color=false) >"$temp_root/forbidden.log" 2>&1; then
    echo "go-arch-lint accepted the forbidden logging -> configuration dependency" >&2
    cat "$temp_root/forbidden.log" >&2
    exit 1
else
    status=$?
fi

if ! grep -Eiq 'logging.*configuration|configuration.*logging' "$temp_root/forbidden.log"; then
    echo "go-arch-lint failed for an unexpected reason (exit $status); expected logging -> configuration:" >&2
    cat "$temp_root/forbidden.log" >&2
    exit 1
fi

echo "Architecture regression check passed: an injected logging -> configuration dependency is rejected."
