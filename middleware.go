package ginfluentd

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

//// CONFIG ////
const (
	LOG_BODIES_ON_ERROR = 1 + iota
	LOG_NO_BODY
	LOG_ALL_BODIES
)

var NOBODYMETHODS = map[string]struct{}{
	"HEAD":    struct{}{},
	"OPTIONS": struct{}{},
	"GET":     struct{}{},
}

type FluentdLoggerConfig struct {
	Host           string
	Port           int
	Env            string
	Tag            string
	DropSize       int
	MaxBodyLogSize int64
	BodyLogPolicy  int
	RetryInterval  time.Duration
}

func New(conf FluentdLoggerConfig) gin.HandlerFunc {
	// Parse configuration, apply default arguments
	// (Host and Port are mandatory)
	if conf.BodyLogPolicy == 0 {
		conf.BodyLogPolicy = LOG_NO_BODY
	}

	if conf.Tag == "" {
		conf.Tag = "gin.requests"
	}

	if conf.DropSize == 0 {
		conf.DropSize = 1000
	}

	if conf.MaxBodyLogSize == 0 {
		conf.MaxBodyLogSize = 10000
	}

	if conf.RetryInterval == 0 {
		conf.RetryInterval = 10 * time.Second
	}

	// Apply configuration
	logQueue := NewLogForwardingQueue(conf)

	// Run the log-forwarding goroutine
	go logQueue.run()

	// return the middleware function
	return func(c *gin.Context) {
		var requestBody, responseBody string
		var responseBodyLeech *LeechedGinResponseWriter
		var requestBodyLeech *LeechedReadCloser

		if conf.BodyLogPolicy != LOG_NO_BODY {
			// Let's use a Leech to pump a limited amount of bytes on the request
			// body into RAM as this body is read
			bodySize := min(c.Request.ContentLength, conf.MaxBodyLogSize)

			// If the Content-Length header ain't set let's use a buffer of
			// MaxBodyLogSize to log the request body.
			if _, ok := NOBODYMETHODS[c.Request.Method]; !ok && c.Request.Header.Get("content-length") == "" {
				bodySize = conf.MaxBodyLogSize
			}
			requestBodyLeech = NewLeechedReadCloser(c.Request.Body, bodySize)
			c.Request.Body = requestBodyLeech

			// Let's do the same with the response body
			responseBodyLeech = NewLeechedGinResponseWriter(c.Writer, conf.MaxBodyLogSize)
			c.Writer = responseBodyLeech
		}

		// Start chrono
		startDate := time.Now()

		// Let's process the request
		c.Next()

		latency := time.Since(startDate)

		// However, the response's Header object will be dereferenced... we'll have
		// to store them them apart since we want to read them from the formatting
		// goroutine
		responseHeaders := make(map[string][]string)
		for name, value := range c.Writer.Header() {
			responseHeaders[name] = value
		}

		responseContentLength := max(c.Writer.Size(), 0)

		// Shall we pass the body as well ? If so let's not dereference it !
		if conf.BodyLogPolicy == LOG_ALL_BODIES || conf.BodyLogPolicy == LOG_BODIES_ON_ERROR && c.Writer.Status() >= 400 {

			// And parse all this to UTF-8 strings
			requestBody = string(requestBodyLeech.data)
			responseBody = string(responseBodyLeech.data)
		}

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
