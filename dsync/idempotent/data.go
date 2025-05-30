package idempotent

import "encoding/base64"

type data struct {
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

func makeData(req, res []byte) data {
	return data{
		Request:  hash(req),
		Response: base64.StdEncoding.EncodeToString(res),
	}
}

// getResponseBytes safely decodes the base64-encoded response
func (d *data) getResponseBytes() ([]byte, error) {
	return base64.StdEncoding.DecodeString(d.Response)
}
