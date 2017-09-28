package ginhttplogger

import (
	"time"

	"github.com/fatih/structs"
	"github.com/sirupsen/logrus"
)

// LogrusLogForwardingQueue simply forwards logs to Logrus, with the specified log level
type LogrusLogForwardingQueue struct {
	Intake        chan Log
	logrusLogger  *logrus.Logger
	retryInterval time.Duration
}

// NewLogrusLogForwardingQueue returns a such a forwarding queue
func NewLogrusLogForwardingQueue(conf AccessLoggerConfig) (q *LogrusLogForwardingQueue) {
	return &LogrusLogForwardingQueue{
		Intake:       make(chan Log, conf.DropSize),
		logrusLogger: conf.LogrusLogger,
	}
}

func (q *LogrusLogForwardingQueue) intake() chan Log {
	return q.Intake
}

func (q *LogrusLogForwardingQueue) run() {
	// Forwards payloads asynchronously
	for {
		logEntry := (<-q.Intake)
		payload := buildPayload(&logEntry)

		// Let's forward the log line to fluentd
		// TODO: change log level depending on the error. I like
		// TODO: paste our request as logrus fields
		// 2xx: debug, 3xx & 4xx: info, 5xx: error
		q.logrusLogger.WithFields(structs.Map(payload)).Info("request processed")
	}
}
