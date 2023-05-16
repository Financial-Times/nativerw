package db

import (
	"context"
	"time"

	"github.com/pborman/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"gopkg.in/mgo.v2/bson"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/mapper"
	"github.com/Financial-Times/upp-go-sdk/pkg/documentdb"
)

const (
	uuidName            = "uuid"
	contentRevisionName = "content-revision"

	mongoConnectionTimeout       = time.Second * 30
	mongoIndexCreationTimeout    = time.Second * 15
	mongoDefaultOperationTimeout = time.Second * 5
)

type MongoConnection struct {
	dbName      string
	client      *mongo.Client
	collections map[string]bool
}

// Connection contains all mongo request logic, including reads, writes and deletes.
type Connection interface {
	EnsureIndex()
	GetSupportedCollections() map[string]bool
	Delete(collection string, uuidString string, revision int64) error
	Write(collection string, resource *mapper.Resource) error
	Read(collection string, uuidString string) (res *mapper.Resource, found bool, err error)
	ReadSingleRevision(collection string, uuidString string, revision int64) (res *mapper.Resource, err error)
	ReadIDs(ctx context.Context, collection string) (chan string, error)
	ReadRevisions(collection string, uuidString string) (res []int64, err error)
	Count(collection string, uuidString string, contentRevision int64) (count int64, err error)
	Ping() error
}

// NewDBConnection dials the mongo cluster, and returns a new handler DB instance
func NewDBConnection(docDBConf documentdb.ConnectionParams, collections []string) (*MongoConnection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), mongoConnectionTimeout)
	defer cancel()
	client, err := documentdb.NewClient(ctx, docDBConf)
	if err != nil {
		return nil, err
	}

	colls := createMapWithAllowedCollections(collections)
	return &MongoConnection{docDBConf.Database, client, colls}, nil
}

func (ma *MongoConnection) GetSupportedCollections() map[string]bool {
	return ma.collections
}

func createMapWithAllowedCollections(collections []string) map[string]bool {
	var collectionMap = make(map[string]bool)
	for _, coll := range collections {
		collectionMap[coll] = true
	}
	return collectionMap
}

func (ma *MongoConnection) EnsureIndex() {
	index := mongo.IndexModel{
		Keys: bsonx.Doc{
			{Key: "uuid", Value: bsonx.Int32(1)},
			{Key: "content-revision", Value: bsonx.Int32(1)},
		},
		Options: options.Index().
			SetName("uuid-revision-index").
			SetUnique(true),
	}

	ctx, cancel := context.WithTimeout(context.Background(), mongoIndexCreationTimeout)
	defer cancel()

	for coll := range ma.collections {
		indexes := ma.client.Database(ma.dbName).Collection(coll).Indexes()
		if _, err := indexes.CreateOne(ctx, index); err != nil {
			logger.WithError(err).Infof("could not EnsureIndex for collection %s", coll)
		}
	}
}

func (ma *MongoConnection) Delete(collection string, uuidString string, revision int64) error {
	coll := ma.client.Database(ma.dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoDefaultOperationTimeout)
	defer cancel()

	bsonUUID := bsonx.Binary(0x04, uuid.Parse(uuidString))
	_, err := coll.DeleteOne(ctx, bsonx.Doc{
		{Key: uuidName, Value: bsonUUID},
		{Key: contentRevisionName, Value: bsonx.Int64(revision)},
	})

	return err
}

func (ma *MongoConnection) Write(collection string, resource *mapper.Resource) error {
	coll := ma.client.Database(ma.dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoDefaultOperationTimeout)
	defer cancel()

	bsonUUID := bsonx.Binary(0x04, uuid.Parse(resource.UUID))

	bsonResource := map[string]interface{}{
		"uuid":             bsonUUID,
		"content":          resource.Content,
		"content-type":     resource.ContentType,
		"origin-system-id": resource.OriginSystemID,
		"schema-version":   resource.SchemaVersion,
		"content-revision": resource.ContentRevision,
	}
	filter := bson.M{
		uuidName:            bsonUUID,
		contentRevisionName: resource.ContentRevision,
	}
	update := bson.M{"$set": bsonResource}
	opts := options.Update().SetUpsert(true)
	_, err := coll.UpdateOne(ctx, filter, update, opts)

	return err
}

