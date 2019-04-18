# Step by step guide of building HTTP service using Golang and TDD

## Intro

In the following guide, I will present how to build an HTTP service using `go1.12.4`, `gorilla/mux` for URL routing, `stretchr/testify` for mocks and assertions and `lib/pq` for Postgres. This is my stack, feel free to use different packages it shouldn't affect much of the following. Also, I will try to follow TDD approach as much as possible. All of the source code is available on GitHub.

```console
git clone https://github.com/jeniok12/golang-tdd-example.git

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
1. Make router accessible for tests.
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

Now lets write an E2E test. We use `httptest.ResponseRecorder` as  `http.ResponseWriter` in our calls to the router. `httptest.ResponseRecorder` will store the HTTP response, so we will be able to assert it with expected result.

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

## Step 2 - Call to other service

On this step we will implement request validation and the actual request to http://forismatic.com/en/api/ in order to get the quote.

We will start by implementing the `quote` package. Create the Quote model.
```golang
// quote/quote.go

package quote

type Quote struct {
  Text   string `json:"quoteText"`
  Author string `json:"quoteAuthor"`
  Lang   string
}
```
Quotes generator interface that will be added to server as a dependency
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
Now we want to inject server dependancies to the handlers. For that we will wrap the `HandlerFunc` inside `Handler`, and move it to different file. 
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
Now we write a test for `handleQuotes`. This test mocks `QuoteGenerator` and test the service if the Quote was generated successfuly and not.

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
Now our test passes but our e2e test is failing because we didn't implemented the actual `QuoteGenerator`. We deal with it in a minute, but first of all let's test `quoteHandler` for negative flow.

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
This is passes too, but seem that we have code duplication that we want to prevent.
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
I prefer to call it as the name of the service it uses. The 'Forismatic' struct holds the `HTTPWrapper` interface of `http.Client` so we will be able to mock it in tests.

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
We still see, the test fail. But it is not `panic: runtime error: invalid memory address` any more. Our service returns 500 beause `Forismatic.Generate` returns not implemented error. Let's implement this method, starting with the test.
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
  "quoteAuthor": "Moshe",
}

var expectedQuote = Quote{
  Author: "Moshe",
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
We already know how to use testCases table so I used it from the start. The new thing here is that this logic depends on `http.Client`. Good news that we don't need to mock it using `testify/mock`. Instead we use `httptest.Server`. This mock server gives us the ability to assert on the external requests and mock the responses. We run the test and see...
```concole
> go test -v ./...
...
=== RUN   TestForismatic_Generate
=== RUN   TestForismatic_Generate/SuccessResponseFromHTTPWrapper
--- FAIL: TestForismatic_Generate (0.00s)
    --- FAIL: TestForismatic_Generate/SuccessResponseFromHTTPWrapper (0.00s)
        quote_test.go:63: 
                Error Trace:    quote_test.go:63
                Error:          Not equal: 
                                expected: &quote.Quote{Text:"Bla Bla Bla", Author:"Moshe", Lang:"en"}
                                actual  : (*quote.Quote)(nil)
                            
                                Diff:
                                --- Expected
                                +++ Actual
                                @@ -1,6 +1,2 @@
                                -(*quote.Quote)({
                                - Text: (string) (len=11) "Bla Bla Bla",
                                - Author: (string) (len=5) "Moshe",
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
Failed as expected, lets implement the `Generate` function
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
All tests are passes, but there one small thing left to fix. Now our e2e service uses the actual `http.Client` to call to `forismatic` service. This is problematic, because we are not in control of how many times our test going to run (in benchmarks). This may create unnesasry load on external service. The good news is that we already know how to do it. Let's use `httptest.Server` again.
```golang
// main_test.go

package main

// ...

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
// ...
```
Now if we run the tests again, the will pass as before, but now there is no call to the external service.
