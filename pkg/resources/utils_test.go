package resources

import (
	"net/http"
	"testing"

	"strings"

	transactionidutils "github.com/Financial-Times/transactionid-utils-go"
	"github.com/stretchr/testify/assert"
)

func TestObtainTxID(t *testing.T) {
	req, _ := http.NewRequest("GET", "/doesnt/matter", nil)
	req.Header.Add("X-Request-Id", "tid_blahblah")
	txid := transactionidutils.GetTransactionIDFromRequest(req)
	assert.Equal(t, "tid_blahblah", txid)
}

func TestObtainTxIDGeneratesANewOneIfNoneAvailable(t *testing.T) {
	req, _ := http.NewRequest("GET", "/doesnt/matter", nil)
	txid := transactionidutils.GetTransactionIDFromRequest(req)
	assert.Contains(t, txid, "tid_")
}

func TestExtractContentTypeHeaderReturnsApplicationJsonIfMissing(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(`{}`))
	contentTypeHeader := extractAttrFromHeader(req, "Content-Type", "application/json", "", "")
	assert.Equal(t, "application/json", contentTypeHeader)
}
func TestExtractContentTypeHeaderReturnsContentType(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(`{}`))
	req.Header.Add("Content-Type", "application/a-fake-type")

	contentTypeHeader := extractAttrFromHeader(req, "Content-Type", "application/a-fake-type", "", "")
	assert.Equal(t, "application/a-fake-type", contentTypeHeader)
}
