package ginhttplogger

import (
	"sync"
)

// Let's mock the LogForwardingQueue to be able to read what's logged
// Due to its minmalistic synchronization primitives, you'll have to
// re-instantiate a queue for each request you want to test
type MockedLogForwardingQueue struct {
	Intake       chan Log
	lastLogEntry Log
	mutex        sync.Mutex
}

func NewMockedLogForwardingQueue(conf FluentdLoggerConfig) (q *MockedLogForwardingQueue) {
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
