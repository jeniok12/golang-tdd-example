// main_test.go

package main

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuoteAPI(t *testing.T) {
	svr := server{
		router: mux.NewRouter(),
	}
	svr.routes()

	req, _ := http.NewRequest("GET", "/quote", nil)
	response := makeHTTPCall(svr.router, req)

	assert.Equal(t, http.StatusOK, response.Code, "Response HTTP status in different than expected")
}

func makeHTTPCall(router *mux.Router, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}
