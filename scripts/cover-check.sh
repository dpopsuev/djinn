#!/usr/bin/env bash
# cover-check.sh — fail if any package coverage drops below baseline.
# Usage: ./scripts/cover-check.sh [coverage-baseline.json]
set -euo pipefail

BASELINE="${1:-coverage-baseline.json}"
FAILED=0

if [ ! -f "$BASELINE" ]; then
    echo "WARN: no baseline found at $BASELINE — skipping check"
    exit 0
fi

echo "Coverage check against baseline ($BASELINE):"
echo "================================================"

go test ./... -race -count=1 -coverprofile=/dev/null -covermode=atomic 2>&1 | grep 'coverage:' | while read -r line; do
    pkg=$(echo "$line" | sed 's/.*github.com\/dpopsuev\/djinn\///' | awk '{print $1}')
    pct=$(echo "$line" | grep -o '[0-9.]*%' | head -1 | tr -d '%')

    if [ -z "$pct" ]; then continue; fi

    # Look up baseline
    baseline=$(python3 -c "
import json, sys
with open('$BASELINE') as f:
    d = json.load(f)
pkg = '$pkg'
print(d.get('packages', {}).get(pkg, -1))
" 2>/dev/null || echo "-1")

    if [ "$baseline" = "-1" ]; then
        printf "  %-30s %6s%% (new — no baseline)\n" "$pkg" "$pct"
        continue
    fi

    diff=$(python3 -c "print(round(float('$pct') - float('$baseline'), 1))")

    if python3 -c "exit(0 if float('$pct') >= float('$baseline') else 1)"; then
        printf "  %-30s %6s%% (baseline: %s%%, %+.1f) ✓\n" "$pkg" "$pct" "$baseline" "$diff"
    else
        printf "  %-30s %6s%% (baseline: %s%%, %+.1f) ✗ REGRESSION\n" "$pkg" "$pct" "$baseline" "$diff"
        FAILED=1
    fi
done

echo "================================================"
if [ "$FAILED" -eq 1 ]; then
    echo "FAIL: coverage regression detected"
    exit 1
fi
echo "PASS: all packages at or above baseline"
