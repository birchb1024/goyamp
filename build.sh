#!/bin/bash
set -euo pipefail
#set -x
script_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
version=$(git describe --abbrev)

export LUA_PATH='./?.lua;./?.lc;'"$script_dir"'/lib/?.lua;'

function setupWorkspaceForBuild {
 echo Change directory to linuxinstall/
 echo then: ansible-playbook -i localhost, --tags development setup-workstation.yaml
}

function buildDocs {
	cp README.asciidoc /tmp/README.asciidoc
	sed -i "s;@@@VERSION@@@;${version};" /tmp/README.asciidoc
	sed -i "s;@@@DATE@@@;$(date +%d.%m.%Y);"  /tmp/README.asciidoc
	asciidoc /tmp/README.asciidoc && mv /tmp/README.html doc
}

args=${1:-bin}

if [[ "${args}" == "depends" ]]
then
  setupWorkspaceForBuild
	exit
fi


if [[ "${args}" == "doc" ]]
then
    buildDocs
	exit
fi

go build -o goyamp -ldflags "-X main.Version=${version}"

(
	cd test
	go test -coverprofile=../coverage.out -coverpkg ./../internal ./ -args "$*"
)

if [[ "${args}" == "coverage" ]]
then
	go tool cover -html=coverage.out
fi

GOOS=windows GOARCH=amd64         go build -o goyamp.exe -ldflags "-X github.com/birchb1024/goyamp.Version=${version}"
GOOS=darwin  GOARCH=amd64         go build -o goyamp_mac -ldflags "-X github.com/birchb1024/goyamp.Version=${version}"
GOOS=linux   GOARCH=arm   GOARM=7 go build -o goyamp_arm7 -ldflags "-X github.com/birchb1024/goyamp.Version=${version}"
GOOS=linux   GOARCH=arm   GOARM=6 go build -o goyamp_arm6 -ldflags "-X github.com/birchb1024/goyamp.Version=${version}"


if [[ "${args}" == "package" ]]
then
    mkdir -p pkg
    buildDocs
    tar zcf pkg/goyamp-"${version}".tgz ./goyamp ./goyamp_mac ./goyamp.exe ./goyamp_arm6 ./goyamp_arm7 doc/README.html examples lib
	  exit
fi

