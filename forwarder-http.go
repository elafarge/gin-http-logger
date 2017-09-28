package ginhttplogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// LogForwardingQueue is a generic interface that forwards logs to a given output stream (stdout,
// http...)
type LogForwardingQueue interface {
	run() // Log forwarding goroutine
	intake() chan Log
}

// HTTPLogForwardingQueue forwards logs to an HTTP backend
type HTTPLogForwardingQueue struct {
	Intake        chan Log
	retryInterval time.Duration
	URL           string
}

// NewHTTPLogForwardingQueue builds a log forwarding queue that sends entries to an HTTP service,
// formatted as JSON
func NewHTTPLogForwardingQueue(conf AccessLoggerConfig) (q *HTTPLogForwardingQueue) {
	return &HTTPLogForwardingQueue{
		Intake:        make(chan Log, conf.DropSize),
		retryInterval: conf.RetryInterval,
		URL:           fmt.Sprintf("http://%s:%d%s", conf.Host, conf.Port, conf.Path),
	}
}

func (q *HTTPLogForwardingQueue) intake() chan Log {
	return q.Intake
}

func (q *HTTPLogForwardingQueue) run() {
	// Forwards payloads asynchronously
	for logEntry := range q.Intake {
		payload := buildPayload(&logEntry)

		// Let's forward the log line to the http server
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			log.Println("[ERROR][fluentd-middleware] Failed to Marshal payload")
			continue
		}

		if _, err := http.Post(q.URL, "application/json", bytes.NewBuffer(payloadBytes)); err != nil {
			log.Println("[WARNING][fluentd-middleware] Impossible to forward request log to fluentd:", err)
			time.Sleep(q.retryInterval)
			q.Intake <- logEntry
		}
	}
}
