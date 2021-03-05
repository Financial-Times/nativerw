package resources

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Financial-Times/nativerw/pkg/mapper"
)

func TestReadContent(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", "methode", "a-real-uuid").
		Return(
			&mapper.Resource{
				ContentType: "application/json",
				Content:     map[string]interface{}{"uuid": "fake-data"}},
			true,
			nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"uuid":"fake-data"}`, strings.TrimSpace(w.Body.String()))
}

func TestReadRevisions(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("ReadRevisions", "universal-publishing", "a-real-uuid").
		Return(
			[]int64{1, 2, 3},
			nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}/revisions", ReadRevisions(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-publishing/a-real-uuid/revisions", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `[1,2,3]`, strings.TrimSpace(w.Body.String()))
}

func TestReadContentWithCharsetDirective(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", "methode", "a-real-uuid").Return(&mapper.Resource{ContentType: "application/json; charset=utf-8", Content: map[string]interface{}{"uuid": "fake-data"}}, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"uuid":"fake-data"}`, strings.TrimSpace(w.Body.String()))
}

func TestReadFailed(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", "methode", "a-real-uuid").Return(&mapper.Resource{}, false, errors.New("i failed"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIDNotFound(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", "methode", "a-real-uuid").Return(&mapper.Resource{}, false, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNoMapperImplemented(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", "methode", "a-real-uuid").Return(&mapper.Resource{ContentType: "application/vnd.fake-mime-type"}, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestUnableToMap(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", "methode", "a-real-uuid").Return(&mapper.Resource{ContentType: "application/json", Content: func() {}}, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	t.Log(w.Body.String())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFailedMongoOnRead(t *testing.T) {
	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestReadIDs(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	ids := make(chan string, 1)

	mongo.On("Open").Return(connection, nil)
	connection.On("ReadIDs", mock.AnythingOfType("*context.timerCtx"), "methode").Return(ids, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/__ids", ReadIDs(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/__ids", http.NoBody)

	go func() {
		ids <- "hi"
		close(ids)
	}()

	router.ServeHTTP(w, req)

	mongo.AssertExpectations(t)
	connection.AssertExpectations(t)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"id":"hi"}`, strings.TrimSpace(w.Body.String()))
}

func TestReadIDsMongoOpenFails(t *testing.T) {
	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/__ids", ReadIDs(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/__ids", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestReadIDsMongoCallFails(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	ids := make(chan string, 1)

	mongo.On("Open").Return(connection, nil)
	connection.On("ReadIDs", mock.AnythingOfType("*context.timerCtx"), "methode").Return(ids, errors.New(`oh no`))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/__ids", ReadIDs(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/methode/__ids", http.NoBody)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
