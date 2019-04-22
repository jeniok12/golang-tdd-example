// handlers_test.go

package main

import (
	"./quote"
	"./recipient"
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
