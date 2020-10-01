package main

import (
	"fmt"
	"github.com/ojaswa1942/go-watch"
	"log"
	"net/http"
	"net/smtp"
)

func main() {
	mux := http.NewServeMux()
	// Normal Panic
	mux.HandleFunc("/panic/", panicDemo)
	// Panic after some http response is already written
	mux.HandleFunc("/panic-after/", panicAfterDemo)
	// No panic, ofc
	mux.HandleFunc("/", hello)

	// Construct your own emailDetails
	// fn doing the work for example
	emailDetails := configureSMTP()
	slackDetails := watch.SlackDetails{WebHookURL: "https://hooks.slack.com/services/"}
	fmt.Println("Listening on Port 3000")
	log.Fatal(http.ListenAndServe(":3000",
		watch.WatchMw(
			mux,
			// Change this to true for in-browser stack trace as response
			watch.WithDevelopment(false),
			watch.WithDebugPath("/debug/boo/"),
			watch.WithEmail(emailDetails),
			watch.WithSlack(slackDetails),
		),
	))
}

func configureSMTP() watch.EmailDetails {
	hostname := "mail.example.com" // Used for TLS validation
	auth := smtp.PlainAuth("", "user", "pass", hostname)

	details := watch.EmailDetails{
		Addr: hostname + ":587", // or as per your configurations
		A:    auth,
		From: "alert@example.com",
		To:   []string{"one@gmail.com", "two@yahoo.com"},
	}

	return details
}

func panicDemo(w http.ResponseWriter, r *http.Request) {
	funcThatPanics()
}

func panicAfterDemo(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "<h1>Hello!</h1>")
	funcThatPanics()
}

func funcThatPanics() {
	panic("Oh no!")
}

func hello(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "<h1>Hello!</h1>")
}
