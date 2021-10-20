package mapper

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	articleCt      = "application/vnd.ft-upp-article+json; version=1.0; charset=utf-8"
	articlePlainCt = "application/json; version=1.0; charset=utf-8"

	octetStreamCt = "application/octet-stream; version=1.0"
	textPlainCt   = "text/plain; charset=iso-8859-1"
)

func TestWrap(t *testing.T) {
	var tests = []struct {
		resource         map[string]interface{}
		uuid             string
		contentType      string
		originSystemID   string
		publishReference string
		schemaVersion    string
		contentRevision  int64
		wantResource     Resource
	}{
		{
			map[string]interface{}{
				"title":  "Title",
				"body":   "This is a body.",
				"brands": []string{"Lex", "Markets"},
			},
			"9694733e-163a-4393-801f-000ab7de5041",
			"application/json",
			"universal-content",
			"tid_blahblahblah",
			"3",
			1436773875771421417,
			Resource{
				UUID: "9694733e-163a-4393-801f-000ab7de5041",
				Content: map[string]interface{}{
					"title":  "Title",
					"body":   "This is a body.",
					"brands": []string{"Lex", "Markets"},
				},
				ContentType:     "application/json",
				OriginSystemID:  "universal-content",
				SchemaVersion:   "3",
				ContentRevision: 1436773875771421417,
			},
		},
	}

	for _, test := range tests {
		result := Wrap(test.resource, test.uuid, test.contentType, test.originSystemID, test.schemaVersion, test.contentRevision)
		if !reflect.DeepEqual(*result, test.wantResource) {
			t.Errorf("Resource: %v\n, Expected: %v\n, Actual: %v", test.resource, test.wantResource, result)
		}
	}
}

func TestJsonMappers(t *testing.T) {
	testResource := &Resource{
		UUID: "9694733e-163a-4393-801f-000ab7de5041",
		Content: map[string]interface{}{
			"title":  "Title",
			"body":   "This is a body.",
			"brands": []interface{}{"Lex", "Markets"},
		},
		ContentType:    "application/json",
		OriginSystemID: "universal-content",
	}

	mockBody := &MockBody{Body: strings.NewReader(`{"body":"This is a body.","brands":["Lex","Markets"],"title":"Title"}`)}
	mockBody.On("Close").Return(nil)
	mockBody.On("Read").Return(nil)

	var writer = bytes.NewBuffer([]byte{})

	outMapper, _ := OutMapperForContentType("application/json; someArbitrary=directive")
	err := outMapper(writer, testResource)

	assert.NoError(t, err, "Shouldn't error")
	assert.Equal(t, `{"body":"This is a body.","brands":["Lex","Markets"],"title":"Title"}`, strings.TrimSpace(writer.String()), "Json should match")

	inMapper, err := InMapperForContentType("application/json; someArbitrary=directive")
	assert.NoError(t, err)

	actual, err := inMapper(mockBody)
	assert.NoError(t, err)

	for k, v := range actual.(map[string]interface{}) {
		assert.EqualValues(t, testResource.Content.(map[string]interface{})[k], v)
	}

	mockBody.AssertExpectations(t)
}

func TestBinaryMappers(t *testing.T) {
	testResource := &Resource{
		UUID:        "9694733e-163a-4393-801f-000ab7de5041",
		Content:     []byte("hi"),
		ContentType: "application/json",
	}

	var writer = bytes.NewBuffer([]byte{})

	outMapper, _ := OutMapperForContentType("application/octet-stream; someArbitrary=directive")
	err := outMapper(writer, testResource)

	assert.NoError(t, err, "Shouldn't error")
	assert.Equal(t, `hi`, strings.TrimSpace(writer.String()))

	inMapper, err := InMapperForContentType("application/octet-stream; someArbitrary=directive")
	assert.NoError(t, err)

	actual, err := inMapper(ioutil.NopCloser(strings.NewReader(`hi`)))
	assert.NoError(t, err)

	assert.Equal(t, testResource.Content, actual)
}

func TestApplicationJsonVariantEval(t *testing.T) {
	assert.True(t, isApplicationJSONVariantWithDirectives(articleCt))
	assert.True(t, isApplicationJSONVariantWithDirectives(articlePlainCt))

	assert.False(t, isApplicationJSONVariantWithDirectives(octetStreamCt))
	assert.False(t, isApplicationJSONVariantWithDirectives(textPlainCt))
}

func TestOctetStreamVariantEval(t *testing.T) {
	assert.True(t, isOctetStreamWithDirectives(octetStreamCt))

	assert.False(t, isOctetStreamWithDirectives(articlePlainCt))
}
