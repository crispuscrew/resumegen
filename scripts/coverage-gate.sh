#!/usr/bin/env sh
# coverage-gate.sh — fail if coverage on domain+usecase falls below GATE.
# Measured packages match ROADMAP.md: ./internal/domain/... and ./internal/usecase/...
set -eu

GATE="${GATE:-50}"
PROFILE="${PROFILE:-coverage.out}"

go test -coverprofile="$PROFILE" ./internal/domain/... ./internal/usecase/...

PCT=$(go tool cover -func="$PROFILE" | awk '/^total:/ { sub(/%$/, "", $NF); print $NF }')

awk -v pct="$PCT" -v gate="$GATE" 'BEGIN {
    if (pct + 0 < gate + 0) { printf "FAIL: %.1f%% < %s%%\n", pct, gate; exit 1 }
    else                    { printf "OK:   %.1f%% >= %s%%\n", pct, gate; exit 0 }
}'
