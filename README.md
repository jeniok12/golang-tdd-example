# Step by step guide of building HTTP service using Golang and TDD

## Intro

In the following guide, I will present how to build an HTTP service using `go1.12.4`, `gorilla/mux` for URL routing, `stretchr/testify` for mocks and assertions and `lib/pq` for Postgres. This is my stack, feel free to use different packages it shouldn't affect much of the following. Also, I will try to follow TDD approach as much as possible. All of the source code is available on GitHub.

```console
git clone https://github.com/jeniok12/golang-tdd-example.git
```

## Instalations
```cosnole
go get -u github.com/gorilla/mux
go get github.com/stretchr/testify
go get github.com/lib/pq
go get -tags 'postgres' -u github.com/golang-migrate/migrate/cmd/migrate
```

## Step 0 - What should we build?

I decided to create an InspiringQuotes service, which I will use from time to time in order to increase my teammate's morale. This service will generate an inspiring quote using [forismatic.com](http://forismatic.com/en/api/) service and send it to list of my teammates (stored in Postgres DB) via Email. In the end, my colleagues will have an example of how to write a testable Golang service in addition to high morale to use it :)

## Step 1 - New Service is born

Let's create a new service. The `main.go` file will look like this:

```golang
// main.go

package main

import (
  "github.com/gorilla/mux"
  "net/http"
)

func main() {
  r := mux.NewRouter()
  log.Fatal(http.ListenAndServe(":8080", svr.router))
}
```

Let's run our server using:

```console
> go run .
```

And then send an HTTP request and see.

```console
> curl http://localhost:8080
404 page not found
```

Now we are ready to write our first test. We want to test the service e2e, but without running it. We will send a sample HTTP request and then run asserts on the HTTP response and on other side effects caused by this call (DB updates, calls to other services and so).In order to do that we need:
1. Make the router accessible for tests.
1. Use `httptest.ResponseRecorder` in order to assert service response.

Let's create a `server` struct that will hold all service dependencies. As for now, it will hold only the router.

```golang
type server struct {
  router *mux.Router
}
```

Next, initialize it in `main` function. So, `main.go` will look:

```golang
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

  log.Fatal(http.ListenAndServe(":8080", svr.router))
}
```

Implement the `routes` method. For now, it will generate no routes.

```golang
// routes.go
package main

func (s *server) routes() {
  // Add routes handlers here.
}
```

Now let's write an E2E test. We use `httptest.ResponseRecorder` as  `http.ResponseWriter` in our calls to the router. `httptest.ResponseRecorder` will store the HTTP response, so we will be able to assert it with the expected result.

```golang
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
```

Run the test by executing:

```console
> go test -v ./...
```

And the result is...

```console
=== RUN   TestQuoteAPI
--- FAIL: TestQuoteAPI (0.00s)
    main_test.go:22:
                Error Trace:    main_test.go:22
                Error:          Not equal:
                                expected: 200
                                actual  : 404
                Test:           TestQuoteAPI
                Messages:       Response HTTP status in different than expected
FAIL
```

Failed as expected.

Now is the time to make this test pass. We Implementing the required route.

```golang
// routes.go
package main

import (
    "net/http"
)

func (s *server) routes() {
    s.router.HandleFunc("/quote", QuotesHandler)
}

func QuotesHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
}
```

and the test passes.

```console
> go test -v ./...
=== RUN   TestQuoteAPI
--- PASS: TestQuoteAPI (0.00s)
PASS
```

## Step 2 - Call to another service

On this step, we will implement request validation and the actual request to http://forismatic.com/en/api/ in order to get the quote.

We will start by implementing the `quote` package. Create a Quote model.

```golang
// quote/quote.go
package quote

type Quote struct {
  Text   string `json:"quoteText"`
  Author string `json:"quoteAuthor"`
  Lang   string `json:"lang"`
}
```

Quotes generator interface that will be added to the server as a dependency.

```golang
// main.go
...
type QuoteGenerator interface {
  Generate(lang string) (*quote.Quote, error)
}

type server struct {
  router         *mux.Router
  quoteGenerator QuoteGenerator
}
...
```

Now we want to inject server dependencies to the handlers. For that, we will wrap the `HandlerFunc` inside `Handler`, and move it to a different file. 

