#!/bin/sh
# Writes the fixtures limu's tests compare with, from Pele's sources.
#
#   ./scripts/oracle.sh [path to pele]
#
# The path defaults to $PELE, then ../pele beside this repository. Node 24 or later runs
# Pele's TypeScript as it is; nothing needs to be installed.
set -eu
root=$(cd "$(dirname "$0")/.." && pwd)
pele=${1:-${PELE:-$root/../pele}}
pele=$(cd "$pele" && pwd)
cd "$root/internal/oracle"
exec node --experimental-transform-types --no-warnings --import ./register.mjs run.mjs "$pele" "$root"
