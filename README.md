# Step by step guide of building HTTP service using Golang and TDD approach

## Intro

In the following guide I will present how to build an HTTP service using `go1.12.4`, `gorilla/mux` for URL routing, `golang/mock` for mocks and `stretchr/testify` for assertions. This is my stuck, feel free to use diferent packages it shouldn't affect much of the following. Also, I will try to follow TDD approch as much as possible. All of the sourec code is avaliable on github.

## Step 0 - What should we build?

I decided to create an InspiringQuotes service, which I will use from time to time in order to increase my teammates morale. This service will generate an inspiring quote using http://forismatic.com/en/api/ service and send it to list of my teammates (stored in Postgres DB) via Email. At the end my colegues will have an example of how to write a testable Golang service in addition to hight morale to use it :)

## Step 1 - New Service is born

Let's create a new service. Our `main.go` file will look like this:
```golang
// main.go

package main

import (
  "github.com/gorilla/mux"
  "net/http"
)

func main() {
  r := mux.NewRouter()
  if err := http.ListenAndServe(":8080", r); err != nil {
    panic(err)
  }
}
```
Let's run our server using:
```console
> go run .
```
And then send a HTTP request and see.
```console
> curl http://localhost:8080
404 page not found
```
Now we are ready to write our first test. We want to test the service e2e, but without running it. We will send sample HTTP request and then run asserts on the HTTP response and on other side effects (DB updates, calls to other services and so).In order to do that we need:
1. Make router accesseble from tests.
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

  if err := http.ListenAndServe(":8080", svr.router); err != nil {
    panic(err)
  }
}
```
Implement the `routes` method. For now it will generate no routes.
```golang
// routes.go

package main

func (s *server) routes() {
	// Add routes handlers here.
}
```
Now lets write an E2E test:
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
> go test -v .
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
> go test -v .
=== RUN   TestQuoteAPI
--- PASS: TestQuoteAPI (0.00s)
PASS
```