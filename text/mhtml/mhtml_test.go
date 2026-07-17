package mhtml_test

import (
	"os"
	"testing"

	"github.com/alextanhongpin/core/text/mhtml"
	"github.com/go-openapi/testify/assert"
)

func Test_Read(t *testing.T) {
	t.Run("with body", func(t *testing.T) {
		f, err := os.Open("testdata/python-telegram-bot.mhtml")
		is := assert.New(t)
		is.NoError(err)
		defer func() {
			_ = f.Close()
		}()

		body := true
		html, metadata, err := mhtml.Read(f, body)
		is.NoError(err)
		is.Equal("https://python-telegram-bot.org/", metadata.URL.String())
		is.Equal("python-telegram-bot", metadata.Title)
		is.NotNil(html)
		is.NotZero(metadata.Date)
	})

	t.Run("without body", func(t *testing.T) {
		f, err := os.Open("testdata/python-telegram-bot.mhtml")
		is := assert.New(t)
		is.NoError(err)
		defer func() {
			_ = f.Close()
		}()

		body := false
		html, metadata, err := mhtml.Read(f, body)
		is.NoError(err)
		is.Equal("https://python-telegram-bot.org/", metadata.URL.String())
		is.Equal("python-telegram-bot", metadata.Title)
		is.Nil(html)
		is.NotZero(metadata.Date)
	})
}
