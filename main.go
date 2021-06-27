package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/{query}", func(w http.ResponseWriter, r *http.Request) {
		v := mux.Vars(r)
		q := v["query"]

		fmt.Fprintf(w, q)
	})

	http.ListenAndServe(":8080", r)
}
