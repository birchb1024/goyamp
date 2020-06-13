#!/bin/bash
set -e
set -u
set -x
script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
version=$(git describe --abbrev)

export LUA_PATH="$script_dir"/lib/?.lua;

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
    tar zcvf pkg/goyamp-${version}.tgz ./goyamp init.lua ./goyamp.exe doc/README.html examples lib
	exit
fi
