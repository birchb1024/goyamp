#!/bin/bash
set -e
set -u
set -x
version=$(git describe --abbrev)

function buildDocs {
	cp README.asciidoc /tmp/README.asciidoc
	sed -i "s;@@@VERSION@@@;${version};" /tmp/README.asciidoc
	sed -i "s;@@@DATE@@@;$(date +%d.%m.%Y);"  /tmp/README.asciidoc
	asciidoc /tmp/README.asciidoc && mv /tmp/README.html doc
}

args=${1:-bin}
if [[ "${args}" == "doc" ]]
then
    buildDocs
	exit
fi

#go build -ldflags "-X github.com/birchb1024/goyamp.Version=${version}"
go build -o goyamp -ldflags "-X github.com/birchb1024/goyamp.Version=${version}" cmd/main.go
strip ./goyamp
(cd test; go test -args $*)
GOOS=windows GOARCH=amd64 go build -o goyamp.exe -ldflags "-X github.com/birchb1024/goyamp.Version=${version}" cmd/main.go
GOOS=darwin GOARCH=amd64 go build -o goyamp_mac -ldflags "-X github.com/birchb1024/goyamp.Version=${version}" cmd/main.go

if [[ "${args}" == "package" ]]
then
    mkdir -p pkg
    buildDocs
    tar zcvf pkg/goyamp-${version}.tgz ./goyamp ./goyamp.exe doc/README.html examples
	exit
fi
