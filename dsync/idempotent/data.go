package idempotent

type data struct {
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

func makeData(req, res []byte) data {
	return data{
		Request:  hash(req),
		Response: string(res),
	}
}
