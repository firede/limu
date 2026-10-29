#!/bin/sh
# Runs every check of the repository: go vet, gofmt and the tests, which compare what limu
# does with fixtures written from Pele and with golden files, in limu's module and in the
# limumygo module, which ./... does not reach, with its playground's shell for browsers,
# which only a build for js/wasm compiles. Works offline once limumygo's modules are
# downloaded, from any directory; Node and Pele are needed only to write the fixtures anew
# (scripts/oracle.sh).
set -eu
cd "$(dirname "$0")/.."
go vet ./...
# The files of the repository, those not yet added to it among them, but not those of
# worktrees kept inside it, which .gitignore leaves out, nor those deleted and not yet
# removed from the index.
unformatted=$(git ls-files -z --cached --others --exclude-standard '*.go' |
	xargs -0 sh -c 'for f; do if [ -e "$f" ]; then printf "%s\0" "$f"; fi; done' sh |
	xargs -0 gofmt -l)
if [ -n "$unformatted" ]; then
	echo "not formatted with gofmt:"
	echo "$unformatted"
	exit 1
fi
go test -count=1 ./...
(cd limumygo && go vet ./... && GOOS=js GOARCH=wasm go vet ./examples/playground && go test -count=1 ./...)
echo PASSED
