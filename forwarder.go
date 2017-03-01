package ginfluentd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Log formatting type to be forwarded into channel
type Log struct {
	context               *gin.Context
	startDate             time.Time
	latency               time.Duration
	requestBody           string
	responseHeaders       http.Header
	responseBody          string
	responseContentLength int64
}

type LogForwardingQueue struct {
	Intake         chan Log
	dropSize       int
	retryInterval  time.Duration
	fluentdURL     string
	fluentdEnv     string
	bodyLogPolicy  int
	maxBodyLogSize int64
}

func NewLogForwardingQueue(conf FluentdLoggerConfig) (q *LogForwardingQueue) {
	return &LogForwardingQueue{
		Intake:         make(chan Log, conf.DropSize),
		retryInterval:  conf.RetryInterval,
		fluentdURL:     fmt.Sprintf("http://%s:%d/%s", conf.Host, conf.Port, conf.Tag),
		fluentdEnv:     conf.Env,
		bodyLogPolicy:  conf.BodyLogPolicy,
		maxBodyLogSize: conf.MaxBodyLogSize,
	}
}

func (q *LogForwardingQueue) formatFluentdPayload(logEntry *Log) (payload []byte, err error) {
	// Let's normalize our headers to match Kong's format as well as our Django logger's
	requestHeaders, requestHeaderSize := normalizeHeaderMap(logEntry.context.Request.Header)
	responseHeaders, responseHeaderSize := normalizeHeaderMap(logEntry.responseHeaders)

	// Let's parse the request and response objects and put that in a JSON-friendly map
	logPayload := FluentdLogLine{
		Env:           q.fluentdEnv,
		TimeStarted:   logEntry.startDate.Format("2006-01-02T15:04:05.999+0000"),
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

func (q *LogForwardingQueue) run() {
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
