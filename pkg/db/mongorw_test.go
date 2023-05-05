package db

import (
	"context"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/mapper"
)

func init() {
	logger.InitLogger("nativerw", "info")
}

func generateResource() *mapper.Resource {
	return &mapper.Resource{
		UUID:            uuid.NewUUID().String(),
		Content:         map[string]interface{}{"randomness": uuid.NewUUID().String()},
		ContentType:     "application/json",
		SchemaVersion:   "14",
		ContentRevision: int64(123),
	}
}

func TestReadWriteDelete(t *testing.T) {
	connection, err := startMongo(t)

	assert.NoError(t, err)

	expectedResource := generateResource()
	err = connection.Write("universal-content", expectedResource)
	assert.NoError(t, err)

	res, found, err := connection.Read("universal-content", expectedResource.UUID)
	assert.True(t, found)
	assert.NoError(t, err)
	assert.Equal(t, expectedResource.ContentType, res.ContentType)
	assert.Equal(t, expectedResource.UUID, res.UUID)
	assert.Equal(t, expectedResource.Content, res.Content)
	assert.Equal(t, expectedResource.SchemaVersion, res.SchemaVersion)
	assert.Equal(t, expectedResource.ContentRevision, res.ContentRevision)

	err = connection.Delete("universal-content", expectedResource.UUID, expectedResource.ContentRevision)
	assert.NoError(t, err)

	_, found, err = connection.Read("universal-content", expectedResource.UUID)

	assert.False(t, found)
	assert.NoError(t, err)
}

func TestGetSupportedCollections(t *testing.T) {
	connection, err := startMongo(t)
	assert.NoError(t, err)

	expected := map[string]bool{"universal-content": true} // this is set in mongo_test.go
	actual := connection.GetSupportedCollections()
	assert.Equal(t, expected, actual)
}

func TestEnsureIndexes(t *testing.T) {
	connection, err := startMongo(t)
	assert.NoError(t, err)

	connection.EnsureIndex()
	indexes := connection.(*MongoConnection).client.Database("native-store").Collection("universal-content").Indexes()

	assert.NoError(t, err)
	count := 0

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	cursor, err := indexes.List(ctx)
	assert.NoError(t, err)

	for cursor.Next(ctx) {
		var index primitive.M
		err = cursor.Decode(&index)
		assert.NoError(t, err)

		if index["name"] == "uuid-revision-index" {
			assert.True(t, index["unique"].(bool))
			assert.Equal(t, primitive.M{"uuid": int32(1), "content-revision": int32(1)}, index["key"])
			count = count + 1
		}
	}

	assert.Equal(t, 1, count)
}

func TestReadIDs(t *testing.T) {
	connection, err := startMongo(t)
	assert.NoError(t, err)

	expectedResource := generateResource()

	err = connection.Write("universal-content", expectedResource)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ids, err := connection.ReadIDs(ctx, "universal-content")

	assert.NoError(t, err)
	found := false

	for uuid := range ids {
		if uuid == expectedResource.UUID {
			found = true
		}
	}

	assert.True(t, found)
}

func TestReadMoreThanOneBatch(t *testing.T) {
	connection, err := startMongo(t)

	assert.NoError(t, err)

	for range make([]struct{}, 64) {
		expectedResource := generateResource()

		err = connection.Write("universal-content", expectedResource)
		assert.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ids, err := connection.ReadIDs(ctx, "universal-content")

	assert.NoError(t, err)
	count := 0

	for range ids {
		count++
	}

	assert.True(t, count >= 64)
}

func TestCancelReadIDs(t *testing.T) {
	connection, err := startMongo(t)

	assert.NoError(t, err)

	for range make([]struct{}, 64) {
		expectedResource := generateResource()

		err = connection.Write("universal-content", expectedResource)
		assert.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // just in case

	ids, err := connection.ReadIDs(ctx, "universal-content")

	assert.NoError(t, err)

	time.Sleep(1 * time.Second) // allow the channel to fill

	uuid := <-ids
	assert.NotEqual(t, "", uuid) // prove one uuid has been retrieved, but let the channel fill and block after
	cancel()                     // cancel the request

	count := 0
	for {
		uuid, ok := <-ids

		if !ok {
			assert.Equal(t, 8, count) // count should be 8, which is the size of the channel
			break
		}

		assert.NotEqual(t, "", uuid) // all uuids should be non-zero
		count++
	}
}
