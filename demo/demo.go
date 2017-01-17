package main

import (
	"errors"

	fluentdLogger "github.com/Dreem-Devices/ginfluentd"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	fdc := fluentdLogger.FluentdLoggerConfig{
		Host:              "localhost",
		Port:              13713,
		Env:               "etienne-test",
		Tag:               "gin.requests",
		DropSize:          10000,
		RetryInterval:     30,
		FieldsToObfuscate: nil,
		MaxBodyLogSize:    10000,
		BodyLogPolicy:     fluentdLogger.LOG_ALL_BODIES,
	}

	r.Use(fluentdLogger.New(fdc))

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
		c.Error(errors.New("Other service not available!"))
	})

	r.POST("/test", func(c *gin.Context) {
		c.Writer.Header().Set("X-Custom-Delirium", "Yo")
		c.JSON(201, gin.H{
			"message": "delirium registered",
		})
	})

	r.Run()
}
