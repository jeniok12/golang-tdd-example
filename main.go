// main.go

package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type server struct {
	router *mux.Router
}

func main() {
	svr := server{
		router: mux.NewRouter(),
	}
	svr.routes()

	log.Fatal(http.ListenAndServe(":8080", svr.router))
}
