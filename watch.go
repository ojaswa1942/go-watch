package watch

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

func WatchMw(app http.Handler, dev bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				stackTrace := string(debug.Stack())
				log.Println("----------WATCH: LOG START----------")
				log.Printf("[WATCH] panic: %v\nFollowing is the stack trace: %s", err, stackTrace)
				log.Println("----------WATCH: LOG END----------")

				if !dev {
					http.Error(w, "Something went wrong!", http.StatusInternalServerError)
					return
				}

				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "<h1>panic: %v</h1><h2>Stack trace:</h2><pre>%s</pre>", err, getLinkTrace(stackTrace))
			}
		}()

		if path := r.URL.Path; strings.Contains(path, "/watch/debug") {
			sourceCodeHandler(w, r)
			return
		}

		nw := &customResponseWriter{ResponseWriter: w}
		app.ServeHTTP(nw, r)
		// Copy contents from  writer to original writer
		nw.flush()
	}
}

func sourceCodeHandler(w http.ResponseWriter, r *http.Request) {
	filePath := r.FormValue("path")
	lineStr := r.FormValue("line")
	lineNumber, err := strconv.Atoi(lineStr)
	if err != nil {
		lineNumber = -1
	}

	fileContent, err := getFileContent(filePath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	writeSource, err := getFormattedSource(fileContent, lineNumber)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = writeSource(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func getFileContent(filePath string) (string, error) {
	file, err := os.Open(strings.Trim(filePath, "\""))
	if err != nil {
		return "", err
	}

	fileBytes := bytes.NewBuffer(nil)
	if _, err = io.Copy(fileBytes, file); err != nil {
		return "", err
	}

	return fileBytes.String(), nil
}

func getFormattedSource(content string, lineNumber int) (func(io.Writer) error, error) {
	var highlightLine [][2]int
	if lineNumber > 0 {
		highlightLine = append(highlightLine, [2]int{lineNumber, lineNumber})
	}

	lexer := lexers.Get("go")
	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return nil, err
	}

	style := styles.Get("monokailight")
	if style == nil {
		style = styles.Fallback
	}
	formatter := html.New(html.TabWidth(2), html.WithLineNumbers(true), html.LineNumbersInTable(true), html.HighlightLines(highlightLine))

	return func(w io.Writer) error {
		err := formatter.Format(w, style, iterator)
		if err != nil {
			return err
		}
		return nil
	}, nil
}

func getLinkTrace(stackTrace string) string {
	re := regexp.MustCompile(`\t.*:\d*`)
	matches := re.ReplaceAllStringFunc(stackTrace, func(match string) string {
		split := strings.Split(match, ":")
		path, lineNum := strings.Trim(split[0], "\t "), split[1]
		return fmt.Sprintf("> <a target=\"_blank\" href=/watch/debug/?line=%s&path=%s>%s:%s</a>", lineNum, url.PathEscape(path), path, lineNum)
	})

	return matches
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
