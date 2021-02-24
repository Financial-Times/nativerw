package resources

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/db"
	"github.com/Financial-Times/nativerw/pkg/mapper"
)

// WriteContent writes a new native record
func WriteContent(mongo db.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		connection, err := mongo.Open()
		if err != nil {
			writeMessage(w, "Failed to connect to the database!", http.StatusServiceUnavailable)
			return
		}

		collectionID := mux.Vars(r)["collection"]
		resourceID := mux.Vars(r)["resource"]
		tid := obtainTxID(r)

		schemaVersion := r.Header.Get(schemaVerisonHeader)

		contentType := extractAttrFromHeader(r, "Content-Type", "application/octet-stream", tid, resourceID)
		inMapper, err := mapper.InMapperForContentType(contentType)
		if err != nil {
			msg := "Unsupported content-type"
			logger.WithMonitoringEvent("SaveToNative", tid, contentType).WithUUID(resourceID).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, err), http.StatusBadRequest)
			return
		}

		originSystemIDHeader := extractAttrFromHeader(r, "Origin-System-Id", "", tid, resourceID)
		content, err := inMapper(r.Body)
		if err != nil {
			msg := "Extracting content from HTTP body failed"
			logger.WithMonitoringEvent("SaveToNative", tid, contentType).WithUUID(resourceID).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, err), http.StatusBadRequest)
			return
		}

		wrappedContent := mapper.Wrap(content, resourceID, contentType, originSystemIDHeader, schemaVersion)

		if err := connection.Write(collectionID, wrappedContent); err != nil {
			msg := "Writing to mongoDB failed"
			logger.WithMonitoringEvent("SaveToNative", tid, contentType).WithUUID(resourceID).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, err), http.StatusInternalServerError)
			return
		}

		logger.WithMonitoringEvent("SaveToNative", tid, contentType).WithUUID(resourceID).Info(
			fmt.Sprintf("Successfully saved, collection=%s, origin-system-id=%s schema-version=%s", collectionID, originSystemIDHeader, schemaVersion))
	}
}
