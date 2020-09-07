package watch

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

type customResponseWriter struct {
	http.ResponseWriter
	writes [][]byte
	status int
}

func (rw *customResponseWriter) Write(b []byte) (int, error) {
	// Pretending that there is no error :(
	rw.writes = append(rw.writes, b)
	return len(b), nil
}

func (rw *customResponseWriter) WriteHeader(statusCode int) {
	// if already set, throw error
	if rw.status != 0 {
		panic(fmt.Sprintf("Status code %d already exists", rw.status))
	}

	rw.status = statusCode
}

// Flushes data and headers to original writer
func (rw *customResponseWriter) flush() error {
	if rw.status != 0 {
		rw.ResponseWriter.WriteHeader(rw.status)
	}

	for _, write := range rw.writes {
		_, err := rw.ResponseWriter.Write(write)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rw *customResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("the ResponseWriter does not support Hijacker Interface")
	}

	return hijacker.Hijack()
}

func (rw *customResponseWriter) Flush() {
	flusher, ok := rw.ResponseWriter.(http.Flusher)
	if ok {
		if rw.status == 0 {
			rw.WriteHeader(http.StatusOK)
		}
		flusher.Flush()
	}
}
