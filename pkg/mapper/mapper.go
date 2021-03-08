package mapper

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"strings"
)

var (
	ErrUnsupportedContentType = errors.New("unsupported content-type, no mapping implementation")
)

// Resource is the representation of a native resource
type Resource struct {
	UUID            string
	Content         interface{}
	ContentType     string
	OriginSystemID  string
	SchemaVersion   string
	ContentRevision int64
}

// Wrap creates a new resource
func Wrap(content interface{}, resourceID, contentType, originSystemID, schemaVersion string, contentRevision int64) *Resource {
	return &Resource{
		UUID:            resourceID,
		Content:         content,
		ContentType:     contentType,
		OriginSystemID:  originSystemID,
		SchemaVersion:   schemaVersion,
		ContentRevision: contentRevision,
	}
}

// OutMapper writes a resource in the required content format
type OutMapper func(io.Writer, *Resource) error

func OutMapperForContentType(contentType string) (OutMapper, error) {
	if isApplicationJSONVariantWithDirectives(contentType) {
		return jsonVariantOutMapper, nil
	}

	if isOctetStreamWithDirectives(contentType) {
		return octetStreamOutMapper, nil
	}

	return nil, ErrUnsupportedContentType
}

func jsonVariantOutMapper(w io.Writer, resource *Resource) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(resource.Content)
}

func octetStreamOutMapper(w io.Writer, resource *Resource) error {
	data := resource.Content.([]byte)
	_, err := io.Copy(w, bytes.NewReader(data))
	return err
}

// InMapper marshals the transport format into a resource
type InMapper func(io.ReadCloser) (interface{}, error)

// InMapperForContentType checks the content type if it's a json variant
// and returns an InMapper. Default mapper for non json variants is an octet stream mapper.
func InMapperForContentType(contentType string) (InMapper, error) {
	if isApplicationJSONVariantWithDirectives(contentType) {
		return jsonVariantInMapper, nil
	}

	if isOctetStreamWithDirectives(contentType) {
		return octetStreamInMapper, nil
	}

	return nil, ErrUnsupportedContentType
}

func jsonVariantInMapper(r io.ReadCloser) (interface{}, error) {
	var c map[string]interface{}
	defer r.Close()
	err := json.NewDecoder(r).Decode(&c)
	return c, err
}

func octetStreamInMapper(r io.ReadCloser) (interface{}, error) {
	defer r.Close()
	return ioutil.ReadAll(r)
}

func isApplicationJSONVariantWithDirectives(contentType string) bool {
	contentType = stripDirectives(contentType)

	if contentType == "application/json" {
		return true
	}

	if strings.HasPrefix(contentType, "application/") &&
		strings.HasSuffix(contentType, "+json") {
		return true
	}

	return false
}

func isOctetStreamWithDirectives(contentType string) bool {
	return stripDirectives(contentType) == "application/octet-stream"
}

func stripDirectives(contentType string) string {
	if strings.Contains(contentType, ";") {
		contentType = strings.Split(contentType, ";")[0]
	}
	return contentType
}
