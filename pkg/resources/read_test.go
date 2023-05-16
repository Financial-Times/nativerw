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
	connection := new(MockConnection)
	connection.On("Read", "universal-content", "a-real-uuid").
		Return(
			&mapper.Resource{
				ContentType: "application/json",
				Content:     map[string]interface{}{"uuid": "fake-data"}},
			true,
			nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"uuid":"fake-data"}`, strings.TrimSpace(w.Body.String()))
}

func TestReadRevisions(t *testing.T) {
	connection := new(MockConnection)
	connection.On("ReadRevisions", "universal-publishing", "a-real-uuid").
		Return(
			[]int64{1, 2, 3},
			nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}/revisions", ReadRevisions(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-publishing/a-real-uuid/revisions", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `[1,2,3]`, strings.TrimSpace(w.Body.String()))
}

func TestReadSingleRevision(t *testing.T) {
	connection := new(MockConnection)
	connection.On("ReadSingleRevision", "universal-content", "a-real-uuid", int64(1)).
		Return(
			&mapper.Resource{
				ContentType: "application/json",
				Content:     map[string]interface{}{"uuid": "fake-data"}},
			nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}/{revision}", ReadSingleRevision(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/a-real-uuid/1", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"uuid":"fake-data"}`, strings.TrimSpace(w.Body.String()))
}

func TestReadContentWithCharsetDirective(t *testing.T) {
	connection := new(MockConnection)
	connection.On("Read", "universal-content", "a-real-uuid").Return(&mapper.Resource{ContentType: "application/json; charset=utf-8", Content: map[string]interface{}{"uuid": "fake-data"}}, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, `{"uuid":"fake-data"}`, strings.TrimSpace(w.Body.String()))
}

func TestReadFailed(t *testing.T) {
	connection := new(MockConnection)
	connection.On("Read", "universal-content", "a-real-uuid").Return(&mapper.Resource{}, false, errors.New("i failed"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestIDNotFound(t *testing.T) {
	connection := new(MockConnection)
	connection.On("Read", "universal-content", "a-real-uuid").Return(&mapper.Resource{}, false, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestNoMapperImplemented(t *testing.T) {
	connection := new(MockConnection)
	connection.On("Read", "universal-content", "a-real-uuid").Return(&mapper.Resource{ContentType: "application/vnd.fake-mime-type"}, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func TestUnableToMap(t *testing.T) {
	connection := new(MockConnection)
	connection.On("Read", "universal-content", "a-real-uuid").Return(&mapper.Resource{ContentType: "application/json", Content: func() {}}, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	t.Log(w.Body.String())
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestFailedMongoOnRead(t *testing.T) {
	connection := new(MockConnection)
	connection.On("Read", "universal-content", "a-real-uuid").Return(&mapper.Resource{}, false, errors.New("no data 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", ReadContent(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/a-real-uuid", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReadIDs(t *testing.T) {
	connection := new(MockConnection)
	ids := make(chan string, 1)
	connection.On("ReadIDs", mock.AnythingOfType("*context.timerCtx"), "universal-content").Return(ids, nil)

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/__ids", ReadIDs(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/__ids", http.NoBody)

	go func() {
		ids <- "hi"
		close(ids)
	}()

	router.ServeHTTP(w, req)

	connection.AssertExpectations(t)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `{"id":"hi"}`, strings.TrimSpace(w.Body.String()))
}

func TestReadIDsMongoCallFails(t *testing.T) {
	ids := make(chan string, 1)

	connection := new(MockConnection)
	connection.On("ReadIDs", mock.AnythingOfType("*context.timerCtx"), "universal-content").Return(ids, errors.New(`oh no`))

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/__ids", ReadIDs(connection)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/universal-content/__ids", http.NoBody)

	router.ServeHTTP(w, req)
	connection.AssertExpectations(t)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}
