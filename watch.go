package watch

import (
	"log"
	"net/http"
)

func WatchMw(app http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			r := recover()
			if r != nil {
				log.Println(r)
				http.Error(w, "Something went wrong!", http.StatusInternalServerError)
			}
		}()

		app.ServeHTTP(w, r)
	}
}