```golang
//handlers.go
package main

import (
  "net/http"
)

func (s *server) handleQuotes() http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
  }
}
```

```golang
//routers.go
package main

func (s *server) routes() {
  s.router.HandleFunc("/quote", s.handleQuotes())
}
```

Now we write a test for `handleQuotes`. This test mocks `QuoteGenerator` and test the service if the Quote was generated successfully and not.

Define a mock for 'QuoteGenerator' using `github.com/stretchr/testify/mock`:

```golang
// handlers_test.go
package main

import (
  // ...
  "github.com/stretchr/testify/mock"
  // ...
)

// ...
type MockQuoteGenerator struct {
  mock.Mock
}

func (m *MockQuoteGenerator) Generate(lang string) (*quote.Quote, error) {
  args := m.Called(lang)
  quote, _ := args.Get(0).(*quote.Quote)
  return quote, args.Error(1)
}
// ...
```

Then write test using this mock

```golang
// handlers_test.go
package main

import (
  "./quote"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/mock"
  "net/http"
  "net/http/httptest"
  "testing"
)

type MockQuoteGenerator struct {
  mock.Mock
}

// ...

func TestHandleQuotes(t *testing.T) {
  mockQuoteGenerator := MockQuoteGenerator{}

  svr := server{
    quoteGenerator: &mockQuoteGenerator,
  }

  rr := httptest.NewRecorder()

  req, _ := http.NewRequest("GET", "/quote", nil)
  req.URL.RawQuery = "lang=en"

  mockQuoteGenerator.On("Generate", "en").Return(&quote.Quote{}, nil)

  svr.handleQuotes()(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code, "Response HTTP status in different than expected")
  mockQuoteGenerator.AssertExpectations(t)
}
```

We make this mock return zeroed struct of `Quote`. And we are able to assert if the method has been called with expected params. If we run the test we see.

```console
> go test -v .
=== RUN   TestHandleQuotes
--- FAIL: TestHandleQuotes (0.00s)
    handlers_test.go:40: FAIL:  Generate(string)
                        at: [handlers_test.go:35]
    handlers_test.go:40: FAIL: 0 out of 1 expectation(s) were met.
                The code you are testing needs to make 1 more call(s).
                at: [handlers_test.go:40]
```

As expected the test failing because we don't call `QuoteGenerator.Generate` in our code. Let's fix it.

```golang
// handlers.go
package main

import (
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
```

Let's run the tests again:

```console
> go test -v .
=== RUN   TestHandleQuotes
--- PASS: TestHandleQuotes (0.00s)
    handlers_test.go:40: PASS:  Generate(string)
=== RUN   TestQuoteAPI
--- FAIL: TestQuoteAPI (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
...
```

Now our test passes but our e2e test is failing because we didn't implement the actual `QuoteGenerator`. We deal with it in a minute, but first of all, let's test `quoteHandler` for negative flow.

We add another test.

```golang
//handlers_test.go
func TestHandleQuotes_QuoteGeneratorError(t *testing.T) {
  mockQuoteGenerator := MockQuoteGenerator{}

  svr := server{
    quoteGenerator: &mockQuoteGenerator,
  }

  rr := httptest.NewRecorder()

  req, _ := http.NewRequest("GET", "/quote", nil)
  req.URL.RawQuery = "lang=en"

  mockQuoteGenerator.On("Generate", "en").Return(nil, errors.New("sample error"))

  svr.handleQuotes()(rr, req)

  assert.Equal(t, http.StatusInternalServerError, rr.Code, "Response HTTP status in different than expected")
  mockQuoteGenerator.AssertExpectations(t)
}
```

This is passing too, but seem that we have code duplication that we want to prevent.

```console
> go test -v .
=== RUN   TestHandleQuotes
--- PASS: TestHandleQuotes (0.00s)
    handlers_test.go:42: PASS:  Generate(string)
=== RUN   TestHandleQuotes_QuoteGeneratorError
--- PASS: TestHandleQuotes_QuoteGeneratorError (0.00s)
    handlers_test.go:62: PASS:  Generate(string)
=== RUN   TestQuoteAPI
--- FAIL: TestQuoteAPI (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
...
```

We can use testCases table with Subtests.

