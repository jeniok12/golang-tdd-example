// quote/quote_test.go

package quote

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

var mockForismaticServiceResponse = map[string]interface{}{
	"quoteText":   "Bla Bla Bla",
	"quoteAuthor": "Bob",
}

var expectedQuote = Quote{
	Author: "Bob",
	Text:   "Bla Bla Bla",
	Lang:   "en",
}

func TestForismatic_Generate(t *testing.T) {
	testCases := []struct {
		name               string
		lang               string
		createMocks        func() *httptest.Server
		expectedQuote      *Quote
		expectedToGetError bool
	}{
		{
			"SuccessResponseFromHTTPWrapper",
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
			&expectedQuote,
			false,
		},
		{
			"ErrorFromHTTPWrapper",
			"en",
			func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
					rw.WriteHeader(http.StatusInternalServerError)
				}))

				return server
			},
			nil,
			true,
		},
	}

	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			server := tC.createMocks()
			defer server.Close()

			forismatic := Forismatic{
				URL:    server.URL,
				Client: server.Client(),
			}

			actulaQuote, err := forismatic.Generate(tC.lang)

			assert.Equal(t, tC.expectedQuote, actulaQuote, "Expected Quote is different from actual")
			if tC.expectedToGetError {
				assert.Error(t, err, "Got no error when expected")
			} else {
				assert.NoError(t, err, "Got error when not expected")
			}
		})
	}
}
