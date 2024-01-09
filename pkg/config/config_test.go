package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigFromReader(t *testing.T) {
	reader := strings.NewReader(`{
         "server": {
            "port": 8080
         },
         "dbName": "native-store",
         "collections": [
            "video",
            "universal-content",
            "pac-metadata",
			"manual-metadata"
         ]
      }`)
	config, err := ReadConfigFromReader(reader)

	assert.NoError(t, err)
	assert.Equal(t, "native-store", config.DBName)
	assert.Equal(t, []string{"video", "universal-content", "pac-metadata", "manual-metadata"}, config.Collections)
	assert.Equal(t, 8080, config.Server.Port)
}

func TestConfigFromReaderFails(t *testing.T) {
	reader := strings.NewReader(`this won't work`)
	_, err := ReadConfigFromReader(reader)

	assert.Error(t, err)
}

func TestConfigFromFile(t *testing.T) {
	file, err := os.CreateTemp("", "test.config.json")
	defer os.Remove(file.Name())

	assert.NoError(t, err)
	_, err = file.Write([]byte(`{
         "server": {
            "port": 8080
         },
         "dbName": "native-store",
         "collections": [
            "video",
            "universal-content",
            "pac-metadata",
			"manual-metadata"
         ]
      }`))
	assert.NoError(t, err)

	config, err := ReadConfig(file.Name())

	assert.NoError(t, err)
	assert.Equal(t, "native-store", config.DBName)
	assert.Equal(t, []string{"video", "universal-content", "pac-metadata", "manual-metadata"}, config.Collections)
	assert.Equal(t, 8080, config.Server.Port)
}
