package resources

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/db"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
)

// PurgeContent deletes the given resource from the given collection
func PurgeContent(connection db.Connection) func(writer http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		tid := transactionidutils.GetTransactionIDFromRequest(r)

		collectionID := mux.Vars(r)["collection"]
		uuid := mux.Vars(r)["resource"]

		contentRevisionStr := mux.Vars(r)["revision"]
		if contentRevisionStr == "" {
			writeMessage(w, "Content revision is missing!", http.StatusBadRequest)
			return
		}
		revision, err := strconv.ParseInt(contentRevisionStr, 10, 64)
		if err != nil {
			msg := "Invalid content-revision"
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, err), http.StatusBadRequest)
			return
		}

		contentTypeHeader := extractAttrFromHeader(r, "Content-Type", "application/json", tid, uuid)

		if err := connection.Delete(collectionID, uuid, revision); err != nil {
			msg := "Deleting from mongoDB failed"
			logger.WithMonitoringEvent("SaveToNative", tid, contentTypeHeader).WithUUID(uuid).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, err), http.StatusInternalServerError)
			return
		}

		logger.WithMonitoringEvent("SaveToNative", tid, contentTypeHeader).WithUUID(uuid).Info("Successfully deleted")
	}
}
