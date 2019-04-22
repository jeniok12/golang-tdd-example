// handlers.go

package main

import (
	"./quote"
	"./recipient"
	"encoding/json"
	"net/http"
)

// HandleQuoteResponse ..
type HandleQuoteResponse struct {
	Quote      *quote.Quote
	Recipients []recipient.Recipient
}

func (s *server) handleQuotes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")

		quote, err := s.quoteGenerator.Generate(lang)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hqr := HandleQuoteResponse{
			Quote: quote,
		}

		resp, err := json.Marshal(hqr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(resp)
	}
}
