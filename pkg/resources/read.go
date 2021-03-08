package resources

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/db"
	"github.com/Financial-Times/nativerw/pkg/mapper"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
)

// ReadContent reads the native data for the given id and collection
func ReadContent(mongo db.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		connection, err := mongo.Open()
		if err != nil {
			writeMessage(w, "Failed to connect to the database!", http.StatusServiceUnavailable)
			return
		}

		tid := transactionidutils.GetTransactionIDFromRequest(r)
		vars := mux.Vars(r)
		resourceID := vars["resource"]
		collection := vars["collection"]

		resource, found, err := connection.Read(collection, resourceID)
		if err != nil {
			msg := "Reading from mongoDB failed."
			logger.WithTransactionID(tid).WithUUID(resourceID).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf(msg+": %v", err.Error()), http.StatusInternalServerError)
			return
		}

		if !found {
			msg := fmt.Sprintf("Resource not found, collection= %v, id= %v", collection, resourceID)
			logger.WithTransactionID(tid).WithUUID(resourceID).Info(msg)

			w.Header().Add("Content-Type", "application/json")
			respBody, _ := json.Marshal(map[string]string{"message": msg})
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, string(respBody))
			return
		}

		contentTypeHeader := resource.ContentType
		w.Header().Add("Content-Type", contentTypeHeader)
		w.Header().Add("Origin-System-Id", resource.OriginSystemID)
		w.Header().Add(SchemaVersionHeader, resource.SchemaVersion)
		w.Header().Add(ContentRevisionHeader, resource.ContentRevision)

		om, err := mapper.OutMapperForContentType(contentTypeHeader)
		if err != nil {
			msg := fmt.Sprintf("Unable to handle resource of type %T", resource)
			logger.WithError(err).WithTransactionID(tid).WithUUID(resourceID).Warn(msg)
			http.Error(w, msg, http.StatusNotImplemented)
			return
		}

		err = om(w, resource)
		if err != nil {
			msg := fmt.Sprintf("Unable to extract native content from resource with id %v. %v", resourceID, err.Error())
			logger.WithTransactionID(tid).WithUUID(resourceID).WithError(err).Errorf(msg)
			http.Error(w, msg, http.StatusInternalServerError)
		} else {
			logger.WithTransactionID(tid).WithUUID(resourceID).Info("Read native content successfully")
		}
	}
}

func ReadIDs(mongo db.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		connection, err := mongo.Open()
		if err != nil {
			writeMessage(w, "Failed to connect to the database!", http.StatusServiceUnavailable)
			return
		}

		vars := mux.Vars(r)
		coll := vars["collection"]
		tid := transactionidutils.GetTransactionIDFromRequest(r)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ids, err := connection.ReadIDs(ctx, coll)
		if err != nil {
			msg := fmt.Sprintf(`Failed to read IDs from mongo for %v! "%v"`, coll, err.Error())
			logger.WithTransactionID(tid).WithError(err).Error(msg)
			http.Error(w, msg, http.StatusServiceUnavailable)
			return
		}

		id := struct {
			ID string `json:"id"`
		}{}

		bw := bufio.NewWriter(w)
		for {
			docID, ok := <-ids
			if !ok {
				break
			}

			id.ID = docID
			jd, _ := json.Marshal(id)

			if _, err = bw.WriteString(string(jd) + "\n"); err != nil {
				logger.WithTransactionID(tid).WithError(err).Error("unable to write string")
			}

			bw.Flush()
			w.(http.Flusher).Flush()
		}
	}
}
