package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/Financial-Times/nativerw/pkg/mapper"
)

func TestPatchContent(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)
	uuid := "a-real-uuid"
	collection := "methode"
	updatedContent := map[string]interface{}{"body": "updated-data"}
	contentType := "application/json"
	httpMethod := "PATCH"
	contentRevision := "2020-11-25T21:48:05.999Z"

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", collection, uuid).Return(&mapper.Resource{ContentType: contentType, Content: map[string]interface{}{}, ContentRevision: contentRevision}, true, nil)
	connection.On("Write", collection, &mapper.Resource{UUID: uuid, Content: updatedContent, ContentType: contentType, ContentRevision: contentRevision}).Return(nil)

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", PatchContent(mongo, &ts)).Methods(httpMethod)

	w := httptest.NewRecorder()
	path := fmt.Sprintf("/%s/%s", collection, uuid)
	req, _ := http.NewRequest(httpMethod, path, strings.NewReader(`{"body": "updated-data"}`))

	req.Header.Add("Content-Type", contentType)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShouldNotUpdatePatchContentEmptyRequestBody(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)
	uuid := "a-real-uuid"
	collection := "methode"
	existingContent := map[string]interface{}{"body": "data"}
	contentType := "application/json"
	httpMethod := "PATCH"
	contentRevision := "2020-11-25T21:48:05.999Z"

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", collection, uuid).Return(&mapper.Resource{ContentType: contentType, Content: existingContent, ContentRevision: contentRevision}, true, nil)
	connection.On("Write", collection, &mapper.Resource{UUID: uuid, Content: existingContent, ContentType: contentType, ContentRevision: contentRevision}).Return(nil)

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", PatchContent(mongo, &ts)).Methods(httpMethod)

	w := httptest.NewRecorder()
	path := fmt.Sprintf("/%s/%s", collection, uuid)
	req, _ := http.NewRequest(httpMethod, path, strings.NewReader(`{}`))

	req.Header.Add("Content-Type", contentType)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPatchContentWithCharsetDirective(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)
	uuid := "a-real-uuid"
	collection := "methode"
	content := map[string]interface{}{"body": "updated-data"}
	contentType := "application/json"
	contentTypeWithCharset := "application/json; charset=utf-8"
	httpMethod := "PATCH"
	contentRevision := "2020-11-25T21:48:05.999Z"

	mongo.On("Open").Return(connection, nil)

	connection.On("Read", collection, uuid).Return(&mapper.Resource{ContentType: contentType, Content: map[string]interface{}{}, ContentRevision: contentRevision}, true, nil)
	connection.On("Write",
		collection,
		&mapper.Resource{
			UUID:            uuid,
			Content:         content,
			ContentType:     contentTypeWithCharset,
			ContentRevision: contentRevision}).
		Return(nil)

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", PatchContent(mongo, &ts)).Methods(httpMethod)

	w := httptest.NewRecorder()
	path := fmt.Sprintf("/%s/%s", collection, uuid)
	req, _ := http.NewRequest(httpMethod, path, strings.NewReader(`{"body": "updated-data"}`))

	req.Header.Add("Content-Type", contentTypeWithCharset)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPatchFailedOnWrite(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)
	uuid := "a-real-uuid"
	collection := "methode"
	content := map[string]interface{}{"body": "updated-data"}
	contentType := "application/json"
	httpMethod := "PATCH"
	contentRevision := "2020-11-25T21:48:05.999Z"

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", collection, uuid).Return(&mapper.Resource{ContentType: contentType, Content: map[string]interface{}{}, ContentRevision: contentRevision}, true, nil)
	connection.On("Write", collection, &mapper.Resource{UUID: uuid, Content: content, ContentType: contentType, ContentRevision: contentRevision}).Return(errors.New("i failed"))

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", PatchContent(mongo, &ts)).Methods(httpMethod)

	w := httptest.NewRecorder()
	path := fmt.Sprintf("/%s/%s", collection, uuid)
	req, _ := http.NewRequest(httpMethod, path, strings.NewReader(`{"body": "updated-data"}`))

	req.Header.Add("Content-Type", contentType)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPatchFailedOnRead(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)
	uuid := "a-real-uuid"
	collection := "methode"
	contentType := "application/json"
	httpMethod := "PATCH"

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", collection, uuid).Return((*mapper.Resource)(nil), false, errors.New("i failed"))

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", PatchContent(mongo, &ts)).Methods(httpMethod)

	w := httptest.NewRecorder()
	path := fmt.Sprintf("/%s/%s", collection, uuid)
	req, _ := http.NewRequest(httpMethod, path, strings.NewReader(`{"body": "updated-data"}`))

	req.Header.Add("Content-Type", contentType)

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestPatchFailedJSON(t *testing.T) {
	mongo := new(MockDB)
	connection := new(MockConnection)

	uuid := "a-real-uuid"
	collection := "methode"
	content := map[string]interface{}{"body": "data"}
	contentType := "application/json"
	httpMethod := "PATCH"

	mongo.On("Open").Return(connection, nil)
	connection.On("Read", collection, uuid).Return(&mapper.Resource{ContentType: contentType, Content: content}, true, nil)

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", PatchContent(mongo, &ts)).Methods(httpMethod)

	w := httptest.NewRecorder()
	path := fmt.Sprintf("/%s/%s", collection, uuid)
	req, _ := http.NewRequest(httpMethod, path, strings.NewReader(`i am not a json`))

	req.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFailedMongoOnPatch(t *testing.T) {
	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", PatchContent(mongo, &ts)).Methods("PATCH")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/methode/a-real-uuid", strings.NewReader(`{}`))
	req.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestFailedMongoOnWrite(t *testing.T) {
	mongo := new(MockDB)
	mongo.On("Open").Return(nil, errors.New("no data 4 u"))

	ts := fixedTimestampCreator{}

	router := mux.NewRouter()
	router.HandleFunc("/{collection}/{resource}", PatchContent(mongo, &ts)).Methods("PATCH")

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/methode/a-real-uuid", strings.NewReader(`{}`))
	req.Header.Add("Content-Type", "application/json")

	router.ServeHTTP(w, req)
	mongo.AssertExpectations(t)
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestPatchContentReflection(t *testing.T) {
	tests := []struct {
		name           string
		description    string
		originalC      string
		patchC         string
		expectedResult string
	}{
		{
			name:           "updating a simple type (int)",
			description:    "myInt should be updated to the value hold by mergeContent",
			originalC:      `{"myInt":0,"mySlice":[1,2,3]}`,
			patchC:         `{"myInt":1,"mySlice":[1,2,3]}`,
			expectedResult: `{"myInt":1,"mySlice":[1,2,3]}`,
		},
		{
			name:           "update a slice",
			description:    "In this case the value of the slice should be the one hold by mergeContent",
			originalC:      `{"myInt":0,"mySlice":[1]}`,
			patchC:         `{"myInt":1,"mySlice":[1,2,3]}`,
			expectedResult: `{"myInt":1,"mySlice":[1,2,3]}`,
		},
		{
			name:           "update a hash (recursion)",
			description:    "In this case a recursion is applied to update the values of a JSON hash (int, slice)",
			originalC:      `{"myInt":0,"mySlice":[1], "myHash":{"rInt":100, "rSlice":["a","a","a"]}}`,
			patchC:         `{"myInt":0,"mySlice":[1,2,3], "myHash":{"rInt":999, "rSlice":[9,9,9]}}`,
			expectedResult: `{"myInt":0,"mySlice":[1,2,3], "myHash":{"rInt":999, "rSlice":[9,9,9]}}`,
		},
		{
			name:           "field updates, original has content that patch does not",
			description:    "In this case mergeContent has a field that does not exist in the original content, the content that is not present should remain in the result",
			originalC:      `{"myInt":0, "myHash":{"newField":999}}`,
			patchC:         `{"myInt":999}`,
			expectedResult: `{"myInt":999, "myHash":{"newField":999}}`,
		},
		{
			name:           "field updates, original has content that patch does not (recursion)",
			description:    "In this case mergeContent has a field that does not exist in the original content, the content that is not present should remain in the result",
			originalC:      `{"myInt":0, "myHash":{"newField":999}}`,
			patchC:         `{"myInt":999}`,
			expectedResult: `{"myInt":999, "myHash":{"newField":999}}`,
		},
		{
			name:           "remove a field (simple type)",
			description:    "In this case mergeContent has a null field and it will be removed from the original content",
			originalC:      `{"myInt":0,"myRemove":999}`,
			patchC:         `{"myInt":0,"myRemove":null}`,
			expectedResult: `{"myInt":0}`,
		},
		{
			name:           "remove a field (slice)",
			description:    "In this case mergeContent has a null array field and it will be removed from the original content",
			originalC:      `{"myInt":0,"myRemove":[9,9,9]}`,
			patchC:         `{"myInt":0,"myRemove":null}`,
			expectedResult: `{"myInt":0}`,
		},
		{
			name:           "remove a field (slice) with recursion",
			description:    "In this case mergeContent has a null array field and it will be removed from the original content",
			originalC:      `{"myInt":0, "myHash":{"myRemove":999,"myInt":1}}`,
			patchC:         `{"myInt":0,"myHash":{"myRemove":null,"myInt":1}}`,
			expectedResult: `{"myInt":0,"myHash":{"myInt":1}}`,
		},
		{
			name:           "add new field",
			description:    "In this case mergeContent has a field that does not exist in the original content",
			originalC:      `{"myInt":0}`,
			patchC:         `{"myInt":0,"myNewHash":{"newField":999}}`,
			expectedResult: `{"myInt":0,"myNewHash":{"newField":999}}`,
		},
		{
			name:           "add new field (recursion)",
			description:    "In this case mergeContent has a field that does not exist in the original content",
			originalC:      `{"myInt":0, "myHash":{"myInt":1}}`,
			patchC:         `{"myHash":{"newField":999}}`,
			expectedResult: `{"myInt":0, "myHash":{"newField":999,"myInt":1}}`,
		},
		{
			name:           "add new field (Hash)",
			description:    "In this case mergeContent has a field that does not exist in the original content",
			originalC:      `{"myInt":0}`,
			patchC:         `{"myInt":0, "myHash":{"newField":999}}`,
			expectedResult: `{"myInt":0, "myHash":{"newField":999}}`,
		},
		{
			name:           "remove a field (hash) with recursion",
			description:    "In this case mergeContent has a null array field and it will be removed from the original content",
			originalC:      `{"myInt":0, "myHash":{"myRemove":999}}`,
			patchC:         `{"myInt":0,"myHash":{"myRemove":null}}`,
			expectedResult: `{"myInt":0}`,
		},
		{
			name:           "remove a field (hash) with more recursion",
			description:    "In this case mergeContent has a null array field and it will be removed from the original content",
			originalC:      `{"myInt":0,"myHash":{"myHash":{"myRemove":999}}}`,
			patchC:         `{"myInt":0,"myHash":{"myHash":{"myRemove":null}}}`,
			expectedResult: `{"myInt":0}`,
		},
		{
			name:           "the patch request is an empty json",
			description:    "In this case mergeContent is an empty json, there should be no changes in the original content",
			originalC:      `{"myInt":0,"myHash":{"myHash":999}}`,
			patchC:         `{}`,
			expectedResult: `{"myInt":0,"myHash":{"myHash":999}}`,
		},
		{
			name:           "the original content is an empty json, patch json has content",
			description:    "In this case original content is an empty json, mergeContent will add its fields",
			originalC:      `{}`,
			patchC:         `{"myInt":0,"myHash":{"myHash":999}}`,
			expectedResult: `{"myInt":0,"myHash":{"myHash":999}}`,
		},
		{
			name:           "Same field different data type",
			description:    "In this case there should be no action, original preserves its data",
			originalC:      `{"myInt":0}`,
			patchC:         `{"myInt":"str"}`,
			expectedResult: `{"myInt":0}`,
		},
	}

	for _, test := range tests {
		var patchC map[string]interface{}
		var originalC map[string]interface{}
		var expectedResult map[string]interface{}

		if err := json.Unmarshal([]byte(test.originalC), &originalC); err != nil {
			fmt.Println(err)
		}

		if err := json.Unmarshal([]byte(test.patchC), &patchC); err != nil {
			fmt.Println(err)
		}

		if err := json.Unmarshal([]byte(test.expectedResult), &expectedResult); err != nil {
			fmt.Println(err)
		}

		res := mergeContent(patchC, originalC)

		if !cmp.Equal(res, expectedResult) {
			t.Errorf("test %s: \n %s \n returned unexpected result got/want: \n %s \n %s ", test.name, test.description, res, expectedResult)
		}
	}
}
