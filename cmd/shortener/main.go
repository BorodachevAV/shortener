package main

import (
	"net/http"
)

func shorten(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	body := "http://localhost:8080/EwHXdJfB"

	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Host", "localhost:8080")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(body))
}
func getUrl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed!", http.StatusMethodNotAllowed)
		return
	}
	body := "Location: https://practicum.yandex.ru/"

	w.WriteHeader(http.StatusTemporaryRedirect)
	w.Write([]byte(body))

}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc(`/`, shorten)
	mux.HandleFunc(`/EwHXdJfB`, getUrl)
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
