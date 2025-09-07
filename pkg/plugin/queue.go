package plugin

import "sync"

// Queue 用于在组件间传递事件的队列
type Queue struct {
	ch       chan *Event
	capacity int
	mu       sync.Mutex
	closed   bool
}

func NewQueue(capacity int) *Queue {
	return &Queue{
		ch:       make(chan *Event, capacity),
		capacity: capacity,
		closed:   false,
	}
}

func (q *Queue) Put(event *Event) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return false
	}

	select {
	case q.ch <- event:
		return true
	default:
		// 队列已满，返回失败
		return false
	}
}

func (q *Queue) Get() (*Event, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil, false
	}

	select {
	case event := <-q.ch:
		return event, true
	default:
		return nil, false
	}
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		close(q.ch)
		q.closed = true
	}
}

func (q *Queue) Len() int {
	return len(q.ch)
}
