package resources

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/Financial-Times/go-logger"

	"github.com/Financial-Times/nativerw/pkg/db"
	"github.com/gorilla/mux"
)

const (
	schemaVerisonHeader = "X-Schema-Version"
)

var uuidRegexp = regexp.MustCompile("^[a-f0-9]{8}-[a-f0-9]{4}-[1-5][a-f0-9]{3}-[a-f0-9]{4}-[a-f0-9]{12}$")

func validateAccess(mongo db.Connection, collectionID, resourceID string) error {
	if mongo.GetSupportedCollections()[collectionID] && uuidRegexp.MatchString(resourceID) {
		return nil
	}
	return errors.New("collection not supported or resourceId not a valid uuid")
}

func validateAccessForCollection(mongo db.Connection, collectionID string) error {
	if mongo.GetSupportedCollections()[collectionID] {
		return nil
	}
	return errors.New("collection not supported.	")
}

// ValidateAccess validates whether the collection exists and the resource ID is in uuid format.
func (f *Filters) ValidateAccess(mongo db.DB) *Filters {
	next := f.next
	f.next = func(w http.ResponseWriter, r *http.Request) {
		connection, err := mongo.Open()
		if err != nil {
			defer r.Body.Close()
			writeMessage(w, "Failed to connect to the database!", http.StatusServiceUnavailable)
			return
		}

		collectionID := mux.Vars(r)["collection"]
		resourceID := mux.Vars(r)["resource"]

		if err := validateAccess(connection, collectionID, resourceID); err != nil {
			defer r.Body.Close()

			tid := obtainTxID(r)
			msg := fmt.Sprintf("Invalid collectionId (%v) or resourceId (%v)", collectionID, resourceID)
			logger.WithTransactionID(tid).WithError(err).Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		next(w, r)
	}
	return f
}

// ValidateSchemaVersion validates whether the X-Schema-Version header is provided and if not fails the request.
func (f *Filters) ValidateSchemaVersion() *Filters {
	next := f.next
	f.next = func(w http.ResponseWriter, r *http.Request) {
		sv := r.Header.Get(schemaVerisonHeader)
		if sv == "" {
			defer r.Body.Close()

			tid := obtainTxID(r)
			msg := fmt.Sprintf("request is missing the %v header", schemaVerisonHeader)
			logger.WithTransactionID(tid).Error(msg)
			http.Error(w, msg, http.StatusBadRequest)

			return
		}

		next(w, r)
	}

	return f
}

// ValidateAccessForCollection validates whether the collection exists
func (f *Filters) ValidateAccessForCollection(mongo db.DB) *Filters {
	next := f.next
	f.next = func(w http.ResponseWriter, r *http.Request) {
		connection, err := mongo.Open()
		if err != nil {
			defer r.Body.Close()
			writeMessage(w, "Failed to connect to the database!", http.StatusServiceUnavailable)
			return
		}

		collection := mux.Vars(r)["collection"]

		if err := validateAccessForCollection(connection, collection); err != nil {
			defer r.Body.Close()
			tid := obtainTxID(r)
			msg := fmt.Sprintf("Invalid collectionId (%v)", collection)
			logger.WithTransactionID(tid).WithError(err).Error(msg)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		next(w, r)
	}
	return f
}

// Filters wraps the next http handler
type Filters struct {
	next func(w http.ResponseWriter, r *http.Request)
}

// Filter creates a new composable filter.
func Filter(next func(w http.ResponseWriter, r *http.Request)) *Filters {
	return &Filters{next}
}

// Build returns the final chained handler
func (f *Filters) Build() func(w http.ResponseWriter, r *http.Request) {
	return f.next
}