```golang
// handlers_test.go
package main

// ...

func TestHandleQuotes(t *testing.T) {
  testCases := []struct {
    name           string
    lang           string
    createMocks    func() *MockQuoteGenerator
    expectedStatus int
  }{
    {
      "QuoteGenerator_Success",
      "en",
      func() *MockQuoteGenerator {
        mockQuoteGenerator := MockQuoteGenerator{}
        mockQuoteGenerator.On("Generate", "en").Return(&quote.Quote{}, nil)
        return &mockQuoteGenerator
      },
      http.StatusOK,
    },
    {
      "QuoteGenerator_Fail",
      "en",
      func() *MockQuoteGenerator {
        mockQuoteGenerator := MockQuoteGenerator{}
        mockQuoteGenerator.On("Generate", "en").Return(nil, errors.New("sample error"))
        return &mockQuoteGenerator
      },
      http.StatusInternalServerError,
    },
  }

  for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
      mockQuoteGenerator := tc.createMocks()
      svr := server{
        quoteGenerator: mockQuoteGenerator,
      }

      rr := httptest.NewRecorder()
      req, _ := http.NewRequest("GET", "/quote", nil)
      req.URL.RawQuery = fmt.Sprintf("lang=%s", tc.lang)

      svr.handleQuotes()(rr, req)

      assert.Equal(t, tc.expectedStatus, rr.Code, "Response HTTP status in different than expected")
      mockQuoteGenerator.AssertExpectations(t)
    })
  }
}
```

Let's run the tests again:

```console
> go test -v .
=== RUN   TestHandleQuotes
=== RUN   TestHandleQuotes/QuoteGenerator_Success
=== RUN   TestHandleQuotes/QuoteGenerator_Fail
--- PASS: TestHandleQuotes (0.00s)
    --- PASS: TestHandleQuotes/QuoteGenerator_Success (0.00s)
        handlers_test.go:69: PASS:      Generate(string)
    --- PASS: TestHandleQuotes/QuoteGenerator_Fail (0.00s)
        handlers_test.go:69: PASS:      Generate(string)
=== RUN   TestQuoteAPI
--- FAIL: TestQuoteAPI (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered]
```

Now let's go back to our failing e2e test and implement the `QuoteGenereator`. We add the implementation to `quote.go` file

```golang
// quote/quote.go
package quote

// ...

// HTTPWrapper ...
type HTTPWrapper interface {
  Do(req *http.Request) (*http.Response, error)
}

// Forismatic ...
type Forismatic struct {
  Client HTTPWrapper
}

// GenerateQuote ...
func (f *Forismatic) Generate(lang string) (*Quote, error) {
  return nil, errors.New("Not implemented")
}
```

I prefer to call it the same as the name of the service it uses. The 'Forismatic' struct holds the `HTTPWrapper` interface of `http.Client` so we will be able to mock it in tests.

Now we inject the actual implementation to our `server`.

```golang
// main.go

// ...

type server struct {
  router         *mux.Router
  quoteGenerator QuoteGenerator
}

func main() {
  // ...
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
  // ...
}
```

```golang
// main_test.go
// ...
func TestQuoteAPI(t *testing.T) {
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

  req, _ := http.NewRequest("GET", "/quote", nil)
  response := makeHTTPCall(svr.router, req)

  assert.Equal(t, http.StatusOK, response.Code, "Response HTTP status in different than expected")
}
// ...
```

Now let's run the tests

```concole
> go test -v .
=== RUN   TestHandleQuotes
=== RUN   TestHandleQuotes/QuoteGenerator_Success
=== RUN   TestHandleQuotes/QuoteGenerator_Fail
--- PASS: TestHandleQuotes (0.00s)
    --- PASS: TestHandleQuotes/QuoteGenerator_Success (0.00s)
        handlers_test.go:69: PASS:      Generate(string)
    --- PASS: TestHandleQuotes/QuoteGenerator_Fail (0.00s)
        handlers_test.go:69: PASS:      Generate(string)
=== RUN   TestQuoteAPI
--- FAIL: TestQuoteAPI (0.00s)
    main_test.go:20: 
                Error Trace:    main_test.go:20
                Error:          Not equal: 
                                expected: 200
                                actual  : 500
                Test:           TestQuoteAPI
                Messages:       Response HTTP status in different than expected
FAIL
```

