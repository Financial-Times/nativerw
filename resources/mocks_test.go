package resources

import (
	"github.com/Financial-Times/nativerw/db"
	"github.com/Financial-Times/nativerw/mapper"
	"github.com/stretchr/testify/mock"
)

type MockConnection struct {
	mock.Mock
	CallArgs []interface{}
}

type MockDB struct {
	mock.Mock
}

func (m *MockDB) Open() (db.Connection, error) {
	args := m.Called()
	conn := args.Get(0)
	if conn != nil {
		return conn.(*MockConnection), args.Error(1)
	}

	return nil, args.Error(1)
}

func (m *MockDB) Await() (db.Connection, error) {
	args := m.Called()
	return args.Get(0).(*MockConnection), args.Error(1)
}

func (m *MockConnection) EnsureIndex() {
	m.Called()
}

func (m *MockConnection) GetSupportedCollections() map[string]bool {
	args := m.Called()
	return args.Get(0).(map[string]bool)
}

func (m *MockConnection) Close() {
	m.Called()
}

func (m *MockConnection) Delete(collection string, uuidString string) error {
	args := m.Called(collection, uuidString)
	return args.Error(0)
}

func (m *MockConnection) Ids(collection string, stopChan chan struct{}, errChan chan error) chan string {
	args := m.Called(collection, stopChan, errChan)
	m.CallArgs = []interface{}{collection, stopChan, errChan}
	return args.Get(0).(chan string)
}

func (m *MockConnection) Write(collection string, resource mapper.Resource) error {
	args := m.Called(collection, resource)
	return args.Error(0)
}

func (m *MockConnection) Read(collection string, uuidString string) (res mapper.Resource, found bool, err error) {
	args := m.Called(collection, uuidString)
	return args.Get(0).(mapper.Resource), args.Bool(1), args.Error(2)
}
