package main

import (
	"net/http"

	"github.com/gomiddleware/logger"
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(r.URL.Path))
}

func main() {
	// create the logger middleware
	log := logger.New()

	// make the http.Hander and wrap it with the log middleware
	handle := http.HandlerFunc(handler)
	http.Handle("/", log(handle))

	http.ListenAndServe(":8080", nil)
}
