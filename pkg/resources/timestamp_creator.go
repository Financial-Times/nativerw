package resources

import "time"

type TimestampCreator interface {
	CreateTimestamp() int64
}

type CurrentTimestampCreator struct{}

func (tc *CurrentTimestampCreator) CreateTimestamp() int64 {
	return time.Now().UTC().UnixNano()
}
