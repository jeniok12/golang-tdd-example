// main_test.go

package main

import (
	"./quote"
	"./recipient"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var srv server
var testRecipientsPersistence *recipient.Persistence

var mockForismaticServiceResponse = map[string]interface{}{
	"quoteText":   "Bla Bla Bla",
	"quoteAuthor": "Bob",
}

var expectedRecipients = []interface{}{
	map[string]interface{}{
		"id":    1.0,
		"name":  "user1",
		"email": "user1@testmail.com",
	},
	map[string]interface{}{
		"id":    2.0,
		"name":  "user2",
		"email": "user2@testmail.com",
	},
	map[string]interface{}{
		"id":    3.0,
		"name":  "user3",
		"email": "user3@testmail.com",
	},
}

var expectedQuote = map[string]interface{}{
	"quoteAuthor": "Bob",
	"quoteText":   "Bla Bla Bla",
	"lang":        "en",
}

func TestMain(m *testing.M) {

	var err error
	testRecipientsPersistence, err = recipient.NewPersistence("localhost", "quotes_test")
	if err != nil {
		panic(err)
	}

	srv = server{
		router:            mux.NewRouter(),
		recipientsFetcher: testRecipientsPersistence,
	}
	srv.routes()

	code := m.Run()

	os.Exit(code)
}

func TestQuoteAPI(t *testing.T) {
	testCases := []struct {
		name            string
		lang            string
		mockHTTPService func() *httptest.Server
		presetDB        func(db *sql.DB) error
		expectedStatus  int
		expectedBody    map[string]interface{}
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
			func(db *sql.DB) error {
				query := "INSERT INTO recipients (id, name, email) VALUES ($1, $2, $3);"
				tx, err := db.Begin()

				for _, r := range expectedRecipients {
					rMap := r.(map[string]interface{})
					_, err = tx.Exec(query, rMap["id"], rMap["name"], rMap["email"])
					if err != nil {
						fmt.Println(fmt.Sprintf("Error: %+v", err))
					}
				}

				tx.Commit()
				return err
			},
			http.StatusOK,
			map[string]interface{}{
				"quote":      expectedQuote,
				"recipients": expectedRecipients,
			},
		},
	}
	for _, tC := range testCases {
		t.Run(tC.name, func(t *testing.T) {
			s := tC.mockHTTPService()
			defer s.Close()

			srv.quoteGenerator = &quote.Forismatic{
				URL:    s.URL,
				Client: s.Client(),
			}

			clearDB(testRecipientsPersistence.DB)
			tC.presetDB(testRecipientsPersistence.DB)

			req, _ := http.NewRequest("GET", "/quote", nil)
			req.URL.RawQuery = fmt.Sprintf("lang=%s", tC.lang)
			response := makeHTTPCall(srv.router, req)

			respBytes, _ := ioutil.ReadAll(response.Body)

			var respMap map[string]interface{}
			_ = json.Unmarshal(respBytes, &respMap)

			assert.Equal(t, tC.expectedStatus, response.Code, "Response HTTP status in different than expected")
			assert.EqualValues(t, tC.expectedBody, respMap, "Response HTTP body in different than expected")
		})
	}
}

func makeHTTPCall(router *mux.Router, req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

func clearDB(db *sql.DB) error {
	_, err := db.Exec("TRUNCATE TABLE recipients")
	return err
}
