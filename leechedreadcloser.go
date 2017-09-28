package ginhttplogger

import (
	"bytes"
	"fmt"
	"io"
)

// LeechedReadCloser is a wrapper around io.ReadCloser that logs the first bytes of a Request's body
// (in our case) into a bytes buffer
type LeechedReadCloser struct {
	originalReadCloser io.ReadCloser

	data             []byte
	maxBodyLogSize   int64
	loggedBytesCount int64
	afterLogReader   io.Reader
}

// NewLeechedReadCloser creates a readCloser which reads and stores at most maxSize bytes
// from a ReadCloser and returns a clone of that same reader, data included
func NewLeechedReadCloser(source io.ReadCloser, maxSize int64) *LeechedReadCloser {
	return &LeechedReadCloser{
		data:               make([]byte, 0, maxSize),
		originalReadCloser: source,
		maxBodyLogSize:     maxSize,
	}
}

// GetLog returns the captured log paylaod
func (l *LeechedReadCloser) GetLog() []byte {
	if l.loggedBytesCount > 0 {
		return l.data[:l.loggedBytesCount]
	}
	return []byte("[Empty or not read by server]")
}

// Read reads stores up to maxSize bytes and keeps on reading
func (l *LeechedReadCloser) Read(b []byte) (n int, err error) {
	spaceLeft := l.maxBodyLogSize - l.loggedBytesCount
	if spaceLeft > 0 {
		// Let's read the request into our Logger (not all of it maybe)
		n, err := l.originalReadCloser.Read(l.data[l.loggedBytesCount:])
		if err != nil && err != io.EOF {
			return 0, fmt.Errorf("Error in LeechedReadCloser: %s", err)
		}

		// Let's reconcatenate what we've already read with the rest of the request
		// in a MultiReader...
		l.afterLogReader = io.MultiReader(bytes.NewReader(l.data[l.loggedBytesCount:l.loggedBytesCount+int64(n)]), l.originalReadCloser)

		l.loggedBytesCount += int64(n)
	}

	return l.afterLogReader.Read(b)
}

// Close closes on the original ReadCloser
func (l *LeechedReadCloser) Close() (err error) {
	return l.originalReadCloser.Close()
}
