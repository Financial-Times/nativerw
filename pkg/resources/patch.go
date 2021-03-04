package resources

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/db"
	"github.com/Financial-Times/nativerw/pkg/mapper"
)

func PatchContent(mongo db.DB, ts TimestampCreator) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		connection, err := mongo.Open()
		if err != nil {
			writeMessage(w, "Failed to connect to the database!", http.StatusServiceUnavailable)
			return
		}

		tid := obtainTxID(r)
		collectionID := mux.Vars(r)["collection"]
		resourceID := mux.Vars(r)["resource"]
		schemaVersion := r.Header.Get(SchemaVersionHeader)
		contentRevision := ts.CreateTimestamp()

		resource, found, err := connection.Read(collectionID, resourceID)
		if err != nil {
			msg := "Reading from mongoDB failed."
			logger.WithTransactionID(tid).WithUUID(resourceID).WithError(err).Error(msg)
			http.Error(w, fmt.Sprintf(msg+": %v", err.Error()), http.StatusInternalServerError)
			return
		}

		if !found {
			msg := fmt.Sprintf("Could not update resource, not found, collection= %v, id= %v", collectionID, resourceID)
			logger.WithTransactionID(tid).WithUUID(resourceID).Info(msg)

			w.Header().Add("Content-Type", "application/json")
			respBody, _ := json.Marshal(map[string]string{"message": msg})
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, string(respBody))
			return
		}

		contentTypeHeader := extractAttrFromHeader(r, "Content-Type", "application/octet-stream", tid, resourceID)
		inMapper, err := mapper.InMapperForContentType(contentTypeHeader)
		if err != nil {
			msg := "Unsupported content-type"
			logger.
				WithMonitoringEvent("SaveToNative", tid, contentTypeHeader).
				WithUUID(resourceID).
				WithError(err).
				Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, err), http.StatusBadRequest)
			return
		}

		originSystemIDHeader := extractAttrFromHeader(r, "Origin-System-Id", "", tid, resourceID)
		content, err := inMapper(r.Body)
		if err != nil {
			msg := "Extracting content from HTTP body failed"
			logger.
				WithMonitoringEvent("SaveToNative", tid, contentTypeHeader).
				WithUUID(resourceID).
				WithError(err).
				Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, err), http.StatusBadRequest)
			return
		}

		originalC, _ := resource.Content.(map[string]interface{})
		PatchC, _ := content.(map[string]interface{})
		patchResult := mergeContent(PatchC, originalC)
		resource.Content = patchResult

		wrappedContent := mapper.Wrap(patchResult, resourceID, contentTypeHeader, originSystemIDHeader, schemaVersion, contentRevision)
		if errWrite := connection.Write(collectionID, wrappedContent); errWrite != nil {
			msg := "Writing to mongoDB failed"
			logger.
				WithMonitoringEvent("UpdatedToNative", tid, contentTypeHeader).
				WithUUID(resourceID).
				WithError(errWrite).
				Error(msg)
			http.Error(w, fmt.Sprintf("%s\n%v\n", msg, errWrite), http.StatusInternalServerError)
			return
		}

		logMsg := fmt.Sprintf("Successfully updated, collection=%s, origin-system-id=%s, schema-version=%s",
			collectionID, originSystemIDHeader, schemaVersion)
		logger.
			WithMonitoringEvent("UpdatedToNative", tid, contentTypeHeader).
			WithUUID(resourceID).
			Info(logMsg)

		om, err := mapper.OutMapperForContentType(contentTypeHeader)
		if err != nil {
			msg := fmt.Sprintf("Unable to handle resource of type %T", resource)
			logger.WithError(err).WithTransactionID(tid).WithUUID(resourceID).Warn(msg)
			http.Error(w, msg, http.StatusNotImplemented)
			return
		}

		w.Header().Add("Content-Type", contentTypeHeader)
		w.Header().Add("Origin-System-Id", resource.OriginSystemID)
		w.Header().Add(SchemaVersionHeader, schemaVersion)
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

// Rules to modify content :
// 1- A field in order to be updated/removed must exists in both data sources (patchC, originalC):
//	1.1 Besides, in case of being updated, the field must be the same type (basic type, or slice).
//	1.2 In case of being removed, the field in PatchC must be 'nil' and must exists in originalC.
//	1.3 An empty patch data will not modify the original data stored in the DB.
// 2- New fields can also be added, whenever the new field does not exists in originalC and it is not 'nil' in PatchC.
// Note: This method always returns an object (check rules above), it never panics and does not returns any errors
// Note: slices are being treated as a single object, therefore a slice in PatchD will always overwrite an originalD's.
// Note:Whenever a delete operation takes place within a hash struct, there is a need to check whether the result hash (parent) remains empty, in that case is removed.(recursive safe)
func mergeContent(patchC, originalC map[string]interface{}) map[string]interface{} {
	res := originalC
	for key := range patchC {
		_, oExists := originalC[key]
		if oExists && compareConditions(patchC[key], originalC[key]) {
			switch patchC[key].(type) {
			case []interface{}:
				res[key] = patchC[key]
			case map[string]interface{}:
				p, _ := patchC[key].(map[string]interface{})
				o, _ := originalC[key].(map[string]interface{})
				res[key] = mergeContent(p, o)
				if emptyHash(res[key]) {
					delete(res, key)
				}
			default:
				if patchC[key] == nil {
					delete(res, key)
				} else if patchC[key] != res[key] {
					res[key] = patchC[key]
				}
			}
		} else if !oExists && patchC[key] != nil {
			res[key] = patchC[key]
		}
	}
	return res
}

func compareConditions(p, o interface{}) bool {
	return reflect.TypeOf(p) == nil || reflect.TypeOf(p) == reflect.TypeOf(o)
}

func emptyHash(v interface{}) bool {
	m, isMap := v.(map[string]interface{})
	return isMap && len(m) == 0
}
