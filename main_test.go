// main_test.go

package main

import (
	"./quote"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockForismaticServiceResponse = map[string]interface{}{
	"quoteText":   "Bla Bla Bla",
	"quoteAuthor": "Moshe",
}

var expectedQuote = quote.Quote{
	Author: "Moshe",
	Text:   "Bla Bla Bla",
	Lang:   "en",
}

func TestQuoteAPI(t *testing.T) {
	testCases := []struct {
		name            string
		lang            string
		mockHTTPService func() *httptest.Server
		expectedStatus  int
		expectedBody    *quote.Quote
	}{
		{
			"SuccessResponseFromForismaticService",
			"en",
			func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					assert.Equal(t, http.MethodGet, req.Method, "Should have different request method")

					assert.Equal(t, "getQuote", req.URL.Query().Get("method"), "Wrong method query param")
					assert.Equal(t, "json", req.URL.Query().Get("format"), "Wrong method query param")
					assert.Equal(t, "en", req.URL.Query().Get("lang"), "Wrong method query param")

					res, _ := json.Marshal(mockForismaticServiceResponse)
					rw.WriteHeader(http.StatusOK)
					rw.Write(res)
				}))

				return server
			},
			http.StatusOK,
			&expectedQuote,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			s := tC.mockHTTPService()
			defer s.Close()

			svr := server{
				router: mux.NewRouter(),
				quoteGenerator: &quote.Forismatic{
					URL:    s.URL,
					Client: s.Client(),
				},
			}
			svr.routes()

			req, _ := http.NewRequest("GET", "/quote", nil)
			req.URL.RawQuery = fmt.Sprintf("lang=%s", tC.lang)
			response := makeHTTPCall(svr.router, req)

			qBytes, _ := ioutil.ReadAll(response.Body)

			var q quote.Quote
			_ = json.Unmarshal(qBytes, &q)

			assert.Equal(t, tC.expectedStatus, response.Code, "Response HTTP status in different than expected")
			assert.Equal(t, tC.expectedBody, &q, "Response HTTP body in different than expected")
		})
	}
}

func makeHTTPCall(router *mux.Router, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}
