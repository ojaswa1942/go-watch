# go-watch-middleware

Middleware for an HTTP server in GoLang to watch for unhandled exceptions. The middleware provides the following features:

- Provide an error boundary for panic
- Email alerts in production
- Slack alrets in production
- Log & display stack trace in development
- Code browser 

## Quick Start
A simple wrapper to your HTTP request multiplexer [ServeMux](https://golang.org/pkg/net/http/#ServeMux) to begin development.
```go
http.ListenAndServe(":3000", watch.WatchMw(mux))
```

## Options
[watch](#go-watch-middleware) uses functional options for various configurations:

Option | Default | Description
---- | :----: | --------
`WithDevelopment(bool)` | `true` | Set environment as development (true) or production (false). 
`WithDebugPath(string)` | `"/watch/debug"` | Path used by `watch` to show in-browser files for debugging (code browser) in development. This option will be silently ignored during production.
`WithEmail(watch.EmailDetails)` | - | Issue email alerts on Failure in Production. This option will be silently ignored during development.
`WithSlack(watch.SlackDetails)` | - | Issue slack alerts on Failure in Production. This option will be silently ignored during development.

You can find sample usage for these arguments [here](example/main.go). Kindly note that the usage example is a non-compulsive sample representation and may not directly represent your use case.

## Environment?
You can toggle between dev and production using `WithDevelopment` [option](#options). The following are things enabled per environment.

- **Production**: Email alerts, Slack alerts, Log
- **Development**: Stack Trace, Code browser (See `WithDebugPath` option), Log