We still see the test fail. But it is not `panic: runtime error: invalid memory address` anymore. Our service returns 500 because `Forismatic.Generate` returns not implemented error. Let's implement this method, starting with the test.

```golang
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

var expectedQuote = map[string]interface{}{
  "quoteAuthor": "Bob",
  "quoteText":   "Bla Bla Bla",
  "lang":        "en",
}

func TestForismatic_Generate(t *testing.T) {
  testCases := []struct {
    name               string
    lang               string
    createMocks        func() *httptest.Server
    expectedStatus  int
        expectedBody    map[string]interface{}
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

      assert.Equal(t, &expectedQuote, actulaQuote, "Expected Quote is different from actual")
      assert.Nil(t, err, "Error is not nil")
    })
  }
}
```

We already know how to use `testCases` table so I used it from the start. The new thing here is that this logic depends on `http.Client`. Good news that we don't need to mock it using `testify/mock`. Instead, we use `httptest.Server`. This mock server gives us the ability to assert on the external requests and mock the responses. We run the test and see...

```console
> go test -v ./...
...
=== RUN   TestForismatic_Generate
=== RUN   TestForismatic_Generate/SuccessResponseFromHTTPWrapper
--- FAIL: TestForismatic_Generate (0.00s)
    --- FAIL: TestForismatic_Generate/SuccessResponseFromHTTPWrapper (0.00s)
        quote_test.go:63: 
                Error Trace:    quote_test.go:63
                Error:          Not equal: 
                                expected: &quote.Quote{Text:"Bla Bla Bla", Author:"Bob", Lang:"en"}
                                actual  : (*quote.Quote)(nil)
                            
                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1,6 +1,2 @@
                                -(*quote.Quote)({
                                - Text: (string) (len=11) "Bla Bla Bla",
                                - Author: (string) (len=5) "Bob",
                                - Lang: (string) (len=2) "en"
                                -})
                                +(*quote.Quote)(<nil>)
                                 
                Test:           TestForismatic_Generate/SuccessResponseFromHTTPWrapper
                Messages:       Expected Quote is different from actual
        quote_test.go:64: 
                Error Trace:    quote_test.go:64
                Error:          Expected nil, but got: &errors.errorString{s:"Not implemented"}
                Test:           TestForismatic_Generate/SuccessResponseFromHTTPWrapper
                Messages:       Error is not nil
FAIL
```

Failed as expected, let's implement the `Generate` function

```golang
// quote/quote.go
package quote

// ...

// Generate ...
func (f *Forismatic) Generate(lang string) (*Quote, error) {
  req, err := http.NewRequest("GET", f.URL, nil)
  if err != nil {
    return nil, err
  }
  req.URL.RawQuery = fmt.Sprintf("method=getQuote&format=json&lang=%s", lang)

  resp, err := f.Client.Do(req)
  if err != nil {
    return nil, err
  }

  if resp != nil && resp.StatusCode != http.StatusOK {
    return nil, errors.New("Not OK response status")
  }

  bodyBytes, err := ioutil.ReadAll(resp.Body)
  if err != nil {
    return nil, err
  }

  var quote Quote
  err = json.Unmarshal(bodyBytes, &quote)
  if err != nil {
    return nil, err
  }
  quote.Lang = lang

  return &quote, nil
}
```

We run the tests and see...

```console
go test -v ./...
=== RUN   TestHandleQuotes
=== RUN   TestHandleQuotes/QuoteGenerator_Success
=== RUN   TestHandleQuotes/QuoteGenerator_Fail
--- PASS: TestHandleQuotes (0.00s)
    --- PASS: TestHandleQuotes/QuoteGenerator_Success (0.00s)
        handlers_test.go:69: PASS:      Generate(string)
    --- PASS: TestHandleQuotes/QuoteGenerator_Fail (0.00s)
        handlers_test.go:69: PASS:      Generate(string)
=== RUN   TestQuoteAPI
--- PASS: TestQuoteAPI (1.24s)
PASS
=== RUN   TestForismatic_Generate
=== RUN   TestForismatic_Generate/SuccessResponseFromHTTPWrapper
--- PASS: TestForismatic_Generate (0.00s)
    --- PASS: TestForismatic_Generate/SuccessResponseFromHTTPWrapper (0.00s)
PASS
```

