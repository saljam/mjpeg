// Package mjpeg implements a simple MJPEG streamer.
//
// Stream objects implement the http.Handler interface, allowing to use them with the net/http package like so:
//	stream = mjpeg.NewStream()
//	http.Handle("/camera", stream)
// Then push new JPEG frames to the connected clients using stream.UpdateJPEG().
package mjpeg

import (
	"fmt"
	"net/http"
	"sync"
)

// Stream represents a single video feed.
type Stream struct {
	m     map[chan []byte]bool
	frame []byte
	lock sync.Mutex
}

const boundaryWord = "MJPEGBOUNDARY"
const headerf = "\r\n" +
	"--" + boundaryWord + "\r\n" +
	"Content-Type: image/jpeg\r\n" +
	"Content-Length: %d\r\n" +
	"X-Timestamp: 0.000000\r\n" +
	"\r\n"

// ServeHTTP responds to HTTP requests with the MJPEG stream, implementing the http.Handler interface.
func (s *Stream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "multipart/x-mixed-replace;boundary="+boundaryWord)

	c := make(chan []byte)
	s.lock.Lock()
	s.m[c] = true
	s.lock.Unlock()

	for {
		b := <-c
		_, err := w.Write(b)
		if err != nil {
			break
		}
	}

	s.lock.Lock()
	delete(s.m, c)
	s.lock.Unlock()
}

// UpdateJPEG pushes a new JPEG frame onto the clients.
func (s *Stream) UpdateJPEG(jpeg []byte) {
	header := fmt.Sprintf(headerf, len(jpeg))
	if len(s.frame) < len(jpeg)+len(header) {
		s.frame = make([]byte, (len(jpeg)+len(header))*2)
	}

	copy(s.frame, header)
	copy(s.frame[len(header):], jpeg)

	s.lock.Lock()
	for c := range s.m {
		c <- s.frame
	}
	s.lock.Unlock()
}

// NewStream initializes and returns a new Stream.
func NewStream() *Stream {
	return &Stream{
		m:     make(map[chan []byte]bool),
		frame: make([]byte, len(headerf)),
	}
}
