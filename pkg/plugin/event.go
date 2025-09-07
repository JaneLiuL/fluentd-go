package plugin

import (
	"time"
)

// a log event
type Event struct {
	Tag       string
	Timestamp time.Time
	Record    map[string]interface{}
}

// NewEvent create a new event
func NewEvent(tag string, record map[string]interface{}) *Event {
	return &Event{
		Tag:       tag,
		Timestamp: time.Now(),
		Record:    record,
	}
}
