#!/bin/sh
# Writes the fixtures limu's tests compare with, from Pele's sources.
#
#   ./scripts/oracle.sh [--full] [path to pele] [suite…]
#
# The path defaults to $PELE, then ../pele beside this repository; with suites named, it must
# be given. Node 24 or later runs Pele's TypeScript as it is; nothing needs to be installed.
# The suites are the modules in internal/oracle/suites; with none named, all of them run. A
# fixture whose JSON has not changed is left as it is.
#
# A fixture of drawings, render.json, holds the lines of its first cases and a digest of the
# lines of each of the rest. With --full, the lines of every case are also written beside it,
# to render.full.json.gz, which git ignores. That file is only for finding which line of a
# case differs from a digest: the tracked fixture is what the tests compare with, and a test
# that finds a difference says when the file is missing or stale.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
full=
if [ "${1:-}" = --full ]; then
	full=--full
	shift
fi
pele=${1:-${PELE:-$root/../pele}}
if [ ! -f "$pele/package.json" ] || [ ! -f "$pele/src/index.ts" ]; then
	echo "oracle.sh: $pele is not a checkout of Pele" >&2
	echo "usage: ./scripts/oracle.sh [--full] [path to pele] [suite…]" >&2
	exit 2
fi
pele=$(cd "$pele" && pwd)
cd "$root/internal/oracle"
if [ $# -gt 0 ]; then shift; fi
exec node --experimental-transform-types --no-warnings --import ./register.mjs run.mjs $full "$pele" "$root" "$@"
