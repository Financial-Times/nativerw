# Native Store Reader Writer (nativerw)
[![Coverage Status](https://coveralls.io/repos/github/Financial-Times/nativerw/badge.svg?branch=master)](https://coveralls.io/github/Financial-Times/nativerw?branch=master)

__Writes any raw content/data from native CMS in mongoDB without transformation.
The same data can then be read from here just like from the original CMS.__

## Installation

You need [Go to be installed](https://golang.org/doc/install). Please read about Go and about [How to Write Go Code](https://golang.org/doc/code.html) before jumping right in. For example, you will need Git, Mercurial, Bazaar installed and working, so that Go can use them to retrieve dependencies. For this additionally you will also need a computer etc. Hope this helps.

For the first time: `go get github.com/Financial-Times/nativerw`.

`go install github.com/Financial-Times/nativerw`

### Building docker

```bash
CGO_ENABLED=0 go build -mod=readonly -a -installsuffix cgo -o nativerw .
docker build -t coco/nativerw .
```

## Running
The following params can be injected in the nativerw app on startup through environment variables:
 - `DB_CLUSTER_ADDRESS` Database cluster address.
 - `DB_USERNAME` Username to connect to database.
 - `DB_PASSWORD` Password to connect to database.
 - `CONFIG` Config file in json format. If not set, the default `config.json` will be used.
 - `TIDS_TO_SKIP` Regular expression defining transaction-id's to be skipped from storing in nativerw
 - `DISABLE-PURGE` Disables the `purge` endpoint

To run locally against `dev` native store:
1. Get the url and credentials for the instance in LastPass
   
    ```bash
   export DB_CLUSTER_ADDRESS="url.to.document-db.cluster:27017"
   export DB_USERNAME="username"
   export DB_PASSWORD="password"
    ````

2. Run with the required ENV vars
    
    ```bash
   TIDS_TO_SKIP=none go run cmd/nativerw/main.go
   ```

## API

The nativerw supports the following endpoints:

* GET `/{collection}/{uuid}` retrieves the latest revision of native document, and returns it in either json or binary (depending on how it is saved).
* GET `/{collection}/{uuid}/revisions` retrieves a list with all the revisions for a specific document
* GET `/{collection}/{uuid}/{revision}` retrieves a specific revision of a document
* POST `/{collection}/{uuid}` inserts a new native document for the given uuid/revision. If the specified revision already exists then no changes are written in the database and 200 OK is returned. Since the MongoDB is historized based on the `revision` field, the updates are treated as inserts in the database.
* PATCH `/{collection}/{uuid}` updates specific fields for the given uuid/revision. If no revision is provided a new one is generated based on the current date/time
* DELETE `/{collection}/{uuid}` marks a document as deleted in the store by inserting new revision in the MongoDB
* DELETE `/{collection}/purge/{uuid}/{revision}` physically deletes a document revision from the store
* GET `/{collection}/__ids` returns all uuids for the given collection on a **best efforts' basis**. If the collection is very large, the endpoint is likely to time out (timeout duration is hardcoded to 10s) before all uuids have been returned. This will be indistinguishable from a request which sends back the complete set of uuids, however, if there are less than ~10,000 uuids returned, you can be fairly confident you have the entire set.
* GET `/__gtg` the good to go endpoint.
* GET `/__health` the health endpoint.

### Logging

* The application uses [go-logger](https://github.com/Financial-Times/go-logger ); the log file is initialised in [app.go](cmd/nativerw/main.go).