Let's add test for the negative flow

```golang
// quote/quote_test.go
// ...
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
// ...
```

All tests are passes, but there one small thing left to fix. Now our e2e service uses the actual `http.Client` to call to `forismatic` service. This is problematic because we are not in control of how many times our test going to run (in benchmarks for example). This may create unnecessary load on external service. The good news is that we already know how to do it. Let's use `httptest.Server` again.

```golang
// main_test.go
package main

// ...

var mockForismaticServiceResponse = map[string]interface{}{
  "quoteText":   "Bla Bla Bla",
  "quoteAuthor": "Bob",
}

var expectedQuote = map[string]interface{}{
  "quoteAuthor": "Bob",
  "quoteText":   "Bla Bla Bla",
  "lang":        "en",
}

func TestQuoteAPI(t *testing.T) {
  testCases := []struct {
    name            string
    lang            string
    mockHTTPService func() *httptest.Server
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
      http.StatusOK,
      expectedQuote,
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

      respBytes, _ := ioutil.ReadAll(response.Body)

      var respMap map[string]interface{}
      _ = json.Unmarshal(respBytes, &respMap)

      assert.Equal(t, tC.expectedStatus, response.Code, "Response HTTP status in different than expected")
      assert.Equal(t, tC.expectedBody, respMap, "Response HTTP body in different than expected")
    })
  }
}
// ...
```

Now if we run the tests again, the will pass as before, but now there is no call to the external service. Another thing that is worth mentioning is that we used `map[string]interface{}` as the service response instead of the real struct (`quote.Quote`). I prefer this way because it is may indicate when we break interface with our users. In such case, the test will fail.


## Step 3 - Working with DB

I assume that you already have Postgres DB installed on your machine (if not please refer to www.postgresql.org). Now we create two DB, the real one and one that will be used for tests.

```console
> createdb quotes
> createdb quotes_test
```

Now we add some migrations files to create the recipient table.

```
> migrate create -ext sql -dir migrations  create_recipient_table
```

```sql
-- migrations\..._create_recipient_table.up.sql
CREATE TABLE recipients
(
  id SERIAL,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  CONSTRAINT recipients_pkey PRIMARY KEY (id)
);
```

Run the migration

```console
> migrate -source file:migrations -database postgres://localhost:5432/quotes?sslmode=disable up
> migrate -source file:migrations -database postgres://localhost:5432/quotes_test?sslmode=disable up
```

Now we add the recipient model
```golang
// recipient/recipient.go
package recipient

import (
  "database/sql"
)

// Recipient ...
type Recipient struct {
  ID    int    `json:"id"`
  Name  string `json:"name"`
  Email string `json:"email"`
}

// Persistence ...
type Persistence struct {
  DB *sql.DB
}

// NewPersistence ...
func NewPersistence(host, dbName string) (*Persistence, error) {
  db, err := sql.Open("postgres", fmt.Sprintf("dbname=%s host=%s sslmode=disable", dbName, host))
  if err != nil {
      return nil, err
  }

  return &Persistence{
      DB: db,
  }, nil
}

// AllRecipients ...
func (p *Persistence) AllRecipients() ([]Recipient, error) {
  var recipients []Recipient

  return recipients, nil
}
```

Now write a test

