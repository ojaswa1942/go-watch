package watch

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"bytes"
	"github.com/alecthomas/chroma/quick"
)

func WatchMw(app http.Handler, dev bool) http.HandlerFunc {
	// app.HandleFunc("/watchMw/debug", func (w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintln(w, "<h1>Hello!</h1>")
	// })

	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				stackTrace := string(debug.Stack())
				log.Printf("[WATCH] panic: %v\nFollowing is the stack trace: %s", err, stackTrace)

				if !dev {
					http.Error(w, "Something went wrong!", http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "<h1>panic: %v</h1><h2>Stack trace:</h2><pre>%s</pre>", err, stackTrace)

				filePath := r.FormValue("path")
				if e := sourceCodeHandler(w, filePath); e != nil {
					http.Error(w, e.Error(), http.StatusInternalServerError)
				}
			}
		}()

		nw := &customResponseWriter{ResponseWriter: w}
		app.ServeHTTP(nw, r)
		// Copy contents from  writer to original writer
		nw.flush()
	}
}

// Writes source code to a dst writer
func sourceCodeHandler(dst io.Writer, filePath string) error {
	// filePath := "/home/ojaswa/Projects/go-watch-middleware/example/main.go"
	file, err := os.Open(filePath) 
	if err != nil {
		return err
	}

	fileBytes := bytes.NewBuffer(nil)
	if _, err = io.Copy(fileBytes, file); err != nil {
		return err
	}

	err = quick.Highlight(dst, fileBytes.String(), "go", "html", "monokai")
	if err != nil {
		return err
	}
	
	return nil
}

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
