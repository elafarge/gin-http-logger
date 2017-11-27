package main

import (
	"encoding/json"
	"errors"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	httpLogger "github.com/elafarge/gin-http-logger"
	formatters "github.com/elafarge/gin-http-logger/logrus-formatters"
)

func main() {
	// Let's format our stdout logs to Fluentd-compatible JSON
	log.SetFormatter(&formatters.FluentdFormatter{"2006-01-02T15:04:05.000000000Z"})

	// Get rid of gin's debug logs
	gin.SetMode(gin.ReleaseMode)

	// Middleware configuration
	r := gin.New()
	alc := httpLogger.AccessLoggerConfig{
		LogrusLogger:   log.StandardLogger(),
		BodyLogPolicy:  httpLogger.LogBodiesOnErrors,
		MaxBodyLogSize: 100,
		DropSize:       5,
		RetryInterval:  5,
	}
	r.Use(httpLogger.New(alc))
	r.Use(gin.Recovery())

	// Route configuration
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "hello dear",
		})
	})

	// Test that errors are caught
	r.GET("/test_error", func(c *gin.Context) {
		c.JSON(500, gin.H{
			"message": "Internal error occured",
		})
		c.Error(errors.New("other service not available"))
	})

	r.POST("/test", func(c *gin.Context) {
		c.Writer.Header().Set("X-Custom-Delirium", "Yo")
		var data map[string]string

		if err := json.NewDecoder(c.Request.Body).Decode(&data); err != nil {
			log.Printf("Error decoding body to JSON: %s", err)
			c.JSON(500, gin.H{"message": "error decoding JSON payload"})
		} else {
			log.Printf("Body: %s", data)
			c.JSON(201, gin.H{
				"message": "delirium registered",
			})
		}
	})

	r.POST("/test_error", func(c *gin.Context) {
		c.Writer.Header().Set("X-Custom-Delirium", "Yo")
		var data map[string]string

		if err := json.NewDecoder(c.Request.Body).Decode(&data); err != nil {
			log.Printf("Error decoding body to JSON: %s", err)
		}
		log.Printf("Body: %s", data)
		c.JSON(409, gin.H{
			"message": "beginning of long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - long error message - ",
		})
	})

	r.POST("/test_form", func(c *gin.Context) {
		c.Request.ParseMultipartForm(100)
		log.Infoln(c.Request.Form)

		c.JSON(409, gin.H{"hell": "o"})
	})

	if err := r.Run(":6060"); err != nil {
		log.Errorf("Error running webserver: %v", err)
	}
}
