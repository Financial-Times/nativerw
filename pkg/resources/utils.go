package resources

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Financial-Times/go-logger"
)

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

func extractAttrFromHeader(r *http.Request, attrName, defValue, tid, resourceID string) string {
	val := r.Header.Get(attrName)

	if val == "" {
		msg := fmt.Sprintf("%s header missing. Default value ('%s') is used.", attrName, defValue)
		logger.WithTransactionID(tid).WithUUID(resourceID).Warn(msg)
		return defValue
	}

	return val
}
