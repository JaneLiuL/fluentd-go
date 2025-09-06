package main

import (
	"time"
)

// Event 表示一个日志事件
type Event struct {
	Tag       string
	Timestamp time.Time
	Record    map[string]interface{}
}

// NewEvent 创建一个新的事件
func NewEvent(tag string, record map[string]interface{}) *Event {
	return &Event{
		Tag:       tag,
		Timestamp: time.Now(),
		Record:    record,
	}
}
