// main.go

package main

import (
	"github.com/gorilla/mux"
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

	if err := http.ListenAndServe(":8080", svr.router); err != nil {
		panic(err)
	}
}
