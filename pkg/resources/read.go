package resources

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/db"
	"github.com/Financial-Times/nativerw/pkg/mapper"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
)

// ReadContent reads the native data for the given id and collection
func ReadContent(connection db.Connection) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

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
		w.Header().Add(ContentRevisionHeader, strconv.FormatInt(resource.ContentRevision, 10))

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

// ReadSingleRevision reads the native data for the given id/collection/revision
func ReadSingleRevision(connection db.Connection) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		tid := transactionidutils.GetTransactionIDFromRequest(r)
		vars := mux.Vars(r)
		uuid := vars["resource"]
		collection := vars["collection"]

		contentRevisionStr := vars["revision"]
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

		resource, err := connection.ReadSingleRevision(collection, uuid, revision)
		if err != nil {
			msg := "Reading from mongoDB failed."
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf(msg+": %v", err.Error()), http.StatusInternalServerError)
			return
		}

		if resource == nil {
			msg := fmt.Sprintf("Resource not found, collection=%v, id=%v, revision=%v", collection, uuid, revision)
			logger.WithTransactionID(tid).WithUUID(uuid).Info(msg)

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
		w.Header().Add(ContentRevisionHeader, strconv.FormatInt(resource.ContentRevision, 10))

		om, err := mapper.OutMapperForContentType(contentTypeHeader)
		if err != nil {
			msg := fmt.Sprintf("Unable to handle resource of type %T", resource)
			logger.WithError(err).WithTransactionID(tid).WithUUID(uuid).Warn(msg)
			http.Error(w, msg, http.StatusNotImplemented)
			return
		}

		err = om(w, resource)
		if err != nil {
			msg := fmt.Sprintf("Unable to extract native content from resource with id %v. %v", uuid, err.Error())
			logger.WithTransactionID(tid).WithUUID(uuid).WithError(err).Errorf(msg)
			http.Error(w, msg, http.StatusInternalServerError)
		} else {
			logger.WithTransactionID(tid).WithUUID(uuid).Info("Read native content successfully")
		}
	}
}

// ReadRevisions returns a list with all the revisions for an uuid
func ReadRevisions(connection db.Connection) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		tid := transactionidutils.GetTransactionIDFromRequest(r)
		vars := mux.Vars(r)
		resourceID := vars["resource"]
		collection := vars["collection"]

		revisions, err := connection.ReadRevisions(collection, resourceID)
		if err != nil {
			msg := "Reading from mongoDB failed."
			logger.WithTransactionID(tid).WithUUID(resourceID).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf(msg+": %v", err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")

		if len(revisions) == 0 {
			msg := fmt.Sprintf("Resource not found, collection=%v, id=%v", collection, resourceID)
			logger.WithTransactionID(tid).WithUUID(resourceID).Info(msg)

			respBody, _ := json.Marshal(map[string]string{"message": msg})
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, string(respBody))
			return
		}

		respBody, err := json.Marshal(revisions)
		if err != nil {
			msg := "Unable to serialize revisions."
			logger.WithTransactionID(tid).WithUUID(resourceID).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf(msg+": %v", err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, string(respBody))
	}
}

func ReadIDs(connection db.Connection) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Entering ReadIDs")

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
