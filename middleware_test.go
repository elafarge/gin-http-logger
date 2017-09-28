package ginhttplogger

import (
	"bytes"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test that the middleware doesn't alter request or response bodies even long ones.
// For that purpose, we'll mimic a file upload. The file will be a randomly generated 1MB array :)
func TestMiddlewareBodyLogging(t *testing.T) {
	// Let's setup Gin for testing
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Let's build our config
	conf := AccessLoggerConfig{
		BodyLogPolicy:  LogAllBodies,
		MaxBodyLogSize: 1000,
		DropSize:       10,
	}

	// Let's inject our middleware into Gin's router
	logQueue := NewMockedLogForwardingQueue(conf)
	router.Use(buildLoggingMiddleware(conf, logQueue))
	go logQueue.run()

	// Let's setup a test route that replies 200 and sends the request body back
	router.POST("/mirror", func(c *gin.Context) {
		var buf bytes.Buffer
		buf.ReadFrom(c.Request.Body)
		defer c.Request.Body.Close()

		c.Writer.Write(mirror(buf.Bytes()))
		c.Status(201)
		c.Writer.Flush()
	})

	// Let's generate our request body
	body := make([]byte, 1000000)
	rand.Read(body)

	// POST it
	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/mirror", bytes.NewReader(body))
	router.ServeHTTP(w, r)

	// Assert that it has remained intact all along the query
	responseBody := w.Body.Bytes()
	assert.Equal(t, body, mirror(responseBody), "Request and reversed response body should be equal")

	// Assert that we logged the beginning of the body as well as the response
	logEntry := logQueue.pop()
	assert.Equal(t, string(body[:1000]), logEntry.requestBody, "Request and reversed response body should be equal")
	assert.Equal(t, string(mirror(body)[:1000]), logEntry.responseBody, "Request and reversed response body should be equal")

}

func mirror(array []byte) (ret []byte) {
	ret = make([]byte, len(array))
	for i := 0; i < len(array); i++ {
		ret[i] = array[len(array)-i-1]
	}
	return ret
}
