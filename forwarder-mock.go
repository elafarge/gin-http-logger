package ginhttplogger

import (
	"sync"
)

// MockedLogForwardingQueue is a mock class for our log forwarding queue
type MockedLogForwardingQueue struct {
	Intake       chan Log
	lastLogEntry Log
	mutex        sync.Mutex
}

// NewMockedLogForwardingQueue creates a new MockedLogForwardingQueue
func NewMockedLogForwardingQueue(conf AccessLoggerConfig) (q *MockedLogForwardingQueue) {
	q = &MockedLogForwardingQueue{
		Intake: make(chan Log, conf.DropSize),
	}
	q.mutex.Lock()
	return
}

func (q *MockedLogForwardingQueue) intake() chan Log {
	return q.Intake
}

func (q *MockedLogForwardingQueue) run() {
	q.lastLogEntry = (<-q.Intake)
	q.mutex.Unlock()
}

func (q *MockedLogForwardingQueue) pop() (logEntry Log) {
	q.mutex.Lock()
	logEntry = q.lastLogEntry
	q.mutex.Unlock()
	return
}
