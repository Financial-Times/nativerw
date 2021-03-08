package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"github.com/Financial-Times/go-logger"
	"github.com/Financial-Times/nativerw/pkg/db"
	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
)

// Hash hashes the given payload in SHA224 + Hex
func Hash(payload string) string {
	hash := sha256.New224()
	_, err := hash.Write([]byte(payload))
	if err != nil {
		logger.WithError(err).Warn("Failed to write hash")
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// CheckNativeHash will check for the X-Native-Hash header and compare it to the current saved copy of the same resource
func (f *Filters) CheckNativeHash(mongo db.DB) *Filters {
	next := f.next

	f.next = func(w http.ResponseWriter, r *http.Request) {
		connection, err := mongo.Open()
		if err != nil {
			defer r.Body.Close()
			writeMessage(w, "Failed to connect to the database!", http.StatusServiceUnavailable)
			return
		}

		nativeHash := r.Header.Get("X-Native-Hash")

		if strings.TrimSpace(nativeHash) != "" {
			defer r.Body.Close()

			tid := transactionidutils.GetTransactionIDFromRequest(r)
			vars := mux.Vars(r)
			matches, err := checkNativeHash(connection, nativeHash, vars["collection"], vars["resource"])

			if err != nil {
				msg := "Unexpected error occurred while checking the native hash"
				logger.WithTransactionID(tid).WithError(err).Error(msg)
				http.Error(w, fmt.Sprintf(msg+" : %v", err.Error()), http.StatusServiceUnavailable)
				return
			}

			if !matches {
				logger.WithTransactionID(tid).Warn("The native hash provided with this request does not match the native content in the store, or the original has been removed!")
				http.Error(w, "The native hash provided with this request does not match the native content in the store.", http.StatusConflict)
				return
			}

			writeMessage(w, "Hash matches existing content, no need to overwrite existing native data.", http.StatusOK)
			return
		}

		next(w, r)
	}

	return f
}

func checkNativeHash(mongo db.Connection, hash string, collection string, id string) (bool, error) {
	resource, found, err := mongo.Read(collection, id)
	if err != nil {
		return false, err
	}

	if !found {
		msg := fmt.Sprintf("Received a carousel publish but the original native content does not exist in the native store! collection=%s" + collection)
		logger.WithTransactionID("").WithUUID(id).Warn(msg)
		return false, nil // no native document for this id, so save it
	}

	data, err := json.Marshal(resource.Content)
	if err != nil {
		return false, err
	}

	existingHash := Hash(string(data))
	return existingHash == hash, nil
}
