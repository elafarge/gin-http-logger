package fluentlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

//// CONFIG ////
const (
	LOG_ALL_BODIES      = 0
	LOG_BODIES_ON_ERROR = 1
	LOG_NO_BODY         = 2
)

type FluentdLoggerConfig struct {
	Host              string
	Port              int
	Env               string
	Tag               string
	DropSize          int
	MaxBodyLogSize    int
	BodyLogPolicy     int
	RetryInterval     float64
	FieldsToObfuscate []string
}

//// Log formatting types ////
type HttpContent struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	Content  string `json:"content"`
}

type RequestLogEntry struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	HTTPVersion string            `json:"http_version"`
	Headers     map[string]string `json:"headers"`
	HeaderSize  int               `json:"headers_size"`
	Content     HttpContent       `json:"content"`
}

type ResponseLogEntry struct {
	Status     int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	HeaderSize int               `json:"headers_size"`
	Content    HttpContent       `json:"content"`
}

type FluentdLogLine struct {
	Env           string           `json:"fluentd_env"`
	TimeStarted   string           `json:"time_started"`
	ClientAddress string           `json:"x_client_address"`
	Time          int64            `json:"time"`
	Request       RequestLogEntry  `json:"request"`
	Response      ResponseLogEntry `json:"response"`
}

//// Log formatting and forwarding mechanics ////
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
	retryInterval  float64
	fluentdURL     string
	fluentdEnv     string
	bodyLogPolicy  int
	maxBodyLogSize int
}

func NewLogForwardingQueue(fluentdURL string, fluentdEnv string, bodyLogPolicy int, maxBodyLogSize int, dropSize int, retryInterval float64) (q *LogForwardingQueue) {
	return &LogForwardingQueue{
		Intake:         make(chan Log, dropSize),
		retryInterval:  retryInterval,
		fluentdURL:     fluentdURL,
		fluentdEnv:     fluentdEnv,
		bodyLogPolicy:  bodyLogPolicy,
		maxBodyLogSize: maxBodyLogSize,
	}
}

func (q *LogForwardingQueue) formatFluentdPayload(logEntry *Log) (payload []byte, err error) {
	// Compute the size of request headers and flatten the header values
	requestHeaderSize := 0
	requestHeaders := make(map[string]string)
	for name, value := range logEntry.context.Request.Header {
		requestHeaderSize += len(name)
		for _, v := range value {
			requestHeaderSize += len(v)
		}
		requestHeaders[name] = strings.Join(value, ",")
	}

	responseHeaderSize := 0
	responseHeaders := make(map[string]string)
	for name, value := range logEntry.responseHeaders {
		responseHeaderSize += len(name)
		for _, v := range value {
			responseHeaderSize += len(v)
		}
		responseHeaders[name] = strings.Join(value, ",")
	}

	// Let's parse the request and response objects and put that in a JSON-friendly map
	logPayload := FluentdLogLine{
		Env:           q.fluentdEnv,
		TimeStarted:   logEntry.startDate.Format("2006-01-02T15:04:05.999999") + "Z",
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
		Response: ResponseLogEntry{
			Status:     logEntry.context.Writer.Status(),
			Headers:    responseHeaders,
			HeaderSize: int(responseHeaderSize),
			Content: HttpContent{
				Size:     logEntry.responseContentLength,
				MimeType: responseHeaders["Content-Type"],
				Content:  logEntry.responseBody,
			},
		},
	}

	payload, err = json.Marshal(logPayload)
	if err != nil {
		return nil, err
	}

	return payload, nil
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

		log.Println("Forwarding this log line to Fluentd:", q.fluentdURL)
		log.Println(string(payload))

		// Let's forward the log line to fluentd
		_, err = http.Post(q.fluentdURL, "application/json", bytes.NewBuffer(payload))
		if err != nil {
			log.Println("[WARNING][fluentd-middleware] Impossible to forward request log to fluentd:", err)
		}
	}
}

// A custom Writer to intercept the response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func New(conf FluentdLoggerConfig) gin.HandlerFunc {
	// Apply configuration
	fluentdURL := fmt.Sprintf("http://%s:%d/%s", conf.Host, conf.Port, conf.Tag)

	logQueue := NewLogForwardingQueue(fluentdURL, conf.Env, conf.BodyLogPolicy, conf.MaxBodyLogSize, conf.DropSize, conf.RetryInterval)
	go logQueue.run()

	// return the middleware function
	return func(c *gin.Context) {
		// Let's intercept the response body with our hacked writer
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// Start chrono
		startDate := time.Now()

		// Let's process the request
		c.Next()

		latency := time.Since(startDate)

		// According the the docs, we're working on a read-only copy of the context in the goroutine:
		// https://github.com/gin-gonic/gin#goroutines-inside-a-middleware
		// However, the response headers and body will be dereferenced... we'll have to keep them apart
		// (see: https://github.com/gin-gonic/gin/blob/master/context.go#L73)
		responseHeaders := make(map[string][]string)
		for name, value := range c.Writer.Header() {
			responseHeaders[name] = value
		}

		// Shall we pass the body as well ? If so let's not dereference it !
		responseBody := ""
		if conf.BodyLogPolicy == LOG_ALL_BODIES || conf.BodyLogPolicy == LOG_BODIES_ON_ERROR && c.Writer.Status() >= 400 {
			buffer := make([]byte, conf.MaxBodyLogSize)
			n, _ := blw.body.Read(buffer)
			responseBody = string(buffer[:n])
		}
		responseContentLength := c.Writer.Size()

		// We'll need to buffer the request body as well, it is destroyed when the request is forwarded
		buffer := make([]byte, 200)
		n, _ := c.Request.Body.Read(buffer)
		requestBody := string(buffer[:n])

		// Let's wrap all that into a channel-friendly struct
		logEntry := Log{
			context:               c.Copy(),
			startDate:             startDate,
			latency:               latency,
			responseHeaders:       responseHeaders,
			requestBody:           requestBody,
			responseBody:          responseBody,
			responseContentLength: int64(responseContentLength),
		}

		select {
		case logQueue.Intake <- logEntry:
		default:
			log.Println("[WARNING][fluentd-middleware] Impossible to forward requests into log queue, channel full.")
		}
	}
}
