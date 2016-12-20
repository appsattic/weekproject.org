package main

import "net/http"

func toSlash(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path + "/"
	http.Redirect(w, r, path, http.StatusFound)
}
