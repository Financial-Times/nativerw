package resources

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	status "github.com/Financial-Times/service-status-go/httphandlers"
)

func TestHealthchecks(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write", healthCheckColl, sampleResource).Return(nil)
	connection.On("Read", healthCheckColl, sampleUUID).Return(sampleResource, true, nil)

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
	connection.On("Write", healthCheckColl, sampleResource).Return(errors.New("no writes 4 u"))
	connection.On("Read", healthCheckColl, sampleUUID).Return(sampleResource, true, errors.New("no reads 4 u"))

	router := mux.NewRouter()
	router.HandleFunc("/__health", Healthchecks(mongo)).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__health", nil)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)

	healthResult := fthealth.HealthResult{}
	dec := json.NewDecoder(w.Body)
	err := dec.Decode(&healthResult)
	assert.NoError(t, err)

	assert.Equal(t, 1.0, healthResult.SchemaVersion)
	assert.Equal(t, "nativerw", healthResult.Name)
	assert.Equal(t, "nativestorereaderwriter", healthResult.SystemCode)
	assert.Equal(t, "Reads and Writes data to the UPP Native Store, in the received (native) format", healthResult.Description)
	assert.False(t, healthResult.Ok)
	assert.Equal(t, uint8(1), healthResult.Severity)

	for _, check := range healthResult.Checks {
		if check.Name == "Write to mongoDB" {
			assert.Equal(t, "Publishing won't work. Writing content to native store is broken.", check.BusinessImpact)
			assert.Equal(t, "Writing to mongoDB is broken. Check mongoDB is up, its disk space, ports, network.", check.TechnicalSummary)
		} else if check.Name == "Read from mongoDB" {
			assert.Equal(t, "Reading content from native store is broken.", check.BusinessImpact)
			assert.Equal(t, "Reading from mongoDB is broken. Check mongoDB is up, its disk space, ports, network.", check.TechnicalSummary)
		} else {
			t.Fail() // a new test has been introduced that isn't covered here
		}
		assert.Equal(t, "https://runbooks.ftops.tech/nativestorereaderwriter", check.PanicGuide)
		assert.False(t, check.Ok)
		assert.Equal(t, uint8(1), check.Severity)
	}
}

func TestGTG(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write", healthCheckColl, sampleResource).Return(nil)
	connection.On("Read", healthCheckColl, sampleUUID).Return(sampleResource, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/__gtg", status.NewGoodToGoHandler(GoodToGo(mongo))).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__gtg", nil)

	router.ServeHTTP(w, req)

	r := w.Result()
	defer r.Body.Close()

	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=US-ASCII", r.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", r.Header.Get("Cache-Control"))
}

func TestGTGFailsOnRead(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", healthCheckColl, sampleUUID).Return(sampleResource, true, errors.New("no reads 4 u"))
	connection.On("Write", healthCheckColl, sampleResource).Return(nil)

	router := mux.NewRouter()
	router.HandleFunc("/__gtg", status.NewGoodToGoHandler(GoodToGo(mongo))).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__gtg", nil)

	router.ServeHTTP(w, req)

	r := w.Result()
	defer r.Body.Close()

	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "text/plain; charset=US-ASCII", r.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", r.Header.Get("Cache-Control"))
}

func TestGTGFailsOnWrite(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	mongo.On("Open").Return(connection, nil)
	connection.On("Write", healthCheckColl, sampleResource).Return(errors.New("no writes 4 u"))
	connection.On("Read", healthCheckColl, sampleUUID).Return(sampleResource, true, nil)

	router := mux.NewRouter()
	router.HandleFunc("/__gtg", status.NewGoodToGoHandler(GoodToGo(mongo))).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__gtg", nil)

	router.ServeHTTP(w, req)

	r := w.Result()
	defer r.Body.Close()

	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "text/plain; charset=US-ASCII", r.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", r.Header.Get("Cache-Control"))
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
	router.HandleFunc("/__gtg", status.NewGoodToGoHandler(GoodToGo(mongo))).Methods("GET")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__gtg", nil)

	router.ServeHTTP(w, req)

	r := w.Result()
	defer r.Body.Close()

	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, "text/plain; charset=US-ASCII", r.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", r.Header.Get("Cache-Control"))
}
