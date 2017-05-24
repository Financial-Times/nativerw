package resources

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestHealthchecks(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write", healthcheckColl, sampleResource).Return(nil)
	connection.On("Read", healthcheckColl, sampleUUID).Return(sampleResource, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/__health", Healthchecks(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__health", nil)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthchecksFail(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write", healthcheckColl, sampleResource).Return(errors.New("no writes 4 u"))
	connection.On("Read", healthcheckColl, sampleUUID).Return(sampleResource, true, errors.New("no reads 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/__health", Healthchecks(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__health", nil)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)

	expected := regexp.MustCompile(`\{"checks":\[\{"businessImpact":"Publishing won't work. Writing content to native store is broken.","checkOutput":"no writes 4 u","lastUpdated":".*","name":"Write to mongoDB","ok":false,"panicGuide":".*","severity":2,"technicalSummary":"Writing to mongoDB is broken. Check mongoDB is up, its disk space, ports, network."\},\{"businessImpact":"Reading content from native store is broken.","checkOutput":"no reads 4 u","lastUpdated":".*","name":"Read from mongoDB","ok":false,"panicGuide":".*","severity":2,"technicalSummary":"Reading from mongoDB is broken. Check mongoDB is up, its disk space, ports, network."\}\],"description":"Checking connectivity and usability of dependent services: mongoDB.","name":"Dependent services healthcheck","schemaVersion":1,"ok":false,"severity":2\}`)
	assert.Regexp(t, expected, strings.TrimSpace(w.Body.String()))
}

func TestG2G(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write", healthcheckColl, sampleResource).Return(nil)
	connection.On("Read", healthcheckColl, sampleUUID).Return(sampleResource, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/__gtg", GoodToGo(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__gtg", nil)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=US-ASCII", w.Result().Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Result().Header.Get("Cache-Control"))
}

func TestG2GFailsOnRead(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", healthcheckColl, sampleUUID).Return(sampleResource, true, errors.New("no reads 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/__gtg", GoodToGo(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__gtg", nil)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "text/plain; charset=US-ASCII", w.Result().Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Result().Header.Get("Cache-Control"))
}

func TestG2GFailsOnWrite(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write", healthcheckColl, sampleResource).Return(errors.New("no writes 4 u"))
	connection.On("Read", healthcheckColl, sampleUUID).Return(sampleResource, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/__gtg", GoodToGo(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__gtg", nil)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "text/plain; charset=US-ASCII", w.Result().Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Result().Header.Get("Cache-Control"))
}

func TestFailedMongoDuringHealthcheck(t *testing.T) {
	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/__health", Healthchecks(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__health", nil)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFailedMongoDuringGTG(t *testing.T) {
	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/__gtg", GoodToGo(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__gtg", nil)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "text/plain; charset=US-ASCII", w.Result().Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Result().Header.Get("Cache-Control"))
}
