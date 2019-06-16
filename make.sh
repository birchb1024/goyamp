#!/bin/bash
set -e
set -u
set -x
version=$(git describe --abbrev)

args=${1:-bin}
if [[ "${args}" == "doc" ]]
then
	asciidoc README.asciidoc && mv README.html doc
	exit
fi

go build -ldflags "-X github.com/birchb1024/goyamp.Version=${version}"
go build -o goyamp -ldflags "-X github.com/birchb1024/goyamp.Version=${version}" cmd/main.go
(cd test; go test -args $*)
