package ginfluentd

import (
	"bytes"
	"fmt"
	"io"
)

// A wrapper around io.ReadCloser that logs the first bytes of a Request's body
// (in our case) into a bytes buffer
type LeechedReadCloser struct {
	originalReadCloser io.ReadCloser

	data             []byte
	maxBodyLogSize   int64
	loggedBytesCount int64
}

func NewLeechedReadCloser(source io.ReadCloser, maxSize int64) *LeechedReadCloser {
	return &LeechedReadCloser{
		data:               make([]byte, maxSize),
		originalReadCloser: source,
		maxBodyLogSize:     maxSize,
	}
}

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
		mr := io.MultiReader(bytes.NewReader(l.data[l.loggedBytesCount:l.loggedBytesCount+n]), l.originalReadCloser)

		l.loggedBytesCount += int64(n)

		// ... and have gin read from it !
		return mr.Read(b)
	}

	return l.originalReadCloser.Read(b)
}

// Calls Close() on the original ReadCloser as well as the leech
func (l *LeechedReadCloser) Close() (err error) {
	return l.originalReadCloser.Close()
}