func (ma *MongoConnection) Read(collection string, uuidString string) (res *mapper.Resource, found bool, err error) {
	coll := ma.client.Database(ma.dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoDefaultOperationTimeout)
	defer cancel()

	bsonUUID := bsonx.Binary(0x04, uuid.Parse(uuidString))
	opts := options.FindOne().
		SetSort(bsonx.Doc{
			{Key: "content-revision", Value: bsonx.Int32(-1)},
		})
	result := coll.FindOne(ctx, bson.M{uuidName: bsonUUID}, opts)

	if err = result.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return res, false, nil
		}
		return res, false, err
	}

	var bsonResource map[string]interface{}
	if err = result.Decode(&bsonResource); err != nil {
		return res, false, err
	}

	res = ma.mapBsonToResource(bsonResource)
	return res, true, nil
}

func (ma *MongoConnection) ReadSingleRevision(collection string, uuidString string, revision int64) (res *mapper.Resource, err error) {
	coll := ma.client.Database(ma.dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoDefaultOperationTimeout)
	defer cancel()

	bsonUUID := bsonx.Binary(0x04, uuid.Parse(uuidString))
	result := coll.FindOne(ctx,
		bson.M{
			uuidName:           bsonUUID,
			"content-revision": revision})

	if err = result.Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	var bsonResource map[string]interface{}
	if err = result.Decode(&bsonResource); err != nil {
		return res, err
	}

	res = ma.mapBsonToResource(bsonResource)
	return res, nil
}

func (ma *MongoConnection) mapBsonToResource(bsonResource map[string]interface{}) *mapper.Resource {
	uuidData := bsonResource["uuid"].(primitive.Binary).Data

	res := &mapper.Resource{
		UUID:        uuid.UUID(uuidData).String(),
		Content:     bsonResource["content"],
		ContentType: bsonResource["content-type"].(string),
	}

	originSystemID, found := bsonResource["origin-system-id"]
	if found {
		res.OriginSystemID = originSystemID.(string)
	}

	schemaVersion, found := bsonResource["schema-version"]
	if found {
		res.SchemaVersion = schemaVersion.(string)
	}

	contentRevision, found := bsonResource["content-revision"]
	if found {
		res.ContentRevision = contentRevision.(int64)
	}

	return res
}

func (ma *MongoConnection) ReadRevisions(collection string, uuidString string) (res []int64, err error) {
	coll := ma.client.Database(ma.dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoDefaultOperationTimeout)
	defer cancel()

	bsonUUID := bsonx.Binary(0x04, uuid.Parse(uuidString))
	opts := options.Find().SetProjection(bson.M{"content-revision": 1})
	cur, err := coll.Find(ctx, bson.M{uuidName: bsonUUID}, opts)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	defer cur.Close(ctx)

	var bsonResource map[string]interface{}
	res = []int64{}
	for cur.Next(ctx) {
		err = cur.Decode(&bsonResource)
		if err != nil {
			return nil, err
		}
		s, ok := bsonResource["content-revision"].(int64)
		if !ok {
			return nil, err
		}
		res = append(res, s)
	}

	return res, nil
}

func (ma *MongoConnection) Count(collection string, uuidString string, contentRevision int64) (count int64, err error) {
	coll := ma.client.Database(ma.dbName).Collection(collection)
	ctx, cancel := context.WithTimeout(context.Background(), mongoDefaultOperationTimeout)
	defer cancel()

	bsonUUID := bsonx.Binary(0x04, uuid.Parse(uuidString))
	n, err := coll.CountDocuments(ctx, bson.M{uuidName: bsonUUID, contentRevisionName: contentRevision})

	if err != nil {
		return 0, err
	}
	return n, nil
}

func (ma *MongoConnection) ReadIDs(ctx context.Context, collection string) (chan string, error) {
	coll := ma.client.Database(ma.dbName).Collection(collection)

	opts := options.Find().
		SetProjection(bson.M{uuidName: true}).
		SetBatchSize(32)
	cur, err := coll.Find(ctx, bson.M{}, opts)

	ids := make(chan string, 8)
	if err != nil {
		return ids, err
	}

	go func() {
		defer cur.Close(ctx)
		defer close(ids)

		var result map[string]interface{}
		for cur.Next(ctx) {
			if ctx.Err() != nil {
				//canceling the context doesn't cancel the `cur.Next()` until the batch fetch is exhausted
				break
			}
			if err := cur.Decode(&result); err != nil {
				break
			}
			ids <- uuid.UUID(result["uuid"].(primitive.Binary).Data).String()
		}
	}()

	return ids, nil
}

func (ma *MongoConnection) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), mongoDefaultOperationTimeout)
	defer cancel()

	return ma.client.Ping(ctx, readpref.Primary())
}
