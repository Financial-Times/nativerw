package config

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
)

func TestConfigFromReader(t *testing.T) {
	randomness := uuid.NewUUID().String()
	reader := strings.NewReader(`{
         "mongos": "` + randomness + `",
         "server": {
            "port": 8080
         },
         "dbName": "native-store",
         "collections": [
            "video",
            "universal-content",
            "wordpress",
            "v1-metadata"
         ]
      }`)
	config, err := ReadConfigFromReader(reader)

	assert.NoError(t, err)
	assert.Equal(t, randomness, config.Mongos)
	assert.Equal(t, "native-store", config.DBName)
	assert.Equal(t, []string{"video", "universal-content", "wordpress", "v1-metadata"}, config.Collections)
	assert.Equal(t, 8080, config.Server.Port)
}

func TestConfigFromReaderFails(t *testing.T) {
	reader := strings.NewReader(`this won't work`)
	_, err := ReadConfigFromReader(reader)

	assert.Error(t, err)
}

func TestConfigFromFile(t *testing.T) {
	randomness := uuid.NewUUID().String()

	file, err := ioutil.TempFile("", "test.config.json")
	defer os.Remove(file.Name())

	assert.NoError(t, err)
	_, err = file.Write([]byte(`{
         "mongos": "` + randomness + `",
         "server": {
            "port": 8080
         },
         "dbName": "native-store",
         "collections": [
            "video",
            "universal-content",
            "wordpress",
            "v1-metadata"
         ]
      }`))
	assert.NoError(t, err)

	config, err := ReadConfig(file.Name())

	assert.NoError(t, err)
	assert.Equal(t, randomness, config.Mongos)
	assert.Equal(t, "native-store", config.DBName)
	assert.Equal(t, []string{"video", "universal-content", "wordpress", "v1-metadata"}, config.Collections)
	assert.Equal(t, 8080, config.Server.Port)
}
