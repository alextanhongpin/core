package circuitbreaker

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestTransportClosesRequestBodyWhenOpened(t *testing.T) {
	cb := MustNew(Config{})
	cb.SetStatus(ForcedOpen)
	body := &closeBody{Reader: strings.NewReader("payload")}
	req, err := http.NewRequest(http.MethodPost, "http://example.invalid", io.NopCloser(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Body = body
	tr := NewTransporter(transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("transport called while open")
		return nil, nil
	}), cb)
	if _, err := tr.RoundTrip(req); err != ErrOpened {
		t.Fatal(err)
	}
	if !body.closed {
		t.Fatal("RoundTrip must close the request body even when rejecting")
	}
}
