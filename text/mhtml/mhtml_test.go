package mhtml_test

import (
	"os"
	"strings"
	"testing"

	"github.com/alextanhongpin/core/text/mhtml"
)

func TestReadSample(t *testing.T) {
	for _, body := range []bool{true, false} {
		f, err := os.Open("testdata/python-telegram-bot.mhtml")
		if err != nil {
			t.Fatal(err)
		}
		html, md, err := mhtml.Read(f, body)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got := md.URL.String(); got != "https://python-telegram-bot.org/" {
			t.Errorf("URL = %q", got)
		}
		if md.Title != "python-telegram-bot" {
			t.Errorf("Title = %q", md.Title)
		}
		if md.Date.IsZero() {
			t.Error("Date is zero")
		}
		if !body {
			if html != nil {
				t.Error("html returned without body")
			}
			continue
		}
		s := string(html)
		if !strings.Contains(s, "<title>python-telegram-bot</title>") {
			t.Error("quoted-printable body was not decoded")
		}
		if strings.Contains(s, `href="https://python-telegram-bot.org/dark.css"`) {
			t.Error("stylesheet was not inlined")
		}
		if !strings.Contains(s, "data:text/css;base64,") || !strings.Contains(s, "data:image/png;base64,") {
			t.Error("resources were not inlined as data URIs")
		}
	}
}

const header = "From: <Saved by Blink>\r\n" +
	"Snapshot-Content-Location: https://example.com/\r\n" +
	"Subject: =?utf-8?q?caf=C3=A9?=\r\n" +
	"MIME-Version: 1.0\r\n"

func archive(params string, parts ...string) string {
	s := header + "Content-Type: multipart/related; boundary=\"B\"" + params + "\r\n\r\n"
	for _, p := range parts {
		s += "--B\r\n" + p + "\r\n"
	}
	return s + "--B--\r\n"
}

func read(t *testing.T, s string) (string, *mhtml.Metadata, error) {
	t.Helper()
	html, md, err := mhtml.Read(strings.NewReader(s), true)
	return string(html), md, err
}

func TestRead(t *testing.T) {
	t.Run("metadata without date", func(t *testing.T) {
		_, md, err := read(t, archive("", "Content-Type: text/html\r\n\r\n<p>hi</p>"))
		if err != nil {
			t.Fatal(err)
		}
		if !md.Date.IsZero() {
			t.Errorf("Date = %v, want zero", md.Date)
		}
		if md.Title != "café" {
			t.Errorf("Title = %q, want decoded subject", md.Title)
		}
	})

	t.Run("no html part is an error, not a panic", func(t *testing.T) {
		_, _, err := read(t, archive("", "Content-Type: image/png\r\nContent-Location: a.png\r\n\r\nx"))
		if err == nil || !strings.Contains(err.Error(), "not text/html") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("part without references is ignored", func(t *testing.T) {
		got, _, err := read(t, archive("",
			"Content-Type: text/html\r\n\r\n<p>hi</p>",
			"Content-Type: image/png\r\n\r\nx",
		))
		if err != nil {
			t.Fatal(err)
		}
		if got != "<p>hi</p>" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("referenced by location and content-id", func(t *testing.T) {
		got, _, err := read(t, archive("",
			"Content-Type: text/html\r\n\r\n<img src=\"https://example.com/a.png\"><img src=\"cid:img1\">",
			"Content-Type: image/png\r\nContent-ID: <img1>\r\nContent-Location: https://example.com/a.png\r\nContent-Transfer-Encoding: base64\r\n\r\neA==",
		))
		if err != nil {
			t.Fatal(err)
		}
		want := `<img src="data:image/png;base64,eA=="><img src="data:image/png;base64,eA==">`
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("longest reference wins and data is not rewritten", func(t *testing.T) {
		got, _, err := read(t, archive("",
			"Content-Type: text/html\r\n\r\n<a href=\"x\"></a><img src=\"x.png\">",
			"Content-Type: text/plain\r\nContent-Location: x\r\n\r\nx.png",
			"Content-Type: image/png\r\nContent-Location: x.png\r\n\r\nx",
		))
		if err != nil {
			t.Fatal(err)
		}
		want := `<a href="data:text/plain;base64,eC5wbmc="></a><img src="data:image/png;base64,eA==">`
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("start parameter selects the root", func(t *testing.T) {
		got, _, err := read(t, archive(`; start="<root>"`,
			"Content-Type: text/html\r\nContent-ID: <frame>\r\n\r\n<b>frame</b>",
			"Content-Type: text/html\r\nContent-ID: <root>\r\n\r\n<iframe src=\"cid:frame\"></iframe>",
		))
		if err != nil {
			t.Fatal(err)
		}
		if want := `<iframe src="data:text/html;base64,PGI+ZnJhbWU8L2I+"></iframe>`; got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("missing start part", func(t *testing.T) {
		_, _, err := read(t, archive(`; start="<nope>"`, "Content-Type: text/html\r\n\r\nx"))
		if err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, _, err := read(t, archive("",
			"Content-Type: text/html\r\n\r\nx",
			"Content-Type: image/png\r\nContent-Location: a\r\nContent-Transfer-Encoding: base64\r\n\r\n!!!",
		))
		if err == nil {
			t.Fatal("want error")
		}
	})
}