```golang
// recipient/recipient_test.go
package recipient

import (
  "database/sql"
  "fmt"
  _ "github.com/lib/pq"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
  "os"
  "testing"
)

var testPersistence *Persistence
var expectedRecipients = []Recipient{
  {
    ID:    1,
    Name:  "user1",
    Email: "user1@testmail.com",
  },
  {
    ID:    2,
    Name:  "user2",
    Email: "user2@testmail.com",
  },
  {
    ID:    3,
    Name:  "user3",
    Email: "user3@testmail.com",
  },
}

func TestMain(m *testing.M) {
  var err error
  testPersistence, err = NewPersistence("localhost", "quotes_test")
  if err != nil {
    panic(err)
  }

  code := m.Run()

  os.Exit(code)
}

func TestAllRecipients(t *testing.T) {
  testCases := []struct {
    name               string
    presetDB           func(db *sql.DB) error
    expectedRecipients []Recipient
    err                error
  }{
    {
      "RecipientsFound",
      func(db *sql.DB) error {
        query := "INSERT INTO recipients (id, name, email) VALUES ($1, $2, $3);"
        tx, err := db.Begin()

        for _, r := range expectedRecipients {
          _, err = tx.Exec(query, r.ID, r.Name, r.Email)
          if err != nil {
            fmt.Println(fmt.Sprintf("Error: %+v", err))
          }
        }

        tx.Commit()
        return err
      },
      expectedRecipients,
      nil,
    },
  }
  for _, tC := range testCases {
    t.Run(tC.name, func(t *testing.T) {
      err := clearDB(testPersistence.DB)
      require.NoErrorf(t, err, "Should have not error when cleaning the DB")

      err = tC.presetDB(testPersistence.DB)
      require.NoErrorf(t, err, "Should have not error when presetting the DB")

      recipients, err := testPersistence.AllRecipients()

      assert.Equal(t, err, tC.err, "Error should be as expected")
      assert.ElementsMatch(t, recipients, tC.expectedRecipients, "Response should be as expected")
    })
  }
}

func clearDB(db *sql.DB) error {
  _, err := db.Exec("TRUNCATE TABLE recipients")
  return err
}
```

This test file needs a bit of explanation.
1. I use `quotes_test` DB for tests. In order to be able to freely truncate all the tables before each test and run clean.
2. The use of `*testing.M`: TestMain gives us more control over how the tests of this package are running.  Here we use it to run a setup. It runs once before all of the tests in this package.
2. We test `recipient.Persistence` without mocking its dependency (DB). Unfortunately, we don't use an ORM here, and even if we did, it is difficult not to use custom SQL scripts that are passed as a string to different DB methods. This SQL script is not compiled, means that errors are available only in run time. Our tests provide such a runtime.
3. Use `require` package after a cleanup and setup. `require` (in contrast to `assert`) package stops the test run if the condition wasn't met. Here it makes sense, if I wasn't able to set up or clean last run data, there is no point to run the following tests.

Let's run the tests:
```console
go test -v ./...
...
=== RUN   TestAllRecipients
=== RUN   TestAllRecipients/RecipientsFound
--- FAIL: TestAllRecipients (0.03s)
    --- FAIL: TestAllRecipients/RecipientsFound (0.03s)
        recipient_test.go:66:
                Error Trace:    recipient_test.go:66
                Error:          Expected value not to be nil.
                Test:           TestAllRecipients/RecipientsFound
                Messages:       Recipients slice should not be nil
        recipient_test.go:67:
                Error Trace:    recipient_test.go:67
                Error:          "[]" should have 3 item(s), but has 0
                Test:           TestAllRecipients/RecipientsFound
                Messages:       Recipient slice should have 3 emails
FAIL
```

Tests are failed, as expected. Now we implement the method.

```golang
// recipient/recipient.go
// ...
func (p *Persistence) AllRecipients() ([]Recipient, error) {
  var recipients []Recipient

  rows, err := p.DB.Query("select * from recipients")
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  for rows.Next() {
    var r Recipient
    if err := rows.Scan(&r.ID, &r.Name, &r.Email); err == nil {
      recipients = append(recipients, r)
    }
  }

  return recipients, nil
}
// ...
```

We run the tests and see

```console
> go test -v ./...
...
=== RUN   TestAllRecipients
=== RUN   TestAllRecipients/RecipientsFound
--- PASS: TestAllRecipients (0.05s)
    --- PASS: TestAllRecipients/RecipientsFound (0.05s)
PASS
...
```

Now we add a negative test

```golang
// recipient/recipient_test.go
// ...
func TestAllRecipients(t *testing.T) {
  testCases := []struct {
    // ...
  }{
    // ...
    {
      "RecipientsNotFound",
      func(db *sql.DB) error {
        return nil
      },
      nil,
      nil,
    },
    // ...
  }
}
// ...
```

We run the tests again ...

```console
> go test -v ./...
=== RUN   TestAllRecipients
=== RUN   TestAllRecipients/RecipientsFound
=== RUN   TestAllRecipients/RecipientsNotFound
--- PASS: TestAllRecipients (0.03s)
    --- PASS: TestAllRecipients/RecipientsFound (0.02s)
    --- PASS: TestAllRecipients/RecipientsNotFound (0.00s)
PASS
```

