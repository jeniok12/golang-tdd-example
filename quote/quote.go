// quote/quote.go

package quote

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

// Quote ...
type Quote struct {
	Text   string `json:"quoteText"`
	Author string `json:"quoteAuthor"`
	Lang   string `json:"lang"`
}

// HTTPWrapper ...
type HTTPWrapper interface {
	Do(req *http.Request) (*http.Response, error)
}

// Forismatic ...
type Forismatic struct {
	URL    string
	Client HTTPWrapper
}

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
