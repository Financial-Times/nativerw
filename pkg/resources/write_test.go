package resources

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/Financial-Times/nativerw/pkg/mapper"
)

type fixedTimestampCreator struct{}

func (f *fixedTimestampCreator) CreateTimestamp() string {
	return "2020-11-25T21:48:05.999Z"
}

func TestWriteContent(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write",
		"methode",
		&mapper.Resource{
			UUID:            "a-real-uuid",
			Content:         map[string]interface{}{},
			ContentType:     "application/json",
			ContentRevision: "2020-11-25T21:48:05.999Z"}).
		Return(nil)

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", WriteContent(mongo, &ts)).Methods("PUT")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/methode/a-real-uuid", strings.NewReader(`{}`))

	req.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWriteContentWithCharsetDirective(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)

	connection.On("Write",
		"methode",
		&mapper.Resource{
			UUID:            "a-real-uuid",
			Content:         map[string]interface{}{},
			ContentType:     "application/json; charset=utf-8",
			ContentRevision: "2020-11-25T21:48:05.999Z"}).
		Return(nil)

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", WriteContent(mongo, &ts)).Methods("PUT")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/methode/a-real-uuid", strings.NewReader(`{}`))

	req.Header.Add("Content-Type", "application/json; charset=utf-8")

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWriteFailed(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write",
		"methode",
		&mapper.Resource{
			UUID:            "a-real-uuid",
			Content:         map[string]interface{}{},
			ContentType:     "application/json",
			ContentRevision: "2020-11-25T21:48:05.999Z"}).
		Return(errors.New("i failed"))

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", WriteContent(mongo, &ts)).Methods("PUT")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/methode/a-real-uuid", strings.NewReader(`{}`))

	req.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestDefaultsToBinaryMapping(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	inMapper, err := mapper.InMapperForContentType("application/octet-stream")
	assert.NoError(t, err)

	content, err := inMapper(ioutil.NopCloser(strings.NewReader(`{}`)))
	assert.NoError(t, err)

	connection.On("Write",
		"methode",
		&mapper.Resource{
			UUID:            "a-real-uuid",
			Content:         content,
			ContentType:     "application/octet-stream",
			ContentRevision: "2020-11-25T21:48:05.999Z"}).
		Return(errors.New("i failed"))

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", WriteContent(mongo, &ts)).Methods("PUT")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/methode/a-real-uuid", strings.NewReader(`{}`))

	req.Header.Add("Content-Type", "application/a-fake-type")

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFailedJSON(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", WriteContent(mongo, &ts)).Methods("PUT")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/methode/a-real-uuid", strings.NewReader(`i am not json`))

	req.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
