// routes.go

package main

import (
	"net/http"
)

func (s *server) routes() {
	s.router.HandleFunc("/quote", QuotesHandler)
}

func QuotesHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
