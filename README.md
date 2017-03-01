Gin-Gonic HTTP log forwarder
============================

A Gin-Gonic middleware forwarding access logs over HTTP (in JSON).

Features
--------
 * Non blocking: heavy calls made in a goroutine separated from the GIN handler
   one.
 * Possible to log the request & response bodies all the time, when the handler
   returns an error code (4xx, 5xx) or never
 * Memory efficient: uses the standard `io` library abstractions to limit
   what's loaded in memory, body logs are truncated to 10000 bytes by default,
   in case of connection failure with the HTTP endpoint, no more than 1000 logs
   will be kept in memory
 * Lightweight but complete

Usage
-----

Like any other Gin-Gonic middleware:

```golang
r := gin.Default()

	fdc := httpLogger.FluentdLoggerConfig{
		Host:           "localhost",
		Port:           13713,
		Env:            "etienne-kubernetes",
		Tag:            "gin.requests",
		BodyLogPolicy:  fluentdLogger.LOG_BODIES_ON_ERROR,
		MaxBodyLogSize: 50,
		DropSize:       5,
		RetryInterval:  5,
	}

	r.Use(fluentdLogger.New(fdc))
```

### Compatible with
 * FluentD (tested)

### Author
 * Étienne Lafarge <etienne@rythm.co>
