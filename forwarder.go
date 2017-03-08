package ginhttplogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Generic Interface for a queue
type LogForwardingQueue interface {
	run() // Log forwarding goroutine
	intake() chan Log
}

// Definition of such a Queue that forwards logs in JSON over HTTP
type HttpLogForwardingQueue struct {
	Intake        chan Log
	retryInterval time.Duration
	fluentdURL    string
	fluentdEnv    string
}

func NewHttpLogForwardingQueue(conf FluentdLoggerConfig) (q *HttpLogForwardingQueue) {
	return &HttpLogForwardingQueue{
		Intake:        make(chan Log, conf.DropSize),
		retryInterval: conf.RetryInterval,
		fluentdURL:    fmt.Sprintf("http://%s:%d/%s", conf.Host, conf.Port, conf.Tag),
		fluentdEnv:    conf.Env,
	}
}

func (q *HttpLogForwardingQueue) intake() chan Log {
	return q.Intake
}

func (q *HttpLogForwardingQueue) formatFluentdPayload(logEntry *Log) (payload []byte, err error) {
	// Let's normalize our headers to match Kong's format as well as our Django logger's
	requestHeaders, requestHeaderSize := normalizeHeaderMap(logEntry.context.Request.Header)
	responseHeaders, responseHeaderSize := normalizeHeaderMap(logEntry.responseHeaders)

	// Let's parse the request and response objects and put that in a JSON-friendly map
	logPayload := FluentdLogLine{
		Env:           q.fluentdEnv,
		TimeStarted:   logEntry.startDate.Format("2006-01-02T15:04:05.999+0100"),
		ClientAddress: logEntry.context.ClientIP(),
		Time:          int64(logEntry.latency.Nanoseconds() / 1000),
		Request: RequestLogEntry{
			Method:      logEntry.context.Request.Method,
			Path:        logEntry.context.Request.URL.Path,
			HTTPVersion: logEntry.context.Request.Proto,
			Headers:     requestHeaders,
			HeaderSize:  requestHeaderSize,
			Content: HttpContent{
				Size:     logEntry.context.Request.ContentLength,
				MimeType: logEntry.context.ContentType(),
				Content:  logEntry.requestBody,
			},
		},
		Errors: logEntry.context.Errors.String(),
		Response: ResponseLogEntry{
			Status:     logEntry.context.Writer.Status(),
			Headers:    responseHeaders,
			HeaderSize: int(responseHeaderSize),
			Content: HttpContent{
				Size:     logEntry.responseContentLength,
				MimeType: responseHeaders["content_type"],
				Content:  logEntry.responseBody,
			},
		},
	}

	return json.Marshal(logPayload)
}

func (q *HttpLogForwardingQueue) run() {
	// Forwards payloads asynchronously
	for {
		logEntry := (<-q.Intake)
		payload, err := q.formatFluentdPayload(&logEntry)

		if err != nil {
			log.Println("[ERROR][fluentd-middleware] Failed to format payload")
			continue
		}

		// Let's forward the log line to fluentd
		if _, err := http.Post(q.fluentdURL, "application/json", bytes.NewBuffer(payload)); err != nil {
			log.Println("[WARNING][fluentd-middleware] Impossible to forward request log to fluentd:", err)
			time.Sleep(q.retryInterval)
			q.Intake <- logEntry
		}
	}
}
