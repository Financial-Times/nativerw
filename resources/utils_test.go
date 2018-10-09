package resources

import (
	"net/http"
	"testing"

	"strings"

	"github.com/stretchr/testify/assert"
)

func TestObtainTxID(t *testing.T) {
	req, _ := http.NewRequest("GET", "/doesnt/matter", nil)
	req.Header.Add("X-Request-Id", "tid_blahblah")
	txid := obtainTxID(req)
	assert.Equal(t, "tid_blahblah", txid)
}

func TestObtainTxIDGeneratesANewOneIfNoneAvailable(t *testing.T) {
	req, _ := http.NewRequest("GET", "/doesnt/matter", nil)
	txid := obtainTxID(req)
	assert.Contains(t, txid, "tid_")
}

func TestExtractContentTypeHeaderReturnsOctetStreamIfMissing(t *testing.T) {
	req, _ := http.NewRequest("PUT", "/", strings.NewReader(`{}`))
	contentTypeHeader := extractAttrFromHeader(req, "Content-Type", "application/octet-stream", "", "")
	assert.Equal(t, "application/octet-stream", contentTypeHeader)
}
func TestExtractContentTypeHeaderReturnsContentType(t *testing.T) {
	req, _ := http.NewRequest("PUT", "/", strings.NewReader(`{}`))
	req.Header.Add("Content-Type", "application/a-fake-type")

	contentTypeHeader := extractAttrFromHeader(req, "Content-Type", "application/a-fake-type", "", "")
	assert.Equal(t, "application/a-fake-type", contentTypeHeader)
}
