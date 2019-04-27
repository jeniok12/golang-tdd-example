// routes.go

package main

func (s *server) routes() {
	s.router.HandleFunc("/quote", s.handleQuotes())
}
