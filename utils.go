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
