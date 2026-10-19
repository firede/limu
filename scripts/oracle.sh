#!/bin/sh
# Writes the fixtures limu's tests compare with, from Pele's sources.
#
#   ./scripts/oracle.sh [path to pele] [suite…]
#
# The path defaults to $PELE, then ../pele beside this repository; with suites named, it must
# be given. Node 24 or later runs Pele's TypeScript as it is; nothing needs to be installed.
# The suites are the modules in internal/oracle/suites; with none named, all of them run. A
# fixture whose JSON has not changed is left as it is.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
pele=${1:-${PELE:-$root/../pele}}
if [ ! -f "$pele/package.json" ] || [ ! -f "$pele/src/index.ts" ]; then
	echo "oracle.sh: $pele is not a checkout of Pele" >&2
	echo "usage: ./scripts/oracle.sh [path to pele] [suite…]" >&2
	exit 2
fi
pele=$(cd "$pele" && pwd)
cd "$root/internal/oracle"
if [ $# -gt 0 ]; then shift; fi
exec node --experimental-transform-types --no-warnings --import ./register.mjs run.mjs "$pele" "$root" "$@"
