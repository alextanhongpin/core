package types

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result[T any] struct {
	Data  *T     `json:"data,omitempty"`
	Error *Error `json:"error,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
	Links *Links `json:"links,omitempty"`
}

type Meta map[string]any

type Links struct {
	Prev  string `json:"prev,omitempty"`
	Next  string `json:"next,omitempty"`
	First string `json:"first,omitempty"`
	Last  string `json:"last,omitempty"`
}
