package watch

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

func WatchMw(app http.Handler, dev bool) http.HandlerFunc {
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
			}
		}()

		nw := &customResponseWriter{ ResponseWriter: w }
		app.ServeHTTP(nw, r)
		// Copy contents from  writer to original writer
		nw.flush()
	}
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
