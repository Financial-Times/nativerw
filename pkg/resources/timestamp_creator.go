package resources

import "time"

type TimestampCreator interface {
	CreateTimestamp() string
}

type CurrentTimestampCreator struct{}

func (tc *CurrentTimestampCreator) CreateTimestamp() string {
	return time.Now().Format("2006-01-02T15:04:05.999Z")
}
