# Native Store Reader Writer (nativerw)
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/nativerw/badge.svg?branch=master)](https://coveralls.io/github/Financial-Times/nativerw?branch=master)

__Writes any raw content/data from native CMS in mongoDB without transformation.
The same data can then be read from here just like from the original CMS.__

## Installation

You need [Go to be installed](https://golang.org/doc/install). Please read about Go and about [How to Write Go Code](https://golang.org/doc/code.html) before jumping right in. For example you will need Git, Mercurial, Bazaar installed and working, so that Go can use them to retrieve dependencies. For this additionally you will also need a computer etc. Hope this helps.

For the first time: `go get github.com/Financial-Times/nativerw` or update: `go get -u github.com/Financial-Times/nativerw`

`go install github.com/Financial-Times/nativerw`

### Building docker

```bash
CGO_ENABLED=0 go build -a -installsuffix cgo -o nativerw .
docker build -t coco/nativerw .
```

## Running

`$GOPATH/bin/nativerw $GOPATH/src/github.com/Financial-Times/nativerw/config.json`

You can override the mongos with -mongos flag, e.g.

`$GOPATH/bin/nativerw -mongos=mongo1:port,mongo2:port $GOPATH/src/github.com/Financial-Times/nativerw/config.json`

## API

The nativerw supports the following endpoints:

* GET `/{collection}/{uuid}` retrieves the native document, and returns it in either json or binary (depending on how it is saved).
* PUT `/{collection}/{uuid}` upserts a new native document for the given uuid.
* GET `/{collection}/__ids` returns all uuids for the given collection on a **best efforts basis**. If the collection is very large, the endpoint is likely to time out (timeout duration is hardcoded to 10s) before all uuids have been returned. This will be indistinguishable from a request which sends back the complete set of uuids, however, if there are less than ~10,000 uuids returned, you can be fairly confident you have the entire set.
* GET `/__gtg` the good to go endpoint.
* GET `/__health` the health endpoint.
