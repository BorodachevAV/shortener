package main

import (
	"net/http"
	"strings"
)

func shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}
	if r.Method != http.MethodPost {
		body := "http://localhost:8080/EwHXdJfB"

		w.Header().Add("Content-Type", "text/plain")
		w.Header().Add("Host", "localhost:8080")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(body))
	} else if r.Method != http.MethodGet {
		shortUrl := strings.Split(r.RequestURI, "/")
		if len(shortUrl) > 2 {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		body := "Location: https://practicum.yandex.ru/"
		w.WriteHeader(http.StatusTemporaryRedirect)
		w.Write([]byte(body))
	}

}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, shorten)
	//mux.HandleFunc(`/qwerty`, getUrl)
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
