// handlers_test.go

package main

import (
	"./quote"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"net/http/httptest"
	"testing"
)

type MockQuoteGenerator struct {
	mock.Mock
}

func (m *MockQuoteGenerator) Generate(lang string) (*quote.Quote, error) {
	args := m.Called(lang)
	quote, _ := args.Get(0).(*quote.Quote)
	return quote, args.Error(1)
}

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
