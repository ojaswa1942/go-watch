package watch

import (
	"bytes"
	"fmt"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
)

func WatchMw(app http.Handler, opts ...WatchHandlerOption) http.HandlerFunc {
	wh := newWatchHandler(opts)

	return func(w http.ResponseWriter, r *http.Request) {
		defer wh.handleExceptions(w)

		if path := r.URL.Path; strings.Contains(path, wh.debugPath) {
			sourceCodeHandler(w, r)
			return
		}

		nw := &customResponseWriter{ResponseWriter: w}
		app.ServeHTTP(nw, r)
		// Copy contents from  writer to original writer
		nw.flush()
	}
}

func (wh *watchHandler) handleExceptions(w http.ResponseWriter) {
	err := recover()
	if err != nil {
		stackTrace := string(debug.Stack())
		log.Println("----------WATCH: LOG START----------")
		log.Printf("[WATCH] panic: %v\nFollowing is the stack trace: %s", err, stackTrace)
		log.Println("----------WATCH: LOG END----------")

		if !wh.dev {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "<h1>panic: %v</h1><h2>Stack trace:</h2><pre>%s</pre>", err, getLinkTrace(stackTrace, wh.debugPath))
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

func getLinkTrace(stackTrace, debugPath string) string {
	re := regexp.MustCompile(`\t.*:\d*`)
	matches := re.ReplaceAllStringFunc(stackTrace, func(match string) string {
		split := strings.Split(match, ":")
		path, lineNum := strings.Trim(split[0], "\t "), split[1]
		return fmt.Sprintf("> <a target=\"_blank\" href=%s?line=%s&path=%s>%s:%s</a>", debugPath, lineNum, url.PathEscape(path), path, lineNum)
	})

	return matches
}
