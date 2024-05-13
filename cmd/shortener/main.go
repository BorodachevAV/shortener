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
	"net/url"
	"sync"
	"time"
)

// алфавит для коротких url
const Charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// генерим сид
var (
	urlsStorage sync.Map
	seededRand  *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	cfg                    = config.New()
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

	//читаем тело
	urlFromRequest, _ := io.ReadAll(r.Body)

	_, err := url.Parse(string(urlFromRequest))
	if err != nil {
		http.Error(w, "not url", http.StatusBadRequest)
		return
	}
	//генерим короткий url
	shortURL := randString(8)
	// сохраняем в мапе
	urlsStorage.Store(shortURL, urlFromRequest)
	//заполняем ответ
	body := fmt.Sprintf("%s/%s", cfg.Cfg.BaseURL, shortURL)
	w.Header().Add("Content-Type", "text/plain")
	w.Header().Add("Host", cfg.Cfg.ServerAddress)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(body))
	if err != nil {
		log.Fatal(err)
	}
}
func expand(w http.ResponseWriter, r *http.Request) {
	shortURL := chi.URLParam(r, "id")
	//проверяем в мапе наличие ключа, отдаем 404 если его нет
	val, ok := urlsStorage.Load(shortURL)

	if ok {
		w.Header().Add("Location", val.(string))
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else {
		http.Error(w, "short url not found", http.StatusNotFound)
		return
	}
}
func main() {
	a := flag.String("a", "localhost:8080", "shortener host")
	b := flag.String("b", "http://localhost:8080", "response host")
	flag.Parse()
	if cfg.Cfg.ServerAddress == "" {
		cfg.Cfg.ServerAddress = *a
	}
	if cfg.Cfg.BaseURL == "" {
		cfg.Cfg.BaseURL = *b
	}
	r := chi.NewRouter()
	r.Post(`/`, shorten)
	r.Get(`/{id}`, expand)
	log.Fatal(http.ListenAndServe(cfg.Cfg.ServerAddress, r))
}
