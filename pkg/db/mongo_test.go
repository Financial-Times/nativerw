package db

import (
	"os"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func startMongo(t *testing.T) (Connection, error) {
	if testing.Short() {
		t.Skip("Mongo integration for long tests only.")
	}

	mongoURL := os.Getenv("MONGO_TEST_URL")
	if strings.TrimSpace(mongoURL) == "" {
		t.Fatal("Please set the environment variable MONGO_TEST_URL to run mongo integration tests (e.g. export MONGO_TEST_URL=mongodb://localhost:27017). Alternatively, run `go test -short` to skip them.")
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURL))
	if err != nil {
		return nil, err
	}

	return &mongoConnection{
		dbName: "native-store",
		client: client,
		collections: map[string]bool{
			"universal-content": true,
		},
	}, nil
}
