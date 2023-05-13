package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

func main() {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json;charset=utf-8")
		fmt.Fprint(w, `{"hello": "10"}`)
		//fmt.Fprint(w, nil)
	}

	w := httptest.NewRecorder()

	//form := url.Values{}
	//form.Add("name", "john")
	//form.Add("age", "10")
	//r := httptest.NewRequest("GET", "/hello?name=hello", strings.NewReader(form.Encode()))
	r := httptest.NewRequest("GET", "/hello?name=hello", strings.NewReader(`{"name": "john", "age": 10}`))
	q := r.URL.Query()
	q.Add("limit", "10")
	q.Add("offset", "20")
	r.Body = io.NopCloser(r.Body)
	r.URL.RawQuery = q.Encode()
	//r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("Content-Type", "application/json;charset=utf-8;")
	handler(w, r)
	ws := w.Result()

	dothttpFormat := format(ws, r)
	fmt.Println(dothttpFormat)
	fmt.Println(parseResponse(ws, dothttpFormat))
}
