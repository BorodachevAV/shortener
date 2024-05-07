package main

import (
	"flag"
	"fmt"
	"github.com/BorodachevAV/shortener/internal/config"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// алфавит для коротких url
const Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// генерим сид
var (
	urlsStorage            = make(map[string]string)
	seededRand  *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	cfg                    = config.New()
	a                      = flag.String("a", "localhost:8080", "shortener host")
	b                      = flag.String("b", "http://localhost:8080", "response host")
)

// генерим короткий url
func randString(length int) string {
	randB := make([]byte, length)
	for i := range randB {
		randB[i] = Charset[seededRand.Intn(len(Charset))]
	}
	return string(randB)
}

func shorten(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	//читаем тело
	urlFromRequest, _ := io.ReadAll(r.Body)
	//генерим короткий url
	shortURL := randString(8)
	// сохраняем в мапе
	urlsStorage[shortURL] = string(urlFromRequest)
	//заполняем ответ
	if cfg.Cfg.ServerAddress == "" {
		cfg.Cfg.ServerAddress = *a
	}
	if cfg.Cfg.BaseURL == "" {
		cfg.Cfg.BaseURL = *b
	}
	body := fmt.Sprintf("%s/%s", cfg.Cfg.BaseURL, shortURL)
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Host", cfg.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err := w.Write([]byte(body))
	if err != nil {
		panic(err)
	}
}
func expand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	shortURL := chi.URLParam(r, "id")
	//проверяем в мапе наличие ключа, отдаем 404 если его нет
	val, ok := urlsStorage[shortURL]

	if ok {
		w.Header().Add("Location", val)
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		http.Error(w, "short url not found", http.StatusNotFound)
		return
	}
}
func main() {
	flag.Parse()
	if cfg.Cfg.ServerAddress == "" {
		cfg.Cfg.ServerAddress = *a
	}
	r := chi.NewRouter()
	r.Post(`/`, shorten)
	r.Get(`/{id}`, expand)
	log.Fatal(http.ListenAndServe(cfg.Cfg.ServerAddress, r))
}
