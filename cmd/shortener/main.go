package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

const Charset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ"

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()))

func randString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = Charset[seededRand.Intn(len(Charset))]
	}
	return string(b)
}

var urlsStorage = make(map[string]string)

func shortener(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Only POST or GET requests are allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Method == http.MethodPost {

		shortUrl := randString(8)
		urlFromRequest, _ := io.ReadAll(r.Body)
		urlsStorage[shortUrl] = string(urlFromRequest)
		body := fmt.Sprintf("http://localhost:8080/%s", shortUrl)
		w.Header().Add("Content-Type", "text/plain")
		w.Header().Add("Host", "localhost:8080")
		w.WriteHeader(http.StatusCreated)
		_, err := w.Write([]byte(body))
		if err != nil {
			panic(err)
		}
	}

	if r.Method == http.MethodGet {
		shortUrl := strings.Split(r.RequestURI, "/")
		if len(shortUrl) != 2 || shortUrl[1] == "" {
			http.Error(w, "Bad Request, not short url", http.StatusBadRequest)
			return
		}
		w.Header().Add("Location", urlsStorage[shortUrl[1]])
		w.WriteHeader(http.StatusTemporaryRedirect)
	}

}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, shortener)
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
