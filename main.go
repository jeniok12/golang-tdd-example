// main.go

package main

import (
	"./quote"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"time"
)

// QuoteGenerator ...
type QuoteGenerator interface {
	Generate(lang string) (*quote.Quote, error)
}

type server struct {
	router         *mux.Router
	quoteGenerator QuoteGenerator
}

func main() {
	svr := server{
		router: mux.NewRouter(),
		quoteGenerator: &quote.Forismatic{
			URL: "http://api.forismatic.com/api/1.0/",
			Client: &http.Client{
				Timeout: 30 * time.Second,
			},
		},
	}
	svr.routes()

	log.Fatal(http.ListenAndServe(":8080", svr.router))
}
