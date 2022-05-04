package resources

import (
	"net/http"
	"time"

	fthealth "github.com/Financial-Times/go-fthealth/v1_1"
	"github.com/Financial-Times/nativerw/pkg/db"
	"github.com/Financial-Times/nativerw/pkg/mapper"
	"github.com/Financial-Times/service-status-go/gtg"
)

const healthCheckColl = "healthcheck"

var sampleResource = &mapper.Resource{
	UUID:        "cda5d6a9-cd25-4d76-8bad-9eaa35e85f4a",
	ContentType: "application/json",
	Content:     "{\"foo\": [\"a\",\"b\"], \"bar\": 10.4}",
}

const (
	sampleUUID = "cda5d6a9-cd25-4d76-8bad-9eaa35e85f4a"
	systemCode = "nativestorereaderwriter"
)

// Healthchecks is the /__health endpoint
func Healthchecks(mongo db.DB) func(w http.ResponseWriter, r *http.Request) {
	return fthealth.Handler(fthealth.TimedHealthCheck{
		HealthCheck: fthealth.HealthCheck{
			SystemCode:  systemCode,
			Name:        "nativerw",
			Description: "Reads and Writes data to the UPP Native Store, in the received (native) format",
			Checks: []fthealth.Check{
				{
					BusinessImpact:   "Publishing won't work. Writing content to native store is broken.",
					Name:             "Write to mongoDB",
					PanicGuide:       "https://dewey.in.ft.com/view/system/NativeStoreReaderWriter",
					Severity:         1,
					TechnicalSummary: "Writing to mongoDB is broken. Check mongoDB is up, its disk space, ports, network.",
					Checker:          checkWritable(mongo),
				},
				{
					BusinessImpact:   "Reading content from native store is broken.",
					Name:             "Read from mongoDB",
					PanicGuide:       "https://dewey.in.ft.com/view/system/NativeStoreReaderWriter",
					Severity:         1,
					TechnicalSummary: "Reading from mongoDB is broken. Check mongoDB is up, its disk space, ports, network.",
					Checker:          checkReadable(mongo),
				},
			},
		},
		Timeout: 10 * time.Second,
	})
}

func checkWritable(mongo db.DB) func() (string, error) {
	return func() (string, error) {
		connection, err := mongo.Open()
		if err != nil {
			return "Failed to establish connection to MongoDB", err
		}

		err = connection.Write(healthCheckColl, sampleResource)
		if err != nil {
			return "Failed to write data to MongoDB, please check the connection.", err
		}

		return "OK", nil
	}
}

func checkReadable(mongo db.DB) func() (string, error) {
	return func() (string, error) {
		connection, err := mongo.Open()
		if err != nil {
			return "Failed to establish connection to MongoDB", err
		}

		_, _, err = connection.Read(healthCheckColl, sampleUUID)
		if err != nil {
			return "Failed to read data from MongoDB, please check the connection.", err
		}

		return "OK", nil
	}
}

// GoodToGo is the /__gtg endpoint
func GoodToGo(mongo db.DB) gtg.StatusChecker {
	checks := []gtg.StatusChecker{
		newStatusChecker(checkReadable(mongo)),
		newStatusChecker(checkWritable(mongo)),
	}
	return gtg.FailFastParallelCheck(checks)
}

func newStatusChecker(check func() (string, error)) gtg.StatusChecker {
	return func() gtg.Status {
		if msg, err := check(); err != nil {
			return gtg.Status{GoodToGo: false, Message: msg}
		}
		return gtg.Status{GoodToGo: true}
	}
}
