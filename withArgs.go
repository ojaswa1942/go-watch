package watch

import (
	"net/smtp"
)

type WatchHandlerOption func(wh *watchHandler)

type watchHandler struct {
	dev, sendEmail, sendSlack bool
	emailDetails   EmailDetails
	slackDetails SlackDetails
	debugPath      string
}

type EmailDetails struct {
	Addr string
	A    smtp.Auth
	From string
	To   []string
}

type SlackDetails struct {
	WebHookURL string
}

func newWatchHandler(opts []WatchHandlerOption) watchHandler {
	wh := watchHandler{
		dev:       true,
		sendEmail: false,
		debugPath: "/watch/debug",
	}

	for _, opt := range opts {
		opt(&wh)
	}

	return wh
}

func WithDevelopment(dev bool) WatchHandlerOption {
	return func(wh *watchHandler) {
		wh.dev = dev
	}
}

func WithEmail(details EmailDetails) WatchHandlerOption {
	return func(wh *watchHandler) {
		wh.sendEmail = true
		wh.emailDetails = details
	}
}

func WithSlack(details SlackDetails) WatchHandlerOption {
	return func(wh *watchHandler) {
		wh.sendSlack = true
		wh.slackDetails = details
	}
}

func WithDebugPath(path string) WatchHandlerOption {
	return func(wh *watchHandler) {
		wh.debugPath = path
	}
}
