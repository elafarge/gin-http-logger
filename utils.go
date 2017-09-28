package ginhttplogger

import (
	"net/http"
	"strings"
)

func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Compute the size of request headers and flatten the header values
func normalizeHeaderMap(headerMap http.Header) (normalizedHeaderMap map[string]string, headerMapSize int) {
	headerMapSize = 0
	normalizedHeaderMap = make(map[string]string)
	for name, value := range headerMap {
		headerMapSize += len(name)
		for _, v := range value {
			headerMapSize += len(v)
		}
		normalizedHeaderMap[strings.ToLower(strings.Replace(name, "-", "_", -1))] = strings.Join(value, ", ")
	}
	return
}

// Formats a given payload as
func buildPayload(logEntry *Log) (logPayload AccessLog) {
	// Let's normalize our headers to match Kong's format as well as our Django logger's
	requestHeaders, requestHeaderSize := normalizeHeaderMap(logEntry.context.Request.Header)
	responseHeaders, responseHeaderSize := normalizeHeaderMap(logEntry.responseHeaders)

	// Let's parse the request and response objects and put that in a JSON-friendly map
	logPayload = AccessLog{
		TimeStarted:   logEntry.startDate.Format("2006-01-02T15:04:05.999+0100"),
		ClientAddress: logEntry.context.ClientIP(),
		Time:          int64(logEntry.latency.Nanoseconds() / 1000),
		Request: RequestLogEntry{
			Method:      logEntry.context.Request.Method,
			Path:        logEntry.context.Request.URL.Path,
			HTTPVersion: logEntry.context.Request.Proto,
			Headers:     requestHeaders,
			HeaderSize:  requestHeaderSize,
			Content: HTTPContent{
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
			Content: HTTPContent{
				Size:     logEntry.responseContentLength,
				MimeType: responseHeaders["content_type"],
				Content:  logEntry.responseBody,
			},
		},
	}

	return logPayload
}
