package db

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Financial-Times/nativerw/config"
	"github.com/Financial-Times/nativerw/logging"
	"github.com/Financial-Times/nativerw/mapper"
	"github.com/pborman/uuid"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"strings"
)

const uuidName = "uuid"

type mongoDB struct {
	config     *config.Configuration
	connection *Optional
	once       *sync.Once
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
	Delete(collection string, uuidString string) error
	Write(collection string, resource mapper.Resource) error
	Read(collection string, uuidString string) (res mapper.Resource, found bool, err error)
	ReadIDs(ctx context.Context, collection string) (chan string, error)
	Close()
}

// NewDBConnection dials the mongo cluster, and returns a new handler DB instance
func NewDBConnection(config *config.Configuration) DB {
	return &mongoDB{config: config}
}

func (m *mongoDB) Await() (Connection, error) {
	if m.connection == nil {
		return nil, errors.New("Please Open() a new connection before awaiting.")
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
				logging.Error(fmt.Sprintf("Couldn't establish connection to mongoDB: %+v", err.Error()))
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
		return nil, errors.New("Mongo connection is not yet initialised!")
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
	connection := &mongoConnection{m.config.DbName, session, collections}

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
		Name:       "uuid-index",
		Key:        []string{"uuid"},
		Background: true,
		Unique:     true,
	}

	for coll := range ma.collections {
		newSession.DB(ma.dbName).C(coll).EnsureIndex(index)
	}
}

func (ma *mongoConnection) Delete(collection string, uuidString string) error {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)
	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(uuidString))}

	return coll.Remove(bson.D{{uuidName, bsonUUID}})
}

func (ma *mongoConnection) Write(collection string, resource mapper.Resource) error {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)

	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(resource.UUID))}
	bsonResource := map[string]interface{}{
		"uuid":         bsonUUID,
		"content":      resource.Content,
		"content-type": resource.ContentType,
	}

	_, err := coll.Upsert(bson.D{{uuidName, bsonUUID}}, bsonResource)

	return err
}

func (ma *mongoConnection) Read(collection string, uuidString string) (res mapper.Resource, found bool, err error) {
	newSession := ma.session.Copy()
	defer newSession.Close()

	coll := newSession.DB(ma.dbName).C(collection)

	bsonUUID := bson.Binary{Kind: 0x04, Data: []byte(uuid.Parse(uuidString))}

	var bsonResource map[string]interface{}

	if err = coll.Find(bson.M{uuidName: bsonUUID}).One(&bsonResource); err != nil {
		if err == mgo.ErrNotFound {
			return res, false, nil
		}
		return res, false, err
	}

	uuidData := bsonResource["uuid"].(bson.Binary).Data

	res = mapper.Resource{
		UUID:        uuid.UUID(uuidData).String(),
		Content:     bsonResource["content"],
		ContentType: bsonResource["content-type"].(string),
	}

	return res, true, nil
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
		return fmt.Errorf("The provided list of MongoDB URLs should have %d instances, but it has %d instead. Provided MongoDB URLs are: %s", expectedMongoNodeCount, actualMongoNodeCount, providedMongoUrls)
	}

	for _, mongoUrl := range mongoUrls {
		urlComponents := strings.Split(mongoUrl, ":")
		noOfUrlComponents := len(urlComponents)

		if noOfUrlComponents != 2 {
			return fmt.Errorf("One of the MongoDB URLs is invalid: %s. It should have host and port.", mongoUrl)
		}
	}

	return nil
}
