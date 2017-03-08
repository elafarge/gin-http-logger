package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"

	httpLogger "github.com/elafarge/gin-http-logger"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	fdc := httpLogger.FluentdLoggerConfig{
		Host:           "localhost",
		Port:           13713,
		Env:            "etienne-test",
		Tag:            "gin.requests",
		BodyLogPolicy:  httpLogger.LOG_BODIES_ON_ERROR,
		MaxBodyLogSize: 1000,
		DropSize:       5,
		RetryInterval:  5,
	}

	r.Use(httpLogger.New(fdc))

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
		var data map[string]string

		var buf bytes.Buffer
		buf.ReadFrom(c.Request.Body)
		log.Printf("BODY: \n ----- \n\n%s\n\n", buf.String())

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

	r.Run()
}
