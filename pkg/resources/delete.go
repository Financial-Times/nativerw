package resources

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/db"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
)

// DeleteContent deletes the given resource from the given collection
func DeleteContent(mongo db.DB) func(writer http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		connection, err := mongo.Open()
		if err != nil {
			writeMessage(w, "Failed to connect to the database!", http.StatusServiceUnavailable)
			return
		}

		collectionID := mux.Vars(r)["collection"]
		resourceID := mux.Vars(r)["resource"]
		tid := transactionidutils.GetTransactionIDFromRequest(r)
		contentTypeHeader := extractAttrFromHeader(r, "Content-Type", "application/octet-stream", tid, resourceID)

		if err := connection.Delete(collectionID, resourceID); err != nil {
			msg := "Deleting from mongoDB failed"
			logger.WithMonitoringEvent("SaveToNative", tid, contentTypeHeader).WithUUID(resourceID).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, err), http.StatusInternalServerError)
			return
		}

		logger.WithMonitoringEvent("SaveToNative", tid, contentTypeHeader).WithUUID(resourceID).Info("Successfully deleted")
	}
}
