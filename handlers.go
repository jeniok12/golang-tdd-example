// handlers.go

package main

import (
	"encoding/json"
	"net/http"
)

func (s *server) handleQuotes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")

		quote, err := s.quoteGenerator.Generate(lang)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp, err := json.Marshal(quote)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}
