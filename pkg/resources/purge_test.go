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
	connection := new(MockConnection)
	connection.On("Delete", "universal-content", "a-real-uuid", int64(123)).Return(nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/purge/{resource}/{revision}", PurgeContent(connection)).Methods("DELETE")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/universal-content/purge/a-real-uuid/123", strings.NewReader(``))

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestFailedDelete(t *testing.T) {
	connection := new(MockConnection)
	connection.On("Delete", "universal-content", "a-real-uuid", int64(123)).Return(errors.New("i failed"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/purge/{resource}/{revision}", PurgeContent(connection)).Methods("DELETE")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/universal-content/purge/a-real-uuid/123", strings.NewReader(``))

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
