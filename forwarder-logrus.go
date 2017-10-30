package ginhttplogger

import (
	"encoding/json"
	"time"

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

		// Let's convert our fields to their JSON counterparts before logging fields as JSON
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			logrus.Errorf("Impossible to marshal log payload to JSON: %v (payload: %v)", err, payload)
			continue
		}
		var payloadJSON map[string]interface{}
		err = json.Unmarshal(payloadBytes, &payloadJSON)
		if err != nil {
			logrus.Errorf("Impossible to unmarshal log payload into map[string]interface{}: %v (payload: %s)", err, payloadBytes)
			continue
		}

		// Let's forward the log line to fluentd
		logger := q.logrusLogger.WithFields(payloadJSON)
		if payload.Response.Status >= 500 {
			logger.Error("server error")
		} else if payload.Response.Status >= 400 {
			logger.Warn("client error")
		} else {
			logger.Info("request processed")
		}
	}
}
