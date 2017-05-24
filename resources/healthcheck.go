package resources

import (
	"net/http"

	fthealth "github.com/Financial-Times/go-fthealth"
	"github.com/Financial-Times/nativerw/db"
	"github.com/Financial-Times/nativerw/mapper"
)

const healthcheckColl = "healthcheck"

var sampleResource = mapper.Resource{
	UUID:        "cda5d6a9-cd25-4d76-8bad-9eaa35e85f4a",
	ContentType: "application/json",
	Content:     "{\"foo\": [\"a\",\"b\"], \"bar\": 10.4}",
}

const sampleUUID = "cda5d6a9-cd25-4d76-8bad-9eaa35e85f4a"

// Healthchecks is the /__health endpoint
func Healthchecks(mongo db.DB) func(w http.ResponseWriter, r *http.Request) {
	return fthealth.Handler("Dependent services healthcheck", "Checking connectivity and usability of dependent services: mongoDB.", []fthealth.Check{
		{
			BusinessImpact:   "Publishing won't work. Writing content to native store is broken.",
			Name:             "Write to mongoDB",
			PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/native-store-reader-writer-run-book",
			Severity:         2,
			TechnicalSummary: "Writing to mongoDB is broken. Check mongoDB is up, its disk space, ports, network.",
			Checker:          checkWritable(mongo),
		},
		{
			BusinessImpact:   "Reading content from native store is broken.",
			Name:             "Read from mongoDB",
			PanicGuide:       "https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/extra-publishing/native-store-reader-writer-run-book",
			Severity:         2,
			TechnicalSummary: "Reading from mongoDB is broken. Check mongoDB is up, its disk space, ports, network.",
			Checker:          checkReadable(mongo),
		},
	}...)
}

func checkWritable(mongo db.DB) func() error {
	return func() error {
		connection, err := mongo.Open()
		if err != nil {
			return err
		}

		return connection.Write(healthcheckColl, sampleResource)
	}
}

func checkReadable(mongo db.DB) func() error {
	return func() error {
		connection, err := mongo.Open()
		if err != nil {
			return err
		}

		_, _, err = connection.Read(healthcheckColl, sampleUUID)
		return err
	}
}

// GoodToGo is the /__gtg endpoint
func GoodToGo(mongo db.DB) func(w http.ResponseWriter, r *http.Request) {
	healthChecks := []func() error{checkReadable(mongo), checkWritable(mongo)}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=US-ASCII")
		w.Header().Set("Cache-Control", "no-cache")
		for _, hCheck := range healthChecks {
			if err := hCheck(); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
	}
}