Now let's app e2e test for that. I want to add the recipients' list to the `/quotes` API response. We expand the API response as follows.

```golang
// handlers.go
package main

// ...

// HandleQuoteResponse ..
type HandleQuoteResponse struct {
  Quote      *quote.Quote          `json:"quote"`
  Recipients []recipient.Recipient `json:"recipients"`
}

func (s *server) handleQuotes() http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    // ...

    hqr := HandleQuoteResponse{
      Quote: quote,
    }

    resp, err := json.Marshal(hqr)

    // ...
  }
}

```

We run the tests and see that the e2e test failed

```console
> go test -v ./...
=== RUN   TestQuoteAPI
=== RUN   TestQuoteAPI/SuccessResponseFromForismaticService
--- FAIL: TestQuoteAPI (0.00s)
    --- FAIL: TestQuoteAPI/SuccessResponseFromForismaticService (0.00s)
        main_test.go:82: 
                Error Trace:    main_test.go:82
                Error:          Not equal: 
                                expected: map[string]interface {}{"lang":"en", "quoteAuthor":"Bob", "quoteText":"Bla Bla Bla"}
                                actual  : map[string]interface {}{"quote":map[string]interface {}{"lang":"en", "quoteAuthor":"Bob", "quoteText":"Bla Bla Bla"}, "recipients":interface {}(nil)}

                                ...

                Test:           TestQuoteAPI/SuccessResponseFromForismaticService
                Messages:       Response HTTP body in different than expected
FAIL
```

We fix the test and assertion for the recipients' collection

```golang
// main_test.go

// ...

var srv server
var testRecipientsPersistence *recipient.Persistence

var mockForismaticServiceResponse = map[string]interface{}{
  "quoteText":   "Bla Bla Bla",
  "quoteAuthor": "Bob",
}

var expectedRecipients = []map[string]interface{}{
  map[string]interface{}{
    "id":    1,
    "name":  "user1",
    "email": "user1@testmail.com",
  },
  map[string]interface{}{
    "id":    2,
    "name":  "user2",
    "email": "user2@testmail.com",
  },
  map[string]interface{}{
    "id":    3,
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
          _, err = tx.Exec(query, r["id"], r["name"], r["email"])
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
      assert.Equal(t, tC.expectedBody, respMap, "Response HTTP body in different than expected")
    })
  }
}

// ...

func clearDB(db *sql.DB) error {
  _, err := db.Exec("TRUNCATE TABLE recipients")
  return err
}
```

Now the e2e test is pre-setting the DB, asserting the request to have recipients slice, but still fails because we didn't implement recipients fetch inside the handler. Let's do it, and fix handler_test as we go.

```golang
// handlers.go
// ...
func (s *server) handleQuotes() http.HandlerFunc {
  return func(w http.ResponseWriter, r *http.Request) {
    // ...
    recipients, err := s.recipientsFetcher.AllRecipients()
    if err != nil {
      w.WriteHeader(http.StatusInternalServerError)
      return
    }

    hqr := HandleQuoteResponse{
      Quote:      quote,
      Recipients: recipients,
    }
    // ...
  }
}
// ...
```

