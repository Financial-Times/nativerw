package resources

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func TestDeleteContent(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	connection.On("Delete", "methode", "a-real-uuid", int64(123)).Return(nil)
	mongo.On("Open").Return(connection, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/purge/{resource}/{revision}", PurgeContent(mongo)).Methods("DELETE")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/methode/purge/a-real-uuid/123", strings.NewReader(``))

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFailedDelete(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	connection.On("Delete", "methode", "a-real-uuid", int64(123)).Return(errors.New("i failed"))
	mongo.On("Open").Return(connection, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/purge/{resource}/{revision}", PurgeContent(mongo)).Methods("DELETE")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/methode/purge/a-real-uuid/123", strings.NewReader(``))

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFailedMongoOnDelete(t *testing.T) {
	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/purge/{resource}/{revision}", PurgeContent(mongo)).Methods("DELETE")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/methode/purge/a-real-uuid/123", strings.NewReader(``))

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
