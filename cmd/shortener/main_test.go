package main

import (
	"bytes"
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestShortener(t *testing.T) {
	t.Run("Get 404", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		expand(w, request)

		res := w.Result()
		assert.Equal(t, 404, res.StatusCode)
		res.Body.Close()
	})

	t.Run("Get 307", func(t *testing.T) {
		url := `https://stackoverflow.com/`
		bodyReader := strings.NewReader(url)
		request := httptest.NewRequest(http.MethodPost, "/", bodyReader)
		w := httptest.NewRecorder()
		shorten(w, request)
		shortURL := w.Result().Body
		shortURL.Close()
		buf := new(bytes.Buffer)
		buf.ReadFrom(shortURL)
		respBytes := buf.String()
		respBytes = respBytes[len(respBytes)-8:]

		request = httptest.NewRequest(http.MethodGet, "/{id}", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", respBytes)
		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))
		w = httptest.NewRecorder()
		expand(w, request)
		res := w.Result()
		res.Body.Close()
		assert.Equal(t, 307, res.StatusCode)
	})

	t.Run("Post 201", func(t *testing.T) {
		bodyReader := strings.NewReader(`https://stackoverflow.com/`)
		request := httptest.NewRequest(http.MethodPost, "/", bodyReader)
		w := httptest.NewRecorder()
		shorten(w, request)

		res := w.Result()
		res.Body.Close()
		assert.Equal(t, 201, res.StatusCode)
	})
}
