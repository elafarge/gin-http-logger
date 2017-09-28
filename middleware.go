package ginhttplogger

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

//// CONFIG ////
const (
	// LogBodiesOnError indicates to log bodies only for 4xx & 5xx errors
	LogBodiesOnErrors = 1 + iota
	// LogNoBody stipulates that we should never log bodies
	LogNoBody
	// LogAllBodies can be used to log all request bodies (use with care)
	LogAllBodies
)

// NoBodyHTTPMethods is the list of methods for which we don't log bodies cause they don't have any
var NoBodyHTTPMethods = map[string]struct{}{
	"HEAD":    struct{}{},
	"OPTIONS": struct{}{},
	"GET":     struct{}{},
}

// AccessLoggerConfig describe the config of our access logger
type AccessLoggerConfig struct {
	LogrusLogger   *logrus.Logger
	Host           string
	Port           int
	Path           string
	DropSize       int
	MaxBodyLogSize int64
	BodyLogPolicy  int
	RetryInterval  time.Duration
}

func buildLoggingMiddleware(conf AccessLoggerConfig, logQueue LogForwardingQueue) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody, responseBody string
		var responseBodyLeech *LeechedGinResponseWriter
		var requestBodyLeech *LeechedReadCloser

		if conf.BodyLogPolicy != LogNoBody {
			// Let's use a Leech to pump a limited amount of bytes on the request
			// body into RAM as this body is read
			bodySize := min(c.Request.ContentLength, conf.MaxBodyLogSize)

			// If the Content-Length header ain't set let's use a buffer of
			// MaxBodyLogSize to log the request body.
			if _, ok := NoBodyHTTPMethods[c.Request.Method]; !ok && c.Request.Header.Get("content-length") == "" {
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
		if conf.BodyLogPolicy == LogAllBodies || conf.BodyLogPolicy == LogBodiesOnErrors && c.Writer.Status() >= 400 {

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
		case logQueue.intake() <- logEntry:
		default:
			log.Println("[WARNING][http-logging-middleware] Impossible to forward requests into log queue, channel full.")
		}
	}
}

// New returns an gin.HandlerFunc that will log our HTTP requests
func New(conf AccessLoggerConfig) gin.HandlerFunc {
	// Parse configuration, apply default arguments
	// (Host and Port are mandatory)
	if conf.BodyLogPolicy == 0 {
		conf.BodyLogPolicy = LogNoBody
	}

	if conf.Path == "" {
		conf.Path = "/gin.requests"
	}

	if conf.DropSize == 0 {
		conf.DropSize = 1024
	}

	if conf.MaxBodyLogSize == 0 {
		conf.MaxBodyLogSize = 4096
	}

	if conf.RetryInterval == 0 {
		conf.RetryInterval = 10 * time.Second
	}

	// Apply configuration
	var logQueue LogForwardingQueue
	if len(conf.Host) > 0 && conf.Port != 0 {
		logQueue = NewHTTPLogForwardingQueue(conf)
	} else if conf.LogrusLogger != nil {
		logQueue = NewLogrusLogForwardingQueue(conf)
	}

	// Run the log-forwarding goroutine
	go logQueue.run()

	return buildLoggingMiddleware(conf, logQueue)
}
