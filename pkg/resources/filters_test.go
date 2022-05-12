package resources

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/Financial-Times/go-logger"
)

func init() {
	logger.InitLogger("nativerw", "info")
}

var testCollections = map[string]bool{
	"universal-content": true,
}

var validationTests = []struct {
	collectionID  string
	resourceID    string
	expectedError error
}{
	{
		"universal-content",
		"9694733e-163a-4393-801f-000ab7de5041",
		nil,
	},
	{
		"other",
		"9694733e-163a-4393-801f-000ab7de5041",
		errors.New("collection not supported or resourceId not a valid uuid"),
	},
}

func TestValidateAccess(t *testing.T) {
	forwarded := false
	next := func(w http.ResponseWriter, r *http.Request) {
		forwarded = true
	}

	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("GetSupportedCollections").Return(testCollections)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", Filter(next).ValidateAccess(mongo).Build()).Methods("GET")

	for _, test := range validationTests {
		forwarded = false
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+test.collectionID+"/"+test.resourceID, ioutil.NopCloser(nil))

		router.ServeHTTP(w, req)
		mongo.AssertExpectations(t)
		if test.expectedError == nil {
			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, forwarded)
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.False(t, forwarded)
		}
	}
}

func TestValidateAccessForCollection(t *testing.T) {
	forwarded := false
	next := func(w http.ResponseWriter, r *http.Request) {
		forwarded = true
	}

	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("GetSupportedCollections").Return(testCollections)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", Filter(next).ValidateAccessForCollection(mongo).Build()).Methods("GET")

	for _, test := range validationTests {
		forwarded = false
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/"+test.collectionID+"/"+test.resourceID, ioutil.NopCloser(nil))

		router.ServeHTTP(w, req)
		mongo.AssertExpectations(t)
		if test.expectedError == nil {
			assert.Equal(t, http.StatusOK, w.Code)
			assert.True(t, forwarded)
		} else {
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.False(t, forwarded)
		}
	}
}

func TestFailedMongoDuringAccessValidation(t *testing.T) {
	next := func(w http.ResponseWriter, r *http.Request) {
		t.Fail()
	}

	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", Filter(next).ValidateAccess(mongo).Build()).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/9694733e-163a-4393-801f-000ab7de5041", ioutil.NopCloser(nil))

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestFailedMongoDuringCollectionValidation(t *testing.T) {
	next := func(w http.ResponseWriter, r *http.Request) {
		t.Fail()
	}

	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", Filter(next).ValidateAccessForCollection(mongo).Build()).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/9694733e-163a-4393-801f-000ab7de5041", ioutil.NopCloser(nil))

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
