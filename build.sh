#!/bin/sh

ORG_PATH="github.com/Financial-Times"
REPO_PATH="${ORG_PATH}/nativerw"
export GOPATH=/gopath
mkdir -p $GOPATH/src/${ORG_PATH}
ln -s ${PWD} $GOPATH/src/${REPO_PATH}
cd $GOPATH/src/${REPO_PATH}
git checkout $TAG || exit 1
go get || exit 1
go test || exit 1
CGO_ENABLED=0 go build -a -installsuffix cgo -ldflags "-s" -o /out/nativerw ${REPO_PATH} || exit 1

