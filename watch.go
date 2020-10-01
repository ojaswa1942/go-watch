package watch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
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
		t := time.Now()

		if !wh.dev {
			http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			if wh.sendEmail {
				// Run in another go-routine to make it non-blocking
				go issueEmail(wh.emailDetails, t.Format(time.UnixDate), err.(string), stackTrace)
			}
			if wh.sendSlack {
				go issueSlack(wh.slackDetails, t.Format(time.UnixDate), err.(string), stackTrace)
			}
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "<html><body><h1>panic: %v</h1><h2>Stack trace:</h2><pre>%s</pre></body></html>", err, getLinkTrace(stackTrace, wh.debugPath))
	}
}

func issueEmail(d EmailDetails, timeError, panicError, stackTrace string) {

	recipients := strings.Join(d.To, ",")
	log.Println("[WATCH]: Issuing panic email alert to", recipients)

	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msgBody := fmt.Sprintf("<h1>Panic Alert!</h1>This is to bring to your attention that your application has hit an unexpected panic.<br />Fortunately, you use <b><a href=\"https://github.com/ojaswa1942/go-watch\">watch</a></b>. Just kidding, here is what you need to know:<br /><h2>Timestamp:</h2>%s<h2>Error:</h2>%s<h2>Stack trace:</h2><pre style=\"background:#1c1b1b;color:#fff;padding:12px;\">%s</pre>",
		timeError, panicError, stackTrace)

	msg := []byte("From: " + d.From + "\r\n" +
		"To: " + recipients + "\r\n" +
		"Subject: [WATCH] Panic Alert!\r\n" +
		mime + "\r\n" +
		"\r\n" +
		msgBody)

	err := smtp.SendMail(d.Addr, d.A, d.From, d.To, msg)
	if err != nil {
		log.Print("[WATCH]: Error while issuing panic email: ", err)
	} else {
		log.Println("[WATCH]: Issued email alerts")
	}
}

func issueSlack(d SlackDetails, timeError, panicError, stackTrace string) {

	webHook := d.WebHookURL
	log.Println("[WATCH]: Issuing panic slack alert to ", webHook)

	txt := ":bangbang: *Panic Alert!* :bangbang:\nThis is to bring to your attention that your application has hit an unexpected panic.\nFortunately, you use <https://github.com/ojaswa1942/go-watch|go-watch>. Just kidding, here is what you need to know:\n ```Timestamp: %s \nError: %s \nStack trace: %s ```"

	slackBody, _ := json.Marshal(map[string]string{"text": fmt.Sprintf(txt, timeError, panicError, stackTrace)})

	req, err := http.NewRequest(http.MethodPost, webHook, bytes.NewBuffer(slackBody))
	if err != nil {
		log.Print("[WATCH]: Error while issuing panic to slack: ", err)
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Print("[WATCH]: Error while issuing panic slack: ", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Print("[WATCH]: Error while issuing panic slack, got response code ", resp.StatusCode)
	} else {
		log.Println("[WATCH]: Issued slack alerts")
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
