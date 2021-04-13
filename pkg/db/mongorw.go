package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pborman/uuid"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/config"
	"github.com/Financial-Times/nativerw/pkg/mapper"
)

const (
	uuidName            = "uuid"
	contentRevisionName = "content-revision"
)

type mongoDB struct {
	config     *config.Configuration
	connection *Optional
}

type mongoConnection struct {
	dbName      string
	session     *mgo.Session
	collections map[string]bool
}

// DB handles opening the initial connection to Mongo
type DB interface {
	Open() (Connection, error)
	Await() (Connection, error)
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
	Count(collection string, uuidString string, contentRevision int64) (count int, err error)
	Close()
}

// NewDBConnection dials the mongo cluster, and returns a new handler DB instance
func NewDBConnection(config *config.Configuration) DB {
	return &mongoDB{config: config}
}

func (m *mongoDB) Await() (Connection, error) {
	if m.connection == nil {
		return nil, errors.New("please Open() a new connection before awaiting")
	}

	if m.connection.Nil() {
		connection, err := m.connection.Block()
		if err != nil {
			return nil, err
		}
		return connection.(*mongoConnection), err
	}
	return m.connection.Get().(*mongoConnection), nil
}

func (m *mongoDB) Open() (Connection, error) {
	if m.connection == nil {
		m.connection = NewOptional(func() (interface{}, error) {
			connection, err := m.openMongoSession()
			for err != nil {
				logger.WithError(err).Error("couldn't establish connection to mongoDB")
				time.Sleep(5 * time.Second)

				connection, err = m.openMongoSession()
			}
			return connection, err
		})

		connection, err := m.connection.Block()
		if err != nil {
			return nil, err
		}

		return connection.(*mongoConnection), err
	}

	if m.connection.Nil() {
		return nil, errors.New("mongo connection is not yet initialised")
	}

	return m.connection.Get().(*mongoConnection), nil
}

func (m *mongoDB) openMongoSession() (*mongoConnection, error) {
	session, err := mgo.DialWithTimeout(m.config.Mongos, 30*time.Second)
	if err != nil {
		return nil, err
	}

	session.SetMode(mgo.Strong, true)
	collections := createMapWithAllowedCollections(m.config.Collections)
	connection := &mongoConnection{m.config.DBName, session, collections}

	return connection, nil
}

func (ma *mongoConnection) GetSupportedCollections() map[string]bool {
	return ma.collections
}

func (ma *mongoConnection) Close() {
	ma.session.Close()
}

func createMapWithAllowedCollections(collections []string) map[string]bool {
	var collectionMap = make(map[string]bool)
	for _, coll := range collections {
		collectionMap[coll] = true
	}
	return collectionMap
}

func (ma *mongoConnection) EnsureIndex() {
	newSession := ma.session.Copy()
	defer newSession.Close()

	index := mgo.Index{
		Name:       "uuid-revision-index",
		Key:        []string{"uuid", "content-revision"},
		Background: true,
		Unique:     true,
	}

	for coll := range ma.collections {
		if err := newSession.DB(ma.dbName).C(coll).EnsureIndex(index); err != nil {
			logger.WithError(err).Info("could not EnsureIndex: %s ", index)
		}
	}
}

func (ma *mongoConnection) Delete(collection string, uuidString string, revision int64) error {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)
	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(uuidString))}

	return coll.Remove(bson.D{
		bson.DocElem{Name: uuidName, Value: bsonUUID},
		bson.DocElem{Name: contentRevisionName, Value: revision},
	})
}

func (ma *mongoConnection) Write(collection string, resource *mapper.Resource) error {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)

	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(resource.UUID))}

	bsonResource := map[string]interface{}{
		"uuid":             bsonUUID,
		"content":          resource.Content,
		"content-type":     resource.ContentType,
		"origin-system-id": resource.OriginSystemID,
		"schema-version":   resource.SchemaVersion,
		"content-revision": resource.ContentRevision,
	}

	_, err := coll.Upsert(
		bson.D{
			bson.DocElem{Name: uuidName, Value: bsonUUID},
			bson.DocElem{Name: contentRevisionName, Value: resource.ContentRevision}},
		bsonResource)

	return err
}

