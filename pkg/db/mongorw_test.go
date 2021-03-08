package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/mapper"
)

func init() {
	logger.InitLogger("nativerw", "info")
}

func generateResource() *mapper.Resource {
	return &mapper.Resource{
		UUID:          uuid.NewUUID().String(),
		Content:       map[string]interface{}{"randomness": uuid.NewUUID().String()},
		ContentType:   "application/json",
		SchemaVersion: "14",
	}
}

func TestReadWriteDelete(t *testing.T) {
	mongo := startMongo(t)
	connection, err := mongo.Open()

	assert.NoError(t, err)
	defer connection.Close()
	expectedResource := generateResource()
	err = connection.Write("methode", expectedResource)
	assert.NoError(t, err)

	res, found, err := connection.Read("methode", expectedResource.UUID)
	assert.True(t, found)
	assert.NoError(t, err)
	assert.Equal(t, expectedResource.ContentType, res.ContentType)
	assert.Equal(t, expectedResource.UUID, res.UUID)
	assert.Equal(t, expectedResource.Content, res.Content)
	assert.Equal(t, expectedResource.SchemaVersion, res.SchemaVersion)

	err = connection.Delete("methode", expectedResource.UUID)
	assert.NoError(t, err)

	_, found, err = connection.Read("methode", expectedResource.UUID)

	assert.False(t, found)
	assert.NoError(t, err)
}

func TestGetSupportedCollections(t *testing.T) {
	mongo := startMongo(t)
	connection, err := mongo.Open()
	assert.NoError(t, err)

	defer connection.Close()

	expected := map[string]bool{"methode": true} // this is set in mongo_test.go
	actual := connection.GetSupportedCollections()
	assert.Equal(t, expected, actual)
}

func TestEnsureIndexes(t *testing.T) {
	mongo := startMongo(t)
	connection, err := mongo.Open()
	assert.NoError(t, err)

	defer connection.Close()

	connection.EnsureIndex()
	indexes, err := connection.(*mongoConnection).session.DB("native-store").C("methode").Indexes()

	assert.NoError(t, err)
	count := 0
	for _, index := range indexes {
		if index.Name == "uuid-revision-index" {
			assert.True(t, index.Background)
			assert.True(t, index.Unique)
			assert.Equal(t, []string{"uuid", "content-revision"}, index.Key)
			count = count + 1
		}
	}

	assert.Equal(t, 1, count)
}

func TestAwaitConnectionFailsIfNotOpened(t *testing.T) {
	mongo := startMongo(t)

	_, err := mongo.Await()
	assert.Error(t, err)
}

func TestAwaitConnectionBlocks(t *testing.T) {
	mongo := startMongo(t)

	mongo.(*mongoDB).connection = NewOptional(func() (interface{}, error) {
		time.Sleep(50 * time.Millisecond)
		return &mongoConnection{dbName: "test-collection"}, nil
	})

	connection, err := mongo.Await()
	assert.NoError(t, err)
	assert.Equal(t, "test-collection", connection.(*mongoConnection).dbName)
}

func TestAwaitConnectionFails(t *testing.T) {
	mongo := startMongo(t)

	mongo.(*mongoDB).connection = NewOptional(func() (interface{}, error) {
		time.Sleep(50 * time.Millisecond)
		return nil, errors.New("went spectacularly wrong")
	})

	connection, err := mongo.Await()
	assert.Error(t, err)
	assert.Nil(t, connection)
}

func TestAwaitConnectionReturnsIfInitialised(t *testing.T) {
	mongo := startMongo(t).(*mongoDB)
	ch := make(chan bool, 1)
	defer func() {
		ch <- true
		close(ch)
	}()

	mongo.connection = NewOptional(func() (interface{}, error) {
		<-ch
		return &mongoConnection{dbName: "psych!-changed-it"}, nil
	})

	mongo.connection.val = &mongoConnection{dbName: "find-me-pls"}

	connection, err := mongo.Await()
	assert.NoError(t, err)
	assert.Equal(t, "find-me-pls", connection.(*mongoConnection).dbName)
}

func TestOpenWillReturnConnectionIfAlreadyInitialised(t *testing.T) {
	mongo := startMongo(t).(*mongoDB)

	mongo.connection = NewOptional(func() (interface{}, error) {
		return &mongoConnection{dbName: "faked"}, nil
	})

	_, err := mongo.connection.Block()
	assert.NoError(t, err)

	connection, err := mongo.Open()
	assert.NoError(t, err)
	assert.Equal(t, "faked", connection.(*mongoConnection).dbName)
}

func TestOpenFailsIfInitialisationFailed(t *testing.T) {
	mongo := startMongo(t).(*mongoDB)

	mongo.connection = NewOptional(func() (interface{}, error) {
		return nil, errors.New("i failed")
	})

	_, _ = mongo.connection.Block()

	connection, err := mongo.Open()
	assert.Error(t, err)
	assert.Nil(t, connection)
}

func TestReadIDs(t *testing.T) {
	mongo := startMongo(t).(*mongoDB)
	connection, err := mongo.Open()

	assert.NoError(t, err)

	defer connection.Close()

	expectedResource := generateResource()

	err = connection.Write("methode", expectedResource)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ids, err := connection.ReadIDs(ctx, "methode")

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
	mongo := startMongo(t).(*mongoDB)
	connection, err := mongo.Open()

	assert.NoError(t, err)

	defer connection.Close()

	for range make([]struct{}, 64) {
		expectedResource := generateResource()

		err = connection.Write("methode", expectedResource)
		assert.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ids, err := connection.ReadIDs(ctx, "methode")

	assert.NoError(t, err)
	count := 0

	for range ids {
		count++
	}

	assert.True(t, count >= 64)
}

func TestCancelReadIDs(t *testing.T) {
	mongo := startMongo(t).(*mongoDB)
	connection, err := mongo.Open()

	assert.NoError(t, err)

	defer connection.Close()

	for range make([]struct{}, 64) {
		expectedResource := generateResource()

		err = connection.Write("methode", expectedResource)
		assert.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // just in case

	ids, err := connection.ReadIDs(ctx, "methode")

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

func TestCheckMongoUrlsValidUrls(t *testing.T) {
	err := CheckMongoUrls("host:port,host2:port2", 2)

	assert.Nil(t, err)
}

func TestCheckMongoUrlsFewerUrlsThanExpected(t *testing.T) {
	err := CheckMongoUrls("host:port,host2:port2", 3)

	assert.NotNil(t, err)
}

func TestCheckMongoUrlsMissingPort(t *testing.T) {
	err := CheckMongoUrls("host:port,host2:", 2)

	assert.NotNil(t, err)
}

func TestCheckMongoUrlsMissingHost(t *testing.T) {
	err := CheckMongoUrls("host:port,:port", 2)

	assert.NotNil(t, err)
}

func TestCheckMongoUrlsInvalidSeparator(t *testing.T) {
	err := CheckMongoUrls("host:port;host1:port1", 2)

	assert.NotNil(t, err)
}