```golang
// handlers_test.go
// ...
import (
  // ...
  "./recipient"
  // ...
)

// ...

type MockRecipientsFetcher struct {
  mock.Mock
}

func (m *MockRecipientsFetcher) AllRecipients() ([]recipient.Recipient, error) {
  args := m.Called()
  r, _ := args.Get(0).([]recipient.Recipient)
  return r, args.Error(1)
}

func TestHandleQuotes(t *testing.T) {
  testCases := []struct {
    name           string
    lang           string
    createMocks    func() (*MockQuoteGenerator, *MockRecipientsFetcher)
    expectedStatus int
  }{
      {
        "QuoteGenerator_Success",
        "en",
        func() (*MockQuoteGenerator, *MockRecipientsFetcher) {
          mockQuoteGenerator := MockQuoteGenerator{}
          mockQuoteGenerator.On("Generate", "en").Return(&quote.Quote{}, nil)

          mockRecipientsFetcher := MockRecipientsFetcher{}
          mockRecipientsFetcher.On("AllRecipients").Return([]recipient.Recipient{}, nil)

          return &mockQuoteGenerator, &mockRecipientsFetcher
        },
        http.StatusOK,
      },
      {
        "QuoteGenerator_Fail",
        "en",
        func() (*MockQuoteGenerator, *MockRecipientsFetcher) {
          mockQuoteGenerator := MockQuoteGenerator{}
          mockQuoteGenerator.On("Generate", "en").Return(nil, errors.New("sample error"))

          mockRecipientsFetcher := MockRecipientsFetcher{}

          return &mockQuoteGenerator, &mockRecipientsFetcher
        },
        http.StatusInternalServerError,
      },
      {
        "RecipientsFetcher_Fail",
        "en",
        func() (*MockQuoteGenerator, *MockRecipientsFetcher) {
          mockQuoteGenerator := MockQuoteGenerator{}
          mockQuoteGenerator.On("Generate", "en").Return(&quote.Quote{}, nil)

          mockRecipientsFetcher := MockRecipientsFetcher{}
          mockRecipientsFetcher.On("AllRecipients").Return(nil, errors.New("sample error"))

          return &mockQuoteGenerator, &mockRecipientsFetcher
        },
        http.StatusInternalServerError,
      },
  }

  for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
      mockQuoteGenerator, mockRecipientsFetcher := tc.createMocks()
      svr := server{
        quoteGenerator:    mockQuoteGenerator,
        recipientsFetcher: mockRecipientsFetcher,
      }

      rr := httptest.NewRecorder()
      req, _ := http.NewRequest("GET", "/quote", nil)
      req.URL.RawQuery = fmt.Sprintf("lang=%s", tc.lang)

      svr.handleQuotes()(rr, req)

      assert.Equal(t, tc.expectedStatus, rr.Code, "Response HTTP status in different than expected")
      mockQuoteGenerator.AssertExpectations(t)
      mockRecipientsFetcher.AssertExpectations(t)
    })
  }
}
```

```golang
// main_test.go

import (
  // ...
  "./recipient"
  // ...
  _ "github.com/lib/pq"
  // ...
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
```

We run the tests and everything is green.

```console
> go test -v ./...
=== RUN   TestHandleQuotes
=== RUN   TestHandleQuotes/QuoteGenerator_Success
=== RUN   TestHandleQuotes/QuoteGenerator_Fail
=== RUN   TestHandleQuotes/RecipientsFetcher_Fail
--- PASS: TestHandleQuotes (0.00s)
    --- PASS: TestHandleQuotes/QuoteGenerator_Success (0.00s)
        handlers_test.go:102: PASS:     Generate(string)
        handlers_test.go:103: PASS:     AllRecipients()
    --- PASS: TestHandleQuotes/QuoteGenerator_Fail (0.00s)
        handlers_test.go:102: PASS:     Generate(string)
    --- PASS: TestHandleQuotes/RecipientsFetcher_Fail (0.00s)
        handlers_test.go:102: PASS:     Generate(string)
        handlers_test.go:103: PASS:     AllRecipients()
=== RUN   TestQuoteAPI
=== RUN   TestQuoteAPI/SuccessResponseFromForismaticService
--- PASS: TestQuoteAPI (0.03s)
    --- PASS: TestQuoteAPI/SuccessResponseFromForismaticService (0.03s)
PASS
=== RUN   TestForismatic_Generate
=== RUN   TestForismatic_Generate/SuccessResponseFromHTTPWrapper
=== RUN   TestForismatic_Generate/ErrorFromHTTPWrapper
--- PASS: TestForismatic_Generate (0.00s)
    --- PASS: TestForismatic_Generate/SuccessResponseFromHTTPWrapper (0.00s)
    --- PASS: TestForismatic_Generate/ErrorFromHTTPWrapper (0.00s)
PASS
=== RUN   TestAllRecipients
=== RUN   TestAllRecipients/RecipientsFound
=== RUN   TestAllRecipients/RecipientsNotFound
--- PASS: TestAllRecipients (0.07s)
    --- PASS: TestAllRecipients/RecipientsFound (0.06s)
    --- PASS: TestAllRecipients/RecipientsNotFound (0.00s)
PASS
```