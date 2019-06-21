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

if [[ "${args}" == "package" ]]
then
    mkdir -p pkg
	asciidoc README.asciidoc && mv README.html doc
    tar zcvf pkg/goyamp-${version}.tgz ./goyamp doc/README.html
	exit
fi
