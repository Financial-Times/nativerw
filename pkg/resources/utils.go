package resources

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/Financial-Times/go-logger"
)

const (
	txHeaderKey    = "X-Request-Id"
	txHeaderLength = 20
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func writeMessage(w http.ResponseWriter, msg string, status int) {
	data, _ := json.Marshal(struct {
		Message string `json:"message"`
	}{msg})

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(data); err != nil {
		logger.WithError(err).Error("could not build response JSON body")
	}
}

func obtainTxID(req *http.Request) string {
	txID := req.Header.Get(txHeaderKey)
	if txID == "" {
		return "tid_" + randSeq(txHeaderLength)
	}
	return txID
}

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] //nolint:gosec
	}
	return string(b)
}

func extractAttrFromHeader(r *http.Request, attrName, defValue, tid, resourceID string) string {
	val := r.Header.Get(attrName)

	if val == "" {
		msg := fmt.Sprintf("%s header missing. Default value ('%s') is used.", attrName, defValue)
		logger.WithTransactionID(tid).WithUUID(resourceID).Warn(msg)
		return defValue
	}

	return val
}