func (ma *mongoConnection) Read(collection string, uuidString string) (res *mapper.Resource, found bool, err error) {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)

	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(uuidString))}

	var bsonResource map[string]interface{}

	if err = coll.Find(bson.M{uuidName: bsonUUID}).Sort("-content-revision").One(&bsonResource); err != nil {
		if err == mgo.ErrNotFound {
			return res, false, nil
		}
		return res, false, err
	}

	res = ma.mapBsonToResource(bsonResource)
	return res, true, nil
}

func (ma *mongoConnection) ReadSingleRevision(collection string, uuidString string, revision int64) (res *mapper.Resource, err error) {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)

	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(uuidString))}

	var bsonResource map[string]interface{}

	err = coll.Find(
		bson.M{
			uuidName:           bsonUUID,
			"content-revision": revision}).
		One(&bsonResource)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	res = ma.mapBsonToResource(bsonResource)
	return res, nil
}

func (ma *mongoConnection) mapBsonToResource(bsonResource map[string]interface{}) *mapper.Resource {
	uuidData := bsonResource["uuid"].(bson.Binary).Data

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

func (ma *mongoConnection) ReadRevisions(collection string, uuidString string) (res []int64, err error) {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)

	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(uuidString))}

	var bsonResource []map[string]interface{}
	if err = coll.Find(bson.M{uuidName: bsonUUID}).Select(bson.M{"content-revision": 1}).All(&bsonResource); err != nil {
		if err == mgo.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}

	res = []int64{}
	for _, v := range bsonResource {
		s, ok := v["content-revision"].(int64)
		if !ok {
			return nil, err
		}
		res = append(res, s)
	}

	return res, nil
}

func (ma *mongoConnection) Count(collection string, uuidString string, contentRevision int64) (count int, err error) {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)

	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(uuidString))}

	n, err := coll.Find(bson.M{uuidName: bsonUUID, contentRevisionName: contentRevision}).Count()
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (ma *mongoConnection) ReadIDs(ctx context.Context, collection string) (chan string, error) {
	ids := make(chan string, 8)

	newSession := ma.session.Copy()
	coll := newSession.DB(ma.dbName).C(collection)

	iter := coll.Find(nil).Select(bson.M{uuidName: true}).Batch(32).Iter()

	if err := iter.Err(); err != nil {
		newSession.Close()
		return ids, err
	}

	go func() {
		defer newSession.Close()
		defer iter.Close()
		defer close(ids)

		var result map[string]interface{}

		for iter.Next(&result) {
			if err := ctx.Err(); err != nil {
				break
			}

			ids <- uuid.UUID(result["uuid"].(bson.Binary).Data).String()
		}
	}()

	return ids, nil
}

func CheckMongoUrls(providedMongoUrls string, expectedMongoNodeCount int) error {
	mongoUrls := strings.Split(providedMongoUrls, ",")
	actualMongoNodeCount := len(mongoUrls)
	if actualMongoNodeCount != expectedMongoNodeCount {
		return fmt.Errorf("the provided list of MongoDB URLs should have %d instances, but it has %d instead. Provided MongoDB URLs are: %s", expectedMongoNodeCount, actualMongoNodeCount, providedMongoUrls)
	}

	for _, mongoURL := range mongoUrls {
		urlComponents := strings.Split(mongoURL, ":")
		noOfURLComponents := len(urlComponents)

		if noOfURLComponents != 2 || urlComponents[0] == "" || urlComponents[1] == "" {
			return fmt.Errorf("one of the MongoDB URLs is invalid: %s. It should have host and port", mongoURL)
		}
	}

	return nil
}
