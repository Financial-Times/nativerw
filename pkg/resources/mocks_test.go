package resources

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/Financial-Times/nativerw/pkg/mapper"
)

type MockConnection struct {
	mock.Mock
	CallArgs []interface{}
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

func (m *MockConnection) Delete(collection string, uuidString string, revision int64) error {
	args := m.Called(collection, uuidString, revision)
	return args.Error(0)
}

func (m *MockConnection) ReadIDs(ctx context.Context, collection string) (chan string, error) {
	args := m.Called(ctx, collection)
	m.CallArgs = []interface{}{ctx, collection}
	return args.Get(0).(chan string), args.Error(1)
}

func (m *MockConnection) Write(collection string, resource *mapper.Resource) error {
	args := m.Called(collection, resource)
	return args.Error(0)
}

func (m *MockConnection) Read(collection string, uuidString string) (res *mapper.Resource, found bool, err error) {
	args := m.Called(collection, uuidString)
	return args.Get(0).(*mapper.Resource), args.Bool(1), args.Error(2)
}

func (m *MockConnection) ReadSingleRevision(collection string, uuidString string, revision int64) (res *mapper.Resource, err error) {
	args := m.Called(collection, uuidString, revision)
	return args.Get(0).(*mapper.Resource), args.Error(1)
}

func (m *MockConnection) ReadRevisions(collection string, uuidString string) (res []int64, err error) {
	args := m.Called(collection, uuidString)
	return args.Get(0).([]int64), args.Error(1)
}

func (m *MockConnection) Count(collection string, uuidString string, contentRevision int64) (count int64, err error) {
	args := m.Called(collection, uuidString, contentRevision)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockConnection) Ping() error {
	m.Called()
	return nil
}
