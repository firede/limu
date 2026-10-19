#!/bin/sh
# Runs every check of the repository: go vet, gofmt and the tests, which compare what limu
# does with fixtures written from Pele and with golden files. Works offline, from any
# directory; Node and Pele are needed only to write the fixtures anew (scripts/oracle.sh).
set -eu
cd "$(dirname "$0")/.."
go vet ./...
# The files of the repository, not those of worktrees kept inside it.
unformatted=$(git ls-files -z '*.go' | xargs -0 gofmt -l)
if [ -n "$unformatted" ]; then
	echo "not formatted with gofmt:"
	echo "$unformatted"
	exit 1
fi
go test -count=1 ./...
echo PASSED
