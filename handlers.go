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
	Quote      *quote.Quote          `json:"quote"`
	Recipients []recipient.Recipient `json:"recipients"`
}

func (s *server) handleQuotes() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := r.URL.Query().Get("lang")

		quote, err := s.quoteGenerator.Generate(lang)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		recipients, err := s.recipientsFetcher.AllRecipients()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hqr := HandleQuoteResponse{
			Quote:      quote,
			Recipients: recipients,
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
