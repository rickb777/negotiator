#!/bin/bash -e
cd "$(dirname $0)"
PATH=$HOME/go/bin:$PATH
unset GOPATH
export GO111MODULE=on
export GOARCH=${1}

function v
{
  echo
  echo $@
  $@
}

if ! type -p shadow; then
  v go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow
fi

if ! type -p goreturns; then
  v go install github.com/sqs/goreturns
fi

v go test -v ./...

v goreturns -l -w *.go */*.go

v go vet ./...

v shadow ./...
