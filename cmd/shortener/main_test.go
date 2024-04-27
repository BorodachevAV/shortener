package main

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShortener(t *testing.T) {
	t.Run("Get 400", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		shortener(w, request)

		res := w.Result()
		assert.Equal(t, 400, res.StatusCode)
	})

	t.Run("Get 404", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/qwerty", nil)
		w := httptest.NewRecorder()
		shortener(w, request)

		res := w.Result()
		assert.Equal(t, 404, res.StatusCode)
	})

	t.Run("Get 307", func(t *testing.T) {
		url := `https://stackoverflow.com/`
		bodyReader := strings.NewReader(url)
		request := httptest.NewRequest(http.MethodPost, "/", bodyReader)
		w := httptest.NewRecorder()
		shortener(w, request)
		shortUrl := w.Result().Body

		buf := new(bytes.Buffer)
		buf.ReadFrom(shortUrl)
		respBytes := buf.String()
		respBytes = respBytes[len(respBytes)-8:]

		request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s", respBytes), nil)
		w = httptest.NewRecorder()
		shortener(w, request)

		res := w.Result()

		// проверяем код ответа
		assert.Equal(t, url, res.Header["Location"][0])
	})

	t.Run("Post 201", func(t *testing.T) {
		bodyReader := strings.NewReader(`https://stackoverflow.com/`)
		request := httptest.NewRequest(http.MethodPost, "/", bodyReader)
		w := httptest.NewRecorder()
		shortener(w, request)

		res := w.Result()
		assert.Equal(t, 201, res.StatusCode)
	})
}
