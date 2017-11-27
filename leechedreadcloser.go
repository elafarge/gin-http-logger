package ginhttplogger

import (
	"io"
)

// LeechedReadCloser is a wrapper around io.ReadCloser that logs the first bytes of a Request's body
// (in our case) into a bytes buffer
type LeechedReadCloser struct {
	originalReadCloser io.ReadCloser

	data             []byte
	maxBodyLogSize   int64
	loggedBytesCount int64
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
		// Let's read the request into our Logger (not all of it maybe), but also let's make sure that
		// we'll be able to to copy all the content we read in l.data into b
		n, err := l.originalReadCloser.Read(l.data[l.loggedBytesCount : l.loggedBytesCount+min(int64(len(b)), spaceLeft)])

		// And copy what was read into the original slice
		copy(b, l.data[l.loggedBytesCount:l.loggedBytesCount+int64(n)])

		// Let's not forget to increment the pointer on the currently logged amount of bytes
		l.loggedBytesCount += int64(n)

		// And return what the Read() call we did on the original ReadCloser just returned, shhhhh
		return n, err
	}

	// Our leecher is full ? Nevermind, let's just call read on the original Reader. Apart from an
	// additional level in the call stack and an if statement, we have no overhead for large bodies :)
	return l.originalReadCloser.Read(b)
}

// Close closes on the original ReadCloser
func (l *LeechedReadCloser) Close() (err error) {
	return l.originalReadCloser.Close()
}
