package main

import (
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

	r.POST("/test", func(c *gin.Context) {
		c.Writer.Header().Set("X-Custom-Delirium", "Yo")
		c.JSON(201, gin.H{
			"message": "delirium registered",
		})
	})

	r.Run()
}
